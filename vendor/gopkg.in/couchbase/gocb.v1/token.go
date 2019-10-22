package gocb

import (
	"encoding/json"
	"fmt"

	"gopkg.in/couchbase/gocbcore.v7"
)

// MutationToken holds the mutation state information from an operation.
type MutationToken struct {
	token  gocbcore.MutationToken
	bucket *Bucket
}

type bucketToken struct {
	SeqNo  uint64 `json:"seqno"`
	VbUuid string `json:"vbuuid"`
}

func (mt bucketToken) MarshalJSON() ([]byte, error) {
	info := []interface{}{mt.SeqNo, mt.VbUuid}
	return json.Marshal(info)
}

func (mt *bucketToken) UnmarshalJSON(data []byte) error {
	info := []interface{}{&mt.SeqNo, &mt.VbUuid}
	return json.Unmarshal(data, &info)
}

type bucketTokens map[string]*bucketToken
type mutationStateData map[string]*bucketTokens

// MutationState holds and aggregates MutationToken's across multiple operations.
type MutationState struct {
	data *mutationStateData
}

// NewMutationState creates a new MutationState for tracking mutation state.
func NewMutationState(tokens ...MutationToken) *MutationState {
	mt := &MutationState{}
	mt.Add(tokens...)
	return mt
}

func (mt *MutationState) addSingle(token MutationToken) {
	if token.bucket == nil {
		return
	}

	if mt.data == nil {
		data := make(mutationStateData)
		mt.data = &data
	}

	bucketName := token.bucket.name
	if (*mt.data)[bucketName] == nil {
		tokens := make(bucketTokens)
		(*mt.data)[bucketName] = &tokens
	}

	vbId := fmt.Sprintf("%d", token.token.VbId)
	stateToken := (*(*mt.data)[bucketName])[vbId]
	if stateToken == nil {
		stateToken = &bucketToken{}
		(*(*mt.data)[bucketName])[vbId] = stateToken
	}

	stateToken.SeqNo = uint64(token.token.SeqNo)
	stateToken.VbUuid = fmt.Sprintf("%d", token.token.VbUuid)
}

// Add includes an operation's mutation information in this mutation state.
func (mt *MutationState) Add(tokens ...MutationToken) {
	for _, v := range tokens {
		mt.addSingle(v)
	}
}

// MarshalJSON marshal's this mutation state to JSON.
func (mt *MutationState) MarshalJSON() ([]byte, error) {
	return json.Marshal(mt.data)
}

// UnmarshalJSON unmarshal's a mutation state from JSON.
func (mt *MutationState) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &mt.data)
}
