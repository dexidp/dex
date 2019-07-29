package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dexidp/dex/api"
)

func newDexClient(hostAndPort, caPath, clientCrt, clientKey string) (api.DexClient, error) {
	cPool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("invalid CA crt file: %s", caPath)
	}
	if cPool.AppendCertsFromPEM(caCert) != true {
		return nil, fmt.Errorf("failed to parse CA crt")
	}

	clientCert, err := tls.LoadX509KeyPair(clientCrt, clientKey)
	if err != nil {
		return nil, fmt.Errorf("invalid client crt file: %s", caPath)
	}

	clientTLSConfig := &tls.Config{
		RootCAs:      cPool,
		Certificates: []tls.Certificate{clientCert},
	}
	creds := credentials.NewTLS(clientTLSConfig)

	conn, err := grpc.Dial(hostAndPort, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return api.NewDexClient(conn), nil
}

func createPassword(cli api.DexClient) error {
	p := api.Password{
		Email: "test@example.com",
		// bcrypt hash of the value "test1" with cost 10
		Hash:     []byte("$2a$10$XVMN/Fid.Ks4CXgzo8fpR.iU1khOMsP5g9xQeXuBm1wXjRX8pjUtO"),
		Username: "test",
		UserId:   "test",
	}

	createReq := &api.CreatePasswordReq{
		Password: &p,
	}

	// Create password.
	if resp, err := cli.CreatePassword(context.TODO(), createReq); err != nil || resp.AlreadyExists {
		if resp.AlreadyExists {
			return fmt.Errorf("Password %s already exists", createReq.Password.Email)
		}
		return fmt.Errorf("failed to create password: %v", err)
	}
	log.Printf("Created password with email %s", createReq.Password.Email)

	// List all passwords.
	resp, err := cli.ListPasswords(context.TODO(), &api.ListPasswordReq{})
	if err != nil {
		return fmt.Errorf("failed to list password: %v", err)
	}

	log.Print("Listing Passwords:\n")
	for _, pass := range resp.Passwords {
		log.Printf("%+v", pass)
	}

	// Verifying correct and incorrect passwords
	log.Print("Verifying Password:\n")
	verifyReq := &api.VerifyPasswordReq{
		Email:    "test@example.com",
		Password: "test1",
	}
	verifyResp, err := cli.VerifyPassword(context.TODO(), verifyReq)
	if err != nil {
		return fmt.Errorf("failed to run VerifyPassword for correct password: %v", err)
	}
	if !verifyResp.Verified {
		return fmt.Errorf("failed to verify correct password: %v", verifyResp)
	}
	log.Printf("properly verified correct password: %t\n", verifyResp.Verified)

	badVerifyReq := &api.VerifyPasswordReq{
		Email:    "test@example.com",
		Password: "wrong_password",
	}
	badVerifyResp, err := cli.VerifyPassword(context.TODO(), badVerifyReq)
	if err != nil {
		return fmt.Errorf("failed to run VerifyPassword for incorrect password: %v", err)
	}
	if badVerifyResp.Verified {
		return fmt.Errorf("verify returned true for incorrect password: %v", badVerifyResp)
	}
	log.Printf("properly failed to verify incorrect password: %t\n", badVerifyResp.Verified)

	log.Print("Listing Passwords:\n")
	for _, pass := range resp.Passwords {
		log.Printf("%+v", pass)
	}

	deleteReq := &api.DeletePasswordReq{
		Email: p.Email,
	}

	// Delete password with email = test@example.com.
	if resp, err := cli.DeletePassword(context.TODO(), deleteReq); err != nil || resp.NotFound {
		if resp.NotFound {
			return fmt.Errorf("Password %s not found", deleteReq.Email)
		}
		return fmt.Errorf("failed to delete password: %v", err)
	}
	log.Printf("Deleted password with email %s", deleteReq.Email)

	return nil
}

func main() {
	caCrt := flag.String("ca-crt", "", "CA certificate")
	clientCrt := flag.String("client-crt", "", "Client certificate")
	clientKey := flag.String("client-key", "", "Client key")
	flag.Parse()

	if *clientCrt == "" || *caCrt == "" || *clientKey == "" {
		log.Fatal("Please provide CA & client certificates and client key. Usage: ./client --ca-crt=<path ca.crt> --client-crt=<path client.crt> --client-key=<path client key>")
	}

	client, err := newDexClient("127.0.0.1:5557", *caCrt, *clientCrt, *clientKey)
	if err != nil {
		log.Fatalf("failed creating dex client: %v ", err)
	}

	if err := createPassword(client); err != nil {
		log.Fatalf("testPassword failed: %v", err)
	}
}
