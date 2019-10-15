package gocbcore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func randomCbUid() []byte {
	out := make([]byte, 8)
	_, err := rand.Read(out)
	if err != nil {
		logWarnf("Crypto read failed: %s", err)
	}
	return out
}

func formatCbUid(data []byte) string {
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x",
		data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7])
}

func getCommandName(command commandCode) string {
	switch command {
	case cmdGet:
		return "CMD_GET"
	case cmdSet:
		return "CMD_SET"
	case cmdAdd:
		return "CMD_ADD"
	case cmdReplace:
		return "CMD_REPLACE"
	case cmdDelete:
		return "CMD_DELETE"
	case cmdIncrement:
		return "CMD_INCREMENT"
	case cmdDecrement:
		return "CMD_DECREMENT"
	case cmdNoop:
		return "CMD_NOOP"
	case cmdAppend:
		return "CMD_APPEND"
	case cmdPrepend:
		return "CMD_PREPEND"
	case cmdStat:
		return "CMD_STAT"
	case cmdTouch:
		return "CMD_TOUCH"
	case cmdGAT:
		return "CMD_GAT"
	case cmdHello:
		return "CMD_HELLO"
	case cmdSASLListMechs:
		return "CMD_SASLLISTMECHS"
	case cmdSASLAuth:
		return "CMD_SASLAUTH"
	case cmdSASLStep:
		return "CMD_SASLSTEP"
	case cmdGetAllVBSeqnos:
		return "CMD_GETALLVBSEQNOS"
	case cmdDcpOpenConnection:
		return "CMD_DCPOPENCONNECTION"
	case cmdDcpAddStream:
		return "CMD_DCPADDSTREAM"
	case cmdDcpCloseStream:
		return "CMD_DCPCLOSESTREAM"
	case cmdDcpStreamReq:
		return "CMD_DCPSTREAMREQ"
	case cmdDcpGetFailoverLog:
		return "CMD_DCPGETFAILOVERLOG"
	case cmdDcpStreamEnd:
		return "CMD_DCPSTREAMEND"
	case cmdDcpSnapshotMarker:
		return "CMD_DCPSNAPSHOTMARKER"
	case cmdDcpMutation:
		return "CMD_DCPMUTATION"
	case cmdDcpDeletion:
		return "CMD_DCPDELETION"
	case cmdDcpExpiration:
		return "CMD_DCPEXPIRATION"
	case cmdDcpFlush:
		return "CMD_DCPFLUSH"
	case cmdDcpSetVbucketState:
		return "CMD_DCPSETVBUCKETSTATE"
	case cmdDcpNoop:
		return "CMD_DCPNOOP"
	case cmdDcpBufferAck:
		return "CMD_DCPBUFFERACK"
	case cmdDcpControl:
		return "CMD_DCPCONTROL"
	case cmdGetReplica:
		return "CMD_GETREPLICA"
	case cmdSelectBucket:
		return "CMD_SELECTBUCKET"
	case cmdObserveSeqNo:
		return "CMD_OBSERVESEQNO"
	case cmdObserve:
		return "CMD_OBSERVE"
	case cmdGetLocked:
		return "CMD_GETLOCKED"
	case cmdUnlockKey:
		return "CMD_UNLOCKKEY"
	case cmdGetMeta:
		return "CMD_GETMETA"
	case cmdSetMeta:
		return "CMD_SETMETA"
	case cmdDelMeta:
		return "CMD_DELMETA"
	case cmdGetClusterConfig:
		return "CMD_GETCLUSTERCONFIG"
	case cmdGetRandom:
		return "CMD_GETRANDOM"
	case cmdSubDocGet:
		return "CMD_SUBDOCGET"
	case cmdSubDocExists:
		return "CMD_SUBDOCEXISTS"
	case cmdSubDocDictAdd:
		return "CMD_SUBDOCDICTADD"
	case cmdSubDocDictSet:
		return "CMD_SUBDOCDICTSET"
	case cmdSubDocDelete:
		return "CMD_SUBDOCDELETE"
	case cmdSubDocReplace:
		return "CMD_SUBDOCREPLACE"
	case cmdSubDocArrayPushLast:
		return "CMD_SUBDOCARRAYPUSHLAST"
	case cmdSubDocArrayPushFirst:
		return "CMD_SUBDOCARRAYPUSHFIRST"
	case cmdSubDocArrayInsert:
		return "CMD_SUBDOCARRAYINSERT"
	case cmdSubDocArrayAddUnique:
		return "CMD_SUBDOCARRAYADDUNIQUE"
	case cmdSubDocCounter:
		return "CMD_SUBDOCCOUNTER"
	case cmdSubDocMultiLookup:
		return "CMD_SUBDOCMULTILOOKUP"
	case cmdSubDocMultiMutation:
		return "CMD_SUBDOCMULTIMUTATION"
	case cmdSubDocGetCount:
		return "CMD_SUBDOCGETCOUNT"
	case cmdGetErrorMap:
		return "CMD_GETERRORMAP"
	default:
		return "CMD_x" + hex.EncodeToString([]byte{byte(command)})
	}
}
