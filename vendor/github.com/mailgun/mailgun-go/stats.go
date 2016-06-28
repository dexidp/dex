package mailgun

import (
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
	r := newHTTPRequest(generateApiUrl(m, statsEndpoint))

	if limit != -1 {
		r.addParameter("limit", strconv.Itoa(limit))
	}
	if skip != -1 {
		r.addParameter("skip", strconv.Itoa(skip))
	}

	if startDate != nil {
		r.addParameter("start-date", startDate.Format(time.RFC3339))
	}

	for _, e := range event {
		r.addParameter("event", e)
	}
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

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
	r := newHTTPRequest(generateApiUrl(m, deleteTagEndpoint) + "/" + tag)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
