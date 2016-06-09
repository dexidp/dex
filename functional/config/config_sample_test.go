package config

import (
	"os"
	"testing"

	"github.com/coreos/dex/client"
	"github.com/coreos/dex/client/manager"
	"github.com/coreos/dex/db"
)

const (
	clientsFile = "../../static/fixtures/clients.json.sample"
)

// TestClientSample makes sure that the clients.json.sample file is valid and can be loaded properly.
func TestClientSample(t *testing.T) {
	f, err := os.Open(clientsFile)
	if err != nil {
		t.Fatalf("could not open file %q: %v", clientsFile, err)
	}
	defer f.Close()

	clients, err := client.ClientsFromReader(f)
	if err != nil {
		t.Fatalf("Error loading Clients: %v", err)
	}

	memDB := db.NewMemDB()
	repo, err := db.NewClientRepoFromClients(memDB, clients)
	if err != nil {
		t.Fatalf("Error creating Clients: %v", err)
	}

	mgr := manager.NewClientManager(repo, db.TransactionFactory(memDB), manager.ManagerOptions{})

	for i, c := range clients {
		ok, err := mgr.Authenticate(c.Client.Credentials)
		if !ok {
			t.Errorf("case %d: couldn't authenticate", i)
		}
		if err != nil {
			t.Errorf("case %d: error authenticating: %v", i, err)
		}
	}

}
