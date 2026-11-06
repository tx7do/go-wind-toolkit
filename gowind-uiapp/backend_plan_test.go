package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

func newPlanTestApp(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := &App{generator: generator.NewGenerator(), aiService: ai.NewService()}
	app.setProjectInfo(&detect.ProjectInfo{Root: root, ModPath: "example.com/test"})
	return app, root
}

func planTestPartitions() []ai.MicroservicePartition {
	return []ai.MicroservicePartition{
		{ServiceName: "user", Tables: []string{"user", "user_role"}, Description: "用户"},
		{ServiceName: "order", Tables: []string{"order"}, Description: "订单"},
	}
}

const planTestDDL = "CREATE TABLE `user` (`id` BIGINT PRIMARY KEY AUTO_INCREMENT, `name` VARCHAR(50) NOT NULL) COMMENT='用户';\n" +
	"CREATE TABLE `user_role` (`id` BIGINT PRIMARY KEY AUTO_INCREMENT, `user_id` BIGINT NOT NULL) COMMENT='角色';\n" +
	"CREATE TABLE `order` (`id` BIGINT PRIMARY KEY AUTO_INCREMENT, `user_id` BIGINT NOT NULL) COMMENT='订单';\n"

func TestAIPlanBackendCode(t *testing.T) {
	app, root := newPlanTestApp(t)

	// 预置一个生成树里必然出现的路径,验证覆盖(非新增)语义
	preexisting := filepath.Join(root, "app", "user", "service", "Makefile")
	if err := os.MkdirAll(filepath.Dir(preexisting), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(preexisting, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := app.AIPlanBackendCode(planTestDDL, "ent", planTestPartitions())
	if !res.Success {
		t.Fatalf("预览失败: %s", res.Error)
	}
	if res.PlanID == "" || len(res.Files) == 0 {
		t.Fatalf("计划 ID 与文件清单不应为空: %d 个文件", len(res.Files))
	}
	if res.NewCount+res.OverwriteCount != len(res.Files) {
		t.Fatalf("新增+覆盖应等于文件总数: %d+%d != %d", res.NewCount, res.OverwriteCount, len(res.Files))
	}

	// 必然生成的文件:Makefile(覆盖)与 order 服务 main.go(新增)
	foundMakefile, foundOrder := false, false
	for _, f := range res.Files {
		if f.Path == "app/user/service/Makefile" {
			foundMakefile = true
			if f.IsNew {
				t.Error("预置的 Makefile 应标注为覆盖")
			}
		}
		if strings.HasPrefix(f.Path, "app/order/") && f.Path != "app/user/service/Makefile" {
			foundOrder = foundOrder || f.IsNew
		}
	}
	if !foundMakefile {
		t.Fatal("文件清单应包含 app/user/service/Makefile")
	}
	if !foundOrder {
		t.Fatal("order 服务文件应为新增")
	}

	// 后处理说明:tidy/buf + 双服务 ent + 双服务 wire
	joined := strings.Join(res.PostProcess, ";")
	for _, want := range []string{"go mod tidy", "buf generate", "ent generate (user)", "wire (order)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("后处理说明应包含 %q,得到: %s", want, joined)
		}
	}

	// 预览文件可读且内容是新生成物(非预置的 "old")
	rc := app.AIReadBackendPlanFile(res.PlanID, "app/user/service/Makefile")
	if !rc.Success || rc.Content == "old" {
		t.Fatalf("预览文件应可读且为生成内容: %v", rc.Error)
	}

	// 穿越路径拒绝
	if r := app.AIReadBackendPlanFile(res.PlanID, "../escape.txt"); r.Success {
		t.Fatal("越出计划目录的路径不应可读")
	}

	app.AIDiscardBackendPlan(res.PlanID)
	if r := app.AIReadBackendPlanFile(res.PlanID, "app/user/service/Makefile"); r.Success {
		t.Fatal("丢弃后计划应不可读")
	}
}

func TestAIPlanBackendCode_InvalidServiceName(t *testing.T) {
	app, _ := newPlanTestApp(t)

	bad := []ai.MicroservicePartition{{ServiceName: "a/b", Tables: []string{"user"}}}
	if res := app.AIPlanBackendCode(planTestDDL, "ent", bad); res.Success {
		t.Fatal("非法服务名应被拒收")
	}
}

func TestAIPlanBackendCode_ReplacesPreviousPlan(t *testing.T) {
	app, _ := newPlanTestApp(t)

	res1 := app.AIPlanBackendCode(planTestDDL, "ent", planTestPartitions())
	if !res1.Success {
		t.Fatalf("第一次预览失败: %s", res1.Error)
	}
	res2 := app.AIPlanBackendCode(planTestDDL, "ent", planTestPartitions())
	if !res2.Success {
		t.Fatalf("第二次预览失败: %s", res2.Error)
	}
	if res1.PlanID == res2.PlanID {
		t.Fatal("两次预览的计划 ID 应不同")
	}
	// 旧计划应已作废
	if r := app.AIReadBackendPlanFile(res1.PlanID, "app/user/service/Makefile"); r.Success {
		t.Fatal("新预览后旧计划应失效")
	}
}

func TestAIApplyBackendCode_StalePlanRejected(t *testing.T) {
	app, _ := newPlanTestApp(t)

	if err := app.AIApplyBackendCode("nonexistent"); err == "" || !strings.Contains(err, "不存在或已过期") {
		t.Fatalf("不存在的计划应报过期错误,得到: %q", err)
	}
}

func TestAIApplyBackendCode_CopiesThenPostProcessFails(t *testing.T) {
	// 断网环境:测试夹具是零依赖空模块,go mod tidy 离线即可完成;
	// 避免真实代理(goproxy.cn)抖动把测试挂死。
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOTOOLCHAIN", "local")

	app, root := newPlanTestApp(t)

	res := app.AIPlanBackendCode(planTestDDL, "ent", planTestPartitions())
	if !res.Success {
		t.Fatalf("预览失败: %s", res.Error)
	}

	// 项目没有 api/buf.yaml,buf generate 必失败——断言拷贝阶段已完成、
	// 错误串注明"文件已写入"、计划已清理。
	err := app.AIApplyBackendCode(res.PlanID)
	if err == "" || !strings.Contains(err, "文件已写入") {
		t.Fatalf("应用应报带\"文件已写入\"前缀的错误,得到: %q", err)
	}
	data, rerr := os.ReadFile(filepath.Join(root, "app", "user", "service", "Makefile"))
	if rerr != nil || string(data) == "old" {
		t.Fatalf("Makefile 应已被拷贝覆盖: %v", rerr)
	}
	if r := app.AIReadBackendPlanFile(res.PlanID, "app/user/service/Makefile"); r.Success {
		t.Fatal("应用后(无论成败)计划应清理")
	}
}
