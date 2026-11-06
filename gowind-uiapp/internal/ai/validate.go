package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	// 独立 parser 模块要求显式注入表达式求值驱动,否则 New() 直接 panic
	// (见 https://github.com/pingcap/parser/issues/43)。
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
)

// DDLDiagnostic 单条 DDL 语法诊断。Line 从 1 起;Column 是解析器的字节近似列
// (CJK 字符多的行会偏几个字符),用作编辑器标注足够。
type DDLDiagnostic struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// tidb 解析器的错误是纯文本,位置只存在于消息里,两种稳定格式:
//   line %d column %d near "%s"...   (语法错, Scanner.Errorf)
//   near '%s' at line %d             (词法错, ParseErrorWith)
var (
	reGrammarLineCol = regexp.MustCompile(`line (\d+) column (\d+)`)
	reTokenAtLine    = regexp.MustCompile(`at line (\d+)`)
)

// extractDiagnostic 从解析错误的文本里提取行/列,提取不到则 0/0 仅留消息。
func extractDiagnostic(err error) DDLDiagnostic {
	msg := err.Error()
	diag := DDLDiagnostic{Message: msg}
	if m := reGrammarLineCol.FindStringSubmatch(msg); m != nil {
		diag.Line, _ = strconv.Atoi(m[1])
		diag.Column, _ = strconv.Atoi(m[2])
	} else if m := reTokenAtLine.FindStringSubmatch(msg); m != nil {
		diag.Line, _ = strconv.Atoi(m[1])
	}
	return diag
}

// ValidateDDLDetailed 用 tidb parser 做真语法校验,返回表名清单与诊断。
// 解析器实例每次新建(New 便宜且实例非并发安全),免锁;空 charset 走默认
// utf8mb4。语法错只回首个错误——够修复重试循环逐条收敛。
func ValidateDDLDetailed(sqlContent string) ([]string, []DDLDiagnostic, error) {
	if strings.TrimSpace(sqlContent) == "" {
		return nil, nil, fmt.Errorf("DDL 内容为空")
	}

	p := parser.New()
	stmts, _, err := p.Parse(sqlContent, "", "")
	if err != nil {
		diag := extractDiagnostic(err)
		diags := []DDLDiagnostic{diag}
		return nil, diags, fmt.Errorf("DDL 语法错误(第 %d 行): %s", diag.Line, diag.Message)
	}

	names := make([]string, 0, len(stmts))
	for _, stmt := range stmts {
		if ct, ok := stmt.(*ast.CreateTableStmt); ok {
			if name := strings.TrimSpace(ct.Table.Name.O); name != "" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("DDL 中没有可识别的 CREATE TABLE 语句")
	}
	return names, nil, nil
}

// ValidateDDL 解析 DDL 文本,返回其中的表名清单(按出现顺序)。
// 解析失败或未解析出任何 CREATE TABLE 都视为无效。ValidateDDLDetailed 的薄包装。
func ValidateDDL(sqlContent string) ([]string, error) {
	tables, _, err := ValidateDDLDetailed(sqlContent)
	return tables, err
}

// normalizeTableName 表名归一化:去空白与包裹符、统一小写,用于
// 划分结果与 DDL 表清单的交叉比对(LLM 输出的表名大小写/反引号不可信)。
func normalizeTableName(name string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(name), "`\"'"))
}

// matchTableName 在 ddlTables 中查找 name 的归一化匹配,返回 DDL 里的原始拼写。
// 生成代码以 DDL 原名为准,避免 LLM 改写大小写导致建表/实体对不上。
func matchTableName(name string, ddlTables []string) (string, bool) {
	target := normalizeTableName(name)
	if target == "" {
		return "", false
	}
	for _, t := range ddlTables {
		if normalizeTableName(t) == target {
			return t, true
		}
	}
	return "", false
}
