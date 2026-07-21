package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuthIdentity struct {
	ent.Schema
}

func (AuthIdentity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_identities"},
	}
}

func (AuthIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.String("provider"),
		field.String("provider_account_id"),
		field.String("email"),
		field.Bool("email_verified").Default(false),
	}
}

func (AuthIdentity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "provider").Unique(),
		index.Fields("provider", "provider_account_id").Unique(),
		index.Fields("account_id"),
		index.Fields("email"),
	}
}

func (AuthIdentity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		timeMixin{},
	}
}
