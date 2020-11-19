package gRPCapi

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tokenID = "tokenID"
	userID  = "userID"
)

// TODO: Make file path (certs and database) configurable with variable env AND Dex must be run
// Warning, Dex server need to be in run state

// SetupSuite - Run before all tests and create a new GrpcApiDexStruct to be store in our test structure
func SetupGrpc() (*GrpcApiDex, error) {
	homeDir := os.Getenv("HOME")
	filDir := homeDir + "/workspace/src/dex/"

	ApiDex, err := NewGrpcApiDex("10.153.191.9:5557", filDir+"ca.crt", filDir+"client.crt", filDir+"client.key")
	if err != nil {
		return nil, err
	}
	return ApiDex, nil
}

// Function to test add, update, delete first authenticate token
func TestFirstAuthTokens(t *testing.T) {

	// Need to start Dex server -> No test available

	// attempt to connect to grpc server
	// grpc, err := SetupGrpc()
	// if err != nil {
	// 	t.Fatalf("The setup to connect to grpc must be valid: %v", err)
	// }
	// if err := grpc.AddUser(User{InternId: "testInternID", Pseudo: "myPseudo", Email: "email", AclTokens: []string{"1", "2"}}); err != nil {
	// 	t.Fatalf("Adding user should be valid: %v", err)
	// }
}

// Function to test add, update delete authenticate user
func TestFirstAuthUsers(t *testing.T) {
	// Not Implemented
}
