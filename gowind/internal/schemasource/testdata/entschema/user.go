package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(64).Comment("用户昵称"),
		field.Int("age").Optional(),
		field.String("email").Optional().Nillable(),
		field.Enum("role").Values("admin", "user"),
		field.String("legacy_column").StorageKey("old_col"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("pets", Pet.Type),
		edge.To("groups", Group.Type),
		edge.To("card", Card.Type).Unique(),
		edge.To("subordinates", User.Type),
	}
}
