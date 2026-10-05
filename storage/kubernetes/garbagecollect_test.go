package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage"
)

func TestGarbageCollectConcurrentDelete(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	resources := []string{
		resourceAuthRequest, resourceAuthCode, resourceDeviceRequest,
		resourceDeviceToken, resourceAuthSession,
	}

	for _, resource := range resources {
		for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
			t.Run(fmt.Sprintf("%s/%d", resource, status), func(t *testing.T) {
				var mu sync.Mutex
				var listed, deleted []string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					switch r.Method {
					case http.MethodGet:
						listed = append(listed, path.Base(r.URL.Path))
						items := []map[string]interface{}{}
						if path.Base(r.URL.Path) == resource {
							items = []map[string]interface{}{
								gcTestObject("expired", now.Add(-time.Second), now.Add(time.Hour)),
								gcTestObject("live", now.Add(time.Hour), now.Add(time.Hour)),
								gcTestObject("boundary", now, now),
							}
						}
						json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
					case http.MethodDelete:
						deleted = append(deleted, path.Base(r.URL.Path))
						w.WriteHeader(status)
					default:
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				}))
				t.Cleanup(server.Close)
				var logs bytes.Buffer
				cli := &client{
					client: server.Client(), baseURL: server.URL,
					logger: slog.New(slog.NewTextHandler(&logs, nil)),
				}

				result, err := cli.GarbageCollect(context.Background(), now)
				mu.Lock()
				defer mu.Unlock()
				require.Equal(t, []string{"expired"}, deleted)
				want := gcTestResult(resource)
				if status == http.StatusOK || status == http.StatusNotFound {
					require.NoError(t, err)
					require.Empty(t, logs.String(), "an already-deleted expired object is not an error")
					require.Equal(t, resources, listed, "cleanup must continue to later resource types")
				} else {
					require.Error(t, err)
					require.Contains(t, logs.String(), "failed to delete")
					if resource == resourceAuthSession {
						want.AuthSessions = 0
					}
				}
				require.Equal(t, want, result)
			})
		}
	}
}

func TestGarbageCollectAuthSessionIdleExpiry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		items := []map[string]interface{}{}
		if path.Base(r.URL.Path) == resourceAuthSession {
			items = append(items, gcTestObject("idle-expired", now.Add(time.Hour), now.Add(-time.Second)))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
	}))
	t.Cleanup(server.Close)
	cli := &client{client: server.Client(), baseURL: server.URL, logger: slog.New(slog.DiscardHandler)}
	result, err := cli.GarbageCollect(context.Background(), now)
	require.NoError(t, err)
	require.Equal(t, storage.GCResult{AuthSessions: 1}, result)
}

func TestGarbageCollectListErrors(t *testing.T) {
	for _, resource := range []string{
		resourceAuthRequest, resourceAuthCode, resourceDeviceRequest,
		resourceDeviceToken, resourceAuthSession,
	} {
		t.Run(resource, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if path.Base(r.URL.Path) == resource {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Write([]byte(`{"items":[]}`))
			}))
			t.Cleanup(server.Close)
			cli := &client{client: server.Client(), baseURL: server.URL, logger: slog.New(slog.DiscardHandler)}
			_, err := cli.GarbageCollect(context.Background(), time.Now())
			require.ErrorContains(t, err, "failed to list")
		})
	}
}

func TestDeleteStillReturnsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	cli := &client{client: server.Client(), baseURL: server.URL}
	require.ErrorIs(t, cli.DeleteAuthRequest(context.Background(), "missing"), storage.ErrNotFound)
}

func gcTestObject(name string, expiry, idleExpiry time.Time) map[string]interface{} {
	return map[string]interface{}{
		"metadata": map[string]string{"name": name},
		"expiry":   expiry, "absoluteExpiry": expiry, "idleExpiry": idleExpiry,
	}
}

func gcTestResult(resource string) storage.GCResult {
	switch resource {
	case resourceAuthRequest:
		return storage.GCResult{AuthRequests: 1}
	case resourceAuthCode:
		return storage.GCResult{AuthCodes: 1}
	case resourceDeviceRequest:
		return storage.GCResult{DeviceRequests: 1}
	case resourceDeviceToken:
		return storage.GCResult{DeviceTokens: 1}
	case resourceAuthSession:
		return storage.GCResult{AuthSessions: 1}
	default:
		panic("unexpected garbage collection resource")
	}
}
