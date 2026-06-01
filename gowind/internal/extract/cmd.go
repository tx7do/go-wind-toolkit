package extract

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	pkgExtract "github.com/tx7do/go-wind-toolkit/gowind/pkg/extract"
)

var (
	extractOrmType string
	extractKeepSrc bool
	extractObj     []string
)

// CmdExtract 提取命令
var CmdExtract = &cobra.Command{
	Use:   "extract <source-service> <target-service> --obj <model> [--obj <model>...]",
	Short: "extract service modules from one service to another",
	Long: `Extract service modules (schema, repo, service, wire, server) from source service to target service.

This is used for microservice evolution — gradually splitting a monolithic service
into smaller, independently deployable services.

ORM type is auto-detected from source service directory structure.

Examples:
  gow extract admin user --obj role
  gow extract admin user --obj role --obj permission
  gow extract admin user --obj role,permission
  gow extract admin user --obj role --orm ent
  gow extract admin user --obj role --keep-source`,
	Args: cobra.ExactArgs(2),
	RunE: runExtract,
}

func init() {
	CmdExtract.Flags().StringArrayVarP(&extractObj, "obj", "o", nil, "object/model names to extract (comma-separated or repeated flag)")
	CmdExtract.Flags().StringVarP(&extractOrmType, "orm", "", "", "ORM type override: ent, gorm (auto-detected by default)")
	CmdExtract.Flags().BoolVarP(&extractKeepSrc, "keep-source", "", false, "Keep source files instead of deleting them")
}

func extractProjectName(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return ""
	}

	if strings.Contains(module, "/") {
		parts := strings.Split(module, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			seg := strings.TrimSpace(parts[i])
			if seg != "" {
				return seg
			}
		}
	}

	return module
}

func runExtract(cmd *cobra.Command, args []string) error {
	sourceService := strings.TrimSpace(args[0])
	targetService := strings.TrimSpace(args[1])

	if sourceService == "" || targetService == "" {
		return fmt.Errorf("source and target service names are required")
	}

	if sourceService == targetService {
		return fmt.Errorf("source and target service cannot be the same")
	}

	// 解析 --obj 参数（支持逗号分隔和重复 flag）
	var filtered []string
	for _, obj := range extractObj {
		for _, part := range strings.Split(obj, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				filtered = append(filtered, part)
			}
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("at least one object name is required, use --obj <name>")
	}

	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	srcValid, err := pkg.IsValidServiceName(inspector.Root, sourceService)
	if err != nil {
		return fmt.Errorf("validate source service: %w", err)
	}
	if !srcValid {
		return fmt.Errorf("source service '%s' does not exist or is not valid", sourceService)
	}

	// 目标服务不存在时，extractor 会自动创建（ensureTargetService）
	_, _ = pkg.IsValidServiceName(inspector.Root, targetService)

	ormType := extractOrmType
	if ormType == "" {
		srcPath := fmt.Sprintf("%s/app/%s/service", inspector.Root, sourceService)
		ormType = pkgExtract.DetectOrmType(srcPath)
		if ormType == "" {
			return fmt.Errorf("cannot detect ORM type for service '%s', please specify with --orm", sourceService)
		}
		fmt.Printf("Auto-detected ORM type: %s\n", ormType)
	}

	projectName := extractProjectName(inspector.ModPath)

	opts := pkgExtract.Options{
		RootPath:      inspector.Root,
		ModulePath:    inspector.ModPath,
		ProjectName:   projectName,
		SourceService: sourceService,
		TargetService: targetService,
		Models:        filtered,
		OrmType:       ormType,
		KeepSource:    extractKeepSrc,
	}

	fmt.Printf("Extracting %d model(s) from [%s] to [%s] (ORM: %s)...\n",
		len(filtered), sourceService, targetService, ormType)

	extractor := pkgExtract.NewExtractor(opts)
	if err = extractor.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	fmt.Printf("\033[32mExtraction completed successfully!\033[m\n")
	fmt.Printf("  Source: %s\n", sourceService)
	fmt.Printf("  Target: %s\n", targetService)
	fmt.Printf("  Models: %s\n", strings.Join(filtered, ", "))

	return nil
}
