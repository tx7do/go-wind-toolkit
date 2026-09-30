package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// ciWorkflowYAML 是 gow new 生成的 GitHub Actions CI 模板。
// 模板仓库的 .github 在脚手架时被剥离,故由本工具统一发射一份可开箱即用的
// 构建 / vet / 测试工作流。go 版本经 go-version-file 从项目 go.mod 读取。
const ciWorkflowYAML = `name: CI

on:
  push:
    branches: [ main, master ]
  pull_request:
    branches: [ main, master ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Download modules
        run: go mod download

      - name: Vet
        run: go vet ./...

      - name: Build
        run: go build ./...

      - name: Test
        run: go test ./...
`

// writeCIWorkflow 在 projectRoot 下生成 .github/workflows/ci.yml(幂等覆盖)。
// 已存在 .github/workflows/ci.yml 时不覆盖用户自定义内容,返回 false。
func writeCIWorkflow(projectRoot string) (bool, error) {
	dir := filepath.Join(projectRoot, ".github", "workflows")
	file := filepath.Join(dir, "ci.yml")
	if _, err := os.Stat(file); err == nil {
		return false, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("create .github/workflows: %w", err)
	}
	if err := os.WriteFile(file, []byte(ciWorkflowYAML), 0o644); err != nil {
		return false, fmt.Errorf("write ci.yml: %w", err)
	}
	return true, nil
}
