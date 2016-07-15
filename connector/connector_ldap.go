package connector

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"

	"fmt"

	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/coreos/dex/pkg/log"
	"github.com/coreos/go-oidc/oidc"

	"gopkg.in/ldap.v2"
)

const (
	LDAPConnectorType         = "ldap"
	LDAPLoginPageTemplateName = "ldap-login.html"
)

func init() {
	RegisterConnectorConfigType(LDAPConnectorType, func() ConnectorConfig { return &LDAPConnectorConfig{} })

	// Set default ldap timeout.
	ldap.DefaultTimeout = 30 * time.Second
}

type LDAPConnectorConfig struct {
	ID string `json:"id"`

	// Host and port of ldap service in form "host:port"
	Host string `json:"host"`

	// UseTLS indicates that the connector should use the TLS port.
	UseTLS bool `json:"useTLS"`
	UseSSL bool `json:"useSSL"`

	// Trusted TLS certificate when connecting to the LDAP server. If empty the
	// host's root certificates will be used.
	CaFile string `json:"caFile"`
	// CertFile and KeyFile are used to specifiy client certificate data.
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`

	MaxIdleConn int `json:"maxIdleConn"`

	NameAttribute  string `json:"nameAttribute"`
	EmailAttribute string `json:"emailAttribute"`

	// The place to start all searches from.
	BaseDN string `json:"baseDN"`

	// Search fields indicate how to search for user records in LDAP.
	SearchBeforeAuth  bool   `json:"searchBeforeAuth"`
	SearchFilter      string `json:"searchFilter"`
	SearchScope       string `json:"searchScope"`
	SearchBindDN      string `json:"searchBindDN"`
	SearchBindPw      string `json:"searchBindPw"`
	SearchGroupFilter string `json:"searchGroupFilter"`

	// BindTemplate is a format string that maps user names to a record to bind as.
	// It's passed both the username entered by the end user and the base DN.
	//
	// For example the bindTemplate
	//
	//     "uid=%u,%d"
	//
	// with the username "johndoe" and basename "ou=People,dc=example,dc=com" would attempt
	// to bind as
	//
	//     "uid=johndoe,ou=People,dc=example,dc=com"
	//
	BindTemplate string `json:"bindTemplate"`

	// DEPRICATED fields that exist for backward compatibility.
	// Use "host" instead of "ServerHost" and "ServerPort"
	ServerHost string        `json:"serverHost"`
	ServerPort uint16        `json:"serverPort"`
	Timeout    time.Duration `json:"timeout"`
}

func (cfg *LDAPConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *LDAPConnectorConfig) ConnectorType() string {
	return LDAPConnectorType
}

type LDAPConnector struct {
	id        string
	namespace url.URL
	loginFunc oidc.LoginFunc
	loginTpl  *template.Template

	baseDN         string
	nameAttribute  string
	emailAttribute string

	searchBeforeAuth  bool
	searchFilter      string
	searchScope       int
	searchBindDN      string
	searchBindPw      string
	searchGroupFilter string

	bindTemplate string

	ldapPool *LDAPPool
}

const defaultPoolCheckTimer = 7200 * time.Second

func (cfg *LDAPConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)
	tpl := tpls.Lookup(LDAPLoginPageTemplateName)
	if tpl == nil {
		return nil, fmt.Errorf("unable to find necessary HTML template")
	}

	if cfg.UseTLS && cfg.UseSSL {
		return nil, fmt.Errorf("Invalid configuration. useTLS and useSSL are mutual exclusive.")
	}

	if len(cfg.CertFile) > 0 && len(cfg.KeyFile) == 0 {
		return nil, fmt.Errorf("Invalid configuration. Both certFile and keyFile must be specified.")
	}

	// Set default values
	if cfg.NameAttribute == "" {
		cfg.NameAttribute = "cn"
	}
	if cfg.EmailAttribute == "" {
		cfg.EmailAttribute = "mail"
	}
	if cfg.MaxIdleConn > 0 {
		cfg.MaxIdleConn = 5
	}
	if cfg.BindTemplate == "" {
		cfg.BindTemplate = "uid=%u,%b"
	} else if cfg.SearchBeforeAuth {
		log.Warningf("bindTemplate not used when searchBeforeAuth specified.")
	}
	searchScope := ldap.ScopeWholeSubtree
	if cfg.SearchScope != "" {
		switch {
		case strings.EqualFold(cfg.SearchScope, "BASE"):
			searchScope = ldap.ScopeBaseObject
		case strings.EqualFold(cfg.SearchScope, "ONE"):
			searchScope = ldap.ScopeSingleLevel
		case strings.EqualFold(cfg.SearchScope, "SUB"):
			searchScope = ldap.ScopeWholeSubtree
		default:
			return nil, fmt.Errorf("Invalid value for searchScope: '%v'. Must be one of 'base', 'one' or 'sub'.", cfg.SearchScope)
		}
	}

	if cfg.Host == "" {
		if cfg.ServerHost == "" {
			return nil, errors.New("no host provided")
		}
		// For backward compatibility construct host form old fields.
		cfg.Host = fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	}

	host, _, err := net.SplitHostPort(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("host is not of form 'host:port': %v", err)
	}

	tlsConfig := &tls.Config{ServerName: host}

	if (cfg.UseTLS || cfg.UseSSL) && len(cfg.CaFile) > 0 {
		buf, err := ioutil.ReadFile(cfg.CaFile)
		if err != nil {
			return nil, err
		}

		rootCertPool := x509.NewCertPool()
		ok := rootCertPool.AppendCertsFromPEM(buf)
		if ok {
			tlsConfig.RootCAs = rootCertPool
		} else {
			return nil, fmt.Errorf("%v: Unable to parse certificate data.", cfg.CaFile)
		}
	}

	if (cfg.UseTLS || cfg.UseSSL) && len(cfg.CertFile) > 0 && len(cfg.KeyFile) > 0 {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	idpc := &LDAPConnector{
		id:                cfg.ID,
		namespace:         ns,
		loginFunc:         lf,
		loginTpl:          tpl,
		baseDN:            cfg.BaseDN,
		nameAttribute:     cfg.NameAttribute,
		emailAttribute:    cfg.EmailAttribute,
		searchBeforeAuth:  cfg.SearchBeforeAuth,
		searchFilter:      cfg.SearchFilter,
		searchGroupFilter: cfg.SearchGroupFilter,
		searchScope:       searchScope,
		searchBindDN:      cfg.SearchBindDN,
		searchBindPw:      cfg.SearchBindPw,
		bindTemplate:      cfg.BindTemplate,
		ldapPool: &LDAPPool{
			MaxIdleConn:    cfg.MaxIdleConn,
			PoolCheckTimer: defaultPoolCheckTimer,
			Host:           cfg.Host,
			UseTLS:         cfg.UseTLS,
			UseSSL:         cfg.UseSSL,
			TLSConfig:      tlsConfig,
		},
	}

	return idpc, nil
}

func (c *LDAPConnector) ID() string {
	return c.id
}

func (c *LDAPConnector) Healthy() error {
	return c.ldapPool.Do(func(c *ldap.Conn) error {
		// Attempt an anonymous bind.
		return c.Bind("", "")
	})
}

func (c *LDAPConnector) LoginURL(sessionKey, prompt string) (string, error) {
	q := url.Values{}
	q.Set("session_key", sessionKey)
	q.Set("prompt", prompt)
	enc := q.Encode()

	return path.Join(c.namespace.Path, "login") + "?" + enc, nil
}

func (c *LDAPConnector) Register(mux *http.ServeMux, errorURL url.URL) {
	route := path.Join(c.namespace.Path, "login")
	mux.Handle(route, handlePasswordLogin(c.loginFunc, c.loginTpl, c, route, errorURL))
}

func (c *LDAPConnector) Sync() chan struct{} {
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-time.After(c.ldapPool.PoolCheckTimer):
				alive, killed := c.ldapPool.CheckConnections()
				if alive > 0 {
					log.Infof("Connector ID=%v idle_conns=%v", c.id, alive)
				}
				if killed > 0 {
					log.Warningf("Connector ID=%v closed %v dead connections.", c.id, killed)
				}
			case <-stop:
				return
			}
		}
	}()
	return stop
}

func (c *LDAPConnector) TrustedEmailProvider() bool {
	return true
}

// A LDAPPool is a Connection Pool for LDAP connections. Use Do() to request connections
// from the pool.
type LDAPPool struct {
	m              sync.Mutex
	conns          map[*ldap.Conn]struct{}
	MaxIdleConn    int
	PoolCheckTimer time.Duration
	Host           string
	UseTLS         bool
	UseSSL         bool
	TLSConfig      *tls.Config
}

// Do runs a function which requires an LDAP connection.
//
// The connection will be unauthenticated with the server and should not be closed by f.
func (p *LDAPPool) Do(f func(conn *ldap.Conn) error) (err error) {
	conn := p.removeRandomConn()
	if conn == nil {
		conn, err = p.ldapConnect()
		if err != nil {
			return err
		}
	}
	defer p.put(conn)
	return f(conn)
}

// put makes a connection ready for re-use and puts it back into the pool. If the connection
// cannot be reused it is discarded. If there already are MaxIdleConn connections in the pool
// the connection is discarded.
func (p *LDAPPool) put(c *ldap.Conn) {
	p.m.Lock()
	if p.conns == nil {
		// First call to Put, initialize map
		p.conns = make(map[*ldap.Conn]struct{})
	}
	if len(p.conns)+1 > p.MaxIdleConn {
		p.m.Unlock()
		c.Close()
		return
	}
	p.m.Unlock()
	// drop to anonymous bind
	err := c.Bind("", "")
	if err != nil {
		// unsupported or disallowed, throw away connection
		log.Warningf("Unable to re-use LDAP Connection after failure to bind anonymously: %v", err)
		c.Close()
		return
	}
	p.m.Lock()
	p.conns[c] = struct{}{}
	p.m.Unlock()
}

// removeConn attempts to remove the provided connection from the pool. If removeConn returns false
// another routine is using the connection and the caller should discard the pointer.
func (p *LDAPPool) removeConn(conn *ldap.Conn) bool {
	p.m.Lock()
	_, ok := p.conns[conn]
	delete(p.conns, conn)
	p.m.Unlock()
	return ok
}

// removeRandomConn attempts to remove a random connection from the pool. If removeRandomConn
// returns nil the pool is empty.
func (p *LDAPPool) removeRandomConn() *ldap.Conn {
	p.m.Lock()
	defer p.m.Unlock()
	for conn := range p.conns {
		delete(p.conns, conn)
		return conn
	}
	return nil
}

// CheckConnections attempts to iterate over all the connections in the pool and check wheter
// they are alive or not. Live connections are put back into the pool, dead ones are discarded.
func (p *LDAPPool) CheckConnections() (int, int) {
	var conns []*ldap.Conn
	var alive, killed int

	// Get snapshot of connection-map while holding Lock
	p.m.Lock()
	for conn := range p.conns {
		conns = append(conns, conn)
	}
	p.m.Unlock()

	// Iterate over snapshot, Get and ping connections.
	// Put live connections back into pool, Close dead ones.
	for _, conn := range conns {
		ok := p.removeConn(conn)
		if ok {
			err := ldapPing(conn)
			if err == nil {
				p.put(conn)
				alive++
			} else {
				conn.Close()
				killed++
			}
		}
	}
	return alive, killed
}

func ldapPing(conn *ldap.Conn) error {
	// Query root DSE
	s := ldap.NewSearchRequest("", ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false, "(objectClass=*)", []string{}, nil)
	_, err := conn.Search(s)
	return err
}

func (p *LDAPPool) ldapConnect() (*ldap.Conn, error) {
	var err error
	var ldapConn *ldap.Conn

	if p.UseSSL {
		ldapConn, err = ldap.DialTLS("tcp", p.Host, p.TLSConfig)
		if err != nil {
			return nil, err
		}
	} else {
		ldapConn, err = ldap.Dial("tcp", p.Host)
		if err != nil {
			return nil, err
		}
		if p.UseTLS {
			err = ldapConn.StartTLS(p.TLSConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	return ldapConn, err
}

// invalidBindCredentials determines if a bind error was the result of invalid
// credentials.
func invalidBindCredentials(err error) bool {
	ldapErr, ok := err.(*ldap.Error)
	if ok {
		return false
	}
	return ldapErr.ResultCode == ldap.LDAPResultInvalidCredentials
}

func (c *LDAPConnector) formatDN(template, username string) string {
	result := template
	result = strings.Replace(result, "%u", ldap.EscapeFilter(username), -1)
	result = strings.Replace(result, "%b", c.baseDN, -1)

	return result
}

func (c *LDAPConnector) Groups(fullUserID string) ([]string, error) {
	if !c.searchBeforeAuth {
		return nil, fmt.Errorf("cannot search without service account")
	}
	if c.searchGroupFilter == "" {
		return nil, fmt.Errorf("no group filter specified")
	}

	var groups []string
	err := c.ldapPool.Do(func(conn *ldap.Conn) error {
		if err := conn.Bind(c.searchBindDN, c.searchBindPw); err != nil {
			if !invalidBindCredentials(err) {
				log.Errorf("failed to connect to LDAP for search bind: %v", err)
			}
			return fmt.Errorf("failed to bind: %v", err)
		}

		req := &ldap.SearchRequest{
			BaseDN: c.baseDN,
			Scope:  c.searchScope,
			Filter: c.formatDN(c.searchGroupFilter, fullUserID),
		}
		resp, err := conn.Search(req)
		if err != nil {
			return fmt.Errorf("search failed: %v", err)
		}
		groups = make([]string, len(resp.Entries))
		for i, entry := range resp.Entries {
			groups[i] = entry.DN
		}
		return nil
	})
	return groups, err
}

func (c *LDAPConnector) Identity(username, password string) (*oidc.Identity, error) {
	var (
		identity *oidc.Identity
		err      error
	)
	if c.searchBeforeAuth {
		err = c.ldapPool.Do(func(conn *ldap.Conn) error {
			if err := conn.Bind(c.searchBindDN, c.searchBindPw); err != nil {
				if !invalidBindCredentials(err) {
					log.Errorf("failed to connect to LDAP for search bind: %v", err)
				}
				return fmt.Errorf("failed to bind: %v", err)
			}

			filter := c.formatDN(c.searchFilter, username)
			req := &ldap.SearchRequest{
				BaseDN:     c.baseDN,
				Scope:      c.searchScope,
				Filter:     filter,
				Attributes: []string{c.nameAttribute, c.emailAttribute},
			}
			resp, err := conn.Search(req)
			if err != nil {
				return fmt.Errorf("search failed: %v", err)
			}
			switch len(resp.Entries) {
			case 0:
				return errors.New("user not found by search")
			case 1:
			default:
				// For now reject searches that return multiple entries to avoid ambiguity.
				log.Errorf("LDAP search %q returned %d entries. Must disambiguate searchFilter.", filter, len(resp.Entries))
				return errors.New("search returned multiple entries")
			}

			entry := resp.Entries[0]
			email := entry.GetAttributeValue(c.emailAttribute)
			if email == "" {
				return fmt.Errorf("no email attribute found")
			}

			identity = &oidc.Identity{
				ID:    entry.DN,
				Name:  entry.GetAttributeValue(c.nameAttribute),
				Email: email,
			}

			// Attempt to bind as the end user.
			return conn.Bind(entry.DN, password)
		})
	} else {
		err = c.ldapPool.Do(func(conn *ldap.Conn) error {
			userBindDN := c.formatDN(c.bindTemplate, username)
			if err := conn.Bind(userBindDN, password); err != nil {
				if !invalidBindCredentials(err) {
					log.Errorf("failed to connect to LDAP for search bind: %v", err)
				}
				return fmt.Errorf("failed to bind: %v", err)
			}

			req := &ldap.SearchRequest{
				BaseDN: userBindDN,
				Scope:  ldap.ScopeBaseObject, // Only attempt to
				Filter: "(objectClass=*)",
			}
			resp, err := conn.Search(req)
			if err != nil {
				return fmt.Errorf("search failed: %v", err)
			}
			if len(resp.Entries) == 0 {
				// Are there cases were a user wouldn't be able to see their own entity?
				return fmt.Errorf("user not found by search")
			}
			entry := resp.Entries[0]
			email := entry.GetAttributeValue(c.emailAttribute)
			if email == "" {
				return fmt.Errorf("no email attribute found")
			}

			identity = &oidc.Identity{
				ID:    entry.DN,
				Name:  entry.GetAttributeValue(c.nameAttribute),
				Email: email,
			}
			return nil
		})
	}
	if err != nil {
		return nil, err
	}
	return identity, nil
}
