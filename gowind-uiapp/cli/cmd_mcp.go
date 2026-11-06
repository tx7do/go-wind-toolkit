package cli

import (
	"os"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/mcp"
	"github.com/spf13/cobra"
)

// mcpCmd MCP stdio 服务:stdout 只走协议帧,日志走 stderr(见 mcp 包纪律)。
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "以 MCP stdio 服务模式运行，把 AI 能力暴露为工具",
	Long: `以 MCP(Model Context Protocol) stdio 服务模式运行。

在 MCP 客户端(如 ZCode/Claude/Cursor)的 mcpServers 配置中指向本命令即可
把 DDL 生成/校验、微服务划分、后端代码生成计划/应用、项目代码审查等能力
作为工具调用。

AI 凭据:持久化配置(%APPDATA%/go-wind-toolkit/ai-config.json)为底,
GOWIND_AI_PROVIDER/_BASE_URL/_API_KEY/_AZURE_API_VERSION/_MODEL 环境变量
覆盖(不回写)。

注意:本命令的 stdout 是协议通道,不要人工解析;日志在 stderr。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logf("MCP stdio 服务启动,等待协议帧...")
		return mcp.NewServer().Run(os.Stdin, os.Stdout)
	},
}
