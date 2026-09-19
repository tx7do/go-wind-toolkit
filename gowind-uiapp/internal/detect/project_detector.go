package detect

import (
	"os"
	"path/filepath"
	"strings"
)

type ProjectInfo struct {
	Root         string   `json:"Root"`
	GoVersion    string   `json:"GoVersion"`
	ModPath      string   `json:"ModPath"`
	Main         bool     `json:"Main"`
	Version      string   `json:"Version"`
	Replace      *Module  `json:"Replace,omitempty"`
	Dependencies []Module `json:"Dependencies,omitempty"`

	Services []string `json:"Services,omitempty"`
	HasApi   bool     `json:"HasApi,omitempty"`
}

type ProjectDetector struct {
}

func NewProjectDetector() *ProjectDetector {
	return &ProjectDetector{}
}

// Detect 检测指定路径下的 Go 项目，返回项目的基本信息。
func (pd *ProjectDetector) Detect(projectPath string) (*ProjectInfo, error) {
	// 清理路径：标准化 Windows 路径，去除控制字符和多余空格
	projectPath = strings.TrimSpace(projectPath)
	projectPath = strings.ReplaceAll(projectPath, "\r\n", "")
	projectPath = strings.ReplaceAll(projectPath, "\n", "")
	projectPath = strings.ReplaceAll(projectPath, "\r", "")
	projectPath = strings.TrimFunc(projectPath, func(r rune) bool {
		return r < 32 && r != '\t'
	})

	// 将正斜杠转换为反斜杠（Windows 兼容）
	projectPath = strings.ReplaceAll(projectPath, "/", "\\")

	// 使用 filepath.Clean 进一步规范化路径
	projectPath = filepath.Clean(projectPath)

	var err error
	var pi *ProjectInfo
	pi, err = pd.detectInternal(projectPath)
	if err != nil {
		return nil, err
	}

	// 保存清理后的 Root 路径
	pi.Root = projectPath

	return pi, nil
}

// detectInternal 执行实际的项目检测逻辑（假设路径已清理）。
func (pd *ProjectDetector) detectInternal(projectPath string) (*ProjectInfo, error) {
	inspector, err := NewModuleInspectorFromGo(projectPath)
	if err != nil {
		return nil, err
	}

	var pi ProjectInfo
	pi.ModPath = inspector.ModPath
	pi.GoVersion = inspector.GoVersion
	pi.Main = inspector.Main
	pi.Version = inspector.Version
	pi.Replace = inspector.Replace
	pi.Dependencies = inspector.Dependencies

	// 收集服务列表
	services, err := pd.collectServices(projectPath)
	if err == nil {
		pi.Services = services
	}

	// 检测是否包含 API 定义
	pi.HasApi = pd.collectApi(projectPath)

	return &pi, nil
}

// collectServices 收集项目中的服务列表，假设服务目录位于 projectPath/app 下。
func (pd *ProjectDetector) collectServices(projectPath string) ([]string, error) {
	var services []string

	servicesPath := filepath.Join(projectPath, "app")
	entries, err := os.ReadDir(servicesPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			services = append(services, entry.Name())
		}
	}

	return services, nil
}

// collectApi 检测项目中是否包含 API 定义，假设 API 定义位于 projectPath/api 下。
func (pd *ProjectDetector) collectApi(projectPath string) bool {
	apiPath := filepath.Join(projectPath, "api")
	_, err := os.Stat(apiPath)
	if err != nil {
		return false
	}

	bufYamlPath := filepath.Join(apiPath, "buf.yaml")
	_, err = os.Stat(bufYamlPath)
	if err != nil {
		return false
	}

	protosPath := filepath.Join(apiPath, "protos")
	_, err = os.Stat(protosPath)
	if err != nil {
		return false
	}

	return true
}
