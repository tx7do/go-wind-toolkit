package main

import (
	"fmt"
	"strconv"

	pgs "github.com/lyft/protoc-gen-star/v2"

	"github.com/menta2k/protoc-gen-redact/v3/redact/v3"
)

// processFields extracts each fields information. ownerName is the enclosing
// message Go name (used to build unique regex variable names).
func (m *Module) processFields(
	field pgs.Field,
	nameWithAlias func(n pgs.Entity) string,
	ownerName string,
) *FieldData {
	// Validate field before processing
	if err := m.validateField(field); err != nil {
		m.Failf("Invalid field: %v", err)
		return nil
	}

	typ := field.Type()
	if typ == nil {
		m.failWithContext(field, "field has nil type")
		return nil
	}

	// Determine if field will be a pointer in generated Go code
	// In proto3, fields with explicit `optional` keyword become pointers
	// These fields are implemented as synthetic oneofs (proto3_optional)
	// Exception: bytes fields are always []byte, never *[]byte, even with explicit optional
	hasExplicitOptional := field.InOneOf() && field.OneOf().IsSynthetic()
	isOptional := hasExplicitOptional && typ.ProtoType() != pgs.BytesT

	flData := &FieldData{
		Name:        m.ctx.Name(field).String(),
		IsMap:       typ.IsMap(),
		IsRepeated:  typ.IsRepeated(),
		IsMessage:   typ.IsEmbed(),
		IsOptional:  isOptional,
		FieldGoType: goTypeName(typ.ProtoType()),
	}
	em := typ.Embed()
	if em == nil {
		if ele := typ.Element(); ele != nil {
			em = ele.Embed()
		}
	}
	// embed message
	if em != nil {
		flData.EmbedMessageName = m.ctx.Name(em).String()
		flData.EmbedMessageNameWithAlias = nameWithAlias(em)
	}

	_redact, fieldRules := false, &redact.FieldRules{}
	// ok := m.must(field.Extension(redact.E_Redact, &_redact))
	ok := m.must(field.Extension(redact.E_Value, &fieldRules))

	// safe field: no option is defined
	if !ok {
		return flData
	}

	// Validate rules before processing
	if err := m.validateRules(fieldRules, field); err != nil {
		m.Fail(err)
		return flData
	}

	// check for custom field rules
	if fieldRules == nil || fieldRules.Values == nil {
		// no field rules
		if !_redact {
			// and redaction is also denied
			return flData
		}
		// default rules will be used
		flData.Redact = true
		flData.RedactionValue = RedactionDefaults(
			typ.ProtoType(),
			typ.IsRepeated() || typ.IsMap(),
		)
		if typ.IsEmbed() {
			flData.NestedEmbedCall = true
		}
		return flData
	}

	// custom field rules are defined, hence prefill defaults
	flData.Redact = true
	flData.RedactionValue = RedactionDefaults(
		typ.ProtoType(),
		typ.IsRepeated() || typ.IsMap(),
	)
	// custom values
	m.redactedCustomValue(flData, field, fieldRules)

	// If this field uses regex masking, build a unique variable name
	// and extract the pattern literal for later declaration.
	if flData.IsRegex {
		var pattern string
		if rr := getRegexRules(field.Type(), fieldRules); rr != nil {
			pattern = rr.GetPattern()
		}
		flData.RegexVarName = regexVarName(ownerName, flData.Name)
		flData.RegexPatternLiteral = strconv.Quote(pattern)
	}

	return flData
}

// getRegexRules extracts *redact.RegexRules from either a direct FieldRules_Regex
// or an ElementRules.Item that contains FieldRules_Regex.
func getRegexRules(typ pgs.FieldType, fr *redact.FieldRules) *redact.RegexRules {
	if fr == nil {
		return nil
	}
	if rr, ok := fr.Values.(*redact.FieldRules_Regex); ok {
		return rr.Regex
	}
	if er, ok := fr.Values.(*redact.FieldRules_Element); ok && er.Element != nil {
		if item := er.Element.Item; item != nil {
			if rr, ok := item.Values.(*redact.FieldRules_Regex); ok {
				return rr.Regex
			}
		}
	}
	return nil
}

// regexVarName builds a unique Go variable name for a precompiled regex.
func regexVarName(owner, field string) string {
	return "_redactRegex_" + owner + "_" + field
}

// hashFuncName maps an algorithm name to the generated helper function name.
func hashFuncName(algo string) string {
	switch algo {
	case "md5":
		return "_redactHashMD5"
	case "sha1":
		return "_redactHashSHA1"
	case "sha256":
		return "_redactHashSHA256"
	}
	return ""
}

func (m *Module) redactedCustomValue(
	flData *FieldData,
	field pgs.Field,
	fieldRules *redact.FieldRules,
) {
	// Validate inputs
	if flData == nil {
		m.Failf("Internal error: nil FieldData for field %s", field.FullyQualifiedName())
		return
	}
	if fieldRules == nil {
		m.Failf("Internal error: nil fieldRules for field %s", field.FullyQualifiedName())
		return
	}

	typ := field.Type()
	if typ == nil {
		m.failWithContext(field, "field type is nil")
		return
	}

	// extract rule information
	info := m.RuleInformation(fieldRules)

	// match field types & rule types with better error message
	if info.ProtoType != 0 && info.ProtoType != typ.ProtoType() {
		err := m.validateTypeMatch(field, info.ProtoType, info.ProtoLabel)
		if err != nil {
			m.Fail(err)
		} else {
			m.failWithInvalidType(field)
		}
		return // unreachable
	}
	if typ.ProtoLabel() == pgs.Repeated && info.ProtoLabel != pgs.Repeated {
		err := m.validateTypeMatch(field, info.ProtoType, info.ProtoLabel)
		if err != nil {
			m.Fail(err)
		} else {
			m.failWithInvalidType(field)
		}
		return // unreachable
	}
	if info.IsRegex {
		// Regex-based masking for string fields
		flData.IsRegex = true
		rr, ok := fieldRules.Values.(*redact.FieldRules_Regex)
		if !ok || rr.Regex == nil {
			m.Failf("Invalid regex rule for field %s", field.Name())
			return
		}
		flData.RegexReplacement = strconv.Quote(rr.Regex.GetReplacement())
		return
	}
	if info.IsMask {
		// Position-based masking for string fields
		flData.IsMask = true
		mr, ok := fieldRules.Values.(*redact.FieldRules_Mask)
		if !ok || mr.Mask == nil {
			m.Failf("Invalid mask rule for field %s", field.Name())
			return
		}
		mc := mr.Mask.GetMaskChar()
		if mc == "" {
			mc = "*"
		}
		flData.MaskKeepFirst = mr.Mask.GetKeepFirst()
		flData.MaskKeepLast = mr.Mask.GetKeepLast()
		flData.MaskChar = strconv.Quote(mc)
		return
	}
	if info.IsEmail {
		// Email-specific masking for string fields
		flData.IsEmail = true
		er, ok := fieldRules.Values.(*redact.FieldRules_Email)
		if !ok || er.Email == nil {
			m.Failf("Invalid email rule for field %s", field.Name())
			return
		}
		mc := er.Email.GetMaskChar()
		if mc == "" {
			mc = "*"
		}
		flData.EmailKeepLocalFirst = er.Email.GetKeepLocalFirst()
		flData.EmailMaskDomain = er.Email.GetMaskDomain()
		flData.EmailMaskChar = strconv.Quote(mc)
		return
	}
	if info.IsTruncate {
		// Truncation for string fields
		flData.IsTruncate = true
		tr, ok := fieldRules.Values.(*redact.FieldRules_Truncate)
		if !ok || tr.Truncate == nil {
			m.Failf("Invalid truncate rule for field %s", field.Name())
			return
		}
		suffix := tr.Truncate.GetSuffix()
		if suffix == "" {
			suffix = "..."
		}
		flData.TruncateLength = tr.Truncate.GetLength()
		flData.TruncateSuffix = strconv.Quote(suffix)
		return
	}
	if info.IsHash {
		// One-way hashing for string fields
		flData.IsHash = true
		flData.HashAlgo = info.HashAlgo
		flData.HashFuncName = hashFuncName(info.HashAlgo)
		return
	}

	if info.ProtoType != pgs.MessageT && info.ProtoLabel != pgs.Repeated {
		// simple type fields
		flData.RedactionValue = fmt.Sprintf("%v", info.RedactionValue)
		return
	}

	// if message type
	if info.ProtoType == pgs.MessageT {
		messageRule, ok := fieldRules.Values.(*redact.FieldRules_Message)
		if !ok {
			m.Failf("Invalid message rule type for field %s", field.Name())
		}
		rule := messageRule.Message
		// default value is nil
		flData.RedactionValue = `nil`
		if rule.Empty {
			// flData.RedactionValue = m.ctx.Type(field).String() + "{}"
			flData.RedactionValue = fmt.Sprintf("&%s{}", flData.EmbedMessageNameWithAlias)
			return
		}
		if rule.Nil {
			flData.RedactionValue = "nil"
			return
		}
		if rule.Skip {
			flData.EmbedSkip = true
			return
		}
		flData.NestedEmbedCall = true
		return
	}

	// else info.ProtoLabel == pgs.Repeated
	elementRule, ok := fieldRules.Values.(*redact.FieldRules_Element)
	if !ok {
		m.Failf("Invalid element rule type for field %s", field.Name())
	}
	rule := elementRule.Element
	if rule.Empty {
		if flData.EmbedMessageNameWithAlias == "" {
			flData.RedactionValue = m.ctx.Type(field).String() + "{}"
			return
		}
		if flData.IsRepeated {
			flData.RedactionValue = fmt.Sprintf("[]*%s{}", flData.EmbedMessageNameWithAlias)
			return
		}
		// map type
		key := m.ctx.Type(field).Key().String()
		flData.RedactionValue = fmt.Sprintf("map[%s]*%s{}", key, flData.EmbedMessageNameWithAlias)
		return
	}
	if rule.Nested {
		// iterate over all items and redact with defaults
		flData.Iterate = true
		flData.RedactionValue = RedactionDefaults(typ.Element().ProtoType(), false)
		if typ.Element().IsEmbed() {
			flData.NestedEmbedCall = true
		}
		return
	}
	if rules := rule.Item; rules != nil && rules.Values != nil {
		if rules.GetElement() != nil {
			// Use the improved error message
			m.failWithNestedError(field)
			return
		}
		if _, ok := rules.Values.(*redact.FieldRules_Regex); ok {
			// Regex-based masking for each element
			flData.IsRegex = true
			flData.Iterate = true
			rr := rules.GetRegex()
			if rr == nil {
				m.Failf("Invalid regex rule for field %s", field.Name())
				return
			}
			flData.RegexReplacement = strconv.Quote(rr.GetReplacement())
			return
		}
		if _, ok := rules.Values.(*redact.FieldRules_Mask); ok {
			// Position-based masking for each element
			flData.IsMask = true
			flData.Iterate = true
			mr := rules.GetMask()
			if mr == nil {
				m.Failf("Invalid mask rule for field %s", field.Name())
				return
			}
			mc := mr.GetMaskChar()
			if mc == "" {
				mc = "*"
			}
			flData.MaskKeepFirst = mr.GetKeepFirst()
			flData.MaskKeepLast = mr.GetKeepLast()
			flData.MaskChar = strconv.Quote(mc)
			return
		}
		if _, ok := rules.Values.(*redact.FieldRules_Email); ok {
			// Email-specific masking for each element
			flData.IsEmail = true
			flData.Iterate = true
			er := rules.GetEmail()
			if er == nil {
				m.Failf("Invalid email rule for field %s", field.Name())
				return
			}
			mc := er.GetMaskChar()
			if mc == "" {
				mc = "*"
			}
			flData.EmailKeepLocalFirst = er.GetKeepLocalFirst()
			flData.EmailMaskDomain = er.GetMaskDomain()
			flData.EmailMaskChar = strconv.Quote(mc)
			return
		}
		if _, ok := rules.Values.(*redact.FieldRules_Truncate); ok {
			// Truncation for each element
			flData.IsTruncate = true
			flData.Iterate = true
			tr := rules.GetTruncate()
			if tr == nil {
				m.Failf("Invalid truncate rule for field %s", field.Name())
				return
			}
			suffix := tr.GetSuffix()
			if suffix == "" {
				suffix = "..."
			}
			flData.TruncateLength = tr.GetLength()
			flData.TruncateSuffix = strconv.Quote(suffix)
			return
		}
		if _, ok := rules.Values.(*redact.FieldRules_Hash); ok {
			// One-way hashing for each element
			flData.IsHash = true
			flData.Iterate = true
			hr := rules.GetHash()
			if hr == nil {
				m.Failf("Invalid hash rule for field %s", field.Name())
				return
			}
			switch hr.GetAlgo() {
			case redact.HashAlgo_MD5:
				flData.HashAlgo = "md5"
			case redact.HashAlgo_SHA1:
				flData.HashAlgo = "sha1"
			case redact.HashAlgo_SHA256:
				flData.HashAlgo = "sha256"
			}
			flData.HashFuncName = hashFuncName(flData.HashAlgo)
			return
		}
		info := m.RuleInformation(rules)
		// match types
		if info.ProtoType != typ.Element().ProtoType() {
			m.failWithInvalidType(field)
			return // unreachable
		}
		// default value is nil
		flData.Iterate = true
		flData.RedactionValue = "nil"
		if info.ProtoType != pgs.MessageT {
			// simple type fields
			flData.RedactionValue = fmt.Sprintf("%v", info.RedactionValue)
		} else {
			// message type embedded field
			messageRule, ok := rules.Values.(*redact.FieldRules_Message)
			if !ok {
				m.Failf("Invalid message rule type for field %s", field.Name())
			}
			rule := messageRule.Message
			flData.RedactionValue = `nil`
			if rule.Empty {
				// flData.RedactionValue = m.ctx.Type(field).String() + "{}"
				flData.RedactionValue = fmt.Sprintf("&%s{}", flData.EmbedMessageNameWithAlias)
				return
			}
			if rule.Nil {
				flData.RedactionValue = "nil"
				return
			}
			if rule.Skip {
				flData.EmbedSkip = true
				return
			}
			flData.NestedEmbedCall = true
		}
	}
}

// RuleInfo response type for Module.RuleInformation
type RuleInfo struct {
	RedactionValue interface{}
	// equivalent field type information
	ProtoType  pgs.ProtoType
	ProtoLabel pgs.ProtoLabel
	// IsRegex indicates the rule is regex-based masking
	IsRegex bool
	// IsMask indicates the rule is position-based masking
	IsMask bool
	// IsEmail indicates the rule is email-specific masking
	IsEmail bool
	// IsTruncate indicates the rule is truncation
	IsTruncate bool
	// IsHash indicates the rule is one-way hashing
	IsHash bool
	// HashAlgo stores the hash algorithm name ("md5", "sha1", "sha256")
	HashAlgo string
}

// RuleInformation returns required information from the redact.FieldRules
func (m *Module) RuleInformation(rules *redact.FieldRules) (res RuleInfo) {
	// custom rules validation and values
	switch rule := rules.Values.(type) {
	case *redact.FieldRules_Float:
		res.ProtoType = pgs.FloatT
		res.RedactionValue = rule.Float
	case *redact.FieldRules_Double:
		res.ProtoType = pgs.DoubleT
		res.RedactionValue = rule.Double
	case *redact.FieldRules_Int32:
		res.ProtoType = pgs.Int32T
		res.RedactionValue = rule.Int32
	case *redact.FieldRules_Int64:
		res.ProtoType = pgs.Int64T
		res.RedactionValue = rule.Int64
	case *redact.FieldRules_Uint32:
		res.ProtoType = pgs.UInt32T
		res.RedactionValue = rule.Uint32
	case *redact.FieldRules_Uint64:
		res.ProtoType = pgs.UInt64T
		res.RedactionValue = rule.Uint64
	case *redact.FieldRules_Sint32:
		res.ProtoType = pgs.SInt32
		res.RedactionValue = rule.Sint32
	case *redact.FieldRules_Sint64:
		res.ProtoType = pgs.SInt64
		res.RedactionValue = rule.Sint64
	case *redact.FieldRules_Fixed32:
		res.ProtoType = pgs.Fixed32T
		res.RedactionValue = rule.Fixed32
	case *redact.FieldRules_Fixed64:
		res.ProtoType = pgs.Fixed64T
		res.RedactionValue = rule.Fixed64
	case *redact.FieldRules_Sfixed32:
		res.ProtoType = pgs.SFixed32
		res.RedactionValue = rule.Sfixed32
	case *redact.FieldRules_Sfixed64:
		res.ProtoType = pgs.SFixed64
		res.RedactionValue = rule.Sfixed64
	case *redact.FieldRules_Bool:
		res.ProtoType = pgs.BoolT
		res.RedactionValue = rule.Bool
	case *redact.FieldRules_String_:
		res.ProtoType = pgs.StringT
		res.RedactionValue = fmt.Sprintf("`%v`", rule.String_)
	case *redact.FieldRules_Bytes:
		res.ProtoType = pgs.BytesT
		res.RedactionValue = fmt.Sprintf("[]byte(`%v`)", string(rule.Bytes))
	case *redact.FieldRules_Enum:
		res.ProtoType = pgs.EnumT
		res.RedactionValue = rule.Enum
	case *redact.FieldRules_Message:
		res.ProtoType = pgs.MessageT
		if rule == nil || rule.Message == nil {
			m.Fail("(redact.custom).message is nil, no option defined")
			return // unreachable
		}
	case *redact.FieldRules_Regex:
		res.ProtoType = pgs.StringT
		res.IsRegex = true
		if rule == nil || rule.Regex == nil {
			m.Fail("(redact.custom).regex is nil, no option defined")
			return // unreachable
		}
	case *redact.FieldRules_Mask:
		res.ProtoType = pgs.StringT
		res.IsMask = true
		if rule == nil || rule.Mask == nil {
			m.Fail("(redact.custom).mask is nil, no option defined")
			return // unreachable
		}
	case *redact.FieldRules_Hash:
		res.ProtoType = pgs.StringT
		res.IsHash = true
		if rule == nil || rule.Hash == nil {
			m.Fail("(redact.custom).hash is nil, no option defined")
			return // unreachable
		}
		switch rule.Hash.GetAlgo() {
		case redact.HashAlgo_MD5:
			res.HashAlgo = "md5"
		case redact.HashAlgo_SHA1:
			res.HashAlgo = "sha1"
		case redact.HashAlgo_SHA256:
			res.HashAlgo = "sha256"
		default:
			m.Failf("Unknown hash algorithm: %v", rule.Hash.GetAlgo())
			return // unreachable
		}
	case *redact.FieldRules_Email:
		res.ProtoType = pgs.StringT
		res.IsEmail = true
		if rule == nil || rule.Email == nil {
			m.Fail("(redact.custom).email is nil, no option defined")
			return // unreachable
		}
	case *redact.FieldRules_Truncate:
		res.ProtoType = pgs.StringT
		res.IsTruncate = true
		if rule == nil || rule.Truncate == nil {
			m.Fail("(redact.custom).truncate is nil, no option defined")
			return // unreachable
		}
	case *redact.FieldRules_Element:
		res.ProtoLabel = pgs.Repeated
		if rule == nil || rule.Element == nil {
			m.Fail("(redact.custom).element is nil, no option defined")
			return // unreachable
		}
	default:
		m.Fail("Something went wrong")
	}
	return res
}

// goTypeName returns the Go type name for a proto type
func goTypeName(pt pgs.ProtoType) string {
	switch pt {
	case pgs.Int32T:
		return "int32"
	case pgs.Int64T:
		return "int64"
	case pgs.UInt32T:
		return "uint32"
	case pgs.UInt64T:
		return "uint64"
	case pgs.SInt32:
		return "int32"
	case pgs.SInt64:
		return "int64"
	case pgs.Fixed32T:
		return "uint32"
	case pgs.Fixed64T:
		return "uint64"
	case pgs.SFixed32:
		return "int32"
	case pgs.SFixed64:
		return "int64"
	case pgs.FloatT:
		return "float32"
	case pgs.DoubleT:
		return "float64"
	case pgs.BoolT:
		return "bool"
	case pgs.StringT:
		return "string"
	case pgs.BytesT:
		return "[]byte"
	default:
		return ""
	}
}
