// Package redirect 收放客户端的重定向策略。
package redirect

import (
	"fmt"
	"net/http"
)

// SameOrigin 只允许同源跳转,并拒绝协议降级。
//
// Go 的默认策略会跟着 3xx 走,并且只为 Authorization / Cookie / Proxy-Authorization
// 这几个头做跨域剥离:自定义的 x-api-key、api-key、x-goog-api-key 会照发,307/308
// 还会把方法与请求正文一并带给新主机。对本仓库这些客户端来说,正文可能是服务配置
// 或对话内容,头里可能是密钥——一跳就送出去了。
func SameOrigin(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	origin := via[0].URL
	if req.URL.Host != origin.Host {
		return fmt.Errorf("拒绝重定向到 %s://%s:与原始主机 %s 不同源,请求头与正文不随跳转外发", req.URL.Scheme, req.URL.Host, origin.Host)
	}
	if origin.Scheme == "https" && req.URL.Scheme != "https" {
		return fmt.Errorf("拒绝从 https 降级到 http:同一主机也不放行")
	}
	return nil
}
