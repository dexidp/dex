package mailgun

import (
	"github.com/mbanzon/simplehttp"
	"strconv"
	"time"
)

type Stat struct {
	Event      string         `json:"event"`
	TotalCount int            `json:"total_count"`
	CreatedAt  string         `json:"created_at"`
	Id         string         `json:"id"`
	Tags       map[string]int `json:"tags"`
}

type statsEnvelope struct {
	TotalCount int    `json:"total_count"`
	Items      []Stat `json:"items"`
}

// GetStats returns a basic set of statistics for different events.
// Events start at the given start date, if one is provided.
// If not, this function will consider all stated events dating to the creation of the sending domain.
func (m *MailgunImpl) GetStats(limit int, skip int, startDate *time.Time, event ...string) (int, []Stat, error) {
	r := simplehttp.NewHTTPRequest(generateApiUrl(m, statsEndpoint))

	if limit != -1 {
		r.AddParameter("limit", strconv.Itoa(limit))
	}
	if skip != -1 {
		r.AddParameter("skip", strconv.Itoa(skip))
	}

	if startDate != nil {
		r.AddParameter("start-date", startDate.Format(time.RFC3339))
	}

	for _, e := range event {
		r.AddParameter("event", e)
	}
	r.SetBasicAuth(basicAuthUser, m.ApiKey())

	var res statsEnvelope
	err := getResponseFromJSON(r, &res)
	if err != nil {
		return -1, nil, err
	} else {
		return res.TotalCount, res.Items, nil
	}
}

// DeleteTag removes all counters for a particular tag, including the tag itself.
func (m *MailgunImpl) DeleteTag(tag string) error {
	r := simplehttp.NewHTTPRequest(generateApiUrl(m, deleteTagEndpoint) + "/" + tag)
	r.SetBasicAuth(basicAuthUser, m.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
