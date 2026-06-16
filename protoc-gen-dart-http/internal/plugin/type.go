package plugin

import "google.golang.org/protobuf/reflect/protoreflect"

type Type struct {
	IsNamed bool
	Name    string

	IsList     bool
	IsMap      bool
	Underlying *Type
}

// Reference returns the Dart type string for this type.
// All proto3 scalar fields are nullable in JSON, so we append "?"
// for non-list, non-map types.
func (typ Type) Reference() string {
	switch {
	case typ.IsMap:
		return "Map<String, " + typ.Underlying.Reference() + ">"
	case typ.IsList:
		return "List<" + typ.Underlying.Reference() + ">"
	default:
		return typ.Name
	}
}

// fieldCategory classifies a field for serialization purposes.
type fieldCategory int

const (
	categoryScalar  fieldCategory = iota // String, int, double, bool, dynamic
	categoryMessage                      // message type → needs fromJson/toJson
	categoryEnum                         // enum type → needs fromString/value
)

func typeFromField(pkg protoreflect.FullName, field protoreflect.FieldDescriptor) Type {
	switch {
	case field.IsMap():
		underlying := namedTypeFromField(pkg, field.MapValue())
		return Type{
			IsMap:      true,
			Underlying: &underlying,
		}
	case field.IsList():
		underlying := namedTypeFromField(pkg, field)
		return Type{
			IsList:     true,
			Underlying: &underlying,
		}
	default:
		return namedTypeFromField(pkg, field)
	}
}

func namedTypeFromField(pkg protoreflect.FullName, field protoreflect.FieldDescriptor) Type {
	switch field.Kind() {
	case protoreflect.StringKind:
		return Type{IsNamed: true, Name: "String"}
	case protoreflect.BytesKind:
		return Type{IsNamed: true, Name: "String"} // base64 encoded in JSON
	case protoreflect.BoolKind:
		return Type{IsNamed: true, Name: "bool"}
	case
		protoreflect.Int32Kind,
		protoreflect.Int64Kind,
		protoreflect.Uint32Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Fixed64Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sint64Kind:
		return Type{IsNamed: true, Name: "int"}
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return Type{IsNamed: true, Name: "double"}
	case protoreflect.MessageKind:
		return typeFromMessage(pkg, field.Message())
	case protoreflect.EnumKind:
		desc := field.Enum()
		if wkt, ok := WellKnownType(field.Enum()); ok {
			return Type{IsNamed: true, Name: wkt.DartType()}
		}
		return Type{IsNamed: true, Name: scopedDescriptorTypeName(pkg, desc)}
	default:
		return Type{IsNamed: true, Name: "dynamic"}
	}
}

func typeFromMessage(pkg protoreflect.FullName, message protoreflect.MessageDescriptor) Type {
	if wkt, ok := WellKnownType(message); ok {
		return Type{IsNamed: true, Name: wkt.DartType()}
	}
	return Type{IsNamed: true, Name: scopedDescriptorTypeName(pkg, message)}
}

// isInt64Field returns true if the field is a 64-bit integer type.
// In proto3 JSON encoding, int64/uint64/sint64/fixed64/sfixed64 and the
// well-known wrappers Int64Value/UInt64Value are represented as JSON strings
// to avoid precision loss, so deserialization must accept both String and int.
func isInt64Field(field protoreflect.FieldDescriptor) bool {
	switch field.Kind() {
	case protoreflect.Int64Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Sint64Kind:
		return true
	case protoreflect.MessageKind:
		if wkt, ok := WellKnownType(field.Message()); ok {
			return wkt == WellKnownInt64Value || wkt == WellKnownUInt64Value
		}
	}
	return false
}

// fieldCategoryOf returns the serialization category for a field.
func fieldCategoryOf(field protoreflect.FieldDescriptor) fieldCategory {
	if field.Kind() == protoreflect.MessageKind {
		if IsWellKnownType(field.Message()) {
			return categoryScalar
		}
		return categoryMessage
	}
	if field.Kind() == protoreflect.EnumKind {
		if IsWellKnownType(field.Enum()) {
			return categoryScalar
		}
		return categoryEnum
	}
	return categoryScalar
}

// mapValueCategoryOf returns the serialization category for a map field's value.
func mapValueCategoryOf(field protoreflect.FieldDescriptor) fieldCategory {
	return fieldCategoryOf(field.MapValue())
}
