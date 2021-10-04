package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table connector
(
    id               text not null  primary key,
    type             text not null,
    name             text not null,
    resource_version text not null,
    config           blob
);
*/

// Connector holds the schema definition for the Client entity.
type Connector struct {
	ent.Schema
}

// Fields of the Connector.
func (Connector) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			SchemaType(textSchema).
			MaxLen(100).
			NotEmpty().
			Unique(),
		field.Text("type").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("name").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("resource_version").
			SchemaType(textSchema),
		field.Bytes("config"),
	}
}

// Edges of the Connector.
func (Connector) Edges() []ent.Edge {
	return []ent.Edge{}
}
