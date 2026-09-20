package schemasource

import (
	"os"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
)

// LoadSQLFromFile 从 schemaPath 载入 SQL 文本:若 schemaPath 指向普通文件
// 则读其内容,否则视作内联 DDL 文本,剥掉 text:// 等 scheme 前缀后原样返回。
// 调用方传的是 NormalizeDSN 之后的值（裸路径会变成 file://<路径>），所以必须先
// 剥 scheme 再判文件，否则文件路径本身会被当成 SQL 文本喂给解析器。
func LoadSQLFromFile(schemaPath string) string {
	text := stripScheme(schemaPath)

	content, err := os.ReadFile(text)
	if err != nil {
		// 不是文件，视作内联 DDL 文本。
		return text
	}

	return string(content)
}

// stripScheme 去掉 DSN 中的 scheme 前缀（如 text://, file://）
func stripScheme(path string) string {
	if idx := strings.Index(path, "://"); idx != -1 {
		return path[idx+3:]
	}
	return path
}

// ParseType 把 SQL 类型文本解析为 atlas schema 类型,依次尝试
// MySQL/PostgreSQL/SQLite 方言解析器,全部失败时返回 UnsupportedType。
func ParseType(raw string) (schema.Type, error) {
	for _, parse := range []func(string) (schema.Type, error){
		mysql.ParseType,
		postgres.ParseType,
		sqlite.ParseType,
	} {
		typ, err := parse(raw)
		if err != nil {
			continue
		}
		// 各方言对未知类型的表示不同且都不报错:mysql 返回 UnsupportedType,
		// postgres/sqlite 返回 UserDefinedType(int8/timestamptz 等 Postgres
		// 类型名会在 mysql 处短路),均须让位给下一方言
		switch typ.(type) {
		case *schema.UnsupportedType, *postgres.UserDefinedType, *sqlite.UserDefinedType:
			continue
		}
		return typ, nil
	}
	return &schema.UnsupportedType{T: raw}, nil
}
