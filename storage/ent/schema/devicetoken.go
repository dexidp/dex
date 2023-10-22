package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table device_token
(
    device_code           text            not null primary key,
    status                text            not null,
    token                 blob,
    expiry                timestamp       not null,
    last_request          timestamp       not null,
    poll_interval         integer         not null,
    code_challenge        text default '' not null,
    code_challenge_method text default '' not null
);
*/

// DeviceToken holds the schema definition for the DeviceToken entity.
type DeviceToken struct {
	ent.Schema
}

// Fields of the DeviceToken.
func (DeviceToken) Fields() []ent.Field {
	return []ent.Field{
		field.Text("device_code").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.Text("status").
			SchemaType(textSchema).
			NotEmpty(),
		field.Bytes("token").Nillable().Optional(),
		field.Time("expiry").
			SchemaType(timeSchema),
		field.Time("last_request").
			SchemaType(timeSchema),
		field.Int("poll_interval"),
		field.Text("code_challenge").
			SchemaType(textSchema).
			Default(""),
		field.Text("code_challenge_method").
			SchemaType(textSchema).
			Default(""),
	}
}

// Edges of the DeviceToken.
func (DeviceToken) Edges() []ent.Edge {
	return []ent.Edge{}
}
