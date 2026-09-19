package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI 助手（DDL 生成、微服务划分、代码审查）",
}

func addAIFlags(cmd *cobra.Command) {
	cmd.Flags().String("provider", "", "AI 服务商: openai | deepseek | anthropic | gemini | ollama | azure | custom（环境变量 GOWIND_AI_PROVIDER）")
	cmd.Flags().String("base-url", "", "API 基础地址（环境变量 GOWIND_AI_BASE_URL）")
	cmd.Flags().String("api-key", "", "API 密钥（环境变量 GOWIND_AI_API_KEY）")
	cmd.Flags().String("azure-api-version", "", "Azure OpenAI 部署的 api-version（环境变量 GOWIND_AI_AZURE_API_VERSION）")
	cmd.Flags().String("model", "", "模型名称（环境变量 GOWIND_AI_MODEL）")
	cmd.Flags().Float64("temperature", 0.7, "温度参数 (0.0-2.0)")
	cmd.Flags().Int("max-tokens", 0, "最大 token 数")
}

// buildAIService 构建 AI 服务:持久化配置为底(GUI 里配置一次即生效),
// 环境变量 GOWIND_AI_* 覆盖,命令行旗标最高优先级。不回写配置文件。
func buildAIService(cmd *cobra.Command) *ai.Service {
	cfg := ai.LoadConfig()
	if v := flagString(cmd, "provider", envOr("GOWIND_AI_PROVIDER", "")); v != "" {
		cfg.Provider = v
	}
	if v := flagString(cmd, "base-url", envOr("GOWIND_AI_BASE_URL", "")); v != "" {
		cfg.BaseURL = v
	}
	if v := flagString(cmd, "api-key", envOr("GOWIND_AI_API_KEY", "")); v != "" {
		cfg.APIKey = v
	}
	if v := flagString(cmd, "azure-api-version", envOr("GOWIND_AI_AZURE_API_VERSION", "")); v != "" {
		cfg.AzureAPIVersion = v
	}
	if v := flagString(cmd, "model", envOr("GOWIND_AI_MODEL", "")); v != "" {
		cfg.Model = v
	}
	if cmd.Flags().Changed("temperature") {
		// 0 是合法取值(确定性输出),仅拒绝负数与超出 [0,2] 的值。
		if t, err := cmd.Flags().GetFloat64("temperature"); err == nil && t >= 0 && t <= 2 {
			cfg.Temperature = t
		}
	}
	if cmd.Flags().Changed("max-tokens") {
		if m, err := cmd.Flags().GetInt("max-tokens"); err == nil && m > 0 {
			cfg.MaxTokens = m
		}
	}

	svc := ai.NewService()
	svc.SetConfigTransient(cfg)
	return svc
}

var aiPresetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "列出 AI 服务商预设",
	Run: func(cmd *cobra.Command, args []string) {
		emit(ai.GetProviderPresets())
	},
}

var aiTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试 AI 连通性",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := buildAIService(cmd).TestConnection()
		if err != nil {
			return err
		}
		emit(result)
		return nil
	},
}

var aiDdlCmd = &cobra.Command{
	Use:   "ddl",
	Short: "根据需求文档生成 MySQL DDL",
	RunE: func(cmd *cobra.Command, args []string) error {
		reqFile := flagString(cmd, "requirements", "")
		if reqFile == "" {
			checkErr(fmt.Errorf("必须指定 --requirements（需求文档 Markdown/文本文件）"))
		}
		requirements, err := readFileContent(reqFile)
		if err != nil {
			checkErr(err)
		}

		svc := buildAIService(cmd)
		var result *ai.StepResult
		if boolFlag(cmd, "stream") {
			// 增量内容实时写 stderr,完整结果仍输出 stdout JSON。
			result, err = svc.GenerateDDLStream(requirements, func(delta string) {
				fmt.Fprint(os.Stderr, delta)
			})
		} else {
			result, err = svc.GenerateDDL(requirements)
		}
		if err != nil {
			return err
		}
		emit(result)
		return nil
	},
}

var aiPartitionCmd = &cobra.Command{
	Use:   "partition",
	Short: "根据 DDL 建议微服务划分",
	RunE: func(cmd *cobra.Command, args []string) error {
		ddlFile := flagString(cmd, "ddl", "")
		if ddlFile == "" {
			checkErr(fmt.Errorf("必须指定 --ddl（DDL 文件）"))
		}
		ddl, err := readFileContent(ddlFile)
		if err != nil {
			checkErr(err)
		}

		partitions, err := buildAIService(cmd).PartitionMicroservices(ddl)
		if err != nil {
			return err
		}
		emit(partitions)
		return nil
	},
}

var aiReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "AI 代码审查（Go/微服务/Kratos 维度）",
	RunE: func(cmd *cobra.Command, args []string) error {
		files := stringSliceFlag(cmd, "files")
		if len(files) == 0 {
			checkErr(fmt.Errorf("必须指定 --files（逗号分隔的文件路径）"))
		}

		contents := map[string]string{}
		for _, path := range files {
			content, err := readFileContent(path)
			if err != nil {
				checkErr(err)
			}
			contents[shortPath(path)] = content
		}

		svc := buildAIService(cmd)
		var result *ai.StepResult
		var err error
		if boolFlag(cmd, "stream") {
			result, err = svc.ReviewCodeStream(contents, func(delta string) {
				fmt.Fprint(os.Stderr, delta)
			})
		} else {
			result, err = svc.ReviewCode(contents)
		}
		if err != nil {
			return err
		}
		emit(result)
		return nil
	},
}

func shortPath(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) > 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return path
}

// maskKey 遮蔽密钥,仅保留末 4 位。
func maskKey(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

// applyAIFlagsToConfig 仅将命令行显式提供的旗标覆盖到 cfg(不回退环境变量)。
func applyAIFlagsToConfig(cmd *cobra.Command, cfg *ai.Config) {
	set := func(name string, dst *string) {
		if cmd.Flags().Changed(name) {
			if v, err := cmd.Flags().GetString(name); err == nil {
				*dst = v
			}
		}
	}
	set("provider", &cfg.Provider)
	set("base-url", &cfg.BaseURL)
	set("api-key", &cfg.APIKey)
	set("azure-api-version", &cfg.AzureAPIVersion)
	set("model", &cfg.Model)
	if cmd.Flags().Changed("temperature") {
		if t, err := cmd.Flags().GetFloat64("temperature"); err == nil && t >= 0 && t <= 2 {
			cfg.Temperature = t
		}
	}
	if cmd.Flags().Changed("max-tokens") {
		if m, err := cmd.Flags().GetInt("max-tokens"); err == nil && m > 0 {
			cfg.MaxTokens = m
		}
	}
}

var aiConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "查看 / 持久化 AI 配置(与 GUI 设置同源)",
}

var aiConfigShowCmd = &cobra.Command{
	Use:   "show",
	Short: "查看已持久化的 AI 配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := ai.LoadConfig()
		if !boolFlag(cmd, "show-secrets") && cfg.APIKey != "" {
			cfg.APIKey = maskKey(cfg.APIKey)
		}
		emit(cfg)
		return nil
	},
}

var aiConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "写入 AI 配置到持久化文件(仅覆盖显式提供的旗标)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := ai.LoadConfig()
		applyAIFlagsToConfig(cmd, cfg)
		if err := ai.SaveConfig(cfg); err != nil {
			return err
		}
		out := *cfg
		if !boolFlag(cmd, "show-secrets") && out.APIKey != "" {
			out.APIKey = maskKey(out.APIKey)
		}
		emit(out)
		return nil
	},
}

var aiFindOpenapiCmd = &cobra.Command{
	Use:   "find-openapi",
	Short: "在项目内查找 OpenAPI 规范文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := flagString(cmd, "path", ".")
		files, err := ai.FindOpenAPIFiles(root)
		if err != nil {
			return err
		}
		msg := ""
		if len(files) == 0 {
			msg = "未找到 OpenAPI 文件"
		}
		emit(ai.OpenAPIResult{Success: true, Files: files, Message: msg})
		return nil
	},
}

// parsePartitions 兼容两种输入:划分结果对象 {success,partitions:[...]} 与裸数组 [...]。
func parsePartitions(raw string) ([]ai.MicroservicePartition, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "[") {
		var arr []ai.MicroservicePartition
		if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
			return nil, fmt.Errorf("--partitions 不是合法的划分数组: %w", err)
		}
		return arr, nil
	}
	var res ai.PartitionResult
	if err := json.Unmarshal([]byte(trimmed), &res); err != nil {
		return nil, fmt.Errorf("--partitions 不是合法的划分结果 JSON: %w", err)
	}
	return res.Partitions, nil
}

var aiBackendCmd = &cobra.Command{
	Use:   "backend",
	Short: "按 AI 微服务划分结果,从 DDL 生成 gRPC 后端代码(等价 GUI 的 AI 后端生成)",
	Long: `读取 DDL 文件与微服务划分 JSON(--partitions,支持 ai partition 的输出对象或其 partitions 数组),
按 {serviceName, tables} 映射生成 gRPC 全栈代码,与 GUI 的 AIGenerateBackendCode 同源。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ddlFile := flagString(cmd, "ddl", "")
		partFile := flagString(cmd, "partitions", "")
		if ddlFile == "" || partFile == "" {
			checkErr(fmt.Errorf("必须指定 --ddl 与 --partitions"))
		}
		ddl, err := readFileContent(ddlFile)
		if err != nil {
			return err
		}
		partitions, err := parsePartitions(readJSONArg(partFile, "--partitions"))
		if err != nil {
			checkErr(err)
		}
		if len(partitions) == 0 {
			checkErr(fmt.Errorf("划分结果为空,无可生成的服务"))
		}

		var opts generator.GeneratorOptions
		var id uint32
		for _, p := range partitions {
			for _, tableName := range p.Tables {
				id++
				opts = append(opts, &generator.Option{ID: id, TableName: tableName, Service: p.ServiceName})
			}
		}

		dbConfig := database.DBConfig{Type: database.DbTypeMySQL, SQLContent: ddl}
		rootPath, projectName := resolveProjectRoot(cmd)
		ormType := flagString(cmd, "orm", "ent")
		servers := stringSliceFlag(cmd, "servers")

		g := generator.NewGenerator()
		g.SetLogger(cliLogger{})
		g.SetOptions(opts)
		g.SetSkipPostProcess(boolFlag(cmd, "skip-postprocess"))

		logf("项目根目录: %s (module: %s)", rootPath, projectName)
		logf("开始按 %d 个划分服务生成 gRPC 代码 (orm=%s)...", len(partitions), ormType)

		if err := g.GenerateGrpcCode(context.Background(), dbConfig, ormType, "per-table", rootPath, projectName, servers); err != nil {
			return err
		}

		emit(map[string]any{
			"success":     true,
			"root":        rootPath,
			"orm":         ormType,
			"postprocess": !boolFlag(cmd, "skip-postprocess"),
			"services":    serviceNames(opts),
		})
		return nil
	},
}

func init() {
	addAIFlags(aiTestCmd)
	addAIFlags(aiDdlCmd)
	addAIFlags(aiPartitionCmd)
	addAIFlags(aiReviewCmd)
	aiDdlCmd.Flags().String("requirements", "", "需求文档文件路径（必填）")
	aiDdlCmd.Flags().Bool("stream", false, "流式生成（增量内容实时输出到 stderr，完整结果仍输出 stdout JSON）")
	aiPartitionCmd.Flags().String("ddl", "", "DDL 文件路径（必填）")
	aiReviewCmd.Flags().StringSlice("files", nil, "要审查的文件路径（逗号分隔，必填）")
	aiReviewCmd.Flags().Bool("stream", false, "流式审查（增量内容实时输出到 stderr，完整结果仍输出 stdout JSON）")

	addAIFlags(aiConfigSetCmd)
	aiConfigShowCmd.Flags().Bool("show-secrets", false, "显示完整 API Key(默认末 4 位脱敏)")
	aiConfigSetCmd.Flags().Bool("show-secrets", false, "输出中显示完整 API Key(默认脱敏)")

	aiFindOpenapiCmd.Flags().String("path", ".", "项目根目录（默认当前目录）")

	addBackendFlags(aiBackendCmd)
	aiBackendCmd.Flags().String("partitions", "", "微服务划分 JSON 文件（ai partition 输出,支持 @file,必填）")

	aiConfigCmd.AddCommand(aiConfigShowCmd, aiConfigSetCmd)
	aiCmd.AddCommand(aiPresetsCmd, aiTestCmd, aiDdlCmd, aiPartitionCmd, aiReviewCmd, aiConfigCmd, aiFindOpenapiCmd, aiBackendCmd)
}
