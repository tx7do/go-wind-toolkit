package ai

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WalkPlanFiles 遍历生成预览树,返回按路径排序的相对路径清单(正斜杠),
// 并对照 projectRoot 标注 isNew:项目内不存在 = 新增,已存在 = 覆盖。
func WalkPlanFiles(planDir, projectRoot string) ([]BackendPlanFile, error) {
	var files []BackendPlanFile
	err := filepath.WalkDir(planDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(planDir, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		size := int64(0)
		if info, ierr := d.Info(); ierr == nil {
			size = info.Size()
		}
		_, statErr := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(rel)))
		files = append(files, BackendPlanFile{Path: rel, Size: size, IsNew: statErr != nil})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历预览目录失败: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// CopyTree 把 srcRoot 整树拷贝到 dstRoot(目录 0o755、文件 0o644,同名覆盖)。
// 相对路径逐个做防穿越校验,任何失败即中止返回错误(已拷贝的文件保留,
// 调用方负责向用户说明部分写入状态)。
func CopyTree(dstRoot, srcRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(srcRoot, path)
		if rerr != nil {
			return rerr
		}
		if rel == "." {
			return nil
		}
		target, verr := safeJoinUnder(dstRoot, rel)
		if verr != nil {
			return verr
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// ReadPlanFile 读取预览树内单个文件,relPath 须落在计划目录内(防穿越)。
func ReadPlanFile(planDir, relPath string) (string, error) {
	p, err := safeJoinUnder(planDir, relPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("读取预览文件失败: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("目标是目录: %s", relPath)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("读取预览文件失败: %w", err)
	}
	return string(data), nil
}

// NewPlanID 生成随机计划 ID(16 位 hex)。
func NewPlanID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败极罕见;退化为时间戳仍保证非空可比较
		return fmt.Sprintf("plan-%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// safeJoinUnder 校验 rel 是 root 下的单个相对路径(拒绝空/./..、绝对路径、
// 任何 ".." 前缀)并返回绝对路径;双保险做 Rel 回验。
func safeJoinUnder(root, rel string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel)))
	if cleaned == "" || cleaned == "." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("非法路径: %s", rel)
	}
	p := filepath.Join(root, cleaned)
	if back, err := filepath.Rel(root, p); err != nil || strings.HasPrefix(back, "..") {
		return "", fmt.Errorf("路径越出根目录: %s", rel)
	}
	return p, nil
}
