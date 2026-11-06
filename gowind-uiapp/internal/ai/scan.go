package ai

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 项目代码扫描的规模上限:防止把超大仓库整包塞进提示词。
const (
	maxReviewFiles      = 200
	maxReviewFileBytes  = 64 << 10 // 单文件 64KB,超过跳过
	maxReviewTotalBytes = 1 << 20  // 总量 1MB,超过截断
)

// reviewSkipDirs 扫描时跳过的目录名(生成物、依赖与产物目录)。
var reviewSkipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".gowind":      true,
	"dist":         true,
	"build":        true,
}

// CodeFileInfo 待审查代码文件
type CodeFileInfo struct {
	Path string `json:"path"` // 相对项目根,统一正斜杠
	Size int64  `json:"size"`
}

// ReviewScanResult 项目代码扫描结果
type ReviewScanResult struct {
	Success  bool           `json:"success"`
	Files    []CodeFileInfo `json:"files"`
	Warnings []string       `json:"warnings,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// ScanProjectCode 扫描项目内可审查的代码文件:
// app/ 下的 Go 源码(排除 _test.go 与 protoc 生成物)与 api/protos/ 下的 proto。
// 超过上限时截断并记入 warnings;结果按路径排序。
func ScanProjectCode(projectRoot string) (*ReviewScanResult, error) {
	res := &ReviewScanResult{Success: true}
	var totalBytes int64
	truncated := false

	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单个不可读路径跳过,不中断整体扫描
		}
		rel, rerr := filepath.Rel(projectRoot, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel != "." && reviewSkipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		if !reviewableRelPath(rel) {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil || info.Size() > maxReviewFileBytes {
			res.Warnings = append(res.Warnings, fmt.Sprintf("文件 %s 超过单文件上限(%dKB),已跳过", rel, maxReviewFileBytes/1024))
			return nil
		}
		if len(res.Files) >= maxReviewFiles || totalBytes+info.Size() > maxReviewTotalBytes {
			truncated = true
			return fs.SkipAll
		}

		res.Files = append(res.Files, CodeFileInfo{Path: rel, Size: info.Size()})
		totalBytes += info.Size()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("扫描项目代码失败: %w", err)
	}
	if truncated {
		res.Warnings = append(res.Warnings, fmt.Sprintf("可审查文件超过上限(%d 个 / %dMB),结果已截断", maxReviewFiles, maxReviewTotalBytes/1024/1024))
	}

	sort.Slice(res.Files, func(i, j int) bool { return res.Files[i].Path < res.Files[j].Path })
	return res, nil
}

// reviewableRelPath 判断相对路径是否属于可审查范围:
// app/ 下的 .go(排除 _test.go、.pb.go)或 api/protos/ 下的 .proto。
func reviewableRelPath(rel string) bool {
	if strings.HasSuffix(rel, ".proto") {
		return strings.HasPrefix(rel, "api/protos/")
	}
	if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
		return false
	}
	if strings.HasSuffix(rel, ".pb.go") || strings.HasSuffix(rel, "_grpc.pb.go") {
		return false
	}
	return strings.HasPrefix(rel, "app/")
}

// ReadProjectFiles 读取选中的项目文件,返回 相对路径 -> 内容。
// 路径必须是项目内的可审查文件(防越出项目根读取任意文件);
// 单文件超限或读取失败跳过并记 warnings;总量超限截断。
func ReadProjectFiles(projectRoot string, relPaths []string) (map[string]string, []string, error) {
	if len(relPaths) == 0 {
		return nil, nil, fmt.Errorf("没有选择任何文件")
	}

	contents := make(map[string]string, len(relPaths))
	var warnings []string
	var totalBytes int64

	for _, rel := range relPaths {
		full, err := resolveProjectFile(projectRoot, rel)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		data, err := readFileCapped(full, maxReviewFileBytes)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("读取 %s 失败: %v", rel, err))
			continue
		}
		if totalBytes+int64(len(data)) > maxReviewTotalBytes {
			warnings = append(warnings, fmt.Sprintf("总量超过 %dMB,其后的文件未纳入审查", maxReviewTotalBytes/1024/1024))
			break
		}
		totalBytes += int64(len(data))
		contents[filepath.ToSlash(filepath.Clean(rel))] = string(data)
	}

	if len(contents) == 0 {
		return nil, warnings, fmt.Errorf("所选文件均不可读")
	}
	return contents, warnings, nil
}

// resolveProjectFile 校验 rel 是项目内的可审查文件并返回绝对路径。
func resolveProjectFile(projectRoot, rel string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel)))
	if cleaned == "" || cleaned == "." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("非法的文件路径: %s", rel)
	}
	if !reviewableRelPath(filepath.ToSlash(cleaned)) {
		return "", fmt.Errorf("文件不在可审查范围内: %s", rel)
	}
	full := filepath.Join(projectRoot, cleaned)
	// 双保险:回验 Rel 仍在项目根内。
	if back, err := filepath.Rel(projectRoot, full); err != nil || strings.HasPrefix(back, "..") {
		return "", fmt.Errorf("文件路径越出项目根: %s", rel)
	}
	return full, nil
}

// readFileCapped 读取至多 cap 字节的文件,超出报错。
func readFileCapped(path string, cap int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > cap {
		return nil, fmt.Errorf("文件超过 %dKB", cap/1024)
	}
	return os.ReadFile(path)
}
