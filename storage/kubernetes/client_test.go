package kubernetes

import (
	"hash"
	"hash/fnv"
	"sync"
	"testing"
)

// This test does not have an explicit error condition but is used
// with the race detector to detect the safety of idToName.
func TestIDToName(t *testing.T) {
	n := 100
	var wg sync.WaitGroup
	wg.Add(n)
	c := make(chan struct{})

	h := func() hash.Hash { return fnv.New64() }

	for i := 0; i < n; i++ {
		go func() {
			<-c
			name := idToName("foo", h)
			_ = name
			wg.Done()
		}()
	}
	close(c)
	wg.Wait()
}

func TestOfflineTokenName(t *testing.T) {
	h := func() hash.Hash { return fnv.New64() }

	userID1 := "john"
	userID2 := "jane"

	id1 := offlineTokenName(userID1, "local", h)
	id2 := offlineTokenName(userID2, "local", h)
	if id1 == id2 {
		t.Errorf("expected offlineTokenName to produce different hashes")
	}
}

func TestNamespaceFromServiceAccountJWT(t *testing.T) {
	namespace, err := namespaceFromServiceAccountJWT(serviceAccountToken)
	if err != nil {
		t.Fatal(err)
	}
	wantNamespace := "dex-test-namespace"
	if namespace != wantNamespace {
		t.Errorf("expected namespace %q got %q", wantNamespace, namespace)
	}
}

var serviceAccountToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJkZXgtdGVzdC1uYW1lc3BhY2UiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlY3JldC5uYW1lIjoiZG90aGVyb2JvdC1zZWNyZXQiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoiZG90aGVyb2JvdCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjQyYjJhOTRmLTk4MjAtMTFlNi1iZDc0LTJlZmQzOGYxMjYxYyIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpkZXgtdGVzdC1uYW1lc3BhY2U6ZG90aGVyb2JvdCJ9.KViBpPwCiBwxDvAjYUUXoVvLVwqV011aLlYQpNtX12Bh8M-QAFch-3RWlo_SR00bcdFg_nZo9JKACYlF_jHMEsf__PaYms9r7vEaSg0jPfkqnL2WXZktzQRyLBr0n-bxeUrbwIWsKOAC0DfFB5nM8XoXljRmq8yAx8BAdmQp7MIFb4EOV9nYthhua6pjzYyaFSiDiYTjw7HtXOvoL8oepodJ3-37pUKS8vdBvnvUoqC4M1YAhkO5L36JF6KV_RfmG8GPEdNQfXotHcsR-3jKi1n8S5l7Xd-rhrGOhSGQizH3dORzo9GvBAhYeqbq1O-NLzm2EQUiMQayIUx7o4g3Kw"

// The following program was used to generate the example token. Since we don't want to
// import Kubernetes, just leave it as a comment.

/*
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/serviceaccount"
	"k8s.io/kubernetes/pkg/util/uuid"
)

func main() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}
	sa := api.ServiceAccount{
		ObjectMeta: api.ObjectMeta{
			Namespace: "dex-test-namespace",
			Name:      "dotherobot",
			UID:       uuid.NewUUID(),
		},
	}
	secret := api.Secret{
		ObjectMeta: api.ObjectMeta{
			Namespace: "dex-test-namespace",
			Name:      "dotherobot-secret",
			UID:       uuid.NewUUID(),
		},
	}
	token, err := serviceaccount.JWTTokenGenerator(key).GenerateToken(sa, secret)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token)
}
*/
