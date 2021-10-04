package storage

import (
	"context"
	"fmt"
	"time"
)

// NewCustomHealthCheckFunc returns a new health check function.
func NewCustomHealthCheckFunc(s Storage, now func() time.Time) func(context.Context) (details interface{}, err error) {
	return func(_ context.Context) (details interface{}, err error) {
		a := AuthRequest{
			ID:       NewID(),
			ClientID: NewID(),

			// Set a short expiry so if the delete fails this will be cleaned up quickly by garbage collection.
			Expiry: now().Add(time.Minute),
		}

		if err := s.CreateAuthRequest(a); err != nil {
			return nil, fmt.Errorf("create auth request: %v", err)
		}

		if err := s.DeleteAuthRequest(a.ID); err != nil {
			return nil, fmt.Errorf("delete auth request: %v", err)
		}

		return nil, nil
	}
}
