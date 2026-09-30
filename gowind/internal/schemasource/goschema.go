package schemasource

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// Go 源码 schema 源(ent schema 目录 / gorm model 目录)的方言标记。
// DSN scheme 与之一一对应:ent://<dir>、gorm://<dir>。
const (
	DialectEntSchema  = "ent-schema"
	DialectGormSchema = "gorm-schema"
)

// ParsedSchema 是 Go 源码 schema 解析器(ent/gorm)的中间产物。
// AutoIncrement 以 "table.column" 为键标记自增列——atlas 通用类型系统
// 没有跨方言的自增属性,DDL 导出与元数据消费都通过它拿自增信息。
type ParsedSchema struct {
	Tables        []*schema.Table
	AutoIncrement map[string]bool
	Warnings      []string
}

func incKey(table, column string) string { return table + "." + column }

func (p *ParsedSchema) markAutoIncrement(table, column string) {
	p.AutoIncrement[incKey(table, column)] = true
}

// memoryInspector 把解析好的 atlas schema 包装成 schema.Inspector,
// 使 ent-schema/gorm-schema 数据源对下游(sqlproto 等)透明可用。
type memoryInspector struct {
	s *schema.Schema
}

func (m *memoryInspector) InspectSchema(_ context.Context, _ string, opts *schema.InspectOptions) (*schema.Schema, error) {
	if opts == nil || len(opts.Tables) == 0 {
		return m.s, nil
	}
	want := make(map[string]bool, len(opts.Tables))
	for _, t := range opts.Tables {
		want[t] = true
	}
	filtered := &schema.Schema{Name: m.s.Name}
	for _, t := range m.s.Tables {
		if want[t.Name] {
			t.Schema = filtered
			filtered.Tables = append(filtered.Tables, t)
		}
	}
	return filtered, nil
}

func (m *memoryInspector) InspectRealm(_ context.Context, _ *schema.InspectRealmOption) (*schema.Realm, error) {
	return nil, fmt.Errorf("schemasource: go source schema does not support realm inspection")
}

func entSchemaProvider(path string) (*Driver, error) {
	return goSchemaDriver(path, DialectEntSchema, ParseEntSchemaDir)
}

func gormSchemaProvider(path string) (*Driver, error) {
	return goSchemaDriver(path, DialectGormSchema, ParseGormSchemaDir)
}

func goSchemaDriver(dir, dialect string, parse func(string) (*ParsedSchema, error)) (*Driver, error) {
	ps, err := parse(dir)
	if err != nil {
		return nil, err
	}
	if len(ps.Tables) == 0 {
		return nil, fmt.Errorf("schemasource: no tables parsed from %s source %q", dialect, dir)
	}
	for _, w := range ps.Warnings {
		fmt.Printf("[WARN] %s: %s\n", dialect, w)
	}
	s := &schema.Schema{Name: "gow"}
	for _, t := range ps.Tables {
		t.Schema = s
		s.Tables = append(s.Tables, t)
	}
	return &Driver{
		Inspector:  &memoryInspector{s: s},
		Dialect:    dialect,
		SchemaName: dir,
	}, nil
}

// parseGoDir 解析目录下的非测试 Go 文件,返回 FileSet 与文件列表。
func parseGoDir(dir string) (*token.FileSet, []*ast.File, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go") && !strings.HasPrefix(fi.Name(), ".")
	}, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, fmt.Errorf("schemasource: parse dir %q: %w", dir, err)
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("schemasource: no go packages found in %q", dir)
	}
	var files []*ast.File
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("schemasource: no go files found in %q", dir)
	}
	return fset, files, nil
}

// litString 取调用第 i 个参数的字符串字面量。
func litString(ce *ast.CallExpr, i int) (string, bool) {
	if ce == nil || i >= len(ce.Args) {
		return "", false
	}
	bl, ok := ce.Args[i].(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(bl.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// litInt 取调用第 i 个参数的整数字面量。
func litInt(ce *ast.CallExpr, i int) (int, bool) {
	if ce == nil || i >= len(ce.Args) {
		return 0, false
	}
	bl, ok := ce.Args[i].(*ast.BasicLit)
	if !ok || bl.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.Atoi(bl.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// splitCallChain 拆解 pkg.Base(args).M1(...).M2(...) 形式的调用链,
// 返回最内层的基础调用与外层方法名到调用的映射。
func splitCallChain(e ast.Expr) (*ast.CallExpr, map[string]*ast.CallExpr) {
	methods := map[string]*ast.CallExpr{}
	cur := e
	for {
		ce, ok := cur.(*ast.CallExpr)
		if !ok {
			return nil, nil
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil, nil
		}
		if _, isPkgIdent := sel.X.(*ast.Ident); isPkgIdent {
			return ce, methods
		}
		methods[sel.Sel.Name] = ce
		cur = sel.X
	}
}

// baseCallPkgFn 返回基础调用的包名与函数名,如 field.String → ("field","String")。
func baseCallPkgFn(base *ast.CallExpr) (string, string) {
	if base == nil {
		return "", ""
	}
	sel, ok := base.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", ""
	}
	return pkg.Name, sel.Sel.Name
}

// selectorName 返回 X.Sel 形式的选择器名(如 Pet.Type → "Pet"),否则返回空。
func selectorName(e ast.Expr) (string, string) {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		return id.Name, sel.Sel.Name
	}
	return "", sel.Sel.Name
}

// newColumn 构造一个 atlas 列:Raw 为 SQL 类型文本,Null 为可空标记。
func newColumn(name, raw string, null bool) *schema.Column {
	typ, err := ParseType(raw)
	if err != nil {
		typ = &schema.StringType{T: "varchar"}
	}
	return &schema.Column{
		Name: name,
		Type: &schema.ColumnType{Type: typ, Raw: raw, Null: null},
	}
}

// setDefault 把 SQL 默认值表达式挂到列上。
func setDefault(col *schema.Column, expr string) {
	col.Default = &schema.NamedDefault{
		Expr: &schema.RawExpr{X: expr},
	}
}
