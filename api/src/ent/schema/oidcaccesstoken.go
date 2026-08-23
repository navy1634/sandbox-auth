package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OIDCAccessToken struct {
	ent.Schema
}

func (OIDCAccessToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oidc_access_tokens"},
	}
}

func (OIDCAccessToken) Fields() []ent.Field {
	return []ent.Field{
		field.Bytes("token_hash").Unique(),
		field.String("client_id"),
		field.Int64("account_id"),
		field.JSON("scope", []string{}),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (OIDCAccessToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expires_at"),
		index.Fields("account_id", "expires_at"),
	}
}
