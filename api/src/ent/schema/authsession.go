package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuthSession struct {
	ent.Schema
}

func (AuthSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_sessions"},
	}
}

func (AuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.Bytes("token_hash").Unique(),
		field.Int64("account_id"),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (AuthSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "expires_at"),
		index.Fields("expires_at"),
	}
}
