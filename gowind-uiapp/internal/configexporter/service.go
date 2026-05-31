package configexporter

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ConfigType 远程配置中心类型
type ConfigType string

const (
	Consul ConfigType = "consul"
	Etcd   ConfigType = "etcd"
	Nacos  ConfigType = "nacos"
)

// RemoteConfig 远程配置参数
type RemoteConfig struct {
	Type        ConfigType `json:"type"`        // 配置中心类型: consul, etcd, nacos
	Endpoint    string     `json:"endpoint"`    // 配置服务器地址
	ProjectName string     `json:"projectName"` // 项目名（key前缀）
	Group       string     `json:"group"`       // Nacos 分组
	Env         string     `json:"env"`         // Nacos 环境
	NamespaceId string     `json:"namespaceId"` // Nacos 命名空间
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name         string   `json:"name"`         // 服务名
	ConfigFiles  []string `json:"configFiles"`  // 配置文件列表
	ConfigFolder string   `json:"configFolder"` // 配置文件夹路径
}

// ExportResult 导出结果
type ExportResult struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Service    string `json:"service,omitempty"`
	FilesCount int    `json:"filesCount,omitempty"`
}

// Validate 验证配置
func (c *RemoteConfig) Validate() string {
	if c.Endpoint == "" {
		return "配置服务器地址不能为空"
	}
	if c.ProjectName == "" {
		return "项目名不能为空"
	}
	switch c.Type {
	case Consul, Etcd, Nacos:
		// OK
	default:
		return "不支持的配置中心类型: " + string(c.Type)
	}
	return ""
}

// GetServiceConfigFolder 获取某一个服务的配置文件夹路径
func GetServiceConfigFolder(projectRoot, app string) string {
	return path.Join(projectRoot, "app", app, "service", "configs")
}

// GetServiceList 获取项目中的服务列表及其配置文件信息
func GetServiceList(projectRoot string) ([]ServiceInfo, error) {
	appRoot := path.Join(projectRoot, "app")
	if _, err := os.Stat(appRoot); os.IsNotExist(err) {
		return nil, fmt.Errorf("app 目录不存在: %s", appRoot)
	}

	names := getFolderNameList(appRoot)
	var services []ServiceInfo

	for _, name := range names {
		configFolder := GetServiceConfigFolder(projectRoot, name)
		files := getConfigFileList(configFolder)

		// 只包含有配置文件的服务
		if len(files) > 0 {
			services = append(services, ServiceInfo{
				Name:         name,
				ConfigFiles:  files,
				ConfigFolder: configFolder,
			})
		}
	}

	return services, nil
}

// GetSupportedTypes 获取支持的配置中心类型列表
func GetSupportedTypes() []map[string]string {
	return []map[string]string{
		{"value": "consul", "label": "Consul"},
		{"value": "etcd", "label": "Etcd"},
		{"value": "nacos", "label": "Nacos"},
	}
}

// ExportAll 导出所有服务配置到远程配置中心
// 通过直接读取配置文件并写入配置中心来实现
func ExportAll(
	typeName string,
	endpoint string,
	projectName string,
	projectRoot string,
	group string,
	env string,
	namespaceId string,
) error {
	services, err := GetServiceList(projectRoot)
	if err != nil {
		return err
	}

	for _, svc := range services {
		if err := ExportOne(typeName, endpoint, projectName, projectRoot, group, env, namespaceId, svc.Name); err != nil {
			return fmt.Errorf("导出服务 %s 失败: %w", svc.Name, err)
		}
	}

	return nil
}

// ExportOne 导出单个服务的配置到远程配置中心
func ExportOne(
	typeName string,
	endpoint string,
	projectName string,
	projectRoot string,
	group string,
	env string,
	namespaceId string,
	serviceName string,
) error {
	// 使用内建实现直接通过 HTTP API 写入配置中心
	return exportDirect(typeName, endpoint, projectName, projectRoot, group, env, namespaceId, serviceName)
}

// exportDirect 直接通过 SDK 写入配置中心（简化实现）
func exportDirect(
	typeName string,
	endpoint string,
	projectName string,
	projectRoot string,
	group string,
	env string,
	namespaceId string,
	serviceName string,
) error {
	configFolder := GetServiceConfigFolder(projectRoot, serviceName)
	files := getConfigFileList(configFolder)
	if len(files) == 0 {
		return fmt.Errorf("服务 %s 没有配置文件", serviceName)
	}

	// 读取所有配置文件内容
	var allContent []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("读取配置文件 %s 失败: %w", file, err)
		}
		allContent = append(allContent, string(content))
	}

	mergedContent := strings.Join(allContent, "\n")

	// 根据类型写入不同的配置中心
	switch ConfigType(typeName) {
	case Consul:
		return writeConsul(endpoint, projectName, serviceName, mergedContent)
	case Etcd:
		return writeEtcd(endpoint, projectName, serviceName, mergedContent)
	case Nacos:
		return writeNacos(endpoint, projectName, serviceName, group, env, namespaceId, mergedContent)
	default:
		return fmt.Errorf("不支持的配置中心类型: %s", typeName)
	}
}

// getFolderNameList 获取当前文件夹下面的所有文件夹名的列表
func getFolderNameList(root string) []string {
	var names []string
	fs, _ := os.ReadDir(root)
	for _, file := range fs {
		if file.IsDir() {
			names = append(names, file.Name())
		}
	}
	return names
}

// getConfigFileList 获取配置文件列表
func getConfigFileList(folder string) []string {
	var files []string

	filepath.Walk(folder, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// 只包含常见配置文件格式
		ext := strings.ToLower(filepath.Ext(p))
		switch ext {
		case ".yaml", ".yml", ".json", ".toml", ".properties", ".txt":
			files = append(files, p)
		}
		return nil
	})

	return files
}

// writeConsul 写入配置到 Consul
func writeConsul(endpoint, project, app, content string) error {
	key := fmt.Sprintf("%s/%s/service/config", project, app)
	consulURL := fmt.Sprintf("http://%s/v1/kv/%s", endpoint, url.PathEscape(key))

	req, err := http.NewRequest("PUT", consulURL, strings.NewReader(content))
	if err != nil {
		return fmt.Errorf("创建 Consul 请求失败: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("写入 Consul 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Consul 返回错误 (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// writeEtcd 写入配置到 Etcd (通过 HTTP JSON API)
func writeEtcd(endpoint, project, app, content string) error {
	// Etcd v3 不支持原生 REST，通常需要 gRPC
	// 这里返回提示信息，建议安装 etcdctl 或使用 Consul/Nacos
	return fmt.Errorf("Etcd 需要 gRPC 协议支持，建议使用 cfgexp CLI 工具或切换到 Consul/Nacos")
}

// writeNacos 写入配置到 Nacos
func writeNacos(endpoint, project, app, group, env, namespaceId, content string) error {
	if group == "" {
		group = "DEFAULT_GROUP"
	}
	if env == "" {
		env = "dev"
	}
	if namespaceId == "" {
		namespaceId = "public"
	}

	dataId := fmt.Sprintf("%s-%s-service-%s.yaml", project, app, env)
	nacosURL := fmt.Sprintf("http://%s/nacos/v1/cs/configs", endpoint)

	form := url.Values{}
	form.Set("dataId", dataId)
	form.Set("group", group)
	form.Set("tenant", namespaceId)
	form.Set("content", content)

	resp, err := http.PostForm(nacosURL, form)
	if err != nil {
		return fmt.Errorf("写入 Nacos 失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Nacos 返回错误 (%d): %s", resp.StatusCode, string(body))
	}

	if strings.ToLower(strings.TrimSpace(string(body))) != "true" {
		return fmt.Errorf("Nacos 发布配置失败: %s", string(body))
	}
	return nil
}
