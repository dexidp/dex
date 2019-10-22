package gocb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	// SearchIndexSourceTypeCouchbase specifies to use a couchbase FTS index sourceType
	SearchIndexSourceTypeCouchbase = "couchbase"

	// SearchIndexSourceTypeMemcached specifies to use a memcached FTS index sourceType
	SearchIndexSourceTypeMemcached = "memcached"

	// SearchIndexIngestControlOpPause specifies to use FTS ingest control op of pause
	SearchIndexIngestControlOpPause = "pause"

	// SearchIndexIngestControlOpResume specifies to use FTS ingest control op of resume
	SearchIndexIngestControlOpResume = "resume"

	// SearchIndexQueryControlOpPause specifies to use FTS query control op of allow
	SearchIndexQueryControlOpPause = "allow"

	// SearchIndexQueryControlOpResume specifies to use FTS ingest control op of disallow
	SearchIndexQueryControlOpResume = "disallow"

	// SearchIndexPlanFreezeControlOpPause specifies to use FTS plan freeze control op of freeze
	SearchIndexPlanFreezeControlOpPause = "freeze"

	// SearchIndexPlanFreezeControlOpResume specifies to use FTS plan freeze control op of unfreeze
	SearchIndexPlanFreezeControlOpResume = "unfreeze"
)

// SearchIndexManager provides methods for performing Couchbase FTS index management.
// Experimental: This API is subject to change at any time.
type SearchIndexManager struct {
	authenticator Authenticator
	httpCli       httpClient
	cluster       searchIndexCluster
}

// SearchIndexDefinitionBuilder provides methods for building a Couchbase FTS index.
type SearchIndexDefinitionBuilder struct {
	data map[string]interface{}
}

type searchIndexDefs struct {
	IndexDefs   map[string]interface{} `json:"indexDefs,omitempty"`
	ImplVersion string                 `json:"implVersion,omitempty"`
}

type searchIndexResp struct {
	Status   string      `json:"status,omitempty"`
	IndexDef interface{} `json:"indexDef,omitempty"`
}

type searchIndexesResp struct {
	Status    string          `json:"status,omitempty"`
	IndexDefs searchIndexDefs `json:"indexDefs,omitempty"`
}

type searchIndexCluster interface {
	getFtsEp() (string, error)
}

func (sim *SearchIndexManager) doSearchIndexRequest(method, uri, contentType, cacheControl string, body io.Reader) (*http.Response, error) {
	if sim.authenticator == nil {
		return nil, errors.New("Cannot perform cluster level queries without Cluster Authenticator.")
	}

	ftsEp, err := sim.cluster.getFtsEp()
	if err != nil {
		return nil, err
	}

	creds, err := sim.authenticator.Credentials(AuthCredsRequest{
		Service:  FtsService,
		Endpoint: ftsEp,
	})
	if err != nil {
		return nil, err
	}

	reqURI := fmt.Sprintf("%s%s", ftsEp, uri)
	req, err := http.NewRequest(method, reqURI, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	if cacheControl != "" {
		req.Header.Add("cache-control", cacheControl)
	}
	if len(creds) == 1 {
		req.SetBasicAuth(creds[0].Username, creds[0].Password)
	}

	return sim.httpCli.Do(req)
}

// GetAllIndexDefinitions retrieves all of the FTS indexes for the cluster.
func (sim *SearchIndexManager) GetAllIndexDefinitions() ([]interface{}, error) {
	res, err := sim.doSearchIndexRequest("GET", "/api/index", "", "", nil)
	if err != nil {
		return nil, err
	}

	var indexesResp searchIndexesResp
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&indexesResp)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	indexDefs := indexesResp.IndexDefs.IndexDefs
	var indexes []interface{}
	for _, index := range indexDefs {
		indexes = append(indexes, index)
	}

	return indexes, nil
}

// GetIndexDefinition retrieves a specific FTS index by name.
func (sim *SearchIndexManager) GetIndexDefinition(indexName string) (interface{}, error) {
	uri := fmt.Sprintf("/api/index/%s", indexName)
	res, err := sim.doSearchIndexRequest("GET", uri, "", "", nil)
	if err != nil {
		return nil, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return nil, err
	}

	var indexResp searchIndexResp
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&indexResp)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return indexResp.IndexDef, nil
}

// CreateIndex creates a FTS index with the specific definition.
func (sim *SearchIndexManager) CreateIndex(builder SearchIndexDefinitionBuilder) error {
	err := builder.validate()
	if err != nil {
		return err
	}

	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(builder.data)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("/api/index/%s", builder.data["name"])
	res, err := sim.doSearchIndexRequest("PUT", uri, "application/json", "no-cache", b)
	if err != nil {
		return err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		if strings.Contains(err.Error(), "already exist") {
			return ErrSearchIndexAlreadyExists
		}
		return err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return nil
}

// DeleteIndex removes the FTS index with the specific name.
func (sim *SearchIndexManager) DeleteIndex(indexName string) (bool, error) {
	uri := fmt.Sprintf("/api/index/%s", indexName)
	res, err := sim.doSearchIndexRequest("DELETE", uri, "", "", nil)
	if err != nil {
		return false, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return false, err
	}

	success, err := sim.checkRespBodyStatusOK(res)
	if err != nil {
		return false, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return success, nil
}

// GetIndexedDocumentCount retrieves the document count for a FTS index.
func (sim *SearchIndexManager) GetIndexedDocumentCount(indexName string) (int, error) {
	uri := fmt.Sprintf("/api/index/%s/count", indexName)
	res, err := sim.doSearchIndexRequest("GET", uri, "", "", nil)
	if err != nil {
		return 0, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return 0, err
	}

	var count struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&count)
	if err != nil {
		return 0, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return count.Count, nil
}

// SetIndexIngestion sets the FTS index ingestion state.
func (sim *SearchIndexManager) SetIndexIngestion(indexName string, op string) (bool, error) {
	if op != SearchIndexIngestControlOpPause && op != SearchIndexIngestControlOpResume {
		return false, ErrSearchIndexInvalidIngestControlOp
	}

	uri := fmt.Sprintf("/api/index/%s/ingestControl/%s", indexName, op)
	res, err := sim.doSearchIndexRequest("POST", uri, "", "", nil)
	if err != nil {
		return false, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return false, err
	}

	success, err := sim.checkRespBodyStatusOK(res)
	if err != nil {
		return false, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return success, nil
}

// SetIndexQuerying sets the FTS index querying ability state.
func (sim *SearchIndexManager) SetIndexQuerying(indexName string, op string) (bool, error) {
	if op != SearchIndexQueryControlOpPause && op != SearchIndexQueryControlOpResume {
		return false, ErrSearchIndexInvalidQueryControlOp
	}

	uri := fmt.Sprintf("/api/index/%s/queryControl/%s", indexName, op)
	res, err := sim.doSearchIndexRequest("POST", uri, "", "", nil)
	if err != nil {
		return false, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return false, err
	}

	success, err := sim.checkRespBodyStatusOK(res)
	if err != nil {
		return false, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return success, nil
}

// SetIndexPlanFreeze sets the FTS index plan freeze state.
func (sim *SearchIndexManager) SetIndexPlanFreeze(indexName string, op string) (bool, error) {
	if op != SearchIndexPlanFreezeControlOpPause && op != SearchIndexPlanFreezeControlOpResume {
		return false, ErrSearchIndexInvalidPlanFreezeControlOp
	}

	uri := fmt.Sprintf("/api/index/%s/planFreezeControl/%s", indexName, op)
	res, err := sim.doSearchIndexRequest("POST", uri, "", "", nil)
	if err != nil {
		return false, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return false, err
	}

	success, err := sim.checkRespBodyStatusOK(res)
	if err != nil {
		return false, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return success, nil
}

// GetAllIndexStats retrieves all search index stats.
func (sim *SearchIndexManager) GetAllIndexStats() (interface{}, error) {
	res, err := sim.doSearchIndexRequest("GET", "/api/stats", "", "", nil)
	if err != nil {
		return nil, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return nil, err
	}

	var stats interface{}
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&stats)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return stats, nil
}

// GetIndexStats retrieves search index stats for the specified index.
func (sim *SearchIndexManager) GetIndexStats(indexName string) (interface{}, error) {
	uri := fmt.Sprintf("/api/stats/index/%s", indexName)
	res, err := sim.doSearchIndexRequest("GET", uri, "", "", nil)
	if err != nil {
		return nil, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return nil, err
	}

	var stats interface{}
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&stats)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return stats, nil
}

// GetAllIndexPartitionInfo retrieves all index partition information.
func (sim *SearchIndexManager) GetAllIndexPartitionInfo() (interface{}, error) {
	res, err := sim.doSearchIndexRequest("GET", "/api/pindex", "", "", nil)
	if err != nil {
		return nil, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return nil, err
	}

	var info struct {
		Status   string      `json:"status"`
		PIndexes interface{} `json:"pindexes"`
	}
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&info)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return info.PIndexes, nil
}

// GetIndexPartitionInfo retrieves a specific index partition information.
func (sim *SearchIndexManager) GetIndexPartitionInfo(pIndexName string) (interface{}, error) {
	uri := fmt.Sprintf("/api/pindex/%s", pIndexName)
	res, err := sim.doSearchIndexRequest("GET", uri, "", "", nil)
	if err != nil {
		return nil, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return nil, err
	}

	var info struct {
		Status string      `json:"status"`
		PIndex interface{} `json:"pindex"`
	}
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&info)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return info.PIndex, nil
}

// GetIndexPartitionIndexedDocumentCount retrieves a specific index partition document count.
func (sim *SearchIndexManager) GetIndexPartitionIndexedDocumentCount(pIndexName string) (int, error) {
	uri := fmt.Sprintf("/api/pindex/%s/count", pIndexName)
	res, err := sim.doSearchIndexRequest("GET", uri, "", "", nil)
	if err != nil {
		return 0, err
	}

	err = sim.checkRespBodyForError(res)
	if err != nil {
		return 0, err
	}

	var count struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	jsonDec := json.NewDecoder(res.Body)
	err = jsonDec.Decode(&count)
	if err != nil {
		return 0, err
	}

	err = res.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return count.Count, nil
}

// AddField adds a field with the specified name to the Couchbase FTS index being built
func (b *SearchIndexDefinitionBuilder) AddField(name string, value interface{}) *SearchIndexDefinitionBuilder {
	if b.data == nil {
		b.data = make(map[string]interface{})
	}

	b.data[name] = value

	return b
}

func (b *SearchIndexDefinitionBuilder) validate() error {
	if b.data["name"] == nil || b.data["name"] == "" {
		return ErrSearchIndexInvalidName
	}
	if b.data["type"] == nil || b.data["type"] == "" {
		return ErrSearchIndexMissingType
	}
	if b.data["sourceName"] == nil || b.data["sourceName"] == "" {
		return ErrSearchIndexInvalidSourceName
	}
	sourceType := b.data["sourceType"]
	if b.data["sourceType"] == nil || (sourceType != SearchIndexSourceTypeCouchbase && sourceType != SearchIndexSourceTypeMemcached) {
		return ErrSearchIndexInvalidSourceType
	}

	return nil
}

func (sim *SearchIndexManager) checkRespBodyStatusOK(resp *http.Response) (bool, error) {
	var success struct {
		Status string `json:"status"`
	}
	jsonDec := json.NewDecoder(resp.Body)
	err := jsonDec.Decode(&success)
	if err != nil {
		return false, err
	}

	return success.Status == "ok", nil
}

// checkRespBodyForError checks the response status code is 200 and if not then extracts errors. Note:
// it closes the response body on error
func (sim *SearchIndexManager) checkRespBodyForError(resp *http.Response) error {
	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		err = resp.Body.Close()
		if err != nil {
			logDebugf("Failed to close socket (%s)", err)
		}
		return clientError{string(data)}
	}

	return nil
}
