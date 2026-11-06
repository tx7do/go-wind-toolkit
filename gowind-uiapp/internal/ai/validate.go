package ai

import (
	"fmt"
	"strings"

	"github.com/tx7do/go-utils/ddl_parser"
)

// ValidateDDL 解析 DDL 文本,返回其中的表名清单(按出现顺序)。
// 解析失败或未解析出任何 CREATE TABLE 都视为无效。
// 注意:底层解析器对个别残缺语句可能跳过而非报错,因此这是"至少可用"级别的校验,
// 不是完整的 SQL 语法检查。
func ValidateDDL(sqlContent string) ([]string, error) {
	if strings.TrimSpace(sqlContent) == "" {
		return nil, fmt.Errorf("DDL 内容为空")
	}

	tables, err := ddlparser.ParseCreateTables(sqlContent)
	if err != nil {
		return nil, fmt.Errorf("DDL 解析失败: %w", err)
	}

	names := make([]string, 0, len(tables))
	for _, t := range tables {
		if name := strings.TrimSpace(t.Name); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("DDL 中没有可识别的 CREATE TABLE 语句")
	}
	return names, nil
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
