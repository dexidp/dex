package wait

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/docker/go-connections/nat"
)

// Implement interface
var _ Strategy = (*HostPortStrategy)(nil)

type HostPortStrategy struct {
	Port nat.Port
	// all WaitStrategies should have a startupTimeout to avoid waiting infinitely
	startupTimeout time.Duration
}

// NewHostPortStrategy constructs a default host port strategy
func NewHostPortStrategy(port nat.Port) *HostPortStrategy {
	return &HostPortStrategy{
		Port:           port,
		startupTimeout: defaultStartupTimeout(),
	}
}

// fluent builders for each property
// since go has neither covariance nor generics, the return type must be the type of the concrete implementation
// this is true for all properties, even the "shared" ones like startupTimeout

// ForListeningPort is a helper similar to those in Wait.java
// https://github.com/testcontainers/testcontainers-java/blob/1d85a3834bd937f80aad3a4cec249c027f31aeb4/core/src/main/java/org/testcontainers/containers/wait/strategy/Wait.java
func ForListeningPort(port nat.Port) *HostPortStrategy {
	return NewHostPortStrategy(port)
}

func (hp *HostPortStrategy) WithStartupTimeout(startupTimeout time.Duration) *HostPortStrategy {
	hp.startupTimeout = startupTimeout
	return hp
}

// WaitUntilReady implements Strategy.WaitUntilReady
func (hp *HostPortStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) (err error) {
	// limit context to startupTimeout
	ctx, cancelContext := context.WithTimeout(ctx, hp.startupTimeout)
	defer cancelContext()

	ipAddress, err := target.Host(ctx)
	if err != nil {
		return
	}

	port, err := target.MappedPort(ctx, hp.Port)
	if err != nil {
		return
	}

	proto := port.Proto()
	portNumber := port.Int()
	portString := strconv.Itoa(portNumber)

	//external check
	dialer := net.Dialer{}
	address := net.JoinHostPort(ipAddress, portString)
	for {
		conn, err := dialer.DialContext(ctx, proto, address)
		defer conn.Close()
		if err != nil {
			if v, ok := err.(*net.OpError); ok {
				if v2, ok := (v.Err).(*os.SyscallError); ok {
					if v2.Err == syscall.ECONNREFUSED {
						time.Sleep(100 * time.Millisecond)
						continue
					}
				}
			}
			return err
		}
		break
	}

	//internal check
	command := buildInternalCheckCommand(hp.Port.Int())
	for {
		exitCode, err := target.Exec(ctx, []string{"/bin/bash", "-c", command})
		if err != nil {
			return errors.Wrapf(err, "host port waiting failed")
		}

		if exitCode == 0 {
			break
		}
	}

	return nil
}

func buildInternalCheckCommand(internalPort int) string {
	command := `(
					cat /proc/net/tcp{,6} | awk '{print $2}' | grep -i :%x ||
					nc -vz -w 1 localhost %d ||
					/bin/bash -c '</dev/tcp/localhost/%d'
				)
				`
	return "true && " + fmt.Sprintf(command, internalPort, internalPort, internalPort)
}
