package plugin

import "google.golang.org/protobuf/reflect/protoreflect"

type Type struct {
	IsNamed bool
	Name    string

	IsList     bool
	IsMap      bool
	Underlying *Type
}

func (t Type) Reference() string {
	switch {
	case t.IsMap:
		return "{ [key: string]: " + t.Underlying.Reference() + " }"
	case t.IsList:
		return t.Underlying.Reference() + "[]"
	default:
		return t.Name
	}
}

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
	case protoreflect.StringKind, protoreflect.BytesKind:
		return Type{IsNamed: true, Name: "string"}
	case protoreflect.BoolKind:
		return Type{IsNamed: true, Name: "boolean"}
	// 64 位整数在 proto3 JSON 中序列化为字符串（JS number 只有 53 位安全整数），
	// 因此 TS 类型映射为 string，与运行时线上格式一致。
	case
		protoreflect.Int64Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Sint64Kind:
		return Type{IsNamed: true, Name: "string"}
	case
		protoreflect.Int32Kind,
		protoreflect.Uint32Kind,
		protoreflect.DoubleKind,
		protoreflect.Fixed32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sint32Kind,
		protoreflect.FloatKind:
		return Type{IsNamed: true, Name: "number"}
	case protoreflect.MessageKind:
		return typeFromMessage(pkg, field.Message())
	case protoreflect.EnumKind:
		desc := field.Enum()
		if wkt, ok := WellKnownType(field.Enum()); ok {
			return Type{IsNamed: true, Name: wkt.Name()}
		}
		return Type{IsNamed: true, Name: scopedDescriptorTypeName(pkg, desc)}
	default:
		return Type{IsNamed: true, Name: "unknown"}
	}
}

func typeFromMessage(pkg protoreflect.FullName, message protoreflect.MessageDescriptor) Type {
	if wkt, ok := WellKnownType(message); ok {
		return Type{IsNamed: true, Name: wkt.Name()}
	}
	return Type{IsNamed: true, Name: scopedDescriptorTypeName(pkg, message)}
}
