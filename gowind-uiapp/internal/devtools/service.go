package devtools

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandResult 命令执行结果
type CommandResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name       string   `json:"name"`
	HasServer  bool     `json:"hasServer"`
	HasConfig  bool     `json:"hasConfig"`
	HasEnt     bool     `json:"hasEnt"`
	EntSchemas []string `json:"entSchemas,omitempty"`
	HasGorm    bool     `json:"hasGorm"`
	GormModels []string `json:"gormModels,omitempty"`
}

// CreateProjectOptions 创建项目选项
type CreateProjectOptions struct {
	Name      string `json:"name"`
	Module    string `json:"module"`
	RepoURL   string `json:"repoUrl"`
	Branch    string `json:"branch"`
	ParentDir string `json:"parentDir"`
}

// AddServiceOptions 添加服务选项
type AddServiceOptions struct {
	ServiceName string   `json:"serviceName"`
	Servers     []string `json:"servers"`
	DbClients   []string `json:"dbClients"`
}

const defaultTemplateRepo = "https://github.com/tx7do/go-wind-admin-template.git"
const defaultGiteeTemplateRepo = "https://gitee.com/tx7do/go-wind-admin-template.git"
const templateModuleName = "github.com/tx7do/go-wind-admin-template"

// GetServices 获取项目中的服务列表及详细信息
func GetServices(projectRoot string) ([]ServiceInfo, error) {
	appDir := filepath.Join(projectRoot, "app")
	entries, err := os.ReadDir(appDir)
	if err != nil {
		return nil, fmt.Errorf("读取 app 目录失败: %w", err)
	}

	var services []ServiceInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		servicePath := filepath.Join(appDir, name, "service")
		info := ServiceInfo{Name: name}

		// 检查是否有 cmd/server
		if _, err := os.Stat(filepath.Join(servicePath, "cmd", "server")); err == nil {
			info.HasServer = true
		}
		// 检查是否有 configs
		if _, err := os.Stat(filepath.Join(servicePath, "configs")); err == nil {
			info.HasConfig = true
		}
		// 检查是否有 ent schema
		schemaDir := filepath.Join(servicePath, "internal", "data", "ent", "schema")
		if entries, err := os.ReadDir(schemaDir); err == nil {
			info.HasEnt = true
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
					info.EntSchemas = append(info.EntSchemas, strings.TrimSuffix(e.Name(), ".go"))
				}
			}
		}

		// 检查是否有 gorm model
		gormDir := filepath.Join(servicePath, "internal", "data", "gorm", "models")
		if entries, err := os.ReadDir(gormDir); err == nil {
			info.HasGorm = true
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
					info.GormModels = append(info.GormModels, strings.TrimSuffix(e.Name(), ".go"))
				}
			}
		}

		services = append(services, info)
	}

	return services, nil
}

// RunService 由 ProcessManager 处理，见 process_manager.go

// RunBufGenerate 运行 buf generate
func RunBufGenerate(projectRoot string) *CommandResult {
	apiPath := filepath.Join(projectRoot, "api")
	if _, err := os.Stat(apiPath); err != nil {
		return &CommandResult{Success: false, Error: "api 目录不存在"}
	}

	// 先检查 buf 是否安装
	if _, err := exec.LookPath("buf"); err != nil {
		return &CommandResult{Success: false, Error: "buf 未安装，请先运行: go install github.com/bufbuild/buf/cmd/buf@latest"}
	}

	// 检查 buf.lock 是否存在，不存在则先 dep update
	lockPath := filepath.Join(apiPath, "buf.lock")
	if _, err := os.Stat(lockPath); err != nil {
		if result := runCommand(apiPath, "buf", "dep", "update"); !result.Success {
			return result
		}
	}

	// 扫描 .gen.yaml 文件
	var genFiles []string
	filepath.WalkDir(apiPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".gen.yaml") {
			genFiles = append(genFiles, path)
		}
		return nil
	})

	if len(genFiles) == 0 {
		return &CommandResult{Success: false, Error: "未找到 .gen.yaml 文件"}
	}

	var allOutput strings.Builder
	for _, genFile := range genFiles {
		relPath, _ := filepath.Rel(apiPath, genFile)
		allOutput.WriteString(fmt.Sprintf("使用模板: %s\n", relPath))
		result := runCommand(apiPath, "buf", "generate", "--template", genFile)
		allOutput.WriteString(result.Output)
		if !result.Success {
			allOutput.WriteString(result.Error)
			return &CommandResult{Success: false, Output: allOutput.String(), Error: fmt.Sprintf("buf generate 失败: %s", relPath)}
		}
	}

	allOutput.WriteString("Proto 代码生成完成\n")
	return &CommandResult{Success: true, Output: allOutput.String()}
}

// RunEntGenerate 运行 ent generate
func RunEntGenerate(projectRoot, serviceName string) *CommandResult {
	servicePath := filepath.Join(projectRoot, "app", serviceName, "service")
	schemaDir := filepath.Join(servicePath, "internal", "data", "ent", "schema")

	if _, err := os.Stat(schemaDir); err != nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("服务 %s 没有 ent schema 目录", serviceName)}
	}

	// 使用 go run entgo.io/ent/cmd/ent
	args := []string{
		"run", "entgo.io/ent/cmd/ent",
		"generate",
		"--feature", "privacy",
		"--feature", "entql",
		"--feature", "sql/modifier",
		"--feature", "sql/upsert",
		"--feature", "sql/lock",
		schemaDir,
	}
	return runCommand(servicePath, "go", args...)
}

// RunEntGenerateAll 对所有有 ent schema 的服务运行 ent generate
func RunEntGenerateAll(projectRoot string) *CommandResult {
	services, err := GetServices(projectRoot)
	if err != nil {
		return &CommandResult{Success: false, Error: err.Error()}
	}

	var allOutput strings.Builder
	var hasError bool
	for _, svc := range services {
		if !svc.HasEnt {
			continue
		}
		allOutput.WriteString(fmt.Sprintf("--- 生成 Ent 代码: %s ---\n", svc.Name))
		result := RunEntGenerate(projectRoot, svc.Name)
		allOutput.WriteString(result.Output)
		if !result.Success {
			allOutput.WriteString(result.Error + "\n")
			hasError = true
		}
	}

	return &CommandResult{
		Success: !hasError,
		Output:  allOutput.String(),
	}
}

// RunWire 运行 wire 生成
func RunWire(projectRoot, serviceName string) *CommandResult {
	serverPath := filepath.Join(projectRoot, "app", serviceName, "service", "cmd", "server")
	if _, err := os.Stat(serverPath); err != nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("服务 %s 的 cmd/server 目录不存在", serviceName)}
	}

	// 优先使用全局 wire，否则用 go run
	if wirePath, err := exec.LookPath("wire"); err == nil {
		return runCommand(serverPath, wirePath)
	}
	return runCommand(serverPath, "go", "run", "-mod=mod", "github.com/google/wire/cmd/wire")
}

// RunWireAll 对所有服务运行 wire
func RunWireAll(projectRoot string) *CommandResult {
	services, err := GetServices(projectRoot)
	if err != nil {
		return &CommandResult{Success: false, Error: err.Error()}
	}

	var allOutput strings.Builder
	var hasError bool
	for _, svc := range services {
		if !svc.HasServer {
			continue
		}
		allOutput.WriteString(fmt.Sprintf("--- 生成 Wire: %s ---\n", svc.Name))
		result := RunWire(projectRoot, svc.Name)
		allOutput.WriteString(result.Output)
		if !result.Success {
			allOutput.WriteString(result.Error + "\n")
			hasError = true
		}
	}

	return &CommandResult{
		Success: !hasError,
		Output:  allOutput.String(),
	}
}

// RunGoModTidy 运行 go mod tidy
func RunGoModTidy(projectRoot string) *CommandResult {
	return runCommand(projectRoot, "go", "mod", "tidy")
}

// CreateProject 创建新项目
func CreateProject(ctx context.Context, opts CreateProjectOptions) *CommandResult {
	if opts.Name == "" {
		return &CommandResult{Success: false, Error: "项目名称不能为空"}
	}
	if opts.ParentDir == "" {
		return &CommandResult{Success: false, Error: "请选择项目父目录"}
	}

	repoURL := opts.RepoURL
	if repoURL == "" {
		repoURL = defaultTemplateRepo
	}

	moduleName := opts.Module
	if moduleName == "" {
		moduleName = opts.Name
	}

	projectDir := filepath.Join(opts.ParentDir, opts.Name)

	// 检查目录是否已存在
	if _, err := os.Stat(projectDir); err == nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("目录已存在: %s", projectDir)}
	}

	var allOutput strings.Builder

	// git clone
	allOutput.WriteString(fmt.Sprintf("正在克隆模板仓库: %s\n", repoURL))
	cloneArgs := []string{"clone", "--depth", "1"}
	if opts.Branch != "" {
		cloneArgs = append(cloneArgs, "-b", opts.Branch)
	}
	cloneArgs = append(cloneArgs, repoURL, projectDir)

	cloneCmd := exec.CommandContext(ctx, "git", cloneArgs...)
	out, err := cloneCmd.CombinedOutput()
	allOutput.WriteString(string(out))
	if err != nil {
		return &CommandResult{Success: false, Output: allOutput.String(), Error: fmt.Sprintf("git clone 失败: %v", err)}
	}

	// 删除 .git 目录
	gitDir := filepath.Join(projectDir, ".git")
	os.RemoveAll(gitDir)
	gitHubDir := filepath.Join(projectDir, ".github")
	os.RemoveAll(gitHubDir)

	// 替换模块名
	allOutput.WriteString(fmt.Sprintf("替换模块名: %s -> %s\n", templateModuleName, moduleName))
	updatedCount, err := replaceInDir(projectDir, templateModuleName, moduleName)
	if err != nil {
		return &CommandResult{Success: false, Output: allOutput.String(), Error: fmt.Sprintf("替换模块名失败: %v", err)}
	}
	allOutput.WriteString(fmt.Sprintf("已更新 %d 个文件\n", updatedCount))

	// go mod tidy
	allOutput.WriteString("运行 go mod tidy...\n")
	tidyResult := RunGoModTidy(projectDir)
	allOutput.WriteString(tidyResult.Output)
	if !tidyResult.Success {
		allOutput.WriteString(tidyResult.Error)
	}

	allOutput.WriteString(fmt.Sprintf("项目 %s 创建成功!\n", projectDir))
	return &CommandResult{Success: true, Output: allOutput.String()}
}

// AddService 向已有项目添加新服务
func AddService(projectRoot string, opts AddServiceOptions) *CommandResult {
	if opts.ServiceName == "" {
		return &CommandResult{Success: false, Error: "服务名称不能为空"}
	}

	servicePath := filepath.Join(projectRoot, "app", opts.ServiceName, "service")
	if _, err := os.Stat(servicePath); err == nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("服务目录已存在: %s", servicePath)}
	}

	// 使用 gowind/pkg/generators 生成服务代码
	servers := opts.Servers
	if len(servers) == 0 {
		servers = []string{"grpc"}
	}
	dbClients := opts.DbClients
	if len(dbClients) == 0 {
		dbClients = []string{"ent"}
	}

	projectName := extractProjectName(projectRoot)

	// 获取模块路径
	modPath, err := getModulePath(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return &CommandResult{Success: false, Error: fmt.Sprintf("读取 go.mod 失败: %v", err)}
	}

	var allOutput strings.Builder
	allOutput.WriteString(fmt.Sprintf("正在创建服务: %s\n", opts.ServiceName))

	// 生成各层代码
	gen := NewServiceGenerator(modPath, projectName, opts.ServiceName, projectRoot, servers, dbClients)
	if err := gen.Generate(); err != nil {
		return &CommandResult{Success: false, Output: allOutput.String(), Error: fmt.Sprintf("服务生成失败: %v", err)}
	}

	allOutput.WriteString(fmt.Sprintf("服务 %s 创建成功!\n", opts.ServiceName))
	return &CommandResult{Success: true, Output: allOutput.String()}
}

// ========== 内部工具函数 ==========

func runCommand(dir string, name string, args ...string) *CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		return &CommandResult{
			Success: false,
			Output:  output,
			Error:   err.Error(),
		}
	}
	return &CommandResult{Success: true, Output: output}
}

func replaceInDir(rootDir, old, new string) (int, error) {
	includeExts := map[string]bool{".go": true, ".mod": true, ".yaml": true, ".yml": true}
	var updated int

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if !includeExts[ext] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(data, []byte(old)) {
			return nil
		}

		newData := bytes.ReplaceAll(data, []byte(old), []byte(new))
		if err := os.WriteFile(path, newData, 0o644); err != nil {
			return err
		}
		updated++
		return nil
	})

	return updated, err
}

func extractProjectName(projectRoot string) string {
	base := filepath.Base(projectRoot)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}

func getModulePath(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("module 路径未找到")
}
