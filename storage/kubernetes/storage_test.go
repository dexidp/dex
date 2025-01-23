package kubernetes

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/dexidp/dex/storage"
	"github.com/dexidp/dex/storage/conformance"
)

const kubeconfigPathVariableName = "DEX_KUBERNETES_CONFIG_PATH"

func TestStorage(t *testing.T) {
	if os.Getenv(kubeconfigPathVariableName) == "" {
		t.Skipf("variable %q not set, skipping kubernetes storage tests\n", kubeconfigPathVariableName)
	}

	suite.Run(t, new(StorageTestSuite))
}

type StorageTestSuite struct {
	suite.Suite
	client *client
}

func expandDir(dir string) (string, error) {
	dir = strings.Trim(dir, `"`)
	if strings.HasPrefix(dir, "~/") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir = filepath.Join(homedir, strings.TrimPrefix(dir, "~/"))
	}
	return dir, nil
}

func (s *StorageTestSuite) SetupTest() {
	kubeconfigPath, err := expandDir(os.Getenv(kubeconfigPathVariableName))
	s.Require().NoError(err)

	config := Config{
		KubeConfigFile: kubeconfigPath,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	kubeClient, err := config.open(logger, true)
	s.Require().NoError(err)

	s.client = kubeClient
}

func (s *StorageTestSuite) TestStorage() {
	newStorage := func() storage.Storage {
		for _, resource := range []string{
			resourceAuthCode,
			resourceAuthRequest,
			resourceDeviceRequest,
			resourceDeviceToken,
			resourceClient,
			resourceRefreshToken,
			resourceKeys,
			resourcePassword,
		} {
			if err := s.client.deleteAll(resource); err != nil {
				s.T().Fatalf("delete all %q failed: %v", resource, err)
			}
		}
		return s.client
	}

	conformance.RunTests(s.T(), newStorage)
	conformance.RunTransactionTests(s.T(), newStorage)
}

func TestURLFor(t *testing.T) {
	tests := []struct {
		apiVersion, namespace, resource, name string

		baseURL string
		want    string
	}{
		{
			"v1", "default", "pods", "a",
			"https://k8s.example.com",
			"https://k8s.example.com/api/v1/namespaces/default/pods/a",
		},
		{
			"foo/v1", "default", "bar", "a",
			"https://k8s.example.com",
			"https://k8s.example.com/apis/foo/v1/namespaces/default/bar/a",
		},
		{
			"foo/v1", "default", "bar", "a",
			"https://k8s.example.com/",
			"https://k8s.example.com/apis/foo/v1/namespaces/default/bar/a",
		},
		{
			"foo/v1", "default", "bar", "a",
			"https://k8s.example.com/",
			"https://k8s.example.com/apis/foo/v1/namespaces/default/bar/a",
		},
		{
			// no namespace
			"foo/v1", "", "bar", "a",
			"https://k8s.example.com",
			"https://k8s.example.com/apis/foo/v1/bar/a",
		},
	}

	for _, test := range tests {
		c := &client{baseURL: test.baseURL}
		got, err := c.urlFor(test.apiVersion, test.namespace, test.resource, test.name)
		if err != nil {
			t.Errorf("got error: %v", err)
		}

		if got != test.want {
			t.Errorf("(&client{baseURL:%q}).urlFor(%q, %q, %q, %q): expected %q got %q",
				test.baseURL,
				test.apiVersion, test.namespace, test.resource, test.name,
				test.want, got,
			)
		}
	}
}

func TestUpdateKeys(t *testing.T) {
	fakeUpdater := func(old storage.Keys) (storage.Keys, error) { return storage.Keys{}, nil }

	tests := []struct {
		name               string
		updater            func(old storage.Keys) (storage.Keys, error)
		getResponseCode    int
		actionResponseCode int
		wantErr            bool
		exactErr           error
	}{
		{
			"Create OK test",
			fakeUpdater,
			404,
			201,
			false,
			nil,
		},
		{
			"Update should be OK",
			fakeUpdater,
			200,
			200,
			false,
			nil,
		},
		{
			"Create conflict should be OK",
			fakeUpdater,
			404,
			409,
			true,
			errors.New("keys already created by another server instance"),
		},
		{
			"Update conflict should be OK",
			fakeUpdater,
			200,
			409,
			true,
			errors.New("keys already rotated by another server instance"),
		},
		{
			"Client error is error",
			fakeUpdater,
			404,
			500,
			true,
			nil,
		},
		{
			"Client error during update is error",
			fakeUpdater,
			200,
			500,
			true,
			nil,
		},
		{
			"Get error is error",
			fakeUpdater,
			500,
			200,
			true,
			nil,
		},
		{
			"Updater error is error",
			func(old storage.Keys) (storage.Keys, error) { return storage.Keys{}, fmt.Errorf("test") },
			200,
			201,
			true,
			nil,
		},
	}

	for _, test := range tests {
		client := newStatusCodesResponseTestClient(test.getResponseCode, test.actionResponseCode)

		err := client.UpdateKeys(context.TODO(), test.updater)
		if err != nil {
			if !test.wantErr {
				t.Fatalf("Test %q: %v", test.name, err)
			}

			if test.exactErr != nil && test.exactErr.Error() != err.Error() {
				t.Fatalf("Test %q: %v, wanted: %v", test.name, err, test.exactErr)
			}
		}
	}
}

func newStatusCodesResponseTestClient(getResponseCode, actionResponseCode int) *client {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(getResponseCode)
		} else {
			w.WriteHeader(actionResponseCode)
		}
		w.Write([]byte(`{}`)) // Empty json is enough, we will test only response codes here
	}))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &client{
		client:  &http.Client{Transport: tr},
		baseURL: s.URL,
		logger:  slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})),
	}
}

func TestRetryOnConflict(t *testing.T) {
	tests := []struct {
		name     string
		action   func() error
		exactErr string
	}{
		{
			"Timeout reached",
			func() error { err := httpErr{status: 409}; return error(&err) },
			"maximum timeout reached while retrying a conflicted request:   Conflict: response from server \"\"",
		},
		{
			"HTTP Error",
			func() error { err := httpErr{status: 500}; return error(&err) },
			"  Internal Server Error: response from server \"\"",
		},
		{
			"Error",
			func() error { return errors.New("test") },
			"test",
		},
		{
			"OK",
			func() error { return nil },
			"",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := retryOnConflict(context.TODO(), testCase.action)
			if testCase.exactErr != "" {
				require.EqualError(t, err, testCase.exactErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRefreshTokenLock(t *testing.T) {
	ctx := context.Background()
	if os.Getenv(kubeconfigPathVariableName) == "" {
		t.Skipf("variable %q not set, skipping kubernetes storage tests\n", kubeconfigPathVariableName)
	}

	kubeconfigPath, err := expandDir(os.Getenv(kubeconfigPathVariableName))
	require.NoError(t, err)

	config := Config{
		KubeConfigFile: kubeconfigPath,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

	kubeClient, err := config.open(logger, true)
	require.NoError(t, err)

	lockCheckPeriod = time.Nanosecond

	// Creating a storage with an existing refresh token and offline session for the user.
	id := storage.NewID()
	r := storage.RefreshToken{
		ID:          id,
		Token:       "bar",
		Nonce:       "foo",
		ClientID:    "client_id",
		ConnectorID: "client_secret",
		Scopes:      []string{"openid", "email", "profile"},
		CreatedAt:   time.Now().UTC().Round(time.Millisecond),
		LastUsed:    time.Now().UTC().Round(time.Millisecond),
		Claims: storage.Claims{
			UserID:        "1",
			Username:      "jane",
			Email:         "jane.doe@example.com",
			EmailVerified: true,
			Groups:        []string{"a", "b"},
		},
		ConnectorData: []byte(`{"some":"data"}`),
	}

	err = kubeClient.CreateRefresh(ctx, r)
	require.NoError(t, err)

	t.Run("Timeout lock error", func(t *testing.T) {
		err = kubeClient.UpdateRefreshToken(ctx, r.ID, func(r storage.RefreshToken) (storage.RefreshToken, error) {
			r.Token = "update-result-1"
			err := kubeClient.UpdateRefreshToken(ctx, r.ID, func(r storage.RefreshToken) (storage.RefreshToken, error) {
				r.Token = "timeout-err"
				return r, nil
			})
			require.Equal(t, fmt.Errorf("timeout waiting for refresh token %s lock", r.ID), err)
			return r, nil
		})
		require.NoError(t, err)

		token, err := kubeClient.GetRefresh(context.TODO(), r.ID)
		require.NoError(t, err)
		require.Equal(t, "update-result-1", token.Token)
	})

	t.Run("Break the lock", func(t *testing.T) {
		var lockBroken bool
		lockTimeout = -time.Hour

		err = kubeClient.UpdateRefreshToken(ctx, r.ID, func(r storage.RefreshToken) (storage.RefreshToken, error) {
			r.Token = "update-result-2"
			if lockBroken {
				return r, nil
			}

			err := kubeClient.UpdateRefreshToken(ctx, r.ID, func(r storage.RefreshToken) (storage.RefreshToken, error) {
				r.Token = "should-break-the-lock-and-finish-updating"
				return r, nil
			})
			require.NoError(t, err)

			lockBroken = true
			return r, nil
		})
		require.NoError(t, err)

		token, err := kubeClient.GetRefresh(context.TODO(), r.ID)
		require.NoError(t, err)
		// Because concurrent update breaks the lock, the final result will be the value of the first update
		require.Equal(t, "update-result-2", token.Token)
	})
}
