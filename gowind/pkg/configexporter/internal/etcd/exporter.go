package etcd

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal/utils"
)

type Exporter struct {
	client  *clientv3.Client
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
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{i.options.Endpoint},
	})
	if err != nil {
		return
	}

	i.client = cli
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
	for _, file := range files {
		content := utils.ReadFile(i.options.ProjectRoot + "/" + file)
		key := i.getServiceConfigEtcdKey(i.options.ProjectName, app, filepath.Base(file))
		fmt.Println(key)
		if err := i.writeConfigToEtcd(key, content); err != nil {
			fmt.Println(err.Error())
		}
	}

	return nil
}

// writeConfigToEtcd 写入配置到 Etcd
func (i *Exporter) writeConfigToEtcd(key string, value []byte) error {
	_, err := i.client.Put(context.Background(), key, string(value))
	return err
}

// getServiceConfigFolder 获取某一个服务的配置文件夹路径
func (i *Exporter) getServiceConfigFolder(root, app string) string {
	return path.Join(root, "app/", app, "/service/configs/")
}

// getServiceConfigEtcdKey 获取配置的 Etcd Key
func (i *Exporter) getServiceConfigEtcdKey(project, app, fileName string) string {
	return fmt.Sprintf("/%s/%s/service/%s", project, app, fileName)
}

// getConfigFileList 获取配置文件列表
func (i *Exporter) getConfigFileList(root, app string) []string {
	p := i.getServiceConfigFolder(root, app)
	return utils.GetFileList(p)
}
