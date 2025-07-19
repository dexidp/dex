package storage

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func (m *mockStorage) GetAuthRequest(ctx context.Context, id string) (AuthRequest, error) {
	if m.getErr != nil {
		return AuthRequest{}, m.getErr
	}
	return AuthRequest{}, ErrNotFound
}

func (m *mockStorage) CreateAuthRequest(ctx context.Context, a AuthRequest) error {
	if m.createErr != nil {
		return m.createErr
	}
	return nil
}

func (m *mockStorage) DeleteAuthRequest(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func TestNewCustomHealthCheckFunc(t *testing.T) {
	ctx := context.Background()

	fixedTime := time.Now()
	now := func() time.Time { return fixedTime }

	tests := []struct {
		name            string
		createErr       error
		deleteErr       error
		expectedErr     error
		expectedDetails interface{}
	}{
		{
			name:            "Success",
			createErr:       nil,
			deleteErr:       nil,
			expectedErr:     nil,
			expectedDetails: nil,
		},
		{
			name:            "Create auth request fails",
			createErr:       errors.New("create failed"),
			deleteErr:       nil,
			expectedErr:     fmt.Errorf("create auth request: %w", errors.New("create failed")),
			expectedDetails: nil,
		},
		{
			name:            "Delete auth request fails",
			createErr:       nil,
			deleteErr:       errors.New("delete failed"),
			expectedErr:     fmt.Errorf("delete auth request: %w", errors.New("delete failed")),
			expectedDetails: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mock storage
			mock := newMockStorage()
			mock.createErr = tc.createErr
			mock.deleteErr = tc.deleteErr

			// Create health check function
			healthCheck := NewCustomHealthCheckFunc(mock, now)

			// Run health check
			details, err := healthCheck(ctx)

			// Verify results
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expectedDetails, details)
		})
	}
}
