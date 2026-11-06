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

// 批处理装箱上限:单批 60 文件或 350KB——每批一次 LLM 调用,
// 上限保证单批提示词规模可控;总量 1MB 上限下典型 2-4 批。
const (
	reviewBatchMaxFiles = 60
	reviewBatchMaxBytes = 350 << 10
)

// BuildReviewBatches 把待审查文件贪心装箱成批次:路径排序后顺序累加,
// 单批超限即封箱开新批;同一服务的文件尽量落在同一批(排序天然保证,
// 除非单个服务自身超限被拆)。
func BuildReviewBatches(files []CodeFileInfo) [][]CodeFileInfo {
	sorted := make([]CodeFileInfo, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var batches [][]CodeFileInfo
	var cur []CodeFileInfo
	var curBytes int64

	flush := func() {
		if len(cur) > 0 {
			batches = append(batches, cur)
			cur = nil
			curBytes = 0
		}
	}

	for _, f := range sorted {
		if len(cur) >= reviewBatchMaxFiles || (len(cur) > 0 && curBytes+f.Size > reviewBatchMaxBytes) {
			flush()
		}
		cur = append(cur, f)
		curBytes += f.Size
	}
	flush()
	return batches
}

// BatchLabel 生成批次的人类可读标签:取首个文件的顶层分组
// (app/<svc> 或 api/protos),跨多个分组时以 "+N 分组" 后缀示意。
func BatchLabel(files []CodeFileInfo) string {
	if len(files) == 0 {
		return "(空)"
	}
	groupOf := func(path string) string {
		if strings.HasPrefix(path, "app/") {
			rest := strings.TrimPrefix(path, "app/")
			if svc, _, found := strings.Cut(rest, "/"); found {
				return "app/" + svc
			}
			return "app"
		}
		if strings.HasPrefix(path, "api/protos/") {
			return "api/protos"
		}
		return path
	}

	first := groupOf(files[0].Path)
	groups := map[string]bool{}
	for _, f := range files {
		g := groupOf(f.Path)
		if g != first {
			groups[g] = true
		}
	}
	if len(groups) == 0 {
		return first
	}
	return fmt.Sprintf("%s+%d 分组", first, len(groups))
}

// RenderReviewReport 把批次结果与总评渲染成 markdown 报告。
func RenderReviewReport(batches []ReviewBatchOutcome, issues []ReviewIssue, synthesis string) string {
	var b strings.Builder
	b.WriteString("# AI 代码审查报告\n\n")
	b.WriteString(fmt.Sprintf("- 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("- 批次数: %d,问题总数: %d\n\n", len(batches), len(issues)))

	b.WriteString("## 总评\n\n")
	if strings.TrimSpace(synthesis) == "" {
		synthesis = "(无)"
	}
	b.WriteString(synthesis)
	b.WriteString("\n\n")

	for _, batch := range batches {
		b.WriteString(fmt.Sprintf("## 批次: %s(%d 个问题)\n\n", batch.Label, batch.IssueCount))
		if batch.Summary != "" {
			b.WriteString(batch.Summary)
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

// SaveReviewReports 以同一时间戳落盘 markdown 报告与结构化 issue JSON,
// 返回两个路径。json 供跨版本 diff,不进历史报告列表(LoadArtifacts 只认 .md)。
func SaveReviewReports(projectRoot string, batches []ReviewBatchOutcome, issues []ReviewIssue, synthesis string) (string, string, error) {
	ts := time.Now().Format("20060102-150405.000")
	mdName := "review-" + ts + ".md"
	jsonName := "review-" + ts + ".json"

	if strings.TrimSpace(projectRoot) == "" {
		return "", "", fmt.Errorf("项目根目录为空")
	}
	dir := artifactDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建产物目录失败: %w", err)
	}

	md := RenderReviewReport(batches, issues, synthesis)
	if err := os.WriteFile(filepath.Join(dir, mdName), []byte(md), 0o644); err != nil {
		return "", "", fmt.Errorf("写入报告失败: %w", err)
	}

	payload := map[string]any{
		"generatedAt": ts,
		"batches":     batches,
		"issues":      issues,
		"synthesis":   synthesis,
	}
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return mdPathOf(dir, mdName), "", fmt.Errorf("序列化审查结果失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, jsonName), jsonData, 0o644); err != nil {
		return mdPathOf(dir, mdName), "", fmt.Errorf("写入审查 JSON 失败: %w", err)
	}
	return mdPathOf(dir, mdName), mdPathOf(dir, jsonName), nil
}

func mdPathOf(dir, name string) string {
	return filepath.Join(dir, name)
}
