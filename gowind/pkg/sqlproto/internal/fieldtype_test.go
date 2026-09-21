package internal

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"ariga.io/atlas/sql/schema"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

// TestMySQLFieldType 覆盖 SQL 类型文本 → Proto 类型的映射。
// 重点是带显示宽度的无符号写法:mysqldump、MySQL 5.7 与 MariaDB 导出的 DDL
// 一律是 `bigint(20) unsigned`,先剥括号的旧实现会把 unsigned 连同精度一起丢掉,
// 映射表里的 *_UNSIGNED 键因此永不可达。
func TestMySQLFieldType(t *testing.T) {
	for _, tt := range []struct {
		sqlType string
		want    string
	}{
		// 无符号:带精度、带 zerofill 都要命中
		{sqlType: "bigint(20) unsigned", want: "uint64"},
		{sqlType: "bigint unsigned", want: "uint64"},
		{sqlType: "int(10) unsigned zerofill", want: "uint32"},
		{sqlType: "int unsigned", want: "uint32"},
		{sqlType: "smallint(5) unsigned", want: "uint32"},
		{sqlType: "tinyint(3) unsigned", want: "uint32"},
		{sqlType: "mediumint(8) unsigned", want: "uint32"},

		// 有符号:精度与修饰词的剥离不能改变结论
		{sqlType: "bigint(20)", want: "int64"},
		{sqlType: "int(11)", want: "int32"},
		{sqlType: "BIGINT(20) UNSIGNED", want: "uint64"},
		{sqlType: "  varchar(100)  ", want: "string"},
		{sqlType: "decimal(10,2)", want: "string"},
		{sqlType: "datetime", want: "google.protobuf.Timestamp"},
		{sqlType: "enum('a','b')", want: "string"},

		// TINYINT(1) 的布尔语义只对有符号成立
		{sqlType: "tinyint(1)", want: "bool"},
		{sqlType: "tinyint(1) unsigned", want: "uint32"},
		{sqlType: "tinyint(4)", want: "int32"},

		// 未映射类型必须返回空串,由调用方决定是否兜底
		{sqlType: "citext", want: ""},
		{sqlType: "", want: ""},
	} {
		t.Run(tt.sqlType, func(t *testing.T) {
			if got := MySQLFieldType(tt.sqlType); got != tt.want {
				t.Fatalf("MySQLFieldType(%q) = %q; want %q", tt.sqlType, got, tt.want)
			}
		})
	}
}

// TestPostgresFieldType 覆盖 Postgres 腿:多词类型名与带精度的写法都要落进映射表。
func TestPostgresFieldType(t *testing.T) {
	for _, tt := range []struct {
		sqlType string
		want    string
	}{
		{sqlType: "character varying(255)", want: "string"},
		{sqlType: "double precision", want: "double"},
		{sqlType: "timestamp with time zone", want: "google.protobuf.Timestamp"},
		{sqlType: "timestamp without time zone", want: "google.protobuf.Timestamp"},
		{sqlType: "timestamp(6) with time zone", want: "google.protobuf.Timestamp"},
		{sqlType: "time with time zone", want: "string"},
		{sqlType: "bit varying(8)", want: "bytes"},
		{sqlType: "numeric(10,2)", want: "string"},
		{sqlType: "int4", want: "int32"},
		{sqlType: "timestamptz", want: "google.protobuf.Timestamp"},
		{sqlType: "  INTEGER  ", want: "int32"},

		{sqlType: "citext", want: ""},
		{sqlType: "text[]", want: ""},
	} {
		t.Run(tt.sqlType, func(t *testing.T) {
			if got := PostgresFieldType(tt.sqlType); got != tt.want {
				t.Fatalf("PostgresFieldType(%q) = %q; want %q", tt.sqlType, got, tt.want)
			}
		})
	}
}

// TestTextLegUnsignedMapping 端到端验证离线 DDL 腿:unsigned 判定与类型映射
// 是两段代码,任何一段回归都会在这里暴露。同时验证同表内前缀重名、以及另一张表
// 里的同名同型列不会把有符号列判成无符号。
func TestTextLegUnsignedMapping(t *testing.T) {
	sqlContent := `
CREATE TABLE users (
  id bigint(20) NOT NULL AUTO_INCREMENT,
  uid bigint(20) unsigned NOT NULL,
  login_count int(10) unsigned zerofill NOT NULL,
  tiny_flag tinyint(1) NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE archives (
  id int(11) NOT NULL,
  uid int(11) unsigned NOT NULL
) ENGINE=InnoDB;
`

	converter, err := NewText(&ConvertOptions{
		schemaPath: sqlContent,
		driver: &schemasource.Driver{
			Dialect:    "text",
			SchemaName: "public",
		},
	})
	if err != nil {
		t.Fatalf("NewText: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("SchemaTables: %v", err)
	}

	want := map[string]map[string]string{
		"users": {
			"id":          "int64",
			"uid":         "uint64",
			"login_count": "uint32",
			"tiny_flag":   "bool",
		},
		"archives": {
			"id":  "int32",
			"uid": "uint32",
		},
	}

	got := make(map[string]map[string]string, len(tables))
	for _, table := range tables {
		got[table.Name] = make(map[string]string, len(table.Fields))
		for _, field := range table.Fields {
			got[table.Name][field.Name] = field.Type
		}
	}

	if len(got) != len(want) {
		t.Fatalf("表数量 = %d, 期望 %d\n实际: %v", len(got), len(want), got)
	}
	for table, fields := range want {
		for field, wantType := range fields {
			if gotType := got[table][field]; gotType != wantType {
				t.Errorf("%s.%s 类型 = %q, 期望 %q\n实际: %v", table, field, gotType, wantType, got)
			}
		}
	}
}

// TestLiveLegAtlasRawMapping 固定住连库腿的输入契约:mysql provider 交给映射函数的
// 是 atlas 从 information_schema.COLUMNS.COLUMN_TYPE 原样带回的 Raw(见
// ariga.io/atlas 的 sql/mysql/inspect_oss.go:addColumn 里 `Raw: typ.String`),
// 与离线 DDL 腿的输入不是同一种文本。MySQL 8.4 起整型不再带显示宽度,而 5.7、
// MariaDB 与带 zerofill 的列仍然带,所以两种形式都要断言;SqlType 也必须留住
// UNSIGNED 后缀——ent/gorm 的 Go 类型映射读的是它。
//
// 放在离线单元测试里而不是只放在连库用例里:矩阵作业没有 MySQL,连库用例在那里
// 一律跳过,这条契约在 CI 上就只剩 ubuntu+MySQL 一个来源。
func TestLiveLegAtlasRawMapping(t *testing.T) {
	column := func(name, raw string) *schema.Column {
		return &schema.Column{Name: name, Type: &schema.ColumnType{Raw: raw}}
	}

	tables := []*schema.Table{{
		Name: "users",
		Columns: []*schema.Column{
			column("id", "bigint"),
			column("uid", "bigint unsigned"),
			column("login_count", "int(10) unsigned zerofill"),
			column("tiny_flag", "tinyint(1)"),
			column("note", "varchar(100)"),
		},
	}}

	tableDatas, err := schemaTables(MySQLFieldType, tables)
	if err != nil {
		t.Fatalf("schemaTables: %v", err)
	}
	if len(tableDatas) != 1 {
		t.Fatalf("users 表不该被当成关联表丢掉,实际得到 %d 张表", len(tableDatas))
	}

	fields := make(map[string]FieldData, len(tableDatas[0].Fields))
	for _, f := range tableDatas[0].Fields {
		fields[f.Name] = f
	}

	for _, c := range []struct {
		name    string
		proto   string
		sqlType string
	}{
		{name: "id", proto: "int64", sqlType: "BIGINT"},
		{name: "uid", proto: "uint64", sqlType: "BIGINT UNSIGNED"},
		{name: "login_count", proto: "uint32", sqlType: "INT UNSIGNED"},
		{name: "tiny_flag", proto: "bool", sqlType: "TINYINT"},
		{name: "note", proto: "string", sqlType: "VARCHAR"},
	} {
		field, ok := fields[c.name]
		if !ok {
			t.Fatalf("字段 %s 没有出现在生成数据里", c.name)
		}
		if field.Type != c.proto {
			t.Errorf("%s 的 Proto 类型 = %q, 期望 %q", c.name, field.Type, c.proto)
		}
		if field.SqlType != c.sqlType {
			t.Errorf("%s 的 SqlType = %q, 期望 %q", c.name, field.SqlType, c.sqlType)
		}
	}
}

// TestUnknownSQLTypeIsNotSilent 未映射类型兜底成 string 时,必须留下能定位到表与列的告警。
func TestUnknownSQLTypeIsNotSilent(t *testing.T) {
	sqlContent := `
CREATE TABLE misc (
  id int NOT NULL,
  label citext NOT NULL
);
`

	converter, err := NewText(&ConvertOptions{
		schemaPath: sqlContent,
		driver: &schemasource.Driver{
			Dialect:    "text",
			SchemaName: "public",
		},
	})
	if err != nil {
		t.Fatalf("NewText: %v", err)
	}

	var tables []*TableData
	logs := captureLog(t, func() {
		got, err := converter.SchemaTables(context.Background())
		if err != nil {
			t.Fatalf("SchemaTables: %v", err)
		}
		tables = got
	})

	if len(tables) != 1 || len(tables[0].Fields) != 2 {
		t.Fatalf("产物不符预期: %+v", tables)
	}

	var label *FieldData
	for i := range tables[0].Fields {
		if tables[0].Fields[i].Name == "label" {
			label = &tables[0].Fields[i]
		}
	}
	if label == nil {
		t.Fatalf("缺少 label 列: %+v", tables[0].Fields)
	}
	if label.Type != "string" {
		t.Errorf("label.Type = %q, 期望兜底为 string", label.Type)
	}

	t.Logf("捕获日志:\n%s", logs)
	if n := strings.Count(logs, "没有对应的 Proto 映射"); n != 1 {
		t.Fatalf("未映射告警条数 = %d, 期望恰好 1 条(label)\n日志:\n%s", n, logs)
	}
	for _, want := range []string{"misc", "label", "citext"} {
		if !strings.Contains(logs, want) {
			t.Errorf("告警日志缺少定位信息 %q\n日志:\n%s", want, logs)
		}
	}
}

// captureLog 捕获标准库 log 的输出,用于断言「不再静默」。
func captureLog(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	flags := log.Flags()
	log.SetFlags(0)
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
	})

	fn()

	return buf.String()
}
