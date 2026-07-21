package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
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
		field.Bytes("webauthn_user_handle").Unique(),
	}
}

func (Account) Indexes() []ent.Index {
	return []ent.Index{}
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
