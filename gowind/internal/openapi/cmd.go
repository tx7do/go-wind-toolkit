// Package openapi 实现 `gow openapi`：用 gnostic 的 protoc-gen-openapi
// 从项目的 proto 定义一键产出 OpenAPI 3.0 文档，为 `gow frontend gen`
// 供料——打通 DDL → 后端 → OpenAPI → 前端的纯 CLI 全栈闭环。
package openapi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/buf"
)

var openapiOut string

// CmdOpenAPI `gow openapi` 命令。
var CmdOpenAPI = &cobra.Command{
	Use:   "openapi [dir]",
	Short: "从 proto 定义生成 OpenAPI 3.0 文档",
	Long: `在指定目录(默认 ./api)运行 buf generate,仅启用 protoc-gen-openapi
(gnostic)插件,把 google.api.http 注解的 proto 服务转换为一份 OpenAPI 3.0
文档,供前端生成(gow frontend)或文档站点消费。

产物默认写在 <dir>/openapi.yaml;--out 可指定目标文件或目录(目录时写入
<out>/openapi.yaml,例如 app/admin/service/cmd/server/assets)。

依赖:
  buf        https://buf.build/docs/installation
  protoc-gen-openapi
             go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         Run,
	SilenceUsage: true,
}

func Run(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	apiPath := "./api"
	if len(args) == 1 {
		apiPath = args[0]
	}
	if _, err := os.Stat(filepath.Join(apiPath, "buf.yaml")); err != nil {
		return fmt.Errorf("%s 下没有 buf.yaml——请指向模块的 api 目录(项目根目录运行时无需参数)", apiPath)
	}

	if _, err := exec.LookPath("protoc-gen-openapi"); err != nil {
		return fmt.Errorf("protoc-gen-openapi 未安装: go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest")
	}

	tmpDir, err := os.MkdirTemp("", "gow-openapi-")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpOut := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(tmpOut, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	tmpl := filepath.Join(tmpDir, "buf.gen.yaml")
	tmplBody := buildBufTemplate(tmpOut)
	if err := os.WriteFile(tmpl, []byte(tmplBody), 0o600); err != nil {
		return fmt.Errorf("写入 buf 模板失败: %w", err)
	}

	ctx := context.Background()
	if err := buf.RunBufGenerate(ctx, apiPath, tmpl); err != nil {
		return err
	}

	generated := filepath.Join(tmpOut, "openapi.yaml")
	if _, err := os.Stat(generated); err != nil {
		return fmt.Errorf("插件未产出 openapi.yaml——确认目录内的 proto 服务带 google.api.http 注解")
	}

	target := resolveTarget(apiPath, openapiOut)
	data, err := os.ReadFile(generated)
	if err != nil {
		return fmt.Errorf("读取生成产物失败: %w", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", target, err)
	}

	fmt.Fprintf(out, "\033[32mSUCCESS: OpenAPI 文档已生成 -> %s\033[m\n", target)
	fmt.Fprintf(out, "后续可运行: gow frontend gen --framework vue-vben --out <前端src目录>\n")
	return nil
}

// buildBufTemplate 构造临时 buf.gen.yaml:
//   - managed 块只为满足 protogen 的 go_package 校验(很多项目靠 buf managed
//     模式统一注入 go_package,极简模板会报 "unable to determine Go import
//     path");protoc-gen-openapi 的产物内容不受 go_package 影响。
//   - strategy: all 必须——buf v2 缺省按文件并发调用插件,每次调用各自产出
//     同名 openapi.yaml,buf 只保留第一个(往往是空文档)。
func buildBufTemplate(tmpOut string) string {
	return "version: v2\n" +
		"managed:\n  enabled: true\n  override:\n    - file_option: go_package_prefix\n      value: generated.openapi.local\n" +
		"plugins:\n  - local: protoc-gen-openapi\n    out: " + filepath.ToSlash(tmpOut) + "\n    strategy: all\n"
}

// resolveTarget 解析最终输出路径:out 为空 → <api>/openapi.yaml;
// out 是已存在目录 → <out>/openapi.yaml;否则 out 原样作为文件路径。
func resolveTarget(apiPath, out string) string {
	target := out
	if target == "" {
		target = filepath.Join(apiPath, "openapi.yaml")
	}
	if st, serr := os.Stat(target); serr == nil && st.IsDir() {
		target = filepath.Join(target, "openapi.yaml")
	}
	return target
}

func init() {
	CmdOpenAPI.Flags().StringVar(&openapiOut, "out", "", "输出目标(文件或目录;缺省 <dir>/openapi.yaml)")
}
