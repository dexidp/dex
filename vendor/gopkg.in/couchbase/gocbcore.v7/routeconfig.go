package gocbcore

import (
	"fmt"
	"strings"
)

type routeConfig struct {
	revId        int64
	uuid         string
	bktType      bucketType
	kvServerList []string
	capiEpList   []string
	mgmtEpList   []string
	n1qlEpList   []string
	ftsEpList    []string
	cbasEpList   []string
	vbMap        *vbucketMap
	ketamaMap    *ketamaContinuum
}

func (config *routeConfig) IsValid() bool {
	if len(config.kvServerList) == 0 || len(config.mgmtEpList) == 0 {
		return false
	}
	switch config.bktType {
	case bktTypeCouchbase:
		return config.vbMap != nil && config.vbMap.IsValid()
	case bktTypeMemcached:
		return config.ketamaMap != nil && config.ketamaMap.IsValid()
	default:
		return false
	}
}

func buildRouteConfig(bk *cfgBucket, useSsl bool, networkType string, firstConnect bool) *routeConfig {
	var kvServerList []string
	var capiEpList []string
	var mgmtEpList []string
	var n1qlEpList []string
	var ftsEpList []string
	var cbasEpList []string
	var bktType bucketType

	switch bk.NodeLocator {
	case "ketama":
		bktType = bktTypeMemcached
	case "vbucket":
		bktType = bktTypeCouchbase
	default:
		logDebugf("Invalid nodeLocator %s", bk.NodeLocator)
		bktType = bktTypeInvalid
	}

	if bk.NodesExt != nil {
		lenNodes := len(bk.Nodes)
		for i, node := range bk.NodesExt {
			hostname := node.Hostname
			ports := node.Services

			if networkType != "default" {
				if altAddr, ok := node.AltAddresses[networkType]; ok {
					hostname = altAddr.Hostname
					if altAddr.Ports != nil {
						ports = *altAddr.Ports
					}
				} else {
					if !firstConnect {
						logDebugf("Invalid config network type %s", networkType)
					}
					continue
				}
			}

			// Hostname blank means to use the same one as was connected to
			if hostname == "" {
				// Note that the SourceHostname will already be IPv6 wrapped
				hostname = bk.SourceHostname
			} else {
				// We need to detect an IPv6 address here and wrap it in the appropriate
				// [] block to indicate its IPv6 for the rest of the system.
				if strings.Contains(hostname, ":") {
					hostname = "[" + hostname + "]"
				}
			}

			if !useSsl {
				if ports.Kv > 0 {
					if i >= lenNodes {
						logDebugf("KV node present in nodesext but not in nodes for %s:%d", hostname, ports.Kv)
					} else {
						kvServerList = append(kvServerList, fmt.Sprintf("%s:%d", hostname, ports.Kv))
					}
				}
				if ports.Capi > 0 {
					capiEpList = append(capiEpList, fmt.Sprintf("http://%s:%d/%s", hostname, ports.Capi, bk.Name))
				}
				if ports.Mgmt > 0 {
					mgmtEpList = append(mgmtEpList, fmt.Sprintf("http://%s:%d", hostname, ports.Mgmt))
				}
				if ports.N1ql > 0 {
					n1qlEpList = append(n1qlEpList, fmt.Sprintf("http://%s:%d", hostname, ports.N1ql))
				}
				if ports.Fts > 0 {
					ftsEpList = append(ftsEpList, fmt.Sprintf("http://%s:%d", hostname, ports.Fts))
				}
				if ports.Cbas > 0 {
					cbasEpList = append(cbasEpList, fmt.Sprintf("http://%s:%d", hostname, ports.Cbas))
				}
			} else {
				if ports.KvSsl > 0 {
					if i >= lenNodes {
						logDebugf("KV node present in nodesext but not in nodes for %s:%d", hostname, ports.KvSsl)
					} else {
						kvServerList = append(kvServerList, fmt.Sprintf("%s:%d", hostname, ports.KvSsl))
					}
				}
				if ports.CapiSsl > 0 {
					capiEpList = append(capiEpList, fmt.Sprintf("https://%s:%d/%s", hostname, ports.CapiSsl, bk.Name))
				}
				if ports.MgmtSsl > 0 {
					mgmtEpList = append(mgmtEpList, fmt.Sprintf("https://%s:%d", hostname, ports.MgmtSsl))
				}
				if ports.N1qlSsl > 0 {
					n1qlEpList = append(n1qlEpList, fmt.Sprintf("https://%s:%d", hostname, ports.N1qlSsl))
				}
				if ports.FtsSsl > 0 {
					ftsEpList = append(ftsEpList, fmt.Sprintf("https://%s:%d", hostname, ports.FtsSsl))
				}
				if ports.CbasSsl > 0 {
					cbasEpList = append(cbasEpList, fmt.Sprintf("https://%s:%d", hostname, ports.CbasSsl))
				}
			}
		}
	} else {
		if useSsl {
			logErrorf("Received config without nodesExt while SSL is enabled.  Generating invalid config.")
			return &routeConfig{}
		}

		if bktType == bktTypeCouchbase {
			kvServerList = bk.VBucketServerMap.ServerList
		}

		for _, node := range bk.Nodes {
			if node.CouchAPIBase != "" {
				// Slice off the UUID as Go's HTTP client cannot handle being passed URL-Encoded path values.
				capiEp := strings.SplitN(node.CouchAPIBase, "%2B", 2)[0]

				capiEpList = append(capiEpList, capiEp)
			}
			if node.Hostname != "" {
				mgmtEpList = append(mgmtEpList, fmt.Sprintf("http://%s", node.Hostname))
			}

			if bktType == bktTypeMemcached {
				// Get the data port. No VBucketServerMap.
				host, err := hostFromHostPort(node.Hostname)
				if err != nil {
					logErrorf("Encountered invalid memcached host/port string. Ignoring node.")
					continue
				}

				curKvHost := fmt.Sprintf("%s:%d", host, node.Ports["direct"])
				kvServerList = append(kvServerList, curKvHost)
			}
		}
	}

	rc := &routeConfig{
		revId:        bk.Rev,
		uuid:         bk.UUID,
		kvServerList: kvServerList,
		capiEpList:   capiEpList,
		mgmtEpList:   mgmtEpList,
		n1qlEpList:   n1qlEpList,
		ftsEpList:    ftsEpList,
		cbasEpList:   cbasEpList,
		bktType:      bktType,
	}

	if bktType == bktTypeCouchbase {
		vbMap := bk.VBucketServerMap.VBucketMap
		numReplicas := bk.VBucketServerMap.NumReplicas
		rc.vbMap = newVbucketMap(vbMap, numReplicas)
	} else if bktType == bktTypeMemcached {
		rc.ketamaMap = newKetamaContinuum(kvServerList)
	}

	return rc
}
