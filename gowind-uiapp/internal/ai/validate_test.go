package ai

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestValidateDDLDetailed_SyntaxError(t *testing.T) {
	// 缺右括号的真语法错:tidb parser 应报错且带行列
	broken := "CREATE TABLE `user` (\n  `id` BIGINT NOT NULL,\n  `name` VARCHAR(50)\n"

	tables, diags, err := ValidateDDLDetailed(broken)
	if err == nil {
		t.Fatalf("残缺 DDL 应报语法错误")
	}
	if tables != nil {
		t.Fatalf("语法错时不应返回表清单: %v", tables)
	}
	if len(diags) != 1 {
		t.Fatalf("应有一条诊断,得到 %d: %+v", len(diags), diags)
	}
	if diags[0].Line < 1 {
		t.Fatalf("诊断应带行号,得到 %+v", diags[0])
	}
	if !strings.Contains(err.Error(), "line") && !strings.Contains(diags[0].Message, "near") {
		t.Fatalf("错误消息应含 tidb parser 的位置描述,得到: %s", err.Error())
	}
}

func TestValidateDDLDetailed_PositionExtraction(t *testing.T) {
	// 两种错误格式的提取
	d1 := extractDiagnostic(fmt.Errorf(`line 3 column 15 near "FOO" ...`))
	if d1.Line != 3 || d1.Column != 15 {
		t.Fatalf("grammar 错格式提取失败: %+v", d1)
	}
	d2 := extractDiagnostic(fmt.Errorf(`near '(?' at line 7`))
	if d2.Line != 7 || d2.Column != 0 {
		t.Fatalf("token 错格式提取失败: %+v", d2)
	}
	d3 := extractDiagnostic(fmt.Errorf("没有位置信息的错误"))
	if d3.Line != 0 || d3.Column != 0 {
		t.Fatalf("无位置错误应得 0/0: %+v", d3)
	}
}

func TestValidateDDLDetailed_NonMySQLSyntax(t *testing.T) {
	// Oracle 方言不属于 MySQL 语法,应被拒(对 MySQL 校验器是特性)
	oracle := "CREATE TABLE t (name VARCHAR2(50));"
	if _, _, err := ValidateDDLDetailed(oracle); err == nil {
		t.Fatal("Oracle 方言应被拒收")
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
