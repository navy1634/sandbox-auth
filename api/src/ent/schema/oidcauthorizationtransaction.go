package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OIDCAuthorizationTransaction struct {
	ent.Schema
}

func (OIDCAuthorizationTransaction) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oidc_authorization_transactions"},
	}
}

func (OIDCAuthorizationTransaction) Fields() []ent.Field {
	return []ent.Field{
		field.Bytes("transaction_hash").Unique(),
		field.String("client_id"),
		field.String("redirect_uri"),
		field.JSON("scope", []string{}),
		field.String("state").Default(""),
		field.String("nonce"),
		field.String("code_challenge"),
		field.String("code_challenge_method"),
		field.Time("expires_at"),
		field.Time("consumed_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (OIDCAuthorizationTransaction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expires_at"),
		index.Fields("client_id", "expires_at"),
	}
}
