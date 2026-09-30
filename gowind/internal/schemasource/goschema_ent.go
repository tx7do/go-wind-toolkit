package schemasource

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"ariga.io/atlas/sql/schema"

	"github.com/go-openapi/inflect"
)

// entFieldSQLTypes 是 ent 字段构建器到 MySQL 风格 SQL 类型文本的映射,
// 与 ent 自身迁移使用的映射保持一致(ent 的 Int/Uint 均为 64 位 → bigint)。
var entFieldSQLTypes = map[string]string{
	"Bool":    "bool",
	"Int":     "bigint",
	"Int8":    "tinyint",
	"Int16":   "smallint",
	"Int32":   "int",
	"Int64":   "bigint",
	"Uint":    "bigint unsigned",
	"Uint8":   "tinyint unsigned",
	"Uint16":  "smallint unsigned",
	"Uint32":  "int unsigned",
	"Uint64":  "bigint unsigned",
	"Float":   "double",
	"Float32": "float",
	"Float64": "double",
	"Bytes":   "blob",
	"Time":    "datetime",
	"JSON":    "json",
	"UUID":    "char(36)",
	"String":  "varchar(255)",
	"Enum":    "varchar(255)",
}

type entNode struct {
	name     string
	table    string
	fields   []*schema.Column
	edges    []*entEdgeInfo
	built    *schema.Table
	warnings []string
}

type entEdgeInfo struct {
	owner    string // 定义该 edge 的节点名
	label    string // edge 标签,如 "pets"
	target   string // 目标节点类型名,如 "Pet"
	dir      string // "to" / "from"
	ref      string // .Ref() 名称
	unique   bool   // .Unique()
	required bool   // .Required()
	via      string // .Via() 指定的中间表名
	matched  bool   // 配对解析标记
}

// ParseEntSchemaDir 以 AST 方式解析 ent schema 目录(服务约定路径
// internal/data/ent/schema),产出 atlas 表模型。Mixin、SchemaType、GoType
// 等运行期特性无法静态还原,会记入 Warnings。
func ParseEntSchemaDir(dir string) (*ParsedSchema, error) {
	_, files, err := parseGoDir(dir)
	if err != nil {
		return nil, err
	}

	ps := &ParsedSchema{AutoIncrement: map[string]bool{}}
	nodes := map[string]*entNode{}

	node := func(name string) *entNode {
		n, ok := nodes[name]
		if !ok {
			n = &entNode{name: name}
			nodes[name] = n
		}
		return n
	}

	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			recv := receiverName(fd)
			if recv == "" {
				continue
			}
			n := node(recv)
			switch fd.Name.Name {
			case "Fields":
				n.parseFields(fd)
			case "Edges":
				n.parseEdges(fd)
			case "Annotations":
				n.parseAnnotations(fd)
			case "Mixin":
				n.warnings = append(n.warnings, fmt.Sprintf("ent: mixin() on %s is not resolved statically; its fields are missing", recv))
			}
		}
	}

	order := make([]string, 0, len(nodes))
	for name := range nodes {
		order = append(order, name)
	}
	sort.Strings(order)

	for _, name := range order {
		n := nodes[name]
		n.built = n.buildTable(ps)
		ps.Tables = append(ps.Tables, n.built)
	}
	resolveEntRelations(ps, nodes, order)

	for _, name := range order {
		ps.Warnings = append(ps.Warnings, nodes[name].warnings...)
	}
	return ps, nil
}

func receiverName(f *ast.FuncDecl) string {
	if f.Recv == nil || len(f.Recv.List) != 1 || f.Name == nil {
		return ""
	}
	typ := f.Recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// returnElements 提取方法体最后一条 return 语句里的复合字面量元素。
func returnElements(f *ast.FuncDecl) ([]ast.Expr, bool) {
	if f.Body == nil || len(f.Body.List) == 0 {
		return nil, false
	}
	ret, ok := f.Body.List[len(f.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil, false
	}
	lit, ok := ret.Results[0].(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	return lit.Elts, true
}

func (n *entNode) parseFields(f *ast.FuncDecl) {
	elts, ok := returnElements(f)
	if !ok {
		n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Fields(): unsupported return form, skipped", n.name))
		return
	}
	for _, e := range elts {
		base, methods := splitCallChain(e)
		pkg, kind := baseCallPkgFn(base)
		if pkg != "field" || kind == "" {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Fields(): unsupported field expression, skipped", n.name))
			continue
		}
		name, ok := litString(base, 0)
		if !ok {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Fields(): field.%s with non-literal name, skipped", n.name, kind))
			continue
		}

		raw, known := entFieldSQLTypes[kind]
		if !known {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: unknown field type %q, mapped to varchar(255)", n.name, name, kind))
			raw = "varchar(255)"
		}
		nullable := false
		if _, ok := methods["Optional"]; ok {
			nullable = true
		}
		if _, ok := methods["Nillable"]; ok {
			nullable = true
		}
		switch kind {
		case "String":
			if l, ok := litInt(methods["MaxLen"], 0); ok && l > 0 {
				raw = fmt.Sprintf("varchar(%d)", l)
			}
		case "Enum":
			if vals := stringArgs(methods["Values"]); len(vals) > 0 {
				raw = "enum(" + strings.Join(sqlQuoteEach(vals), ",") + ")"
			}
		}
		if _, ok := methods["GoType"]; ok {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: GoType() override is ignored", n.name, name))
		}
		if _, ok := methods["SchemaType"]; ok {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: SchemaType() override is ignored, default type used", n.name, name))
		}

		col := newColumn(name, raw, nullable)
		if c, ok := litString(methods["Comment"], 0); ok {
			col.SetComment(c)
		}
		if k, ok := litString(methods["StorageKey"], 0); ok {
			col.Name = k
		}
		n.fields = append(n.fields, col)
	}
}

func (n *entNode) parseEdges(f *ast.FuncDecl) {
	elts, ok := returnElements(f)
	if !ok {
		n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Edges(): unsupported return form, skipped", n.name))
		return
	}
	for _, e := range elts {
		base, methods := splitCallChain(e)
		pkg, kind := baseCallPkgFn(base)
		if pkg != "edge" || (kind != "To" && kind != "From") {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Edges(): unsupported edge expression, skipped", n.name))
			continue
		}
		var pairs [][2]string // {label, target}
		if kind == "From" {
			// edge.From("owner").Type(User.Type).Ref("pets") — 目标在 .Type() 调用里。
			label, ok := litString(base, 0)
			if !ok {
				n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Edges(): non-literal edge label, skipped", n.name))
				continue
			}
			target, sel := "", ""
			if tc := methods["Type"]; tc != nil && len(tc.Args) > 0 {
				target, sel = selectorName(tc.Args[0])
			}
			if sel != "Type" || target == "" {
				n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: unsupported edge target, skipped", n.name, label))
				continue
			}
			pairs = [][2]string{{label, target}}
		} else {
			// edge.To("pets", Pet.Type, "toys", Toy.Type ...) — 标签与类型成对出现。
			for i := 0; i+1 < len(base.Args); i += 2 {
				label, ok := litString(base, i)
				if !ok {
					n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Edges(): non-literal edge label, skipped", n.name))
					continue
				}
				target, sel := selectorName(base.Args[i+1])
				if sel != "Type" || target == "" {
					n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: unsupported edge target, skipped", n.name, label))
					continue
				}
				pairs = append(pairs, [2]string{label, target})
			}
		}
		for _, p := range pairs {
			ed := &entEdgeInfo{owner: n.name, label: p[0], target: p[1], dir: strings.ToLower(kind)}
			if r, ok := litString(methods["Ref"], 0); ok {
				ed.ref = r
			}
			if _, ok := methods["Unique"]; ok {
				ed.unique = true
			}
			if _, ok := methods["Required"]; ok {
				ed.required = true
			}
			if v, ok := litString(methods["Via"], 0); ok {
				ed.via = v
			}
			n.edges = append(n.edges, ed)
		}
	}
}

func (n *entNode) parseAnnotations(f *ast.FuncDecl) {
	elts, ok := returnElements(f)
	if !ok {
		return
	}
	for _, e := range elts {
		if u, isUnary := e.(*ast.UnaryExpr); isUnary && u.Op == token.AND {
			e = u.X
		}
		lit, ok := e.(*ast.CompositeLit)
		if !ok {
			continue
		}
		if sel, ok := lit.Type.(*ast.SelectorExpr); ok && sel.Sel.Name != "Annotation" {
			n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.Annotations(): %s ignored", n.name, sel.Sel.Name))
			continue
		}
		for _, kv := range lit.Elts {
			kvExpr, ok := kv.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kvExpr.Key.(*ast.Ident)
			if !ok || key.Name != "Table" {
				continue
			}
			if bl, ok := kvExpr.Value.(*ast.BasicLit); ok && bl.Kind == token.STRING {
				if v, err := strconv.Unquote(bl.Value); err == nil {
					n.table = v
				}
			}
		}
	}
}

// buildTable 生成节点表:合成 id 主键 + 显式字段列。
func (n *entNode) buildTable(ps *ParsedSchema) *schema.Table {
	t := &schema.Table{Name: n.tableName()}
	id := newColumn("id", "bigint", false)
	t.Columns = append(t.Columns, id)
	ps.markAutoIncrement(t.Name, "id")
	setPK(t, id)
	t.Columns = append(t.Columns, n.fields...)
	return t
}

func (n *entNode) tableName() string {
	if n.table != "" {
		return n.table
	}
	return inflect.Underscore(inflect.Pluralize(n.name))
}

func setPK(t *schema.Table, cols ...*schema.Column) {
	idx := &schema.Index{Table: t, Name: "PRIMARY"}
	for _, c := range cols {
		idx.Parts = append(idx.Parts, &schema.IndexPart{C: c})
	}
	t.PrimaryKey = idx
}

// resolveEntRelations 把 edge 定义还原为外键列与 m2m 中间表:
//   - To/From 成对且 From 侧 Unique → From 表持外键列(O2O/O2M);
//   - From 侧非 Unique,或 To 侧未配对且非 Unique → m2m 中间表(IsJoinTable);
//   - 未配对的 Unique To → To 表自身持外键列。
func resolveEntRelations(ps *ParsedSchema, nodes map[string]*entNode, order []string) {
	findFrom := func(targetNode string, e *entEdgeInfo) *entEdgeInfo {
		n := nodes[targetNode]
		if n == nil {
			return nil
		}
		for _, f := range n.edges {
			if f.dir == "from" && !f.matched && f.ref == e.label && f.target == e.owner {
				return f
			}
		}
		return nil
	}

	for _, name := range order {
		n := nodes[name]
		for _, e := range n.edges {
			if e.dir != "to" {
				continue
			}
			target := nodes[e.target]
			if target == nil {
				n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: edge target %q not found in schema dir, skipped", n.name, e.label, e.target))
				continue
			}
			f := findFrom(e.target, e)
			switch {
			case f != nil && f.unique:
				f.matched, e.matched = true, true
				addEntFK(target, n, f)
			case f != nil:
				f.matched, e.matched = true, true
				addEntJoin(ps, n, target, e.via, e.label)
			case e.unique:
				e.matched = true
				addEntFK(n, target, e)
			default:
				e.matched = true
				addEntJoin(ps, target, n, e.via, e.label)
			}
		}
	}
	// 仍未配对的 From 边:Unique → 本表持外键;非 Unique → 中间表。
	for _, name := range order {
		n := nodes[name]
		for _, g := range n.edges {
			if g.dir != "from" || g.matched {
				continue
			}
			target := nodes[g.target]
			if target == nil {
				n.warnings = append(n.warnings, fmt.Sprintf("ent: %s.%s: edge target %q not found in schema dir, skipped", n.name, g.label, g.target))
				continue
			}
			g.matched = true
			if g.unique {
				addEntFK(n, target, g)
			} else {
				label := g.ref
				if label == "" {
					label = g.label
				}
				addEntJoin(ps, target, n, g.via, label)
			}
		}
	}
}

// addEntFK 在 child 表上补外键列(引用 parent 表的 id)。
func addEntFK(child, parent *entNode, e *entEdgeInfo) {
	colName := inflect.Underscore(e.label) + "_id"
	var col *schema.Column
	for _, c := range child.built.Columns {
		if c.Name == colName {
			col = c
			break
		}
	}
	if col == nil {
		col = newColumn(colName, "bigint", !e.required)
		child.built.Columns = append(child.built.Columns, col)
	}
	child.built.ForeignKeys = append(child.built.ForeignKeys, &schema.ForeignKey{
		Symbol:   fmt.Sprintf("fk_%s_%s", child.built.Name, colName),
		Columns:  []*schema.Column{col},
		RefTable: parent.built,
		OnDelete: schema.Cascade,
	})
}

// addEntJoin 为 owner.toLabel → other 的 m2m 关系生成中间表。
// 中间表两条单列外键 + 双列主键,恰好满足 IsJoinTable 判定。
func addEntJoin(ps *ParsedSchema, owner, other *entNode, via, label string) {
	name := via
	if name == "" {
		name = inflect.Underscore(owner.name) + "_" + label
	}
	for _, t := range ps.Tables {
		if t.Name == name {
			return
		}
	}
	a := newColumn(inflect.Underscore(inflect.Singularize(owner.name))+"_id", "bigint", false)
	bName := inflect.Underscore(inflect.Singularize(other.name)) + "_id"
	if bName == a.Name {
		// 自引用 m2m(如 User.friends):用边标签的单数作第二列名,避免重名。
		bName = inflect.Underscore(inflect.Singularize(label)) + "_id"
		if bName == a.Name {
			bName = a.Name + "_2"
		}
	}
	b := newColumn(bName, "bigint", false)
	jt := &schema.Table{Name: name, Columns: []*schema.Column{a, b}}
	setPK(jt, a, b)
	jt.ForeignKeys = []*schema.ForeignKey{
		{Symbol: "fk_" + name + "_" + a.Name, Columns: []*schema.Column{a}, RefTable: owner.built, OnDelete: schema.Cascade},
		{Symbol: "fk_" + name + "_" + b.Name, Columns: []*schema.Column{b}, RefTable: other.built, OnDelete: schema.Cascade},
	}
	ps.Tables = append(ps.Tables, jt)
}

// stringArgs 收集调用的全部字符串字面量参数。
func stringArgs(ce *ast.CallExpr) []string {
	if ce == nil {
		return nil
	}
	var out []string
	for _, a := range ce.Args {
		if bl, ok := a.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			if v, err := strconv.Unquote(bl.Value); err == nil {
				out = append(out, v)
			}
		}
	}
	return out
}

func sqlQuoteEach(vals []string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, "'"+strings.ReplaceAll(v, "'", "''")+"'")
	}
	return out
}
