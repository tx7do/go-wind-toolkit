package plugin

import (
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	wellKnownPrefix = "google.protobuf."
)

type WellKnown string

// https://developers.google.com/protocol-buffers/docs/reference/google.protobuf
const (
	WellKnownAny       WellKnown = "google.protobuf.Any"
	WellKnownDuration  WellKnown = "google.protobuf.Duration"
	WellKnownEmpty     WellKnown = "google.protobuf.Empty"
	WellKnownFieldMask WellKnown = "google.protobuf.FieldMask"
	WellKnownStruct    WellKnown = "google.protobuf.Struct"
	WellKnownTimestamp WellKnown = "google.protobuf.Timestamp"

	// Wrapper types.
	WellKnownFloatValue  WellKnown = "google.protobuf.FloatValue"
	WellKnownInt64Value  WellKnown = "google.protobuf.Int64Value"
	WellKnownInt32Value  WellKnown = "google.protobuf.Int32Value"
	WellKnownUInt64Value WellKnown = "google.protobuf.UInt64Value"
	WellKnownUInt32Value WellKnown = "google.protobuf.UInt32Value"
	WellKnownBytesValue  WellKnown = "google.protobuf.BytesValue"
	WellKnownDoubleValue WellKnown = "google.protobuf.DoubleValue"
	WellKnownBoolValue   WellKnown = "google.protobuf.BoolValue"
	WellKnownStringValue WellKnown = "google.protobuf.StringValue"

	// Descriptor types.
	WellKnownValue     WellKnown = "google.protobuf.Value"
	WellKnownNullValue WellKnown = "google.protobuf.NullValue"
	WellKnownListValue WellKnown = "google.protobuf.ListValue"

	// google.type types.
	WellKnownLatLng    WellKnown = "google.type.LatLng"
	WellKnownDate      WellKnown = "google.type.Date"
	WellKnownTimeOfDay WellKnown = "google.type.TimeOfDay"
	WellKnownMoney     WellKnown = "google.type.Money"
	WellKnownDayOfWeek WellKnown = "google.type.DayOfWeek"
	WellKnownMonth     WellKnown = "google.type.Month"
)

func IsWellKnownType(desc protoreflect.Descriptor) bool {
	switch desc.(type) {
	case protoreflect.MessageDescriptor, protoreflect.EnumDescriptor:
		fullName := string(desc.FullName())
		return strings.HasPrefix(fullName, wellKnownPrefix) || strings.HasPrefix(fullName, "google.type.")
	default:
		return false
	}
}

func WellKnownType(desc protoreflect.Descriptor) (WellKnown, bool) {
	if !IsWellKnownType(desc) {
		return "", false
	}
	return WellKnown(desc.FullName()), true
}

// Name returns a short identifier for the well-known type.
func (wkt WellKnown) Name() string {
	return "wellKnown" + shortName(string(wkt))
}

// DartType returns the Dart type that maps to this well-known type.
func (wkt WellKnown) DartType() string {
	switch wkt {
	case WellKnownAny:
		return "Map<String, dynamic>"
	case WellKnownDuration:
		return "String"
	case WellKnownEmpty:
		return "Map<String, dynamic>"
	case WellKnownTimestamp:
		return "String"
	case WellKnownFieldMask:
		return "String"
	case WellKnownFloatValue:
		return "double"
	case WellKnownDoubleValue:
		return "double"
	case WellKnownInt64Value:
		return "int"
	case WellKnownInt32Value:
		return "int"
	case WellKnownUInt64Value:
		return "int"
	case WellKnownUInt32Value:
		return "int"
	case WellKnownBytesValue:
		return "String"
	case WellKnownStringValue:
		return "String"
	case WellKnownBoolValue:
		return "bool"
	case WellKnownStruct:
		return "Map<String, dynamic>"
	case WellKnownValue:
		return "dynamic"
	case WellKnownNullValue:
		return "String"
	case WellKnownListValue:
		return "List<dynamic>"
	case WellKnownLatLng:
		return "Map<String, dynamic>"
	case WellKnownDate:
		return "Map<String, dynamic>"
	case WellKnownTimeOfDay:
		return "Map<String, dynamic>"
	case WellKnownMoney:
		return "Map<String, dynamic>"
	case WellKnownDayOfWeek:
		return "String"
	case WellKnownMonth:
		return "String"
	default:
		return "dynamic"
	}
}

// TypeDeclaration returns a Dart comment documenting the well-known type mapping.
func (wkt WellKnown) TypeDeclaration() string {
	var w writer
	shortName := shortName(string(wkt))
	switch wkt {
	case WellKnownAny:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `Map<String, dynamic>`.")
		w.P("///")
		w.P("/// In JSON: `{\"@type\": xxx, ...}`.")
	case WellKnownDuration:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `String` (e.g. `\"3.5s\"`).")
	case WellKnownEmpty:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `Map<String, dynamic>`.")
	case WellKnownTimestamp:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `String` (RFC 3339, e.g. `\"2021-01-01T00:00:00Z\"`).")
	case WellKnownFieldMask:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `String` (comma-separated camelCase paths).")
	case WellKnownFloatValue, WellKnownDoubleValue:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `double`.")
	case WellKnownInt64Value, WellKnownInt32Value, WellKnownUInt64Value, WellKnownUInt32Value:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `int`.")
	case WellKnownBytesValue, WellKnownStringValue:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `String`.")
	case WellKnownBoolValue:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `bool`.")
	case WellKnownStruct:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `Map<String, dynamic>`.")
	case WellKnownValue:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `dynamic`.")
	case WellKnownNullValue:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `String`.")
	case WellKnownListValue:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `List<dynamic>`.")
	default:
		w.P("/// Well-known type: ", shortName)
		w.P("///")
		w.P("/// Maps to Dart: `", wkt.DartType(), "`.")
	}
	return w.String()
}

// shortName extracts the short type name from the full proto name.
// e.g. "google.protobuf.Timestamp" → "Timestamp"
//
//	"google.type.LatLng" → "LatLng"
func shortName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

type writer struct {
	b strings.Builder
}

func (w *writer) P(ss ...string) {
	for _, s := range ss {
		// strings.Builder never returns an error, so safe to ignore
		_, _ = w.b.WriteString(s)
	}
	_, _ = w.b.WriteString("\n")
}

func (w *writer) String() string {
	return w.b.String()
}
