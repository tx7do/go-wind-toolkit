package main

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 与仓库其他插件的 golden 回归同思路：生成结果与提交的 golden 逐字节对比，
// 生成器输出漂移即失败。go test -update 可刷新 golden。
var update = flag.Bool("update", false, "refresh golden files")

func TestGolden(t *testing.T) {
	specs, err := LoadSpecs([]string{"testdata/specs"})
	if err != nil {
		t.Fatalf("load specs: %v", err)
	}
	outs, err := generateAll(context.Background(), []string{"testdata/proto"}, specs)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	goldenDir := filepath.Join("testdata", "golden")
	if *update {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	seen := make(map[string]bool, len(outs))
	for _, out := range outs {
		golden := filepath.Join(goldenDir, strings.ReplaceAll(out.name, "/", "_")+".golden")
		seen[golden] = true

		if *update {
			if err := os.WriteFile(golden, out.content, 0o644); err != nil {
				t.Fatalf("update golden %s: %v", golden, err)
			}
			continue
		}

		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("read golden %s: %v (run go test -update to create)", golden, err)
		}
		if !bytes.Equal(want, out.content) {
			t.Errorf("%s drifted from golden %s\n--- want ---\n%s\n--- got ---\n%s",
				out.name, golden, want, out.content)
		}
	}

	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		full := filepath.Join(goldenDir, e.Name())
		if !seen[full] {
			t.Errorf("stale golden %s no longer produced (delete it)", full)
		}
	}
}
