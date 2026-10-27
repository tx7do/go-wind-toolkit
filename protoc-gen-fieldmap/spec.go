// Package main 提供字段映射代码生成器的 spec 模型与加载校验。
//
// spec 描述"某个 proto 消息 → 某个适配器包"的字段映射规则。每个适配器
// 一份 spec 文件（YAML），生成器据此产出强类型的映射函数，替代手工维护
// 的 if-guard + option 拼装样板。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// Style 生成风格。
const (
	// StyleOption 追加式：向 []TARGET 追加上游 With* 选项（gorm 适配器风格）。
	StyleOption = "option"
	// StyleStruct 构造式：返回填充好的上游配置结构体指针（minio 适配器风格）。
	StyleStruct = "struct"
	// StyleFill 填充式：向上游"先构造后填字段"的配置对象逐字段赋值
	// （registry/config 域的 api.DefaultConfig() + 逐字段覆盖模式）。
	StyleFill = "fill"
)

// Guard 值卫语句集合（空串 = auto，按字段类型推导）。
const (
	guardAuto     = ""
	guardNone     = "none"     // 无条件应用
	guardEmpty    = "empty"    // v != ""
	guardZero     = "zero"     // v != 0
	guardPositive = "positive" // v > 0
	guardNotNil   = "not_nil"  // v != nil
	guardLen      = "len"      // len(v) > 0
)

// Cast 值转换集合。
const (
	castInt     = "int"
	castInt32   = "int32"
	castInt64   = "int64"
	castUint    = "uint"
	castUint32  = "uint32"
	castUint64  = "uint64"
	castFloat32 = "float32"
	castFloat64 = "float64"
	castSeconds = "seconds" // 数值秒 → time.Duration
	castMillis  = "millis"  // 数值毫秒 → time.Duration
)

// Output 描述生成产物的落点。
type Output struct {
	// File 生成文件的相对路径（相对 -out 根目录），必须以 .go 结尾。
	File string `json:"file"`
	// Package 生成文件的 Go 包名（即目标适配器包名）。
	Package string `json:"package"`
}

// FieldSpec 单个 proto 字段的映射规则。
type FieldSpec struct {
	// Target 目标名：option 风格为上游选项方法名（如 WithDSN，可带包限定
	// 前缀 "gormCrud.WithDSN"）；struct 风格为目标结构体字段名，缺省取
	// proto 字段名的驼峰形式。
	Target string `json:"target,omitempty"`
	// Guard 卫语句；空串 = auto（按 proto 字段类型推导）。
	Guard string `json:"guard,omitempty"`
	// Cast 值转换；空串 = 原样传递。
	Cast string `json:"cast,omitempty"`
	// Spread 展开传参（option 风格专用）：repeated 字段以 TARGET(v...) 形式
	// 传给变参选项函数（如 WithAddresses(v...)）。仅限 list 形状字段。
	Spread bool `json:"spread,omitempty"`
	// Skip 显式跳过该字段（记录"此处有意不映射"，生成器同时校验字段存在）。
	Skip bool `json:"skip,omitempty"`
}

// MessageSpec 单条映射规则的主体。
type MessageSpec struct {
	// Proto 目标 proto 文件（相对 -I 包含根的路径，如 bootstrap/v1/database.proto）。
	Proto string `json:"proto"`
	// Type 消息全名，如 "Database.SQL"（可省略 proto 包前缀，自动后缀匹配）。
	Type string `json:"type"`
	// Func 生成的映射函数名。
	Func string `json:"func"`
	// Style 生成风格：option（默认）或 struct。
	Style string `json:"style,omitempty"`
	// TargetType option 风格：选项元素类型（如 "gormCrud.Option"，限定符须在
	// imports 中声明）；struct 风格：目标结构体类型（如 "minioPlugin.Config"）。
	TargetType string `json:"target_type"`
	// TargetQualifier option 风格专用：字段 target 缺省包限定符（如
	// "gormCrud"）。声明后各字段的 target 免写 "gormCrud." 前缀；仍支持
	// 个别字段显式携带前缀以跨包引用。
	TargetQualifier string `json:"target_qualifier,omitempty"`
	// Param 映射函数的参数名，默认 c。
	Param string `json:"param,omitempty"`
	// TargetParam fill 风格的被填充对象参数名，默认 cfg。
	TargetParam string `json:"target_param,omitempty"`
	// Append option 风格的选项切片参数名，默认 opts。
	Append string `json:"append,omitempty"`
	// Comment 函数文档；缺省取 proto 消息的 leading comments。
	Comment string `json:"comment,omitempty"`
	// Fields 字段映射表，key = proto 字段名。
	Fields map[string]FieldSpec `json:"fields"`
}

// Spec 一份完整的生成规则（一个 YAML 文件一条）。
type Spec struct {
	Output Output `json:"output"`
	// Imports 生成文件需要的第三方导入：限定符 → 导入路径。
	// 消息所在的 conf 包与 time 由生成器自动补齐，无需声明。
	Imports map[string]string `json:"imports,omitempty"`
	Message MessageSpec       `json:"message"`

	// source spec 文件路径，仅用于报错定位。
	source string
}

// Source 返回 spec 文件路径。
func (s *Spec) Source() string { return s.source }

// LoadSpecs 加载 -spec 参数：文件（单条 spec）或目录（逐个 *.yaml/*.yml）。
// 返回按输出文件名排序的 spec 列表，保证多次运行产物稳定。
func LoadSpecs(paths []string) ([]*Spec, error) {
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("spec %q: %w", p, err)
		}
		if !info.IsDir() {
			files = append(files, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, fmt.Errorf("spec dir %q: %w", p, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			switch strings.ToLower(filepath.Ext(e.Name())) {
			case ".yaml", ".yml":
				files = append(files, filepath.Join(p, e.Name()))
			}
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no spec files found in %v", paths)
	}
	sort.Strings(files)

	specs := make([]*Spec, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read spec %s: %w", f, err)
		}
		var s Spec
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse spec %s: %w", f, err)
		}
		// 统一为正斜杠，保证生成文件头在 Windows/Linux 上一致（golden 稳定）。
		s.source = filepath.ToSlash(f)
		if err := s.validate(); err != nil {
			return nil, fmt.Errorf("spec %s: %w", f, err)
		}
		specs = append(specs, &s)
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].Output.File < specs[j].Output.File
	})
	return specs, nil
}

// validate 校验 spec 的静态规则；涉及 proto 的字段存在性与类型适配性
// 在生成阶段结合描述符校验。
func (s *Spec) validate() error {
	if s.Output.File == "" {
		return fmt.Errorf("output.file is required")
	}
	if !strings.HasSuffix(s.Output.File, ".go") {
		return fmt.Errorf("output.file %q must end with .go", s.Output.File)
	}
	if s.Output.Package == "" {
		return fmt.Errorf("output.package is required")
	}
	m := &s.Message
	if m.Proto == "" {
		return fmt.Errorf("message.proto is required")
	}
	if m.Type == "" {
		return fmt.Errorf("message.type is required")
	}
	if m.Func == "" {
		return fmt.Errorf("message.func is required")
	}
	if m.Style == "" {
		m.Style = StyleOption
	}
	switch m.Style {
	case StyleOption, StyleStruct, StyleFill:
	default:
		return fmt.Errorf("message.style %q: want %q, %q or %q", m.Style, StyleOption, StyleStruct, StyleFill)
	}
	if m.TargetType == "" {
		return fmt.Errorf("message.target_type is required")
	}
	if m.Param == "" {
		m.Param = "c"
	}
	if m.TargetParam == "" {
		m.TargetParam = "cfg"
	}
	if m.TargetParam == m.Param {
		return fmt.Errorf("message.target_param %q must differ from param", m.TargetParam)
	}
	if m.Append == "" {
		m.Append = "opts"
	}
	if len(m.Fields) == 0 {
		return fmt.Errorf("message.fields is required (at least one entry; use skip: true to record intent)")
	}
	for qualifier, path := range s.Imports {
		if !isIdent(qualifier) {
			return fmt.Errorf("imports qualifier %q is not a valid identifier", qualifier)
		}
		if path == "" {
			return fmt.Errorf("imports[%s]: empty path", qualifier)
		}
	}
	if m.Style == StyleOption {
		if _, _, err := s.splitQualified(m.TargetType); err != nil {
			return fmt.Errorf("message.target_type: %w", err)
		}
		if m.TargetQualifier != "" {
			if _, ok := s.Imports[m.TargetQualifier]; !ok {
				return fmt.Errorf("message.target_qualifier %q is not declared in imports", m.TargetQualifier)
			}
		}
	}
	for key, fs := range m.Fields {
		if err := s.validateField(key, &fs); err != nil {
			return err
		}
	}
	return nil
}

func (s *Spec) validateField(key string, fs *FieldSpec) error {
	switch fs.Guard {
	case guardAuto, guardNone, guardEmpty, guardZero, guardPositive, guardNotNil, guardLen:
	default:
		return fmt.Errorf("fields[%s].guard %q: want auto/none/empty/zero/positive/not_nil/len", key, fs.Guard)
	}
	switch fs.Cast {
	case "", castInt, castInt32, castInt64, castUint, castUint32, castUint64, castFloat32, castFloat64, castSeconds, castMillis:
	default:
		return fmt.Errorf("fields[%s].cast %q: unsupported", key, fs.Cast)
	}
	if fs.Spread && s.Message.Style != StyleOption {
		return fmt.Errorf("fields[%s]: spread is only supported in %q style", key, StyleOption)
	}
	if fs.Skip {
		return nil
	}
	if s.Message.Style == StyleStruct && fs.Guard != guardAuto && fs.Guard != guardNone {
		return fmt.Errorf("fields[%s]: struct style does not support guard %q (zero values are applied as-is)", key, fs.Guard)
	}
	if fs.Target == "" && (s.Message.Style == StyleStruct || s.Message.Style == StyleFill) {
		// struct/fill 风格缺省目标 = proto 字段名驼峰化，生成阶段解析。
		return nil
	}
	if fs.Target == "" {
		return fmt.Errorf("fields[%s].target is required in %s style", key, s.Message.Style)
	}
	switch s.Message.Style {
	case StyleOption:
		if _, _, err := s.splitQualified(fs.Target); err != nil {
			return fmt.Errorf("fields[%s].target: %w", key, err)
		}
	case StyleStruct, StyleFill:
		// 目标是上游结构体的字段名，不允许包限定。
		if _, _, found := strings.Cut(fs.Target, "."); found {
			return fmt.Errorf("fields[%s].target %q must be a bare field name in %s style", key, fs.Target, s.Message.Style)
		}
	}
	return nil
}

// splitQualified 把 "qual.Name" 拆为限定符与裸名；限定符必须在 imports 中
// 声明（无需包限定的裸名返回空限定符）。返回 (qualifier, bare, error)。
func (s *Spec) splitQualified(name string) (string, string, error) {
	qual, bare, found := strings.Cut(name, ".")
	if !found {
		return "", name, nil
	}
	if _, ok := s.Imports[qual]; !ok {
		return "", "", fmt.Errorf("qualifier %q is not declared in imports", qual)
	}
	return qual, bare, nil
}

// qualifiedType 渲染类型引用：有包限定时用 imports 中的限定符，否则裸名。
func (s *Spec) qualifiedType(name string) string {
	qual, bare, _ := s.splitQualified(name)
	if qual == "" {
		return name
	}
	return qual + "." + bare
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
