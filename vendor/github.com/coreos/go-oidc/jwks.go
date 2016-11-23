package oidc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pquerna/cachecontrol"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
	jose "gopkg.in/square/go-jose.v2"
)

// keysExpiryDelta is the allowed clock skew between a client and the OpenID Connect
// server.
//
// When keys expire, they are valid for this amount of time after.
//
// If the keys have not expired, and an ID Token claims it was signed by a key not in
// the cache, if and only if the keys expire in this amount of time, the keys will be
// updated.
const keysExpiryDelta = 30 * time.Second

func newRemoteKeySet(ctx context.Context, jwksURL string, now func() time.Time) *remoteKeySet {
	if now == nil {
		now = time.Now
	}
	return &remoteKeySet{jwksURL: jwksURL, ctx: ctx, now: now}
}

type remoteKeySet struct {
	jwksURL string
	ctx     context.Context
	now     func() time.Time

	// guard all other fields
	mu sync.Mutex

	// inflightCtx is the context of the current HTTP request to update the keys.
	// Its Err() method returns any errors encountered during that attempt.
	//
	// If nil, there is no inflight request.
	inflightCtx context.Context

	// A set of cached keys and their expiry.
	cachedKeys []jose.JSONWebKey
	expiry     time.Time
}

// errContext is a context with a customizable Err() return value.
type errContext struct {
	context.Context

	cf  context.CancelFunc
	err error
}

func newErrContext(parent context.Context) *errContext {
	ctx, cancel := context.WithCancel(parent)
	return &errContext{ctx, cancel, nil}
}

func (e errContext) Err() error {
	return e.err
}

// cancel cancels the errContext causing listeners on Done() to return.
func (e errContext) cancel(err error) {
	e.err = err
	e.cf()
}

func (r *remoteKeySet) keysWithIDFromCache(keyIDs []string) ([]jose.JSONWebKey, bool) {
	r.mu.Lock()
	keys, expiry := r.cachedKeys, r.expiry
	r.mu.Unlock()

	// Have the keys expired?
	if expiry.Add(keysExpiryDelta).Before(r.now()) {
		return nil, false
	}

	var signingKeys []jose.JSONWebKey
	for _, key := range keys {
		if contains(keyIDs, key.KeyID) {
			signingKeys = append(signingKeys, key)
		}
	}

	if len(signingKeys) == 0 {
		// Are the keys about to expire?
		if r.now().Add(keysExpiryDelta).After(expiry) {
			return nil, false
		}
	}

	return signingKeys, true
}
func (r *remoteKeySet) keysWithID(ctx context.Context, keyIDs []string) ([]jose.JSONWebKey, error) {
	keys, ok := r.keysWithIDFromCache(keyIDs)
	if ok {
		return keys, nil
	}

	var inflightCtx context.Context
	func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		// If there's not a current inflight request, create one.
		if r.inflightCtx == nil {
			// Use the remoteKeySet's context instead of the requests context
			// because a re-sync is unique to the keys set and will span multiple
			// requests.
			errCtx := newErrContext(r.ctx)
			r.inflightCtx = errCtx

			go func() {
				// TODO(ericchiang): Upstream Kubernetes request that we recover every time
				// we spawn a goroutine, because panics in a goroutine will bring down the
				// entire program. There's no way to recover from another goroutine's panic.
				//
				// Most users actually want to let the panic propagate and bring down the
				// program because it implies some unrecoverable state.
				//
				// Add a context key to allow the recover behavior.
				//
				// See: https://github.com/coreos/go-oidc/issues/89

				// Sync keys and close inflightCtx when that's done.
				errCtx.cancel(r.updateKeys(r.inflightCtx))

				r.mu.Lock()
				defer r.mu.Unlock()
				r.inflightCtx = nil
			}()
		}

		inflightCtx = r.inflightCtx
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-inflightCtx.Done():
		if err := inflightCtx.Err(); err != nil {
			return nil, err
		}
	}

	// Since we've just updated keys, we don't care about the cache miss.
	keys, _ = r.keysWithIDFromCache(keyIDs)
	return keys, nil
}

func (r *remoteKeySet) updateKeys(ctx context.Context) error {
	req, err := http.NewRequest("GET", r.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("oidc: can't create request: %v", err)
	}

	resp, err := ctxhttp.Do(ctx, clientFromContext(ctx), req)
	if err != nil {
		return fmt.Errorf("oidc: get keys failed %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("oidc: read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oidc: get keys failed: %s %s", resp.Status, body)
	}

	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal(body, &keySet); err != nil {
		return fmt.Errorf("oidc: failed to decode keys: %v %s", err, body)
	}

	// If the server doesn't provide cache control headers, assume the
	// keys expire immediately.
	expiry := r.now()

	_, e, err := cachecontrol.CachableResponse(req, resp, cachecontrol.Options{})
	if err == nil && e.After(expiry) {
		expiry = e
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.cachedKeys = keySet.Keys
	r.expiry = expiry

	return nil
}
