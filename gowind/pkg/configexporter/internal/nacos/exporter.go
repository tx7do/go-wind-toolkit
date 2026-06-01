package nacos

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal/utils"
)

type Exporter struct {
	client  config_client.IConfigClient
	options *internal.Options
}

func NewExporter(options *internal.Options) *Exporter {
	cli := &Exporter{
		options: options,
	}

	cli.init()

	return cli
}

func (i *Exporter) init() {
	if i.options.Group == "" {
		i.options.Group = "DEFAULT_GROUP"
	}
	if i.options.Env == "" {
		i.options.Env = "dev"
	}
	if i.options.NamespaceId == "" {
		i.options.NamespaceId = "public"
	}

	// Nacos 服务器配置
	serverConfig := []constant.ServerConfig{
		{
			IpAddr:      "127.0.0.1",
			Port:        8848,
			ContextPath: "/nacos",
			Scheme:      "http",
		},
	}

	// 客户端配置
	clientConfig := constant.ClientConfig{
		NamespaceId:         i.options.NamespaceId, // 如果使用默认命名空间，留空即可
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		//RotateTime:          "1h",
		//MaxAge:              3,
		LogLevel: "debug",
	}

	// 创建配置客户端
	configClient, err := clients.CreateConfigClient(map[string]any{
		"serverConfigs": serverConfig,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		log.Fatalf("创建Nacos配置客户端失败: %v", err)
	}

	i.client = configClient
}

// Export 导入所有的配置
func (i *Exporter) Export() error {
	apps := utils.GetFolderNameList(i.options.ProjectRoot + "app/")
	for _, app := range apps {
		_ = i.ExportOneService(app)
	}

	return nil
}

// ExportOneService 导入单个配置
func (i *Exporter) ExportOneService(app string) error {
	files := i.getConfigFileList(i.options.ProjectRoot, app)

	var configType string
	var allContent []byte
	for _, file := range files {
		content := utils.ReadFile(i.options.ProjectRoot + "/" + file)
		allContent = append(allContent, content...)
		allContent = append(allContent, "\n"...)

		configType = getConfigType(file)
	}

	key := i.getServiceConfigNacosKey(i.options.ProjectName, app, configType)
	fmt.Println(key)
	if err := i.writeConfigToNacos(key, i.options.Group, configType, allContent); err != nil {
		fmt.Println(err.Error())
	}

	return nil
}

// writeConfigToNacos 写入配置到 Nacos
func (i *Exporter) writeConfigToNacos(key, group, configType string, value []byte) error {
	success, err := i.client.PublishConfig(vo.ConfigParam{
		DataId:  key,
		Group:   group,
		Content: string(value),
		Type:    configType, // 配置类型，可选值：properties, json, xml, yaml, text, html
	})
	if err != nil {
		log.Fatalf("发布配置失败: %v", err)
		return err
	}

	if success {
		//fmt.Printf("配置发布成功！DataId: %s, Group: %s\n", key, group)
	} else {
		fmt.Printf("配置发布失败！DataId: %s, Group: %s\n", key, group)
	}

	return err
}

// getServiceConfigFolder 获取某一个服务的配置文件夹路径
func (i *Exporter) getServiceConfigFolder(root, app string) string {
	return path.Join(root, "app/", app, "/service/configs/")
}

// getServiceConfigNacosKeySingleFile 获取配置的 Nacos Key
func (i *Exporter) getServiceConfigNacosKeySingleFile(project, app, fileName string) string {
	return fmt.Sprintf("%s-%s-service-%s", project, app, fileName)
}

func (i *Exporter) getServiceConfigNacosKey(project, app, configType string) string {
	return fmt.Sprintf("%s-%s-service-%s.%s", project, app, i.options.Env, configType)
}

// getConfigFileList 获取配置文件列表
func (i *Exporter) getConfigFileList(root, app string) []string {
	p := i.getServiceConfigFolder(root, app)
	return utils.GetFileList(p)
}

func getConfigType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName)) // 获取文件后缀并转换为小写
	switch ext {
	case ".properties":
		return "properties"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".txt":
		return "text"
	case ".html", ".htm":
		return "html"
	default:
		return "unknown" // 未知类型
	}
}
