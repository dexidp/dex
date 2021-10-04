package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table refresh_token
(
    id                        text    not null  primary key,
    client_id                 text    not null,
    scopes                    blob    not null,
    nonce                     text    not null,
    claims_user_id            text    not null,
    claims_username           text    not null,
    claims_email              text    not null,
    claims_email_verified     integer not null,
    claims_groups             blob    not null,
    connector_id              text    not null,
    connector_data            blob,
    token                     text      default '' not null,
    created_at                timestamp default '0001-01-01 00:00:00 UTC' not null,
    last_used                 timestamp default '0001-01-01 00:00:00 UTC' not null,
    claims_preferred_username text      default '' not null,
    obsolete_token            text      default ''
);
*/

// RefreshToken holds the schema definition for the RefreshToken entity.
type RefreshToken struct {
	ent.Schema
}

// Fields of the RefreshToken.
func (RefreshToken) Fields() []ent.Field {
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

		field.Text("token").
			SchemaType(textSchema).
			Default(""),
		field.Text("obsolete_token").
			SchemaType(textSchema).
			Default(""),

		field.Time("created_at").
			SchemaType(timeSchema).
			Default(time.Now),
		field.Time("last_used").
			SchemaType(timeSchema).
			Default(time.Now),
	}
}

// Edges of the RefreshToken.
func (RefreshToken) Edges() []ent.Edge {
	return []ent.Edge{}
}
