package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WebauthnCredential struct {
	ent.Schema
}

func (WebauthnCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "webauthn_credentials"},
	}
}

func (WebauthnCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Bytes("credential_id").Unique(),
		field.JSON("credential_json", json.RawMessage{}),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("last_used_at").Optional().Nillable(),
	}
}

func (WebauthnCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id"),
	}
}
