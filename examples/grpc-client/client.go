package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.33.0"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dexidp/dex/api/v2"
)

func InitTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (func(context.Context) error, error) {
	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func newDexClient(hostAndPort, caPath, clientCrt, clientKey string) (api.DexClient, error) {
	cPool := x509.NewCertPool()
	caCert, err := os.ReadFile(caPath)
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
	conn, err := grpc.NewClient(hostAndPort,
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)

	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}
	return api.NewDexClient(conn), nil
}

func createPassword(cli api.DexClient, ctx context.Context) error {
	ctx, span := otel.Tracer("grpc-client").Start(ctx, "createPassword")
	defer span.End()
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
	if resp, err := cli.CreatePassword(ctx, createReq); err != nil || resp.AlreadyExists {
		if resp != nil && resp.AlreadyExists {
			return fmt.Errorf("Password %s already exists", createReq.Password.Email)
		}
		return fmt.Errorf("failed to create password: %v", err)
	}
	log.Printf("Created password with email %s", createReq.Password.Email)

	// List all passwords.
	resp, err := cli.ListPasswords(ctx, &api.ListPasswordReq{})
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
	verifyResp, err := cli.VerifyPassword(ctx, verifyReq)
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
	badVerifyResp, err := cli.VerifyPassword(ctx, badVerifyReq)
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
	if resp, err := cli.DeletePassword(ctx, deleteReq); err != nil || resp.NotFound {
		if resp != nil && resp.NotFound {
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
	ctx := context.Background()
	serviceNameVal := semconv.ServiceNameKey.String("grpc-client")
	res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, serviceNameVal))

	conn, err := grpc.NewClient("localhost:4317",
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("failed to create gRPC connection to collector: %w", err)
	}
	provider, err := InitTracerProvider(ctx, res, conn)
	if err != nil {
		log.Printf("failed to init tracer: %w", err)
	}
	defer func() {
		if err := provider(ctx); err != nil {
			log.Printf("failed to shutdown tracer provider: %v", err)
		}
	}()
	if *clientCrt == "" || *caCrt == "" || *clientKey == "" {
		log.Fatal("Please provide CA & client certificates and client key. Usage: ./client --ca-crt=<path ca.crt> --client-crt=<path client.crt> --client-key=<path client key>")
	}

	client, err := newDexClient("127.0.0.1:5557", *caCrt, *clientCrt, *clientKey)
	if err != nil {
		log.Fatalf("failed creating dex client: %v ", err)
	}

	if err := createPassword(client, ctx); err != nil {
		log.Fatalf("testPassword failed: %v", err)
	}
}
