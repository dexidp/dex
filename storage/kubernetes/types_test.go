package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

// scripts/manifests/crds holds a manifest for every CRD the storage
// registers, so a deployment that pre-creates CRDs (crdHandling: check, or
// a crdHandling: skip flow that relies on CRDs being pre-created) can
// install the complete set. This test keeps the folder and
// customResourceDefinitions() in sync, so a CRD added to the storage cannot
// ship without a manifest again.
func TestManifestsMatchCustomResourceDefinitions(t *testing.T) {
	crds := customResourceDefinitions(crdAPIVersion)
	require.NotEmpty(t, crds)

	dir := filepath.Join("..", "..", "scripts", "manifests", "crds")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	manifests := make(map[string]k8sapi.CustomResourceDefinition, len(entries))
	matched := make(map[string]bool, len(entries))
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
		matched[entry.Name()] = false
	}

	for _, crd := range crds {
		name := crd.Spec.Names.Plural + ".yaml"

		file, ok := manifests[name]
		if !ok {
			t.Errorf("scripts/manifests/crds/%s is missing: no manifest for the CRD %q registered by customResourceDefinitions()", name, crd.ObjectMeta.Name)
			continue
		}
		matched[name] = true

		want := crd
		// customResourceDefinitions does not set ListKind: the API server
		// defaults it to <kind>List when the CRD is created. The committed
		// manifests spell that default out, so accept it on the file side.
		if want.Spec.Names.ListKind == "" {
			want.Spec.Names.ListKind = want.Spec.Names.Kind + "List"
		}
		require.Equal(t, want, file, "scripts/manifests/crds/%s does not match the CRD registered by the storage", name)
	}

	for name, ok := range matched {
		if !ok {
			t.Errorf("scripts/manifests/crds/%s has no matching CRD in customResourceDefinitions()", name)
		}
	}
}
