package schemasource

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	atlassqlite "ariga.io/atlas/sql/sqlite"
	"entgo.io/ent/dialect"

	// 纯 Go 的 sqlite 驱动:CLI 经 go install 分发,不能要求用户机器有
	// CGO 工具链(mattn/go-sqlite3 在无 gcc 的 Windows 上装不上)。
	_ "modernc.org/sqlite"
)

// sqliteProvider 打开本地 SQLite 数据库文件。DSN 形如
// sqlite://<文件路径>(sqlite3:// 等价);相对路径按进程 CWD 解析。
// 内存库(":memory:")同样可用,适合测试。
func sqliteProvider(dsn string) (*Driver, error) {
	path := strings.TrimPrefix(dsn, "sqlite3://")
	path = strings.TrimPrefix(path, "sqlite://")
	if path == "" {
		return nil, fmt.Errorf("sqlite 数据源缺少文件路径(形如 sqlite://data.db)")
	}
	// ":memory:" 是驱动的内存库约定;其余路径要求存在,防止拼错路径后
	// 静默建一个空库、生成出零张表还"成功"。
	if path != ":memory:" {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("sqlite 文件不存在: %s", path)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	drv, err := atlassqlite.Open(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Driver{
		Closer:    db,
		Inspector: drv,
		Dialect:   dialect.SQLite,
	}, nil
}

func init() {
	Default.RegisterProvider(sqliteProvider, "sqlite", "sqlite3")
}
