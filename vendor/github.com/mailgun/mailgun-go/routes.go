package mailgun

import (
	"strconv"
)

// A Route structure contains information on a configured or to-be-configured route.
// The Priority field indicates how soon the route works relative to other configured routes.
// Routes of equal priority are consulted in chronological order.
// The Description field provides a human-readable description for the route.
// Mailgun ignores this field except to provide the description when viewing the Mailgun web control panel.
// The Expression field lets you specify a pattern to match incoming messages against.
// The Actions field contains strings specifying what to do
// with any message which matches the provided expression.
// The CreatedAt field provides a time-stamp for when the route came into existence.
// Finally, the ID field provides a unique identifier for this route.
//
// When creating a new route, the SDK only uses a subset of the fields of this structure.
// In particular, CreatedAt and ID are meaningless in this context, and will be ignored.
// Only Priority, Description, Expression, and Actions need be provided.
type Route struct {
	Priority    int      `json:"priority,omitempty"`
	Description string   `json:"description,omitempty"`
	Expression  string   `json:"expression,omitempty"`
	Actions     []string `json:"actions,omitempty"`

	CreatedAt string `json:"created_at,omitempty"`
	ID        string `json:"id,omitempty"`
}

// GetRoutes returns the complete set of routes configured for your domain.
// You use routes to configure how to handle returned messages, or
// messages sent to a specfic address on your domain.
// See the Mailgun documentation for more information.
func (mg *MailgunImpl) GetRoutes(limit, skip int) (int, []Route, error) {
	r := newHTTPRequest(generatePublicApiUrl(routesEndpoint))
	if limit != DefaultLimit {
		r.addParameter("limit", strconv.Itoa(limit))
	}
	if skip != DefaultSkip {
		r.addParameter("skip", strconv.Itoa(skip))
	}
	r.setClient(mg.Client())
	r.setBasicAuth(basicAuthUser, mg.ApiKey())

	var envelope struct {
		TotalCount int     `json:"total_count"`
		Items      []Route `json:"items"`
	}
	err := getResponseFromJSON(r, &envelope)
	if err != nil {
		return -1, nil, err
	}
	return envelope.TotalCount, envelope.Items, nil
}

// CreateRoute installs a new route for your domain.
// The route structure you provide serves as a template, and
// only a subset of the fields influence the operation.
// See the Route structure definition for more details.
func (mg *MailgunImpl) CreateRoute(prototype Route) (Route, error) {
	r := newHTTPRequest(generatePublicApiUrl(routesEndpoint))
	r.setClient(mg.Client())
	r.setBasicAuth(basicAuthUser, mg.ApiKey())
	p := newUrlEncodedPayload()
	p.addValue("priority", strconv.Itoa(prototype.Priority))
	p.addValue("description", prototype.Description)
	p.addValue("expression", prototype.Expression)
	for _, action := range prototype.Actions {
		p.addValue("action", action)
	}
	var envelope struct {
		Message string `json:"message"`
		*Route  `json:"route"`
	}
	err := postResponseFromJSON(r, p, &envelope)
	return *envelope.Route, err
}

// DeleteRoute removes the specified route from your domain's configuration.
// To avoid ambiguity, Mailgun identifies the route by unique ID.
// See the Route structure definition and the Mailgun API documentation for more details.
func (mg *MailgunImpl) DeleteRoute(id string) error {
	r := newHTTPRequest(generatePublicApiUrl(routesEndpoint) + "/" + id)
	r.setClient(mg.Client())
	r.setBasicAuth(basicAuthUser, mg.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}

// GetRouteByID retrieves the complete route definition associated with the unique route ID.
func (mg *MailgunImpl) GetRouteByID(id string) (Route, error) {
	r := newHTTPRequest(generatePublicApiUrl(routesEndpoint) + "/" + id)
	r.setClient(mg.Client())
	r.setBasicAuth(basicAuthUser, mg.ApiKey())
	var envelope struct {
		Message string `json:"message"`
		*Route  `json:"route"`
	}
	err := getResponseFromJSON(r, &envelope)
	return *envelope.Route, err
}

// UpdateRoute provides an "in-place" update of the specified route.
// Only those route fields which are non-zero or non-empty are updated.
// All other fields remain as-is.
func (mg *MailgunImpl) UpdateRoute(id string, route Route) (Route, error) {
	r := newHTTPRequest(generatePublicApiUrl(routesEndpoint) + "/" + id)
	r.setClient(mg.Client())
	r.setBasicAuth(basicAuthUser, mg.ApiKey())
	p := newUrlEncodedPayload()
	if route.Priority != 0 {
		p.addValue("priority", strconv.Itoa(route.Priority))
	}
	if route.Description != "" {
		p.addValue("description", route.Description)
	}
	if route.Expression != "" {
		p.addValue("expression", route.Expression)
	}
	if route.Actions != nil {
		for _, action := range route.Actions {
			p.addValue("action", action)
		}
	}
	// For some reason, this API function just returns a bare Route on success.
	// Unsure why this is the case; it seems like it ought to be a bug.
	var envelope Route
	err := putResponseFromJSON(r, p, &envelope)
	return envelope, err
}
