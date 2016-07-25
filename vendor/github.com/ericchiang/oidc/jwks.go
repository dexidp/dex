package oidc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pquerna/cachecontrol"
	"golang.org/x/net/context"
	jose "gopkg.in/square/go-jose.v1"
)

// No matter what insist on caching keys. This is so our request code can be
// asynchronous from matching keys. If the request code retrieved keys that
// expired immediately, the goroutine to match a JWT to a key would always see
// expired keys.
//
// TODO(ericchiang): Review this logic.
var minCache = 2 * time.Minute

type cachedKeys struct {
	keys   map[string]jose.JsonWebKey // immutable
	expiry time.Time
}

type remoteKeySet struct {
	client *http.Client

	// "jwks_uri" from discovery.
	keysURL string

	// The value is always of type *cachedKeys.
	//
	// To ensure consistency always call keyCache.Store when holding cond.L.
	keyCache atomic.Value

	// cond.L guards all following fields. sync.Cond is used in place of a mutex
	// so multiple processes can wait on a single request to update keys.
	cond sync.Cond
	// Is there an existing request to get the remote keys?
	inflight bool
	// If the last attempt to refresh keys failed, the error will be saved here.
	//
	// TODO(ericchiang): If a routine sets this before calling cond.Broadcast(),
	// there's no guarentee that a routine calling cond.Wait() will actual see
	// the error called by the previous routine. Since Broadcast() unlocks
	// cond.L and Wait() must reacquire the lock, other routines waiting on the
	// lock might acquire it first. Maybe just log the error?
	lastErr error
}

func newRemoteKeySet(ctx context.Context, jwksURL string) *remoteKeySet {
	r := &remoteKeySet{
		client:  contextClient(ctx),
		keysURL: jwksURL,
		cond:    sync.Cond{L: new(sync.Mutex)},
	}
	return r
}

func (r *remoteKeySet) verifyJWT(jwt string) (payload []byte, err error) {
	jws, err := jose.ParseSigned(jwt)
	if err != nil {
		return nil, fmt.Errorf("parsing jwt: %v", err)
	}
	keyIDs := make([]string, len(jws.Signatures))
	for i, signature := range jws.Signatures {
		keyIDs[i] = signature.Header.KeyID
	}
	key, err := r.getKey(keyIDs)
	if err != nil {
		return nil, fmt.Errorf("oidc: %s", err)
	}
	return jws.Verify(key)
}

func (r *remoteKeySet) getKeyFromCache(keyIDs []string) (*jose.JsonWebKey, bool) {
	cachedKeys, ok := r.keyCache.Load().(*cachedKeys)
	if !ok {
		return nil, false
	}
	if time.Now().After(cachedKeys.expiry) {
		return nil, false
	}
	for _, keyID := range keyIDs {
		if key, ok := cachedKeys.keys[keyID]; ok {
			return &key, true
		}
	}
	return nil, false
}

func (r *remoteKeySet) getKey(keyIDs []string) (*jose.JsonWebKey, error) {
	// Fast path. Just do an atomic load.
	if key, ok := r.getKeyFromCache(keyIDs); ok {
		return key, nil
	}

	// Didn't find keys, use the slow path.
	r.cond.L.Lock()
	defer r.cond.L.Unlock()

	// Check again within the mutex.
	if key, ok := r.getKeyFromCache(keyIDs); ok {
		return key, nil
	}

	// Keys have expired or we're trying to verify a JWT we don't have a key for.

	if !r.inflight {
		// There isn't currently an inflight request to update keys, start a
		// goroutine to do so.
		r.inflight = true
		go func() {
			newKeys, newExpiry, err := requestKeys(r.client, r.keysURL)

			r.cond.L.Lock()
			defer r.cond.L.Unlock()

			r.inflight = false
			if err != nil {
				r.lastErr = err
			} else {
				r.keyCache.Store(&cachedKeys{newKeys, newExpiry})
				r.lastErr = nil
			}

			r.cond.Broadcast() // Wake all r.cond.Wait() calls.
		}()
	}

	// Wait for r.cond.Broadcast() to be called. This unlocks r.cond.L and
	// reacquires it after its done waiting.
	r.cond.Wait()

	if key, ok := r.getKeyFromCache(keyIDs); ok {
		return key, nil
	}
	if r.lastErr != nil {
		return nil, r.lastErr
	}
	return nil, errors.New("no signing keys can validate the signature")
}

func requestKeys(client *http.Client, keysURL string) (map[string]jose.JsonWebKey, time.Time, error) {
	req, err := http.NewRequest("GET", keysURL, nil)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("can't create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("can't GET new keys %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("can't fetch new keys: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("can't fetch new keys: %s %s", resp.Status, body)
	}

	var keySet jose.JsonWebKeySet
	if err := json.Unmarshal(body, &keySet); err != nil {
		return nil, time.Time{}, fmt.Errorf("can't decode keys: %v %s", err, body)
	}

	keys := make(map[string]jose.JsonWebKey, len(keySet.Keys))
	for _, key := range keySet.Keys {
		keys[key.KeyID] = key
	}

	minExpiry := time.Now().Add(minCache)

	if _, expiry, err := cachecontrol.CachableResponse(req, resp, cachecontrol.Options{}); err == nil {
		if minExpiry.Before(expiry) {
			return keys, expiry, nil
		}
	}
	return keys, minExpiry, nil
}
