package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type OIDCClient struct {
	ent.Schema
}

func (OIDCClient) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oidc_clients"},
	}
}

func (OIDCClient) Fields() []ent.Field {
	return []ent.Field{
		field.String("client_id").Unique(),
		field.Bytes("client_secret_hash"),
		field.Bool("disabled").Default(false),
	}
}

func (OIDCClient) Mixin() []ent.Mixin {
	return []ent.Mixin{
		timeMixin{},
	}
}
