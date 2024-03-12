package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/go-jose/go-jose/v4"

	"github.com/dexidp/dex/storage"
)

/* Original SQL table:
create table keys
(
    id                text      not null  primary key,
    verification_keys blob      not null,
    signing_key       blob      not null,
    signing_key_pub   blob      not null,
    next_rotation     timestamp not null
);
*/

// Keys holds the schema definition for the Keys entity.
type Keys struct {
	ent.Schema
}

// Fields of the Keys.
func (Keys) Fields() []ent.Field {
	return []ent.Field{
		field.Text("id").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.JSON("verification_keys", []storage.VerificationKey{}),
		field.JSON("signing_key", jose.JSONWebKey{}),
		field.JSON("signing_key_pub", jose.JSONWebKey{}),
		field.Time("next_rotation").
			SchemaType(timeSchema),
	}
}

// Edges of the Keys.
func (Keys) Edges() []ent.Edge {
	return []ent.Edge{}
}
