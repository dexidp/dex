package config

type WebhookConfig struct {
	URL                  string                `json:"url"`
	InsecureSkipVerify   bool                  `json:"insecureSkipVerify"`
	TLSRootCAFile        string                `json:"tlsRootCAFile"`
	ClientAuthentication *ClientAuthentication `json:"clientAuthentication"`
}

type ClientAuthentication struct {
	ClientCertificateFile string `json:"clientCertificateFile"`
	ClientKeyFile         string `json:"clientKeyFile"`
	ClientCAFile          string `json:"clientCAFile"`
}
