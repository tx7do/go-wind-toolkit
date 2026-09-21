package sqlproto

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// converterFixtureDDL 是裸建表语句:NormalizeDSN 认不出 scheme 也认不出文件,会补上
// text:// 前缀走离线分支,因此用例不连任何数据库。
// 原用例连的是硬编码口令的 Postgres、且对返回值与产物零断言,连不上也恒绿。
var converterFixtureDDL = `CREATE TABLE users (
	id bigint(20) NOT NULL AUTO_INCREMENT,
	name varchar(100) NOT NULL,
	total decimal(10,2) NOT NULL,
	created_at datetime NOT NULL,
	uid bigint(20) unsigned NOT NULL,
	PRIMARY KEY (id)
);

CREATE TABLE order_logs (
	id bigint(20) NOT NULL AUTO_INCREMENT,
	note text NULL,
	PRIMARY KEY (id)
);`

func TestConvert_FromDDLText_WritesPerTableProto(t *testing.T) {
	out := t.TempDir()
	dsn := converterFixtureDDL
	moduleName, sourceModuleName, moduleVersion, serviceType := "admin", "user", "v1", "grpc"

	tables, err := Convert(
		context.Background(),
		&dsn, &out,
		&moduleName, &sourceModuleName, &moduleVersion,
		&serviceType,
		"per-table",
		nil, nil, nil,
		true,
	)
	require.NoError(t, err)

	names := make([]string, 0, len(tables))
	for _, tb := range tables {
		names = append(names, tb.Name)
	}
	assert.Equal(t, []string{"users", "order_logs"}, names, "两张表都应被探测到")

	// Convert 的 outputPath 就是 proto 落地目录,调用方(sqlkratos)负责拼 api/protos。
	contents := protoFileContents(t, out)
	assert.Equal(t, []string{"order_log.proto", "user.proto"}, sortedBasenames(contents),
		"per-table 策略下每张表产出一个单数命名的 proto")

	// 断言前先打印,红的时候一次就能看清真实内容,不用再来一轮 CI。
	t.Logf("生成的 user.proto:\n%s", contents["user.proto"])
	assert.Contains(t, contents["user.proto"], "message User")
	// 列类型 → proto 类型的映射:bigint→int64、varchar→string、
	// decimal→string(proto 无 decimal)、datetime→Timestamp。
	assert.Contains(t, contents["user.proto"], "optional int64 id = 1")
	assert.Contains(t, contents["user.proto"], "optional string name = 2")
	assert.Contains(t, contents["user.proto"], "optional string total = 3")
	assert.Contains(t, contents["user.proto"], "optional google.protobuf.Timestamp created_at = 4")
	// `bigint(20) unsigned` 必须是 uint64:先剥括号的旧实现会连 unsigned 一起剥掉,
	// 而列名 id 又会在整文件子串搜索里命中 uid 那行,两个缺陷都落在这一列上。
	assert.Contains(t, contents["user.proto"], "optional uint64 uid = 5")
	// 表名 order_logs 要先转单数 order_log,再作为包名与消息名落地。
	assert.Contains(t, contents["order_log.proto"], "message OrderLog")
	assert.Contains(t, contents["order_log.proto"], "package order_log.service.v1")
	assert.Contains(t, contents["order_log.proto"], "optional string note = 2")
}

// 连接串分支必须把连不上库显式报错。原用例把错误丢给 `_ =`,配置写错与生成成功
// 在日志里完全同形。
func TestConvert_UnreachableDatabaseFailsLoudly(t *testing.T) {
	out := t.TempDir()
	dsn := "postgres://postgres@127.0.0.1:1/example?sslmode=disable"
	moduleName, sourceModuleName, moduleVersion, serviceType := "admin", "user", "v1", "grpc"

	_, err := Convert(
		context.Background(),
		&dsn, &out,
		&moduleName, &sourceModuleName, &moduleVersion,
		&serviceType,
		"per-table",
		nil, nil, nil,
		true,
	)
	require.Error(t, err, "连不上数据库时 Convert 必须返回错误")

	// 失败要失败得干净:报错后不留半截 proto。
	requireNoProtoFiles(t, out)
}

// TestConvert_ProtoFieldsKeepPresence 钉住"模型 message 的每个字段都带 optional"这条
// 不变量。它看起来像缺陷(NOT NULL 列也是 optional),把 optional 按列的可空性收窄
// 却会让生成物编译不过:更新路径对每一列都发 SetNillableXxx(req.Data.Xxx),而标量
// 字段只有带 presence 时才是指针(见 generators/types.go 的 ProtoField 注释)。
//
// 两个 message 都要断言:user.proto 全列 NOT NULL,order_log.proto 的 note 可空——
// 任何一侧丢掉 optional 都在这里转红。
func TestConvert_ProtoFieldsKeepPresence(t *testing.T) {
	out := t.TempDir()
	dsn := converterFixtureDDL
	moduleName, sourceModuleName, moduleVersion, serviceType := "admin", "user", "v1", "grpc"

	_, err := Convert(
		context.Background(),
		&dsn, &out,
		&moduleName, &sourceModuleName, &moduleVersion,
		&serviceType,
		"per-table",
		nil, nil, nil,
		true,
	)
	require.NoError(t, err)

	contents := protoFileContents(t, out)
	for _, c := range []struct{ file, message string }{
		{file: "user.proto", message: "User"},
		{file: "order_log.proto", message: "OrderLog"},
	} {
		decls := protoFieldDecls(t, contents[c.file], c.message)
		// 断言前先打印,红的时候一次就能看清真实内容,不用再来一轮 CI。
		t.Logf("%s 的 message %s 字段声明:\n%s", c.file, c.message, strings.Join(decls, "\n"))
		require.NotEmpty(t, decls, "%s 里没有 message %s", c.file, c.message)
		for _, decl := range decls {
			assert.True(t, strings.HasPrefix(decl, "optional "),
				"字段声明丢了 optional(生成的更新路径会编译不过): %s", decl)
		}
	}

	// 两侧的分布前提:NOT NULL 列与可空列各至少一条。
	assert.Contains(t, contents["user.proto"], "optional string name = 2")
	assert.Contains(t, contents["order_log.proto"], "optional string note = 2")
}

// protoFieldDecls 取出 message <name> 体内的字段声明行(已去缩进)。
func protoFieldDecls(t *testing.T, proto, name string) []string {
	t.Helper()

	lines := strings.Split(proto, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "message "+name+" {" {
			start = i + 1
			break
		}
	}
	require.NotEqual(t, -1, start, "找不到 message "+name)

	var decls []string
	for _, line := range lines[start:] {
		if strings.TrimSpace(line) == "}" {
			break
		}
		if protoFieldDeclRe.MatchString(line) {
			decls = append(decls, strings.TrimSpace(line))
		}
	}

	return decls
}

// protoFieldDeclRe 匹配字段声明首行:"[optional|repeated] <类型> <字段名> = <编号>",
// 其后跟 `[`(带注解)或 `;`。注解续行 json_name = "…" 与
// (gnostic.openapi.v3.property) = {…} 只有一个词落在 `=` 之前,不成这个形状。
var protoFieldDeclRe = regexp.MustCompile(`^\s*(?:optional |repeated )?[\w.]+ \w+ = \d+ [\[;]`)

func protoFileContents(t *testing.T, dir string) map[string]string {
	t.Helper()

	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".proto" {
			return err
		}
		b, readErr := os.ReadFile(p)
		require.NoError(t, readErr)
		files[filepath.Base(p)] = string(b)
		return nil
	}))

	return files
}

func sortedBasenames(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	// 稳定顺序便于精确比较
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}

func requireNoProtoFiles(t *testing.T, dir string) {
	t.Helper()

	for name := range protoFileContents(t, dir) {
		t.Fatalf("连接失败却留下了生成物: %s", name)
	}
}
