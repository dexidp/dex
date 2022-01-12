package external

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type gRPCConnectorConfig struct {
	Port      int    `json:"port"`
	CAPath    string `json:"tlsClientCA"`
	ClientCrt string `json:"tlsCert"`
	ClientKey string `json:"tlsKey"`
}

func grpcConn(port int, ca, crt, key string) (*grpc.ClientConn, error) {
	cPool := x509.NewCertPool()
	caCert, err := os.ReadFile(ca)
	if err != nil {
		return nil, fmt.Errorf("invalid CA crt file: %s", ca)
	}

	if !cPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA crt")
	}

	clientCert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return nil, fmt.Errorf("invalid client crt file: %s", crt)
	}

	clientTLSConfig := &tls.Config{
		RootCAs:      cPool,
		Certificates: []tls.Certificate{clientCert},
	}
	creds := credentials.NewTLS(clientTLSConfig)

	hostAndPort := fmt.Sprintf("127.0.0.1:%d", port)

	conn, err := grpc.Dial(hostAndPort, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}

	return conn, nil
}
