package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OIDCAuthorizationCode struct {
	ent.Schema
}

func (OIDCAuthorizationCode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oidc_authorization_codes"},
	}
}

func (OIDCAuthorizationCode) Fields() []ent.Field {
	return []ent.Field{
		field.Bytes("code_hash").Unique(),
		field.String("client_id"),
		field.String("redirect_uri"),
		field.Int64("account_id"),
		field.JSON("scope", []string{}),
		field.String("nonce"),
		field.String("code_challenge"),
		field.String("code_challenge_method"),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (OIDCAuthorizationCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expires_at"),
		index.Fields("client_id", "expires_at"),
		index.Fields("account_id", "expires_at"),
	}
}
