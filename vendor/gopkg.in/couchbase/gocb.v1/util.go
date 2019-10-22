package gocb

import (
	"net/http"
	"time"
)

// Wrapper around net.http.Client.Do().
// This allows a per-request timeout without setting the timeout on the Client object
// directly.
// The third parameter is the duration for the request itself.
func doHttpWithTimeout(cli *http.Client, req *http.Request, timeout time.Duration) (resp *http.Response, err error) {
	if timeout.Seconds() == 0 {
		// No timeout
		resp, err = cli.Do(req)
		return
	}

	tmoch := make(chan struct{})
	timer := time.AfterFunc(timeout, func() {
		tmoch <- struct{}{}
	})

	req.Cancel = tmoch
	resp, err = cli.Do(req)
	timer.Stop()
	return
}
