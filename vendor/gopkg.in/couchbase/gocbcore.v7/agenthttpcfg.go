package gocbcore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type configStreamBlock struct {
	Bytes []byte
}

func (i *configStreamBlock) UnmarshalJSON(data []byte) error {
	i.Bytes = make([]byte, len(data))
	copy(i.Bytes, data)
	return nil
}

func hostnameFromUri(uri string) string {
	uriInfo, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	hostname, err := hostFromHostPort(uriInfo.Host)
	if err != nil {
		return uri
	}

	return hostname
}

func (agent *Agent) httpLooper(firstCfgFn func(*cfgBucket, string, error) bool) {
	waitPeriod := agent.confHttpRetryDelay
	maxConnPeriod := agent.confHttpRedialPeriod

	var iterNum uint64 = 1
	iterSawConfig := false
	seenNodes := make(map[string]uint64)
	isFirstTry := true

	logDebugf("HTTP Looper starting.")
	for {
		routingInfo := agent.routingInfo.Get()
		if routingInfo == nil {
			// Shutdown the looper if the agent is shutdown
			break
		}

		var pickedSrv string
		for _, srv := range routingInfo.mgmtEpList {
			if seenNodes[srv] >= iterNum {
				continue
			}
			pickedSrv = srv
			break
		}

		if pickedSrv == "" {
			logDebugf("Pick Failed.")
			// All servers have been visited during this iteration
			if isFirstTry {
				logDebugf("Could not find any alive http hosts.")
				firstCfgFn(nil, "", ErrBadHosts)
				break
			}

			if !iterSawConfig {
				logDebugf("Looper waiting...")
				// Wait for a period before trying again if there was a problem...
				// We also watch for the client being shut down.
				select {
				case <-time.After(waitPeriod):
				case <-agent.closeNotify:
				}
			}
			logDebugf("Looping again.")
			// Go to next iteration and try all servers again
			iterNum++
			iterSawConfig = false
			continue
		}

		logDebugf("Http Picked: %s.", pickedSrv)

		seenNodes[pickedSrv] = iterNum

		hostname := hostnameFromUri(pickedSrv)
		logDebugf("HTTP Hostname: %s.", hostname)

		var resp *http.Response
		// 1 on success, 0 on failure for node, -1 for generic failure
		var doConfigRequest func(bool) int

		doConfigRequest = func(is2x bool) int {
			streamPath := "bs"
			if is2x {
				streamPath = "bucketsStreaming"
			}
			// HTTP request time!
			uri := fmt.Sprintf("%s/pools/default/%s/%s", pickedSrv, streamPath, agent.bucket)
			logDebugf("Requesting config from: %s.", uri)

			req, err := http.NewRequest("GET", uri, nil)
			if err != nil {
				logDebugf("Failed to build HTTP config request. %v", err)
				return 0
			}

			creds, err := getMgmtAuthCreds(agent.auth, pickedSrv)
			if err != nil {
				logDebugf("Failed to build get config credentials. %v", err)
				return 0
			}

			req.SetBasicAuth(creds.Username, creds.Password)

			resp, err = agent.httpCli.Do(req)
			if err != nil {
				logDebugf("Failed to connect to host. %v", err)
				return 0
			}

			if resp.StatusCode != 200 {
				if resp.StatusCode == 401 {
					logDebugf("Failed to connect to host, bad auth.")
					firstCfgFn(nil, "", ErrAuthError)
					return -1
				} else if resp.StatusCode == 404 {
					if is2x {
						logDebugf("Failed to connect to host, bad bucket.")
						firstCfgFn(nil, "", ErrAuthError)
						return -1
					}

					return doConfigRequest(true)
				}
				logDebugf("Failed to connect to host, unexpected status code: %v.", resp.StatusCode)
				return 0
			}
			return 1
		}

		switch doConfigRequest(false) {
		case 0:
			continue
		case -1:
			break
		}

		logDebugf("Connected.")

		var autoDisconnected int32

		// Autodisconnect eventually
		go func() {
			select {
			case <-time.After(maxConnPeriod):
			case <-agent.closeNotify:
			}

			logDebugf("Automatically resetting our HTTP connection")

			atomic.StoreInt32(&autoDisconnected, 1)

			err := resp.Body.Close()
			if err != nil {
				logErrorf("Socket close failed during auto-dc (%s)", err)
			}
		}()

		dec := json.NewDecoder(resp.Body)
		configBlock := new(configStreamBlock)
		for {
			err := dec.Decode(configBlock)
			if err != nil {
				if atomic.LoadInt32(&autoDisconnected) == 1 {
					// If we know we intentionally disconnected, we know we do not
					// need to close the client, nor log an error, since this was
					// expected behaviour
					break
				}

				logWarnf("Config block decode failure (%s)", err)

				if err != io.EOF {
					err = resp.Body.Close()
					if err != nil {
						logErrorf("Socket close failed after decode fail (%s)", err)
					}
				}

				break
			}

			logDebugf("Got Block: %v", string(configBlock.Bytes))

			bkCfg, err := parseConfig(configBlock.Bytes, hostname)
			if err != nil {
				logDebugf("Got error while parsing config: %v", err)

				err = resp.Body.Close()
				if err != nil {
					logErrorf("Socket close failed after parsing fail (%s)", err)
				}

				break
			}

			logDebugf("Got Config.")

			iterSawConfig = true
			if isFirstTry {
				logDebugf("HTTP Config Init")
				if !firstCfgFn(bkCfg, pickedSrv, nil) {
					logDebugf("Got error while activating first config")

					err = resp.Body.Close()
					if err != nil {
						logErrorf("Socket close failed after config init (%s)", err)
					}

					break
				}
				isFirstTry = false
			} else {
				logDebugf("HTTP Config Update")
				agent.updateConfig(bkCfg)
			}
		}

		logDebugf("HTTP, Setting %s to iter %d", pickedSrv, iterNum)
	}

	close(agent.httpLooperDoneSig)
}
