package ent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

func RunGenerate(cmd *cobra.Command, args []string) error {
	cmdArgs, _ := pkg.SplitArgs(cmd, args)

	inspector, err := pkg.NewModuleInspectorFromGo(cmd.Context(), "")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	// 先在模块根目录运行 `go mod tidy`
	if err = pkg.GoModTidy(cmd.Context(), inspector.Root); err != nil {
		return err
	}

	var serviceName string

	if len(cmdArgs) > 0 {
		serviceName = strings.TrimSpace(cmdArgs[0])
		if serviceName == "" {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: service name is required\033[m\n")
			return fmt.Errorf("service name is required")
		}

		var valid bool
		valid, err = pkg.IsValidServiceName(inspector.Root, serviceName)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
			return err
		}

		if !valid {
			err = fmt.Errorf("service '%s' does not exist or is not valid (missing cmd/server or configs)", serviceName)
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
			return err
		}
	} else {
		// 未指定服务名称，检查当前目录是否为服务目录

		var wd string
		wd, err = os.Getwd()
		if err != nil {
			fmt.Printf("os.Getwd error: %v\n", err)
		}

		serviceName, err = pkg.ExtractServiceName(inspector.Root, wd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
			return err
		}
	}

	// 如果没有传入 service 或者为空，则遍历所有 service
	if len(serviceName) == 0 {
		return generateEntAllService(cmd.Context(), inspector.Root)
	}

	servicePath := filepath.Join(inspector.Root, "app", serviceName, "service")
	err = generateEnt(cmd.Context(), servicePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: generate for service %s failed: %v\033[m\n", serviceName, err)
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Generated ent for service %s successfully.\n", serviceName)
	return nil
}

// generateEntAllService 在模块根目录下查找 app 目录，并对其中每个包含 internal/data/ent/schema 的服务目录执行 generateEnt。
func generateEntAllService(ctx context.Context, projectRootPath string) error {
	appDir := filepath.Join(projectRootPath, "app")
	entries, err := os.ReadDir(appDir)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: failed to read app directory: %s\033[m\n", err.Error())
		return err
	}

	var lastErr error
	var processed int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serviceName := entry.Name()
		servicePath := filepath.Join(appDir, serviceName, "service")

		entSchemaPath := filepath.Join(servicePath, "internal", "data", "ent", "schema")
		if _, statErr := os.Stat(entSchemaPath); statErr != nil {
			// 没有 service 子目录则跳过
			continue
		}

		processed++
		if genErr := generateEnt(ctx, servicePath); genErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: generate for service %s failed: %v\033[m\n", serviceName, genErr)
			lastErr = genErr
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "Generated ent for service %s successfully.\n", serviceName)
		}
	}
	if processed == 0 {
		err = fmt.Errorf("no services found under %s", appDir)
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}
	return lastErr
}

// generateEnt 在指定服务目录下执行 ent code generation，要求该目录下存在 internal/data/ent/schema 目录。
func generateEnt(ctx context.Context, serviceRootPath string) error {
	target := filepath.Join(serviceRootPath, "internal", "data", "ent", "schema")
	e := NewEntCmd(target)
	return e.RunGenerate(ctx,
		"--feature", "privacy",
		"--feature", "entql",
		"--feature", "sql/modifier",
		"--feature", "sql/upsert",
		"--feature", "sql/lock",
		// 生成 migrate 包的版本化迁移 API(NamedDiff),供 gow migrate --versioned 使用。
		"--feature", "sql/versioned-migration",
	)
}

// GenerateService 为指定服务执行 ent code generation，供其他命令（如 migrate）
// 在缺少 ent/migrate 包时自动补齐。要求服务目录下存在 internal/data/ent/schema。
func GenerateService(ctx context.Context, serviceRootPath string) error {
	return generateEnt(ctx, serviceRootPath)
}

func RunAdd(cmd *cobra.Command, args []string) error {
	// 最少需要 service 和 schemas
	if len(args) < 2 {
		_ = cmd.Help()
		return fmt.Errorf("usage: ent add <service> <schemas>")
	}

	service := strings.TrimSpace(args[0])
	if service == "" {
		_ = cmd.Help()
		return fmt.Errorf("service name is empty")
	}

	// 支持多个参数或一个逗号分隔字符串，去掉空格并拆分
	namesArg := strings.Join(args[1:], "")
	namesArg = strings.ReplaceAll(namesArg, " ", "")
	if namesArg == "" {
		_ = cmd.Help()
		return fmt.Errorf("no schema names provided")
	}

	names := strings.Split(namesArg, ",")
	// 过滤空名称
	var filtered []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			filtered = append(filtered, n)
		}
	}
	if len(filtered) == 0 {
		_ = cmd.Help()
		return fmt.Errorf("no valid schema names after parsing")
	}

	inspector, err := pkg.NewModuleInspectorFromGo(cmd.Context(), "")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	servicePath := filepath.Join(inspector.Root, "app", service, "service")
	// EntCmd.TargetDir 一律取 ent schema 目录本身;`ent new` 靠绝对 --target 定落点,
	// 不再依赖工作目录恰好是 internal/data。
	target := filepath.Join(servicePath, "internal", "data", "ent", "schema")

	e := NewEntCmd(target)
	if err = e.RunNew(cmd.Context(), names); err != nil {
		return err
	}

	// 新增 schema 后立即重新生成 ent 运行时,保证代码可直接编译使用。
	if err = generateEnt(cmd.Context(), servicePath); err != nil {
		return fmt.Errorf("regenerate ent code: %w", err)
	}

	fmt.Printf("Added schema(s) [%s] to service [%s]; ent code regenerated.\n",
		strings.Join(filtered, ", "), service)
	return nil
}
