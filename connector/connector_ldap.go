package connector

import (
	"crypto/tls"
	"crypto/x509"

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
}

type LDAPConnectorConfig struct {
	ID                   string        `json:"id"`
	ServerHost           string        `json:"serverHost"`
	ServerPort           uint16        `json:"serverPort"`
	Timeout              time.Duration `json:"timeout"`
	UseTLS               bool          `json:"useTLS"`
	UseSSL               bool          `json:"useSSL"`
	CertFile             string        `json:"certFile"`
	KeyFile              string        `json:"keyFile"`
	CaFile               string        `json:"caFile"`
	SkipCertVerification bool          `json:"skipCertVerification"`
	MaxIdleConn          int           `json:"maxIdleConn"`
	BaseDN               string        `json:"baseDN"`
	NameAttribute        string        `json:"nameAttribute"`
	EmailAttribute       string        `json:"emailAttribute"`
	SearchBeforeAuth     bool          `json:"searchBeforeAuth"`
	SearchFilter         string        `json:"searchFilter"`
	SearchScope          string        `json:"searchScope"`
	SearchBindDN         string        `json:"searchBindDN"`
	SearchBindPw         string        `json:"searchBindPw"`
	BindTemplate         string        `json:"bindTemplate"`
	TrustedEmailProvider bool          `json:"trustedEmailProvider"`
}

func (cfg *LDAPConnectorConfig) ConnectorID() string {
	return cfg.ID
}

func (cfg *LDAPConnectorConfig) ConnectorType() string {
	return LDAPConnectorType
}

type LDAPConnector struct {
	id                   string
	idp                  *LDAPIdentityProvider
	namespace            url.URL
	trustedEmailProvider bool
	loginFunc            oidc.LoginFunc
	loginTpl             *template.Template
}

func (cfg *LDAPConnectorConfig) Connector(ns url.URL, lf oidc.LoginFunc, tpls *template.Template) (Connector, error) {
	ns.Path = path.Join(ns.Path, httpPathCallback)
	tpl := tpls.Lookup(LDAPLoginPageTemplateName)
	if tpl == nil {
		return nil, fmt.Errorf("unable to find necessary HTML template")
	}

	// defaults
	const defaultNameAttribute = "cn"
	const defaultEmailAttribute = "mail"
	const defaultBindTemplate = "uid=%u,%b"
	const defaultSearchScope = ldap.ScopeWholeSubtree
	const defaultMaxIdleConns = 5
	const defaultPoolCheckTimer = 7200 * time.Second

	if cfg.UseTLS && cfg.UseSSL {
		return nil, fmt.Errorf("Invalid configuration. useTLS and useSSL are mutual exclusive.")
	}

	if len(cfg.CertFile) > 0 && len(cfg.KeyFile) == 0 {
		return nil, fmt.Errorf("Invalid configuration. Both certFile and keyFile must be specified.")
	}

	nameAttribute := defaultNameAttribute
	if len(cfg.NameAttribute) > 0 {
		nameAttribute = cfg.NameAttribute
	}

	emailAttribute := defaultEmailAttribute
	if len(cfg.EmailAttribute) > 0 {
		emailAttribute = cfg.EmailAttribute
	}

	bindTemplate := defaultBindTemplate
	if len(cfg.BindTemplate) > 0 {
		if cfg.SearchBeforeAuth {
			log.Warningf("bindTemplate not used when searchBeforeAuth specified.")
		}
		bindTemplate = cfg.BindTemplate
	}

	searchScope := defaultSearchScope
	if len(cfg.SearchScope) > 0 {
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

	if cfg.Timeout != 0 {
		ldap.DefaultTimeout = cfg.Timeout * time.Millisecond
	}

	tlsConfig := &tls.Config{
		ServerName:         cfg.ServerHost,
		InsecureSkipVerify: cfg.SkipCertVerification,
	}

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

	maxIdleConn := defaultMaxIdleConns
	if cfg.MaxIdleConn > 0 {
		maxIdleConn = cfg.MaxIdleConn
	}

	ldapPool := &LDAPPool{
		MaxIdleConn:    maxIdleConn,
		PoolCheckTimer: defaultPoolCheckTimer,
		ServerHost:     cfg.ServerHost,
		ServerPort:     cfg.ServerPort,
		UseTLS:         cfg.UseTLS,
		UseSSL:         cfg.UseSSL,
		TLSConfig:      tlsConfig,
	}

	idp := &LDAPIdentityProvider{
		baseDN:           cfg.BaseDN,
		nameAttribute:    nameAttribute,
		emailAttribute:   emailAttribute,
		searchBeforeAuth: cfg.SearchBeforeAuth,
		searchFilter:     cfg.SearchFilter,
		searchScope:      searchScope,
		searchBindDN:     cfg.SearchBindDN,
		searchBindPw:     cfg.SearchBindPw,
		bindTemplate:     bindTemplate,
		ldapPool:         ldapPool,
	}

	idpc := &LDAPConnector{
		id:                   cfg.ID,
		idp:                  idp,
		namespace:            ns,
		trustedEmailProvider: cfg.TrustedEmailProvider,
		loginFunc:            lf,
		loginTpl:             tpl,
	}

	return idpc, nil
}

func (c *LDAPConnector) ID() string {
	return c.id
}

func (c *LDAPConnector) Healthy() error {
	ldapConn, err := c.idp.ldapPool.Acquire()
	if err == nil {
		c.idp.ldapPool.Put(ldapConn)
	}
	return err
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
	mux.Handle(route, handleLoginFunc(c.loginFunc, c.loginTpl, c.idp, route, errorURL))
}

func (c *LDAPConnector) Sync() chan struct{} {
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-time.After(c.idp.ldapPool.PoolCheckTimer):
				alive, killed := c.idp.ldapPool.CheckConnections()
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
	return c.trustedEmailProvider
}

// A LDAPPool is a Connection Pool for LDAP connections
// Initialize exported fields and use Acquire() to get a connection.
// Use Put() to put it back into the pool.
type LDAPPool struct {
	m              sync.Mutex
	conns          map[*ldap.Conn]struct{}
	MaxIdleConn    int
	PoolCheckTimer time.Duration
	ServerHost     string
	ServerPort     uint16
	UseTLS         bool
	UseSSL         bool
	TLSConfig      *tls.Config
}

// Acquire removes and returns a random connection from the pool. A new connection is returned
// if there are no connections available in the pool.
func (p *LDAPPool) Acquire() (*ldap.Conn, error) {
	conn := p.removeRandomConn()
	if conn != nil {
		return conn, nil
	}
	return p.ldapConnect()
}

// Put makes a connection ready for re-use and puts it back into the pool. If the connection
// cannot be reused it is discarded. If there already are MaxIdleConn connections in the pool
// the connection is discarded.
func (p *LDAPPool) Put(c *ldap.Conn) {
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
				p.Put(conn)
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

type LDAPIdentityProvider struct {
	baseDN           string
	nameAttribute    string
	emailAttribute   string
	searchBeforeAuth bool
	searchFilter     string
	searchScope      int
	searchBindDN     string
	searchBindPw     string
	bindTemplate     string
	ldapPool         *LDAPPool
}

func (p *LDAPPool) ldapConnect() (*ldap.Conn, error) {
	var err error
	var ldapConn *ldap.Conn

	log.Debugf("LDAPConnect()")
	if p.UseSSL {
		ldapConn, err = ldap.DialTLS("tcp", fmt.Sprintf("%s:%d", p.ServerHost, p.ServerPort), p.TLSConfig)
		if err != nil {
			return nil, err
		}
	} else {
		ldapConn, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", p.ServerHost, p.ServerPort))
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

func (m *LDAPIdentityProvider) ParseString(template, username string) string {
	result := template
	result = strings.Replace(result, "%u", username, -1)
	result = strings.Replace(result, "%b", m.baseDN, -1)

	return result
}

func (m *LDAPIdentityProvider) Identity(username, password string) (*oidc.Identity, error) {
	var err error
	var bindDN, ldapUid, ldapName, ldapEmail string
	var ldapConn *ldap.Conn

	ldapConn, err = m.ldapPool.Acquire()
	if err != nil {
		return nil, err
	}
	defer m.ldapPool.Put(ldapConn)

	if m.searchBeforeAuth {
		err = ldapConn.Bind(m.searchBindDN, m.searchBindPw)
		if err != nil {
			return nil, err
		}

		filter := m.ParseString(m.searchFilter, username)

		attributes := []string{
			m.nameAttribute,
			m.emailAttribute,
		}

		s := ldap.NewSearchRequest(m.baseDN, m.searchScope, ldap.NeverDerefAliases, 0, 0, false, filter, attributes, nil)

		sr, err := ldapConn.Search(s)
		if err != nil {
			return nil, err
		}
		if len(sr.Entries) == 0 {
			err = fmt.Errorf("Search returned no match. filter='%v' base='%v'", filter, m.baseDN)
			return nil, err
		}

		bindDN = sr.Entries[0].DN
		ldapName = sr.Entries[0].GetAttributeValue(m.nameAttribute)
		ldapEmail = sr.Entries[0].GetAttributeValue(m.emailAttribute)

		// prepare LDAP connection for bind as user
		m.ldapPool.Put(ldapConn)
		ldapConn, err = m.ldapPool.Acquire()
		if err != nil {
			return nil, err
		}
	} else {
		bindDN = m.ParseString(m.bindTemplate, username)
	}

	// authenticate user
	err = ldapConn.Bind(bindDN, password)
	if err != nil {
		return nil, err
	}

	ldapUid = bindDN

	return &oidc.Identity{
		ID:    ldapUid,
		Name:  ldapName,
		Email: ldapEmail,
	}, nil
}
