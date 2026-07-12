package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

type Account struct {
	ent.Schema
}

func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "accounts"},
	}
}

func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.String("provider"),
		field.String("provider_account_id"),
		field.String("email"),
		field.Bool("email_verified").Default(false),
		field.String("name").Default(""),
		field.String("picture").Default(""),
		field.String("display_name").Default(""),
		field.String("bio").Default(""),
		field.Time("registered_at").Optional().Nillable(),
		field.Bytes("webauthn_user_handle").Unique(),
	}
}

func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider", "provider_account_id").Unique(),
		index.Fields("email"),
	}
}

func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		timeMixin{},
	}
}

type timeMixin struct {
	mixin.Schema
}

func (timeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
