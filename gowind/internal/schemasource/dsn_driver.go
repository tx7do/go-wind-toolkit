package schemasource

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeDSNWithDriver 是 NormalizeDSN 的 driver 提示版:调用方已从
// 独立参数(--driver)知道数据源类别时,用它消除"同一段内容到底按什么
// 解释"的歧义。归一顺序:
//
//  1. 已有 scheme 原样返回;
//  2. SQL 正文补 text://(排在 driver 之前:--driver mysql + 粘贴的 DDL
//     是真实存在的用法,不能把正文当 DSN 拼前缀);
//  3. driver 生效:mysql 补 mysql://;postgres 先做 key-value DSN → URL
//     的转换,再补 postgres://;
//  4. driver 为空且上面都未命中时**原样返回**——不认识的裸源交给下游再判:
//     个别分支(如 sqlorm 的 gorm 路径)的接受面比 NormalizeDSN 宽,
//     这里提前收紧会把合法输入拦死;
//  5. driver 不认识必须报错——放行会被当成 SQL 正文"成功生成零张表"。
func NormalizeDSNWithDriver(dsn, driver string) (string, error) {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return "", fmt.Errorf("schemasource: 数据源为空")
	}
	if strings.Contains(trimmed, "://") {
		return trimmed, nil
	}
	if looksLikeSQLText(trimmed) {
		return "text://" + trimmed, nil
	}

	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "":
		return trimmed, nil
	case "mysql":
		return "mysql://" + trimmed, nil
	case "postgresql", "postgres":
		if isPostgresKeyValueDSN(trimmed) {
			return convertPostgresKeyValueToURL(trimmed), nil
		}
		return "postgres://" + trimmed, nil
	default:
		return "", fmt.Errorf(
			"schemasource: 不支持的 driver %q(支持 mysql、postgres、postgresql;数据源自带 scheme 时请留空)", driver)
	}
}

// isPostgresKeyValueDSN 检测 PostgreSQL key-value 形式 DSN:
// 空格分隔的 key=value 对,如 "host=localhost port=5432 user=postgres"。
func isPostgresKeyValueDSN(dsn string) bool {
	return strings.Contains(dsn, "=") && strings.Contains(dsn, " ")
}

// convertPostgresKeyValueToURL 把 PostgreSQL key-value DSN 转成 URL 形式。
// Input:  "host=localhost port=5432 user=postgres password=xxx dbname=mydb sslmode=disable"
// Output: "postgres://postgres:xxx@localhost:5432/mydb?sslmode=disable"
//
// 编码交给 url.URL:手写 url.QueryEscape 走的是表单规则,空格编成 "+",而 userinfo
// 与 path 里的 "+" 是字面量,含空格或 "+" 的口令会被原样送到服务端。
func convertPostgresKeyValueToURL(dsn string) string {
	parts := strings.Fields(dsn)
	vals := make(map[string]string)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			vals[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	host := vals["host"]
	if host == "" {
		host = "localhost"
	}
	port := vals["port"]
	if port == "" {
		port = "5432"
	}
	user := vals["user"]
	password := vals["password"]
	dbname := vals["dbname"]

	u := &url.URL{
		Scheme: "postgres",
		Host:   host + ":" + port,
	}
	switch {
	case user != "" && password != "":
		u.User = url.UserPassword(user, password)
	case user != "":
		// 无口令时不能留 "user:" 的空冒号:那会被解成一个空口令。
		u.User = url.User(user)
	}
	if dbname != "" {
		u.Path = "/" + dbname
	}

	// Values.Encode 按键排序,同一份 DSN 每次得到同一个字符串。
	params := url.Values{}
	for k, v := range vals {
		switch k {
		case "host", "port", "user", "password", "dbname":
			// already handled
		default:
			params.Set(k, v)
		}
	}
	u.RawQuery = params.Encode()

	return u.String()
}
