package plugin

import (
	"sort"
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http/internal/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type messageGenerator struct {
	pkg     protoreflect.FullName
	message protoreflect.MessageDescriptor
}

func (m messageGenerator) Generate(f *codegen.File) {
	commentGenerator{descriptor: m.message}.generateLeading(f, 0)
	className := scopedDescriptorTypeName(m.pkg, m.message)

	fields := make([]protoreflect.FieldDescriptor, 0, m.message.Fields().Len())
	rangeFields(m.message, func(field protoreflect.FieldDescriptor) {
		fields = append(fields, field)
	})
	sort.Slice(fields, func(i, j int) bool {
		return localeCompare(fields[i].JSONName(), fields[j].JSONName())
	})

	// class declaration
	f.P("class ", className, " {")

	// Fields
	for _, field := range fields {
		commentGenerator{descriptor: field}.generateLeading(f, 1)
		fieldType := typeFromField(m.pkg, field)
		ref := fieldType.Reference()
		if isNullableDartType(ref) {
			f.P(t(1), ref, "? ", dartFieldName(field.JSONName()), ";")
		} else {
			f.P(t(1), ref, " ", dartFieldName(field.JSONName()), ";")
		}
	}
	f.P()

	// Constructor
	f.P(t(1), className, "({")
	for _, field := range fields {
		f.P(t(2), "this.", dartFieldName(field.JSONName()), ",")
	}
	f.P(t(1), "});")
	f.P()

	// fromJson
	f.P(t(1), "factory ", className, ".fromJson(Map<String, dynamic> json) {")
	f.P(t(2), "return ", className, "(")
	for _, field := range fields {
		f.P(t(3), dartFieldName(field.JSONName()), ": ", fromJsonExpr(m.pkg, field), ",")
	}
	f.P(t(2), ");")
	f.P(t(1), "}")
	f.P()

	// toJson
	f.P(t(1), "Map<String, dynamic> toJson() {")
	f.P(t(2), "final json = <String, dynamic>{};")
	for _, field := range fields {
		for _, line := range toJsonStmt(field) {
			f.P(t(2), line)
		}
	}
	f.P(t(2), "return json;")
	f.P(t(1), "}")
	f.P()

	// toString
	toStringParts := make([]string, 0, len(fields))
	for _, field := range fields {
		fname := dartFieldName(field.JSONName())
		toStringParts = append(toStringParts, fname+": $"+fname)
	}
	// Escape $ in className so Dart does not treat it as string interpolation.
	escapedClassName := strings.ReplaceAll(className, "$", "\\$")
	toStringVal := escapedClassName + "(" + strings.Join(toStringParts, ", ") + ")"
	f.P(t(1), "@override")
	f.P(t(1), "String toString() {")
	f.P(t(2), "return '", toStringVal, "';")
	f.P(t(1), "}")
	f.P()

	// operator ==
	f.P(t(1), "@override")
	f.P(t(1), "bool operator ==(Object other) =>")
	f.P(t(2), "identical(this, other) ||")
	f.P(t(2), "other is ", className, " &&")
	f.P(t(3), "runtimeType == other.runtimeType")
	for _, field := range fields {
		fname := dartFieldName(field.JSONName())
		f.P(t(3), "&& ", fname, " == other.", fname)
	}
	f.P(t(2), ";")
	f.P()

	// hashCode
	f.P(t(1), "@override")
	f.P(t(1), "int get hashCode => Object.hashAll([")
	for _, field := range fields {
		fname := dartFieldName(field.JSONName())
		f.P(t(2), fname, ",")
	}
	f.P(t(1), "]);")
	f.P()

	// copyWith
	f.P(t(1), className, " copyWith({")
	for _, field := range fields {
		fieldType := typeFromField(m.pkg, field)
		ref := fieldType.Reference()
		fname := dartFieldName(field.JSONName())
		if isNullableDartType(ref) {
			f.P(t(2), ref, "? ", fname, ",")
		} else {
			f.P(t(2), ref, " ", fname, ",")
		}
	}
	f.P(t(1), "}) {")
	f.P(t(2), "return ", className, "(")
	for _, field := range fields {
		fname := dartFieldName(field.JSONName())
		f.P(t(3), fname, ": ", fname, " ?? this.", fname, ",")
	}
	f.P(t(2), ");")
	f.P(t(1), "}")

	f.P("}")
	f.P()
}

// fromJsonExpr returns the Dart expression to deserialize a field from JSON.
func fromJsonExpr(pkg protoreflect.FullName, field protoreflect.FieldDescriptor) string {
	jsonKey := dartString(field.JSONName())

	switch {
	case field.IsMap():
		valType := namedTypeFromField(pkg, field.MapValue())
		valCat := fieldCategoryOf(field.MapValue())
		switch valCat {
		case categoryMessage:
			return "(json[" + jsonKey + "] as Map<String, dynamic>?)?.map((k, v) => MapEntry(k, " + valType.Name + ".fromJson(v as Map<String, dynamic>)))"
		case categoryEnum:
			return "(json[" + jsonKey + "] as Map<String, dynamic>?)?.map((k, v) => MapEntry(k, " + valType.Name + ".fromString(v as String)))"
		default:
			if valType.Name == "double" {
				return "(json[" + jsonKey + "] as Map<String, dynamic>?)?.map((k, v) => MapEntry(k, (v as num).toDouble()))"
			}
			if valType.Name == "dynamic" {
				return "(json[" + jsonKey + "] as Map<String, dynamic>?)?.map((k, v) => MapEntry(k, v))"
			}
			return "(json[" + jsonKey + "] as Map<String, dynamic>?)?.map((k, v) => MapEntry(k, v as " + valType.Name + "))"
		}

	case field.IsList():
		valType := namedTypeFromField(pkg, field)
		valCat := fieldCategoryOf(field)
		switch valCat {
		case categoryMessage:
			return "(json[" + jsonKey + "] as List<dynamic>?)?.map((e) => " + valType.Name + ".fromJson(e as Map<String, dynamic>)).toList()"
		case categoryEnum:
			return "(json[" + jsonKey + "] as List<dynamic>?)?.map((e) => " + valType.Name + ".fromString(e as String)).toList()"
		default:
			if valType.Name == "double" {
				return "(json[" + jsonKey + "] as List<dynamic>?)?.map((e) => (e as num).toDouble()).toList()"
			}
			if valType.Name == "dynamic" {
				return "json[" + jsonKey + "] as List<dynamic>?"
			}
			return "(json[" + jsonKey + "] as List<dynamic>?)?.map((e) => e as " + valType.Name + ").toList()"
		}

	default:
		cat := fieldCategoryOf(field)
		fieldType := namedTypeFromField(pkg, field)
		switch cat {
		case categoryMessage:
			return "json[" + jsonKey + "] != null ? " + fieldType.Name + ".fromJson(json[" + jsonKey + "] as Map<String, dynamic>) : null"
		case categoryEnum:
			return "json[" + jsonKey + "] != null ? " + fieldType.Name + ".fromString(json[" + jsonKey + "] as String) : null"
		default:
			if fieldType.Name == "double" {
				return "(json[" + jsonKey + "] as num?)?.toDouble()"
			}
			if fieldType.Name == "dynamic" {
				return "json[" + jsonKey + "]"
			}
			return "json[" + jsonKey + "] as " + fieldType.Name + "?"
		}
	}
}

// toJsonStmt returns Dart statements to serialize a field to JSON.
func toJsonStmt(field protoreflect.FieldDescriptor) []string {
	jsonKey := dartString(field.JSONName())
	fname := dartFieldName(field.JSONName())

	switch {
	case field.IsMap():
		valCat := fieldCategoryOf(field.MapValue())
		switch valCat {
		case categoryMessage:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + "!.map((k, v) => MapEntry(k, v.toJson()));",
			}
		case categoryEnum:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + "!.map((k, v) => MapEntry(k, v.value));",
			}
		default:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + ";",
			}
		}

	case field.IsList():
		cat := fieldCategoryOf(field)
		switch cat {
		case categoryMessage:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + "!.map((e) => e.toJson()).toList();",
			}
		case categoryEnum:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + "!.map((e) => e.value).toList();",
			}
		default:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + ";",
			}
		}

	default:
		cat := fieldCategoryOf(field)
		switch cat {
		case categoryMessage:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + "!.toJson();",
			}
		case categoryEnum:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + "!.value;",
			}
		default:
			return []string{
				"if (" + fname + " != null) json[" + jsonKey + "] = " + fname + ";",
			}
		}
	}
}
