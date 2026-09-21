package schemasource

import (
	"fmt"
	"os"
	"strings"
)

// sqlTextPrefixes 是 DDL/SQL 脚本的起手式:关键字后面必然跟空白,
// 所以 "SET " 不会把名为 SET 的账号误判成正文。
var sqlTextPrefixes = []string{
	"CREATE ", "ALTER ", "DROP ", "TRUNCATE ", "RENAME ",
	"INSERT ", "UPDATE ", "DELETE ", "SELECT ", "WITH ", "SET ",
	"BEGIN ", "COMMIT ", "START TRANSACTION", "TABLE ", "COMMENT ",
	"--", "#", "/*",
}

// NormalizeDSN 把用户给的数据源补成带 scheme 的形式:
// 已有 scheme 的原样返回;指向磁盘对象的补 file://;SQL 正文补 text://;
// go-sql-driver 原生格式(root:pass@tcp(host:port)/db)补 mysql://。
//
// 认不出来的必须报错,不能再兜底成 text://:那是把 MySQL 原生 DSN 当 SQL 正文,
// 解析出零张表也算"成功",用户只会看到空产物。
func NormalizeDSN(dsn string) (string, error) {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return "", fmt.Errorf("schemasource: 数据源为空")
	}

	if strings.Contains(trimmed, "://") {
		return trimmed, nil
	}

	if _, err := os.Stat(trimmed); err == nil {
		return "file://" + trimmed, nil
	}

	if looksLikeSQLText(trimmed) {
		return "text://" + trimmed, nil
	}

	if isMySQLNativeDSN(trimmed) {
		return "mysql://" + trimmed, nil
	}

	return "", fmt.Errorf(
		"schemasource: 无法识别数据源 %q:既不是带 scheme 的 DSN、也不是已存在的文件、也不是 SQL 文本。\n"+
			"支持的写法: mysql://root:pass@tcp(host:port)/db、root:pass@tcp(host:port)/db、"+
			"postgres://user:pass@host:port/db(PostgreSQL 不接受 key=value 形式)、file://路径、text://DDL 正文",
		truncateForError(trimmed))
}

// looksLikeSQLText 只看起手:DDL 脚本总是以语句关键字或注释开头,
// 而 DSN 与路径不会。整段正文里出现 @tcp(...) 的注释不算 DSN——所以这一步排在 DSN 识别之前。
func looksLikeSQLText(s string) bool {
	upper := strings.ToUpper(s)
	for _, p := range sqlTextPrefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	return false
}

// isMySQLNativeDSN 认 go-sql-driver 的原生格式。判据是协议标记:
// 只有 tcp(/unix(/linux( 这种写法能确定它是连接串而不是别的什么东西,
// 而 mysql.ParseDSN 过于宽松(裸 "db/x" 也能解析成库名),不能单独作依据。
func isMySQLNativeDSN(s string) bool {
	lower := strings.ToLower(s)
	for _, marker := range []string{"@tcp(", "@unix(", "@linux("} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, marker := range []string{"tcp(", "unix(", "linux("} {
		if strings.HasPrefix(lower, marker) {
			return true
		}
	}
	return false
}

// truncateForError 报错里回显用户输入就够了,一段 DDL 全文会把日志冲掉。
func truncateForError(s string) string {
	if len(s) <= 120 {
		return s
	}
	return s[:120] + "…"
}
