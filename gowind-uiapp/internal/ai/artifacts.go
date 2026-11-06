package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// aiArtifactDirName 项目内 AI 产物目录(相对项目根)。
const aiArtifactDirName = ".gowind/ai"

// ArtifactKind AI 产物的种类。requirements/ddl/partitions 是固定文件名、
// 后写覆盖前写;review 是审查报告,按时间戳命名、持续累积。
type ArtifactKind string

const (
	ArtifactRequirements ArtifactKind = "requirements"
	ArtifactDDL          ArtifactKind = "ddl"
	ArtifactPartitions   ArtifactKind = "partitions"
	ArtifactReview       ArtifactKind = "review"
)

// artifactFiles 固定文件名 kind -> 文件名。review 走时间戳命名,不在表内。
var artifactFiles = map[ArtifactKind]string{
	ArtifactRequirements: "requirements.md",
	ArtifactDDL:          "ddl.sql",
	ArtifactPartitions:   "partitions.json",
}

// ArtifactSaveResult 保存产物结果
type ArtifactSaveResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ReviewFile 一份已落盘的审查报告
type ReviewFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	ModTime string `json:"modTime"`
}

// Artifacts 项目内已保存的 AI 产物快照
type Artifacts struct {
	Requirements string       `json:"requirements"`
	DDL          string       `json:"ddl"`
	Partitions   string       `json:"partitions"`
	Reviews      []ReviewFile `json:"reviews,omitempty"`
}

// artifactDir 返回项目内的 AI 产物目录绝对路径。
func artifactDir(projectRoot string) string {
	return filepath.Join(projectRoot, aiArtifactDirName)
}

// SaveArtifact 把产物写入项目 .gowind/ai/ 目录,返回落盘路径。
// kind 必须是 ArtifactKind 之一;review 用时间戳命名,其余为固定文件名。
func SaveArtifact(projectRoot string, kind ArtifactKind, content string) (string, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", fmt.Errorf("项目根目录为空")
	}
	var name string
	if kind == ArtifactReview {
		name = "review-" + time.Now().Format("20060102-150405.000") + ".md"
	} else {
		file, ok := artifactFiles[kind]
		if !ok {
			return "", fmt.Errorf("未知的产物类型: %s", kind)
		}
		name = file
	}

	dir := artifactDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建产物目录失败: %w", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("写入产物失败: %w", err)
	}
	return path, nil
}

// SavePartitionsArtifact 把划分结果序列化为 JSON 后落盘。
func SavePartitionsArtifact(projectRoot string, partitions []MicroservicePartition) (string, error) {
	data, err := json.MarshalIndent(partitions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化划分结果失败: %w", err)
	}
	return SaveArtifact(projectRoot, ArtifactPartitions, string(data))
}

// LoadArtifacts 读取项目内已保存的 AI 产物。目录不存在返回空快照;
// 单个文件读取失败跳过该文件,不阻断其余产物。
// Reviews 按文件名倒序(最新在前):文件名内嵌毫秒时间戳,字典序即时间序,
// 且不受文件系统修改时间精度的影响。
func LoadArtifacts(projectRoot string) *Artifacts {
	arts := &Artifacts{}
	dir := artifactDir(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return arts
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		switch e.Name() {
		case "requirements.md":
			arts.Requirements = string(data)
		case "ddl.sql":
			arts.DDL = string(data)
		case "partitions.json":
			arts.Partitions = string(data)
		default:
			if strings.HasPrefix(e.Name(), "review-") && strings.HasSuffix(e.Name(), ".md") {
				info, err := e.Info()
				modTime := ""
				if err == nil {
					modTime = info.ModTime().Format("2006-01-02 15:04:05")
				}
				arts.Reviews = append(arts.Reviews, ReviewFile{
					Name:    e.Name(),
					Path:    filepath.Join(dir, e.Name()),
					ModTime: modTime,
				})
			}
		}
	}

	sort.Slice(arts.Reviews, func(i, j int) bool {
		return arts.Reviews[i].Name > arts.Reviews[j].Name
	})
	return arts
}

// LoadReviewReport 读取指定审查报告的内容。
// name 必须是产物目录下的单个文件名且形如 review-*.md,拒绝任何路径成分。
func LoadReviewReport(projectRoot, name string) (string, error) {
	if strings.TrimSpace(name) == "" || filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) || name != filepath.Base(name) {
		return "", fmt.Errorf("非法的报告文件名")
	}
	if !strings.HasPrefix(name, "review-") || !strings.HasSuffix(name, ".md") {
		return "", fmt.Errorf("不是审查报告文件")
	}
	data, err := os.ReadFile(filepath.Join(artifactDir(projectRoot), name))
	if err != nil {
		return "", fmt.Errorf("读取审查报告失败: %w", err)
	}
	return string(data), nil
}
