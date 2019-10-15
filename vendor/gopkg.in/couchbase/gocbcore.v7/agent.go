// Package gocbcore implements methods for low-level communication
// with a Couchbase Server cluster.
package gocbcore

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/http2"
	"gopkg.in/couchbaselabs/gocbconnstr.v1"
)

// Agent represents the base client handling connections to a Couchbase Server.
// This is used internally by the higher level classes for communicating with the cluster,
// it can also be used to perform more advanced operations with a cluster.
type Agent struct {
	clientId             string
	userString           string
	auth                 AuthProvider
	bucket               string
	tlsConfig            *tls.Config
	initFn               memdInitFunc
	networkType          string
	useMutationTokens    bool
	useKvErrorMaps       bool
	useEnhancedErrors    bool
	useCompression       bool
	useDurations         bool
	disableDecompression bool

	compressionMinSize  int
	compressionMinRatio float64

	closeNotify       chan struct{}
	cccpLooperDoneSig chan struct{}
	httpLooperDoneSig chan struct{}

	configLock  sync.Mutex
	routingInfo routeDataPtr
	kvErrorMap  kvErrorMapPtr
	numVbuckets int

	tracer           opentracing.Tracer
	noRootTraceSpans bool

	serverFailuresLock sync.Mutex
	serverFailures     map[string]time.Time

	httpCli *http.Client

	confHttpRedialPeriod time.Duration
	confHttpRetryDelay   time.Duration
	confCccpMaxWait      time.Duration
	confCccpPollPeriod   time.Duration

	serverConnectTimeout time.Duration
	serverWaitTimeout    time.Duration
	nmvRetryDelay        time.Duration
	kvPoolSize           int
	maxQueueSize         int

	zombieLock      sync.RWMutex
	zombieOps       []*zombieLogEntry
	useZombieLogger bool

	dcpPriority  DcpAgentPriority
	useDcpExpiry bool
}

// ServerConnectTimeout gets the timeout for each server connection, including all authentication steps.
func (agent *Agent) ServerConnectTimeout() time.Duration {
	return agent.serverConnectTimeout
}

// SetServerConnectTimeout sets the timeout for each server connection.
func (agent *Agent) SetServerConnectTimeout(timeout time.Duration) {
	agent.serverConnectTimeout = timeout
}

// HttpClient returns a pre-configured HTTP Client for communicating with
// Couchbase Server.  You must still specify authentication information
// for any dispatched requests.
func (agent *Agent) HttpClient() *http.Client {
	return agent.httpCli
}

func (agent *Agent) getErrorMap() *kvErrorMap {
	return agent.kvErrorMap.Get()
}

// AuthFunc is invoked by the agent to authenticate a client.
type AuthFunc func(client AuthClient, deadline time.Time) error

// AgentConfig specifies the configuration options for creation of an Agent.
type AgentConfig struct {
	UserString           string
	MemdAddrs            []string
	HttpAddrs            []string
	TlsConfig            *tls.Config
	BucketName           string
	NetworkType          string
	AuthHandler          AuthFunc
	Auth                 AuthProvider
	UseMutationTokens    bool
	UseKvErrorMaps       bool
	UseEnhancedErrors    bool
	UseCompression       bool
	UseDurations         bool
	DisableDecompression bool

	CompressionMinSize  int
	CompressionMinRatio float64

	HttpRedialPeriod time.Duration
	HttpRetryDelay   time.Duration
	CccpMaxWait      time.Duration
	CccpPollPeriod   time.Duration

	ConnectTimeout       time.Duration
	ServerConnectTimeout time.Duration
	NmvRetryDelay        time.Duration
	KvPoolSize           int
	MaxQueueSize         int

	HttpMaxIdleConns        int
	HttpMaxIdleConnsPerHost int
	HttpIdleConnTimeout     time.Duration

	Tracer                 opentracing.Tracer
	NoRootTraceSpans       bool
	UseZombieLogger        bool
	ZombieLoggerInterval   time.Duration
	ZombieLoggerSampleSize int

	DcpAgentPriority DcpAgentPriority
	UseDcpExpiry     bool

	// Username specifies the username to use when connecting.
	// DEPRECATED
	Username string

	// Password specifies the password to use when connecting.
	// DEPRECATED
	Password string
}

// FromConnStr populates the AgentConfig with information from a
// Couchbase Connection String.
// Supported options are:
//   cacertpath (string) - Path to the CA certificate
//   certpath (string) - Path to your authentication certificate
//   keypath (string) - Path to your authentication key
//   config_total_timeout (int) - Maximum period to attempt to connect to cluster in ms.
//   config_node_timeout (int) - Maximum period to attempt to connect to a node in ms.
//   http_redial_period (int) - Maximum period to keep HTTP config connections open in ms.
//   http_retry_delay (int) - Period to wait between retrying nodes for HTTP config in ms.
//   config_poll_floor_interval (int) - Minimum time to wait between fetching configs via CCCP in ms.
//   config_poll_interval (int) - Period to wait between CCCP config polling in ms.
//   kv_pool_size (int) - The number of connections to establish per node.
//   max_queue_size (int) - The maximum size of the operation queues per node.
//   use_kverrmaps (bool) - Whether to enable error maps from the server.
//   use_enhanced_errors (bool) - Whether to enable enhanced error information.
//   fetch_mutation_tokens (bool) - Whether to fetch mutation tokens for operations.
//   compression (bool) - Whether to enable network-wise compression of documents.
//   compression_min_size (int) - The minimal size of the document to consider compression.
//   compression_min_ratio (float64) - The minimal compress ratio (compressed / original) for the document to be sent compressed.
//   server_duration (bool) - Whether to enable fetching server operation durations.
//   http_max_idle_conns (int) - Maximum number of idle http connections in the pool.
//   http_max_idle_conns_per_host (int) - Maximum number of idle http connections in the pool per host.
//   http_idle_conn_timeout (int) - Maximum length of time for an idle connection to stay in the pool in ms.
//   network (string) - The network type to use
func (config *AgentConfig) FromConnStr(connStr string) error {
	baseSpec, err := gocbconnstr.Parse(connStr)
	if err != nil {
		return err
	}

	spec, err := gocbconnstr.Resolve(baseSpec)
	if err != nil {
		return err
	}

	fetchOption := func(name string) (string, bool) {
		optValue := spec.Options[name]
		if len(optValue) == 0 {
			return "", false
		}
		return optValue[len(optValue)-1], true
	}

	// Grab the resolved hostnames into a set of string arrays
	var httpHosts []string
	for _, specHost := range spec.HttpHosts {
		httpHosts = append(httpHosts, fmt.Sprintf("%s:%d", specHost.Host, specHost.Port))
	}

	var memdHosts []string
	for _, specHost := range spec.MemdHosts {
		memdHosts = append(memdHosts, fmt.Sprintf("%s:%d", specHost.Host, specHost.Port))
	}

	// Get bootstrap_on option to determine which, if any, of the bootstrap nodes should be cleared
	switch val, _ := fetchOption("bootstrap_on"); val {
	case "http":
		memdHosts = nil
		if len(httpHosts) == 0 {
			return errors.New("bootstrap_on=http but no HTTP hosts in connection string")
		}
	case "cccp":
		httpHosts = nil
		if len(memdHosts) == 0 {
			return errors.New("bootstrap_on=cccp but no CCCP/Memcached hosts in connection string")
		}
	case "both":
	case "":
		// Do nothing
		break
	default:
		return errors.New("bootstrap_on={http,cccp,both}")
	}
	config.MemdAddrs = memdHosts
	config.HttpAddrs = httpHosts

	var tlsConfig *tls.Config
	if spec.UseSsl {
		var certpath string
		var keypath string
		var cacertpaths []string

		if len(spec.Options["cacertpath"]) > 0 || len(spec.Options["keypath"]) > 0 {
			cacertpaths = spec.Options["cacertpath"]
			certpath, _ = fetchOption("certpath")
			keypath, _ = fetchOption("keypath")
		} else {
			cacertpaths = spec.Options["certpath"]
		}

		tlsConfig = &tls.Config{}

		if len(cacertpaths) > 0 {
			roots := x509.NewCertPool()

			for _, path := range cacertpaths {
				cacert, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				ok := roots.AppendCertsFromPEM(cacert)
				if !ok {
					return ErrInvalidCert
				}
			}

			tlsConfig.RootCAs = roots
		} else {
			tlsConfig.InsecureSkipVerify = true
		}

		if certpath != "" && keypath != "" {
			cert, err := tls.LoadX509KeyPair(certpath, keypath)
			if err != nil {
				return err
			}

			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}
	config.TlsConfig = tlsConfig

	if spec.Bucket != "" {
		config.BucketName = spec.Bucket
	}

	if valStr, ok := fetchOption("config_total_timeout"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("config_total_timeout option must be a number")
		}
		config.ConnectTimeout = time.Duration(val) * time.Millisecond
	}

	if valStr, ok := fetchOption("config_node_timeout"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("config_node_timeout option must be a number")
		}
		config.ServerConnectTimeout = time.Duration(val) * time.Millisecond
	}

	// This option is experimental
	if valStr, ok := fetchOption("http_redial_period"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("http redial period option must be a number")
		}
		config.HttpRedialPeriod = time.Duration(val) * time.Millisecond
	}

	// This option is experimental
	if valStr, ok := fetchOption("http_retry_delay"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("http retry delay option must be a number")
		}
		config.HttpRetryDelay = time.Duration(val) * time.Millisecond
	}

	// This option is deprecated (see config_poll_floor_interval)
	if valStr, ok := fetchOption("cccp_max_wait"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("cccp max wait option must be a number")
		}
		config.CccpMaxWait = time.Duration(val) * time.Millisecond
	}

	if valStr, ok := fetchOption("config_poll_floor_interval"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("config pool floor interval option must be a number")
		}
		config.CccpMaxWait = time.Duration(val) * time.Millisecond
	}

	// This option is deprecated (see config_poll_interval)
	if valStr, ok := fetchOption("cccp_poll_period"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("cccp pool period option must be a number")
		}
		config.CccpPollPeriod = time.Duration(val) * time.Millisecond
	}

	if valStr, ok := fetchOption("config_poll_interval"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("config pool interval option must be a number")
		}
		config.CccpPollPeriod = time.Duration(val) * time.Millisecond
	}

	// This option is experimental
	if valStr, ok := fetchOption("kv_pool_size"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("kv pool size option must be a number")
		}
		config.KvPoolSize = int(val)
	}

	// This option is experimental
	if valStr, ok := fetchOption("max_queue_size"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("max queue size option must be a number")
		}
		config.MaxQueueSize = int(val)
	}

	if valStr, ok := fetchOption("use_kverrmaps"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("use_kverrmaps option must be a boolean")
		}
		config.UseKvErrorMaps = val
	}

	if valStr, ok := fetchOption("use_enhanced_errors"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("use_enhanced_errors option must be a boolean")
		}
		config.UseEnhancedErrors = val
	}

	if valStr, ok := fetchOption("fetch_mutation_tokens"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("fetch_mutation_tokens option must be a boolean")
		}
		config.UseMutationTokens = val
	}

	if valStr, ok := fetchOption("compression"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("compression option must be a boolean")
		}
		config.UseCompression = val
	}

	if valStr, ok := fetchOption("compression_min_size"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("compression_min_size option must be an int")
		}
		config.CompressionMinSize = int(val)
	}

	if valStr, ok := fetchOption("compression_min_ratio"); ok {
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return fmt.Errorf("compression_min_size option must be an int")
		}
		config.CompressionMinRatio = val
	}

	if valStr, ok := fetchOption("server_duration"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("server_duration option must be a boolean")
		}
		config.UseDurations = val
	}

	if valStr, ok := fetchOption("http_max_idle_conns"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("http max idle connections option must be a number")
		}
		config.HttpMaxIdleConns = int(val)
	}

	if valStr, ok := fetchOption("http_max_idle_conns_per_host"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("http max idle connections per host option must be a number")
		}
		config.HttpMaxIdleConnsPerHost = int(val)
	}

	if valStr, ok := fetchOption("http_idle_conn_timeout"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("http idle connection timeout option must be a number")
		}
		config.HttpIdleConnTimeout = time.Duration(val) * time.Millisecond
	}

	if valStr, ok := fetchOption("orphaned_response_logging"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("orphaned_response_logging option must be a boolean")
		}
		config.UseZombieLogger = val
	}

	if valStr, ok := fetchOption("orphaned_response_logging_interval"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("orphaned_response_logging_interval option must be a number")
		}
		config.ZombieLoggerInterval = time.Duration(val) * time.Millisecond
	}

	if valStr, ok := fetchOption("orphaned_response_logging_sample_size"); ok {
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			return fmt.Errorf("orphaned_response_logging_sample_size option must be a number")
		}
		config.ZombieLoggerSampleSize = int(val)
	}

	if valStr, ok := fetchOption("network"); ok {
		if valStr == "default" {
			valStr = ""
		}

		config.NetworkType = valStr
	}

	if valStr, ok := fetchOption("dcp_priority"); ok {
		var priority DcpAgentPriority
		switch valStr {
		case "":
			priority = DcpAgentPriorityLow
		case "low":
			priority = DcpAgentPriorityLow
		case "medium":
			priority = DcpAgentPriorityMed
		case "high":
			priority = DcpAgentPriorityHigh
		default:
			return fmt.Errorf("dcp_priority must be one of low, medium or high")
		}
		config.DcpAgentPriority = priority
	}

	if valStr, ok := fetchOption("enable_expiry_opcode"); ok {
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return fmt.Errorf("enable_expiry_opcode option must be a boolean")
		}
		config.UseDcpExpiry = val
	}

	return nil
}

func makeDefaultAuthHandler(authProvider AuthProvider, bucketName string) AuthFunc {
	return func(client AuthClient, deadline time.Time) error {
		creds, err := getKvAuthCreds(authProvider, client.Address())
		if err != nil {
			return err
		}

		if creds.Username != "" || creds.Password != "" {
			if err := SaslAuthPlain(creds.Username, creds.Password, client, deadline); err != nil {
				return err
			}
		}

		if client.SupportsFeature(FeatureSelectBucket) {
			if err := client.ExecSelectBucket([]byte(bucketName), deadline); err != nil {
				return err
			}
		}

		return nil
	}
}

func normalizeAgentConfig(configIn *AgentConfig) *AgentConfig {
	config := *configIn

	// If the user does not provide an authentication provider, we should use
	// the deprecated username/password fields to create one.
	if config.Auth == nil {
		config.Auth = &PasswordAuthProvider{
			Username: config.Username,
			Password: config.Password,
		}
	}

	// TODO: The location of this happening is a bit strange
	if config.AuthHandler == nil {
		config.AuthHandler = makeDefaultAuthHandler(config.Auth, config.BucketName)
	}

	return &config
}

// CreateAgent creates an agent for performing normal operations.
func CreateAgent(configIn *AgentConfig) (*Agent, error) {
	config := normalizeAgentConfig(configIn)

	initFn := func(client *syncClient, deadline time.Time, agent *Agent) error {
		return config.AuthHandler(client, deadline)
	}

	return createAgent(config, initFn)
}

// CreateDcpAgent creates an agent for performing DCP operations.
func CreateDcpAgent(configIn *AgentConfig, dcpStreamName string, openFlags DcpOpenFlag) (*Agent, error) {
	config := normalizeAgentConfig(configIn)

	// We wrap the authorization system to force DCP channel opening
	//   as part of the "initialization" for any servers.
	initFn := func(client *syncClient, deadline time.Time, agent *Agent) error {
		if err := config.AuthHandler(client, deadline); err != nil {
			return err
		}

		if err := client.ExecOpenDcpConsumer(dcpStreamName, openFlags, deadline); err != nil {
			return err
		}
		if err := client.ExecEnableDcpNoop(180*time.Second, deadline); err != nil {
			return err
		}
		var priority string
		switch agent.dcpPriority {
		case DcpAgentPriorityLow:
			priority = "low"
		case DcpAgentPriorityMed:
			priority = "medium"
		case DcpAgentPriorityHigh:
			priority = "high"
		}
		if err := client.ExecDcpControl("set_priority", priority, deadline); err != nil {
			return err
		}
		if agent.useDcpExpiry {
			if err := client.ExecDcpControl("enable_expiry_opcode", "true", deadline); err != nil {
				return err
			}
		}
		if err := client.ExecEnableDcpClientEnd(deadline); err != nil {
			return err
		}
		return client.ExecEnableDcpBufferAck(8*1024*1024, deadline)
	}

	return createAgent(config, initFn)
}

func createAgent(config *AgentConfig, initFn memdInitFunc) (*Agent, error) {
	// TODO(brett19): Put all configurable options in the AgentConfig

	logDebugf("SDK Version: gocb/%s", goCbCoreVersionStr)
	logDebugf("Creating new agent: %+v", config)

	httpTransport := &http.Transport{
		TLSClientConfig: config.TlsConfig,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        config.HttpMaxIdleConns,
		MaxIdleConnsPerHost: config.HttpMaxIdleConnsPerHost,
		IdleConnTimeout:     config.HttpIdleConnTimeout,
	}
	err := http2.ConfigureTransport(httpTransport)
	if err != nil {
		logDebugf("failed to configure http2: %s", err)
	}

	tracer := config.Tracer
	if tracer == nil {
		tracer = opentracing.NoopTracer{}
	}

	c := &Agent{
		clientId:    formatCbUid(randomCbUid()),
		userString:  config.UserString,
		bucket:      config.BucketName,
		auth:        config.Auth,
		tlsConfig:   config.TlsConfig,
		initFn:      initFn,
		networkType: config.NetworkType,
		httpCli: &http.Client{
			Transport: httpTransport,
		},
		closeNotify:          make(chan struct{}),
		useZombieLogger:      config.UseZombieLogger,
		tracer:               tracer,
		useMutationTokens:    config.UseMutationTokens,
		useKvErrorMaps:       config.UseKvErrorMaps,
		useEnhancedErrors:    config.UseEnhancedErrors,
		useCompression:       config.UseCompression,
		compressionMinSize:   32,
		compressionMinRatio:  0.83,
		useDurations:         config.UseDurations,
		noRootTraceSpans:     config.NoRootTraceSpans,
		serverFailures:       make(map[string]time.Time),
		serverConnectTimeout: 7000 * time.Millisecond,
		serverWaitTimeout:    5 * time.Second,
		nmvRetryDelay:        100 * time.Millisecond,
		kvPoolSize:           1,
		maxQueueSize:         2048,
		confHttpRetryDelay:   10 * time.Second,
		confHttpRedialPeriod: 10 * time.Second,
		confCccpMaxWait:      3 * time.Second,
		confCccpPollPeriod:   2500 * time.Millisecond,
		dcpPriority:          config.DcpAgentPriority,
		disableDecompression: config.DisableDecompression,
		useDcpExpiry:         config.UseDcpExpiry,
	}

	connectTimeout := 60000 * time.Millisecond
	if config.ConnectTimeout > 0 {
		connectTimeout = config.ConnectTimeout
	}

	if config.ServerConnectTimeout > 0 {
		c.serverConnectTimeout = config.ServerConnectTimeout
	}
	if config.NmvRetryDelay > 0 {
		c.nmvRetryDelay = config.NmvRetryDelay
	}
	if config.KvPoolSize > 0 {
		c.kvPoolSize = config.KvPoolSize
	}
	if config.MaxQueueSize > 0 {
		c.maxQueueSize = config.MaxQueueSize
	}
	if config.HttpRetryDelay > 0 {
		c.confHttpRetryDelay = config.HttpRetryDelay
	}
	if config.HttpRedialPeriod > 0 {
		c.confHttpRedialPeriod = config.HttpRedialPeriod
	}
	if config.CccpMaxWait > 0 {
		c.confCccpMaxWait = config.CccpMaxWait
	}
	if config.CccpPollPeriod > 0 {
		c.confCccpPollPeriod = config.CccpPollPeriod
	}
	if config.CompressionMinSize > 0 {
		c.compressionMinSize = config.CompressionMinSize
	}
	if config.CompressionMinRatio > 0 {
		c.compressionMinRatio = config.CompressionMinRatio
		if c.compressionMinRatio >= 1.0 {
			c.compressionMinRatio = 1.0
		}
	}

	deadline := time.Now().Add(connectTimeout)
	if err := c.connect(config.MemdAddrs, config.HttpAddrs, deadline); err != nil {
		return nil, err
	}

	if config.UseZombieLogger {
		zombieLoggerInterval := 10 * time.Second
		zombieLoggerSampleSize := 10
		if config.ZombieLoggerInterval > 0 {
			zombieLoggerInterval = config.ZombieLoggerInterval
		}
		if config.ZombieLoggerSampleSize > 0 {
			zombieLoggerSampleSize = config.ZombieLoggerSampleSize
		}
		// zombieOps must have a static capacity for its lifetime, the capacity should
		// never be altered so that it is consistent across the zombieLogger and
		// recordZombieResponse.
		c.zombieOps = make([]*zombieLogEntry, 0, zombieLoggerSampleSize)
		go c.zombieLogger(zombieLoggerInterval, zombieLoggerSampleSize)
	}

	return c, nil
}

func (agent *Agent) connect(memdAddrs, httpAddrs []string, deadline time.Time) error {
	logDebugf("Attempting to connect...")

	for _, thisHostPort := range memdAddrs {
		logDebugf("Trying server at %s", thisHostPort)

		srvDeadlineTm := time.Now().Add(agent.serverConnectTimeout)
		if srvDeadlineTm.After(deadline) {
			srvDeadlineTm = deadline
		}

		logDebugf("Trying to connect")
		client, err := agent.dialMemdClient(thisHostPort)
		if IsErrorStatus(err, StatusAuthError) ||
			IsErrorStatus(err, StatusAccessError) {
			return err
		} else if err != nil {
			logDebugf("Connecting failed! %v", err)
			continue
		}

		disconnectClient := func() {
			err := client.Close()
			if err != nil {
				logErrorf("Failed to shut down client connection (%s)", err)
			}
		}

		syncCli := syncClient{
			client: client,
		}

		logDebugf("Attempting to request CCCP configuration")
		cccpBytes, err := syncCli.ExecCccpRequest(srvDeadlineTm)
		if err != nil {
			logDebugf("Failed to retrieve CCCP config. %v", err)
			disconnectClient()
			continue
		}

		hostName, err := hostFromHostPort(thisHostPort)
		if err != nil {
			logErrorf("Failed to parse CCCP source address. %v", err)
			disconnectClient()
			continue
		}

		bk, err := parseConfig(cccpBytes, hostName)
		if err != nil {
			logDebugf("Failed to parse CCCP configuration. %v", err)
			disconnectClient()
			continue
		}

		if !bk.supportsCccp() {
			logDebugf("Bucket does not support CCCP")
			disconnectClient()
			break
		}

		routeCfg := agent.buildFirstRouteConfig(bk, thisHostPort)
		logDebugf("Using network type %s for connections", agent.networkType)
		if !routeCfg.IsValid() {
			logDebugf("Configuration was deemed invalid %+v", routeCfg)
			disconnectClient()
			continue
		}

		logDebugf("Successfully connected")

		// Build some fake routing data, this is used to indicate that
		//  client is "alive".  A nil routeData causes immediate shutdown.
		agent.routingInfo.Update(nil, &routeData{
			revId: -1,
		})

		// TODO(brett19): Save the client that we build for bootstrap
		disconnectClient()

		if routeCfg.vbMap != nil {
			agent.numVbuckets = routeCfg.vbMap.NumVbuckets()
		} else {
			agent.numVbuckets = 0
		}

		agent.applyConfig(routeCfg)

		agent.cccpLooperDoneSig = make(chan struct{})
		go agent.cccpLooper()

		return nil
	}

	signal := make(chan error, 1)

	var epList []string
	for _, hostPort := range httpAddrs {
		if !agent.IsSecure() {
			epList = append(epList, fmt.Sprintf("http://%s", hostPort))
		} else {
			epList = append(epList, fmt.Sprintf("https://%s", hostPort))
		}
	}
	agent.routingInfo.Update(nil, &routeData{
		revId:      -1,
		mgmtEpList: epList,
	})

	var routeCfg *routeConfig

	logDebugf("Starting HTTP looper! %v", epList)
	agent.httpLooperDoneSig = make(chan struct{})
	go agent.httpLooper(func(cfg *cfgBucket, srcServer string, err error) bool {
		if err != nil {
			signal <- err
			return true
		}

		newRouteCfg := agent.buildFirstRouteConfig(cfg, srcServer)
		if !newRouteCfg.IsValid() {
			// Something is invalid about this config, keep trying
			return false
		}

		routeCfg = newRouteCfg
		signal <- nil
		return true
	})

	err := <-signal
	if err != nil {
		return err
	}

	if routeCfg.vbMap != nil {
		agent.numVbuckets = routeCfg.vbMap.NumVbuckets()
	} else {
		agent.numVbuckets = 0
	}

	agent.applyConfig(routeCfg)

	return nil
}

func (agent *Agent) buildFirstRouteConfig(bk *cfgBucket, srcServer string) *routeConfig {
	if agent.networkType != "" && agent.networkType != "auto" {
		return buildRouteConfig(bk, agent.IsSecure(), agent.networkType, true)
	}

	defaultRouteConfig := buildRouteConfig(bk, agent.IsSecure(), "default", true)

	// First we check if the source server is from the defaults list
	srcInDefaultConfig := false
	for _, endpoint := range defaultRouteConfig.kvServerList {
		if endpoint == srcServer {
			srcInDefaultConfig = true
		}
	}
	for _, endpoint := range defaultRouteConfig.mgmtEpList {
		if endpoint == srcServer {
			srcInDefaultConfig = true
		}
	}
	if srcInDefaultConfig {
		agent.networkType = "default"
		return defaultRouteConfig
	}

	// Next lets see if we have an external config, if so, default to that
	externalRouteCfg := buildRouteConfig(bk, agent.IsSecure(), "external", true)
	if externalRouteCfg.IsValid() {
		agent.networkType = "external"
		return externalRouteCfg
	}

	// If all else fails, default to the implicit default config
	agent.networkType = "default"
	return defaultRouteConfig
}

// Close shuts down the agent, disconnecting from all servers and failing
// any outstanding operations with ErrShutdown.
func (agent *Agent) Close() error {
	agent.configLock.Lock()

	// Clear the routingInfo so no new operations are performed
	//   and retrieve the last active routing configuration
	routingInfo := agent.routingInfo.Clear()
	if routingInfo == nil {
		agent.configLock.Unlock()
		return ErrShutdown
	}

	// Notify everyone that we are shutting down
	close(agent.closeNotify)

	// Shut down the client multiplexer which will close all its queues
	// effectively causing all the clients to shut down.
	muxCloseErr := routingInfo.clientMux.Close()

	// Drain all the pipelines and error their requests, then
	//  drain the dead queue and error those requests.
	routingInfo.clientMux.Drain(func(req *memdQRequest) {
		req.tryCallback(nil, ErrShutdown)
	})

	agent.configLock.Unlock()

	// Wait for our external looper goroutines to finish, note that if the
	// specific looper wasn't used, it will be a nil value otherwise it
	// will be an open channel till its closed to signal completion.
	if agent.cccpLooperDoneSig != nil {
		<-agent.cccpLooperDoneSig
	}
	if agent.httpLooperDoneSig != nil {
		<-agent.httpLooperDoneSig
	}

	// Close the transports so that they don't hold open goroutines.
	if tsport, ok := agent.httpCli.Transport.(*http.Transport); ok {
		tsport.CloseIdleConnections()
	} else {
		logDebugf("Could not close idle connections for transport")
	}

	return muxCloseErr
}

// IsSecure returns whether this client is connected via SSL.
func (agent *Agent) IsSecure() bool {
	return agent.tlsConfig != nil
}

// BucketUUID returns the UUID of the bucket we are connected to.
func (agent *Agent) BucketUUID() string {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return ""
	}

	return routingInfo.uuid
}

// KeyToVbucket translates a particular key to its assigned vbucket.
func (agent *Agent) KeyToVbucket(key []byte) uint16 {
	// TODO(brett19): The KeyToVbucket Bucket API should return an error

	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return 0
	}

	if routingInfo.vbMap == nil {
		return 0
	}

	return routingInfo.vbMap.VbucketByKey(key)
}

// KeyToServer translates a particular key to its assigned server index.
func (agent *Agent) KeyToServer(key []byte, replicaIdx uint32) int {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return -1
	}

	if routingInfo.vbMap != nil {
		serverIdx, err := routingInfo.vbMap.NodeByKey(key, replicaIdx)
		if err != nil {
			return -1
		}

		return serverIdx
	}

	if routingInfo.ketamaMap != nil {
		serverIdx, err := routingInfo.ketamaMap.NodeByKey(key)
		if err != nil {
			return -1
		}

		return serverIdx
	}

	return -1
}

// VbucketToServer returns the server index for a particular vbucket.
func (agent *Agent) VbucketToServer(vbID uint16, replicaIdx uint32) int {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return -1
	}

	if routingInfo.vbMap == nil {
		return -1
	}

	serverIdx, err := routingInfo.vbMap.NodeByVbucket(vbID, replicaIdx)
	if err != nil {
		return -1
	}

	return serverIdx
}

// NumVbuckets returns the number of VBuckets configured on the
// connected cluster.
func (agent *Agent) NumVbuckets() int {
	return agent.numVbuckets
}

func (agent *Agent) bucketType() bucketType {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return bktTypeInvalid
	}

	return routingInfo.bktType
}

// NumReplicas returns the number of replicas configured on the
// connected cluster.
func (agent *Agent) NumReplicas() int {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return 0
	}

	if routingInfo.vbMap == nil {
		return 0
	}

	return routingInfo.vbMap.NumReplicas()
}

// NumServers returns the number of servers accessible for K/V.
func (agent *Agent) NumServers() int {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return 0
	}
	return routingInfo.clientMux.NumPipelines()
}

// TODO(brett19): Update VbucketsOnServer to return all servers.
// Otherwise, we could race the route map update and get a
// non-continuous list of vbuckets for each server.

// VbucketsOnServer returns the list of VBuckets for a server.
func (agent *Agent) VbucketsOnServer(index int) []uint16 {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil
	}

	if routingInfo.vbMap == nil {
		return nil
	}

	vbList := routingInfo.vbMap.VbucketsByServer(0)

	if len(vbList) <= index {
		// Invalid server index
		return nil
	}

	return vbList[index]
}

// ClientId returns the unique id for this agent
func (agent *Agent) ClientId() string {
	return agent.clientId
}

// CapiEps returns all the available endpoints for performing
// map-reduce queries.
func (agent *Agent) CapiEps() []string {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil
	}
	return routingInfo.capiEpList
}

// MgmtEps returns all the available endpoints for performing
// management queries.
func (agent *Agent) MgmtEps() []string {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil
	}
	return routingInfo.mgmtEpList
}

// N1qlEps returns all the available endpoints for performing
// N1QL queries.
func (agent *Agent) N1qlEps() []string {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil
	}
	return routingInfo.n1qlEpList
}

// FtsEps returns all the available endpoints for performing
// FTS queries.
func (agent *Agent) FtsEps() []string {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil
	}
	return routingInfo.ftsEpList
}

// CbasEps returns all the available endpoints for performing
// CBAS queries.
func (agent *Agent) CbasEps() []string {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil
	}
	return routingInfo.cbasEpList
}
