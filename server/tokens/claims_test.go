package tokens

import (
	"testing"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
)

func TestGetClientID(t *testing.T) {
	cid, err := GetClientID(Audience{}, "")
	require.Equal(t, "", cid)
	require.Equal(t, "no audience is set, could not find ClientID", err.Error())

	cid, err = GetClientID(Audience{"a"}, "")
	require.Equal(t, "a", cid)
	require.NoError(t, err)

	cid, err = GetClientID(Audience{"a", "b"}, "azp")
	require.Equal(t, "azp", cid)
	require.NoError(t, err)
}

func TestGetAudience(t *testing.T) {
	require.Equal(t, Audience{"client-id"}, GetAudience("client-id", []string{}))
	require.Equal(t, Audience{"client-id"}, GetAudience("client-id", []string{"ascope"}))
	require.Equal(t, Audience{"aa", "bb", "client-id"},
		GetAudience("client-id", []string{"ascope", "audience:server:client_id:aa", "audience:server:client_id:bb"}))
}

func TestGenSubject(t *testing.T) {
	sub, err := GenSubject("foo", "bar")
	require.NoError(t, err)
	require.Equal(t, "CgNmb28SA2Jhcg", sub)
}

func TestGenAndParseSubjectPlainFormat(t *testing.T) {
	sub, err := GenSubjectWithFormat("user:one@example.com", "oidc|github", SubjectFormatPlain)
	require.NoError(t, err)
	require.Equal(t, "user%3Aone%40example.com|oidc%7Cgithub", sub)

	userID, connectorID, err := ParseSubject(sub)
	require.NoError(t, err)
	require.Equal(t, "user:one@example.com", userID)
	require.Equal(t, "oidc|github", connectorID)
}

func TestGenAndParseSubjectPlainFormatWithReversedOrder(t *testing.T) {
	sub, err := GenSubjectWithFormatAndOrder("user:one@example.com", "oidc|github", SubjectFormatPlain, SubjectOrderConnectorUser)
	require.NoError(t, err)
	require.Equal(t, "oidc%7Cgithub|user%3Aone%40example.com", sub)

	userID, connectorID, err := ParseSubjectWithOrder(sub, SubjectOrderConnectorUser)
	require.NoError(t, err)
	require.Equal(t, "user:one@example.com", userID)
	require.Equal(t, "oidc|github", connectorID)
}

func TestParseSubjectLegacyFormat(t *testing.T) {
	userID, connectorID, err := ParseSubject("CgNmb28SA2Jhcg")
	require.NoError(t, err)
	require.Equal(t, "foo", userID)
	require.Equal(t, "bar", connectorID)
}

func TestParseSubjectFormat(t *testing.T) {
	format, err := ParseSubjectFormat("")
	require.NoError(t, err)
	require.Equal(t, SubjectFormatBase64, format)

	format, err = ParseSubjectFormat("plain")
	require.NoError(t, err)
	require.Equal(t, SubjectFormatPlain, format)

	format, err = ParseSubjectFormat("legacy")
	require.NoError(t, err)
	require.Equal(t, SubjectFormatBase64, format)

	format, err = ParseSubjectFormat("pair")
	require.NoError(t, err)
	require.Equal(t, SubjectFormatPlain, format)

	_, err = ParseSubjectFormat("something-else")
	require.ErrorContains(t, err, `unsupported subject format "something-else"`)
}

func TestParseSubjectOrder(t *testing.T) {
	order, err := ParseSubjectOrder("")
	require.NoError(t, err)
	require.Equal(t, SubjectOrderUserConnector, order)

	order, err = ParseSubjectOrder("user-connector")
	require.NoError(t, err)
	require.Equal(t, SubjectOrderUserConnector, order)

	order, err = ParseSubjectOrder("connector-user")
	require.NoError(t, err)
	require.Equal(t, SubjectOrderConnectorUser, order)

	order, err = ParseSubjectOrder("reversed")
	require.NoError(t, err)
	require.Equal(t, SubjectOrderConnectorUser, order)

	_, err = ParseSubjectOrder("something-else")
	require.ErrorContains(t, err, `unsupported subject order "something-else"`)
}

func TestAccessTokenHash(t *testing.T) {
	// at_hash value and access_token returned by Google.
	const (
		googleAccessTokenHash = "piwt8oCH-K2D9pXlaS1Y-w"
		googleAccessToken     = "ya29.CjHSA1l5WUn8xZ6HanHFzzdHdbXm-14rxnC7JHch9eFIsZkQEGoWzaYG4o7k5f6BnPLj"
	)

	atHash, err := AccessTokenHash(jose.RS256, googleAccessToken)
	require.NoError(t, err)
	require.Equal(t, googleAccessTokenHash, atHash)
}
