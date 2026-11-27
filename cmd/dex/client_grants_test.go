package main

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
)

func TestStaticClientGrantTypes(t *testing.T) {
	var configuration Config
	require.NoError(t, yaml.Unmarshal([]byte(`staticClients:
- id: application
  grantTypes: [client_credentials]
- id: legacy
`), &configuration))
	require.Len(t, configuration.StaticClients, 2)
	require.Equal(t, []string{"client_credentials"}, configuration.StaticClients[0].GrantTypes)
	require.Nil(t, configuration.StaticClients[1].GrantTypes)
}
