package generators

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/tx7do/go-utils/stringcase"
)

var funcMap template.FuncMap

func init() {
	funcMap = template.FuncMap{
		"newlineIf": newlineIf,
		"newline":   newline,

		"upper": strings.ToUpper, // 转换为大写
		"lower": strings.ToLower, // 转换为小写

		"camel":  stringcase.LowerCamelCase, // 转换为 camelCase
		"pascal": stringcase.ToPascalCase,   // 转换为 PascalCase
		"kebab":  stringcase.KebabCase,      // 转换为 kebab-case
		"snake":  stringcase.SnakeCase,      // 转换为 snake_case

		"renderImports":          renderImports,
		"renderFormalParameters": renderFormalParameters,
		"renderInParameters":     renderInParameters,

		"renderServiceName": renderServiceName,
		"renderRepoName":    renderRepoName,
		"renderServerName":  renderServerName,
		"apiPackageAlias":   apiPackageAlias,
	}
}

// getIndent 返回指定数量的制表符缩进字符串，便于在模板中复用。
func getIndent(tabs int) string {
	if tabs < 0 {
		tabs = 0
	}
	return strings.Repeat("\t", tabs)
}

// newlineIf 根据条件返回换行或空字符串，便于在模板中复用。
func newlineIf(condition bool) string {
	if condition {
		return "\n"
	}
	return ""
}

// newline 始终返回换行符，便于在模板中复用。
func newline() string {
	return "\n"
}

// renderImports 将 string 或 []string 转换为 Go import 列表的多行字符串。
// 支持空值跳过。
func renderImports(v any) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case []string:
		var sb strings.Builder
		for _, p := range t {
			if p == "" {
				continue
			}
			sb.WriteString("\t\"")
			sb.WriteString(p)
			sb.WriteString("\"\n")
		}
		return sb.String()
	case string:
		if t == "" {
			return ""
		}
		return "\t" + t + "\n"
	default:
		return ""
	}
}

// renderFormalParameters 接受 string / []string / []any，返回多行形式参数，
// 每行以制表符开头并以逗号和换行结束。例如:
//
//	\tname Type,\n
func renderFormalParameters(v any, tabs ...int) string {
	if v == nil {
		return ""
	}

	tabCount := 1
	if len(tabs) > 0 && tabs[0] >= 0 {
		tabCount = tabs[0]
	}
	indent := getIndent(tabCount)

	var sb strings.Builder
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s != "" {
			sb.WriteString(indent)
			sb.WriteString(s)
			sb.WriteString(",\n")
		}
	case []string:
		for _, item := range t {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			sb.WriteString(indent)
			sb.WriteString(item)
			sb.WriteString(",\n")
		}
	case []any:
		for _, x := range t {
			s := strings.TrimSpace(fmt.Sprint(x))
			if s == "" {
				continue
			}
			sb.WriteString(indent)
			sb.WriteString(s)
			sb.WriteString(",\n")
		}
	default:
		// 尝试处理其他 slice 类型（反射开销略高），否则返回空
		return ""
	}
	return sb.String()
}

// renderInParameters 接受 string / []string / []any，返回多行调用实参，
// 每行以制表符开头并以逗号和换行结束。例如:
//
//	\tparam,\n
func renderInParameters(v any, tabs ...int) string {
	if v == nil {
		return ""
	}

	tabCount := 1
	if len(tabs) > 0 && tabs[0] >= 0 {
		tabCount = tabs[0]
	}
	indent := getIndent(tabCount)

	var sb strings.Builder
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s != "" {
			sb.WriteString(indent)
			sb.WriteString(s)
			sb.WriteString(",\n")
		}
	case []string:
		for _, item := range t {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			sb.WriteString(indent)
			sb.WriteString(item)
			sb.WriteString(",\n")
		}
	case []any:
		for _, x := range t {
			s := strings.TrimSpace(fmt.Sprint(x))
			if s == "" {
				continue
			}
			sb.WriteString(indent)
			sb.WriteString(s)
			sb.WriteString(",\n")
		}
	default:
		return ""
	}
	return sb.String()
}

func renderServiceName(v any) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case string:
		return stringcase.ToPascalCase(t) + "Service"
	default:
		return ""
	}
}

func renderRepoName(v any) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case string:
		return stringcase.ToPascalCase(t) + "Repo"
	default:
		return ""
	}
}

func renderServerName(v any) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case string:
		return stringcase.ToPascalCase(t) + "Server"
	default:
		return ""
	}
}

// apiPackageAlias 根据 API 包名和版本号生成 import 别名，例如 "user" + "v1" → "userV1"
func apiPackageAlias(name string, version string) string {
	return stringcase.LowerCamelCase(name) + stringcase.UpperCamelCase(version)
}
