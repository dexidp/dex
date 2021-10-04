package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
			SchemaType(textSchema).
			NotEmpty(),
		field.JSON("redirect_uris", []string{}).
			Optional(),
		field.JSON("trusted_peers", []string{}).
			Optional(),
		field.Bool("public"),
		field.Text("name").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("logo_url").
			SchemaType(textSchema).
			NotEmpty(),
	}
}

// Edges of the OAuth2Client.
func (OAuth2Client) Edges() []ent.Edge {
	return []ent.Edge{}
}
