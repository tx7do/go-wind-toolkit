// Package frontend 实现 `gow frontend`：从 OpenAPI 3.0 规范生成前端 CRUD 代码。
// 生成引擎在 pkg/frontendgen（与桌面端共用同一实现）。
package frontend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/frontendgen"
)

var (
	frontendOpenAPI      string
	frontendFramework    string
	frontendServiceName  string
	frontendTags         []string
	frontendTypes        []string
	frontendModulePath   string
	frontendRouterModule string
	frontendOut          string
	frontendDryRun       bool
	frontendStdout       bool
)

// CmdFrontend `gow frontend` 命令组。
var CmdFrontend = &cobra.Command{
	Use:   "frontend",
	Short: "前端 CRUD 代码生成（vue-element / vue-vben / react）",
	Long: `从 OpenAPI 3.0 规范生成前端管理页代码。

子命令:
  gen    生成 composable/hooks、列表页、编辑抽屉、路由、国际化
  parse  列出规范中可生成的服务（tag）

生成路径相对前端项目 src 目录（api/composables、views/app、router/routes、locales）。`,
	SilenceUsage: true,
}

var frontendGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "从 OpenAPI 规范生成前端 CRUD 代码",
	Long: `从 OpenAPI 3.0 规范生成前端 CRUD 代码: composable/hooks、列表页、编辑抽屉、路由、国际化。
生成路径相对前端项目 src 目录（如 api/composables、views/app、router/routes、locales）。

vue-vben 的国际化产物是合并式片段: 目标 locales/langs/{lang}/page.json（menu.json）存在时按键合并写回，
不存在时新建独立片段文件。

示例:
  gow frontend gen --openapi ./api/openapi.yaml --framework vue-vben --out ./admin-web/src
  gow frontend gen --openapi http://localhost:8000/q/openapi.yaml --framework react --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := buildOptions()
		if err != nil {
			return err
		}

		files, err := frontendgen.Generate(opts)
		if err != nil {
			return err
		}

		switch {
		case frontendStdout:
			return emitJSON(map[string]any{"files": files})
		case frontendDryRun:
			fmt.Fprintf(cmd.OutOrStdout(), "\033[36m[DRY-RUN] preview only — nothing was written. %d files would be generated:\033[m\n", len(files))
			for _, f := range files {
				fmt.Fprintf(cmd.OutOrStdout(), "  \033[36m+\033[m %s (%s, %d bytes)\n", f.Path, f.Type, len(f.Content))
			}
			return nil
		default:
			if frontendOut == "" {
				return fmt.Errorf("必须指定 --out（前端项目 src 目录），或使用 --dry-run / --stdout")
			}
			results, err := frontendgen.WriteFiles(files, frontendOut)
			if err != nil {
				return err
			}
			created, overwritten, merged := 0, 0, 0
			for _, r := range results {
				switch r.Action {
				case frontendgen.ActionCreated:
					created++
					fmt.Fprintf(cmd.OutOrStdout(), "  \033[32mcreated\033[m     %s\n", r.Path)
				case frontendgen.ActionOverwrite:
					overwritten++
					fmt.Fprintf(cmd.OutOrStdout(), "  \033[33moverwritten\033[m %s\n", r.Path)
				default:
					merged++
					fmt.Fprintf(cmd.OutOrStdout(), "  \033[36m%-11s\033[m %s\n", r.Action, r.Path)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\033[32mSUCCESS: generated %d files (%d created, %d overwritten, %d merged) -> %s\033[m\n",
				len(results), created, overwritten, merged, frontendOut)
			return nil
		}
	},
}

var frontendParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "解析 OpenAPI 规范，列出可生成的服务（tag）",
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := loadOpenAPISpec(frontendOpenAPI)
		if err != nil {
			return err
		}
		services := frontendgen.ExtractServices(spec)
		if len(services) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no services found in the OpenAPI spec")
			return nil
		}
		for _, svc := range services {
			fmt.Fprintf(cmd.OutOrStdout(), "  \033[32m%s\033[m  ops=%d", svc.TagName, len(svc.Operations))
			if svc.TypePrefix != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  prefix=%s", svc.TypePrefix)
			}
			if svc.BasePath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  basePath=%s", svc.BasePath)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\033[32mSUCCESS: found %d services\033[m\n", len(services))
		return nil
	},
}

// buildOptions 校验公共参数并组装 frontendgen.Options。
func buildOptions() (frontendgen.Options, error) {
	if frontendOpenAPI == "" {
		return frontendgen.Options{}, fmt.Errorf("必须指定 --openapi（OpenAPI YAML 文件路径或 http(s) URL）")
	}
	framework, ok := frontendgen.ParseFramework(frontendFramework)
	if !ok {
		return frontendgen.Options{}, fmt.Errorf("无效的 --framework %q（可选 vue-element / vue-vben / react）", frontendFramework)
	}
	spec, err := loadOpenAPISpec(frontendOpenAPI)
	if err != nil {
		return frontendgen.Options{}, err
	}
	modulePathMap, err := parseModulePathMap(frontendModulePath)
	if err != nil {
		return frontendgen.Options{}, err
	}
	routerModules, err := parseRouterModules(frontendRouterModule)
	if err != nil {
		return frontendgen.Options{}, err
	}
	opts := frontendgen.Options{
		Spec:              spec,
		Framework:         framework,
		ServiceName:       frontendServiceName,
		ModulePathMap:     modulePathMap,
		GenerateTypes:     frontendTypes,
		Tags:              frontendTags,
		RouterModules:     routerModules,
		AutoRouterModules: true,
	}
	return opts, nil
}

// openAPISpecClient 带超时的拉取客户端:http.Get 用的是无超时的 DefaultClient,
// 一个只接受连接不回数据的地址能把 CLI 永久挂住。
var openAPISpecClient = &http.Client{Timeout: 30 * time.Second}

// maxOpenAPISpecBytes 响应正文上限:OpenAPI 文档正常在几十到几百 KB,
// 不设上限就等于让远端决定 CLI 的内存占用。
const maxOpenAPISpecBytes = 10 << 20

// loadOpenAPISpec 加载 OpenAPI 规格（本地文件或 http(s) URL）。
func loadOpenAPISpec(ref string) (*frontendgen.Spec, error) {
	if ref == "" {
		return nil, fmt.Errorf("必须指定 --openapi（OpenAPI YAML 文件路径或 http(s) URL）")
	}

	var data []byte
	if strings.HasPrefix(strings.ToLower(ref), "http://") || strings.HasPrefix(strings.ToLower(ref), "https://") {
		resp, err := openAPISpecClient.Get(ref)
		if err != nil {
			return nil, fmt.Errorf("拉取 OpenAPI 失败: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("拉取 OpenAPI 失败: HTTP %d", resp.StatusCode)
		}
		// 多读 1 字节用来分辨"正好等于上限"和"被截断"。
		data, err = io.ReadAll(io.LimitReader(resp.Body, maxOpenAPISpecBytes+1))
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 响应失败: %w", err)
		}
		if len(data) > maxOpenAPISpecBytes {
			return nil, fmt.Errorf("OpenAPI 文档超过 %d MiB 上限,拒绝读入内存", maxOpenAPISpecBytes>>20)
		}
	} else {
		var err error
		data, err = os.ReadFile(ref)
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 文件失败: %w", err)
		}
	}

	return frontendgen.ParseOpenAPIYAML(data)
}

// parseModulePathMap 解析 --module-path-map（JSON 字符串或 @文件）。
func parseModulePathMap(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, nil
	}
	content, err := readJSONArg(raw, "--module-path-map")
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return nil, fmt.Errorf("--module-path-map 不是合法的 JSON 对象: %w", err)
	}
	return m, nil
}

// parseRouterModules 解析 --router-modules（JSON 数组；"auto"/空 → 自动检测）。
func parseRouterModules(raw string) ([]frontendgen.RouterModuleConfig, error) {
	if raw == "" || raw == "auto" {
		return nil, nil
	}
	content, err := readJSONArg(raw, "--router-modules")
	if err != nil {
		return nil, err
	}
	var modules []frontendgen.RouterModuleConfig
	if err := json.Unmarshal([]byte(content), &modules); err != nil {
		return nil, fmt.Errorf("--router-modules 不是合法的 JSON 数组: %w", err)
	}
	return modules, nil
}

// readJSONArg 支持 "@file" 形式读取文件内容。
func readJSONArg(raw, flagName string) (string, error) {
	if !strings.HasPrefix(raw, "@") {
		return raw, nil
	}
	content, err := os.ReadFile(strings.TrimPrefix(raw, "@"))
	if err != nil {
		return "", fmt.Errorf("读取 %s 参数文件失败: %w", flagName, err)
	}
	return string(content), nil
}

// emitJSON 向 stdout 输出缩进 JSON（--stdout 模式，供管道消费）。
func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	frontendGenCmd.Flags().StringVar(&frontendOpenAPI, "openapi", "", "OpenAPI 3.0 YAML 文件路径或 http(s) URL（必填）")
	frontendGenCmd.Flags().StringVar(&frontendFramework, "framework", "", "目标框架: vue-element | vue-vben | react（必填）")
	frontendGenCmd.Flags().StringSliceVar(&frontendTags, "tags", nil, "要生成的服务 tag 名（缺省=全部服务）")
	frontendGenCmd.Flags().StringSliceVar(&frontendTypes, "types", nil, "文件类型: composable,page,drawer,router,locale（缺省=全部；react 的 composable 自动映射为 hooks）")
	frontendGenCmd.Flags().StringVar(&frontendServiceName, "service-name", "admin", "生成代码的服务名（generated 导入路径前缀）")
	frontendGenCmd.Flags().StringVar(&frontendModulePath, "module-path-map", "", "文件名->模块路径 JSON（如 {\"role\":\"permission/role\"}），支持 @file")
	frontendGenCmd.Flags().StringVar(&frontendRouterModule, "router-modules", "auto", "路由模块分组 JSON 数组（@file），缺省 auto 按 basePath 自动检测")
	frontendGenCmd.Flags().StringVar(&frontendOut, "out", "", "输出目录（前端项目 src 目录）")
	frontendGenCmd.Flags().BoolVar(&frontendDryRun, "dry-run", false, "只输出文件清单，不写盘")
	frontendGenCmd.Flags().BoolVar(&frontendStdout, "stdout", false, "不写盘，输出全部文件内容 JSON")

	frontendParseCmd.Flags().StringVar(&frontendOpenAPI, "openapi", "", "OpenAPI 3.0 YAML 文件路径或 http(s) URL（必填）")

	CmdFrontend.AddCommand(frontendGenCmd, frontendParseCmd)
}
