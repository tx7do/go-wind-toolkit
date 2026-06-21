package main

import (
	"fmt"
	"strconv"
	"strings"

	pgs "github.com/lyft/protoc-gen-star/v2"
	"github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/redact/v1"
	"google.golang.org/grpc/codes"
)

// hasRedactUsage returns true if the file (or any entity within it)
// uses at least one redact annotation. When false, no .pb.redact.go
// file is generated, keeping output clean for proto files that don't
// need redaction.
func (m *Module) hasRedactUsage(file pgs.File) bool {
	// Service-level & method-level annotations
	for _, srv := range file.Services() {
		if srv == nil {
			continue
		}
		var skipSrv bool
		if m.must(srv.Extension(redact.E_ServiceSkip, &skipSrv)) {
			return true
		}
		var internalSrv bool
		if m.must(srv.Extension(redact.E_InternalService, &internalSrv)) {
			return true
		}
		for _, meth := range srv.Methods() {
			if meth == nil {
				continue
			}
			var skipMeth bool
			if m.must(meth.Extension(redact.E_MethodSkip, &skipMeth)) {
				return true
			}
			var internalMeth bool
			if m.must(meth.Extension(redact.E_InternalMethod, &internalMeth)) {
				return true
			}
		}
	}

	// File-level auto_detect option
	var ad redact.AutoDetectRules
	if m.must(file.Extension(redact.E_AutoDetect, &ad)) {
		return true
	}

	// Message-level & field-level annotations
	for _, msg := range file.AllMessages() {
		if msg == nil {
			continue
		}
		var msgNil bool
		if m.must(msg.Extension(redact.E_Nil, &msgNil)) {
			return true
		}
		var msgEmpty bool
		if m.must(msg.Extension(redact.E_Empty, &msgEmpty)) {
			return true
		}
		var msgIgnored bool
		if m.must(msg.Extension(redact.E_Ignored, &msgIgnored)) {
			return true
		}
		for _, fld := range msg.Fields() {
			if fld == nil {
				continue
			}
			var fr redact.FieldRules
			if m.must(fld.Extension(redact.E_Value, &fr)) {
				return true
			}
		}
	}

	return false
}

// collectHelperFlags inspects a single FieldData and sets the corresponding
// helper-function flags on ProtoFileData so the template knows which helpers
// and imports to generate.
func collectHelperFlags(data *ProtoFileData, fld *FieldData) {
	if fld == nil {
		return
	}
	if fld.IsMask {
		data.NeedMaskHelper = true
	}
	if fld.IsEmail {
		data.NeedEmailHelper = true
	}
	if fld.IsTruncate {
		data.NeedTruncateHelper = true
	}
	if fld.IsHash {
		switch fld.HashAlgo {
		case "md5":
			data.NeedHashMD5 = true
		case "sha1":
			data.NeedHashSHA1 = true
		case "sha256":
			data.NeedHashSHA256 = true
		}
	}
	if fld.IsUUID {
		data.NeedUUIDHelper = true
	}
	if fld.IsIP {
		data.NeedIPHelper = true
	}
	if fld.IsURL {
		data.NeedURLHelper = true
	}
	if fld.IsFixedLength {
		data.NeedFixedLengthHelper = true
	}
	if fld.IsCondition {
		data.NeedConditionHelper = true
	}
	if fld.IsCustom {
		data.NeedCustomHelper = true
	}
}

// regexDeclCollector is used during message processing to collect
// all unique regex variable declarations needed by the generated file.

const (
	// defaultErrMsg: for the service method/rpc redaction
	defaultErrMsg = `Permission Denied. Method: "%service%.%method%" has been redacted`
	// error message format specifiers
	specifierMethod  = "%method%"
	specifierService = "%service%"
)

// Process processes the file and adds its generated code into Module.Artifacts
func (m *Module) Process(file pgs.File) {
	// Validate file before processing
	if err := m.validateFile(file); err != nil {
		m.Failf("Cannot process file: %v", err)
		return
	}

	// Add panic recovery for robustness
	defer m.recoverFromPanic(fmt.Sprintf("processing file %s", file.Name()))

	// check file option: FileSkip
	fileSkip := false
	m.must(file.Extension(redact.E_FileSkip, &fileSkip))
	if fileSkip {
		m.Debug(fmt.Sprintf("Skipping file %s due to file_skip option", file.Name()))
		return
	}

	// Skip generation entirely if the file does not use any redact annotations.
	// This avoids creating empty/pointless .pb.redact.go files.
	if !m.hasRedactUsage(file) {
		m.Debug(fmt.Sprintf("Skipping file %s: no redact annotations found", file.Name()))
		return
	}

	// imports and their aliases
	path2Alias, alias2Path := m.importPaths(file)
	nameWithAlias := func(n pgs.Entity) string {
		imp := m.ctx.ImportPath(n).String()
		name := m.ctx.Name(n).String()
		if alias := path2Alias[imp]; alias != "" {
			name = alias + "." + name
		}
		return name
	}

	data := &ProtoFileData{
		Source:     file.Name().String(),
		Package:    m.ctx.PackageName(file).String(),
		Imports:    alias2Path,
		References: m.references(file, nameWithAlias),
		Services:   make([]*ServiceData, 0, len(file.Services())),
		Messages:   make([]*MessageData, 0, len(file.AllMessages())),
	}

	// all services
	for _, srv := range file.Services() {
		data.Services = append(data.Services, m.processService(srv, nameWithAlias))
	}

	// all messages
	for _, msg := range file.AllMessages() {
		data.Messages = append(data.Messages, m.processMessage(msg, nameWithAlias, true))
	}

	// Apply auto-detect rules (file-level option)
	m.applyAutoDetect(file, data, nameWithAlias)

	// Collect regex variable declarations from all messages
	seen := map[string]bool{}
	for _, msg := range data.Messages {
		for _, fld := range msg.Fields {
			if fld.IsRegex && fld.RegexVarName != "" && !seen[fld.RegexVarName] {
				seen[fld.RegexVarName] = true
				data.RegexDeclarations = append(data.RegexDeclarations, RegexDecl{
					VarName:        fld.RegexVarName,
					PatternLiteral: fld.RegexPatternLiteral,
				})
			}
		}
		for _, oneof := range msg.Oneofs {
			for _, fld := range oneof.Fields {
				if fld.IsRegex && fld.RegexVarName != "" && !seen[fld.RegexVarName] {
					seen[fld.RegexVarName] = true
					data.RegexDeclarations = append(data.RegexDeclarations, RegexDecl{
						VarName:        fld.RegexVarName,
						PatternLiteral: fld.RegexPatternLiteral,
					})
				}
			}
		}
	}

	// Collect helper function flags and imports for mask/email/truncate/hash
	for _, msg := range data.Messages {
		for _, fld := range msg.Fields {
			collectHelperFlags(data, fld)
		}
		for _, oneof := range msg.Oneofs {
			for _, fld := range oneof.Fields {
				collectHelperFlags(data, fld.FieldData)
			}
		}
	}

	// Ensure imports map is initialized
	if data.Imports == nil {
		data.Imports = map[string]string{}
	}

	// Add regexp import if any regex declarations are needed
	if len(data.RegexDeclarations) > 0 {
		data.Imports["regexp"] = "regexp"
	}

	// Add imports based on helper flags
	if data.NeedMaskHelper || data.NeedEmailHelper || data.NeedTruncateHelper {
		data.Imports["strings"] = "strings"
	}
	if data.NeedHashMD5 || data.NeedHashSHA1 || data.NeedHashSHA256 {
		data.Imports["fmt"] = "fmt"
		if data.NeedHashMD5 {
			data.Imports["md5"] = "crypto/md5"
		}
		if data.NeedHashSHA1 {
			data.Imports["sha1"] = "crypto/sha1"
		}
		if data.NeedHashSHA256 {
			data.Imports["sha256"] = "crypto/sha256"
		}
	}

	// UUID helper needs crypto/sha1 and fmt
	if data.NeedUUIDHelper {
		data.Imports["sha1"] = "crypto/sha1"
		data.Imports["fmt"] = "fmt"
	}

	// IP helper needs net and strings
	if data.NeedIPHelper {
		data.Imports["net"] = "net"
		data.Imports["strings"] = "strings"
	}

	// URL helper needs net/url and strings
	if data.NeedURLHelper {
		data.Imports["url"] = "net/url"
		data.Imports["strings"] = "strings"
	}

	// FixedLength helper needs strings
	if data.NeedFixedLengthHelper {
		data.Imports["strings"] = "strings"
	}

	// Condition helper needs os
	if data.NeedConditionHelper {
		data.Imports["os"] = "os"
	}

	// Custom helper needs the redact package
	if data.NeedCustomHelper {
		data.Imports["redact"] = "github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact/redact/v1"
	}

	// render file in the template
	name := m.ctx.OutputPath(file).SetExt(".redact.go")
	m.AddGeneratorTemplateFile(name.String(), m.tmpl, data)
}

// processService extracts all pgs.Service and their pgs.Method(s) information and
// structures them into ServiceData
func (m *Module) processService(
	srv pgs.Service,
	nameWithAlias func(n pgs.Entity) string,
) *ServiceData {
	// Validate service before processing
	if err := m.validateService(srv); err != nil {
		m.Failf("Cannot process service: %v", err)
		return nil
	}

	defer m.recoverFromPanic(fmt.Sprintf("processing service %s", srv.FullyQualifiedName()))

	srvData := &ServiceData{
		Name:    m.ctx.Name(srv).String(),
		Methods: make([]*MethodData, 0, len(srv.Methods())),
	}

	// check service option: ServiceSkip
	srvSkip := false
	m.must(srv.Extension(redact.E_ServiceSkip, &srvSkip))
	if srvSkip {
		srvData.Skip = true
		m.Debug(fmt.Sprintf("Service %s is marked as skipped", srv.FullyQualifiedName()))
		// continue
	}

	// check internal service options
	srvInternal := false
	m.must(srv.Extension(redact.E_InternalService, &srvInternal))
	srvCode := uint32(codes.PermissionDenied) // default code
	if !m.must(srv.Extension(redact.E_InternalServiceCode, &srvCode)) {
		srvCode = uint32(codes.PermissionDenied)
	}

	// Validate status code with better error message
	if err := m.validateStatusCode(srvCode, srv.FullyQualifiedName()); err != nil {
		m.Fail(err)
		return nil
	}
	srvErrMsg := ""
	if !m.must(srv.Extension(redact.E_InternalServiceErrMessage, &srvErrMsg)) {
		srvErrMsg = defaultErrMsg
	}

	// methods
	for _, meth := range srv.Methods() {
		// Validate method before processing
		if err := m.validateMethod(meth); err != nil {
			m.Failf("Cannot process method %s: %v", meth.FullyQualifiedName(), err)
			continue
		}

		in := meth.Input()
		out := meth.Output()

		// Additional safety checks
		if in == nil {
			m.Failf("Method %s has nil input message", meth.FullyQualifiedName())
			continue
		}
		if out == nil {
			m.Failf("Method %s has nil output message", meth.FullyQualifiedName())
			continue
		}

		methData := &MethodData{
			Name:            m.ctx.Name(meth).String(),
			Input:           nameWithAlias(in),
			Output:          m.processMessage(out, nameWithAlias),
			ClientStreaming: meth.ClientStreaming(),
			ServerStreaming: meth.ServerStreaming(),
		}
		srvData.Methods = append(srvData.Methods, methData)

		// check method skip options
		methSkip := false
		m.must(meth.Extension(redact.E_MethodSkip, &methSkip))
		if methSkip || srvSkip {
			methData.Skip = true
			if methSkip {
				m.Debug(fmt.Sprintf("Method %s is marked as skipped", meth.FullyQualifiedName()))
			}
			continue
		}

		methInternal := false
		m.must(meth.Extension(redact.E_InternalMethod, &methInternal))
		methCode := srvCode // serviceCode
		if !m.must(meth.Extension(redact.E_InternalMethodCode, &methCode)) {
			methCode = srvCode
		}

		// Validate method status code with better error message
		if err := m.validateStatusCode(methCode, meth.FullyQualifiedName()); err != nil {
			m.Fail(err)
			continue
		}
		methErrMsg := srvErrMsg
		if !m.must(meth.Extension(redact.E_InternalMethodErrMessage, &methErrMsg)) {
			methErrMsg = srvErrMsg
		}

		// apply format specifiers
		methErrMsg = strings.ReplaceAll(methErrMsg, specifierMethod, methData.Name)
		methErrMsg = strings.ReplaceAll(methErrMsg, specifierService, srvData.Name)

		methData.ErrMessage = "`" + methErrMsg + "`"
		methData.StatusCode = codes.Code(methCode).String()
		methData.Internal = srvInternal || methInternal
	}
	return srvData
}

// applyAutoDetect scans all string fields in the file for names matching
// the auto_detect patterns and applies the default_action rules to them.
// Fields that already have explicit redaction rules are left unchanged.
func (m *Module) applyAutoDetect(
	file pgs.File,
	data *ProtoFileData,
	nameWithAlias func(n pgs.Entity) string,
) {
	var ad redact.AutoDetectRules
	if !m.must(file.Extension(redact.E_AutoDetect, &ad)) {
		return
	}
	if ad.DefaultAction == nil || ad.DefaultAction.Values == nil || len(ad.Patterns) == 0 {
		return
	}

	// Build a lookup map: message name -> field name -> *FieldData
	msgFieldMap := make(map[string]map[string]*FieldData)
	for _, msg := range data.Messages {
		fm := make(map[string]*FieldData)
		for _, fld := range msg.Fields {
			fm[fld.Name] = fld
		}
		for _, oneof := range msg.Oneofs {
			for _, fld := range oneof.Fields {
				fm[fld.Name] = fld.FieldData
			}
		}
		msgFieldMap[msg.Name] = fm
	}

	for _, msg := range file.AllMessages() {
		msgName := m.ctx.Name(msg).String()
		fm, ok := msgFieldMap[msgName]
		if !ok {
			continue
		}
		for _, field := range msg.Fields() {
			fieldName := m.ctx.Name(field).String()
			fld, ok := fm[fieldName]
			if !ok || fld == nil || fld.Redact {
				continue
			}
			// Auto-detect only applies to scalar string fields
			typ := field.Type()
			if typ == nil || typ.ProtoType() != pgs.StringT {
				continue
			}
			// Check name against patterns (case-insensitive contains)
			lowerName := strings.ToLower(field.Name().String())
			matched := false
			for _, p := range ad.Patterns {
				if strings.Contains(lowerName, strings.ToLower(p)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			// Apply default action
			fld.Redact = true
			fld.RedactionValue = RedactionDefaults(pgs.StringT, false)
			m.redactedCustomValue(fld, field, ad.DefaultAction)
			// Handle regex var name setup
			if fld.IsRegex {
				var pattern string
				if rr := getRegexRules(typ, ad.DefaultAction); rr != nil {
					pattern = rr.GetPattern()
				}
				fld.RegexVarName = regexVarName(msgName, fld.Name)
				fld.RegexPatternLiteral = strconv.Quote(pattern)
			}
		}
	}
}

// processMessage extracts all pgs.Message and their pgs.Field(s) information and
// structures them into MessageData
func (m *Module) processMessage(
	msg pgs.Message,
	nameWithAlias func(n pgs.Entity) string,
	wantFields ...bool,
) *MessageData {
	// Validate message before processing
	if err := m.validateMessage(msg); err != nil {
		m.Failf("Cannot process message: %v", err)
		return nil
	}

	defer m.recoverFromPanic(fmt.Sprintf("processing message %s", msg.FullyQualifiedName()))

	msgData := &MessageData{
		Name:      m.ctx.Name(msg).String(),
		WithAlias: nameWithAlias(msg),
		Fields:    make([]*FieldData, 0, len(msg.Fields())*2),
	}

	// check message ignore options
	msgData.Ignore = false
	m.must(msg.Extension(redact.E_Ignored, &msgData.Ignore))
	if msgData.Ignore {
		m.Debug(fmt.Sprintf("Message %s is marked as ignored", msg.FullyQualifiedName()))
		return msgData
	}

	// check message nil options
	msgData.ToNil = false
	m.must(msg.Extension(redact.E_Nil, &msgData.ToNil))

	// check message empty options
	msgData.ToEmpty = false
	m.must(msg.Extension(redact.E_Empty, &msgData.ToEmpty))

	// Log warning if both nil and empty are set (validation should have caught this)
	if msgData.ToNil && msgData.ToEmpty {
		m.Debug(fmt.Sprintf("Warning: Message %s has both nil and empty options - this is invalid", msg.FullyQualifiedName()))
	}

	if len(wantFields) > 0 {
		for _, field := range msg.Fields() {
			// Skip fields that belong to non-synthetic oneofs;
			// they are processed as part of OneofData below
			if field.InOneOf() && !field.OneOf().IsSynthetic() {
				continue
			}
			msgData.Fields = append(msgData.Fields, m.processFields(field, nameWithAlias, msgData.Name))
		}

		// Process non-synthetic oneofs (real oneof groups)
		for _, oneOf := range msg.RealOneOfs() {
			oneofData := &OneofData{
				Name:   m.ctx.Name(oneOf).String(),
				Fields: make([]*OneofFieldData, 0, len(oneOf.Fields())),
			}
			for _, field := range oneOf.Fields() {
				fieldData := m.processFields(field, nameWithAlias, msgData.Name)
				oneofData.Fields = append(oneofData.Fields, &OneofFieldData{
					FieldData:       fieldData,
					WrapperTypeName: msgData.Name + "_" + fieldData.Name,
				})
			}
			msgData.Oneofs = append(msgData.Oneofs, oneofData)
		}
	}
	return msgData
}
