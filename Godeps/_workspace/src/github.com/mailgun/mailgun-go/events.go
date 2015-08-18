package mailgun

import (
	"fmt"
	"github.com/mbanzon/simplehttp"
	"time"
)

// Events are open-ended, loosely-defined JSON documents.
// They will always have an event and a timestamp field, however.
type Event map[string]interface{}

// noTime always equals an uninitialized Time structure.
// It's used to detect when a time parameter is provided.
var noTime time.Time

// GetEventsOptions lets the caller of GetEvents() specify how the results are to be returned.
// Begin and End time-box the results returned.
// ForceAscending and ForceDescending are used to force Mailgun to use a given traversal order of the events.
// If both ForceAscending and ForceDescending are true, an error will result.
// If none, the default will be inferred from the Begin and End parameters.
// Limit caps the number of results returned.  If left unspecified, Mailgun assumes 100.
// Compact, if true, compacts the returned JSON to minimize transmission bandwidth.
// Otherwise, the JSON is spaced appropriately for human consumption.
// Filter allows the caller to provide more specialized filters on the query.
// Consult the Mailgun documentation for more details.
type GetEventsOptions struct {
	Begin, End                               time.Time
	ForceAscending, ForceDescending, Compact bool
	Limit                                    int
	Filter                                   map[string]string
}

// EventIterator maintains the state necessary for paging though small parcels of a larger set of events.
type EventIterator struct {
	events           []Event
	nextURL, prevURL string
	mg               Mailgun
}

// NewEventIterator creates a new iterator for events.
// Use GetFirstPage to retrieve the first batch of events.
// Use Next and Previous thereafter as appropriate to iterate through sets of data.
func (mg *MailgunImpl) NewEventIterator() *EventIterator {
	return &EventIterator{mg: mg}
}

// Events returns the most recently retrieved batch of events.
// The length is guaranteed to fall between 0 and the limit set in the GetEventsOptions structure passed to GetFirstPage.
func (ei *EventIterator) Events() []Event {
	return ei.events
}

// GetFirstPage retrieves the first batch of events, according to your criteria.
// See the GetEventsOptions structure for more details on how the fields affect the data returned.
func (ei *EventIterator) GetFirstPage(opts GetEventsOptions) error {
	if opts.ForceAscending && opts.ForceDescending {
		return fmt.Errorf("collation cannot at once be both ascending and descending")
	}

	payload := simplehttp.NewUrlEncodedPayload()
	if opts.Limit != 0 {
		payload.AddValue("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Compact {
		payload.AddValue("pretty", "no")
	}
	if opts.ForceAscending {
		payload.AddValue("ascending", "yes")
	}
	if opts.ForceDescending {
		payload.AddValue("ascending", "no")
	}
	if opts.Begin != noTime {
		payload.AddValue("begin", formatMailgunTime(&opts.Begin))
	}
	if opts.End != noTime {
		payload.AddValue("end", formatMailgunTime(&opts.End))
	}
	if opts.Filter != nil {
		for k, v := range opts.Filter {
			payload.AddValue(k, v)
		}
	}

	url, err := generateParameterizedUrl(ei.mg, eventsEndpoint, payload)
	if err != nil {
		return err
	}
	return ei.fetch(url)
}

// Retrieves the chronologically previous batch of events, if any exist.
// You know you're at the end of the list when len(Events())==0.
func (ei *EventIterator) GetPrevious() error {
	return ei.fetch(ei.prevURL)
}

// Retrieves the chronologically next batch of events, if any exist.
// You know you're at the end of the list when len(Events())==0.
func (ei *EventIterator) GetNext() error {
	return ei.fetch(ei.nextURL)
}

// GetFirstPage, GetPrevious, and GetNext all have a common body of code.
// fetch completes the API fetch common to all three of these functions.
func (ei *EventIterator) fetch(url string) error {
	r := simplehttp.NewHTTPRequest(url)
	r.SetBasicAuth(basicAuthUser, ei.mg.ApiKey())
	var response map[string]interface{}
	err := getResponseFromJSON(r, &response)
	if err != nil {
		return err
	}

	items := response["items"].([]interface{})
	ei.events = make([]Event, len(items))
	for i, item := range items {
		ei.events[i] = item.(map[string]interface{})
	}

	pagings := response["paging"].(map[string]interface{})
	links := make(map[string]string, len(pagings))
	for key, page := range pagings {
		links[key] = page.(string)
	}
	ei.nextURL = links["next"]
	ei.prevURL = links["previous"]
	return err
}
