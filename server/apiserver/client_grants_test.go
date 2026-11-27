package apiserver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dexidp/dex/api/v2"
	"github.com/dexidp/dex/storage/memory"
)

func TestClientGrantTypesRoundTrip(t *testing.T) {
	logger := newLogger(t)
	client := newAPI(t, memory.New(logger), logger)
	t.Cleanup(client.Close)
	ctx := t.Context()
	grantTypes := []string{"authorization_code", "refresh_token"}
	created, err := client.CreateClient(ctx, &api.CreateClientReq{Client: &api.Client{
		Id: "restricted", Secret: "test-secret", GrantTypes: grantTypes,
	}})
	require.NoError(t, err)
	require.Equal(t, grantTypes, created.Client.GrantTypes)
	stored, err := client.GetClient(ctx, &api.GetClientReq{Id: "restricted"})
	require.NoError(t, err)
	require.Equal(t, grantTypes, stored.Client.GrantTypes)
	listed, err := client.ListClients(ctx, &api.ListClientReq{})
	require.NoError(t, err)
	require.Len(t, listed.Clients, 1)
	require.Equal(t, grantTypes, listed.Clients[0].GrantTypes)

	for _, testCase := range []struct {
		name   string
		update *api.GrantTypes
		want   []string
	}{
		{name: "omitted", want: grantTypes},
		{name: "replace", update: &api.GrantTypes{GrantTypes: []string{"client_credentials"}}, want: []string{"client_credentials"}},
		{name: "preserve", want: []string{"client_credentials"}},
		{name: "clear", update: &api.GrantTypes{}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := client.UpdateClient(t.Context(), &api.UpdateClientReq{Id: "restricted", GrantTypes: testCase.update})
			require.NoError(t, err)
			stored, err := client.GetClient(t.Context(), &api.GetClientReq{Id: "restricted"})
			require.NoError(t, err)
			require.Equal(t, testCase.want, stored.Client.GrantTypes)
		})
	}
}
