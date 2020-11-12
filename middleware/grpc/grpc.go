// Package grpc implements support for external middleware.
package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/middleware"
	"github.com/dexidp/dex/middleware/grpc/api"

	"github.com/dexidp/dex/pkg/log"

	"github.com/dexidp/dex/version"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// The current API version; if you update this, you will need to update the
// tests accordingly.
const apiVersion = 1

// The error we return if the API version found is incompatible.
type incompatibleAPIVersion struct {
	endpoint   string
	gotVersion int32
}

func (i incompatibleAPIVersion) Error() string {
	return fmt.Sprintf("grpc: API version %d from endpoint %q is too old; minimum supported version is %d",
		i.gotVersion, i.endpoint, apiVersion)
}

// Returns true if the error is a grpc incompatible API version error.
func IsIncompatibleAPIVersion(err error) bool {
	_, ok := err.(*incompatibleAPIVersion)
	return ok
}

// Config holds the configuration parameters for the grpc middleware.
// This middleware provides the ability to call out to an external
// implementation using gRPC.
//
// An example config:
//
//     type: grpc
//     config:
//       endpoint: localhost:1234
//		 timeout: 1s
//       tls:
//         caCert: /path/to/ca-cert.pem
//         client:
//           cert: /path/to/client-cert.pem
//           key: /path/to/client-key.pem
//
type Config struct {
	// The endpoint to contact.
	Endpoint string `json:"endpoint"`

	// Says how long we're prepared to wait for a reply (default 30s).
	Timeout string `json:"timeout,omitempty"`

	// If true, run in insecure mode (default false).
	Insecure bool `json:"insecure,omitempty"`

	// TLS configuration.
	TLS *TLSConfig `json:"tls,omitempty"`
}

// TLSConfig holds paths to certificates and/or private keys.
type TLSConfig struct {
	// The path to the root certificate for the server CA.  If not
	// specified, we will use the system roots instead.
	CACert string `json:"caCert,omitempty"`

	// The client certificate data, if any.
	Client *ClientCert `json:"client,omitempty"`
}

// ClientCert holds client certificate information.
type ClientCert struct {
	// The path to the client certificate itself.
	Cert string `json:"cert"`

	// The path to the client's private key.
	Key string `json:"key"`
}

// Open returns a gRPC Middleware
func (c *Config) Open(logger log.Logger) (middleware.Middleware, error) {
	ua := fmt.Sprintf("Dex/%s (%s %s; Go %s)",
		version.Version,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version())

	options := []grpc.DialOption{
		grpc.WithUserAgent(ua),
	}

	if c.Insecure {
		options = append(options, grpc.WithInsecure())
		logger.Warnf("grpc: INSECURE connection to endpoint %q", c.Endpoint)
	} else if c.TLS != nil {
		config := &tls.Config{}
		useConfig := false

		if c.TLS.CACert != "" {
			caCertPem, err := ioutil.ReadFile(c.TLS.CACert)
			if err != nil {
				return nil, fmt.Errorf("grpc: unable to read CA certificate %q: %v", c.TLS.CACert, err)
			}
			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(caCertPem) {
				return nil, fmt.Errorf("grpc: failed to add CA certificate for endpoint %q", c.Endpoint)
			}

			config.RootCAs = certPool
			useConfig = true
		}

		if c.TLS.Client != nil {
			clientCert, err := tls.LoadX509KeyPair(c.TLS.Client.Cert,
				c.TLS.Client.Key)
			if err != nil {
				return nil, fmt.Errorf("grpc: failed to load client key pair for endpoint %q", c.Endpoint)
			}

			config.Certificates = []tls.Certificate{clientCert}
			useConfig = true
		}

		if useConfig {
			creds := credentials.NewTLS(config)
			options = append(options, grpc.WithTransportCredentials(creds))
		}
	}

	conn, err := grpc.Dial(c.Endpoint, options...)
	if err != nil {
		return nil, fmt.Errorf("grpc: unable to dial endpoint %q: %v",
			c.Endpoint, err)
	}

	timeout := 30 * time.Second
	if c.Timeout != "" {
		timeout, err = time.ParseDuration(c.Timeout)
		if err != nil {
			return nil, fmt.Errorf("grpc: unable to parse timeout: %q",
				c.Timeout)
		}
	}

	return &grpcMiddleware{
		Config:          *c,
		Conn:            conn,
		TimeoutDuration: timeout,
		Logger:          logger,
	}, nil
}

type grpcMiddleware struct {
	Config

	Client          api.MiddlewareClient
	Conn            *grpc.ClientConn
	TimeoutDuration time.Duration

	Logger log.Logger
}

// When we first connect, get the server info and check the API version
func (g *grpcMiddleware) doConnect(ctx context.Context) (api.MiddlewareClient, error) {
	tctx, cancel := context.WithTimeout(ctx, g.TimeoutDuration)
	defer cancel()

	client := api.NewMiddlewareClient(g.Conn)

	infoReq := api.ServerInfoReq{}
	info, err := client.GetInfo(tctx, &infoReq)
	if err != nil {
		return nil, err
	}

	if info.Api < apiVersion {
		return nil, &incompatibleAPIVersion{endpoint: g.Endpoint, gotVersion: info.Api}
	}

	g.Logger.Infof("grpc: connected to endpoint %q, %s (%s), API version %d",
		g.Endpoint,
		info.Name,
		info.Version,
		info.Api)

	return client, nil
}

// Send the identity to the remote middleware to process it
func (g *grpcMiddleware) Process(ctx context.Context, identity connector.Identity) (connector.Identity, error) {
	customClaims := identity.CustomClaims
	custom, err := json.Marshal(customClaims)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("grpc: unable to marshal custom claims: %v", err)
	}

	ident := &api.Identity{
		UserId:            identity.UserID,
		Username:          identity.Username,
		PreferredUsername: identity.PreferredUsername,
		Email:             identity.Email,
		EmailVerified:     identity.EmailVerified,
		Groups:            identity.Groups,
		CustomClaims:      custom,
	}

	if g.Client == nil {
		g.Client, err = g.doConnect(ctx)
		if err != nil {
			if IsIncompatibleAPIVersion(err) {
				return connector.Identity{}, err
			}
			return connector.Identity{}, fmt.Errorf("grpc: middleware RPC error on first connect: %v", err)
		}
	}

	tctx, cancel := context.WithTimeout(ctx, g.TimeoutDuration)
	defer cancel()

	request := api.ProcessReq{Identity: ident}
	ident, err = g.Client.Process(tctx, &request)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("grpc: middleware RPC error: %v", err)
	}

	var claims map[string]interface{}
	err = json.Unmarshal(ident.CustomClaims, &claims)
	if err != nil {
		return connector.Identity{}, fmt.Errorf("grpc: unable to unmarshal custom claims: %v", err)
	}

	return connector.Identity{
		UserID:            ident.UserId,
		Username:          ident.Username,
		PreferredUsername: ident.PreferredUsername,
		Email:             ident.Email,
		EmailVerified:     ident.EmailVerified,
		Groups:            ident.Groups,
		CustomClaims:      claims,
		ConnectorData:     identity.ConnectorData,
	}, nil
}
