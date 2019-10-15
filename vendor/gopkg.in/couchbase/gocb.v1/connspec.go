package gocb

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
)

type connSpecScheme int

const (
	csPlainMcd  connSpecScheme = 11210
	csPlainHttp connSpecScheme = 8091
	csSslMcd    connSpecScheme = 11207
	csSslHttp   connSpecScheme = 18091
	csInvalid   connSpecScheme = 0
)

// Gets the implicit port for the scheme
func (scheme connSpecScheme) DefaultPort() int {
	return int(scheme)
}

func (scheme connSpecScheme) IsMCD() bool {
	return scheme == csPlainMcd || scheme == csSslMcd
}

func (scheme connSpecScheme) IsHTTP() bool {
	return scheme == csPlainHttp || scheme == csSslHttp
}

func (scheme connSpecScheme) IsSSL() bool {
	return scheme == csSslMcd || scheme == csSslHttp
}

func (scheme connSpecScheme) String() string {
	switch scheme {
	case csPlainHttp:
		return "http"
	case csPlainMcd:
		return "couchbase"
	case csSslHttp:
		return "https"
	case csSslMcd:
		return "couchbases"
	default:
		return ""
	}
}

func (scheme *connSpecScheme) load(s string) {
	switch s {
	case "couchbase":
		*scheme = csPlainMcd
	case "couchbases":
		*scheme = csSslMcd
	case "http":
		*scheme = csPlainHttp
	case "https":
		*scheme = csSslHttp
	}
}

// A single address stored within a connection string
type connSpecAddr struct {
	Host string
	Port uint16
}

func (a *connSpecAddr) HostPort() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

// A parsed connection string
type connSpec struct {
	Scheme            connSpecScheme
	MemcachedHosts    []*connSpecAddr
	HttpHosts         []*connSpecAddr
	Bucket            string
	Options           url.Values
	hasExplicitPort   bool
	hasExplicitScheme bool
}

// Loads a raw host definition into the spec
// Host is the raw hostname (without the port) and port is the port used. May be 0
func (cs *connSpec) addRawHost(host string, port int) error {
	// The port is "implicit", so we can derive the neighboring port
	if port != 0 && port != csPlainHttp.DefaultPort() && !cs.hasExplicitScheme {
		return errors.New("Ambiguous port without scheme")
	}
	if cs.hasExplicitScheme && cs.Scheme.IsMCD() && port == csPlainHttp.DefaultPort() {
		return errors.New("couchbase://host:8091 not supported for couchbase:// scheme. Use couchbase://host")
	}
	if port == 0 || port == cs.Scheme.DefaultPort() || port == csPlainHttp.DefaultPort() {
		tmpHtHost := &connSpecAddr{Host: host}
		tmpMcHost := &connSpecAddr{Host: host}
		if cs.Scheme.IsSSL() {
			tmpHtHost.Port = uint16(csSslHttp.DefaultPort())
			tmpMcHost.Port = uint16(csSslMcd.DefaultPort())
		} else {
			tmpHtHost.Port = uint16(csPlainHttp.DefaultPort())
			tmpMcHost.Port = uint16(csPlainMcd.DefaultPort())
		}

		cs.HttpHosts = append(cs.HttpHosts, tmpHtHost)
		cs.MemcachedHosts = append(cs.MemcachedHosts, tmpMcHost)

	} else {
		// Explicit non-standard port. Just add the port without anything funny
		tmpHost := &connSpecAddr{Host: host, Port: uint16(port)}
		if cs.Scheme.IsMCD() {
			cs.MemcachedHosts = append(cs.MemcachedHosts, tmpHost)
		} else {
			cs.HttpHosts = append(cs.HttpHosts, tmpHost)
		}
	}

	return nil
}

// Parses a connection string into a structure more easily consumed by the library.
func parseConnSpec(connStr string) (out connSpec, err error) {
	partMatcher := regexp.MustCompile(`((.*):\/\/)?(([^\/?:]*)(:([^\/?:@]*))?@)?([^\/?]*)(\/([^\?]*))?(\?(.*))?`)
	hostMatcher := regexp.MustCompile(`([^;\,\:]+)(:([0-9]*))?(;\,)?`)
	parts := partMatcher.FindStringSubmatch(connStr)

	if parts[2] != "" {
		(&out.Scheme).load(parts[2])
		if out.Scheme == csInvalid {
			err = fmt.Errorf("Unknown scheme '%s'", parts[2])
			return
		}
		out.hasExplicitScheme = true
	} else {
		out.Scheme = csPlainMcd
	}

	if parts[7] != "" {
		hosts := hostMatcher.FindAllStringSubmatch(parts[7], -1)
		for _, hostInfo := range hosts {
			port := 0
			if hostInfo[3] != "" {
				port, err = strconv.Atoi(hostInfo[3])
				if err != nil {
					return
				}
				out.hasExplicitPort = true
			}
			err = out.addRawHost(hostInfo[1], port)
			if err != nil {
				return
			}
		}
	}

	if len(out.HttpHosts) == 0 && len(out.MemcachedHosts) == 0 {
		err = out.addRawHost("127.0.0.1", 0)
		if err != nil {
			return
		}
	}

	if parts[9] != "" {
		out.Bucket, err = url.QueryUnescape(parts[9])
		if err != nil {
			return
		}
	}

	if parts[11] != "" {
		out.Options, err = url.ParseQuery(parts[11])
		if err != nil {
			return
		}
	}

	return
}

func csResolveDnsSrv(spec *connSpec) bool {
	if len(spec.MemcachedHosts) > 1 || len(spec.HttpHosts) > 1 {
		return false
	}

	if spec.hasExplicitPort || spec.Scheme.IsHTTP() {
		return false
	}

	srvHostname := spec.HttpHosts[0].Host
	_, addrs, err := net.LookupSRV(spec.Scheme.String(), "tcp", srvHostname)
	if err != nil || len(addrs) == 0 {
		return false
	}

	var hostList []*connSpecAddr
	for _, srvRecord := range addrs {
		hostList = append(hostList, &connSpecAddr{srvRecord.Target, srvRecord.Port})
	}

	spec.HttpHosts = nil
	spec.MemcachedHosts = nil

	if spec.Scheme.IsHTTP() {
		spec.HttpHosts = hostList
	} else {
		spec.MemcachedHosts = hostList
	}

	return true
}
