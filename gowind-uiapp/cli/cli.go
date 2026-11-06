// Package cli 将 gowind-uiapp 桌面应用的能力（项目探测、数据库、后端/前端
// 代码生成、AI 助手、远程配置）以非交互命令行方式暴露。同一可执行文件:
// 不带参数启动 GUI,带参数进入本命令行模式。
//
// 约定:
//   - 结果输出到 stdout,日志输出到 stderr
//   - 默认输出缩进 JSON(人可读);--json 输出单行紧凑 JSON(机器友好)
//   - 退出码: 0 成功 / 1 失败 / 2 用法错误
//
// 项目脚手架与开发工具(buf/ent/wire/tidy)不在此重复,请使用 gow CLI
// (gow new / gow add service / gow api / gow ent / gow wire)。
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var version = "dev"

var jsonCompact bool

var rootCmd = &cobra.Command{
	Use:           "gowind-uiapp",
	Short:         "go-wind-toolkit 命令行模式（代码生成、数据库、AI、配置中心）；不带参数启动图形界面",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Run 以命令行模式执行,返回进程退出码。
func Run(args []string) int {
	attachParentConsole()

	rootCmd.PersistentFlags().BoolVar(&jsonCompact, "json", false, "以单行紧凑 JSON 输出结果（默认缩进 JSON）")

	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(backendCmd)
	rootCmd.AddCommand(frontendCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(mcpCmd)

	if err := rootCmd.Execute(); err != nil {
		return exitCodeFor(err)
	}
	return 0
}

// exitCodeFor 区分用法错误(2)与运行失败(1)。
func exitCodeFor(err error) int {
	if errors.Is(err, pflag.ErrHelp) {
		return 0
	}
	msg := err.Error()
	for _, marker := range []string{
		"unknown flag", "unknown command", "unknown shorthand flag",
		"flag needs an argument", "invalid argument", "accepts",
		"requires at least", "must be",
	} {
		if strings.Contains(msg, marker) {
			return 2
		}
	}
	fail(err)
	return 1
}

// emit 输出结果到 stdout（JSON）
func emit(v any) {
	var data []byte
	var err error
	if jsonCompact {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		fail(err)
	}
	fmt.Println(string(data))
}

// fail 输出错误信息（stderr）
func fail(err error) {
	if jsonCompact {
		fmt.Println(mustJSON(map[string]string{"error": err.Error()}))
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	}
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}
