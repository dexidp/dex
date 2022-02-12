package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

/* Original SQL table:
create table device_request
(
    user_code     text      not null  primary key,
    device_code   text      not null,
    client_id     text      not null,
    client_secret text,
    scopes        blob      not null,
    expiry        timestamp not null
);
*/

// DeviceRequest holds the schema definition for the DeviceRequest entity.
type DeviceRequest struct {
	ent.Schema
}

// Fields of the DeviceRequest.
func (DeviceRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Text("user_code").
			SchemaType(textSchema).
			NotEmpty().
			Unique(),
		field.Text("device_code").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("client_id").
			SchemaType(textSchema).
			NotEmpty(),
		field.Text("client_secret").
			SchemaType(textSchema).
			NotEmpty(),
		field.JSON("scopes", []string{}).
			Optional(),
		field.Time("expiry").
			SchemaType(timeSchema),
	}
}

// Edges of the DeviceRequest.
func (DeviceRequest) Edges() []ent.Edge {
	return []ent.Edge{}
}
