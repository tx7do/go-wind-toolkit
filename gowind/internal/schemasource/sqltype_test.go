package schemasource

import (
	"reflect"
	"testing"
)

func TestParseSQLType(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want SQLType
	}{
		{raw: "int", want: SQLType{Base: "INT"}},
		{raw: "  bigint(20)  ", want: SQLType{Base: "BIGINT", Args: "20"}},
		{raw: "bigint(20) unsigned", want: SQLType{Base: "BIGINT", Args: "20", Unsigned: true}},
		{raw: "INT UNSIGNED", want: SQLType{Base: "INT", Unsigned: true}},
		{raw: "int(10) unsigned zerofill", want: SQLType{Base: "INT", Args: "10", Unsigned: true}},
		{raw: "decimal(10,2)", want: SQLType{Base: "DECIMAL", Args: "10,2"}},
		{raw: "enum('a','b')", want: SQLType{Base: "ENUM", Args: "'a','b'"}},
		// 枚举字面量里的单词不是修饰词,否则 'unsigned' 这种取值会把列判成无符号
		{raw: "enum('signed','unsigned')", want: SQLType{Base: "ENUM", Args: "'signed','unsigned'"}},
		{raw: "timestamp(6) with time zone", want: SQLType{Base: "TIMESTAMP", Args: "6"}},
		{raw: "character varying(255)", want: SQLType{Base: "CHARACTER VARYING", Args: "255"}},
		{raw: "varchar(", want: SQLType{Base: "VARCHAR"}},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			if got := ParseSQLType(tt.raw); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseSQLType(%q) = %+v; want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSQLTypeKey(t *testing.T) {
	for _, tt := range []struct{ raw, want string }{
		{raw: "bigint(20)", want: "BIGINT"},
		// 先剥括号会把 unsigned 连同精度一起丢掉,Key 必须仍然保留它
		{raw: "bigint(20) unsigned", want: "BIGINT UNSIGNED"},
		{raw: "tinyint(1) unsigned zerofill", want: "TINYINT UNSIGNED"},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			if got := ParseSQLType(tt.raw).Key(); got != tt.want {
				t.Fatalf("ParseSQLType(%q).Key() = %q; want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// unsignedFixtureDDL 覆盖 unsigned 判定会遇到的各种原文:引号包裹的表名列名、
// 前缀重名的列(uid/id)、索引与主键里再次出现的列名、字符串字面量与行注释里的
// 同型文本、另一张表里的同名同型列。
const unsignedFixtureDDL = `
-- 建表脚本,uid bigint unsigned 只是注释
CREATE TABLE IF NOT EXISTS ` + "`users`" + ` (
  ` + "`id` bigint(20) NOT NULL AUTO_INCREMENT," + `
  ` + "`uid` bigint(20) unsigned NOT NULL DEFAULT '0'," + `
  ` + "`login_count` int(10) unsigned zerofill NOT NULL," + `
  ` + "`amount` decimal(10,2) NOT NULL DEFAULT '1,000.00'," + `
  ` + "`remark` varchar(64) NOT NULL COMMENT '写法如 id bigint unsigned'," + `
  PRIMARY KEY (` + "`id`" + `),
  KEY ` + "`idx_uid`" + ` (` + "`uid`" + `)
) ENGINE=InnoDB;

CREATE TABLE scores (
  id bigint unsigned NOT NULL,
  score int NOT NULL
);

CREATE TABLE legacy_scores (
  id int NOT NULL,
  score int unsigned NOT NULL
);

CREATE TABLE trailing (
  id bigint NOT NULL, -- 参照 id bigint unsigned
  note text NULL /* note int unsigned */
);
`

func TestColumnIsUnsigned(t *testing.T) {
	for _, tt := range []struct {
		name  string
		table string
		col   string
		typ   string
		want  bool
	}{
		{name: "前缀重名的列不受 uid ... unsigned 影响", table: "users", col: "id", typ: "bigint(20)", want: false},
		{name: "无引号书写命中", table: "users", col: "uid", typ: "bigint(20)", want: true},
		{name: "zerofill 后缀不影响判定", table: "users", col: "login_count", typ: "int(10)", want: true},
		{name: "精度里的逗号不截断片段", table: "users", col: "amount", typ: "decimal(10,2)", want: false},
		{name: "同名列按表体隔离", table: "scores", col: "score", typ: "int", want: false},
		{name: "另一张表的同名列各自判定", table: "legacy_scores", col: "score", typ: "int", want: true},
		{name: "scores.id 自身就是无符号", table: "scores", col: "id", typ: "bigint", want: true},
		{name: "行注释里的同型文本不算", table: "trailing", col: "id", typ: "bigint", want: false},
		{name: "块注释里的同型文本不算", table: "trailing", col: "note", typ: "text", want: false},
		{name: "COMMENT 字面量里的同型文本不算", table: "users", col: "remark", typ: "varchar(64)", want: false},
		{name: "表名缺失时退回全文但仍按类型锚定", table: "", col: "uid", typ: "bigint(20)", want: true},
		{name: "表名列名大小写无关", table: "USERS", col: "UID", typ: "BIGINT(20)", want: true},
		// 类型文本缺失时只能退化到「列名成词 + 其后是单词」的弱锚定
		{name: "colType 为空仍按词边界判定", table: "users", col: "uid", typ: "", want: true},
		{name: "colType 为空时主键子句不算列定义", table: "users", col: "id", typ: "", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ColumnIsUnsigned(tt.table, tt.col, tt.typ, unsignedFixtureDDL); got != tt.want {
				t.Fatalf("ColumnIsUnsigned(%q, %q, %q) = %v; want %v\ntable body:\n%s",
					tt.table, tt.col, tt.typ, got, tt.want, unsignedFixtureDDL)
			}
		})
	}
}

func TestColumnIsUnsignedUpperCaseDDL(t *testing.T) {
	ddl := "CREATE TABLE T (A INT UNSIGNED NOT NULL, B INT NOT NULL);"

	if !ColumnIsUnsigned("t", "a", "int", ddl) {
		t.Fatalf("ColumnIsUnsigned 大写 DDL 中的 A INT UNSIGNED 应判为无符号:\n%s", ddl)
	}
	if ColumnIsUnsigned("t", "b", "int", ddl) {
		t.Fatalf("ColumnIsUnsigned 大写 DDL 中的 B INT 应判为有符号:\n%s", ddl)
	}
}

func TestColumnIsUnsignedNoUnsignedAtAll(t *testing.T) {
	ddl := "CREATE TABLE posts (\n  id INT PRIMARY KEY,\n  title VARCHAR(255) NOT NULL\n);"

	for _, col := range []struct{ name, typ string }{{"id", "INT"}, {"title", "VARCHAR(255)"}} {
		if ColumnIsUnsigned("posts", col.name, col.typ, ddl) {
			t.Errorf("posts.%s 应为有符号:\n%s", col.name, ddl)
		}
	}
}
