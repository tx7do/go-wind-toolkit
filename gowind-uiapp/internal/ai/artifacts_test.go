package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveArtifact_FixedNames(t *testing.T) {
	root := t.TempDir()

	path, err := SaveArtifact(root, ArtifactDDL, testDDL)
	if err != nil {
		t.Fatalf("保存 ddl 失败: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(path), ".gowind/ai/ddl.sql") {
		t.Fatalf("落盘路径不符: %s", path)
	}

	if _, err := SaveArtifact(root, ArtifactRequirements, "需求文档"); err != nil {
		t.Fatalf("保存 requirements 失败: %v", err)
	}

	arts := LoadArtifacts(root)
	if arts.DDL != testDDL || arts.Requirements != "需求文档" {
		t.Fatalf("roundtrip 内容不符: ddl=%.20q req=%q", arts.DDL, arts.Requirements)
	}
}

func TestSaveArtifact_UnknownKind(t *testing.T) {
	root := t.TempDir()
	if _, err := SaveArtifact(root, ArtifactKind("bogus"), "x"); err == nil {
		t.Fatal("未知 kind 应报错")
	}
}

func TestSaveArtifact_Overwrite(t *testing.T) {
	root := t.TempDir()
	if _, err := SaveArtifact(root, ArtifactDDL, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveArtifact(root, ArtifactDDL, "v2"); err != nil {
		t.Fatal(err)
	}
	if got := LoadArtifacts(root).DDL; got != "v2" {
		t.Fatalf("固定文件名应后写覆盖前写,得到 %q", got)
	}
}

func TestSaveArtifact_ReviewTimestamped(t *testing.T) {
	root := t.TempDir()

	p1, err := SaveArtifact(root, ArtifactReview, "报告一")
	if err != nil {
		t.Fatal(err)
	}
	// 时间戳含毫秒;同秒连写也不允许互相覆盖,名字必须不同。
	time.Sleep(2 * time.Millisecond)
	p2, err := SaveArtifact(root, ArtifactReview, "报告二")
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 {
		t.Fatalf("两份报告路径不应相同: %s", p1)
	}

	arts := LoadArtifacts(root)
	if len(arts.Reviews) != 2 {
		t.Fatalf("应列出 2 份报告,得到 %d", len(arts.Reviews))
	}
	// 倒序:最新在前。路径包含的毫秒段使字典序与时间序一致。
	if arts.Reviews[0].Path != p2 || arts.Reviews[1].Path != p1 {
		t.Fatalf("报告应按时间倒序,得到 [%s, %s]", arts.Reviews[0].Path, arts.Reviews[1].Path)
	}
	for _, r := range arts.Reviews {
		if r.ModTime == "" {
			t.Fatalf("报告 %s 缺少 ModTime", r.Name)
		}
	}
}

func TestLoadArtifacts_MissingDir(t *testing.T) {
	arts := LoadArtifacts(filepath.Join(t.TempDir(), "nope"))
	if arts == nil || arts.DDL != "" || arts.Requirements != "" || arts.Partitions != "" || len(arts.Reviews) != 0 {
		t.Fatalf("目录缺失应返回空快照,得到 %+v", arts)
	}
}

func TestSavePartitionsArtifact(t *testing.T) {
	root := t.TempDir()
	if _, err := SavePartitionsArtifact(root, []MicroservicePartition{
		{ServiceName: "user", Tables: []string{"user"}, Description: "用户"},
	}); err != nil {
		t.Fatal(err)
	}
	arts := LoadArtifacts(root)
	if !strings.Contains(arts.Partitions, `"serviceName": "user"`) {
		t.Fatalf("partitions.json 应为缩进 JSON,得到: %.60s", arts.Partitions)
	}
}

func TestSaveArtifact_EmptyRoot(t *testing.T) {
	if _, err := SaveArtifact("  ", ArtifactDDL, "x"); err == nil {
		t.Fatal("空项目根应报错")
	}
}

func TestSaveArtifact_CreatesDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deep", "nested")
	if _, err := SaveArtifact(root, ArtifactDDL, "x"); err != nil {
		t.Fatalf("应自动创建多级目录: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gowind", "ai", "ddl.sql")); err != nil {
		t.Fatalf("文件应已落盘: %v", err)
	}
}
