package mailgun

// Campaigns have been deprecated since development work on this SDK commenced.
// Please refer to http://documentation.mailgun.com/api_reference .
type Campaign struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	CreatedAt         string `json:"created_at"`
	DeliveredCount    int    `json:"delivered_count"`
	ClickedCount      int    `json:"clicked_count"`
	OpenedCount       int    `json:"opened_count"`
	SubmittedCount    int    `json:"submitted_count"`
	UnsubscribedCount int    `json:"unsubscribed_count"`
	BouncedCount      int    `json:"bounced_count"`
	ComplainedCount   int    `json:"complained_count"`
	DroppedCount      int    `json:"dropped_count"`
}

type campaignsEnvelope struct {
	TotalCount int        `json:"total_count"`
	Items      []Campaign `json:"items"`
}

// Campaigns have been deprecated since development work on this SDK commenced.
// Please refer to http://documentation.mailgun.com/api_reference .
func (m *MailgunImpl) GetCampaigns() (int, []Campaign, error) {
	r := newHTTPRequest(generateApiUrl(m, campaignsEndpoint))
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	var envelope campaignsEnvelope
	err := getResponseFromJSON(r, &envelope)
	if err != nil {
		return -1, nil, err
	}
	return envelope.TotalCount, envelope.Items, nil
}

// Campaigns have been deprecated since development work on this SDK commenced.
// Please refer to http://documentation.mailgun.com/api_reference .
func (m *MailgunImpl) CreateCampaign(name, id string) error {
	r := newHTTPRequest(generateApiUrl(m, campaignsEndpoint))
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	payload := newUrlEncodedPayload()
	payload.addValue("name", name)
	if id != "" {
		payload.addValue("id", id)
	}
	_, err := makePostRequest(r, payload)
	return err
}

// Campaigns have been deprecated since development work on this SDK commenced.
// Please refer to http://documentation.mailgun.com/api_reference .
func (m *MailgunImpl) UpdateCampaign(oldId, name, newId string) error {
	r := newHTTPRequest(generateApiUrl(m, campaignsEndpoint) + "/" + oldId)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())

	payload := newUrlEncodedPayload()
	payload.addValue("name", name)
	if newId != "" {
		payload.addValue("id", newId)
	}
	_, err := makePostRequest(r, payload)
	return err
}

// Campaigns have been deprecated since development work on this SDK commenced.
// Please refer to http://documentation.mailgun.com/api_reference .
func (m *MailgunImpl) DeleteCampaign(id string) error {
	r := newHTTPRequest(generateApiUrl(m, campaignsEndpoint) + "/" + id)
	r.setClient(m.Client())
	r.setBasicAuth(basicAuthUser, m.ApiKey())
	_, err := makeDeleteRequest(r)
	return err
}
