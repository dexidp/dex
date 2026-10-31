package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/server/oauth2"
	"github.com/dexidp/dex/storage/memory"
)

// baseConfig is the minimum normalizeConfig accepts: storage and an issuer.
// Building it needs no server, so the config rules can be tested on their own.
func baseConfig(t *testing.T) Config {
	return Config{
		Issuer:  "https://dex.example.com",
		Storage: memory.New(newLogger(t)),
		Web:     WebConfig{Dir: "../web"},
	}
}

func TestNormalizeConfigDefaults(t *testing.T) {
	c := baseConfig(t)

	rc, err := normalizeConfig(&c)
	require.NoError(t, err)

	require.Equal(t, []string{oauth2.ResponseTypeCode}, c.SupportedResponseTypes)
	require.Equal(t, []string{"Authorization"}, c.AllowedHeaders)
	require.Equal(t, []string{oauth2.PKCEMethodS256, oauth2.PKCEMethodPlain}, c.PKCE.CodeChallengeMethodsSupported)

	require.Equal(t, "https://dex.example.com", rc.issuerURL.String())
	require.NotNil(t, rc.now)
	require.NotNil(t, rc.templates)
	require.Equal(t, map[string]bool{oauth2.ResponseTypeCode: true}, rc.responseTypes)

	// Without AllowedGrantTypes every implemented grant is advertised, sorted,
	// and the implicit grant is absent because no implicit response type is set.
	// Sorted by value, so the two grant-type URNs come last.
	require.Equal(t, []string{
		oauth2.GrantTypeAuthorizationCode,
		oauth2.GrantTypeClientCredentials,
		oauth2.GrantTypeRefreshToken,
		oauth2.GrantTypeDeviceCode,
		oauth2.GrantTypeTokenExchange,
	}, rc.grantTypes)
}

func TestNormalizeConfigGrantTypes(t *testing.T) {
	t.Run("an implicit response type adds the implicit grant", func(t *testing.T) {
		c := baseConfig(t)
		c.SupportedResponseTypes = []string{oauth2.ResponseTypeCode, oauth2.ResponseTypeToken}

		rc, err := normalizeConfig(&c)
		require.NoError(t, err)
		require.Contains(t, rc.grantTypes, oauth2.GrantTypeImplicit)
	})

	t.Run("a password connector adds the password grant", func(t *testing.T) {
		c := baseConfig(t)
		c.PasswordConnector = "local"

		rc, err := normalizeConfig(&c)
		require.NoError(t, err)
		require.Contains(t, rc.grantTypes, oauth2.GrantTypePassword)
	})

	t.Run("AllowedGrantTypes narrows the set", func(t *testing.T) {
		c := baseConfig(t)
		c.AllowedGrantTypes = []string{oauth2.GrantTypeRefreshToken, "not-a-grant"}

		rc, err := normalizeConfig(&c)
		require.NoError(t, err)
		require.Equal(t, []string{oauth2.GrantTypeRefreshToken}, rc.grantTypes)
	})
}

func TestNormalizeConfigRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		errMsg string
	}{
		{
			name:   "nil storage",
			mutate: func(c *Config) { c.Storage = nil },
			errMsg: "storage cannot be nil",
		},
		{
			name:   "unparseable issuer",
			mutate: func(c *Config) { c.Issuer = "://" },
			errMsg: "can't parse issuer URL",
		},
		{
			name:   "unknown response type",
			mutate: func(c *Config) { c.SupportedResponseTypes = []string{"nonsense"} },
			errMsg: `unsupported response_type "nonsense"`,
		},
		{
			name: "unknown PKCE method",
			mutate: func(c *Config) {
				c.PKCE.CodeChallengeMethodsSupported = []string{"S512"}
			},
			errMsg: `unsupported PKCE challenge method "S512"`,
		},
		{
			name: "dynamic registration without an initial access token",
			mutate: func(c *Config) {
				c.DynamicClientRegistration = &DynamicClientRegistrationConfig{}
			},
			errMsg: "dynamic client registration requires an initial access token",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := baseConfig(t)
			tc.mutate(&c)

			_, err := normalizeConfig(&c)
			require.ErrorContains(t, err, tc.errMsg)
		})
	}
}
