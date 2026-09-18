package entimport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestText(t *testing.T) {
	sql := `
CREATE TABLE users (
	id bigint(20) unsigned NOT NULL COMMENT '用户ID',
	other_id BIGINT(20) unsigned NOT NULL COMMENT '其他ID',
	enum_column enum('a','b','c','d') DEFAULT NULL COMMENT '枚举类型字段',
	int_column int(10) DEFAULT '0' COMMENT '整型字段',
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin COMMENT '用户基本信息表';

CREATE TABLE users1 (
	id bigint(20) unsigned PRIMARY KEY AUTO_INCREMENT,
	other_id bigint(20) unsigned NOT NULL,
	enum_column enum('a','b','c','d') DEFAULT NULL,
	int_column int(10) DEFAULT '0',
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
`

	text, err := NewText(&ImportOptions{
		schemaPath: sql,
	})
	assert.Nil(t, err)

	mutations, err := text.SchemaMutations(context.Background())
	assert.Nil(t, err)

	schemaPath := t.TempDir()
	// WriteSchema 要求 schema 目录位于某个 Go module 内(与真实生成目标一致)
	require.NoError(t, os.WriteFile(filepath.Join(schemaPath, "go.mod"), []byte("module example.com/scratch\n\ngo 1.21\n"), 0o644))
	if err = WriteSchema(mutations, WithSchemaPath(schemaPath)); err != nil {
		t.Fatalf("entimport: schema writing failed - %v", err)
	}
}

func TestInspectSchema_TableIncludeFilter(t *testing.T) {
	sql := `
CREATE TABLE users (
	id bigint(20) unsigned NOT NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE roles (
	id bigint(20) unsigned NOT NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB;
`

	text, err := NewText(&ImportOptions{schemaPath: sql})
	assert.Nil(t, err)

	// 不带过滤：全部表
	var s schema.Schema
	_, err = text.InspectSchema(context.Background(), sql, &schema.InspectOptions{}, &s)
	assert.Nil(t, err)
	assert.Len(t, s.Tables, 2)

	// 带 Tables 过滤：只剩指定表
	s = schema.Schema{}
	_, err = text.InspectSchema(context.Background(), sql, &schema.InspectOptions{
		Tables: []string{"roles"},
	}, &s)
	assert.Nil(t, err)
	assert.Len(t, s.Tables, 1)
	assert.Equal(t, "roles", s.Tables[0].Name)
}

// TestInspectSchema_PostgresDDL 覆盖 Postgres 方言 DDL 的解析:
// 带引号的 schema 限定表名、int8/timestamptz 等类型、表级
// CONSTRAINT ... PRIMARY KEY、以及 COMMENT ON 独立语句(issue #13 的表结构形态)。
func TestInspectSchema_PostgresDDL(t *testing.T) {
	sql := `CREATE TABLE "public"."addresses" (
  "id" int8 NOT NULL,
  "create_time" timestamptz(6) NOT NULL,
  "delete_time" timestamptz(6),
  "consignee" varchar COLLATE "pg_catalog"."default" NOT NULL,
  "lng" float4,
  "default" boolean NOT NULL DEFAULT false,
  "tenant_id" int8 NOT NULL,
  CONSTRAINT "addresses_pkey" PRIMARY KEY ("id")
);

COMMENT ON COLUMN "public"."addresses"."consignee" IS '收件人';
COMMENT ON TABLE "public"."addresses" IS '地址表';
`

	text, err := NewText(&ImportOptions{schemaPath: sql})
	require.NoError(t, err)

	var s schema.Schema
	_, err = text.InspectSchema(context.Background(), sql, &schema.InspectOptions{}, &s)
	require.NoError(t, err)
	require.Len(t, s.Tables, 1)

	table := s.Tables[0]
	assert.Equal(t, "addresses", table.Name, "应去掉引号与 schema 限定")
	assert.Equal(t, "地址表", tableComment(table))

	// 表级 CONSTRAINT ... PRIMARY KEY 应识别为主键
	require.NotNil(t, table.PrimaryKey)
	require.Len(t, table.PrimaryKey.Parts, 1)
	assert.Equal(t, "id", table.PrimaryKey.Parts[0].C.Name)

	cols := make(map[string]*schema.Column, len(table.Columns))
	for _, c := range table.Columns {
		cols[c.Name] = c
	}

	idType, ok := cols["id"].Type.Type.(*schema.IntegerType)
	require.True(t, ok, "int8 应解析为 IntegerType,得到 %T", cols["id"].Type.Type)
	assert.Equal(t, "int8", idType.T)

	tenantType, ok := cols["tenant_id"].Type.Type.(*schema.IntegerType)
	require.True(t, ok)
	assert.Equal(t, "int8", tenantType.T)

	_, ok = cols["create_time"].Type.Type.(*schema.TimeType)
	assert.True(t, ok, "timestamptz(6) 应解析为 TimeType,得到 %T", cols["create_time"].Type.Type)
	assert.False(t, cols["create_time"].Type.Null)

	_, ok = cols["delete_time"].Type.Type.(*schema.TimeType)
	assert.True(t, ok)
	assert.True(t, cols["delete_time"].Type.Null)

	_, ok = cols["consignee"].Type.Type.(*schema.StringType)
	assert.True(t, ok)
	assert.Equal(t, "收件人", columnComment(cols["consignee"]))

	_, ok = cols["lng"].Type.Type.(*schema.FloatType)
	assert.True(t, ok)

	_, ok = cols["default"].Type.Type.(*schema.BoolType)
	assert.True(t, ok, "boolean 应解析为 BoolType,得到 %T", cols["default"].Type.Type)

	// 生成的 ent schema 全链路可写出
	mutations, err := text.SchemaMutations(context.Background())
	require.NoError(t, err)
	schemaDir := t.TempDir()
	// WriteSchema 要求 schema 目录位于某个 Go module 内(与真实生成目标一致)
	require.NoError(t, os.WriteFile(filepath.Join(schemaDir, "go.mod"), []byte("module example.com/scratch\n\ngo 1.21\n"), 0o644))
	require.NoError(t, WriteSchema(mutations, WithSchemaPath(schemaDir)))
}

func tableComment(table *schema.Table) string {
	for _, attr := range table.Attrs {
		if c, ok := attr.(*schema.Comment); ok {
			return c.Text
		}
	}
	return ""
}

func columnComment(column *schema.Column) string {
	for _, attr := range column.Attrs {
		if c, ok := attr.(*schema.Comment); ok {
			return c.Text
		}
	}
	return ""
}
