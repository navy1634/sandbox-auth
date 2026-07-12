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

type WebauthnSession struct {
	ent.Schema
}

func (WebauthnSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "webauthn_sessions"},
	}
}

func (WebauthnSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique(),
		field.Int64("account_id").Optional().Nillable(),
		field.String("ceremony"),
		field.JSON("session_json", json.RawMessage{}),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (WebauthnSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expires_at"),
	}
}
