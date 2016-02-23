package db

import (
	"sync"
	"testing"

	"github.com/coreos/dex/connector"
)

// TestConcurrentSqliteConns tests concurrent writes to a single in memory database.
func TestConcurrentSqliteConns(t *testing.T) {
	dbMap := NewMemDB()
	repo := NewConnectorConfigRepo(dbMap)

	var (
		once sync.Once
		wg   sync.WaitGroup
	)

	n := 1000
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			configs := []connector.ConnectorConfig{
				&connector.LocalConnectorConfig{ID: "local"},
			}
			// Setting connector configs both deletes and writes to a single table
			// within a transaction.
			if err := repo.Set(configs); err != nil {
				// Don't print 1000 errors, only the first.
				once.Do(func() {
					t.Errorf("concurrent connections to sqlite3: %v", err)
				})
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
