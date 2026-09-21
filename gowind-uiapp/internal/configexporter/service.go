package configexporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/redirect"
	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/svcname"
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

	// 凭据(可选)。Etcd: gRPC 用户名/密码;Nacos: 登录接口换取 accessToken。
	Username string `json:"username"`
	Password string `json:"password"`

	// Etcd TLS(可选,PEM 文本): 自定义 CA / 客户端证书 / 客户端私钥。
	CaCertPem     string `json:"caCertPem"`
	ClientCertPem string `json:"clientCertPem"`
	ClientKeyPem  string `json:"clientKeyPem"`
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
func ExportAll(rc *RemoteConfig, projectRoot string) error {
	services, err := GetServiceList(projectRoot)
	if err != nil {
		return err
	}

	for _, svc := range services {
		if err := ExportOne(rc, projectRoot, svc.Name); err != nil {
			return fmt.Errorf("导出服务 %s 失败: %w", svc.Name, err)
		}
	}

	return nil
}

// ExportOne 导出单个服务的配置到远程配置中心
func ExportOne(rc *RemoteConfig, projectRoot string, serviceName string) error {
	// serviceName 也是前端传进来的。它可以逃到 app/ 之外,把项目外的任意
	// .yaml/.json/.txt 读出来发到远端配置中心——这条路径上是"读+外发",
	// 比写更难察觉。ExportAll 用磁盘目录名枚举,不经过这里。
	if err := svcname.Validate(serviceName); err != nil {
		return err
	}

	// 使用内建实现直接通过 HTTP API 写入配置中心
	return exportDirect(rc, projectRoot, serviceName)
}

// exportDirect 直接通过 SDK 写入配置中心（简化实现）
func exportDirect(rc *RemoteConfig, projectRoot string, serviceName string) error {
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
	switch rc.Type {
	case Consul:
		return writeConsul(rc.Endpoint, rc.ProjectName, serviceName, mergedContent)
	case Etcd:
		return writeEtcd(rc, serviceName, mergedContent)
	case Nacos:
		return writeNacos(rc, serviceName, mergedContent)
	default:
		return fmt.Errorf("不支持的配置中心类型: %s", rc.Type)
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

// httpClient 带超时的共享 HTTP 客户端:配置中心不可达或挂起时请求会在
// 限时内失败,而不是永久占住 Wails 绑定 goroutine。
//
// 重定向策略见 redirect.SameOrigin:这些请求的正文就是服务配置本身,还可能挂着
// Nacos 的 accessToken,不能跟着一个 3xx 送给别的主机。
var httpClient = &http.Client{
	Timeout:       30 * time.Second,
	CheckRedirect: redirect.SameOrigin,
}

// endpointScheme 拆出 endpoint 的协议与 "host[:port][/前缀]"。
//
// 不带协议时按 http 处理,与这几个客户端历史上的行为一致。但显式写了 https://
// 就必须真的用 https:旧实现把前缀盲剔掉再硬拼 http://,于是一个只监听 TLS 的
// 配置中心完全连不上(实测:明文请求打到 TLS 端口拿 400),用户也没有别的开关
// 能要求加密。http/https 之外的协议直接拒绝——把它们当主机名拼进 URL 只会得到
// "http://gopher://x/..." 这种谁也发不到的地址。
func endpointScheme(endpoint string) (scheme, hostPort string, err error) {
	if i := strings.Index(endpoint, "://"); i >= 0 {
		s := strings.ToLower(endpoint[:i])
		if s != "http" && s != "https" {
			return "", "", fmt.Errorf("不支持的 endpoint 协议 %q(只支持 http/https)", s)
		}
		return s, endpoint[i+3:], nil
	}
	return "http", endpoint, nil
}

// requireTLSForCredentials 拒绝在非回环主机上用明文通道送凭据。
//
// 只拦"带凭据 + 明文 + 远端"这一组合:本地 httptest/端口转发场景保持不变,
// 否则每台开发机都得先自签证书才能试配置中心。
func requireTLSForCredentials(scheme, hostPort, what string) error {
	if scheme == "https" || isLoopback(hostPort) {
		return nil
	}
	return fmt.Errorf("%s 需要用户名口令,但 endpoint %q 走的是明文 http:请改用 https://,或通过本地端口转发", what, hostPort)
}

// isLoopback 判断 "host[:port]" 是否是机器自己。
func isLoopback(hostPort string) bool {
	host := hostPort
	if h, _, splitErr := net.SplitHostPort(hostPort); splitErr == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// writeConsul 写入配置到 Consul
func writeConsul(endpoint, project, app, content string) error {
	scheme, hostPort, err := endpointScheme(endpoint)
	if err != nil {
		return fmt.Errorf("Consul endpoint 无效: %w", err)
	}

	// key 分段转义、保留 "/" 分隔符,与 etcd 路径形成的键形态一致;
	// 整段 PathEscape 会把 "/" 编成 %2F,读侧按斜杠路径无法取回。
	key := strings.Join([]string{
		url.PathEscape(project), url.PathEscape(app), "service", "config",
	}, "/")
	consulURL := fmt.Sprintf("%s://%s/v1/kv/%s", scheme, strings.TrimSuffix(hostPort, "/"), key)

	req, err := http.NewRequest("PUT", consulURL, strings.NewReader(content))
	if err != nil {
		return fmt.Errorf("创建 Consul 请求失败: %w", err)
	}

	resp, err := httpClient.Do(req)
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

// writeEtcd 写入配置到 Etcd (v3 gRPC 协议)。
// key 约定与 Consul 一致: <project>/<app>/service/config
// rc.Username/rc.Password 提供 gRPC 认证;rc.*Pem 提供自定义 CA 与客户端
// 证书(见 buildTLSConfig)。
func writeEtcd(rc *RemoteConfig, app, content string) error {
	endpoints, secure, err := normalizeEtcdEndpoints(rc.Endpoint)
	if err != nil {
		return fmt.Errorf("Etcd endpoint 无效: %w", err)
	}
	if len(endpoints) == 0 {
		return fmt.Errorf("Etcd endpoint 为空")
	}

	tlsCfg, err := buildTLSConfig(rc.CaCertPem, rc.ClientCertPem, rc.ClientKeyPem)
	if err != nil {
		return fmt.Errorf("Etcd TLS 配置无效: %w", err)
	}
	if secure && tlsCfg == nil {
		// 写了 https:// 却没给 CA:用系统根证书池,而不是退回明文。
		tlsCfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsCfg,
	}
	if rc.Username != "" {
		cfg.Username = rc.Username
		cfg.Password = rc.Password
		if tlsCfg == nil {
			for _, ep := range endpoints {
				if err := requireTLSForCredentials("http", ep, "Etcd 认证"); err != nil {
					return err
				}
			}
		}
	}

	cli, err := clientv3.New(cfg)
	if err != nil {
		return fmt.Errorf("连接 Etcd 失败: %w", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := fmt.Sprintf("%s/%s/service/config", rc.ProjectName, app)
	if _, err := cli.Put(ctx, key, content); err != nil {
		return fmt.Errorf("写入 Etcd 失败: %w", err)
	}
	return nil
}

// buildTLSConfig 由可选的 PEM 文本构建 etcd 客户端 TLS 配置:
//   - ca 非空: 解析为根证书池(自定义 CA)
//   - cert/key 非空: 解析为客户端证书对(mTLS)
//   - 全部为空: 返回 (nil, nil),表示使用系统默认
//
// 任何一段 PEM 无法解析即报错;cert 与 key 必须成对出现。
func buildTLSConfig(ca, cert, key string) (*tls.Config, error) {
	if ca == "" && cert == "" && key == "" {
		return nil, nil
	}
	if (cert == "") != (key == "") {
		return nil, fmt.Errorf("客户端证书与私钥必须成对提供")
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return nil, fmt.Errorf("CA 证书 PEM 解析失败")
		}
		cfg.RootCAs = pool
	}
	if cert != "" {
		pair, err := tls.X509KeyPair([]byte(cert), []byte(key))
		if err != nil {
			return nil, fmt.Errorf("客户端证书对解析失败: %w", err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}
	return cfg, nil
}

// normalizeEtcdEndpoints 归一化 endpoint 为 etcd 要求的 "host:port" 列表,并报告
// 这批节点是否要求 TLS:支持逗号分隔多节点,容忍 http(s):// 前缀与尾部斜杠。
//
// 混用 http 与 https 会拒绝:clientv3 只有一份 tls.Config,一次连接里两种期望
// 没法同时满足,静默按明文连过去等于替用户挑了协议。
func normalizeEtcdEndpoints(endpoint string) (hostPorts []string, secure bool, err error) {
	var sawPlain, sawTLS bool
	for _, part := range strings.Split(endpoint, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		scheme, hostPort, e := endpointScheme(part)
		if e != nil {
			return nil, false, e
		}
		hostPort = strings.TrimSuffix(hostPort, "/")
		if hostPort == "" {
			continue
		}
		if scheme == "https" {
			sawTLS = true
		} else {
			sawPlain = true
		}
		hostPorts = append(hostPorts, hostPort)
	}
	if sawPlain && sawTLS {
		return nil, false, fmt.Errorf("Etcd endpoint 不能混用 http 与 https: %q", endpoint)
	}
	return hostPorts, sawTLS, nil
}

// writeNacos 写入配置到 Nacos。
// rc.Username/rc.Password 非空时先走登录接口换取 accessToken,并随发布
// 请求一并提交(服务端关闭鉴权时凭据可留空,行为与之前一致)。
func writeNacos(rc *RemoteConfig, app, content string) error {
	group := rc.Group
	env := rc.Env
	namespaceId := rc.NamespaceId
	if group == "" {
		group = "DEFAULT_GROUP"
	}
	if env == "" {
		env = "dev"
	}
	if namespaceId == "" {
		namespaceId = "public"
	}

	scheme, hostPort, err := endpointScheme(rc.Endpoint)
	if err != nil {
		return err
	}
	if rc.Username != "" && rc.Password != "" {
		if err := requireTLSForCredentials(scheme, hostPort, "Nacos"); err != nil {
			return err
		}
	}

	dataId := fmt.Sprintf("%s-%s-service-%s.yaml", rc.ProjectName, app, env)
	nacosURL := fmt.Sprintf("%s://%s/nacos/v1/cs/configs", scheme, strings.TrimSuffix(hostPort, "/"))

	form := url.Values{}
	form.Set("dataId", dataId)
	form.Set("group", group)
	form.Set("tenant", namespaceId)
	form.Set("content", content)

	if rc.Username != "" && rc.Password != "" {
		token, err := nacosLogin(rc.Endpoint, rc.Username, rc.Password)
		if err != nil {
			return err
		}
		form.Set("accessToken", token)
	}

	resp, err := httpClient.PostForm(nacosURL, form)
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

// nacosLogin 走 Nacos 登录接口换取 accessToken。
func nacosLogin(endpoint, username, password string) (string, error) {
	scheme, hostPort, err := endpointScheme(endpoint)
	if err != nil {
		return "", err
	}
	// 这个函数是口令真正出门的地方,所以这里再判一次,不依赖调用方已经判过。
	if err := requireTLSForCredentials(scheme, hostPort, "Nacos 登录"); err != nil {
		return "", err
	}
	loginURL := fmt.Sprintf("%s://%s/nacos/v1/auth/login", scheme, strings.TrimSuffix(hostPort, "/"))

	resp, err := httpClient.PostForm(loginURL, url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return "", fmt.Errorf("Nacos 登录请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Nacos 登录失败 (%d)", resp.StatusCode)
	}

	var out struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("Nacos 登录响应无效")
	}
	return out.AccessToken, nil
}
