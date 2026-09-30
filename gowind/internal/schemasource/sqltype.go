package schemasource

import (
	"regexp"
	"strings"
)

// SQLType 是拆开后的 SQL 列类型文本。
type SQLType struct {
	Base     string // 大写基名,如 "BIGINT";已去掉精度与 unsigned/zerofill 等修饰词
	Args     string // 括号内原文,如 "20"、"10,2";无括号时为 ""
	Unsigned bool   // 是否声明为 UNSIGNED
}

// ParseSQLType 拆解列类型文本。
//
// 顺序很关键:先按 '(' 截断再找 unsigned 的写法,会让 `bigint(20) unsigned` 的
// unsigned 连同精度一起被丢弃,映射表里所有 *_UNSIGNED 键因此永不可达。
func ParseSQLType(raw string) SQLType {
	base, rest := raw, ""
	if open := strings.IndexByte(raw, '('); open != -1 {
		base, rest = raw[:open], raw[open:]
	}

	// 括号里是精度或枚举字面量:既不属于类型名,也不参与 unsigned 判定。
	// 修饰词只可能跟在类型名后(INT UNSIGNED)或右括号后(BIGINT(20) UNSIGNED)。
	args, tail := "", rest
	if close := strings.LastIndexByte(rest, ')'); close >= 0 {
		args, tail = rest[1:close], rest[close+1:]
	}

	return SQLType{
		Base:     normalizeTypeWords(base),
		Args:     args,
		Unsigned: unsignedWordRe.MatchString(base) || unsignedWordRe.MatchString(tail),
	}
}

// normalizeTypeWords 去掉修饰词与多余空白,得到用作映射键的大写类型名。
// 多词类型名(TIMESTAMP WITH TIME ZONE)按单个空格归一后原样保留。
func normalizeTypeWords(s string) string {
	s = zeroFillWordRe.ReplaceAllString(s, "")
	s = unsignedWordRe.ReplaceAllString(s, "")
	return strings.ToUpper(spaceRe.ReplaceAllString(strings.TrimSpace(s), " "))
}

// Key 返回类型映射表的键,无符号类型带 " UNSIGNED" 后缀。
func (t SQLType) Key() string {
	if t.Unsigned {
		return t.Base + " UNSIGNED"
	}
	return t.Base
}

// ColumnIsUnsigned 报告 DDL 文本中 tableName 表的 colName 列是否声明为 UNSIGNED。
//
// ddl_parser 不会把 unsigned 带进 ColumnDef.Type(`INT UNSIGNED` 只给出 "int"),
// 只能回原文确认。原文匹配必须同时锚定三件事:落在该表的定义体内、列名成词、
// 列名后紧跟该列自己的类型。少任何一条都会误判——整文件子串搜索曾让列 id
// 命中 `uid bigint unsigned` 与另一张表里的同名列,把有符号列判成无符号。
func ColumnIsUnsigned(tableName, colName, colType, sqlContent string) bool {
	name := strings.ToLower(strings.TrimSpace(colName))
	if name == "" {
		return false
	}

	text := strings.ToLower(blankDDLCommentsAndLiterals(sqlContent))
	body, ok := ddlTableBody(text, strings.ToLower(strings.TrimSpace(tableName)))
	if !ok {
		// 表体定位不到(非 CREATE TABLE 写法)时退回全文:锚定仍然生效,只是失去表的边界。
		body = text
	}

	typeBase := strings.ToLower(ParseSQLType(colType).Base)

	for from := 0; ; {
		at := strings.Index(body[from:], name)
		if at == -1 {
			return false
		}
		at += from

		end, ok := columnDefStart(body, at+len(name), name, typeBase)
		if ok && unsignedWordRe.MatchString(body[end:columnDefEnd(body, end)]) {
			return true
		}
		from = at + len(name)
	}
}

// columnDefStart 校验 at 处的一次 colName 出现是否就是该列的定义起点,
// 返回其类型文本的起始下标。要求:列名成词(允许被引号包裹)、其后紧跟类型基名。
func columnDefStart(text string, at int, name, typeBase string) (int, bool) {
	if at >= len(text) {
		return 0, false
	}
	if isIdentByte(text[at]) {
		// 命中了 uid、signed_id 这类更长的标识符
		return 0, false
	}
	if before := at - len(name) - 1; before >= 0 && isIdentByte(text[before]) {
		return 0, false
	}

	pos := at
	if text[pos] == '`' || text[pos] == '"' || text[pos] == ']' {
		pos++
	}
	for pos < len(text) && isSpaceByte(text[pos]) {
		pos++
	}
	if pos >= len(text) {
		return 0, false
	}
	if typeBase == "" {
		return pos, isIdentByte(text[pos])
	}
	if !strings.HasPrefix(text[pos:], typeBase) {
		return 0, false
	}
	if after := pos + len(typeBase); after < len(text) && isIdentByte(text[after]) {
		return 0, false
	}

	return pos, true
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// columnDefEnd 返回 from 处列类型定义的结束下标:括号深度为 0 时遇到逗号、换行,
// 或遇到闭合表体的右括号即止。
func columnDefEnd(text string, from int) int {
	depth := 0
	for i := from; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return i
			}
			depth--
		case ',', '\n':
			if depth == 0 {
				return i
			}
		}
	}

	return len(text)
}

// ddlTableBody 取出 CREATE TABLE 语句的定义体(不含最外层括号)。
// sql 需已归一为小写;表名按末端名比较,故 public.users 与 `users` 都能匹配 users。
// 同名表重复出现时取第一处。
func ddlTableBody(sql, tableName string) (string, bool) {
	if tableName == "" {
		return "", false
	}

	for _, m := range createTableRe.FindAllStringSubmatchIndex(sql, -1) {
		if trimTableName(sql[m[2]:m[3]]) != tableName {
			continue
		}
		open := m[1] - 1
		depth := 0
		for i := open; i < len(sql); i++ {
			switch sql[i] {
			case '(':
				depth++
			case ')':
				if depth--; depth == 0 {
					return sql[open+1 : i], true
				}
			}
		}
	}

	return "", false
}

// trimTableName 去掉表名的 schema 限定与引号包裹,只保留末端名。
func trimTableName(raw string) string {
	if idx := strings.LastIndexByte(raw, '.'); idx != -1 {
		raw = raw[idx+1:]
	}
	return strings.Trim(raw, "\"`[]")
}

var (
	unsignedWordRe = regexp.MustCompile(`(?i)\bunsigned\b`)
	zeroFillWordRe = regexp.MustCompile(`(?i)\bzerofill\b`)
	spaceRe        = regexp.MustCompile(`\s+`)

	// createTableRe 匹配 `CREATE TABLE [IF NOT EXISTS] <表名>(`,表名捕获组允许
	// 反引号/双引号/方括号包裹与 schema 限定。
	createTableRe = regexp.MustCompile(
		`(?is)\bcreate\s+table\s+(?:if\s+not\s+exists\s+)?` +
			`((?:"[^"]*"|` + "`" + `[^` + "`" + `]*` + "`" + `|\[[^\]]*\]|[^\s(]+)` +
			`(?:\.(?:"[^"]*"|` + "`" + `[^` + "`" + `]*` + "`" + `|\[[^\]]*\]|[^\s(]+)*)*)\s*\(`,
	)
)

// blankDDLCommentsAndLiterals 把 SQL 注释与字符串字面量的内容替换成空格,长度不变。
// 列名可能恰好出现在 COMMENT '...' 或 `-- ...` 里并带上 "bigint unsigned" 之类的字样,
// 这类文本必须先屏蔽,否则词边界 + 类型锚定也拦不住它。
func blankDDLCommentsAndLiterals(sql string) string {
	out := []byte(sql)

	blank := func(from, to int) {
		if from < 0 {
			from = 0
		}
		if to > len(out) {
			to = len(out)
		}
		for i := from; i < to; i++ {
			out[i] = ' '
		}
	}

	for i := 0; i < len(out); {
		switch {
		case out[i] == '\'':
			end := i + 1
			for end < len(out) {
				if out[end] != '\'' {
					end++
					continue
				}
				if end+1 < len(out) && out[end+1] == '\'' {
					end += 2 // 字面量内的 '' 是引号转义,不是结束
					continue
				}
				break
			}
			blank(i+1, end)
			i = end + 1

		case out[i] == '/' && i+1 < len(out) && out[i+1] == '*':
			end := indexOf(out, i+2, "*/")
			if end == -1 {
				blank(i+2, len(out))
				return string(out)
			}
			blank(i+2, end)
			i = end + 2

		case out[i] == '#', out[i] == '-' && i+1 < len(out) && out[i+1] == '-':
			end := indexOfByte(out, i, '\n')
			if end == -1 {
				blank(i, len(out))
				return string(out)
			}
			blank(i, end)
			i = end

		default:
			i++
		}
	}

	return string(out)
}

func indexOf(b []byte, from int, sep string) int {
	if idx := strings.Index(string(b[from:]), sep); idx != -1 {
		return from + idx
	}
	return -1
}

func indexOfByte(b []byte, from int, c byte) int {
	for i := from; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// isIdentByte 判断字节是否属于标识符字符;>=0x80 归入标识符,避免在多字节字符中间切开。
func isIdentByte(c byte) bool {
	return c >= 0x80 || c == '_' || c == '$' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')
}
