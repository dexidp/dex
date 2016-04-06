package connector

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"

	"github.com/coreos/dex/user"
)

func TestNewConnectorConfigFromType(t *testing.T) {
	tests := []struct {
		typ  string
		want interface{}
	}{
		{
			typ:  LocalConnectorType,
			want: &LocalConnectorConfig{},
		},
		{
			typ:  OIDCConnectorType,
			want: &OIDCConnectorConfig{},
		},
	}

	for i, tt := range tests {
		got, err := NewConnectorConfigFromType(tt.typ)
		if err != nil {
			t.Errorf("case %d: expected nil err: %v", i, err)
			continue
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("case %d: want=%v got=%v", i, tt.want, got)
		}
	}
}

func TestNewConnectorConfigFromTypeUnrecognized(t *testing.T) {
	_, err := NewConnectorConfigFromType("foo")
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}
}

func TestNewConnectorConfigFromMap(t *testing.T) {
	user.PasswordHasher = func(plaintext string) ([]byte, error) {
		return []byte(strings.ToUpper(plaintext)), nil
	}
	defer func() {
		user.PasswordHasher = user.DefaultPasswordHasher
	}()

	tests := []struct {
		m    map[string]interface{}
		want ConnectorConfig
	}{
		{
			m: map[string]interface{}{
				"type": "local",
				"id":   "foo",
			},
			want: &LocalConnectorConfig{
				ID: "foo",
			},
		},
		{
			m: map[string]interface{}{
				"type":         "oidc",
				"id":           "bar",
				"issuerURL":    "http://example.com",
				"clientID":     "client123",
				"clientSecret": "whaaaaa",
			},
			want: &OIDCConnectorConfig{
				ID:           "bar",
				IssuerURL:    "http://example.com",
				ClientID:     "client123",
				ClientSecret: "whaaaaa",
			},
		},
	}

	for i, tt := range tests {
		got, err := newConnectorConfigFromMap(tt.m)
		if err != nil {
			t.Errorf("case %d: want nil error: %v", i, err)
			continue
		}

		if diff := pretty.Compare(tt.want, got); diff != "" {
			t.Errorf("case %d: Compare(want, got): %v", i, diff)
		}
	}
}

func TestNewConnectorConfigFromMapFail(t *testing.T) {
	tests := []map[string]interface{}{
		// no type
		map[string]interface{}{
			"id": "bar",
		},

		// type not string
		map[string]interface{}{
			"id": 123,
		},
	}

	for i, tt := range tests {
		_, err := newConnectorConfigFromMap(tt)
		if err == nil {
			t.Errorf("case %d: want non-nil error", i)
		}
	}
}
