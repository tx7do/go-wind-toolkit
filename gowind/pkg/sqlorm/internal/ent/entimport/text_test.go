package entimport

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
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

	schemaPath := ".\\schema"
	if err = WriteSchema(mutations, WithSchemaPath(schemaPath)); err != nil {
		log.Fatalf("entimport: schema writing failed - %v", err)
	}
}
