package gocb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/opentracing/opentracing-go"
)

type viewError struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

type viewResponse struct {
	TotalRows int               `json:"total_rows,omitempty"`
	Rows      []json.RawMessage `json:"rows,omitempty"`
	Error     string            `json:"error,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Errors    []viewError       `json:"errors,omitempty"`
}

func (e *viewError) Error() string {
	return e.Message + " - " + e.Reason
}

// ViewResults implements an iterator interface which can be used to iterate over the rows of the query results.
type ViewResults interface {
	One(valuePtr interface{}) error
	Next(valuePtr interface{}) bool
	NextBytes() []byte
	Close() error
}

// ViewResultMetrics allows access to the TotalRows value from the view response.  This is
// implemented as an additional interface to maintain ABI compatibility for the 1.x series.
type ViewResultMetrics interface {
	TotalRows() int
}

type viewResults struct {
	index     int
	rows      []json.RawMessage
	totalRows int
	err       error
	endErr    error
}

func (r *viewResults) Next(valuePtr interface{}) bool {
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

func (r *viewResults) NextBytes() []byte {
	if r.err != nil {
		return nil
	}

	if r.index+1 >= len(r.rows) {
		return nil
	}
	r.index++

	return r.rows[r.index]
}

func (r *viewResults) Close() error {
	if r.err != nil {
		return r.err
	}

	if r.endErr != nil {
		return r.endErr
	}

	return nil
}

func (r *viewResults) One(valuePtr interface{}) error {
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

func (r *viewResults) TotalRows() int {
	return r.totalRows
}

func (b *Bucket) executeViewQuery(tracectx opentracing.SpanContext, viewType, ddoc, viewName string, options url.Values) (ViewResults, error) {
	capiEp, err := b.getViewEp()
	if err != nil {
		return nil, err
	}

	reqUri := fmt.Sprintf("%s/_design/%s/%s/%s?%s", capiEp, ddoc, viewType, viewName, options.Encode())

	req, err := http.NewRequest("GET", reqUri, nil)
	if err != nil {
		return nil, err
	}

	if b.cluster.auth != nil {
		userPass, err := getSingleCredential(b.cluster.auth, AuthCredsRequest{
			Service:  CapiService,
			Endpoint: capiEp,
			Bucket:   b.name,
		})
		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(userPass.Username, userPass.Password)
	} else {
		req.SetBasicAuth(b.name, b.password)
	}

	dtrace := b.tracer.StartSpan("dispatch",
		opentracing.ChildOf(tracectx))

	resp, err := doHttpWithTimeout(b.client.HttpClient(), req, b.viewTimeout)
	if err != nil {
		dtrace.Finish()
		return nil, err
	}

	dtrace.Finish()

	strace := b.tracer.StartSpan("streaming",
		opentracing.ChildOf(tracectx))

	viewResp := viewResponse{}
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&viewResp)
	if err != nil {
		strace.Finish()
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		logDebugf("Failed to close socket (%s)", err)
	}

	strace.Finish()

	if resp.StatusCode != 200 {
		if viewResp.Error != "" {
			return nil, &viewError{
				Message: viewResp.Error,
				Reason:  viewResp.Reason,
			}
		}

		return nil, &viewError{
			Message: "HTTP Error",
			Reason:  fmt.Sprintf("Status code was %d.", resp.StatusCode),
		}
	}

	var endErrs MultiError
	for _, endErr := range viewResp.Errors {
		endErrs.add(&viewError{
			Message: endErr.Message,
			Reason:  endErr.Reason,
		})
	}

	return &viewResults{
		index:     -1,
		rows:      viewResp.Rows,
		totalRows: viewResp.TotalRows,
		endErr:    endErrs.get(),
	}, nil
}

// ExecuteViewQuery performs a view query and returns a list of rows or an error.
func (b *Bucket) ExecuteViewQuery(q *ViewQuery) (ViewResults, error) {
	span := b.tracer.StartSpan("ExecuteViewQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "views"})
	defer span.Finish()

	ddoc, name, opts, err := q.getInfo()
	if err != nil {
		return nil, err
	}

	return b.executeViewQuery(span.Context(), "_view", ddoc, name, opts)
}

// ExecuteSpatialQuery performs a spatial query and returns a list of rows or an error.
func (b *Bucket) ExecuteSpatialQuery(q *SpatialQuery) (ViewResults, error) {
	span := b.tracer.StartSpan("ExecuteSpatialQuery",
		opentracing.Tag{Key: "couchbase.service", Value: "views"})
	defer span.Finish()

	ddoc, name, opts, err := q.getInfo()
	if err != nil {
		return nil, err
	}

	return b.executeViewQuery(span.Context(), "_spatial", ddoc, name, opts)
}
