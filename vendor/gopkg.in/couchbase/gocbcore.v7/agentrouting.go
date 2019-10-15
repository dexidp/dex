package gocbcore

import (
	"crypto/tls"
	"encoding/json"
	"sort"
	"time"
)

type memdInitFunc func(*syncClient, time.Time, *Agent) error

func checkSupportsFeature(srvFeatures []HelloFeature, feature HelloFeature) bool {
	for _, srvFeature := range srvFeatures {
		if srvFeature == feature {
			return true
		}
	}
	return false
}

func (agent *Agent) dialMemdClient(address string) (*memdClient, error) {
	// Copy the tls configuration since we need to provide the hostname for each
	// server that we connect to so that the certificate can be validated properly.
	var tlsConfig *tls.Config
	if agent.tlsConfig != nil {
		host, err := hostFromHostPort(address)
		if err != nil {
			logErrorf("Failed to parse address for TLS config (%s)", err)
		}

		tlsConfig = cloneTLSConfig(agent.tlsConfig)
		tlsConfig.ServerName = host
	}

	deadline := time.Now().Add(agent.serverConnectTimeout)

	memdConn, err := dialMemdConn(address, tlsConfig, deadline)
	if err != nil {
		logDebugf("Failed to connect. %v", err)
		return nil, err
	}

	client := newMemdClient(agent, memdConn)

	sclient := syncClient{
		client: client,
	}

	logDebugf("Fetching cluster client data")
	var features []HelloFeature

	// Send the TLS flag, which has unknown effects.
	features = append(features, FeatureTls)

	// Indicate that we understand XATTRs
	features = append(features, FeatureXattr)

	// Indicates that we understand select buckets.
	features = append(features, FeatureSelectBucket)

	// If the user wants to use KV Error maps, lets enable them
	if agent.useKvErrorMaps {
		features = append(features, FeatureXerror)
	}

	// If the user wants to use mutation tokens, lets enable them
	if agent.useMutationTokens {
		features = append(features, FeatureSeqNo)
	}

	// If the user wants on-the-wire compression, lets try to enable it
	if agent.useCompression {
		features = append(features, FeatureSnappy)
	}

	if agent.useDurations {
		features = append(features, FeatureDurations)
	}

	agentName := "gocbcore/" + goCbCoreVersionStr
	if agent.userString != "" {
		agentName += " " + agent.userString
	}

	clientInfo := struct {
		Agent  string `json:"a"`
		ConnId string `json:"i"`
	}{
		Agent:  agentName,
		ConnId: client.connId,
	}
	clientInfoBytes, err := json.Marshal(clientInfo)
	if err != nil {
		logDebugf("Failed to generate client info string: %s", err)
	}
	clientInfoStr := string(clientInfoBytes)

	srvFeatures, err := sclient.ExecHello(clientInfoStr, features, deadline)
	if err != nil {
		logDebugf("Failed to HELLO with server (%s)", err)
	}

	logDebugf("Client Features: %+v", features)
	logDebugf("Server Features: %+v", srvFeatures)

	client.features = srvFeatures

	if checkSupportsFeature(srvFeatures, FeatureXerror) {
		errMapData, err := sclient.ExecGetErrorMap(1, deadline)
		if err == nil {
			errMap, err := parseKvErrorMap(errMapData)
			if err == nil {
				logDebugf("Fetched error map: %+v", errMap)

				// Tell the local client to use this error map
				client.SetErrorMap(errMap)

				// Check if we need to switch the agent itself to a better
				//  error map revision.
				for {
					origMap := agent.kvErrorMap.Get()
					if origMap != nil && errMap.Revision < origMap.Revision {
						break
					}

					if agent.kvErrorMap.Update(origMap, errMap) {
						break
					}
				}
			} else {
				logDebugf("Failed to parse kv error map (%s)", err)
			}
		} else {
			logDebugf("Failed to fetch kv error map (%s)", err)
		}
	}

	logDebugf("Authenticating...")
	err = agent.initFn(&sclient, deadline, agent)
	if err != nil {
		logDebugf("Failed to authenticate. %v", err)

		closeErr := client.Close()
		if closeErr != nil {
			logWarnf("Failed to close authentication client (%s)", closeErr)
		}

		return nil, err
	}

	return client, nil
}

func (agent *Agent) slowDialMemdClient(address string) (*memdClient, error) {
	agent.serverFailuresLock.Lock()
	failureTime := agent.serverFailures[address]
	agent.serverFailuresLock.Unlock()

	if !failureTime.IsZero() {
		waitedTime := time.Since(failureTime)
		if waitedTime < agent.serverWaitTimeout {
			time.Sleep(agent.serverWaitTimeout - waitedTime)
		}
	}

	client, err := agent.dialMemdClient(address)
	if err != nil {
		agent.serverFailuresLock.Lock()
		agent.serverFailures[address] = time.Now()
		agent.serverFailuresLock.Unlock()

		return nil, err
	}

	return client, nil
}

type memdQRequestSorter []*memdQRequest

func (list memdQRequestSorter) Len() int {
	return len(list)
}

func (list memdQRequestSorter) Less(i, j int) bool {
	return list[i].dispatchTime.Before(list[j].dispatchTime)
}

func (list memdQRequestSorter) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

// Accepts a cfgBucket object representing a cluster configuration and rebuilds the server list
//  along with any routing information for the Client.  Passing no config will refresh the existing one.
//  This method MUST NEVER BLOCK due to its use from various contention points.
func (agent *Agent) applyConfig(cfg *routeConfig) {
	// Check some basic things to ensure consistency!
	if cfg.vbMap != nil && cfg.vbMap.NumVbuckets() != agent.numVbuckets {
		logErrorf("Received a configuration with a different number of vbuckets.  Ignoring.")
		return
	}

	// Only a single thing can modify the config at any time
	agent.configLock.Lock()
	defer agent.configLock.Unlock()

	newRouting := &routeData{
		revId:      cfg.revId,
		uuid:       cfg.uuid,
		capiEpList: cfg.capiEpList,
		mgmtEpList: cfg.mgmtEpList,
		n1qlEpList: cfg.n1qlEpList,
		ftsEpList:  cfg.ftsEpList,
		cbasEpList: cfg.cbasEpList,
		vbMap:      cfg.vbMap,
		ketamaMap:  cfg.ketamaMap,
		bktType:    cfg.bktType,
		source:     cfg,
	}

	kvPoolSize := agent.kvPoolSize
	maxQueueSize := agent.maxQueueSize
	newRouting.clientMux = newMemdClientMux(cfg.kvServerList, kvPoolSize, maxQueueSize, agent.slowDialMemdClient)

	oldRouting := agent.routingInfo.Get()
	if oldRouting == nil {
		return
	}

	if newRouting.revId == 0 {
		logDebugf("Unversioned configuration data, ")
	} else if newRouting.revId == oldRouting.revId {
		logDebugf("Ignoring configuration with identical revision number")
		return
	} else if newRouting.revId <= oldRouting.revId {
		logDebugf("Ignoring new configuration as it has an older revision id")
		return
	}

	// Attempt to atomically update the routing data
	if !agent.routingInfo.Update(oldRouting, newRouting) {
		logErrorf("Someone preempted the config update, skipping update")
		return
	}

	logDebugf("Switching routing data (update)...")
	logDebugf("New Routing Data:\n%s", newRouting.DebugString())

	if oldRouting.clientMux == nil {
		// This is a new agent so there is no existing muxer.  We can
		// simply start the new muxer.
		newRouting.clientMux.Start()
	} else {
		// Get the new muxer to takeover the pipelines from the older one
		newRouting.clientMux.Takeover(oldRouting.clientMux)

		// Gather all the requests from all the old pipelines and then
		//  sort and redispatch them (which will use the new pipelines)
		var requestList []*memdQRequest
		oldRouting.clientMux.Drain(func(req *memdQRequest) {
			requestList = append(requestList, req)
		})

		sort.Sort(memdQRequestSorter(requestList))

		for _, req := range requestList {
			agent.stopCmdTrace(req)
			agent.requeueDirect(req)
		}
	}
}

func (agent *Agent) updateConfig(bk *cfgBucket) {
	if bk == nil {
		// Use the existing config if none was passed.
		oldRouting := agent.routingInfo.Get()
		if oldRouting == nil {
			// If there is no previous config, we can't do anything
			return
		}

		agent.applyConfig(oldRouting.source)
	} else {
		// Normalize the cfgBucket to a routeConfig and apply it.
		routeCfg := buildRouteConfig(bk, agent.IsSecure(), agent.networkType, false)
		if !routeCfg.IsValid() {
			// We received an invalid configuration, lets shutdown.
			err := agent.Close()
			if err != nil {
				logErrorf("Invalid config caused agent close failure (%s)", err)
			}

			return
		}

		agent.applyConfig(routeCfg)
	}
}

func (agent *Agent) routeRequest(req *memdQRequest) (*memdPipeline, error) {
	routingInfo := agent.routingInfo.Get()
	if routingInfo == nil {
		return nil, ErrShutdown
	}

	var srvIdx int
	repId := req.ReplicaIdx

	// Route to specific server
	if repId < 0 {
		srvIdx = -repId - 1
	} else {
		var err error

		if routingInfo.bktType == bktTypeCouchbase {
			if req.Key != nil {
				req.Vbucket = routingInfo.vbMap.VbucketByKey(req.Key)
			}

			srvIdx, err = routingInfo.vbMap.NodeByVbucket(req.Vbucket, uint32(repId))
			if err != nil {
				return nil, err
			}
		} else if routingInfo.bktType == bktTypeMemcached {
			if repId > 0 {
				// Error. Memcached buckets don't understand replicas!
				return nil, ErrInvalidReplica
			}

			if len(req.Key) == 0 {
				// Non-broadcast keyless Memcached bucket request
				return nil, ErrCliInternalError
			}

			srvIdx, err = routingInfo.ketamaMap.NodeByKey(req.Key)
			if err != nil {
				return nil, err
			}
		}
	}

	return routingInfo.clientMux.GetPipeline(srvIdx), nil
}

func (agent *Agent) dispatchDirect(req *memdQRequest) error {
	agent.startCmdTrace(req)

	for {
		pipeline, err := agent.routeRequest(req)
		if err != nil {
			return err
		}

		err = pipeline.SendRequest(req)
		if err == errPipelineClosed {
			continue
		} else if err == errPipelineFull {
			return ErrOverload
		} else if err != nil {
			return err
		}

		break
	}

	return nil
}

func (agent *Agent) dispatchDirectToAddress(req *memdQRequest, address string) error {
	agent.startCmdTrace(req)

	// We set the ReplicaIdx to a negative number to ensure it is not redispatched
	// and we check that it was 0 to begin with to ensure it wasn't miss-used.
	if req.ReplicaIdx != 0 {
		return ErrInvalidReplica
	}
	req.ReplicaIdx = -999999999

	for {
		routingInfo := agent.routingInfo.Get()
		if routingInfo == nil {
			return ErrShutdown
		}

		var foundPipeline *memdPipeline
		for _, pipeline := range routingInfo.clientMux.pipelines {
			if pipeline.Address() == address {
				foundPipeline = pipeline
				break
			}
		}

		if foundPipeline == nil {
			return ErrInvalidServer
		}

		err := foundPipeline.SendRequest(req)
		if err == errPipelineClosed {
			continue
		} else if err == errPipelineFull {
			return ErrOverload
		} else if err != nil {
			return err
		}

		break
	}

	return nil
}

func (agent *Agent) requeueDirect(req *memdQRequest) {
	agent.startCmdTrace(req)

	handleError := func(err error) {
		logErrorf("Reschedule failed, failing request (%s)", err)

		req.tryCallback(nil, err)
	}

	for {
		pipeline, err := agent.routeRequest(req)
		if err != nil {
			handleError(err)
			return
		}

		err = pipeline.RequeueRequest(req)
		if err == errPipelineClosed {
			continue
		} else if err != nil {
			handleError(err)
			return
		}

		break
	}
}
