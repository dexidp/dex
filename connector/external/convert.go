package external

import (
	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/connector/external/sdk"
)

func toSDKScopes(s connector.Scopes) *sdk.Scopes {
	return &sdk.Scopes{
		OfflineAccess: s.OfflineAccess,
		Groups:        s.Groups,
	}
}

func toSDKIdentity(id connector.Identity) *sdk.Identity {
	return &sdk.Identity{
		UserId:            id.UserID,
		Username:          id.Username,
		PreferredUsername: id.PreferredUsername,
		Email:             id.Email,
		EmailVerified:     id.EmailVerified,
		Groups:            id.Groups,
		ConnectorData:     id.ConnectorData,
	}
}

func toConnectorIdentity(id *sdk.Identity) connector.Identity {
	return connector.Identity{
		UserID:            id.UserId,
		Username:          id.Username,
		PreferredUsername: id.PreferredUsername,
		Email:             id.Email,
		EmailVerified:     id.EmailVerified,
		Groups:            id.Groups,
		ConnectorData:     id.ConnectorData,
	}
}
