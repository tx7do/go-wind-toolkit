package internal

type ImporterType string

const (
	Apollo     ImporterType = "apollo"
	Consul     ImporterType = "consul"
	Etcd       ImporterType = "etcd"
	Kubernetes ImporterType = "kubernetes"
	Nacos      ImporterType = "nacos"
	Polaris    ImporterType = "polaris"
)

// Options 参数
type Options struct {
	Service ImporterType

	Endpoint string // 配置服务器远程地址

	ProjectName string // 项目名
	ProjectRoot string // 项目根目录

	MergeSingle bool // 是否合并单个配置文件

	Group       string // for nacos
	Env         string // for nacos
	NamespaceId string // for nacos
}
