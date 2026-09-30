package schemasource

import (
	"strings"
	"testing"
)

func TestBuildMySQLDDL(t *testing.T) {
	ps := parseGormFixture(t)
	stmts := BuildMySQLDDL(ps)

	byName := map[string]string{}
	for _, s := range stmts {
		head := strings.TrimPrefix(s, "CREATE TABLE `")
		byName[strings.SplitN(head, "`", 2)[0]] = s
	}

	users := byName["users"]
	if users == "" {
		t.Fatal("no CREATE TABLE for users")
	}
	for _, want := range []string{
		"`id` bigint unsigned NOT NULL AUTO_INCREMENT",
		"PRIMARY KEY (`id`)",
		"`name` varchar(64) NOT NULL COMMENT '用户名'",
		"`age` bigint DEFAULT 0",
		"`deleted_at` datetime",
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	} {
		if !strings.Contains(users, want) {
			t.Errorf("users DDL missing %q:\n%s", want, users)
		}
	}

	posts := byName["blog_posts"]
	for _, want := range []string{
		"`status` varchar(255) DEFAULT 'draft'",
		"`views` bigint DEFAULT 0",
		"`uuid` char(36) NOT NULL",
		"`title` varchar(200) NOT NULL COMMENT '标题'",
	} {
		if !strings.Contains(posts, want) {
			t.Errorf("blog_posts DDL missing %q:\n%s", want, posts)
		}
	}

	// 复合主键中间表
	jt := byName["user_groups"]
	if !strings.Contains(jt, "PRIMARY KEY (`group_id`, `user_id`)") {
		t.Errorf("user_groups DDL missing composite PK:\n%s", jt)
	}

	if !strings.Contains(strings.Join(stmts, ";\n"), "CREATE TABLE `pets`") {
		t.Error("expected pets table in DDL output")
	}
}
