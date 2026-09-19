package entimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/contrib/schemast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatSchemaFiles_NonGoFile 测试非 go 文件的跳过
func TestFormatSchemaFiles_NonGoFile(t *testing.T) {
	schemaPath := t.TempDir()

	// 创建一个非 go 文件
	nonGoFile := filepath.Join(schemaPath, "readme.txt")
	err := os.WriteFile(nonGoFile, []byte("README"), 0644)
	require.NoError(t, err)

	// 应该不报错
	err = formatSchemaFiles(schemaPath)
	assert.NoError(t, err)
}

// TestFormatSchemaFiles_EmptyDir 测试空目录的处理
func TestFormatSchemaFiles_EmptyDir(t *testing.T) {
	schemaPath := t.TempDir()

	// 空目录应该不报错
	err := formatSchemaFiles(schemaPath)
	assert.NoError(t, err)
}

// TestTableName_TypeNameConversion 测试表名和类型名的转换
func TestTableName_TypeNameConversion(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"users", "User"},
		{"order_items", "OrderItem"},
		{"user_roles", "UserRole"},
		{"a", "A"},
		{"table_with_underscores", "TableWithUnderscore"}, // inflect.Singularize 会去掉最后的 s
	}

	for _, tc := range testCases {
		result := typeName(tc.input)
		assert.Equal(t, tc.expected, result, "typeName(%s) should be %s", tc.input, tc.expected)
	}
}

// TestTableName_TableNameConversion 测试反向转换
func TestTableName_TableNameConversion(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"User", "users"},
		{"OrderItem", "order_items"},
		{"UserRole", "user_roles"},
		{"A", "as"},
		{"TableWithUnderscore", "table_with_underscores"}, // inflect.Pluralize 的行为
	}

	for _, tc := range testCases {
		result := tableName(tc.input)
		assert.Equal(t, tc.expected, result, "tableName(%s) should be %s", tc.input, tc.expected)
	}
}

// TestUpsertRelation_SelfReference 测试自引用关系的 edge 创建
func TestUpsertRelation_SelfReference(t *testing.T) {
	category := &schemast.UpsertSchema{Name: "Category"}

	opts := relOptions{
		uniqueEdgeFromParent: true,
		recursive:            true,
		refName:              "categories",
		edgeField:            "parent_id",
	}

	upsertRelation(category, category, opts)

	// 自引用应该创建 2 个不同的 edge
	assert.Equal(t, 2, len(category.Edges))

	// 验证 edge 名称不同
	names := make(map[string]bool)
	for _, e := range category.Edges {
		names[e.Descriptor().Name] = true
	}
	assert.Len(t, names, 2)
}

// TestUpsertManyToMany_MissingTables 测试 M2M 表中缺少父表的情况
func TestUpsertManyToMany_MissingTables(t *testing.T) {
	mutations := make(map[string]schemast.Mutator)

	// 只创建 join 表，不创建关联表
	joinTable := &schema.Table{
		Name: "user_roles",
		ForeignKeys: []*schema.ForeignKey{
			{RefTable: &schema.Table{Name: "users"}},
			{RefTable: &schema.Table{Name: "roles"}},
		},
	}

	err := upsertManyToMany(mutations, joinTable)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "join table")
}

// TestUpsertOneToX_NullForeignKey 测试 NULL FK 的处理
func TestUpsertOneToX_NullForeignKey(t *testing.T) {
	parent := &schemast.UpsertSchema{Name: "Parent"}
	child := &schemast.UpsertSchema{Name: "Child"}

	mutations := map[string]schemast.Mutator{
		"parents": parent,
		"children": child,
	}

	// 模拟一个 nullable 的外键
	nullFkTable := &schema.Table{
		Name: "children",
		ForeignKeys: []*schema.ForeignKey{
			{
				Columns: []*schema.Column{{Name: "parent_id"}},
				RefTable: &schema.Table{Name: "parents"},
			},
		},
	}

	upsertOneToX(mutations, nullFkTable)

	// 应该正常创建 edge
	assert.Equal(t, 1, len(child.Edges))
	assert.Equal(t, 1, len(parent.Edges))
}

// TestImport_DSNValidation 测试 DSN 验证
func TestImport_DSNValidation(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// nil DSN
	err := Importer(ctx, nil, &tmpDir, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dsn is nil")

	// nil schemaPath
	dsn := "mysql://localhost/test"
	err = Importer(ctx, &dsn, nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema path is nil")
}
