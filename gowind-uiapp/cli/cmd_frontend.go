package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/frontendgen"
)

var frontendCmd = &cobra.Command{
	Use:   "frontend",
	Short: "前端 CRUD 代码生成（vue-element / vue-vben / react）",
}

var frontendGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "从 OpenAPI 规范生成前端 CRUD 代码",
	Long: `从 OpenAPI 3.0 规范生成前端 CRUD 代码: composable/hooks、列表页、编辑抽屉、路由、国际化。
生成路径相对前端项目 src 目录（如 api/composables、views/app、router/routes、locales）。

vue-vben 的国际化产物是合并式片段: 目标 locales/langs/{lang}/page.json（menu.json）存在时按键合并写回，
不存在时新建独立片段文件。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		openapiRef := flagString(cmd, "openapi", "")
		if openapiRef == "" {
			checkErr(fmt.Errorf("必须指定 --openapi（OpenAPI YAML 文件路径或 http(s) URL）"))
		}

		frameworkName := flagString(cmd, "framework", "")
		framework, ok := frontendgen.ParseFramework(frameworkName)
		if !ok {
			checkErr(fmt.Errorf("无效的 --framework %q（可选 vue-element / vue-vben / react）", frameworkName))
		}

		spec, err := loadOpenAPISpec(openapiRef)
		if err != nil {
			return err
		}

		opts := frontendgen.Options{
			Spec:              spec,
			Framework:         framework,
			ServiceName:       flagString(cmd, "service-name", "admin"),
			ModulePathMap:     parseModulePathMap(cmd),
			GenerateTypes:     stringSliceFlag(cmd, "types"),
			Tags:              stringSliceFlag(cmd, "tags"),
			RouterModules:     parseRouterModules(cmd),
			AutoRouterModules: true,
		}

		files, err := frontendgen.Generate(opts)
		if err != nil {
			return err
		}

		switch {
		case boolFlag(cmd, "stdout"):
			emit(map[string]any{"files": files})
		case boolFlag(cmd, "dry-run"):
			manifest := make([]map[string]any, 0, len(files))
			for _, f := range files {
				manifest = append(manifest, map[string]any{
					"path":        f.Path,
					"type":        f.Type,
					"description": f.Description,
					"serviceName": f.ServiceName,
					"bytes":       len(f.Content),
				})
			}
			emit(map[string]any{"files": manifest, "count": len(manifest)})
		default:
			outDir := flagString(cmd, "out", "")
			if outDir == "" {
				checkErr(fmt.Errorf("必须指定 --out（前端项目 src 目录），或使用 --dry-run / --stdout"))
			}
			results, err := frontendgen.WriteFiles(files, outDir)
			if err != nil {
				return err
			}
			emit(map[string]any{"results": results, "count": len(results)})
		}
		return nil
	},
}

var frontendParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "解析 OpenAPI 规范,列出可生成的服务(tag)——等价 GUI 的服务预览",
	RunE: func(cmd *cobra.Command, args []string) error {
		openapiRef := flagString(cmd, "openapi", "")
		if openapiRef == "" {
			checkErr(fmt.Errorf("必须指定 --openapi（OpenAPI YAML 文件路径或 http(s) URL）"))
		}
		spec, err := loadOpenAPISpec(openapiRef)
		if err != nil {
			return err
		}
		emit(frontendgen.ExtractServices(spec))
		return nil
	},
}

// loadOpenAPISpec 加载 OpenAPI 规格（本地文件或 URL）
func loadOpenAPISpec(ref string) (*frontendgen.Spec, error) {
	var data []byte

	if strings.HasPrefix(strings.ToLower(ref), "http://") || strings.HasPrefix(strings.ToLower(ref), "https://") {
		resp, err := http.Get(ref)
		if err != nil {
			return nil, fmt.Errorf("拉取 OpenAPI 失败: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("拉取 OpenAPI 失败: HTTP %d", resp.StatusCode)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 响应失败: %w", err)
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

// parseModulePathMap 解析 --module-path-map（JSON 字符串或 @文件）
func parseModulePathMap(cmd *cobra.Command) map[string]string {
	raw := flagString(cmd, "module-path-map", "")
	if raw == "" {
		return nil
	}
	raw = readJSONArg(raw, "--module-path-map")

	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		checkErr(fmt.Errorf("--module-path-map 不是合法的 JSON 对象: %w", err))
	}
	return m
}

// parseRouterModules 解析 --router-modules（@文件，JSON 数组；缺省 auto）
func parseRouterModules(cmd *cobra.Command) []frontendgen.RouterModuleConfig {
	raw := flagString(cmd, "router-modules", "")
	if raw == "" || raw == "auto" {
		return nil
	}
	raw = readJSONArg(raw, "--router-modules")

	var modules []frontendgen.RouterModuleConfig
	if err := json.Unmarshal([]byte(raw), &modules); err != nil {
		checkErr(fmt.Errorf("--router-modules 不是合法的 JSON 数组: %w", err))
	}
	return modules
}

// readJSONArg 支持 "@file" 形式读取文件内容
func readJSONArg(raw, flagName string) string {
	if strings.HasPrefix(raw, "@") {
		content, err := readFileContent(strings.TrimPrefix(raw, "@"))
		if err != nil {
			checkErr(err)
		}
		return content
	}
	return raw
}

func init() {
	frontendGenCmd.Flags().String("openapi", "", "OpenAPI 3.0 YAML 文件路径或 http(s) URL（必填）")
	frontendGenCmd.Flags().String("framework", "", "目标框架: vue-element | vue-vben | react（必填）")
	frontendGenCmd.Flags().StringSlice("tags", nil, "要生成的服务 tag 名（缺省=全部服务）")
	frontendGenCmd.Flags().StringSlice("types", nil, "文件类型: composable,page,drawer,router,locale（缺省=全部；react 的 composable 自动映射为 hooks）")
	frontendGenCmd.Flags().String("service-name", "admin", "生成代码的服务名（generated 导入路径前缀）")
	frontendGenCmd.Flags().String("module-path-map", "", "文件名->模块路径 JSON（如 {\"role\":\"permission/role\"}），支持 @file")
	frontendGenCmd.Flags().String("router-modules", "auto", "路由模块分组 JSON 数组（@file），缺省 auto 按 basePath 自动检测")
	frontendGenCmd.Flags().String("out", "", "输出目录（前端项目 src 目录）")
	frontendGenCmd.Flags().Bool("dry-run", false, "只输出文件清单，不写盘")
	frontendGenCmd.Flags().Bool("stdout", false, "不写盘，输出全部文件内容 JSON")

	frontendParseCmd.Flags().String("openapi", "", "OpenAPI 3.0 YAML 文件路径或 http(s) URL（必填）")

	frontendCmd.AddCommand(frontendGenCmd, frontendParseCmd)
}
