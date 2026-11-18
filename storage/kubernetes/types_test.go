package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

// Every CRD the storage registers must have a matching manifest in scripts/manifests/crds.
func TestManifestsMatchCustomResourceDefinitions(t *testing.T) {
	crds := customResourceDefinitions(crdAPIVersion)
	require.NotEmpty(t, crds)

	dir := filepath.Join("..", "..", "scripts", "manifests", "crds")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	manifests := make(map[string]k8sapi.CustomResourceDefinition, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)

		var crd k8sapi.CustomResourceDefinition
		err = yaml.Unmarshal(data, &crd)
		require.NoError(t, err, "unmarshal %s", entry.Name())

		manifests[entry.Name()] = crd
	}

	for _, crd := range crds {
		name := crd.Spec.Names.Plural + ".yaml"

		file, ok := manifests[name]
		if !ok {
			t.Errorf("scripts/manifests/crds/%s is missing: no manifest for the CRD %q registered by customResourceDefinitions()", name, crd.ObjectMeta.Name)
			continue
		}
		delete(manifests, name)

		want := crd
		// customResourceDefinitions does not set ListKind: the API server
		// defaults it to <kind>List when the CRD is created. The committed
		// manifests spell that default out, so accept it on the file side.
		if want.Spec.Names.ListKind == "" {
			want.Spec.Names.ListKind = want.Spec.Names.Kind + "List"
		}
		assert.Equal(t, want, file, "scripts/manifests/crds/%s does not match the CRD registered by the storage", name)
	}

	for name := range manifests {
		t.Errorf("scripts/manifests/crds/%s has no matching CRD in customResourceDefinitions()", name)
	}
}
