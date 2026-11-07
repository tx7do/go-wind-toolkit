package redirect

import (
	"net/http"
	"strings"
	"testing"
)

// TestSameOrigin 直接喂构造好的请求给策略函数:是否放行是纯判断,不需要真跑一次跳转。
func TestSameOrigin(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "同源换路径", from: "http://a:8500/v1/kv/x", to: "http://a:8500/v1/kv/y"},
		{name: "换主机", from: "http://a:8500/v1/kv/x", to: "http://evil:9999/collect", wantErr: true},
		{name: "换端口", from: "http://a:8500/v1/kv/x", to: "http://a:9999/collect", wantErr: true},
		{name: "https 降级", from: "https://a:8500/v1/kv/x", to: "http://a:8500/v1/kv/y", wantErr: true},
		{name: "http 升级", from: "http://a:8500/v1/kv/x", to: "https://a:8500/v1/kv/y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := http.NewRequest("PUT", tt.from, strings.NewReader("body"))
			if err != nil {
				t.Fatal(err)
			}
			next, err := http.NewRequest("PUT", tt.to, strings.NewReader("body"))
			if err != nil {
				t.Fatal(err)
			}
			if err := SameOrigin(next, []*http.Request{first}); (err != nil) != tt.wantErr {
				t.Errorf("SameOrigin(%s -> %s) err = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

// TestSameOrigin_FirstHopHasNoOrigin 没有 via 时没有"原始来源"可比,不该 panic。
func TestSameOrigin_FirstHopHasNoOrigin(t *testing.T) {
	req, err := http.NewRequest("GET", "http://a:8500/v1/kv/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := SameOrigin(req, nil); err != nil {
		t.Errorf("via 为空时应放行: %v", err)
	}
}
