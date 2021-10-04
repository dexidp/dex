package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table auth_request
(
    id                        text      not null  primary key,
    client_id                 text      not null,
    response_types            blob      not null,
    scopes                    blob      not null,
    redirect_uri              text      not null,
    nonce                     text      not null,
    state                     text      not null,
    force_approval_prompt     integer   not null,
    logged_in                 integer   not null,
    claims_user_id            text      not null,
    claims_username           text      not null,
    claims_email              text      not null,
    claims_email_verified     integer   not null,
    claims_groups             blob      not null,
    connector_id              text      not null,
    connector_data            blob,
    expiry                    timestamp not null,
    claims_preferred_username text default '' not null,
    code_challenge            text default '' not null,
    code_challenge_method     text default '' not null
);
*/

// AuthRequest holds the schema definition for the AuthRequest entity.
type AuthRequest struct {
	ent.Schema
}

// Fields of the AuthRequest.
func (AuthRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.Text("client_id").
			SchemaType(textSchema),
		field.JSON("scopes", []string{}).
			Optional(),
		field.JSON("response_types", []string{}).
			Optional(),
		field.Text("redirect_uri").
			SchemaType(textSchema),
		field.Text("nonce").
			SchemaType(textSchema),
		field.Text("state").
			SchemaType(textSchema),

		field.Bool("force_approval_prompt"),
		field.Bool("logged_in"),

		field.Text("claims_user_id").
			SchemaType(textSchema),
		field.Text("claims_username").
			SchemaType(textSchema),
		field.Text("claims_email").
			SchemaType(textSchema),
		field.Bool("claims_email_verified"),
		field.JSON("claims_groups", []string{}).
			Optional(),
		field.Text("claims_preferred_username").
			SchemaType(textSchema).
			Default(""),

		field.Text("connector_id").
			SchemaType(textSchema),
		field.Bytes("connector_data").
			Nillable().
			Optional(),
		field.Time("expiry").
			SchemaType(timeSchema),

		field.Text("code_challenge").
			SchemaType(textSchema).
			Default(""),
		field.Text("code_challenge_method").
			SchemaType(textSchema).
			Default(""),
	}
}

// Edges of the AuthRequest.
func (AuthRequest) Edges() []ent.Edge {
	return []ent.Edge{}
}
