package plugin

import (
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// scopedDescriptorTypeName returns the Dart type name for a descriptor,
// scoped to the given package. Nested types are flattened using '$'
// (matching Dart protobuf convention), and cross-package references are
// prefixed with a PascalCase package prefix.
func scopedDescriptorTypeName(pkg protoreflect.FullName, desc protoreflect.Descriptor) string {
	name := string(desc.Name())
	var prefix string
	if desc.Parent() != desc.ParentFile() {
		prefix = descriptorTypeName(desc.Parent()) + "$"
	}
	if desc.ParentFile().Package() != pkg {
		prefix = packagePrefix(desc.ParentFile().Package()) + prefix
	}
	return prefix + name
}

// descriptorTypeName returns the flattened Dart type name for a descriptor,
// using '$' to separate nested levels (matching Dart protobuf convention).
func descriptorTypeName(desc protoreflect.Descriptor) string {
	name := string(desc.Name())
	var prefix string
	if desc.Parent() != desc.ParentFile() {
		prefix = descriptorTypeName(desc.Parent()) + "$"
	}
	return prefix + name
}

// packagePrefix converts a proto package name to a PascalCase prefix for
// cross-package type references.
// e.g. "einride.example.syntax.v1" -> "EinrideExampleSyntaxV1"
func packagePrefix(pkg protoreflect.FullName) string {
	parts := strings.Split(string(pkg), ".")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func rangeFields(message protoreflect.MessageDescriptor, f func(field protoreflect.FieldDescriptor)) {
	for i := 0; i < message.Fields().Len(); i++ {
		f(message.Fields().Get(i))
	}
}

func rangeMethods(methods protoreflect.MethodDescriptors, f func(method protoreflect.MethodDescriptor)) {
	for i := 0; i < methods.Len(); i++ {
		f(methods.Get(i))
	}
}

func rangeEnumValues(enum protoreflect.EnumDescriptor, f func(value protoreflect.EnumValueDescriptor, last bool)) {
	for i := 0; i < enum.Values().Len(); i++ {
		if i == enum.Values().Len()-1 {
			f(enum.Values().Get(i), true)
		} else {
			f(enum.Values().Get(i), false)
		}
	}
}

func t(n int) string {
	return strings.Repeat("  ", n)
}

// dartString wraps s in Dart single quotes, escaping backslashes and single
// quotes as needed.
func dartString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}

// lowerCamel converts a PascalCase or UPPER_SNAKE_CASE name to lowerCamelCase.
func lowerCamel(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 32
	}
	return string(r)
}

// protoEnumToDartName converts an UPPER_SNAKE_CASE proto enum value name to
// lowerCamelCase suitable for a Dart enum value.
// e.g. "STATUS_ACTIVE" → "statusActive", "ACTIVE" → "active"
func protoEnumToDartName(s string) string {
	parts := strings.Split(s, "_")
	var sb strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			sb.WriteString(strings.ToLower(p))
		} else {
			if len(p) > 0 {
				sb.WriteString(strings.ToUpper(p[:1]))
				sb.WriteString(strings.ToLower(p[1:]))
			}
		}
	}
	return sb.String()
}

// localeCompare compares two strings in a way that matches JavaScript's
// String.prototype.localeCompare for the common cases encountered in generated
// code.
func localeCompare(a, b string) bool {
	return sortKey(a) < sortKey(b)
}

// sortKey transforms a string into a comparison key where:
//   - Non-alphanumeric characters (punctuation, symbols) sort before alphanumerics
//   - Digits sort after punctuation but before letters
//   - Letters are compared case-insensitively (lowercase form used)
func sortKey(s string) string {
	var sb strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z':
			sb.WriteByte(2)
			sb.WriteByte(c)
		case c >= 'A' && c <= 'Z':
			sb.WriteByte(2)
			sb.WriteByte(c + 32)
		case c >= '0' && c <= '9':
			sb.WriteByte(1)
			sb.WriteByte(c)
		default:
			sb.WriteByte(0)
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// lowerFirst returns s with its first character lowercased.
func lowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	bytes := []byte(s)
	if bytes[0] >= 'A' && bytes[0] <= 'Z' {
		bytes[0] += 32
	}
	return string(bytes)
}

// dartReservedWords is the set of Dart reserved words and built-in identifiers
// that cannot be used as field names without escaping.
var dartReservedWords = map[string]bool{
	"assert": true, "break": true, "case": true, "catch": true, "class": true,
	"const": true, "continue": true, "default": true, "do": true, "else": true,
	"enum": true, "extends": true, "false": true, "final": true, "finally": true,
	"for": true, "if": true, "in": true, "is": true, "new": true, "null": true,
	"rethrow": true, "return": true, "super": true, "switch": true, "this": true,
	"throw": true, "true": true, "try": true, "var": true, "void": true,
	"while": true, "with": true, "abstract": true, "as": true, "covariant": true,
	"deferred": true, "dynamic": true, "export": true, "extension": true,
	"external": true, "factory": true, "Function": true, "get": true, "hide": true,
	"implements": true, "import": true, "interface": true, "library": true,
	"operator": true, "mixin": true, "part": true, "set": true, "static": true,
	"typedef": true, "late": true, "required": true, "call": true, "await": true,
	"yield": true, "sync": true, "async": true, "show": true,
}

// dartFieldName returns a safe Dart field name. If name is a Dart reserved
// word, an underscore is appended.
func dartFieldName(name string) string {
	if dartReservedWords[name] {
		return name + "_"
	}
	return name
}

// isNullableDartType returns false for Dart types that are inherently nullable
// (like dynamic) and don't accept the ? suffix.
func isNullableDartType(typeName string) bool {
	return typeName != "dynamic"
}
