package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Group holds the schema definition for the Group entity.
type Group struct {
	ent.Schema
}

func (Group) Mixin() []ent.Mixin {
	return []ent.Mixin{}
}

func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("users").Type(User.Type).Ref("groups"),
	}
}
