package sql

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coreos/dex/storage"
)

type gc struct {
	now  func() time.Time
	conn *conn
}

func (gc gc) run() error {
	for _, table := range []string{"auth_request", "auth_code"} {
		_, err := gc.conn.Exec(`delete from `+table+` where expiry < $1`, gc.now())
		if err != nil {
			return fmt.Errorf("gc %s: %v", table, err)
		}
		// TODO(ericchiang): when we have levelled logging print how many rows were gc'd
	}
	return nil
}

type withCancel struct {
	storage.Storage
	cancel context.CancelFunc
}

func (w withCancel) Close() error {
	w.cancel()
	return w.Storage.Close()
}

func withGC(conn *conn, now func() time.Time) storage.Storage {
	ctx, cancel := context.WithCancel(context.Background())
	run := (gc{now, conn}).run
	go func() {
		for {
			select {
			case <-time.After(time.Second * 30):
				if err := run(); err != nil {
					log.Printf("gc failed: %v", err)
				}
			case <-ctx.Done():
			}
		}
	}()
	return withCancel{conn, cancel}
}
