package plugin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/protowalk"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// 本文件负责为每个消息类发射二进制 proto 编解码成员：
//
//   - Uint8List writeToBuffer()   —— 编码入口
//   - void _writeTo(w)            —— 按字段号序逐字段写出（嵌套复用）
//   - factory X.fromBuffer(bytes) —— 解码入口
//   - static X _readFrom(r)       —— 字段循环 + 按 fieldNumber 分发
//
// plain 类字段全部可空，null 即 proto3 字段缺位，语义与 optional 对齐。
// 64 位整型在 flutter web 上受 JS number 精度限制（与 JSON 表示同病）。

// wireField 返回按字段号升序的字段列表（wire 编解码的发射顺序）。
func wireFields(message protoreflect.MessageDescriptor) []protoreflect.FieldDescriptor {
	fields := make([]protoreflect.FieldDescriptor, 0, message.Fields().Len())
	protowalk.RangeFields(message, func(field protoreflect.FieldDescriptor) {
		fields = append(fields, field)
	})
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Number() < fields[j].Number()
	})
	return fields
}

// wireGuard 生成字段写出的空值守卫：先拷贝到局部变量再判空（局部变量判空
// 后类型已提升，表达式内不需要也不应该再写 !）。字段缺位时整体跳过写出。
func wireGuard(n int, fname, expr string) []string {
	return []string{
		"final f" + fmt.Sprint(n) + " = " + fname + ";",
		"if (f" + fmt.Sprint(n) + " != null) {",
		"  " + expr,
		"}",
	}
}

// wireWriteStmts 返回 _writeTo 内单个字段的写出语句块。
func wireWriteStmts(pkg protoreflect.FullName, field protoreflect.FieldDescriptor) []string {
	fname := dartFieldName(field.JSONName())
	n := int(field.Number())

	guard := func(expr string) []string {
		return wireGuard(n, fname, expr)
	}

	switch {
	case field.IsMap():
		return wireMapWriteStmts(pkg, field)

	case field.IsList():
		if fieldCategoryOf(field) == categoryScalar && field.Kind() == protoreflect.MessageKind {
			// repeated WKT 标量（如 repeated Timestamp）
			wkt := mustWellKnown(field.Message())
			if !wkt.isWireSupported() {
				return guard(wkt.wireUnsupportedStmt())
			}
			// 元素无关的 WKT（如 Empty）用 _ 通配，避免未使用变量。
			loopVar := "e"
			if !wkt.wireWriteUsesElement() {
				loopVar = "_"
			}
			return guard("for (final " + loopVar + " in f" + fmt.Sprint(n) + ") {" +
				" " + wkt.wireWriteCall(pkg, n, loopVar) + " }")
		}
		switch field.Kind() {
		case protoreflect.MessageKind:
			return guard("for (final e in f" + fmt.Sprint(n) + ") {" +
				" final cw = ProtoWireWriter(); e._writeTo(cw); w.writeRaw(" + fmt.Sprint(n) + ", cw.toBuffer()); }")
		case protoreflect.StringKind:
			return guard("w.writeStringList(" + fmt.Sprint(n) + ", f" + fmt.Sprint(n) + ");")
		case protoreflect.BytesKind:
			return guard("w.writeBytesList(" + fmt.Sprint(n) + ", f" + fmt.Sprint(n) + ");")
		case protoreflect.EnumKind:
			// WKT 枚举（NullValue/DayOfWeek/Month）映射为 String，无 wire 枚举可写。
			if wkt, ok := WellKnownType(field.Enum()); ok {
				return guard(wkt.wireUnsupportedStmt())
			}
			enumName := namedTypeFromField(pkg, field).Name
			return guard("w.writePackedVarintList(" + fmt.Sprint(n) +
				", f" + fmt.Sprint(n) + ".map((e) => e.wire).toList());" +
				" // " + enumName)
		default:
			if m := wirePackedWriter(field.Kind()); m != "" {
				if field.Kind() == protoreflect.FloatKind || field.Kind() == protoreflect.DoubleKind {
					return guard("w." + m + "(" + fmt.Sprint(n) + ", f" + fmt.Sprint(n) + ");")
				}
				return guard("w." + m + "(" + fmt.Sprint(n) + ", f" + fmt.Sprint(n) + ");")
			}
			// 不支持的打包列表：仅在字段实际出现时抛错，缺位时其余字段仍可编码。
			return guard("throw UnsupportedError('dart-http wire: list field " + string(field.FullName()) + "');")
		}

	default:
		switch field.Kind() {
		case protoreflect.MessageKind:
			if IsWellKnownType(field.Message()) {
				wkt := mustWellKnown(field.Message())
				return guard(wkt.wireWriteCall(pkg, n, "f"+fmt.Sprint(n)))
			}
			typ := namedTypeFromField(pkg, field).Name
			return guard("final cw = ProtoWireWriter(); f" + fmt.Sprint(n) + "._writeTo(cw); w.writeRaw(" + fmt.Sprint(n) + ", cw.toBuffer()); // " + typ)
		case protoreflect.EnumKind:
			// WKT 枚举（NullValue/DayOfWeek/Month）映射为 String，无 wire 枚举可写。
			if wkt, ok := WellKnownType(field.Enum()); ok {
				return guard(wkt.wireUnsupportedStmt())
			}
			enumName := namedTypeFromField(pkg, field).Name
			return guard("w.writeEnum(" + fmt.Sprint(n) + ", f" + fmt.Sprint(n) + ".wire); // " + enumName)
		default:
			m := wireScalarWriter(field.Kind())
			if m == "" {
				// 不支持的字段：仅在字段实际出现时抛错，缺位时其余字段仍可编码。
				return guard("throw UnsupportedError('dart-http wire: field " + string(field.FullName()) + "');")
			}
			return guard("w." + m + "(" + fmt.Sprint(n) + ", f" + fmt.Sprint(n) + ");")
		}
	}
}

// wireReadCaseStmts 返回 _readFrom 内单个字段的 switch case 语句块
// （含 case 头与大括号）。
func wireReadCaseStmts(pkg protoreflect.FullName, field protoreflect.FieldDescriptor) []string {
	fname := dartFieldName(field.JSONName())
	n := int(field.Number())
	head := "case " + fmt.Sprint(n) + ": {"

	var body []string

	switch {
	case field.IsMap():
		body = wireMapReadBody(pkg, field, fname)

	case field.IsList():
		if fieldCategoryOf(field) == categoryScalar && field.Kind() == protoreflect.MessageKind {
			wkt := mustWellKnown(field.Message())
			if !wkt.isWireSupported() {
				body = []string{wkt.wireUnsupportedStmt()}
				break
			}
			body = []string{"m." + fname + " = r.readNestedList().map((er) => " +
				wkt.wireReadExpr(pkg, "er") + ").toList();"}
		}
		if body == nil {
			switch field.Kind() {
			case protoreflect.MessageKind:
				typ := namedTypeFromField(pkg, field).Name
				body = []string{"m." + fname + " = r.readNestedList().map(" + typ + "._readFrom).toList();"}
			case protoreflect.StringKind:
				body = []string{"m." + fname + " = r.readStringList();"}
			case protoreflect.BytesKind:
				body = []string{"m." + fname + " = r.readBytesListText();"}
			case protoreflect.EnumKind:
				// WKT 枚举（NullValue/DayOfWeek/Month）映射为 String，无 wire 枚举可读。
				if wkt, ok := WellKnownType(field.Enum()); ok {
					body = []string{wkt.wireUnsupportedStmt()}
					break
				}
				enumName := namedTypeFromField(pkg, field).Name
				body = []string{"m." + fname + " = r.readVarintList().map(" + enumName + ".fromWire).toList();"}
			default:
				m := wirePackedReader(field.Kind())
				if m == "" {
					body = []string{"r.skip();"}
				} else {
					body = []string{"m." + fname + " = r." + m + ";"}
				}
			}
		}

	default:
		switch field.Kind() {
		case protoreflect.MessageKind:
			if IsWellKnownType(field.Message()) {
				wkt := mustWellKnown(field.Message())
				if !wkt.isWireSupported() {
					body = []string{wkt.wireUnsupportedStmt()}
					break
				}
				body = []string{"m." + fname + " = " + wkt.wireReadExpr(pkg, "r") + ";"}
			} else {
				typ := namedTypeFromField(pkg, field).Name
				body = []string{"m." + fname + " = " + typ + "._readFrom(r.readNested());"}
			}
		case protoreflect.EnumKind:
			// WKT 枚举（NullValue/DayOfWeek/Month）映射为 String，无 wire 枚举可读。
			if wkt, ok := WellKnownType(field.Enum()); ok {
				body = []string{wkt.wireUnsupportedStmt()}
				break
			}
			enumName := namedTypeFromField(pkg, field).Name
			body = []string{"m." + fname + " = " + enumName + ".fromWire(r.readEnum());"}
		default:
			m := wireScalarReader(field.Kind())
			if m == "" {
				body = []string{"r.skip();"}
			} else {
				body = []string{"m." + fname + " = r." + m + ";"}
			}
		}
	}

	out := []string{head}
	for _, line := range body {
		out = append(out, "  "+line)
	}
	// 纯 throw 的 case 不再补 break（throw 之后的 break 永远不可达，
	// 会被分析器标为 dead code）。
	if !(len(body) == 1 && strings.HasPrefix(body[0], "throw ")) {
		out = append(out, "  break;")
	}
	out = append(out, "}")
	return out
}

// wireMapKeyWriteExpr 把 Dart 侧 Map<String,…> 的 String 键还原成 wire 写入所需字面量。
// Dart 侧 map 键统一归一为 String（见 Type.Reference），按 proto 键型转回。
func wireMapKeyWriteExpr(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.StringKind:
		return "k"
	case protoreflect.BoolKind:
		return "(k == 'true')"
	default: // proto map 键只能是整型/布尔/字符串
		return "int.parse(k)"
	}
}

// wireMapWriteStmts map 字段：线上是 repeated entry{1:key, 2:value}。
func wireMapWriteStmts(pkg protoreflect.FullName, field protoreflect.FieldDescriptor) []string {
	fname := dartFieldName(field.JSONName())
	n := int(field.Number())
	keyMethod := wireScalarWriter(field.MapKey().Kind())
	if keyMethod == "" {
		return wireGuard(n, fname, "throw UnsupportedError('dart-http wire: map key "+string(field.FullName())+"');")
	}

	var setValue string
	switch fieldCategoryOf(field.MapValue()) {
	case categoryMessage:
		if IsWellKnownType(field.MapValue().Message()) {
			wkt := mustWellKnown(field.MapValue().Message())
			setValue = wkt.wireWriteCall(pkg, 2, "v")
		} else {
			setValue = "final vw = ProtoWireWriter(); v._writeTo(vw); ew.writeRaw(2, vw.toBuffer());"
		}
	case categoryEnum:
		setValue = "ew.writeEnum(2, v.wire);"
	default:
		vm := wireScalarWriter(field.MapValue().Kind())
		if vm == "" {
			return wireGuard(n, fname, "throw UnsupportedError('dart-http wire: map value "+string(field.FullName())+"');")
		}
		setValue = "ew." + vm + "(2, v);"
	}

	return []string{
		"final f" + fmt.Sprint(n) + " = " + fname + ";",
		"if (f" + fmt.Sprint(n) + " != null) {",
		"  f" + fmt.Sprint(n) + ".forEach((k, v) {",
		"    final ew = ProtoWireWriter();",
		"    ew." + keyMethod + "(1, " + wireMapKeyWriteExpr(field.MapKey().Kind()) + ");",
		"    " + setValue,
		"    w.writeRaw(" + fmt.Sprint(n) + ", ew.toBuffer());",
		"  });",
		"}",
	}
}

// wireMapReadBody map 字段的解码循环体（不含 case 头尾）。
func wireMapReadBody(pkg protoreflect.FullName, field protoreflect.FieldDescriptor, fname string) []string {
	valType := namedTypeFromField(pkg, field.MapValue())

	keyRead := wireScalarReader(field.MapKey().Kind())
	if keyRead == "" {
		return []string{"r.skip();"}
	}

	// Dart 侧 map 键统一为 String（见 Type.Reference）：非字符串键读出后转 String。
	keyReadExpr := "er." + keyRead
	if field.MapKey().Kind() != protoreflect.StringKind {
		keyReadExpr += ".toString()"
	}

	var valDecl string
	var valRead string
	switch fieldCategoryOf(field.MapValue()) {
	case categoryMessage:
		if IsWellKnownType(field.MapValue().Message()) {
			wkt := mustWellKnown(field.MapValue().Message())
			valDecl = valType.Name + "?"
			valRead = "v = " + wkt.wireReadExpr(pkg, "er") + ";"
		} else {
			valDecl = valType.Name + "?"
			valRead = "v = " + valType.Name + "._readFrom(er.readNested());"
		}
	case categoryEnum:
		valDecl = valType.Name + "?"
		valRead = "v = " + valType.Name + ".fromWire(er.readEnum());"
	default:
		vm := wireScalarReader(field.MapValue().Kind())
		if vm == "" {
			return []string{"r.skip();"}
		}
		valDecl = valType.Name + "?"
		valRead = "v = er." + vm + ";"
	}

	return []string{
		"final er = r.readNested();",
		"String? k;",
		valDecl + " v;",
		"while (er.next()) {",
		"  switch (er.fieldNumber) {",
		"    case 1: {",
		"      k = " + keyReadExpr + ";",
		"      break;",
		"    }",
		"    case 2: {",
		"      " + valRead,
		"      break;",
		"    }",
		"    default: er.skip();",
		"  }",
		"}",
		"if (k != null && v != null) {",
		"  (m." + fname + " ??= {})[k] = v;",
		"}",
	}
}

// wireScalarWriter 标量字段（含 bytes 的 base64 语义）写出方法名。
func wireScalarWriter(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.BoolKind:
		return "writeBool"
	case protoreflect.Int32Kind:
		return "writeInt32"
	case protoreflect.Uint32Kind:
		return "writeUint32"
	case protoreflect.Int64Kind:
		return "writeInt64"
	case protoreflect.Uint64Kind:
		return "writeUint64"
	case protoreflect.Sint32Kind:
		return "writeSint32"
	case protoreflect.Sint64Kind:
		return "writeSint64"
	case protoreflect.Fixed32Kind:
		return "writeFixed32"
	case protoreflect.Sfixed32Kind:
		return "writeSfixed32"
	case protoreflect.Fixed64Kind:
		return "writeFixed64"
	case protoreflect.Sfixed64Kind:
		return "writeSfixed64"
	case protoreflect.FloatKind:
		return "writeFloat"
	case protoreflect.DoubleKind:
		return "writeDouble"
	case protoreflect.StringKind:
		return "writeString"
	default:
		return ""
	}
}

// wireScalarReader 标量字段读取方法名。
func wireScalarReader(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.BoolKind:
		return "readBool()"
	case protoreflect.Int32Kind:
		return "readInt32()"
	case protoreflect.Uint32Kind:
		return "readUint32()"
	case protoreflect.Int64Kind:
		return "readInt64()"
	case protoreflect.Uint64Kind:
		return "readUint64()"
	case protoreflect.Sint32Kind:
		return "readSint32()"
	case protoreflect.Sint64Kind:
		return "readSint64()"
	case protoreflect.Fixed32Kind:
		return "readFixed32()"
	case protoreflect.Sfixed32Kind:
		return "readSfixed32()"
	case protoreflect.Fixed64Kind:
		return "readFixed64()"
	case protoreflect.Sfixed64Kind:
		return "readSfixed64()"
	case protoreflect.FloatKind:
		return "readFloat()"
	case protoreflect.DoubleKind:
		return "readDouble()"
	case protoreflect.StringKind:
		return "readString()"
	default:
		return ""
	}
}

// wirePackedWriter packed repeated 标量的写出方法名。
func wirePackedWriter(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.BoolKind:
		return "writePackedBoolList"
	case protoreflect.Int32Kind, protoreflect.Uint32Kind:
		return "writePackedVarintList"
	case protoreflect.Int64Kind, protoreflect.Uint64Kind:
		return "writePackedVarintList"
	case protoreflect.Sint32Kind:
		return "writePackedSint32List"
	case protoreflect.Sint64Kind:
		return "writePackedSint64List"
	case protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind:
		return "writePackedFixed32List"
	case protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
		return "writePackedFixed64List"
	case protoreflect.FloatKind:
		return "writePackedFloatList"
	case protoreflect.DoubleKind:
		return "writePackedDoubleList"
	default:
		return ""
	}
}

// wirePackedReader packed repeated 标量的读取表达式（运行时兼容 packed/单值）。
func wirePackedReader(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.BoolKind:
		return "readBoolList()"
	case protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.EnumKind:
		return "readVarintList()"
	case protoreflect.Int64Kind, protoreflect.Uint64Kind:
		return "readVarintList()"
	case protoreflect.Sint32Kind:
		return "readZigzag32List()"
	case protoreflect.Sint64Kind:
		return "readZigzag64List()"
	case protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind:
		return "readFixed32List()"
	case protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
		return "readFixed64List()"
	case protoreflect.FloatKind:
		return "readFloatList()"
	case protoreflect.DoubleKind:
		return "readDoubleList()"
	default:
		return ""
	}
}

// mustWellKnown 已知映射的 WKT 描述符（调用方保证 IsWellKnownType）。
func mustWellKnown(msg protoreflect.MessageDescriptor) WellKnown {
	wkt, ok := WellKnownType(msg)
	if !ok {
		panic("not a well-known type: " + string(msg.FullName()))
	}
	return wkt
}

// wireWriteCall WKT 标量字段的写出语句（表达式形式，含分号由调用方控制）。
func (wkt WellKnown) wireWriteCall(pkg protoreflect.FullName, fieldNumber int, dartExpr string) string {
	n := fmt.Sprint(fieldNumber)
	switch wkt {
	case WellKnownTimestamp:
		return "w.writeTimestamp(" + n + ", " + dartExpr + ");"
	case WellKnownDuration:
		return "w.writeDuration(" + n + ", " + dartExpr + ");"
	case WellKnownFieldMask:
		return "w.writeFieldMask(" + n + ", " + dartExpr + ");"
	case WellKnownEmpty:
		return "w.writeEmpty(" + n + ");"
	case WellKnownInt32Value:
		return "w.writeInt32Value(" + n + ", " + dartExpr + ");"
	case WellKnownInt64Value:
		return "w.writeInt64Value(" + n + ", " + dartExpr + ");"
	case WellKnownUInt32Value:
		return "w.writeUint32Value(" + n + ", " + dartExpr + ");"
	case WellKnownUInt64Value:
		return "w.writeUint64Value(" + n + ", " + dartExpr + ");"
	case WellKnownBoolValue:
		return "w.writeBoolValue(" + n + ", " + dartExpr + ");"
	case WellKnownStringValue:
		return "w.writeStringValue(" + n + ", " + dartExpr + ");"
	case WellKnownBytesValue:
		return "w.writeBytesValue(" + n + ", " + dartExpr + ");"
	case WellKnownFloatValue:
		return "w.writeFloatValue(" + n + ", " + dartExpr + ");"
	case WellKnownDoubleValue:
		return "w.writeDoubleValue(" + n + ", " + dartExpr + ");"
	default:
		return wkt.wireUnsupportedStmt()
	}
}

// wireReadExpr WKT 标量字段的读取表达式（基于读取器变量 readerVar）。
func (wkt WellKnown) wireReadExpr(pkg protoreflect.FullName, readerVar string) string {
	switch wkt {
	case WellKnownTimestamp:
		return readerVar + ".readTimestampIso()"
	case WellKnownDuration:
		return readerVar + ".readDurationText()"
	case WellKnownFieldMask:
		return readerVar + ".readFieldMaskText()"
	case WellKnownEmpty:
		return readerVar + ".readEmpty()"
	case WellKnownInt32Value:
		return readerVar + ".readInt32Value()"
	case WellKnownInt64Value:
		return readerVar + ".readInt64Value()"
	case WellKnownUInt32Value:
		return readerVar + ".readUint32Value()"
	case WellKnownUInt64Value:
		return readerVar + ".readUint64Value()"
	case WellKnownBoolValue:
		return readerVar + ".readBoolValue()"
	case WellKnownStringValue:
		return readerVar + ".readStringValue()"
	case WellKnownBytesValue:
		return readerVar + ".readBytesValueText()"
	case WellKnownFloatValue:
		return readerVar + ".readFloatValue()"
	case WellKnownDoubleValue:
		return readerVar + ".readDoubleValue()"
	default:
		return "(throw UnsupportedError('dart-http wire: " + string(wkt) + " not supported'))"
	}
}

// wireUnsupportedHint 提示字符串（未支持的 WKT 组）。
func wireUnsupportedHint() string {
	return strings.Join([]string{"Any/Struct/Value/ListValue/google.type.*"}, ", ")
}
