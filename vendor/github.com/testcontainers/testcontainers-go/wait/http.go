package wait

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/go-connections/nat"
)

// Implement interface
var _ Strategy = (*HTTPStrategy)(nil)

type HTTPStrategy struct {
	// all Strategies should have a startupTimeout to avoid waiting infinitely
	startupTimeout time.Duration

	// additional properties
	Port              nat.Port
	Path              string
	StatusCodeMatcher func(status int) bool
	UseTLS            bool
	AllowInsecure     bool
}

// NewHTTPStrategy constructs a HTTP strategy waiting on port 80 and status code 200
func NewHTTPStrategy(path string) *HTTPStrategy {
	return &HTTPStrategy{
		startupTimeout:    defaultStartupTimeout(),
		Port:              "80/tcp",
		Path:              path,
		StatusCodeMatcher: defaultStatusCodeMatcher,
		UseTLS:            false,
	}

}

func defaultStatusCodeMatcher(status int) bool {
	return status == http.StatusOK
}

// fluent builders for each property
// since go has neither covariance nor generics, the return type must be the type of the concrete implementation
// this is true for all properties, even the "shared" ones like startupTimeout

func (ws *HTTPStrategy) WithStartupTimeout(startupTimeout time.Duration) *HTTPStrategy {
	ws.startupTimeout = startupTimeout
	return ws
}

func (ws *HTTPStrategy) WithPort(port nat.Port) *HTTPStrategy {
	ws.Port = port
	return ws
}

func (ws *HTTPStrategy) WithStatusCodeMatcher(statusCodeMatcher func(status int) bool) *HTTPStrategy {
	ws.StatusCodeMatcher = statusCodeMatcher
	return ws
}

func (ws *HTTPStrategy) WithTLS(useTLS bool) *HTTPStrategy {
	ws.UseTLS = useTLS
	return ws
}

func (ws *HTTPStrategy) WithAllowInsecure(allowInsecure bool) *HTTPStrategy {
	ws.AllowInsecure = allowInsecure
	return ws
}

// ForHTTP is a convenience method similar to Wait.java
// https://github.com/testcontainers/testcontainers-java/blob/1d85a3834bd937f80aad3a4cec249c027f31aeb4/core/src/main/java/org/testcontainers/containers/wait/strategy/Wait.java
func ForHTTP(path string) *HTTPStrategy {
	return NewHTTPStrategy(path)
}

// WaitUntilReady implements Strategy.WaitUntilReady
func (ws *HTTPStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) (err error) {
	// limit context to startupTimeout
	ctx, cancelContext := context.WithTimeout(ctx, ws.startupTimeout)
	defer cancelContext()

	ipAddress, err := target.Host(ctx)
	if err != nil {
		return
	}

	port, err := target.MappedPort(ctx, ws.Port)
	if err != nil {
		return
	}

	if port.Proto() != "tcp" {
		return errors.New("Cannot use HTTP client on non-TCP ports")
	}

	portNumber := port.Int()
	portString := strconv.Itoa(portNumber)

	address := net.JoinHostPort(ipAddress, portString)

	var proto string
	if ws.UseTLS {
		proto = "https"
	} else {
		proto = "http"
	}

	url := fmt.Sprintf("%s://%s%s", proto, address, ws.Path)

	tripper := http.DefaultTransport

	if ws.AllowInsecure {
		tripper.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := http.Client{Timeout: ws.startupTimeout, Transport: tripper}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	Retry:
	for {
		select {
		case <-ctx.Done():
			break Retry
		default:
			resp, err := client.Do(req)
			if err != nil || !ws.StatusCodeMatcher(resp.StatusCode) {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			break Retry
		}
	}

	return nil
}
