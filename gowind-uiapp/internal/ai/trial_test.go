package ai

import (
	"errors"
	"strings"
	"testing"
)

var errFakeConvert = errors.New("fake convert failure")

// TestTrialConvertDDL_AcceptsStandardMySQL 标准 MySQL 建表语句必须能被
// 真实生成管线消费——这是裁判的门面用例,若这里失败说明接入本身有问题。
func TestTrialConvertDDL_AcceptsStandardMySQL(t *testing.T) {
	if err := trialConvertDDL(testDDL); err != nil {
		t.Fatalf("标准 DDL 应通过试生成: %v", err)
	}
}

// TestTrialConvertDDL_RejectsMalformed 只靠 tidb parser 拦不住的"伪建表"
// 也可能被生成管线的建表解析拒绝:裁判要能独立否决。
func TestTrialConvertDDL_RejectsMalformed(t *testing.T) {
	if err := trialConvertDDL("CREATE TABLE"); err == nil {
		t.Fatal("残缺建表语句应被生成管线拒绝")
	}
}

// TestGenerateDDLValidated_ConvertRetry 校验通过但管线拒绝时,重试提示必须
// 走转换修复文案(而不是解析错误文案),且成功后不再多余调用。
func TestGenerateDDLValidated_ConvertRetry(t *testing.T) {
	// testDDL 本身可被管线消费,这里用桩替换裁判制造一次"管线否决"。
	orig := trialFn
	defer func() { trialFn = orig }()
	calls := 0
	trialFn = func(sql string) error {
		calls++
		if calls == 1 {
			return errFakeConvert
		}
		return nil
	}

	svc, mock := newTestService(testDDL, testDDL)
	result, err := svc.GenerateDDLValidated("用户管理", nil)
	if err != nil {
		t.Fatalf("第二次应成功: %v", err)
	}
	if !result.Success {
		t.Fatalf("最终应成功: %v", result.Error)
	}
	if calls != 2 {
		t.Fatalf("裁判应被调用 2 次,实际 %d", calls)
	}
	if !strings.Contains(mock.prompts[1], "cannot consume it") || !strings.Contains(mock.prompts[1], testDDL) {
		t.Fatalf("修复提示应是转换失败文案并带被否决的 DDL,得到: %.160s", mock.prompts[1])
	}
}
