package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table offline_session
(
    user_id        text not null,
    conn_id        text not null,
    refresh        blob not null,
    connector_data blob,
    primary key (user_id, conn_id)
);
*/

// OfflineSession holds the schema definition for the OfflineSession entity.
type OfflineSession struct {
	ent.Schema
}

// Fields of the OfflineSession.
func (OfflineSession) Fields() []ent.Field {
	return []ent.Field{
		// Using id field here because it's impossible to create multi-key primary yet
		field.Text("id").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.Text("user_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("conn_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.Bytes("refresh"),
		field.Bytes("connector_data").Nillable().Optional(),
	}
}

// Edges of the OfflineSession.
func (OfflineSession) Edges() []ent.Edge {
	return []ent.Edge{}
}
