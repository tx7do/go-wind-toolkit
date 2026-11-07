// Package configcmd 实现 `gow config`：把服务的本地配置导出到远程配置中心
// (Consul/Etcd/Nacos),与桌面端/CLI 同一引擎(pkg/configexporter)。
package configcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	ce "github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter"
)

var (
	cfgType      string
	cfgEndpoint  string
	cfgProject   string
	cfgGroup     string
	cfgEnv       string
	cfgNamespace string
	cfgUsername  string
	cfgPassword  string
	cfgCaCert    string
	cfgClientCer string
	cfgClientKey string
	cfgService   string
	cfgPath      string
	cfgDryRun    bool
)

// CmdConfig `gow config` 命令组。
var CmdConfig = &cobra.Command{
	Use:   "config",
	Short: "服务配置导出到远程配置中心(Consul/Etcd/Nacos)",
	Long: `将 app/<服务>/service/configs 下的配置文件导出到远程配置中心。

子命令:
  types    列出支持的配置中心类型
  services 列出项目内可导出的服务
  export   执行导出(缺省全部服务,--service 指定单个)`,
	SilenceUsage: true,
}

func buildRemoteConfig(cmd *cobra.Command) (ce.RemoteConfig, error) {
	rc := ce.RemoteConfig{
		Type:        ce.ConfigType(cfgType),
		Endpoint:    cfgEndpoint,
		ProjectName: cfgProject,
		Group:       cfgGroup,
		Env:         cfgEnv,
		NamespaceId: cfgNamespace,
		Username:    cfgUsername,
		Password:    cfgPassword,
	}
	if p := cfgCaCert; p != "" {
		data, err := os.ReadFile(p)
		if err != nil {
			return rc, fmt.Errorf("读取 ca-cert 失败: %w", err)
		}
		rc.CaCertPem = string(data)
	}
	if p := cfgClientCer; p != "" {
		data, err := os.ReadFile(p)
		if err != nil {
			return rc, fmt.Errorf("读取 client-cert 失败: %w", err)
		}
		rc.ClientCertPem = string(data)
	}
	if p := cfgClientKey; p != "" {
		data, err := os.ReadFile(p)
		if err != nil {
			return rc, fmt.Errorf("读取 client-key 失败: %w", err)
		}
		rc.ClientKeyPem = string(data)
	}
	if msg := rc.Validate(); msg != "" {
		return rc, fmt.Errorf("%s", msg)
	}
	return rc, nil
}

var configTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "列出支持的配置中心类型",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, t := range ce.GetSupportedTypes() {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-8s %s\n", t["value"], t["label"])
		}
		return nil
	},
}

var configServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "列出项目内可导出的服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := ce.GetServiceList(cfgPath)
		if err != nil {
			return err
		}
		for _, svc := range services {
			fmt.Fprintf(cmd.OutOrStdout(), "  %-16s %d 个配置文件\n", svc.Name, len(svc.ConfigFiles))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\033[32mSUCCESS: %d 个服务\033[m\n", len(services))
		return nil
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出服务配置到配置中心",
	RunE: func(cmd *cobra.Command, args []string) error {
		rc, err := buildRemoteConfig(cmd)
		if err != nil {
			return err
		}

		root, err := filepath.Abs(cfgPath)
		if err != nil {
			return err
		}

		if cfgDryRun {
			services, err := ce.GetServiceList(root)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\033[36m[DRY-RUN] 将导出以下服务到 %s (%s):\033[m\n", rc.Endpoint, rc.Type)
			for _, svc := range services {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%d 个配置文件)\n", svc.Name, len(svc.ConfigFiles))
			}
			return nil
		}

		if cfgService != "" {
			if err := ce.ExportOne(&rc, root, cfgService); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\033[32mSUCCESS: 服务 %s 已导出到 %s\033[m\n", cfgService, rc.Endpoint)
			return nil
		}

		if err := ce.ExportAll(&rc, root); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\033[32mSUCCESS: 全部服务已导出到 %s\033[m\n", rc.Endpoint)
		return nil
	},
}

func init() {
	configExportCmd.Flags().StringVar(&cfgType, "type", "", "配置中心类型: consul | etcd | nacos（必填）")
	configExportCmd.Flags().StringVar(&cfgEndpoint, "endpoint", "", "配置中心地址，如 http://localhost:8500（必填）")
	configExportCmd.Flags().StringVar(&cfgProject, "project", "", "项目名（配置 key 前缀，必填）")
	configExportCmd.Flags().StringVar(&cfgGroup, "group", "", "Nacos 分组")
	configExportCmd.Flags().StringVar(&cfgEnv, "env", "", "Nacos 环境")
	configExportCmd.Flags().StringVar(&cfgNamespace, "namespace-id", "", "Nacos 命名空间")
	configExportCmd.Flags().StringVar(&cfgUsername, "username", "", "Etcd/Nacos 用户名(服务端开启认证时使用)")
	configExportCmd.Flags().StringVar(&cfgPassword, "password", "", "Etcd/Nacos 密码(服务端开启认证时使用)")
	configExportCmd.Flags().StringVar(&cfgCaCert, "ca-cert", "", "Etcd TLS:CA 证书 PEM 文件路径")
	configExportCmd.Flags().StringVar(&cfgClientCer, "client-cert", "", "Etcd TLS:客户端证书 PEM 文件路径")
	configExportCmd.Flags().StringVar(&cfgClientKey, "client-key", "", "Etcd TLS:客户端私钥 PEM 文件路径")
	configExportCmd.Flags().StringVar(&cfgService, "service", "", "只导出指定服务（缺省全部）")
	configExportCmd.Flags().StringVar(&cfgPath, "path", ".", "项目根目录")
	configExportCmd.Flags().BoolVar(&cfgDryRun, "dry-run", false, "只列出将导出的服务，不实际写入")

	configServicesCmd.Flags().StringVar(&cfgPath, "path", ".", "项目根目录")
	CmdConfig.AddCommand(configTypesCmd, configServicesCmd, configExportCmd)
}
