package schemasource

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertPostgresKeyValueToURL_RoundTrips 转换后的 URL 必须能被解析回同样的
// 分量——口令里带 / : @ # % + 这类字符时,拼出来的串很容易把 host 或 path 改掉。
func TestConvertPostgresKeyValueToURL_RoundTrips(t *testing.T) {
	for _, password := range []string{"plain", "p@ss", "p:ss", "p/ss", "p#ss", "p%ss", "p+ss", "p=ss&x=1"} {
		t.Run(password, func(t *testing.T) {
			dsn := convertPostgresKeyValueToURL("host=db.internal port=6000 user=app password=" + password + " dbname=mydb")

			u, err := url.Parse(dsn)
			require.NoError(t, err, "生成的 URL 无法解析: %s", dsn)
			assert.Equal(t, "db.internal:6000", u.Host, "host 被口令里的字符改写掉了: %s", dsn)
			assert.Equal(t, "/mydb", u.Path, "path 被改写掉了: %s", dsn)
			require.NotNil(t, u.User)
			assert.Equal(t, "app", u.User.Username())
			got, ok := u.User.Password()
			require.True(t, ok, "口令丢失: %s", dsn)
			assert.Equal(t, password, got, "口令没有原样往返: %s", dsn)
		})
	}
}

// TestConvertPostgresKeyValueToURL_StableOrder 剩余参数来自 map 遍历,顺序必须稳定,
// 否则同一份配置每次生成的 DSN 都不一样。
func TestConvertPostgresKeyValueToURL_StableOrder(t *testing.T) {
	in := "user=app dbname=mydb sslmode=disable search_path=public application_name=gow"
	first := convertPostgresKeyValueToURL(in)
	for i := 0; i < 30; i++ {
		require.Equal(t, first, convertPostgresKeyValueToURL(in), "第 %d 次生成不一致", i+1)
	}
}

// TestNormalizeDSNWithDriver 表驱动固定归一契约:
// 自描述源(scheme/SQL 正文)不受 driver 门控;driver 只对裸 DSN 生效。
func TestNormalizeDSNWithDriver(t *testing.T) {
	cases := []struct {
		name   string
		source string
		driver string
		want   string
	}{
		{"已有 scheme 原样返回", "ent://app/core/service/internal/data/ent/schema", "", "ent://app/core/service/internal/data/ent/schema"},
		{"已有 scheme 时 driver 不参与", "mysql://root@tcp(localhost:3306)/test", "postgres", "mysql://root@tcp(localhost:3306)/test"},
		{"SQL 正文补 text://(不受 driver 门控)", "CREATE TABLE users (id bigint NOT NULL);", "sqlite", "text://CREATE TABLE users (id bigint NOT NULL);"},
		{"SQL 正文(注释起手)补 text://", "-- schema\nCREATE TABLE users (id int);", "mysql", "text://-- schema\nCREATE TABLE users (id int);"},
		{"裸源无 driver 原样放行(下游接受面更宽)", "some_bare_source", "", "some_bare_source"},
		{"mysql 裸 DSN 补 scheme", "root@tcp(localhost:3306)/test", "mysql", "mysql://root@tcp(localhost:3306)/test"},
		{"driver 大小写不敏感", "root@tcp(localhost:3306)/test", "MySQL", "mysql://root@tcp(localhost:3306)/test"},
		{"postgresql key-value 转 URL", "host=localhost port=5432 user=postgres dbname=mydb", "postgresql", "postgres://postgres@localhost:5432/mydb"},
		{"postgres 别名同样转 URL", "host=localhost port=5432 user=postgres dbname=mydb", "postgres", "postgres://postgres@localhost:5432/mydb"},
		{"postgres 非 key-value 只补 scheme", "localhost:5432/mydb", "postgres", "postgres://localhost:5432/mydb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeDSNWithDriver(c.source, c.driver)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// TestNormalizeDSNWithDriver_RejectsUnknownDriver 需要 driver 提示却给了
// 本管线没有 provider 的驱动时必须立刻报错:放行会被当成 SQL 正文
// "成功生成零张表",调用方误以为生成成功。
func TestNormalizeDSNWithDriver_RejectsUnknownDriver(t *testing.T) {
	for _, driver := range []string{"invalid_driver", "sqlite", "oracle"} {
		t.Run(driver, func(t *testing.T) {
			got, err := NormalizeDSNWithDriver("some_bare_source", driver)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "不支持的 driver")
			assert.Contains(t, err.Error(), driver)
			assert.Empty(t, got)
		})
	}
}

// TestNormalizeDSNWithDriver_Empty 空数据源恒报错,与 driver 无关。
func TestNormalizeDSNWithDriver_Empty(t *testing.T) {
	_, err := NormalizeDSNWithDriver("   ", "mysql")
	require.Error(t, err)
}
