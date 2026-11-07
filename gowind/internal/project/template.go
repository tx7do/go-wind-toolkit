// template.go 判定脚手架模板来源:本地目录 vs git 仓库。
package project

import (
	"os"
	"path/filepath"
	"strings"
)

// isLocalTemplateDir 报告 layout 是否指向本地模板目录。
// 判定条件:像路径(绝对、./ ../、含路径分隔符)且存在且是目录——
// 二者都满足才算本地模板;裸名字仍视为 git 仓库 URL,避免歧义。
func isLocalTemplateDir(layout string) bool {
	if layout == "" {
		return false
	}
	looksLikePath := filepath.IsAbs(layout) ||
		layout == "." || layout == ".." ||
		strings.HasPrefix(layout, "./") || strings.HasPrefix(layout, ".\\") ||
		strings.HasPrefix(layout, "../") || strings.HasPrefix(layout, "..\\") ||
		strings.HasPrefix(layout, "~") ||
		strings.ContainsAny(layout, "/\\")
	if !looksLikePath {
		return false
	}
	info, err := os.Stat(layout)
	return err == nil && info.IsDir()
}
