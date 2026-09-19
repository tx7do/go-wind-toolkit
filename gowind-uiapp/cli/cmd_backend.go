package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/generator"
)

var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "后端 Kratos 微服务代码生成（proto + ORM + service + server + wire + config）",
}

// tableMapping 表 -> 服务映射条目（--mapping JSON 文件的元素）
type tableMapping struct {
	Table        string `json:"table"`
	Service      string `json:"service"`
	ProtoPackage string `json:"protoPackage,omitempty"`
	Exclude      bool   `json:"exclude,omitempty"`
}

func addBackendFlags(cmd *cobra.Command) {
	cmd.Flags().String("ddl", "", "DDL 文件路径（CREATE TABLE 语句，与 --dsn 二选一）")
	cmd.Flags().String("dsn", "", "数据库连接串（可用环境变量 GOWIND_DSN）")
	cmd.Flags().String("driver", "mysql", "数据库类型: mysql | postgresql | sqlite")
	cmd.Flags().String("mapping", "", "表映射 JSON 文件: [{\"table\":\"user\",\"service\":\"identity\"}]")
	cmd.Flags().StringSlice("tables", nil, "表映射简写: --tables user:identity,role:permission")
	cmd.Flags().String("orm", "ent", "ORM 类型: ent | gorm")
	cmd.Flags().StringSlice("servers", []string{"grpc"}, "生成的传输层，逗号分隔: grpc,rest,websocket（对应 gow generate -s）")
	cmd.Flags().String("strategy", "per-table", "proto 包策略: per-table | by-service | custom")
	cmd.Flags().String("out", ".", "项目根目录（生成到 app/<服务名>/service，默认当前目录）")
	cmd.Flags().Bool("skip-postprocess", false, "跳过生成后的 tidy/buf/ent/wire 后处理链")
	cmd.Flags().String("service-name", "", "REST 网关服务名（rest 命令必填）")
}

// resolveMappings 解析表映射（--mapping 文件优先，其次 --tables 简写）
func resolveMappings(cmd *cobra.Command) []tableMapping {
	var mappings []tableMapping

	if mappingFile := flagString(cmd, "mapping", ""); mappingFile != "" {
		content, err := readFileContent(mappingFile)
		if err != nil {
			checkErr(err)
		}
		if err := json.Unmarshal([]byte(content), &mappings); err != nil {
			checkErr(fmt.Errorf("解析映射文件 %s 失败: %w", mappingFile, err))
		}
	} else if inline := stringSliceFlag(cmd, "tables"); len(inline) > 0 {
		for _, item := range inline {
			parts := strings.SplitN(item, ":", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				checkErr(fmt.Errorf("--tables 条目 %q 格式错误，应为 表名:服务名", item))
			}
			mappings = append(mappings, tableMapping{Table: parts[0], Service: parts[1]})
		}
	}

	if len(mappings) == 0 {
		checkErr(fmt.Errorf("必须通过 --mapping 文件或 --tables 简写指定表到服务的映射"))
	}
	return mappings
}

// buildBackendInputs 组装数据库配置与生成器选项
func buildBackendInputs(cmd *cobra.Command) (database.DBConfig, generator.GeneratorOptions) {
	mappings := resolveMappings(cmd)

	var dbConfig database.DBConfig
	if ddlFile := flagString(cmd, "ddl", ""); ddlFile != "" {
		content, err := readFileContent(ddlFile)
		if err != nil {
			checkErr(err)
		}
		dbConfig = database.DBConfig{Type: database.DbTypeMySQL, SQLContent: content}
	} else {
		dsn := flagString(cmd, "dsn", "")
		if dsn == "" {
			dsn = envOr("GOWIND_DSN", "")
		}
		if dsn == "" {
			checkErr(fmt.Errorf("必须提供 --ddl 文件或 --dsn（或环境变量 GOWIND_DSN）"))
		}
		dbConfig = database.DBConfig{
			Type:   database.DbType(flagString(cmd, "driver", "mysql")),
			UseDSN: true,
			DSN:    dsn,
		}
	}

	var opts generator.GeneratorOptions
	for i, m := range mappings {
		opts = append(opts, &generator.Option{
			ID:           uint32(i + 1),
			TableName:    m.Table,
			Service:      m.Service,
			Exclude:      m.Exclude,
			ProtoPackage: m.ProtoPackage,
		})
	}
	return dbConfig, opts
}

// resolveProjectRoot 探测项目根目录与模块名
func resolveProjectRoot(cmd *cobra.Command) (rootPath, projectName string) {
	rootPath = flagString(cmd, "out", ".")
	absRoot, err := filepath.Abs(rootPath)
	if err == nil {
		rootPath = absRoot
	}

	if info, err := detect.NewProjectDetector().Detect(rootPath); err == nil && info.ModPath != "" {
		return info.Root, info.ModPath
	}

	// 非 Go 模块目录：从目录名推导项目名
	projectName = filepath.Base(rootPath)
	return rootPath, projectName
}

var backendGrpcCmd = &cobra.Command{
	Use:   "grpc",
	Short: "生成 gRPC 全栈微服务代码（含 go mod tidy / buf / ent / wire 后处理）",
	Long: `按表到服务的映射生成 gRPC 微服务全栈代码: proto、ORM、data、service、server、main、config、Makefile。
生成后自动执行后处理链: go mod tidy -> buf generate -> ent generate (ent ORM 时) -> wire。
与 GUI 的 gRPC 代码生成完全同源。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbConfig, opts := buildBackendInputs(cmd)
		rootPath, projectName := resolveProjectRoot(cmd)

		strategy := flagString(cmd, "strategy", "per-table")
		ormType := flagString(cmd, "orm", "ent")
		servers := stringSliceFlag(cmd, "servers")

		g := generator.NewGenerator()
		g.SetLogger(cliLogger{})
		g.SetOptions(opts)
		g.SetSkipPostProcess(boolFlag(cmd, "skip-postprocess"))

		logf("项目根目录: %s (module: %s)", rootPath, projectName)
		logf("开始生成 gRPC 代码 (orm=%s, strategy=%s)...", ormType, strategy)

		if err := g.GenerateGrpcCode(context.Background(), dbConfig, ormType, strategy, rootPath, projectName, servers); err != nil {
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

var backendRestCmd = &cobra.Command{
	Use:   "rest",
	Short: "生成 REST 网关服务代码（不生成 ORM/data/repo，无后处理）",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbConfig, opts := buildBackendInputs(cmd)
		rootPath, projectName := resolveProjectRoot(cmd)

		restServiceName := flagString(cmd, "service-name", "")
		if restServiceName == "" {
			checkErr(fmt.Errorf("必须指定 --service-name（REST 网关服务名，如 admin-portal）"))
		}
		strategy := flagString(cmd, "strategy", "per-table")

		g := generator.NewGenerator()
		g.SetLogger(cliLogger{})
		g.SetOptions(opts)

		logf("项目根目录: %s (module: %s)", rootPath, projectName)
		logf("开始生成 REST 服务 %s 代码...", restServiceName)

		if err := g.GenerateRestCode(context.Background(), restServiceName, "", strategy, dbConfig, rootPath, projectName); err != nil {
			return err
		}

		emit(map[string]any{
			"success":     true,
			"root":        rootPath,
			"serviceName": restServiceName,
			"services":    serviceNames(opts),
		})
		return nil
	},
}

func serviceNames(opts generator.GeneratorOptions) []string {
	seen := map[string]bool{}
	var names []string
	for _, opt := range opts {
		if opt.Exclude || opt.Service == "" || seen[opt.Service] {
			continue
		}
		seen[opt.Service] = true
		names = append(names, opt.Service)
	}
	return names
}

// cliLogger 生成过程日志 -> stderr
type cliLogger struct{}

func (cliLogger) Infof(format string, args ...any)  { logf("[gen] "+format, args...) }
func (cliLogger) Errorf(format string, args ...any) { logf("[gen][ERROR] "+format, args...) }

func init() {
	addBackendFlags(backendGrpcCmd)
	addBackendFlags(backendRestCmd)
	backendCmd.AddCommand(backendGrpcCmd, backendRestCmd)
}
