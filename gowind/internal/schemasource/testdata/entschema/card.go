package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Card holds the schema definition for the Card entity.
type Card struct {
	ent.Schema
}

func (Card) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "member_cards"},
	}
}

func (Card) Fields() []ent.Field {
	return []ent.Field{
		field.String("number"),
	}
}

func (Card) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner").Type(User.Type).Ref("card").Unique(),
	}
}
