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
