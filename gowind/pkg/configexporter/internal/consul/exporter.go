package consul

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul/api"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal/utils"
)

type Exporter struct {
	client  *api.Client
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
	client, err := api.NewClient(&api.Config{
		Address: i.options.Endpoint,
	})
	if err != nil {
		panic(err)
	}

	i.client = client
}

// Export 导入所有的配置
func (i *Exporter) Export() error {
	apps := utils.GetFolderNameList(path.Join(i.options.ProjectRoot, "app/"))
	for _, app := range apps {
		_ = i.ExportOneService(app)
	}

	return nil
}

// ExportOneService 导入单个配置
func (i *Exporter) ExportOneService(app string) error {
	files := i.getConfigFileList(i.options.ProjectRoot, app)
	for _, file := range files {
		content := utils.ReadFile(path.Join(i.options.ProjectRoot, file))
		key := i.getServiceConfigConsulKey(i.options.ProjectName, app, filepath.Base(file))
		fmt.Println(key)
		if err := i.writeConfigToConsul(key, content); err != nil {
			fmt.Println(err.Error())
		}
	}

	return nil
}

// writeConfigToConsul 写入配置到Consul
func (i *Exporter) writeConfigToConsul(key string, value []byte) error {
	if _, err := i.client.KV().Put(&api.KVPair{Key: key, Value: value}, nil); err != nil {
		return err
	}
	return nil
}

// getServiceConfigFolder 获取某一个服务的配置文件夹路径
func (i *Exporter) getServiceConfigFolder(root, app string) string {
	return path.Join(root, "app/", app, "/service/configs/")
}

func (i *Exporter) getServiceConfigConsulKey(project, app, fileName string) string {
	key := path.Join(project, "/", app, "/service/", fileName)
	key = strings.Replace(key, "\\", "/", -1)
	return key
}

// getConfigFileList 获取配置文件列表
func (i *Exporter) getConfigFileList(root, app string) []string {
	p := i.getServiceConfigFolder(root, app)
	return utils.GetFileList(p)
}
