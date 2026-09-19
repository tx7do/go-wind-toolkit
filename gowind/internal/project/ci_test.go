package project

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriteCIWorkflow(t *testing.T) {
	root := t.TempDir()

	written, err := writeCIWorkflow(root)
	if err != nil {
		t.Fatalf("writeCIWorkflow: %v", err)
	}
	if !written {
		t.Fatal("expected written=true on first call")
	}

	file := filepath.Join(root, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read ci.yml: %v", err)
	}

	// 必须是合法 YAML。
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("ci.yml is not valid YAML: %v", err)
	}
	if _, ok := doc["jobs"]; !ok {
		t.Fatal("ci.yml missing jobs section")
	}

	// 幂等:已存在时不覆盖、返回 false。
	os.WriteFile(file, []byte("# custom\n"), 0o644)
	written2, err := writeCIWorkflow(root)
	if err != nil {
		t.Fatalf("second writeCIWorkflow: %v", err)
	}
	if written2 {
		t.Fatal("expected written=false when file already exists")
	}
	kept, _ := os.ReadFile(file)
	if string(kept) != "# custom\n" {
		t.Fatal("existing ci.yml was overwritten")
	}
}
