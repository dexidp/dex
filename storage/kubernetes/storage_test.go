package kubernetes

import (
	"fmt"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/dex/storage"
	"github.com/coreos/dex/storage/conformance"
)

const testKubeConfigEnv = "DEX_KUBECONFIG"

func TestLoadClient(t *testing.T) {
	loadClient(t)
}

func loadClient(t *testing.T) *client {
	config := Config{
		KubeConfigFile: os.Getenv(testKubeConfigEnv),
	}
	if config.KubeConfigFile == "" {
		t.Skipf("test environment variable %q not set, skipping", testKubeConfigEnv)
	}
	logger := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: &logrus.TextFormatter{DisableColors: true},
		Level:     logrus.DebugLevel,
	}
	s, err := config.open(logger, true)
	if err != nil {
		t.Fatal(err)
	}
	return s
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
		got := c.urlFor(test.apiVersion, test.namespace, test.resource, test.name)
		if got != test.want {
			t.Errorf("(&client{baseURL:%q}).urlFor(%q, %q, %q, %q): expected %q got %q",
				test.baseURL,
				test.apiVersion, test.namespace, test.resource, test.name,
				test.want, got,
			)
		}
	}
}

func TestStorage(t *testing.T) {
	client := loadClient(t)
	newStorage := func() storage.Storage {
		for _, resource := range []string{
			resourceAuthCode,
			resourceAuthRequest,
			resourceClient,
			resourceRefreshToken,
			resourceKeys,
			resourcePassword,
		} {
			if err := client.deleteAll(resource); err != nil {
				// Fatalf sometimes doesn't print the error message.
				fmt.Fprintf(os.Stderr, "delete all %q failed: %v\n", resource, err)
				t.Fatalf("delete all %q failed: %v", resource, err)
			}
		}
		return client
	}

	conformance.RunTests(t, newStorage)
	conformance.RunTransactionTests(t, newStorage)
}
