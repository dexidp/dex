package storage

import "testing"

func TestDynamicClientRegistrationPolicy(t *testing.T) {
	client := Client{
		DynamicallyRegistered: true,
		GrantTypes:            []string{"authorization_code"},
		ResponseTypes:         []string{"code id_token"},
		AllowedScopes:         []string{"openid", "profile"},
	}

	if !client.AllowsGrantType("authorization_code") || client.AllowsGrantType("refresh_token") {
		t.Fatal("registered grant-type policy was not enforced")
	}
	if !client.AllowsResponseType([]string{"id_token", "code"}) || client.AllowsResponseType([]string{"code"}) {
		t.Fatal("registered response-type set was not enforced")
	}
	if !client.AllowsScopes([]string{"openid"}) || client.AllowsScopes([]string{"openid", "email"}) {
		t.Fatal("registered scope policy was not enforced")
	}
}

func TestLegacyClientHasNoRegistrationPolicy(t *testing.T) {
	client := Client{}
	if !client.AllowsGrantType("anything") || !client.AllowsResponseType([]string{"anything"}) || !client.AllowsScopes([]string{"anything"}) {
		t.Fatal("legacy clients must retain server-wide policy behavior")
	}
}
