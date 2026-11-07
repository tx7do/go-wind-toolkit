package plugin

import (
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	wellKnownPrefix  = "google.protobuf."
	googleTypePrefix = "google.type."
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

	// google.type types — kept in lockstep with protoc-gen-dart-http.
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
		return strings.HasPrefix(fullName, wellKnownPrefix) || strings.HasPrefix(fullName, googleTypePrefix)
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

func (wkt WellKnown) TypeDeclaration() string {
	w := new(codegen.File)
	switch wkt {
	case WellKnownAny:
		w.P("// If the Any contains a value that has a special JSON mapping,")
		w.P("// it will be converted as follows:")
		w.P("// {\"@type\": xxx, \"value\": yyy}.")
		w.P("// Otherwise, the value will be converted into a JSON object,")
		w.P("// and the \"@type\" field will be inserted to indicate the actual data type.")
		w.P("interface ", wkt.Name(), " {")
		w.P("  ", "'@type': string;")
		w.P("  [key: string]: unknown;")
		w.P("}")
	case WellKnownDuration:
		w.P("// Generated output always contains 0, 3, 6, or 9 fractional digits,")
		w.P("// depending on required precision, followed by the suffix \"s\".")
		w.P("// Accepted are any fractional digits (also none) as long as they fit")
		w.P("// into nano-seconds precision and the suffix \"s\" is required.")
		w.P("type ", wkt.Name(), " = string;")
	case WellKnownEmpty:
		w.P("// An empty JSON object")
		w.P("type ", wkt.Name(), " = Record<never, never>;")
	case WellKnownTimestamp:
		w.P("// Encoded using RFC 3339, where generated output will always be Z-normalized")
		w.P("// and uses 0, 3, 6 or 9 fractional digits.")
		w.P("// Offsets other than \"Z\" are also accepted.")
		w.P("type ", wkt.Name(), " = string;")
	case WellKnownFieldMask:
		w.P("// In JSON, a field mask is encoded as a single string where paths are")
		w.P("// separated by a comma. Fields name in each path are converted")
		w.P("// to/from lower-camel naming conventions.")
		w.P("// As an example, consider the following message declarations:")
		w.P("//")
		w.P("//     message Profile {")
		w.P("//       User user = 1;")
		w.P("//       Photo photo = 2;")
		w.P("//     }")
		w.P("//     message User {")
		w.P("//       string display_name = 1;")
		w.P("//       string address = 2;")
		w.P("//     }")
		w.P("//")
		w.P("// In proto a field mask for `Profile` may look as such:")
		w.P("//")
		w.P("//     mask {")
		w.P("//       paths: \"user.display_name\"")
		w.P("//       paths: \"photo\"")
		w.P("//     }")
		w.P("//")
		w.P("// In JSON, the same mask is represented as below:")
		w.P("//")
		w.P("//     {")
		w.P("//       mask: \"user.displayName,photo\"")
		w.P("//     }")
		w.P("type ", wkt.Name(), " = string;")
	case WellKnownFloatValue,
		WellKnownDoubleValue,
		WellKnownInt64Value,
		WellKnownInt32Value,
		WellKnownUInt64Value,
		WellKnownUInt32Value:
		w.P("type ", wkt.Name(), " = null | number;")
	case WellKnownBytesValue, WellKnownStringValue:
		w.P("type ", wkt.Name(), " = null | string;")
	case WellKnownBoolValue:
		w.P("type ", wkt.Name(), " = null | boolean;")
	case WellKnownStruct:
		w.P("// Any JSON value.")
		w.P("type ", wkt.Name(), " = Record<string, unknown>;")
	case WellKnownValue:
		w.P("type ", wkt.Name(), " = unknown;")
	case WellKnownNullValue:
		w.P("type ", wkt.Name(), " = null;")
	case WellKnownListValue:
		w.P("type ", wkt.Name(), " = ", WellKnownValue.Name(), "[];")
	case WellKnownLatLng:
		w.P("// An object that represents a latitude/longitude pair")
		w.P("// (canonical JSON: {\"latitude\": number, \"longitude\": number}).")
		w.P("type ", wkt.Name(), " = Record<string, unknown>;")
	case WellKnownDate:
		w.P("// Represents a whole or partial calendar date")
		w.P("// (canonical JSON: {\"year\": number, \"month\": number, \"day\": number};")
		w.P("// omitted fields are zero, not optional).")
		w.P("type ", wkt.Name(), " = Record<string, unknown>;")
	case WellKnownTimeOfDay:
		w.P("// Represents a time of day")
		w.P("// (canonical JSON: {\"hours\": number, \"minutes\": number,")
		w.P("// \"seconds\": number, \"nanos\": number}; omitted fields are zero).")
		w.P("type ", wkt.Name(), " = Record<string, unknown>;")
	case WellKnownMoney:
		w.P("// Represents an amount of money with its currency type")
		w.P("// (canonical JSON: {\"currencyCode\": string, \"units\": string, \"nanos\": number};")
		w.P("// `units` is a 64-bit amount encoded as a decimal string).")
		w.P("type ", wkt.Name(), " = Record<string, unknown>;")
	case WellKnownDayOfWeek, WellKnownMonth:
		w.P("// Enum serialized as its string value in canonical JSON.")
		w.P("type ", wkt.Name(), " = string;")
	default:
		w.P("// No mapping for this well known type is generated, yet.")
		w.P("type ", wkt.Name(), " = unknown;")
	}
	return string(w.Content())
}
