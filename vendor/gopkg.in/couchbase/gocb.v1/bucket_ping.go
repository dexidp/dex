package gocb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/couchbase/gocbcore.v7"
)

func diagServiceString(service ServiceType) string {
	switch service {
	case MemdService:
		return "kv"
	case CapiService:
		return "view"
	case MgmtService:
		return "mgmt"
	case N1qlService:
		return "n1ql"
	case FtsService:
		return "fts"
	case CbasService:
		return "cbas"
	}
	return "?"
}

// PingServiceEntry represents a single entry in a ping report.
type PingServiceEntry struct {
	Service  ServiceType
	Endpoint string
	Success  bool
	Latency  time.Duration
}

// PingReport encapsulates the details from a executed ping operation.
type PingReport struct {
	Services []PingServiceEntry
}

type jsonPingServiceEntry struct {
	Remote    string `json:"remote"`
	LatencyUs uint64 `json:"latency_us"`
	Success   bool   `json:"success"`
}

type jsonPingReport struct {
	Version  int                               `json:"version"`
	Id       string                            `json:"id"`
	Sdk      string                            `json:"sdk"`
	Services map[string][]jsonPingServiceEntry `json:"services"`
}

// MarshalJSON generates a JSON representation of this ping report.
func (report *PingReport) MarshalJSON() ([]byte, error) {
	jsonReport := jsonPingReport{
		Version:  1,
		Id:       uuid.New().String(),
		Sdk:      "gocb/" + Version() + " " + "gocbcore/" + gocbcore.Version(),
		Services: make(map[string][]jsonPingServiceEntry),
	}

	for _, service := range report.Services {
		serviceStr := diagServiceString(service.Service)
		jsonReport.Services[serviceStr] = append(jsonReport.Services[serviceStr], jsonPingServiceEntry{
			Remote:    service.Endpoint,
			LatencyUs: uint64(service.Latency / time.Nanosecond),
		})
	}

	return json.Marshal(&jsonReport)
}

func (b *Bucket) pingKv() (pingsOut []gocbcore.PingResult, errOut error) {
	signal := make(chan bool, 1)

	op, err := b.client.Ping(func(results []gocbcore.PingResult) {
		pingsOut = make([]gocbcore.PingResult, len(results))
		for pingIdx, ping := range results {
			// We rewrite the cancelled errors into timeout errors here.
			if ping.Error == gocbcore.ErrCancelled {
				ping.Error = ErrTimeout
			}
			pingsOut[pingIdx] = ping
		}
		signal <- true
	})
	if err != nil {
		return nil, err
	}

	timeoutTmr := gocbcore.AcquireTimer(b.opTimeout)
	select {
	case <-signal:
		gocbcore.ReleaseTimer(timeoutTmr, false)
		return
	case <-timeoutTmr.C:
		gocbcore.ReleaseTimer(timeoutTmr, true)
		if !op.Cancel() {
			<-signal
			return
		}
		return nil, ErrTimeout
	}
}

// Ping will ping a list of services and verify they are active and
// responding in an acceptable period of time.
//
// Experimental: This API is subject to change at any time.
func (b *Bucket) Ping(services []ServiceType) (*PingReport, error) {
	numServices := 0
	waitCh := make(chan error, 10)
	report := &PingReport{}
	var reportLock sync.Mutex

	if services == nil {
		services = []ServiceType{
			MemdService,
		}
		if b.client.N1qlEps() != nil {
			services = append(services, N1qlService)
		}
		if b.client.FtsEps() != nil {
			services = append(services, FtsService)
		}
		if b.client.CbasEps() != nil {
			services = append(services, CbasService)
		}
	}

	httpReq := func(service ServiceType, endpoint, url string) (time.Duration, error) {
		c := b.cluster

		startTime := time.Now()

		client := b.client.HttpClient()

		reqUri := fmt.Sprintf("%s/%s", endpoint, url)
		req, err := http.NewRequest("GET", reqUri, nil)
		if err != nil {
			return 0, err
		}

		timeout := 60 * time.Second
		if service == N1qlService {
			if b.n1qlTimeout < c.n1qlTimeout {
				timeout = b.n1qlTimeout
			} else {
				timeout = c.n1qlTimeout
			}
		} else if service == FtsService {
			if b.ftsTimeout < c.ftsTimeout {
				timeout = b.ftsTimeout
			} else {
				timeout = c.ftsTimeout
			}
		} else if service == CbasService {
			timeout = c.analyticsTimeout
		}

		resp, err := doHttpWithTimeout(client, req, timeout)
		if err != nil {
			return 0, err
		}

		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close http request: %s", err)
		}

		pingLatency := time.Now().Sub(startTime)

		return pingLatency, err
	}

	for _, serviceType := range services {
		switch serviceType {
		case MemdService:
			numServices++
			go func() {
				pings, err := b.pingKv()
				if err != nil {
					logWarnf("Failed to ping KV for report: %s", err)
					waitCh <- nil
					return
				}

				reportLock.Lock()
				// We intentionally ignore errors here and simply include
				// any non-error pings that we have received.  Note that
				// gocbcore's ping command, when cancelled, still returns
				// any pings that had occurred before the operation was
				// cancelled and then marks the rest as errors.
				for _, ping := range pings {
					wasSuccess := true
					if ping.Error != nil {
						wasSuccess = false
					}

					report.Services = append(report.Services, PingServiceEntry{
						Service:  MemdService,
						Endpoint: ping.Endpoint,
						Success:  wasSuccess,
						Latency:  ping.Latency,
					})
				}
				reportLock.Unlock()
				waitCh <- nil
			}()
		case CapiService:
			// View Service is not currently supported as a ping target
		case N1qlService:
			numServices++
			go func() {
				pingLatency := time.Duration(0)

				endpoint, err := b.getN1qlEp()
				if err == nil {
					pingLatency, err = httpReq(N1qlService, endpoint, "/admin/ping")
				}

				reportLock.Lock()
				if err != nil {
					report.Services = append(report.Services, PingServiceEntry{
						Service:  N1qlService,
						Endpoint: endpoint,
						Success:  false,
					})
				} else {
					report.Services = append(report.Services, PingServiceEntry{
						Service:  N1qlService,
						Endpoint: endpoint,
						Success:  true,
						Latency:  pingLatency,
					})
				}
				reportLock.Unlock()

				waitCh <- nil
			}()
		case FtsService:
			numServices++
			go func() {
				pingLatency := time.Duration(0)

				endpoint, err := b.getFtsEp()
				if err == nil {
					pingLatency, err = httpReq(FtsService, endpoint, "/api/ping")
				}

				reportLock.Lock()
				if err != nil {
					report.Services = append(report.Services, PingServiceEntry{
						Service:  FtsService,
						Endpoint: endpoint,
						Success:  false,
					})
				} else {
					report.Services = append(report.Services, PingServiceEntry{
						Service:  FtsService,
						Endpoint: endpoint,
						Success:  true,
						Latency:  pingLatency,
					})
				}
				reportLock.Unlock()

				waitCh <- nil
			}()
		case CbasService:
			numServices++
			go func() {
				pingLatency := time.Duration(0)

				endpoint, err := b.getCbasEp()
				if err == nil {
					pingLatency, err = httpReq(CbasService, endpoint, "/admin/ping")
				}

				reportLock.Lock()
				if err != nil {
					report.Services = append(report.Services, PingServiceEntry{
						Service:  CbasService,
						Endpoint: endpoint,
						Success:  false,
					})
				} else {
					report.Services = append(report.Services, PingServiceEntry{
						Service:  CbasService,
						Endpoint: endpoint,
						Success:  true,
						Latency:  pingLatency,
					})
				}
				reportLock.Unlock()

				waitCh <- nil
			}()
		}
	}

	for i := 0; i < numServices; i++ {
		<-waitCh
	}

	return report, nil
}
