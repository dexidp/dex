package kubernetes

import (
	"fmt"
	"log"
)

// TODO(ericchiang): Complete this.

type multiErr []error

func (m multiErr) Error() string {
	return fmt.Sprintf("errors encountered: %s", m)
}

func (cli *client) gcAuthRequests() error {
	var authRequests AuthRequestList
	if err := cli.list(resourceAuthRequest, &authRequests); err != nil {
		return err
	}
	for _, authRequest := range authRequests.AuthRequests {
		if cli.now().After(authRequest.Expiry) {
			if err := cli.delete(resourceAuthRequest, authRequest.ObjectMeta.Name); err != nil {
				log.Printf("failed to detele auth request: %v", err)
			}
		}
	}
	return nil
}
