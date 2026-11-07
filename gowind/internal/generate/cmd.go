package generate

import (	"path/filepath"
	"bytes"

	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	sqlkratos "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlkratos"
)

// CmdGenerate represents the generate command
var CmdGenerate = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "generate CRUD code from database schema",
	Long:    "Generate complete Kratos microservice code (proto, ORM, service, server, wiring, config) from an existing database, SQL file, or Go schema source (ent schema dir via ent://, gorm model dir via gorm://). Module registration follows the target service form: anchor injection into hand-written wiring.go, or wire provider sets for legacy services. Example: gow generate",
	RunE:         Run,
	SilenceUsage: true,
}

var (
	genDSN           string
	genDriver        string
	genServiceName   string
	genOrmType       string
	genServers       []string
	genTables        []string
	genExcludeTables []string
	genModuleVersion string
	genProtoOnly     bool
	genSkipORM       bool
	genSkipConfig    bool
	genTests       bool
	genCheck       bool
	genSkipMakefile  bool
	genSourceModule  string
	genDryRun        bool
)

func init() {
	CmdGenerate.Flags().StringVarP(&genDSN, "dsn", "", "", "Database source name (DSN), e.g. mysql://user:pass@tcp(localhost:3306)/dbname; Go schema sources: ent://<schema dir>, gorm://<model dir>")
	CmdGenerate.Flags().StringVarP(&genDriver, "driver", "", "mysql", "Database driver: mysql, postgres")
	CmdGenerate.Flags().StringVarP(&genServiceName, "service", "", "", "Service name (module name)")
	CmdGenerate.Flags().StringVarP(&genOrmType, "orm", "", "ent", "ORM type: ent, gorm")
	CmdGenerate.Flags().StringArrayVarP(&genServers, "servers", "s", []string{"grpc"}, "Server types: grpc, rest, websocket")
	CmdGenerate.Flags().StringArrayVarP(&genTables, "tables", "t", nil, "Tables to include (default: all tables)")
	CmdGenerate.Flags().StringArrayVarP(&genExcludeTables, "exclude-tables", "", nil, "Tables to exclude")
	CmdGenerate.Flags().StringVarP(&genModuleVersion, "module-version", "", "v1", "API module version")
	CmdGenerate.Flags().BoolVarP(&genProtoOnly, "proto-only", "", false, "Only generate proto files")
	CmdGenerate.Flags().BoolVarP(&genSkipORM, "skip-orm", "", false, "Skip ORM code generation")
	CmdGenerate.Flags().BoolVarP(&genSkipConfig, "skip-config", "", false, "Skip config file generation")
	CmdGenerate.Flags().BoolVarP(&genTests, "tests", "", false, "Generate unit test scaffolds (fake repo) for CRUD services")
	CmdGenerate.Flags().BoolVarP(&genCheck, "check", "", false, "Drift guard: generate into a temp dir and diff against the target, exit non-zero on drift")
	CmdGenerate.Flags().BoolVarP(&genSkipMakefile, "skip-makefile", "", false, "Skip Makefile generation")
	CmdGenerate.Flags().StringVarP(&genSourceModule, "source-module", "", "", "Source module name for REST service")
	CmdGenerate.Flags().BoolVarP(&genDryRun, "dry-run", "n", false, "Validate the data source, resolve tables and preview the plan without writing anything")
}

func Run(cmd *cobra.Command, args []string) error {
	// 交互式获取缺失参数
	if genDSN == "" {
		prompt := &survey.Input{
			Message: "Database DSN?",
			Help:    "Database connection string (e.g. mysql://user:pass@tcp(localhost:3306)/dbname), or a Go schema source: ent://<schema dir> / gorm://<model dir>",
		}
		if err := survey.AskOne(prompt, &genDSN); err != nil || genDSN == "" {
			return nil
		}
	}

	if genServiceName == "" {
		prompt := &survey.Input{
			Message: "Service name?",
			Help:    "The service/module name to generate code for.",
		}
		if err := survey.AskOne(prompt, &genServiceName); err != nil || genServiceName == "" {
			return nil
		}
	}

	// 获取项目信息
	inspector, err := pkg.NewModuleInspectorFromGo(cmd.Context(), "")
	if err != nil {
		return err
	}

	projectName := extractProjectName(inspector.ModPath)
	outputPath := inspector.Root

	sourceModule := genSourceModule
	if sourceModule == "" {
		sourceModule = genServiceName
	}

	// 构建 DSN 前缀（如果用户没有提供 scheme）
	dsn := genDSN

	// 裸目录路径多半是想用 Go 源码 schema 源,给出明确指引。
	if !strings.Contains(dsn, "://") {
		if fi, statErr := os.Stat(dsn); statErr == nil && fi.IsDir() {
			return fmt.Errorf("source %q is a directory; use ent://<dir> (ent schema dir) or gorm://<dir> (gorm model dir)", dsn)
		}
	}

	// Go 源码 schema 源与 ORM 选择必须一致(dry-run 也要能发现)。
	switch {
	case strings.HasPrefix(dsn, "ent://") && genOrmType != "ent":
		return fmt.Errorf("source ent:// requires --orm ent (got %q)", genOrmType)
	case strings.HasPrefix(dsn, "gorm://") && genOrmType != "gorm":
		return fmt.Errorf("source gorm:// requires --orm gorm (got %q)", genOrmType)
	}

	// 逗号分隔写法(-s grpc,rest)与重复 flag 写法(-s grpc -s rest)等价。
	genServers = pkg.SplitFlagList(genServers)

	opts := sqlkratos.GeneratorOptions{
		Driver:           genDriver,
		Source:           dsn,
		IncludedTables:   genTables,
		ExcludedTables:   genExcludeTables,
		OutputPath:       outputPath,
		SourceModuleName: sourceModule,
		ModuleName:       genServiceName,
		ModuleVersion:    genModuleVersion,
		OrmType:          genOrmType,
		ProjectName:      projectName,
		ServiceName:      genServiceName,
		Servers:          genServers,
		UseRepo:          true,
		GenerateProto:    true,
		GenerateServer:   !genProtoOnly,
		GenerateService:  !genProtoOnly,
		GenerateORM:      !genProtoOnly && !genSkipORM,
		GenerateData:     !genProtoOnly,
		GenerateMain:     !genProtoOnly,
		GenerateConfig:   !genProtoOnly && !genSkipConfig,
		GenerateMakefile: !genProtoOnly && !genSkipMakefile,
		GenerateTests:    genTests && !genProtoOnly,
	}

	fmt.Printf("Generating code for service [%s] with ORM [%s]...\n", genServiceName, genOrmType)

	if genDryRun {
		tables, terr := sqlkratos.PlanTables(cmd.Context(), opts)
		if terr != nil {
			return fmt.Errorf("resolve tables: %w", terr)
		}
		fmt.Printf("Plan for service [%s]:\n", genServiceName)
		fmt.Printf("  Source:      %s (driver: %s)\n", dsn, genDriver)
		fmt.Printf("  ORM:         %s\n", genOrmType)
		fmt.Printf("  Servers:     %s\n", strings.Join(genServers, ", "))
		fmt.Printf("  API version: %s\n", genModuleVersion)
		fmt.Printf("  Output:      %s\n", outputPath)
		var skips []string
		if genProtoOnly {
			skips = append(skips, "proto-only (no server/service/ORM/config/Makefile)")
		}
		if genSkipORM {
			skips = append(skips, "ORM")
		}
		if genSkipConfig {
			skips = append(skips, "config")
		}
		if genSkipMakefile {
			skips = append(skips, "Makefile")
		}
		if len(skips) > 0 {
			fmt.Printf("  Skipped:     %s\n", strings.Join(skips, ", "))
		}
		fmt.Printf("  Resolved %d table(s):\n", len(tables))
		for _, t := range tables {
			note := ""
			if len(t.Fields) == 0 {
				note = "  (no fields — will be skipped)"
			}
			fmt.Printf("    - %s (%d field(s))%s\n", t.Name, len(t.Fields), note)
		}
		fmt.Printf("\033[36m[DRY-RUN] preview only — nothing was written. Re-run without --dry-run to execute.\033[m\n")
		return nil
	}

	if genCheck {
		return runDriftCheck(cmd, opts)
	}

	if err := sqlkratos.Generate(cmd.Context(), opts); err != nil {
		return err
	}

	fmt.Printf("\033[32mService [%s] generated successfully!\033[m\n", genServiceName)
	return nil
}

// runDriftCheck 把同一份生成参数跑进临时目录,与目标工作区逐文件比对:
// 生成侧有而工作区没有 = 漏改;两侧都有但内容不同 = 手改或过期。
// wiring.go 不参与比对——真实项目会在其上注入装配行,与全新生成必然不同。
// 有任何漂移时返回错误(退出码非 0),供 CI 门禁使用。
func runDriftCheck(cmd *cobra.Command, opts sqlkratos.GeneratorOptions) error {
	out := cmd.OutOrStdout()

	tmpDir, err := os.MkdirTemp("", "gowind-gen-check-")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// ent ORM 生成要求 schema 目录位于 Go 模块内(WriteSchema 做 packages
	// 加载),临时目录补一个桩 go.mod——生成完即弃,不影响任何产物。
	if werr := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module check.local\n\ngo 1.24\n"), 0o644); werr != nil {
		return fmt.Errorf("写入桩 go.mod 失败: %w", werr)
	}

	checkOpts := opts
	checkOpts.OutputPath = tmpDir
	checkOpts.GenerateTests = genTests
	if err := sqlkratos.Generate(cmd.Context(), checkOpts); err != nil {
		return fmt.Errorf("漂移检查的基准生成失败: %w", err)
	}

	type drift struct {
		rel  string
		kind string // missing / modified
	}
	var drifts []drift

	err = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, rerr := filepath.Rel(tmpDir, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		// 桩 go.mod 与装配注入过的 wiring.go 都不是可比产物。
		if rel == "go.mod" || filepath.Base(rel) == "wiring.go" {
			return nil
		}

		target := filepath.Join(opts.OutputPath, filepath.FromSlash(rel))
		tdata, terr := os.ReadFile(target)
		if terr != nil {
			if os.IsNotExist(terr) {
				drifts = append(drifts, drift{rel: rel, kind: "missing"})
				return nil
			}
			return terr
		}
		gdata, gerr := os.ReadFile(path)
		if gerr != nil {
			return gerr
		}
		if !bytes.Equal(tdata, gdata) {
			drifts = append(drifts, drift{rel: rel, kind: "modified"})
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(drifts) == 0 {
		fmt.Fprintf(out, "\033[32mSUCCESS: no drift — generated output matches the working tree.\033[m\n")
		return nil
	}

	fmt.Fprintf(out, "\033[33mDRIFT detected (%d file(s)) — generation output differs from the working tree:\033[m\n", len(drifts))
	for _, d := range drifts {
		if d.kind == "missing" {
			fmt.Fprintf(out, "  \033[33m~\033[m %s (generated file missing in the working tree)\n", d.rel)
		} else {
			fmt.Fprintf(out, "  \033[31m!\033[m %s (differs from fresh generation — hand-edited or outdated?)\n", d.rel)
		}
	}
	return fmt.Errorf("%d 个生成文件与工作区不一致,请重新生成或回退手改", len(drifts))
}

func extractProjectName(module string) string {
	if module == "" {
		return ""
	}
	if idx := len(module) - 1; idx >= 0 {
		for i := len(module) - 1; i >= 0; i-- {
			if module[i] == '/' {
				return module[i+1:]
			}
		}
	}
	return module
}
