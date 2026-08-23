package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OIDCClientRedirectURI struct {
	ent.Schema
}

func (OIDCClientRedirectURI) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "oidc_client_redirect_uris"},
	}
}

func (OIDCClientRedirectURI) Fields() []ent.Field {
	return []ent.Field{
		field.String("client_id"),
		field.String("redirect_uri"),
	}
}

func (OIDCClientRedirectURI) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("client_id", "redirect_uri").Unique(),
		index.Fields("client_id"),
	}
}
