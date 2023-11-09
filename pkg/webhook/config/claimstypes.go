package config

type ClaimsMutatingHook struct {
	// Name is the name of the webhook
	Name           string   `json:"name"`
	Type           HookType `json:"type"`
	AcceptedClaims []string `json:"claims"`
	Config         *WebhookConfig
}

type ClaimsValidatingHook struct {
	// Name is the name of the webhook
	Name string `json:"name"`
	// To be modified to enum?
	Type           HookType       `json:"type"`
	AcceptedClaims []string       `json:"claims"`
	Config         *WebhookConfig `json:"config"`
}

type TokenClaimsHooks struct {
	MutatingHooks   []ClaimsMutatingHook   `json:"mutatingHooks"`
	ValidatingHooks []ClaimsValidatingHook `json:"validatingHooks"`
}
