package schemasource

import (
	"fmt"
	"strings"

	"ariga.io/atlas/sql/schema"
)

// BuildMySQLDDL 把解析产物还原为 MySQL 风格 CREATE TABLE 语句(每张表一条),
// 供 gorm rawsql 回转生成 DAO 使用。仅输出列、主键与注释——外键与索引对
// gorm gen 的模型/DAO 产物没有影响,且 rawsql 解析器对它们的兼容性无需依赖。
func BuildMySQLDDL(ps *ParsedSchema) []string {
	var out []string
	for _, t := range ps.Tables {
		var b strings.Builder
		fmt.Fprintf(&b, "CREATE TABLE `%s` (\n", t.Name)
		var parts []string
		for _, c := range t.Columns {
			parts = append(parts, "  "+buildMySQLColumnDDL(ps, t, c))
		}
		if t.PrimaryKey != nil && len(t.PrimaryKey.Parts) > 0 {
			cols := make([]string, 0, len(t.PrimaryKey.Parts))
			for _, p := range t.PrimaryKey.Parts {
				cols = append(cols, "`"+p.C.Name+"`")
			}
			parts = append(parts, "  PRIMARY KEY ("+strings.Join(cols, ", ")+")")
		}
		b.WriteString(strings.Join(parts, ",\n"))
		b.WriteString("\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
		for _, attr := range t.Attrs {
			if cm, ok := attr.(*schema.Comment); ok && cm.Text != "" {
				fmt.Fprintf(&b, " COMMENT=%s", sqlStringLiteral(cm.Text))
				break
			}
		}
		out = append(out, b.String())
	}
	return out
}

func buildMySQLColumnDDL(ps *ParsedSchema, t *schema.Table, c *schema.Column) string {
	var b strings.Builder
	fmt.Fprintf(&b, "`%s` %s", c.Name, c.Type.Raw)
	isPK := t.PrimaryKey != nil && len(t.PrimaryKey.Parts) == 1 && t.PrimaryKey.Parts[0].C == c
	if isPK && ps.AutoIncrement[incKey(t.Name, c.Name)] {
		b.WriteString(" NOT NULL AUTO_INCREMENT")
	} else if !c.Type.Null {
		b.WriteString(" NOT NULL")
	}
	if c.Default != nil {
		if expr := defaultExpr(c.Default); expr != "" {
			fmt.Fprintf(&b, " DEFAULT %s", expr)
		}
	}
	for _, attr := range c.Attrs {
		if cm, ok := attr.(*schema.Comment); ok && cm.Text != "" {
			fmt.Fprintf(&b, " COMMENT %s", sqlStringLiteral(cm.Text))
			break
		}
	}
	return b.String()
}

func defaultExpr(def schema.Expr) string {
	if def == nil {
		return ""
	}
	switch e := def.(type) {
	case *schema.RawExpr:
		return e.X
	case *schema.NamedDefault:
		return defaultExpr(e.Expr)
	case *schema.Literal:
		v := e.V
		switch {
		case isNumericLiteral(v):
			return v
		case strings.EqualFold(v, "CURRENT_TIMESTAMP"), strings.EqualFold(v, "NULL"):
			return strings.ToUpper(v)
		default:
			return sqlStringLiteral(v)
		}
	}
	return ""
}

func isNumericLiteral(v string) bool {
	if v == "" {
		return false
	}
	s := strings.TrimPrefix(strings.TrimPrefix(v, "-"), "+")
	isFloat := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == '.':
			if isFloat {
				return false
			}
			isFloat = true
		default:
			return false
		}
	}
	return true
}

func sqlStringLiteral(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}
