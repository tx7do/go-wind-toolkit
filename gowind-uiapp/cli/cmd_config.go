package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	ce "github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "远程配置中心导出（Consul / Nacos）",
}

var configTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "列出支持的配置中心类型",
	Run: func(cmd *cobra.Command, args []string) {
		emit(ce.GetSupportedTypes())
	},
}

var configServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "扫描项目中有配置文件的服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := ce.GetServiceList(flagString(cmd, "path", "."))
		if err != nil {
			return err
		}
		emit(services)
		return nil
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出服务配置到配置中心",
	Long: `将 app/<服务>/service/configs 下的配置文件导出到远程配置中心。
指定 --service 时只导出该服务，否则导出全部服务。
Etcd endpoint 支持 host:port 或 http(s)://host:port，逗号分隔多节点。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		typeName := flagString(cmd, "type", "")
		endpoint := flagString(cmd, "endpoint", "")
		projectName := flagString(cmd, "project", "")
		if typeName == "" || endpoint == "" || projectName == "" {
			checkErr(fmt.Errorf("必须指定 --type、--endpoint、--project"))
		}

		rc := ce.RemoteConfig{
			Type:        ce.ConfigType(typeName),
			Endpoint:    endpoint,
			ProjectName: projectName,
			Group:       flagString(cmd, "group", ""),
			Env:         flagString(cmd, "env", ""),
			NamespaceId: flagString(cmd, "namespace-id", ""),
			Username:    flagString(cmd, "username", ""),
			Password:    flagString(cmd, "password", ""),
		}
		// TLS 材料(仅 Etcd):从文件读取 PEM。
		if p := flagString(cmd, "ca-cert", ""); p != "" {
			data, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("读取 ca-cert 失败: %w", err)
			}
			rc.CaCertPem = string(data)
		}
		if p := flagString(cmd, "client-cert", ""); p != "" {
			data, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("读取 client-cert 失败: %w", err)
			}
			rc.ClientCertPem = string(data)
		}
		if p := flagString(cmd, "client-key", ""); p != "" {
			data, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("读取 client-key 失败: %w", err)
			}
			rc.ClientKeyPem = string(data)
		}
		if msg := rc.Validate(); msg != "" {
			checkErr(fmt.Errorf("%s", msg))
		}

		projectRoot := flagString(cmd, "path", ".")
		serviceName := flagString(cmd, "service", "")

		if boolFlag(cmd, "dry-run") {
			services, err := ce.GetServiceList(projectRoot)
			if err != nil {
				return err
			}
			emit(map[string]any{
				"dryRun":   true,
				"type":     typeName,
				"endpoint": endpoint,
				"project":  projectName,
				"services": services,
			})
			return nil
		}

		if serviceName != "" {
			if err := ce.ExportOne(&rc, projectRoot, serviceName); err != nil {
				return err
			}
			emit(map[string]any{"success": true, "service": serviceName})
			return nil
		}

		if err := ce.ExportAll(&rc, projectRoot); err != nil {
			return err
		}
		emit(map[string]any{"success": true, "all": true})
		return nil
	},
}

func init() {
	configServicesCmd.Flags().String("path", ".", "项目根目录")

	configExportCmd.Flags().String("type", "", "配置中心类型: consul | nacos（必填）")
	configExportCmd.Flags().String("endpoint", "", "配置中心地址，如 http://localhost:8500（必填）")
	configExportCmd.Flags().String("project", "", "项目名（配置 key 前缀，必填）")
	configExportCmd.Flags().String("group", "", "Nacos 分组")
	configExportCmd.Flags().String("env", "", "Nacos 环境")
	configExportCmd.Flags().String("namespace-id", "", "Nacos 命名空间")
	configExportCmd.Flags().String("username", "", "Etcd/Nacos 用户名(服务端开启认证时使用)")
	configExportCmd.Flags().String("password", "", "Etcd/Nacos 密码(服务端开启认证时使用)")
	configExportCmd.Flags().String("ca-cert", "", "Etcd TLS:CA 证书 PEM 文件路径")
	configExportCmd.Flags().String("client-cert", "", "Etcd TLS:客户端证书 PEM 文件路径")
	configExportCmd.Flags().String("client-key", "", "Etcd TLS:客户端私钥 PEM 文件路径")
	configExportCmd.Flags().String("service", "", "只导出指定服务（缺省全部）")
	configExportCmd.Flags().String("path", ".", "项目根目录")
	configExportCmd.Flags().Bool("dry-run", false, "只列出将导出的服务，不实际写入")

	configCmd.AddCommand(configTypesCmd, configServicesCmd, configExportCmd)
}
