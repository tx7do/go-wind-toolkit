package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestLoadOpenAPISpec_RejectsOversizedResponse 远端回多大的正文,CLI 就只读多大。
// 断言的是"报上限错误",不是"读进来后解析失败"——后者说明上限没起作用。
func TestLoadOpenAPISpec_RejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 超过上限一点点就够,别真写 10MiB 以上。
		chunk := strings.Repeat("a", 64<<10)
		for i := 0; i <= maxOpenAPISpecBytes/(64<<10); i++ {
			if _, err := fmt.Fprint(w, chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	_, err := loadOpenAPISpec(srv.URL)
	if err == nil {
		t.Fatal("超限响应应当被拒绝")
	}
	if !strings.Contains(err.Error(), "上限") {
		t.Errorf("错误应当指向大小上限,实际: %v", err)
	}
}

// TestLoadOpenAPISpec_TimesOutOnStalledResponse 只接受连接不回数据的地址不能把
// CLI 永久挂住。这里把超时改短以免测试自己跑满 30 秒。
func TestLoadOpenAPISpec_TimesOutOnStalledResponse(t *testing.T) {
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	// 顺序要紧:httptest.Server.Close 会等在途请求上,所以必须先放开 handler 再关服务。
	// 注册顺序 = LIFO 执行顺序,srv.Close 先注册才能最后跑。
	defer srv.Close()
	defer close(release)

	orig := openAPISpecClient
	defer func() { openAPISpecClient = orig }()
	openAPISpecClient = &http.Client{Timeout: 300 * time.Millisecond}

	done := make(chan error, 1)
	go func() {
		_, err := loadOpenAPISpec(srv.URL)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("挂起的响应应当以超时失败")
		}
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
			t.Errorf("错误应当是超时,实际: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("超过 5 秒仍未返回:客户端没有超时上限")
	}
}
