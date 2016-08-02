package kubernetes

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"
)

// gc begins the gc process for Kubernetes.
func (cli *client) gc(ctx context.Context, every time.Duration) {
	handleErr := func(err error) { log.Println(err.Error()) }

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}

		// TODO(ericchiang): On failures, run garbage collection more often.
		log.Println("kubernetes: running garbage collection")
		cli.gcAuthRequests(handleErr)
		cli.gcAuthCodes(handleErr)
		log.Printf("kubernetes: garbage collection finished, next run at %s", cli.now().Add(every))
	}
}

func (cli *client) gcAuthRequests(handleErr func(error)) {
	var authRequests AuthRequestList
	if err := cli.list(resourceAuthRequest, &authRequests); err != nil {
		handleErr(fmt.Errorf("failed to list auth requests: %v", err))
		return
	}
	for _, authRequest := range authRequests.AuthRequests {
		if cli.now().After(authRequest.Expiry) {
			if err := cli.delete(resourceAuthRequest, authRequest.ObjectMeta.Name); err != nil {
				handleErr(fmt.Errorf("failed to detele auth request: %v", err))
			}
		}
	}
}

func (cli *client) gcAuthCodes(handleErr func(error)) {
	var authCodes AuthCodeList
	if err := cli.list(resourceAuthCode, &authCodes); err != nil {
		handleErr(fmt.Errorf("failed to list auth codes: %v", err))
		return
	}
	for _, authCode := range authCodes.AuthCodes {
		if cli.now().After(authCode.Expiry) {
			if err := cli.delete(resourceAuthCode, authCode.ObjectMeta.Name); err != nil {
				handleErr(fmt.Errorf("failed to delete auth code: %v", err))
			}
		}
	}
}
