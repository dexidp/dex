package gocb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
)

type analyticsError struct {
	Code    uint32 `json:"code"`
	Message string `json:"msg"`
}

func (e *analyticsError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// AnalyticsWarning represents any warning generating during the execution of an Analytics query.
type AnalyticsWarning struct {
	Code    uint32 `json:"code"`
	Message string `json:"msg"`
}

type analyticsResponse struct {
	RequestId       string                   `json:"requestID"`
	ClientContextId string                   `json:"clientContextID"`
	Results         []json.RawMessage        `json:"results,omitempty"`
	Errors          []analyticsError         `json:"errors,omitempty"`
	Warnings        []AnalyticsWarning       `json:"warnings,omitempty"`
	Status          string                   `json:"status,omitempty"`
	Signature       interface{}              `json:"signature,omitempty"`
	Metrics         analyticsResponseMetrics `json:"metrics,omitempty"`
	Handle          string                   `json:"handle,omitempty"`
}

type analyticsResponseMetrics struct {
	ElapsedTime      string `json:"elapsedTime"`
	ExecutionTime    string `json:"executionTime"`
	ResultCount      uint   `json:"resultCount"`
	ResultSize       uint   `json:"resultSize"`
	MutationCount    uint   `json:"mutationCount,omitempty"`
	SortCount        uint   `json:"sortCount,omitempty"`
	ErrorCount       uint   `json:"errorCount,omitempty"`
	WarningCount     uint   `json:"warningCount,omitempty"`
	ProcessedObjects uint   `json:"processedObjects,omitempty"`
}

type analyticsResponseHandle struct {
	Status string `json:"status,omitempty"`
	Handle string `json:"handle,omitempty"`
}

type analyticsMultiError []analyticsError

func (e *analyticsMultiError) Error() string {
	return (*e)[0].Error()
}

func (e *analyticsMultiError) Code() uint32 {
	return (*e)[0].Code
}

// AnalyticsResultMetrics encapsulates various metrics gathered during a queries execution.
type AnalyticsResultMetrics struct {
	ElapsedTime      time.Duration
	ExecutionTime    time.Duration
	ResultCount      uint
	ResultSize       uint
	MutationCount    uint
	SortCount        uint
	ErrorCount       uint
	WarningCount     uint
	ProcessedObjects uint
}

// AnalyticsDeferredResultHandle allows access to the handle of a deferred Analytics query.
//
// Experimental: This API is subject to change at any time.
type AnalyticsDeferredResultHandle interface {
	One(valuePtr interface{}) error
	Next(valuePtr interface{}) bool
	NextBytes() []byte
	Close() error

	Status() (string, error)
}

type analyticsDeferredResultHandle struct {
	handleUri string
	status    string
	rows      *analyticsRows
	err       error
	client    *http.Client
	creds     []UserPassPair
	hasResult bool
	timeout   time.Duration
}

type analyticsRows struct {
	closed bool
	index  int
	rows   []json.RawMessage
}

// AnalyticsResults allows access to the results of a Analytics query.
type AnalyticsResults interface {
	One(valuePtr interface{}) error
	Next(valuePtr interface{}) bool
	NextBytes() []byte
	Close() error

	RequestId() string
	ClientContextId() string
	Status() string
	Warnings() []AnalyticsWarning
	Signature() interface{}
	Metrics() AnalyticsResultMetrics
	Handle() AnalyticsDeferredResultHandle
}

type analyticsResults struct {
	rows            *analyticsRows
	err             error
	requestId       string
	clientContextId string
	status          string
	warnings        []AnalyticsWarning
	signature       interface{}
	metrics         AnalyticsResultMetrics
	handle          AnalyticsDeferredResultHandle
}

func (r *analyticsResults) Next(valuePtr interface{}) bool {
	if r.err != nil {
		return false
	}

	row := r.NextBytes()
	if row == nil {
		return false
	}

	r.err = json.Unmarshal(row, valuePtr)
	if r.err != nil {
		return false
	}

	return true
}

func (r *analyticsResults) NextBytes() []byte {
	if r.err != nil {
		return nil
	}

	return r.rows.NextBytes()
}

func (r *analyticsResults) Close() error {
	r.rows.Close()
	return r.err
}

func (r *analyticsResults) One(valuePtr interface{}) error {
	if !r.Next(valuePtr) {
		err := r.Close()
		if err != nil {
			return err
		}
		return ErrNoResults
	}

	// Ignore any errors occurring after we already have our result
	err := r.Close()
	if err != nil {
		// Return no error as we got the one result already.
		return nil
	}

	return nil
}

func (r *analyticsResults) Warnings() []AnalyticsWarning {
	return r.warnings
}

func (r *analyticsResults) Status() string {
	return r.status
}

func (r *analyticsResults) Signature() interface{} {
	return r.signature
}

func (r *analyticsResults) Metrics() AnalyticsResultMetrics {
	return r.metrics
}

func (r *analyticsResults) RequestId() string {
	if !r.rows.closed {
		panic("Result must be closed before accessing meta-data")
	}

	return r.requestId
}

func (r *analyticsResults) ClientContextId() string {
	if !r.rows.closed {
		panic("Result must be closed before accessing meta-data")
	}

	return r.clientContextId
}

// Experimental: This API is subject to change at any time.
func (r *analyticsResults) Handle() AnalyticsDeferredResultHandle {
	return r.handle
}

func (r *analyticsRows) NextBytes() []byte {
	if r.index+1 >= len(r.rows) {
		r.closed = true
		return nil
	}
	r.index++

	return r.rows[r.index]
}

func (r *analyticsRows) Close() {
	r.closed = true
}

func (r *analyticsDeferredResultHandle) Next(valuePtr interface{}) bool {
	if r.err != nil {
		return false
	}

	row := r.NextBytes()
	if row == nil {
		return false
	}

	r.err = json.Unmarshal(row, valuePtr)
	if r.err != nil {
		return false
	}

	return true
}

func (r *analyticsDeferredResultHandle) NextBytes() []byte {
	if r.err != nil {
		return nil
	}

	if r.status == "success" && !r.hasResult {
		req, err := http.NewRequest("GET", r.handleUri, nil)
		if err != nil {
			r.err = err
			return nil
		}

		if len(r.creds) == 1 {
			req.SetBasicAuth(r.creds[0].Username, r.creds[0].Password)
		}

		err = r.executeHandle(r.client, req, r.timeout, &r.rows.rows)
		if err != nil {
			r.err = err
			return nil
		}
		r.hasResult = true
	} else if r.status != "success" {
		return nil
	}

	return r.rows.NextBytes()
}

func (r *analyticsDeferredResultHandle) Close() error {
	r.rows.Close()
	return r.err
}

func (r *analyticsDeferredResultHandle) One(valuePtr interface{}) error {
	if !r.Next(valuePtr) {
		err := r.Close()
		if err != nil {
			return err
		}
		return ErrNoResults
	}

	// Ignore any errors occurring after we already have our result
	err := r.Close()
	if err != nil {
		// Return no error as we got the one result already.
		return nil
	}

	return nil
}

func (r *analyticsDeferredResultHandle) Status() (string, error) {
	req, err := http.NewRequest("GET", r.handleUri, nil)
	if err != nil {
		r.err = err
		return "", err
	}

	if len(r.creds) == 1 {
		req.SetBasicAuth(r.creds[0].Username, r.creds[0].Password)
	}

	var analyticsResponse *analyticsResponseHandle
	err = r.executeHandle(r.client, req, r.timeout, &analyticsResponse)
	if err != nil {
		r.err = err
		return "", err
	}

	r.status = analyticsResponse.Status
	r.handleUri = analyticsResponse.Handle
	return r.status, nil
}

func (r *analyticsDeferredResultHandle) executeHandle(client *http.Client, req *http.Request, timeout time.Duration, valuePtr interface{}) error {
	resp, err := doHttpWithTimeout(r.client, req, r.timeout)
	if err != nil {
		return err
	}

	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(valuePtr)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	return nil
}

func (c *Cluster) executeAnalyticsQuery(tracectx opentracing.SpanContext, analyticsEp string, opts map[string]interface{}, creds []UserPassPair, timeout time.Duration, client *http.Client) (AnalyticsResults, error) {
	reqUri := fmt.Sprintf("%s/analytics/service", analyticsEp)

	priority, priorityCastok := opts["priority"].(int)
	if priorityCastok {
		delete(opts, "priority")
	}

	if len(creds) > 1 {
		opts["creds"] = creds
	}

	reqJson, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqUri, bytes.NewBuffer(reqJson))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if priorityCastok {
		req.Header.Set("Analytics-Priority", strconv.Itoa(priority))
	}

	if len(creds) == 1 {
		req.SetBasicAuth(creds[0].Username, creds[0].Password)
	}

	dtrace := c.agentConfig.Tracer.StartSpan("dispatch",
		opentracing.ChildOf(tracectx))

	resp, err := doHttpWithTimeout(client, req, timeout)
	if err != nil {
		dtrace.Finish()
		return nil, err
	}

	dtrace.Finish()

	strace := c.agentConfig.Tracer.StartSpan("streaming",
		opentracing.ChildOf(tracectx))

	analyticsResp := analyticsResponse{}
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&analyticsResp)
	if err != nil {
		strace.Finish()
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	strace.SetTag("couchbase.operation_id", analyticsResp.RequestId)
	strace.Finish()

	elapsedTime, err := time.ParseDuration(analyticsResp.Metrics.ElapsedTime)
	if err != nil {
		logDebugf("Failed to parse elapsed time duration (%s)", err)
	}

	executionTime, err := time.ParseDuration(analyticsResp.Metrics.ExecutionTime)
	if err != nil {
		logDebugf("Failed to parse execution time duration (%s)", err)
	}

	if len(analyticsResp.Errors) > 0 {
		return nil, (*analyticsMultiError)(&analyticsResp.Errors)
	}

	if resp.StatusCode != 200 {
		return nil, &viewError{
			Message: "HTTP Error",
			Reason:  fmt.Sprintf("Status code was %d.", resp.StatusCode),
		}
	}

	return &analyticsResults{
		requestId:       analyticsResp.RequestId,
		clientContextId: analyticsResp.ClientContextId,
		rows: &analyticsRows{
			rows:  analyticsResp.Results,
			index: -1,
		},
		signature: analyticsResp.Signature,
		status:    analyticsResp.Status,
		warnings:  analyticsResp.Warnings,
		metrics: AnalyticsResultMetrics{
			ElapsedTime:      elapsedTime,
			ExecutionTime:    executionTime,
			ResultCount:      analyticsResp.Metrics.ResultCount,
			ResultSize:       analyticsResp.Metrics.ResultSize,
			MutationCount:    analyticsResp.Metrics.MutationCount,
			SortCount:        analyticsResp.Metrics.SortCount,
			ErrorCount:       analyticsResp.Metrics.ErrorCount,
			WarningCount:     analyticsResp.Metrics.WarningCount,
			ProcessedObjects: analyticsResp.Metrics.ProcessedObjects,
		},
		handle: &analyticsDeferredResultHandle{
			handleUri: analyticsResp.Handle,
			rows: &analyticsRows{
				index: -1,
			},
			status:  analyticsResp.Status,
			client:  client,
			creds:   creds,
			timeout: timeout,
		},
	}, nil
}

// Performs a spatial query and returns a list of rows or an error.
func (c *Cluster) doAnalyticsQuery(tracectx opentracing.SpanContext, b *Bucket, q *AnalyticsQuery, params interface{}) (AnalyticsResults, error) {
	var err error
	var cbasEp string
	var timeout time.Duration
	var creds []UserPassPair
	var selectedB *Bucket

	if b != nil {
		if b.analyticsTimeout < c.analyticsTimeout {
			timeout = b.analyticsTimeout
		} else {
			timeout = c.analyticsTimeout
		}

		selectedB = b
	} else {
		if c.auth == nil {
			panic("Cannot perform cluster level queries without Cluster Authenticator.")
		}

		tmpB, err := c.randomBucket()
		if err != nil {
			return nil, err
		}

		timeout = c.analyticsTimeout

		selectedB = tmpB
	}

	client := selectedB.client.HttpClient()
	clientContextId := selectedB.client.ClientId()
	retryBehavior := selectedB.analyticsQueryRetryBehavior

	execOpts := make(map[string]interface{})
	for k, v := range q.options {
		execOpts[k] = v
	}
	if params != nil {
		args, isArray := params.([]interface{})
		if isArray {
			execOpts["args"] = args
		} else {
			mapArgs, isMap := params.(map[string]interface{})
			if isMap {
				for key, value := range mapArgs {
					if !strings.HasPrefix(key, "$") {
						key = "$" + key
					}
					execOpts[key] = value
				}
			} else {
				panic("Invalid params argument passed")
			}
		}
	}

	_, castok := execOpts["client_context_id"]
	if !castok {
		execOpts["client_context_id"] = clientContextId
	}

	tmostr, castok := execOpts["timeout"].(string)
	if castok {
		var err error
		timeout, err = time.ParseDuration(tmostr)
		if err != nil {
			return nil, err
		}
	} else {
		// Set the timeout string to its default variant
		execOpts["timeout"] = timeout.String()
	}

	var retries uint
	var res AnalyticsResults
	start := time.Now()
	for time.Now().Sub(start) <= time.Duration(timeout) {
		retries++
		cbasEp, err = selectedB.getCbasEp()
		if err != nil {
			return nil, err
		}

		if b != nil {
			if c.auth != nil {
				creds, err = c.auth.Credentials(AuthCredsRequest{
					Service:  CbasService,
					Endpoint: cbasEp,
					Bucket:   b.name,
				})
				if err != nil {
					return nil, err
				}
			} else {
				creds = []UserPassPair{
					{
						Username: b.name,
						Password: b.password,
					},
				}
			}
		} else {
			creds, err = c.auth.Credentials(AuthCredsRequest{
				Service:  CbasService,
				Endpoint: cbasEp,
			})
			if err != nil {
				return nil, err
			}

		}

		etrace := c.agentConfig.Tracer.StartSpan("execute",
			opentracing.ChildOf(tracectx))
		res, err = c.executeAnalyticsQuery(tracectx, cbasEp, execOpts, creds, timeout, client)
		if err == nil {
			etrace.Finish()
			return res, nil
		}

		etrace.Finish()

		analyticsErr, isAnalyticsErr := err.(*analyticsMultiError)
		if !isAnalyticsErr {
			return nil, err
		}
		if analyticsErr.Code() != 21002 && analyticsErr.Code() != 23000 && analyticsErr.Code() != 23003 && analyticsErr.Code() != 23007 {
			return nil, err
		}

		if retryBehavior == nil || !retryBehavior.CanRetry(retries) {
			break
		}

		time.Sleep(retryBehavior.NextInterval(retries))
	}

	return res, err
}

// ExecuteAnalyticsQuery performs an analytics query and returns a list of rows or an error.
func (c *Cluster) ExecuteAnalyticsQuery(q *AnalyticsQuery, params interface{}) (AnalyticsResults, error) {
	span := c.agentConfig.Tracer.StartSpan("ExecuteAnalyticsQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "cbas"})
	defer span.Finish()

	return c.doAnalyticsQuery(span.Context(), nil, q, params)
}
