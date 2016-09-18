package kubernetes

import (
	"os"
	"testing"

	"github.com/coreos/dex/storage/conformance"
)

func TestLoadClient(t *testing.T) {
	loadClient(t)
}

func loadClient(t *testing.T) *client {
	if os.Getenv("KUBECONFIG") == "" {
		t.Skip()
	}
	var config Config
	s, err := config.open()
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
		c := &client{baseURL: test.baseURL, prependResourceNameToAPIGroup: false}
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
	conformance.RunTestSuite(t, client)
}
