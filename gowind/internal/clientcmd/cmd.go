// Package clientcmd 实现 `gow client`：用本仓库的 TS/Dart HTTP 客户端插件
// 从 proto 生成客户端 SDK,为前后端分离仓库补上"后端 → 前端类型"的 CLI 环节。
package clientcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/buf"
)

var clientOut string

// CmdClient `gow client` 命令组。
var CmdClient = &cobra.Command{
	Use:   "client",
	Short: "从 proto 生成前端客户端 SDK(TypeScript / Dart)",
	Long: `在指定目录(默认 ./api/protos)运行 buf generate,用本仓库的客户端插件
产出 HTTP 客户端代码:

  typescript  protoc-gen-typescript-http(纯接口,传输由调用方注入)
  dart        protoc-gen-dart-http(含二进制 wire 编解码)

产物为各 proto 包一份客户端(如 <out>/core/service/v1/index.ts),写入
--out 目录。依赖 buf 与对应插件(gow doctor 可体检):
  go install github.com/tx7do/go-wind-toolkit/protoc-gen-typescript-http@latest
  go install github.com/tx7do/go-wind-toolkit/protoc-gen-dart-http@latest`,
	SilenceUsage: true,
}

func clientCmd(target string, plugin string) *cobra.Command {
	return &cobra.Command{
		Use:   target + " [dir]",
		Short: "生成 " + target + " 客户端",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			protoPath := "./api/protos"
			if len(args) == 1 {
				protoPath = args[0]
			}
			if _, err := os.Stat(filepath.Join(protoPath, "buf.yaml")); err != nil {
				return fmt.Errorf("%s 下没有 buf.yaml——请指向模块的 proto 目录", protoPath)
			}
			if _, err := exec.LookPath(plugin); err != nil {
				return fmt.Errorf("%s 未安装: go install github.com/tx7do/go-wind-toolkit/%s@latest", plugin, plugin)
			}
			if clientOut == "" {
				return fmt.Errorf("必须指定 --out(客户端输出目录)")
			}
			if err := os.MkdirAll(clientOut, 0o755); err != nil {
				return fmt.Errorf("创建输出目录失败: %w", err)
			}

			tmpDir, err := os.MkdirTemp("", "gow-client-")
			if err != nil {
				return fmt.Errorf("创建临时目录失败: %w", err)
			}
			defer func() { _ = os.RemoveAll(tmpDir) }()

			tmpl := filepath.Join(tmpDir, "buf.gen.yaml")
			// managed 兜底无 go_package 的 proto(protogen 强制要求,产物内容
			// 不受影响);strategy: all 必须——buf v2 按文件并发调用插件,
			// dart 的 transport.dart / ts 的 DEFAULT_HOST 常量文件会因重复名
			// 被丢弃。
			tmplBody := "version: v2\n" +
				"managed:\n  enabled: true\n  override:\n    - file_option: go_package_prefix\n      value: generated.client.local\n" +
				"plugins:\n  - local: " + plugin + "\n    out: " + filepath.ToSlash(clientOut) + "\n    strategy: all\n"
			if err := os.WriteFile(tmpl, []byte(tmplBody), 0o600); err != nil {
				return fmt.Errorf("写入 buf 模板失败: %w", err)
			}

			if err := buf.RunBufGenerate(context.Background(), protoPath, tmpl); err != nil {
				return err
			}

			fmt.Fprintf(out, "\033[32mSUCCESS: %s 客户端已生成 -> %s\033[m\n", target, clientOut)
			return nil
		},
	}
}

func init() {
	ts := clientCmd("typescript", "protoc-gen-typescript-http")
	dart := clientCmd("dart", "protoc-gen-dart-http")
	for _, c := range []*cobra.Command{ts, dart} {
		c.Flags().StringVar(&clientOut, "out", "", "客户端输出目录（必填）")
		CmdClient.AddCommand(c)
	}
}
