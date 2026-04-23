package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// UserIdentity holds the schema definition for the UserIdentity entity.
type UserIdentity struct {
	ent.Schema
}

// Fields of the UserIdentity.
func (UserIdentity) Fields() []ent.Field {
	return []ent.Field{
		// Using id field here because it's impossible to create multi-key primary yet
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
		field.Text("claims_user_id").
			SchemaType(textSchema).
			Default(""),
		field.Text("claims_username").
			SchemaType(textSchema).
			Default(""),
		field.Text("claims_preferred_username").
			SchemaType(textSchema).
			Default(""),
		field.Text("claims_email").
			SchemaType(textSchema).
			Default(""),
		field.Bool("claims_email_verified").
			Default(false),
		field.JSON("claims_groups", []string{}).
			Optional(),
		field.Bytes("consents"),
		field.Bytes("mfa_secrets").
			Nillable().
			Optional(),
		field.Bytes("webauthn_credentials").
			Nillable().
			Optional(),
		field.Time("created_at").
			SchemaType(timeSchema),
		field.Time("last_login").
			SchemaType(timeSchema),
		field.Time("blocked_until").
			SchemaType(timeSchema),
	}
}

// Edges of the UserIdentity.
func (UserIdentity) Edges() []ent.Edge {
	return []ent.Edge{}
}
