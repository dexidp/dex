package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// AuthSession holds the schema definition for the AuthSession entity.
type AuthSession struct {
	ent.Schema
}

// Fields of the AuthSession.
func (AuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.Text("user_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("connector_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("nonce").
			SchemaType(textSchema).
			NotEmpty(),
		field.Bytes("client_states"),
		field.Time("created_at").
			SchemaType(timeSchema),
		field.Time("last_activity").
			SchemaType(timeSchema),
		field.Text("ip_address").
			SchemaType(textSchema).
			Default(""),
		field.Text("user_agent").
			SchemaType(textSchema).
			Default(""),
		field.Time("absolute_expiry").
			SchemaType(timeSchema),
		field.Time("idle_expiry").
			SchemaType(timeSchema),
	}
}

// Edges of the AuthSession.
func (AuthSession) Edges() []ent.Edge {
	return []ent.Edge{}
}
