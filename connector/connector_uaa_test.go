package connector

import (
	"testing"
)

func TestUAAConnectorConfigInvalidserverURLNotAValidURL(t *testing.T) {
	cc := UAAConnectorConfig{
		ID:           "uaa",
		ClientID:     "test-client",
		ClientSecret: "test-client-secret",
		ServerURL:    "https//login.apigee.com",
	}

	_, err := cc.Connector(ns, lf, templates)
	if err == nil {
		t.Fatal("Expected UAAConnector initialization to fail when UAA URL is an invalid URL")
	}
}

func TestUAAConnectorConfigInvalidserverURLNotAbsolute(t *testing.T) {
	cc := UAAConnectorConfig{
		ID:           "uaa",
		ClientID:     "test-client",
		ClientSecret: "test-client-secret",
		ServerURL:    "/uaa",
	}

	_, err := cc.Connector(ns, lf, templates)
	if err == nil {
		t.Fatal("Expected UAAConnector initialization to fail when UAA URL is not an aboslute URL")
	}
}

func TestUAAConnectorConfigValidserverURL(t *testing.T) {
	cc := UAAConnectorConfig{
		ID:           "uaa",
		ClientID:     "test-client",
		ClientSecret: "test-client-secret",
		ServerURL:    "https://login.apigee.com",
	}

	_, err := cc.Connector(ns, lf, templates)
	if err != nil {
		t.Fatal(err)
	}
}
