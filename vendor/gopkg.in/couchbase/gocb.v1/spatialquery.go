package gocb

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SpatialQuery represents a pending spatial query.
type SpatialQuery struct {
	ddoc    string
	name    string
	options url.Values
}

// Stale specifies the level of consistency required for this query.
func (vq *SpatialQuery) Stale(stale StaleMode) *SpatialQuery {
	if stale == Before {
		vq.options.Set("stale", "false")
	} else if stale == None {
		vq.options.Set("stale", "ok")
	} else if stale == After {
		vq.options.Set("stale", "update_after")
	} else {
		panic("Unexpected stale option")
	}
	return vq
}

// Skip specifies how many results to skip at the beginning of the result list.
func (vq *SpatialQuery) Skip(num uint) *SpatialQuery {
	vq.options.Set("skip", strconv.FormatUint(uint64(num), 10))
	return vq
}

// Limit specifies a limit on the number of results to return.
func (vq *SpatialQuery) Limit(num uint) *SpatialQuery {
	vq.options.Set("limit", strconv.FormatUint(uint64(num), 10))
	return vq
}

// Bbox specifies the bounding region to use for the spatial query.
func (vq *SpatialQuery) Bbox(bounds []float64) *SpatialQuery {
	if len(bounds) == 4 {
		vq.options.Set("bbox", fmt.Sprintf("%f,%f,%f,%f", bounds[0], bounds[1], bounds[2], bounds[3]))
	} else {
		vq.options.Del("bbox")
	}
	return vq
}

// Development specifies whether to query the production or development design document.
func (vq *SpatialQuery) Development(val bool) *SpatialQuery {
	if val {
		if !strings.HasPrefix(vq.ddoc, "dev_") {
			vq.ddoc = "dev_" + vq.ddoc
		}
	} else {
		vq.ddoc = strings.TrimPrefix(vq.ddoc, "dev_")
	}
	return vq
}

// Custom allows specifying custom query options.
func (vq *SpatialQuery) Custom(name, value string) *SpatialQuery {
	vq.options.Set(name, value)
	return vq
}

func (vq *SpatialQuery) getInfo() (string, string, url.Values, error) {
	return vq.ddoc, vq.name, vq.options, nil
}

// NewSpatialQuery creates a new SpatialQuery object from a design document and view name.
func NewSpatialQuery(ddoc, name string) *SpatialQuery {
	return &SpatialQuery{
		ddoc:    ddoc,
		name:    name,
		options: url.Values{},
	}
}
