package entimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteSchema_ProcessCWDOutsideGoModule 复现 issue #13:
// GUI 进程从 Dock/Finder 启动时 CWD 不在任何 Go module 内(macOS 下为 /),
// go/packages 对此失败无兜底,schemast.Load 会报 "missing package information"。
// WriteSchema 须自行把 CWD 定位到 schema 目录后再加载。
func TestWriteSchema_ProcessCWDOutsideGoModule(t *testing.T) {
	// 模拟用户项目:go.mod 在项目根,schema 目录在其子路径下
	projectDir := t.TempDir()
	schemaDir := filepath.Join(projectDir, "app", "core", "service", "internal", "data", "ent", "schema")
	require.NoError(t, os.MkdirAll(schemaDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(
		"module example.com/scratch\n\ngo 1.21\n",
	), 0o644))

	text, err := NewText(&ImportOptions{schemaPath: "CREATE TABLE addresses (\n" +
		"\tid bigint NOT NULL,\n" +
		"\tconsignee varchar NOT NULL,\n" +
		"\tPRIMARY KEY (id)\n" +
		") COMMENT '地址表';\n"})
	require.NoError(t, err)

	mutations, err := text.SchemaMutations(context.Background())
	require.NoError(t, err)

	origWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	// 把进程 CWD 切到模块外的临时目录,模拟 macOS GUI 启动环境
	outside := t.TempDir()
	require.NoError(t, os.Chdir(outside))

	require.NoError(t, WriteSchema(mutations, WithSchemaPath(schemaDir)))

	generated, err := os.ReadDir(schemaDir)
	require.NoError(t, err)
	assert.NotEmpty(t, generated, "schema files should be written")

	// 进程 CWD 应被恢复
	wd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, outside, wd)
}

// TestWriteSchema_SchemaDirOutsideGoModule 保证错误可诊断:
// schema 目录不在任何 Go module 内时,报错应提示缺少 go.mod,而非裸的
// "missing package information"。
func TestWriteSchema_SchemaDirOutsideGoModule(t *testing.T) {
	outside := t.TempDir()
	schemaDir := filepath.Join(outside, "ent", "schema")
	require.NoError(t, os.MkdirAll(schemaDir, 0o755))

	text, err := NewText(&ImportOptions{schemaPath: "CREATE TABLE users (\n" +
		"\tid bigint NOT NULL,\n" +
		"\tPRIMARY KEY (id)\n" +
		");\n"})
	require.NoError(t, err)

	mutations, err := text.SchemaMutations(context.Background())
	require.NoError(t, err)

	origWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	require.NoError(t, os.Chdir(outside))

	err = WriteSchema(mutations, WithSchemaPath(schemaDir))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go mod init")
}
