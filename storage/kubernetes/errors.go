package kubernetes

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/dexidp/dex/storage"
)

const (
	errCouldNotCreateCRD                      = "failed creating custom resources definitions"
	errCouldNotCreateK8sClient                = "could not create a kubernetes client"
	errCRDsNotReady                           = "gave up waiting for CRDs to become active in kubernetes cluster"
	errSpecifiedBothKubeConfigAndInCluster    = "cannot specify both 'inCluster' and 'kubeConfigFile'"
	errSpecifiedNeitherKubeConfigAndInCluster = "must specify either 'inCluster' or 'kubeConfigFile'"
	errWrongSessionRetrieved                  = "get offline session: wrong session retrieved"
)

// checkHTTPErr will do a best effort to convert from an http response to a storage.Error
func checkHTTPErr(r *http.Response, validStatusCodes ...int) error {
	for _, status := range validStatusCodes {
		if r.StatusCode == status {
			return nil
		}
	}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 2<<15)) // 64 KiB
	if err != nil {
		return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("read response body: %v", err)}
	}

	// Check this case after we read the body so the connection can be reused.
	if r.StatusCode == http.StatusNotFound {
		return storage.Error{Code: storage.ErrNotFound}
	}
	if r.Request.Method == http.MethodPost && r.StatusCode == http.StatusConflict {
		return storage.Error{Code: storage.ErrAlreadyExists}
	}

	var url, method string
	if r.Request != nil {
		method = r.Request.Method
		url = r.Request.URL.String()
	}
	return storage.Error{Code: storage.ErrStorageProviderInternalError, Details: fmt.Sprintf("%s %s %s: response from server \"%s\"", method, url, http.StatusText(r.StatusCode), bytes.TrimSpace(body))}
}
