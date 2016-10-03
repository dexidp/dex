package sql

import (
	"fmt"
	"time"
)

type gc struct {
	now  func() time.Time
	conn *conn
}

var tablesWithGC = []string{"auth_request", "auth_code"}

func (gc gc) run() error {
	for _, table := range tablesWithGC {
		_, err := gc.conn.Exec(`delete from `+table+` where expiry < $1`, gc.now())
		if err != nil {
			return fmt.Errorf("gc %s: %v", table, err)
		}
		// TODO(ericchiang): when we have levelled logging print how many rows were gc'd
	}
	return nil
}
