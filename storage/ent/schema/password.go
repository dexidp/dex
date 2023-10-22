package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table password
(
    email    text not null  primary key,
    hash     blob not null,
    username text not null,
    user_id  text not null
);
*/

// Password holds the schema definition for the Password entity.
type Password struct {
	ent.Schema
}

// Fields of the Password.
func (Password) Fields() []ent.Field {
	return []ent.Field{
		field.Text("email").
			SchemaType(textSchema).
			StorageKey("email"). // use email as ID field to make querying easier
			NotEmpty().
			Unique(),
		field.Bytes("hash"),
		field.Text("username").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("user_id").
			SchemaType(textSchema).
			NotEmpty(),
	}
}

// Edges of the Password.
func (Password) Edges() []ent.Edge {
	return []ent.Edge{}
}
