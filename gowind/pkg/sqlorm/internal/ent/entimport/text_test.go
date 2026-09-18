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
