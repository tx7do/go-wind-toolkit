package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func planFile(path string, size int64) CodeFileInfo {
	return CodeFileInfo{Path: path, Size: size}
}

func TestBuildReviewBatches_SameServiceStaysTogether(t *testing.T) {
	files := []CodeFileInfo{
		planFile("app/user/service/a.go", 100),
		planFile("app/user/service/b.go", 100),
		planFile("app/order/service/c.go", 100),
		planFile("api/protos/user.proto", 100),
	}

	batches := BuildReviewBatches(files)
	if len(batches) != 1 {
		t.Fatalf("小文件应单批装下,得到 %d 批", len(batches))
	}
	if len(batches[0]) != 4 {
		t.Fatalf("单批应含 4 个文件,得到 %d", len(batches[0]))
	}
	// 排序后 api/protos 排最前
	if batches[0][0].Path != "api/protos/user.proto" {
		t.Fatalf("批内应按路径排序,得到 %s", batches[0][0].Path)
	}
}

func TestBuildReviewBatches_SplitsOnFileCap(t *testing.T) {
	var files []CodeFileInfo
	for i := 0; i < reviewBatchMaxFiles+10; i++ {
		files = append(files, planFile("app/user/f"+string(rune('a'+i%26))+string(rune('a'+i/26))+".go", 10))
	}

	batches := BuildReviewBatches(files)
	if len(batches) != 2 {
		t.Fatalf("超过单批文件上限应拆两批,得到 %d 批", len(batches))
	}
	if len(batches[0]) != reviewBatchMaxFiles {
		t.Fatalf("首批应恰好 60 个文件,得到 %d", len(batches[0]))
	}
}

func TestBuildReviewBatches_SplitsOnSizeCap(t *testing.T) {
	files := []CodeFileInfo{
		planFile("app/user/big1.go", reviewBatchMaxBytes-100),
		planFile("app/user/big2.go", reviewBatchMaxBytes), // 装不进第一批
	}

	batches := BuildReviewBatches(files)
	if len(batches) != 2 {
		t.Fatalf("超尺寸应拆两批,得到 %d 批", len(batches))
	}
	if batches[0][0].Path != "app/user/big1.go" || batches[1][0].Path != "app/user/big2.go" {
		t.Fatalf("拆分位置不符: %v", batches)
	}
}

func TestBatchLabel(t *testing.T) {
	single := []CodeFileInfo{planFile("app/user/service/a.go", 1)}
	if got := BatchLabel(single); got != "app/user" {
		t.Fatalf("单服务标签不符: %q", got)
	}

	cross := []CodeFileInfo{
		planFile("app/user/service/a.go", 1),
		planFile("app/order/service/b.go", 1),
		planFile("api/protos/x.proto", 1),
	}
	got := BatchLabel(cross)
	// BatchLabel 不排序,以首个文件的分组开头 + 其余分组计数
	if !strings.HasPrefix(got, "app/user+") || !strings.Contains(got, "2 分组") {
		t.Fatalf("跨分组标签不符: %q", got)
	}

	if got := BatchLabel(nil); got != "(空)" {
		t.Fatalf("空批标签不符: %q", got)
	}
}

func TestRenderReviewReport(t *testing.T) {
	batches := []ReviewBatchOutcome{
		{Label: "app/user", Summary: "用户服务整体健康", IssueCount: 2},
		{Label: "app/order", Summary: "订单服务有严重问题", IssueCount: 1},
	}
	issues := []ReviewIssue{
		{Severity: "critical", File: "app/order/service/c.go", Line: 12, Title: "空指针", Detail: "可能解引用 nil"},
	}
	md := RenderReviewReport(batches, issues, "先修空指针。")

	for _, want := range []string{
		"# AI 代码审查报告", "## 总评", "先修空指针。",
		"## 批次: app/user(2 个问题)", "用户服务整体健康",
		"## 批次: app/order(1 个问题)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("报告应包含 %q", want)
		}
	}
}

func TestSaveReviewReports(t *testing.T) {
	root := t.TempDir()
	batches := []ReviewBatchOutcome{{Label: "app/user", Summary: "s", IssueCount: 1}}
	issues := []ReviewIssue{{Severity: "major", File: "a.go", Line: 3, Title: "t", Detail: "d"}}

	mdPath, jsonPath, err := SaveReviewReports(root, batches, issues, "总评")
	if err != nil {
		t.Fatalf("落盘失败: %v", err)
	}
	if filepath.Base(mdPath) == filepath.Base(jsonPath) {
		t.Fatal("md 与 json 路径不应相同")
	}
	if !strings.HasSuffix(filepath.ToSlash(mdPath), ".md") || !strings.HasSuffix(filepath.ToSlash(jsonPath), ".json") {
		t.Fatalf("后缀不符: %s, %s", mdPath, jsonPath)
	}
	// 同一时间戳:去掉扩展名后应一致
	if strings.TrimSuffix(filepath.Base(mdPath), ".md") != strings.TrimSuffix(filepath.Base(jsonPath), ".json") {
		t.Fatal("md 与 json 应共用同一时间戳基名")
	}

	// md 进历史列表,json 不进
	arts := LoadArtifacts(root)
	if len(arts.Reviews) != 1 {
		t.Fatalf("历史列表应只有 1 份报告,得到 %d", len(arts.Reviews))
	}
	if !strings.HasSuffix(arts.Reviews[0].Name, ".md") {
		t.Fatalf("历史列表应只含 md: %s", arts.Reviews[0].Name)
	}

	// json 内容可反序列化回结构
	data, rerr := os.ReadFile(jsonPath)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !strings.Contains(string(data), `"severity": "major"`) {
		t.Fatalf("json 应含结构化 issue: %.120s", data)
	}
}
