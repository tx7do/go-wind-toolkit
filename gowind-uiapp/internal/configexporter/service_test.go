package configexporter

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/redirect"
)

func TestNormalizeEtcdEndpoints(t *testing.T) {
	tests := []struct {
		in      string
		want    []string
		wantTLS bool
		wantErr string
	}{
		{in: "localhost:2379", want: []string{"localhost:2379"}},
		{in: "http://localhost:2379", want: []string{"localhost:2379"}},
		{in: "https://etcd.example.com:2379/", want: []string{"etcd.example.com:2379"}, wantTLS: true},
		{in: "http://n1:2379, n2:2379 ,http://n3:2379", want: []string{"n1:2379", "n2:2379", "n3:2379"}},
		{in: "", want: nil},
		{in: " , ,", want: nil},
		{in: "https://n1:2379,n2:2379", wantErr: "不能混用"},
		{in: "gopher://n1:2379", wantErr: "不支持的 endpoint 协议"},
	}
	for _, tt := range tests {
		got, secure, err := normalizeEtcdEndpoints(tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("normalizeEtcdEndpoints(%q) err = %v, want 含 %q", tt.in, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeEtcdEndpoints(%q) 意外报错: %v", tt.in, err)
		}
		if secure != tt.wantTLS {
			t.Errorf("normalizeEtcdEndpoints(%q) secure = %v, want %v", tt.in, secure, tt.wantTLS)
		}
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("normalizeEtcdEndpoints(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestEndpointScheme(t *testing.T) {
	tests := []struct {
		in       string
		wantSch  string
		wantHost string
		wantErr  string
	}{
		{in: "consul:8500", wantSch: "http", wantHost: "consul:8500"},
		{in: "http://consul:8500", wantSch: "http", wantHost: "consul:8500"},
		{in: "HTTPS://consul:8500", wantSch: "https", wantHost: "consul:8500"},
		{in: "https://gw.example.com/consul", wantSch: "https", wantHost: "gw.example.com/consul"},
		{in: "ftp://consul:8500", wantErr: "不支持的 endpoint 协议"},
	}
	for _, tt := range tests {
		scheme, hostPort, err := endpointScheme(tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("endpointScheme(%q) err = %v, want 含 %q", tt.in, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("endpointScheme(%q) 意外报错: %v", tt.in, err)
			continue
		}
		if scheme != tt.wantSch || hostPort != tt.wantHost {
			t.Errorf("endpointScheme(%q) = (%q,%q), want (%q,%q)", tt.in, scheme, hostPort, tt.wantSch, tt.wantHost)
		}
	}
}

func TestIsLoopback(t *testing.T) {
	for _, in := range []string{"localhost:2379", "127.0.0.1:8500", "127.5.6.7:8500", "[::1]:2379", "localhost"} {
		if !isLoopback(in) {
			t.Errorf("isLoopback(%q) = false, 本机地址", in)
		}
	}
	for _, in := range []string{"consul.internal:8500", "10.0.0.5:2379", "example.com:443"} {
		if isLoopback(in) {
			t.Errorf("isLoopback(%q) = true, 这不是本机", in)
		}
	}
}

// genTestPEM 在测试内自签发一对 CA + 叶证书,返回三段 PEM。
func genTestPEM(t *testing.T) (caPEM, certPEM, keyPEM string) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		NotAfter:              time.Now().Add(time.Hour),
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, caKey.Public(), caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-leaf"},
		NotAfter:     time.Now().Add(time.Hour),
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, leafKey.Public(), caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		t.Fatal(err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

// buildTLSConfig 行为契约:全空→nil;CA+证书对→RootCAs 与 Certificates 均就绪;
// 垃圾 PEM / 单边证书对→错误。
func TestBuildTLSConfig(t *testing.T) {
	if cfg, err := buildTLSConfig("", "", ""); cfg != nil || err != nil {
		t.Fatalf("empty input: cfg=%v err=%v, want nil/nil", cfg, err)
	}

	ca, cert, key := genTestPEM(t)
	cfg, err := buildTLSConfig(ca, cert, key)
	if err != nil {
		t.Fatalf("valid material: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil || len(cfg.Certificates) != 1 {
		t.Fatalf("valid material: cfg=%v, want RootCAs + 1 client certificate", cfg)
	}

	if _, err = buildTLSConfig("not a pem", "", ""); err == nil {
		t.Fatal("garbage CA PEM must error")
	}
	if _, err = buildTLSConfig("", cert, ""); err == nil {
		t.Fatal("cert without key must error")
	}
	if _, err = buildTLSConfig("", "", key); err == nil {
		t.Fatal("key without cert must error")
	}
}

// Nacos 登录流程:带凭据→先登录并把 accessToken 附到发布请求;
// 无凭据→不登录、不带 token(与免认证服务端行为一致)。
func TestWriteNacosAuthFlow(t *testing.T) {
	var loginCalled bool
	var publishToken string
	var publishCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nacos/v1/auth/login":
			loginCalled = true
			_ = r.ParseForm()
			if r.Form.Get("username") != "u" || r.Form.Get("password") != "p" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"accessToken": "tok-123"})
		case "/nacos/v1/cs/configs":
			publishCalls++
			_ = r.ParseForm()
			publishToken = r.Form.Get("accessToken")
			_, _ = w.Write([]byte("true"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ep := srv.Listener.Addr().String()

	rc := &RemoteConfig{
		Type:        Nacos,
		Endpoint:    ep,
		ProjectName: "p",
		Username:    "u",
		Password:    "p",
	}
	if err := writeNacos(rc, "app", "content"); err != nil {
		t.Fatalf("with credentials: %v", err)
	}
	if !loginCalled {
		t.Fatal("login endpoint must be called when credentials are set")
	}
	if publishToken != "tok-123" {
		t.Fatalf("publish accessToken = %q, want tok-123", publishToken)
	}

	loginCalled = false
	publishToken = "<unset>"
	rc2 := &RemoteConfig{Type: Nacos, Endpoint: ep, ProjectName: "p"}
	if err := writeNacos(rc2, "app", "content"); err != nil {
		t.Fatalf("without credentials: %v", err)
	}
	if loginCalled {
		t.Fatal("login must not be called without credentials")
	}
	if publishToken != "" {
		t.Fatalf("publish accessToken = %q, want empty", publishToken)
	}
}

// TestWriteNacos_RejectsCredentialsOverPlaintext 远端 endpoint + 明文 http + 带口令
// 必须拒发。这里刻意用一个不存在的域名:如果实现先拨号再报错,错误里会是 DNS 失败
// 而不是我们的拒绝文案,断言就能分辨两者。
func TestWriteNacos_RejectsCredentialsOverPlaintext(t *testing.T) {
	rc := &RemoteConfig{
		Type:        Nacos,
		Endpoint:    "nacos.example.com:8848",
		ProjectName: "p",
		Username:    "u",
		Password:    "s3cret",
	}
	err := writeNacos(rc, "app", "content")
	if err == nil {
		t.Fatal("明文 http 下带凭据必须被拒绝")
	}
	if !strings.Contains(err.Error(), "明文 http") {
		t.Errorf("错误应当是协议拒绝,实际: %v", err)
	}
	if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "dial") {
		t.Errorf("拒绝要发生在拨号之前,实际: %v", err)
	}
}

// TestWriteConsul_HonorsEndpointScheme 三条腿合起来证明协议是从 endpoint 读出来的:
// 无 scheme 与显式 http 走明文;显式 https 打到 TLS 监听器上能成功——旧实现硬拼
// http://,同一个请求会以 "HTTP 400 Client sent an HTTP request to an HTTPS server"
// 失败(实测)。
func TestWriteConsul_HonorsEndpointScheme(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		_, _ = w.Write([]byte("true"))
	})

	plain := httptest.NewServer(handler)
	defer plain.Close()
	tlsSrv := httptest.NewTLSServer(handler)
	defer tlsSrv.Close()

	// TLS 腿需要信任测试自签证书;沿用包内客户端的重定向策略,别把被测行为换掉。
	orig := httpClient
	defer func() { httpClient = orig }()
	httpClient = &http.Client{
		Timeout:       30 * time.Second,
		Transport:     tlsSrv.Client().Transport,
		CheckRedirect: redirect.SameOrigin,
	}

	plainAddr := plain.Listener.Addr().String()
	tlsAddr := tlsSrv.Listener.Addr().String()

	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{name: "无 scheme 仍走明文", endpoint: plainAddr},
		{name: "显式 http", endpoint: "http://" + plainAddr},
		{name: "显式 https 打 TLS 服务", endpoint: "https://" + tlsAddr},
		{name: "显式 https 打明文服务必须失败", endpoint: "https://" + plainAddr, wantErr: true},
		{name: "未知协议", endpoint: "ftp://" + plainAddr, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotMethod, gotBody = "", "", ""
			err := writeConsul(tt.endpoint, "proj", "app", "cfg-body")
			if tt.wantErr {
				if err == nil {
					t.Fatal("这一条本该失败,却报告成功")
				}
				return
			}
			if err != nil {
				t.Fatalf("writeConsul(%q) = %v", tt.endpoint, err)
			}
			if gotMethod != "PUT" {
				t.Errorf("method = %q, want PUT", gotMethod)
			}
			if gotPath != "/v1/kv/proj/app/service/config" {
				t.Errorf("path = %q", gotPath)
			}
			if gotBody != "cfg-body" {
				t.Errorf("body = %q", gotBody)
			}
		})
	}
}

// TestWriteConsul_CrossOriginRedirectDoesNotLeak 配置中心回一个指向外主机的 307 时,
// 配置正文不得跟着发出去。
//
// 用 307 而不是 302:Go 的客户端会把 302 的 PUT 降级成无正文的 GET,那样即使不设
// 策略也测不出正文外泄;307 保留方法与正文,才是要防的那条路径。
func TestWriteConsul_CrossOriginRedirectDoesNotLeak(t *testing.T) {
	var leaked int
	var leakedBody string
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked++
		body, _ := io.ReadAll(r.Body)
		leakedBody = string(body)
		_, _ = w.Write([]byte("ok"))
	}))
	defer collector.Close()

	consul := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, collector.URL+"/kv", http.StatusTemporaryRedirect)
	}))
	defer consul.Close()

	if err := writeConsul(consul.Listener.Addr().String(), "proj", "app", "SECRET-CFG"); err == nil {
		t.Fatal("跨主机重定向应当让写入失败")
	}
	if leaked != 0 {
		t.Fatalf("配置正文被重定向发到了另一台主机 %d 次,body=%q", leaked, leakedBody)
	}
}
