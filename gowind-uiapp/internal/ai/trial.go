package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto"
)

// trialConvertDDL 用真实的 proto 生成管线当 DDL 的"语义裁判":
// ValidateDDLDetailed(tidb parser)只保证语法,这里再走一遍建表解析与列类型映射
// (sqlproto 的 text 驱动 → InspectSchema → convertTable),能拦下"语法正确但
// 下游管线消费不了"的产物。exportProto=false,只校验不渲染业务文件;
// 唯一的磁盘副作用是临时输出目录,函数返回前清理。
func trialConvertDDL(sqlContent string) error {
	return trialFn(sqlContent)
}

// trialFn 间接层:测试用桩替换,生产恒为 sqlTrial。
var trialFn = sqlTrial

func sqlTrial(sqlContent string) error {
	tmpDir, err := os.MkdirTemp("", "gowind-ai-trial-")
	if err != nil {
		return fmt.Errorf("创建试生成临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// text:// 后缀不是已存在的文件路径时视作内联 DDL 原样解析。
	dsn := "text://" + sqlContent
	out := tmpDir
	module := "trial.gowind.local/v1"
	if _, err := sqlproto.Convert(context.Background(),
		&dsn, &out,
		&module, nil, nil, nil,
		"", nil, nil, nil,
		false,
	); err != nil {
		return fmt.Errorf("DDL 无法被生成管线消费: %w", err)
	}
	return nil
}
