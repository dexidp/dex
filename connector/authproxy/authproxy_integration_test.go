package authproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dexidp/dex/connector"
)

// The fixed port on which the Go test backend listens.
// nginx is configured to proxy_pass to host.docker.internal:18562.
const testBackendPort = "18562"

const testAuthProxyURLEnv = "DEX_AUTHPROXY_URL"

// identityResult holds the result of HandleCallback invoked on the proxied request.
type identityResult struct {
	Identity connector.Identity `json:"identity"`
	Error    string             `json:"error,omitempty"`
}

// startTestBackend starts an HTTP server that receives proxied requests from nginx,
// invokes HandleCallback on each request, and returns the identity as JSON.
// The caller must call the returned cleanup function to shut down the server.
func startTestBackend(t *testing.T, conn *callback) (cleanup func()) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ident, err := conn.HandleCallback(connector.Scopes{Groups: true}, r)
		result := identityResult{Identity: ident}
		if err != nil {
			result.Error = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	listener, err := net.Listen("tcp", ":"+testBackendPort)
	if err != nil {
		t.Fatalf("failed to listen on port %s: %v", testBackendPort, err)
	}

	server := &http.Server{Handler: mux}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("backend server error: %v", err)
		}
	}()

	return func() {
		server.Close()
		wg.Wait()
	}
}

// subtest describes a single integration test case for the authproxy connector.
type integrationSubtest struct {
	name   string
	config Config
	// path is the nginx location path to hit (e.g., "/default", "/minimal").
	path string

	wantErr bool
	want    connector.Identity
}

func TestIntegrationAuthProxy(t *testing.T) {
	proxyURL := os.Getenv(testAuthProxyURLEnv)
	if proxyURL == "" {
		t.Skipf("test environment variable %q not set, skipping authproxy integration tests", testAuthProxyURLEnv)
	}

	tests := []integrationSubtest{
		{
			name:   "all headers set",
			config: Config{},
			path:   "/default",
			want: connector.Identity{
				UserID:            "uid-12345",
				Username:          "testuser",
				PreferredUsername: "Test User",
				Email:             "testuser@example.com",
				EmailVerified:     true,
				Groups:            []string{"group1", "group2", "group3"},
			},
		},
		{
			name:   "minimal headers - fallback to X-Remote-User",
			config: Config{},
			path:   "/minimal",
			want: connector.Identity{
				UserID:            "janedoe",
				Username:          "janedoe",
				PreferredUsername: "janedoe",
				Email:             "janedoe",
				EmailVerified:     true,
			},
		},
		{
			name:    "no auth headers - expect error",
			config:  Config{},
			path:    "/no-headers",
			wantErr: true,
		},
		{
			name: "custom group separator",
			config: Config{
				GroupHeaderSeparator: ";",
			},
			path: "/custom-separator",
			want: connector.Identity{
				UserID:            "uid-99999",
				Username:          "johndoe",
				PreferredUsername: "John Doe",
				Email:             "johndoe@example.com",
				EmailVerified:     true,
				Groups:            []string{"admins", "developers", "ops"},
			},
		},
		{
			name: "all headers set with static groups",
			config: Config{
				Groups: []string{"static-group1", "static-group2"},
			},
			path: "/default",
			want: connector.Identity{
				UserID:            "uid-12345",
				Username:          "testuser",
				PreferredUsername: "Test User",
				Email:             "testuser@example.com",
				EmailVerified:     true,
				Groups:            []string{"group1", "group2", "group3", "static-group1", "static-group2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := slog.New(slog.DiscardHandler)
			c, err := tt.config.Open("test-authproxy", l)
			if err != nil {
				t.Fatalf("failed to open connector: %v", err)
			}

			cb := c.(*callback)
			cleanup := startTestBackend(t, cb)
			defer cleanup()

			// Give the backend a moment to start.
			time.Sleep(50 * time.Millisecond)

			// Make a request through the nginx proxy.
			reqURL := fmt.Sprintf("%s%s", proxyURL, tt.path)

			resp, err := http.Get(reqURL)
			if err != nil {
				t.Fatalf("failed to make request through proxy: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			var result identityResult
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("failed to decode response: %v\nbody: %s", err, string(body))
			}

			if tt.wantErr {
				if result.Error == "" {
					t.Fatal("expected error from HandleCallback, got none")
				}
				return
			}

			if result.Error != "" {
				t.Fatalf("unexpected error from HandleCallback: %s", result.Error)
			}

			got := result.Identity
			if got.UserID != tt.want.UserID {
				t.Errorf("UserID: got %q, want %q", got.UserID, tt.want.UserID)
			}
			if got.Username != tt.want.Username {
				t.Errorf("Username: got %q, want %q", got.Username, tt.want.Username)
			}
			if got.PreferredUsername != tt.want.PreferredUsername {
				t.Errorf("PreferredUsername: got %q, want %q", got.PreferredUsername, tt.want.PreferredUsername)
			}
			if got.Email != tt.want.Email {
				t.Errorf("Email: got %q, want %q", got.Email, tt.want.Email)
			}
			if got.EmailVerified != tt.want.EmailVerified {
				t.Errorf("EmailVerified: got %v, want %v", got.EmailVerified, tt.want.EmailVerified)
			}
			if len(got.Groups) != len(tt.want.Groups) {
				t.Errorf("Groups length: got %d, want %d (got: %v, want: %v)", len(got.Groups), len(tt.want.Groups), got.Groups, tt.want.Groups)
			} else {
				for i := range tt.want.Groups {
					if got.Groups[i] != tt.want.Groups[i] {
						t.Errorf("Groups[%d]: got %q, want %q", i, got.Groups[i], tt.want.Groups[i])
					}
				}
			}
		})
	}
}
