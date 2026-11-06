package ai

import "testing"

// testDDL 供各测试复用的最小合法 DDL:三张表,含反引号与注释。
// 不带前导换行——cleanMarkdownFence 会 TrimSpace,常量须与清理后的产物一致。
const testDDL = `CREATE TABLE ` + "`user`" + ` (
  ` + "`id`" + ` BIGINT NOT NULL AUTO_INCREMENT,
  ` + "`name`" + ` VARCHAR(50) NOT NULL,
  PRIMARY KEY (` + "`id`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户';

CREATE TABLE ` + "`user_role`" + ` (
  ` + "`id`" + ` BIGINT NOT NULL AUTO_INCREMENT,
  ` + "`user_id`" + ` BIGINT NOT NULL,
  ` + "`role`" + ` VARCHAR(50) NOT NULL,
  PRIMARY KEY (` + "`id`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色';

CREATE TABLE ` + "`order`" + ` (
  ` + "`id`" + ` BIGINT NOT NULL AUTO_INCREMENT,
  ` + "`user_id`" + ` BIGINT NOT NULL,
  ` + "`amount`" + ` DECIMAL(10,2) NOT NULL,
  PRIMARY KEY (` + "`id`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单';`

func TestValidateDDL(t *testing.T) {
	names, err := ValidateDDL(testDDL)
	if err != nil {
		t.Fatalf("合法 DDL 不应报错: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("应解析出 3 张表,得到 %d: %v", len(names), names)
	}
	if names[0] != "user" || names[1] != "user_role" || names[2] != "order" {
		t.Fatalf("表名或顺序不符: %v", names)
	}
}

func TestValidateDDL_Errors(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"空内容", "   "},
		{"无 CREATE TABLE", "SELECT 1;"},
		{"纯解释文字", "抱歉,我无法生成 DDL。"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ValidateDDL(c.sql); err == nil {
				t.Fatalf("%q 不应通过校验", c.sql)
			}
		})
	}
}

func TestMatchTableName(t *testing.T) {
	ddlTables := []string{"user", "user_role", "order"}

	if got, ok := matchTableName("`USER`", ddlTables); !ok || got != "user" {
		t.Fatalf("反引号+大写应归一化匹配并返回 DDL 原名,得到 %q,%v", got, ok)
	}
	if _, ok := matchTableName("ghost", ddlTables); ok {
		t.Fatal("幻觉表名不应匹配成功")
	}
	if _, ok := matchTableName("  ", ddlTables); ok {
		t.Fatal("空白表名不应匹配成功")
	}
}
