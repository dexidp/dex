package gocbcore

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

// Cas represents a unique revision of a document.  This can be used
// to perform optimistic locking.
type Cas uint64

// VbUuid represents a unique identifier for a particular vbucket history.
type VbUuid uint64

// SeqNo is a sequential mutation number indicating the order and precise
// position of a write that has occurred.
type SeqNo uint64

// MutationToken represents a particular mutation within the cluster.
type MutationToken struct {
	VbId   uint16
	VbUuid VbUuid
	SeqNo  SeqNo
}

// PendingOp represents an outstanding operation within the client.
// This can be used to cancel an operation before it completes.
type PendingOp interface {
	Cancel() bool
}

type multiPendingOp struct {
	ops          []PendingOp
	completedOps uint32
}

func (mp *multiPendingOp) Cancel() bool {
	var failedCancels uint32
	for _, op := range mp.ops {
		if !op.Cancel() {
			failedCancels++
		}
	}
	return mp.CompletedOps()-failedCancels == 0
}

func (mp *multiPendingOp) CompletedOps() uint32 {
	return atomic.LoadUint32(&mp.completedOps)
}

func (mp *multiPendingOp) IncrementCompletedOps() uint32 {
	return atomic.AddUint32(&mp.completedOps, 1)
}

func (agent *Agent) waitAndRetryOperation(req *memdQRequest, waitDura time.Duration) {
	if waitDura == 0 {
		agent.requeueDirect(req)
	} else {
		time.AfterFunc(waitDura, func() {
			agent.requeueDirect(req)
		})
	}
}

func (agent *Agent) waitAndRetryNmv(req *memdQRequest) {
	agent.waitAndRetryOperation(req, agent.nmvRetryDelay)
}

func (agent *Agent) handleOpNmv(resp *memdQResponse, req *memdQRequest) {
	// Grab just the hostname from the source address
	sourceHost, err := hostFromHostPort(resp.sourceAddr)
	if err != nil {
		logErrorf("NMV response source address was invalid, skipping config update")
		agent.waitAndRetryNmv(req)
		return
	}

	// Try to parse the value as a bucket configuration
	bk, err := parseConfig(resp.Value, sourceHost)
	if err == nil {
		agent.updateConfig(bk)
	}

	// Redirect it!  This may actually come back to this server, but I won't tell
	//   if you don't ;)
	agent.waitAndRetryNmv(req)
}

func (agent *Agent) getKvErrMapData(code StatusCode) *kvErrorMapError {
	if agent.useKvErrorMaps {
		errMap := agent.kvErrorMap.Get()
		if errMap != nil {
			if errData, ok := errMap.Errors[uint16(code)]; ok {
				return &errData
			}
		}
	}
	return nil
}

func (agent *Agent) makeMemdError(code StatusCode, errMapData *kvErrorMapError, ehData []byte) error {
	if code == StatusSuccess {
		return nil
	}

	if agent.useEnhancedErrors {
		var err *KvError
		if errMapData != nil {
			err = &KvError{
				Code:        code,
				Name:        errMapData.Name,
				Description: errMapData.Description,
			}
		} else {
			err = newSimpleError(code)
		}

		if ehData != nil {
			var enhancedData struct {
				Error struct {
					Context string `json:"context"`
					Ref     string `json:"ref"`
				} `json:"error"`
			}
			if parseErr := json.Unmarshal(ehData, &enhancedData); parseErr == nil {
				err.Context = enhancedData.Error.Context
				err.Ref = enhancedData.Error.Ref
			}
		}

		return err
	}

	if ok, err := findMemdError(code); ok {
		return err
	}

	if errMapData != nil {
		return KvError{
			Code:        code,
			Name:        errMapData.Name,
			Description: errMapData.Description,
		}
	}

	return newSimpleError(code)
}

func (agent *Agent) makeBasicMemdError(code StatusCode) error {
	if !agent.useKvErrorMaps {
		return agent.makeMemdError(code, nil, nil)
	}

	errMapData := agent.getKvErrMapData(code)
	return agent.makeMemdError(code, errMapData, nil)
}

func (agent *Agent) handleOpRoutingResp(resp *memdQResponse, req *memdQRequest, err error) (bool, error) {
	if resp.Magic == resMagic {
		// Temporary backwards compatibility handling...
		if resp.Status == StatusLocked {
			switch req.Opcode {
			case cmdSet:
				resp.Status = StatusKeyExists
			case cmdReplace:
				resp.Status = StatusKeyExists
			case cmdDelete:
				resp.Status = StatusKeyExists
			default:
				resp.Status = StatusTmpFail
			}
		}

		if resp.Status == StatusNotMyVBucket {
			agent.handleOpNmv(resp, req)
			return true, nil
		} else if resp.Status == StatusSuccess {
			return false, nil
		}

		kvErrData := agent.getKvErrMapData(resp.Status)
		if kvErrData != nil {
			for _, attr := range kvErrData.Attributes {
				if attr == "auto-retry" {
					retryWait := kvErrData.Retry.CalculateRetryDelay(req.retryCount)
					maxDura := time.Duration(kvErrData.Retry.MaxDuration) * time.Millisecond
					if time.Now().Sub(req.dispatchTime)+retryWait > maxDura {
						break
					}

					req.retryCount++
					agent.waitAndRetryOperation(req, retryWait)
					return true, nil
				}
			}
		}

		if DatatypeFlag(resp.Datatype)&DatatypeFlagJson != 0 {
			err = agent.makeMemdError(resp.Status, kvErrData, resp.Value)

			if !IsErrorStatus(err, StatusSuccess) &&
				!IsErrorStatus(err, StatusKeyNotFound) &&
				!IsErrorStatus(err, StatusKeyExists) {
				logDebugf("detailed error: %+v", err)
			}
		} else {
			err = agent.makeMemdError(resp.Status, kvErrData, nil)
		}
	}

	return false, err
}

func (agent *Agent) dispatchOp(req *memdQRequest) (PendingOp, error) {
	req.owner = agent
	req.dispatchTime = time.Now()

	err := agent.dispatchDirect(req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (agent *Agent) dispatchOpToAddress(req *memdQRequest, address string) (PendingOp, error) {
	req.owner = agent
	req.dispatchTime = time.Now()

	err := agent.dispatchDirectToAddress(req, address)
	if err != nil {
		return req, nil
	}
	return req, nil
}
