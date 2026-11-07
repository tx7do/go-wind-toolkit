package frontendgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// errorEntry 错误词表条目(protoc-gen-wind-errors 的 _errors_i18n.<lang>.json)。
type errorEntry struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// canonicalLocale 把词表的语言名归一到前端 locale 目录名:
// 插件的 i18n_langs 常给 zh/en,前端目录惯用 zh-CN/en-US。
func canonicalLocale(lang string) string {
	switch strings.ToLower(strings.ReplaceAll(lang, "_", "-")) {
	case "zh", "zh-cn", "zh-hans":
		return "zh-CN"
	case "en", "en-us":
		return "en-US"
	default:
		return lang
	}
}

// localeDirForFramework 返回某框架 locale 文件的目录前缀(相对 src)。
func localeDirForFramework(fw Framework, locale string) string {
	if fw == FrameworkVueVben {
		return "locales/langs/" + locale
	}
	return "locales/" + locale
}

// generateErrorI18N 合并错误词表文件,按语言产出 errors.json——
// 错误文案后端不出网,前端按 reason 本地化(与 WindError 契约配套)。
// 同一语言的多个 proto 词表按键合并,reason 冲突先到先得(生成器产出的
// reason 全局唯一,冲突仅防手改)。词表缺失/解析失败静默跳过:错误词表是
// 增强产物,不应拖垮主生成。
func generateErrorI18N(opts Options) []GeneratedFile {
	if len(opts.ErrorI18NFiles) == 0 {
		return nil
	}

	merged := map[string]map[string]string{} // locale -> reason -> message
	for _, file := range opts.ErrorI18NFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var entries map[string]errorEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			continue
		}
		locale := canonicalLocale(extractI18NLang(file))
		if locale == "" {
			continue
		}
		if merged[locale] == nil {
			merged[locale] = map[string]string{}
		}
		for reason, e := range entries {
			if _, dup := merged[locale][reason]; !dup {
				merged[locale][reason] = e.Message
			}
		}
	}

	var locales []string
	for locale := range merged {
		locales = append(locales, locale)
	}
	sort.Strings(locales)

	files := make([]GeneratedFile, 0, len(locales))
	for _, locale := range locales {
		reasons := merged[locale]
		keys := make([]string, 0, len(reasons))
		for k := range reasons {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var b strings.Builder
		b.WriteString("{\n")
		for i, k := range keys {
			b.WriteString("  \"")
			b.WriteString(jsonEscape(k))
			b.WriteString("\": \"")
			b.WriteString(jsonEscape(reasons[k]))
			b.WriteString("\"")
			if i < len(keys)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("}\n")

		files = append(files, GeneratedFile{
			Path:        localeDirForFramework(opts.Framework, locale) + "/errors.json",
			Content:     b.String(),
			Description: "wind 错误文案词表(按 reason 本地化,与后端 WindError 契约配套)",
			Type:        TypeLocale,
		})
	}
	return files
}

// extractI18NLang 从词表文件名提取语言段:
// <proto>_errors_i18n.<lang>.json → <lang>;不符合命名约定的返回空串。
func extractI18NLang(file string) string {
	base := filepath.Base(file)
	base = strings.TrimSuffix(base, ".json")
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}
