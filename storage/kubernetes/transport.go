package kubernetes

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
)

// transport is a simple http.Transport wrapper
type transport struct {
	updateReq func(r *http.Request)
	base      http.RoundTripper
}

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	t.updateReq(r2)
	return t.base.RoundTrip(r2)
}

func wrapRoundTripper(base http.RoundTripper, user k8sapi.AuthInfo, inCluster bool) http.RoundTripper {
	if inCluster {
		inClusterTransportHelper := newInClusterTransportHelper(user)
		return transport{
			updateReq: func(r *http.Request) {
				inClusterTransportHelper.UpdateToken()
				r.Header.Set("Authorization", "Bearer "+inClusterTransportHelper.GetToken())
			},
			base: base,
		}
	}

	if user.Token != "" {
		return transport{
			updateReq: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer "+user.Token)
			},
			base: base,
		}
	}

	if user.Username != "" && user.Password != "" {
		return transport{
			updateReq: func(r *http.Request) {
				r.SetBasicAuth(user.Username, user.Password)
			},
			base: base,
		}
	}

	return base
}

// renewTokenPeriod is the interval after which dex will read the token from a well-known file.
//   By Kubernetes documentation, this interval should be at least one minute long.
//   Kubernetes client-go v0.15+ uses 10 seconds long interval.
//   Dex uses the reasonable value between these two.
const renewTokenPeriod = 30 * time.Second

// inClusterTransportHelper is capable of safely updating the user token.
//   BoundServiceAccountTokenVolume feature is enabled in Kubernetes >=1.21 by default.
//   With this feature, the service account token in the pod becomes periodically updated.
//   Therefore, Dex needs to re-read the token from the disk after some time to be sure that it uses the valid token.
type inClusterTransportHelper struct {
	mu   sync.RWMutex
	info k8sapi.AuthInfo

	expiry time.Time
	now    func() time.Time

	tokenLocation string
}

func newInClusterTransportHelper(info k8sapi.AuthInfo) *inClusterTransportHelper {
	user := &inClusterTransportHelper{
		info:          info,
		now:           time.Now,
		tokenLocation: "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}

	user.UpdateToken()

	return user
}

func (c *inClusterTransportHelper) UpdateToken() {
	c.mu.RLock()
	exp := c.expiry
	c.mu.RUnlock()

	if !c.now().After(exp) {
		// Do not need to update token yet
		return
	}

	token, err := os.ReadFile(c.tokenLocation)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.info.Token = string(token)
	c.expiry = c.now().Add(renewTokenPeriod)
}

func (c *inClusterTransportHelper) GetToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.info.Token
}
