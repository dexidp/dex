package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table auth_code
(
    id                        text      not null  primary key,
    client_id                 text      not null,
    scopes                    blob      not null,
    nonce                     text      not null,
    redirect_uri              text      not null,
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

// AuthCode holds the schema definition for the AuthCode entity.
type AuthCode struct {
	ent.Schema
}

// Fields of the AuthCode.
func (AuthCode) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.Text("client_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.JSON("scopes", []string{}).
			Optional(),
		field.Text("nonce").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("redirect_uri").
			SchemaType(textSchema).
			NotEmpty(),

		field.Text("claims_user_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("claims_username").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("claims_email").
			SchemaType(textSchema).
			NotEmpty(),
		field.Bool("claims_email_verified"),
		field.JSON("claims_groups", []string{}).
			Optional(),
		field.Text("claims_preferred_username").
			SchemaType(textSchema).
			Default(""),

		field.Text("connector_id").
			SchemaType(textSchema).
			NotEmpty(),
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

// Edges of the AuthCode.
func (AuthCode) Edges() []ent.Edge {
	return []ent.Edge{}
}
