package main

// ProtoFileData defines custom data type for Proto File info needed in template
type ProtoFileData struct {
	Source  string
	Package string
	// Imports: alias -> import-path
	Imports    map[string]string
	References []string
	Services   []*ServiceData
	Messages   []*MessageData
	// RegexDeclarations holds all precompiled regex variable declarations
	// needed by the generated code. Each entry is unique by VarName.
	RegexDeclarations []RegexDecl

	// --- Helper function flags ---
	NeedMaskHelper        bool // _redactMask
	NeedEmailHelper       bool // _redactEmail
	NeedTruncateHelper    bool // _redactTruncate
	NeedHashMD5           bool // _redactHashMD5
	NeedHashSHA1          bool // _redactHashSHA1
	NeedHashSHA256        bool // _redactHashSHA256
	NeedUUIDHelper        bool // _redactUUID
	NeedIPHelper          bool // _redactIP
	NeedURLHelper         bool // _redactURL
	NeedFixedLengthHelper bool // _redactFixedLength
	NeedConditionHelper   bool // _redactCondCheck
	NeedCustomHelper      bool // _redactCustom (wraps redact.ApplyCustomRedactor)
}

// RegexDecl represents a package-level precompiled regex variable
// to be declared in the generated file.
type RegexDecl struct {
	// VarName is the unique Go variable name, e.g. "_redactRegex_User_Phone"
	VarName string
	// PatternLiteral is the Go string literal for the regex pattern,
	// already escaped via strconv.Quote, e.g. `"^(\\d{3})\\d{4}(\\d{4})$"`
	PatternLiteral string
}

// ServiceData defines custom data type for Service info needed in template
type ServiceData struct {
	Name    string
	Skip    bool
	Methods []*MethodData
}

// MethodData defines custom data type for Method info needed in template
type MethodData struct {
	Name            string
	Skip            bool
	Input           string
	Output          *MessageData // will only contain name and options (ignore, nil, empty)
	Internal        bool
	StatusCode      string
	ErrMessage      string
	ClientStreaming bool // true if client sends a stream of requests
	ServerStreaming bool // true if server sends a stream of responses
}

// MessageData defines custom data type for Message info needed in template
type MessageData struct {
	Name      string
	WithAlias string

	Fields  []*FieldData
	Oneofs  []*OneofData
	Ignore  bool
	ToNil   bool
	ToEmpty bool
}

// OneofData defines custom data type for a protobuf oneof group
type OneofData struct {
	Name   string            // Go name of the oneof field in the parent struct
	Fields []*OneofFieldData // Fields within this oneof
}

// HasRedactableFields returns true if at least one field in the oneof has redaction enabled
func (o *OneofData) HasRedactableFields() bool {
	for _, f := range o.Fields {
		if f.Redact {
			return true
		}
	}
	return false
}

// OneofFieldData wraps FieldData with oneof-specific information
type OneofFieldData struct {
	*FieldData
	WrapperTypeName string // Go wrapper type name (e.g., "MessageName_FieldName")
}

// FieldData defines custom data type for Field info needed in template
type FieldData struct {
	Name string
	// Redact using RedactionValue
	Redact         bool
	RedactionValue string
	FieldGoType    string // Go type for the field (e.g., "int32", "string", "bool")

	IsMap      bool // IsMap: true for Map types
	IsRepeated bool // IsRepeated: true for Repeated types
	IsMessage  bool // IsMessage: true for Message type(& not Repeated/Map)
	IsOptional bool // IsOptional: true for optional types

	// Iterate will only be used for Repeated/Map types and it specifies
	// whether or not to iterate each entry to be redacted
	Iterate bool

	// NestedEmbedCall will only be used for Message Types and it specifies
	// whether or not the embed message should be called for redaction.
	NestedEmbedCall bool

	// EmbedSkip will only be used for Message Types and it specifies
	// whether or not the embed message should be skipped.
	EmbedSkip bool

	// EmbedMessageName: name of embed message which is in case of Repeated or
	// Map or Message type field
	EmbedMessageName          string
	EmbedMessageNameWithAlias string

	// --- Regex-based masking fields ---

	// IsRegex: true when this field uses regex-based masking
	IsRegex bool
	// RegexVarName: the package-level precompiled regexp variable name
	RegexVarName string
	// RegexPatternLiteral: the Go string literal for the pattern,
	// already escaped via strconv.Quote
	RegexPatternLiteral string
	// RegexReplacement: the Go string literal for the replacement,
	// already escaped via strconv.Quote
	RegexReplacement string

	// --- Mask-based redaction fields ---
	IsMask        bool
	MaskKeepFirst uint32
	MaskKeepLast  uint32
	MaskChar      string // already quoted via strconv.Quote

	// --- Email-based redaction fields ---
	IsEmail             bool
	EmailKeepLocalFirst uint32
	EmailMaskDomain     bool
	EmailMaskChar       string // already quoted via strconv.Quote

	// --- Truncate-based redaction fields ---
	IsTruncate     bool
	TruncateLength uint32
	TruncateSuffix string // already quoted via strconv.Quote

	// --- Hash-based redaction fields ---
	IsHash   bool
	HashAlgo string // "md5", "sha1", "sha256"
	// HashFuncName is the generated helper function name, e.g. "_redactHashMD5"
	HashFuncName string

	// --- UUID-based redaction fields ---
	IsUUID bool

	// --- IP-based redaction fields ---
	IsIP         bool
	IPKeepOctets uint32
	IPMaskChar   string // already quoted via strconv.Quote

	// --- URL-based redaction fields ---
	IsURL        bool
	URLMaskQuery bool
	URLMaskChar  string // already quoted via strconv.Quote

	// --- FixedLength-based redaction fields ---
	IsFixedLength   bool
	FixedLengthChar string // already quoted via strconv.Quote

	// --- Custom redaction fields ---
	IsCustom       bool
	CustomFuncName string // registered redactor name, already quoted

	// --- Condition-based redaction fields ---
	IsCondition bool
	// CondEnvVar is the Go string literal for the env var name
	CondEnvVar string // already quoted via strconv.Quote
	// CondEnvVal is the Go string literal for the expected env var value
	CondEnvVal string // already quoted via strconv.Quote
}
