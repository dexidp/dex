package config

// HookRequestScope is the context of the request
type HookRequestScope struct {
	// Headers is the headers of the request
	Headers []string `json:"headers"`
	// Params is the params of the request
	Params []string `json:"params"`
}

type ConnectorFilterHook struct {
	// Name is the name of the webhook
	Name string `json:"name"`
	// To be modified to enum?
	Type HookType `json:"type"`
	// RequestScope is the context of the request
	RequestScope *HookRequestScope `json:"requestContext"`
	// Config is the configuration of the webhook
	Config *WebhookConfig `json:"config"`
}

type ConnectorFilterHooks struct {
	FilterHooks []*ConnectorFilterHook `json:"filterHooks"`
}
