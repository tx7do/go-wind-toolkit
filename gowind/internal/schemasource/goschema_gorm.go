package schemasource

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"

	gormschema "gorm.io/gorm/schema"
)

var gormNaming = &gormschema.NamingStrategy{}

// gormColumn / gormTable 复用 gorm 官方命名策略,与 AutoMigrate 结果一致。
func gormColumn(column string) string { return gormNaming.ColumnName("", column) }
func gormTable(model string) string   { return gormNaming.TableName(model) }

// gormBasicSQLTypes / gormSelectorSQL 与 gorm MySQL 驱动 AutoMigrate 的映射一致。
var gormBasicSQLTypes = map[string]string{
	"string":   "varchar(255)",
	"bool":     "bool",
	"int":      "bigint",
	"int8":     "tinyint",
	"int16":    "smallint",
	"int32":    "int",
	"int64":    "bigint",
	"uint":     "bigint unsigned",
	"uint8":    "tinyint unsigned",
	"uint16":   "smallint unsigned",
	"uint32":   "int unsigned",
	"uint64":   "bigint unsigned",
	"uintptr":  "bigint unsigned",
	"byte":     "tinyint unsigned",
	"rune":     "int",
	"float32":  "float",
	"float64":  "double",
}

var gormSelectorSQL = map[string]string{
	"time.Time":       "datetime",
	"time.Duration":   "bigint",
	"uuid.UUID":       "char(36)",
	"decimal.Decimal": "decimal(10,2)",
	"json.RawMessage": "json",
	"datatypes.JSON":  "json",
	"types.JSON":      "json",
}

type gormAssoc struct {
	owner      string // 声明关联的结构体名
	field      string // 字段名
	target     string // 目标结构体名
	slice      bool   // 字段是结构体切片
	foreignKey string // tag foreignKey(字段名)
	references string // tag references(字段名)
	joinTable  string // tag many2many 指定的中间表名
}

type gormModel struct {
	name          string
	structType    *ast.StructType
	isModel       bool
	columns       []*schema.Column
	fieldToColumn map[string]string
	primaryKey    *schema.Column
	autoIncrement map[string]bool
	associations  []gormAssoc
	warnings      []string
	built         *schema.Table
}

type gormSource struct {
	ps         *ParsedSchema
	structs    map[string]*ast.StructType
	aliases    map[string]string // type X int → "int"
	tableNames map[string]string // TableName() 方法声明者 → 表名
	models     map[string]*gormModel
	embeddedAs map[string]bool // 被其它结构体匿名嵌入的类型(非模型)
}

// ParseGormSchemaDir 以 AST 方式解析 gorm model 目录,产出 atlas 表模型。
// 模型判定:拥有 TableName 方法、primaryKey tag、ID/Id 字段或嵌入 gorm.Model
// 的结构体视为模型;仅作为嵌入基类型出现的结构体不作为模型。
// 表/列命名复用 gorm 官方 NamingStrategy,与 AutoMigrate 结果一致。
func ParseGormSchemaDir(dir string) (*ParsedSchema, error) {
	_, files, err := parseGoDir(dir)
	if err != nil {
		return nil, err
	}

	g := &gormSource{
		ps:         &ParsedSchema{AutoIncrement: map[string]bool{}},
		structs:    map[string]*ast.StructType{},
		aliases:    map[string]string{},
		tableNames: map[string]string{},
		models:     map[string]*gormModel{},
		embeddedAs: map[string]bool{},
	}

	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					switch t := ts.Type.(type) {
					case *ast.StructType:
						g.structs[ts.Name.Name] = t
					case *ast.Ident, *ast.SelectorExpr:
						if name := exprTypeName(t); name != "" {
							g.aliases[ts.Name.Name] = name
						}
					}
				}
			case *ast.FuncDecl:
				recv := receiverName(d)
				if recv == "" || d.Name == nil {
					continue
				}
				if d.Name.Name == "TableName" {
					if ret, ok := returnStringLiteral(d); ok {
						g.tableNames[recv] = ret
					}
				}
			}
		}
	}

	// 预扫描:标记被匿名嵌入的本地结构体(它们只是基类型,不是模型)。
	for _, st := range g.structs {
		for _, fl := range st.Fields.List {
			if len(fl.Names) != 0 {
				continue
			}
			name, pkg := embedName(fl.Type)
			if pkg == nil && name != "" {
				g.embeddedAs[name] = true
			}
		}
	}

	names := make([]string, 0, len(g.structs))
	for name := range g.structs {
		if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		g.models[name] = &gormModel{
			name:          name,
			structType:    g.structs[name],
			fieldToColumn: map[string]string{},
			autoIncrement: map[string]bool{},
		}
	}

	for _, name := range names {
		g.detectModel(g.models[name])
	}
	for _, name := range names {
		if m := g.models[name]; m.isModel {
			g.parseFields(m, m.name, m.structType, "", map[string]bool{name: true})
		}
	}
	g.buildTables(names)
	g.resolveAssociations(names)

	for _, name := range names {
		if m := g.models[name]; m.isModel {
			g.ps.Warnings = append(g.ps.Warnings, m.warnings...)
		}
	}
	return g.ps, nil
}

// detectModel 判定结构体是否按 gorm 模型对待。
func (g *gormSource) detectModel(m *gormModel) {
	if _, ok := g.tableNames[m.name]; ok {
		m.isModel = true
		return
	}
	if g.embeddedAs[m.name] {
		return
	}
	if m.structType == nil {
		return
	}
	for _, fl := range m.structType.Fields.List {
		tag := parseGormTag(fl.Tag)
		if tag.primaryKey {
			m.isModel = true
			return
		}
		if len(fl.Names) == 0 {
			if name, pkg := embedName(fl.Type); name == "Model" && pkg != nil && pkg.Name == "gorm" {
				m.isModel = true
				return
			}
			continue
		}
		for _, id := range fl.Names {
			if id.Name == "ID" || id.Name == "Id" {
				m.isModel = true
				return
			}
		}
	}
}

func exprTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return exprTypeName(t.X)
	}
	return ""
}

// embedName 返回匿名嵌入字段的类型名与包标识(gorm.Model → ("Model", gorm))。
func embedName(e ast.Expr) (string, *ast.Ident) {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name, nil
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, pkg
		}
	}
	return "", nil
}

func returnStringLiteral(f *ast.FuncDecl) (string, bool) {
	if f.Body == nil {
		return "", false
	}
	for _, st := range f.Body.List {
		ret, ok := st.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		if bl, ok := ret.Results[0].(*ast.BasicLit); ok && bl.Kind == token.STRING {
			if v, err := strconv.Unquote(bl.Value); err == nil {
				return v, true
			}
		}
	}
	return "", false
}

type gormTag struct {
	column       string
	typ          string
	size         int
	primaryKey   bool
	autoIncr     bool
	notNull      bool
	explicitNull bool
	def          string
	comment      string
	embedded     bool
	prefix       string
	skip         bool
	foreignKey   string
	references   string
	joinTable    string
}

func parseGormTag(lit *ast.BasicLit) gormTag {
	var t gormTag
	if lit == nil || lit.Kind != token.STRING {
		return t
	}
	raw, err := strconv.Unquote(lit.Value)
	if err != nil {
		return t
	}
	// 结构体 tag 含多个 section(gorm:"...";json:"..."),先取 gorm 段。
	gormVal := reflect.StructTag(raw).Get("gorm")
	if gormVal == "" {
		return t
	}
	if gormVal == "-" || strings.HasPrefix(gormVal, "-:") {
		t.skip = true
		return t
	}
	for _, part := range strings.Split(gormVal, ";") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		key, val := p, ""
		if i := strings.Index(p, ":"); i >= 0 {
			key, val = strings.TrimSpace(p[:i]), p[i+1:]
		}
		switch strings.ToLower(key) {
		case "column":
			t.column = val
		case "type":
			t.typ = val
		case "size":
			if n, err := strconv.Atoi(val); err == nil {
				t.size = n
			}
		case "primarykey":
			t.primaryKey = true
		case "autoincrement":
			t.autoIncr = true
		case "not null":
			t.notNull = true
		case "null":
			t.explicitNull = true
		case "default":
			t.def = val
		case "comment":
			t.comment = val
		case "embedded":
			t.embedded = true
		case "embeddedprefix":
			t.prefix = val
		case "foreignkey":
			t.foreignKey = val
		case "references":
			t.references = val
		case "many2many":
			t.joinTable = val
		}
	}
	return t
}

// parseFields 展开一个结构体(含匿名嵌入基类型)的字段到模型 m。
func (g *gormSource) parseFields(m *gormModel, current string, st *ast.StructType, prefix string, visiting map[string]bool) {
	if st == nil {
		return
	}
	for _, fl := range st.Fields.List {
		tag := parseGormTag(fl.Tag)
		if tag.skip {
			continue
		}
		if len(fl.Names) == 0 {
			g.parseEmbedded(m, current, fl, tag, prefix, visiting)
			continue
		}
		fieldName := fl.Names[0].Name
		if fieldName[0] >= 'a' && fieldName[0] <= 'z' {
			continue // 未导出
		}
		if isFuncType(fl.Type) {
			continue
		}
		if tag.embedded {
			base := exprTypeName(fl.Type)
			if st2, ok := g.structs[base]; ok {
				if !visiting[base] {
					visiting[base] = true
					g.parseFields(m, base, st2, prefix+tag.prefix, visiting)
					delete(visiting, base)
				}
			} else {
				m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s.%s: embedded type %q not found in dir, skipped", current, fieldName, base))
			}
			continue
		}
		g.parseField(m, current, fieldName, fl.Type, tag, prefix)
	}
}

func isFuncType(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.FuncType:
		return true
	case *ast.StarExpr:
		return isFuncType(t.X)
	}
	return false
}

func (g *gormSource) parseEmbedded(m *gormModel, current string, fl *ast.Field, tag gormTag, prefix string, visiting map[string]bool) {
	name, pkg := embedName(fl.Type)
	if name == "" {
		m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s: unsupported embedded field, skipped", current))
		return
	}
	if pkg != nil && pkg.Name == "gorm" && name == "Model" {
		g.injectGormModel(m, prefix)
		return
	}
	if pkg != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s: embedded %s.%s not resolvable statically, skipped", current, pkg.Name, name))
		return
	}
	st, ok := g.structs[name]
	if !ok {
		m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s: embedded type %s not found in dir, skipped", current, name))
		return
	}
	if visiting[name] {
		return
	}
	visiting[name] = true
	g.parseFields(m, name, st, prefix+tag.prefix, visiting)
	delete(visiting, name)
}

func (g *gormSource) injectGormModel(m *gormModel, prefix string) {
	id := newColumn(prefix+"id", "bigint unsigned", false)
	m.addColumn(m.name, id, "ID", true, true)
	for _, c := range []struct{ name, raw string }{
		{"created_at", "datetime"},
		{"updated_at", "datetime"},
		{"deleted_at", "datetime"},
	} {
		m.addColumn(m.name, newColumn(prefix+c.name, c.raw, true), "", false, false)
	}
}

// parseField 解析一个具名标量字段或关联字段。
func (g *gormSource) parseField(m *gormModel, current, fieldName string, typ ast.Expr, tag gormTag, prefix string) {
	nullable := false
	if star, ok := typ.(*ast.StarExpr); ok {
		nullable = true
		typ = star.X
	}

	colName := tag.column
	if colName == "" {
		colName = gormColumn(fieldName)
	}
	if prefix != "" {
		colName = prefix + colName
	}

	switch t := typ.(type) {
	case *ast.Ident:
		if _, isStruct := g.structs[t.Name]; isStruct {
			m.associations = append(m.associations, gormAssoc{
				owner: m.name, field: fieldName, target: t.Name,
				foreignKey: tag.foreignKey, references: tag.references, joinTable: tag.joinTable,
			})
			return
		}
	case *ast.ArrayType:
		if ident, ok := t.Elt.(*ast.Ident); ok && ident.Name == "byte" {
			col := newColumn(colName, "blob", nullable || tag.explicitNull)
			m.addColumn(current, col, fieldName, tag.primaryKey, tag.autoIncr)
			g.applyAttrs(col, tag)
			return
		}
		target := exprTypeName(t.Elt)
		if _, isStruct := g.structs[target]; isStruct {
			m.associations = append(m.associations, gormAssoc{
				owner: m.name, field: fieldName, target: target, slice: true,
				foreignKey: tag.foreignKey, references: tag.references, joinTable: tag.joinTable,
			})
			return
		}
		m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s.%s: unsupported slice element %q, mapped to json", current, fieldName, target))
		col := newColumn(colName, "json", true)
		m.addColumn(current, col, fieldName, tag.primaryKey, tag.autoIncr)
		return
	case *ast.MapType:
		m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s.%s: map field mapped to json", current, fieldName))
		col := newColumn(colName, "json", true)
		m.addColumn(current, col, fieldName, tag.primaryKey, tag.autoIncr)
		return
	}

	raw, ok := g.scalarType(typ)
	if !ok {
		m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s.%s: unresolved type %q, mapped to varchar(255)", current, fieldName, exprTypeName(typ)))
		raw = "varchar(255)"
	}
	if tag.typ != "" {
		raw = tag.typ
	} else if tag.size > 0 && strings.HasPrefix(raw, "varchar") {
		raw = fmt.Sprintf("varchar(%d)", tag.size)
	}

	null := nullable || tag.explicitNull || (tag.def != "" && !tag.notNull)
	if tag.notNull || tag.primaryKey {
		null = false
	}
	col := newColumn(colName, raw, null)
	m.addColumn(current, col, fieldName, tag.primaryKey, tag.autoIncr)
	g.applyAttrs(col, tag)
}

// scalarType 把字段类型解析为 SQL 类型文本(含同包别名一层展开)。
func (g *gormSource) scalarType(e ast.Expr) (string, bool) {
	switch t := e.(type) {
	case *ast.Ident:
		if raw, ok := gormBasicSQLTypes[t.Name]; ok {
			return raw, true
		}
		if under, ok := g.aliases[t.Name]; ok {
			if raw, ok := gormBasicSQLTypes[under]; ok {
				return raw, true
			}
			if raw, ok := gormSelectorSQL[under]; ok {
				return raw, true
			}
		}
		return "", false
	case *ast.SelectorExpr:
		raw, ok := gormSelectorSQL[exprTypeName(e)]
		return raw, ok
	}
	return "", false
}

func (g *gormSource) applyAttrs(col *schema.Column, tag gormTag) {
	if tag.comment != "" {
		col.SetComment(tag.comment)
	}
	if tag.def != "" {
		setDefault(col, normalizeDefaultExpr(tag.def))
	}
}

func normalizeDefaultExpr(def string) string {
	upper := strings.ToUpper(def)
	if upper == "CURRENT_TIMESTAMP" || upper == "NULL" {
		return upper
	}
	if _, err := strconv.ParseFloat(def, 64); err == nil {
		return def
	}
	if strings.HasPrefix(def, "'") {
		return def
	}
	return "'" + strings.ReplaceAll(def, "'", "''") + "'"
}

func (m *gormModel) addColumn(current string, col *schema.Column, fieldName string, primaryKey, autoIncr bool) {
	if fieldName != "" {
		m.fieldToColumn[fieldName] = col.Name
	}
	m.columns = append(m.columns, col)
	if primaryKey {
		if m.primaryKey == nil {
			m.primaryKey = col
		} else {
			m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s: composite primary key not supported, column %q kept as plain", current, col.Name))
		}
	}
	if autoIncr {
		m.autoIncrement[col.Name] = true
	}
}

// fieldColumn 把 Go 字段名翻译为列名(优先已解析列映射,退回命名策略)。
func (m *gormModel) fieldColumn(fieldName string) string {
	if fieldName == "" {
		if m.primaryKey != nil {
			return m.primaryKey.Name
		}
		return "id"
	}
	if col, ok := m.fieldToColumn[fieldName]; ok {
		return col
	}
	return gormColumn(fieldName)
}

// buildTables 为模型生成 atlas 表。
func (g *gormSource) buildTables(names []string) {
	for _, name := range names {
		m := g.models[name]
		if !m.isModel {
			continue
		}
		t := &schema.Table{Name: m.tableName(g)}
		for _, col := range m.columns {
			t.Columns = append(t.Columns, col)
			if m.autoIncrement[col.Name] {
				g.ps.markAutoIncrement(t.Name, col.Name)
			}
		}
		pk := m.primaryKey
		if pk == nil {
			for _, col := range t.Columns {
				if col.Name == "id" {
					pk = col
					break
				}
			}
		}
		if pk != nil {
			setPK(t, pk)
		}
		m.built = t
		g.ps.Tables = append(g.ps.Tables, t)
	}
}

func (m *gormModel) tableName(g *gormSource) string {
	if t, ok := g.tableNames[m.name]; ok {
		return t
	}
	return gormTable(m.name)
}

// resolveAssociations 把结构体间关联还原为外键约束、缺失时合成的外键列,
// 以及 many2many 中间表(双列主键 + 两条单列外键,满足 IsJoinTable 判定)。
func (g *gormSource) resolveAssociations(names []string) {
	for _, name := range names {
		m := g.models[name]
		if !m.isModel || m.built == nil {
			continue
		}
		for _, a := range m.associations {
			target := g.models[a.target]
			if target == nil || !target.isModel || target.built == nil {
				m.warnings = append(m.warnings, fmt.Sprintf("gorm: %s.%s: association target %q is not a model, stored as json", m.name, a.field, a.target))
				ensureColumn(m.built, gormColumn(a.field), "json")
				continue
			}

			switch {
			case a.joinTable != "":
				g.addJoinTable(m, target, a)
			case !a.slice:
				// has-one:目标模型已有指回本模型的 <属主>ID 列。
				backRef := gormColumn(m.name + "ID")
				if a.foreignKey == "" && target.hasColumn(backRef) {
					addFKConstraint(target.built, backRef, m.built, m.fieldColumn(a.references))
					break
				}
				// belongs-to:外键列在本表,gorm 约定列名 <字段>ID,tag 优先。
				fkCol := m.fieldColumn(a.foreignKey)
				if a.foreignKey == "" {
					fkCol = gormColumn(a.field + "ID")
				}
				ensureColumn(m.built, fkCol, pkRawType(target.built))
				addFKConstraint(m.built, fkCol, target.built, target.fieldColumn(a.references))
			default:
				// has-many:外键在目标表,默认列名 <属主模型>ID。
				fkCol := gormColumn(m.name + "ID")
				if a.foreignKey != "" {
					fkCol = target.fieldColumn(a.foreignKey)
				}
				ensureColumn(target.built, fkCol, pkRawType(m.built))
				addFKConstraint(target.built, fkCol, m.built, m.fieldColumn(a.references))
			}
		}
	}
}

// explicitFK 返回 belongs-to 的外键字段名:tag 优先,否则 gorm 约定 <字段>ID。
func (g *gormSource) explicitFK(m *gormModel, a gormAssoc) string {
	if a.foreignKey != "" {
		return a.foreignKey
	}
	return a.field + "ID"
}

func (m *gormModel) hasColumn(name string) bool {
	for _, c := range m.columns {
		if c.Name == name {
			return true
		}
	}
	return false
}

func (g *gormSource) addJoinTable(m, target *gormModel, a gormAssoc) {
	name := a.joinTable
	for _, t := range g.ps.Tables {
		if t.Name == name {
			return // 用户已显式建模该中间表
		}
	}
	aCol := newColumn(m.fieldColumnOrDefault(a.foreignKey, m.name), pkRawType(m.built), false)
	bCol := newColumn(target.fieldColumnOrDefault(a.references, target.name), pkRawType(target.built), false)
	jt := &schema.Table{Name: name, Columns: []*schema.Column{aCol, bCol}}
	setPK(jt, aCol, bCol)
	jt.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "fk_" + name + "_" + aCol.Name, Columns: []*schema.Column{aCol}, RefTable: m.built, OnDelete: schema.Cascade},
		{Symbol: "fk_" + name + "_" + bCol.Name, Columns: []*schema.Column{bCol}, RefTable: target.built, OnDelete: schema.Cascade},
	}
	g.ps.Tables = append(g.ps.Tables, jt)
}

// fieldColumnOrDefault 无显式 tag 时按 gorm m2m 约定 <模型>ID 命名。
func (m *gormModel) fieldColumnOrDefault(fieldName, modelName string) string {
	if fieldName != "" {
		return m.fieldColumn(fieldName)
	}
	return gormColumn(modelName + "ID")
}

func ensureColumn(t *schema.Table, name, raw string) *schema.Column {
	for _, c := range t.Columns {
		if c.Name == name {
			return c
		}
	}
	col := newColumn(name, raw, true)
	t.Columns = append(t.Columns, col)
	return col
}

func pkRawType(t *schema.Table) string {
	if t.PrimaryKey != nil && len(t.PrimaryKey.Parts) > 0 && t.PrimaryKey.Parts[0].C != nil {
		return t.PrimaryKey.Parts[0].C.Type.Raw
	}
	return "bigint"
}

// addFKConstraint 在 child 表 colName 列与 parent 表 refColName 列之间建外键;
// child 缺少该列时按 parent 主键类型补一列。
func addFKConstraint(child *schema.Table, colName string, parent *schema.Table, refColName string) {
	col := ensureColumn(child, colName, pkRawType(parent))
	refCol := columnOf(parent, refColName)
	if refCol == nil {
		refCol = columnOf(parent, "id")
	}
	if refCol == nil && parent.PrimaryKey != nil && len(parent.PrimaryKey.Parts) > 0 {
		refCol = parent.PrimaryKey.Parts[0].C
	}
	if refCol == nil {
		return
	}
	symbol := child.Name + "." + col.Name + "->" + parent.Name + "." + refCol.Name
	for _, fk := range child.ForeignKeys {
		if fk.Symbol == symbol {
			return
		}
	}
	child.ForeignKeys = append(child.ForeignKeys, &schema.ForeignKey{
		Symbol:   symbol,
		Columns:  []*schema.Column{col},
		RefTable: parent,
		RefColumns: []*schema.Column{refCol},
		OnDelete: schema.Cascade,
	})
}

func columnOf(t *schema.Table, name string) *schema.Column {
	for _, c := range t.Columns {
		if c.Name == name {
			return c
		}
	}
	return nil
}
