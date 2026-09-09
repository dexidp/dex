package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/dexidp/dex/storage"
)

/* Original SQL table:
create table client
(
    id            text    not null  primary key,
    secret        text    not null,
    redirect_uris blob    not null,
    trusted_peers blob    not null,
    public        integer not null,
    name          text    not null,
    logo_url      text    not null
);
*/

// OAuth2Client holds the schema definition for the Client entity.
type OAuth2Client struct {
	ent.Schema
}

// Fields of the OAuth2Client.
func (OAuth2Client) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			SchemaType(textSchema).
			MaxLen(100).
			NotEmpty().
			Unique(),
		field.Text("secret").
			SchemaType(textSchema),
		field.JSON("redirect_uris", []string{}).
			Optional(),
		field.JSON("trusted_peers", []string{}).
			Optional(),
		field.Bool("public"),
		field.Text("name").
			SchemaType(textSchema),
		field.Text("logo_url").
			SchemaType(textSchema),
		field.Bool("dynamically_registered").Default(false),
		field.JSON("grant_types", []string{}).Optional(),
		field.JSON("response_types", []string{}).Optional(),
		field.JSON("allowed_scopes", []string{}).Optional(),
		field.Text("token_endpoint_auth_method").SchemaType(textSchema).Default("").Optional(),
		field.Int64("registration_time").Default(0),
		field.Text("registration_token_id").SchemaType(textSchema).Default("").Optional(),
		field.Int64("registration_expires_at").Default(0),
		field.JSON("allowed_connectors", []string{}).
			Optional(),
		field.JSON("mfa_chain", []string{}).
			Optional(),
		field.JSON("post_logout_redirect_uris", []string{}).
			Optional(),
		field.JSON("sso_shared_with", []string{}).
			Optional(),
		field.Text("backchannel_logout_uri").
			SchemaType(textSchema).
			Default("").
			Optional(),
		field.Text("refresh_token_lifetime").
			SchemaType(textSchema).
			Default("").
			Optional(),
		field.JSON("client_credentials_claims", &storage.ClientCredentialsClaims{}).
			Optional(),
	}
}

// Edges of the OAuth2Client.
func (OAuth2Client) Edges() []ent.Edge {
	return []ent.Edge{}
}
