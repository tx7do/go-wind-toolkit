package ai

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/frontendgen"
)

// openAPISpecClient 带超时的拉取客户端:http.Get 用的是无超时的 DefaultClient,
// 一个只接受连接不回数据的地址能把调用方永久挂住。
var openAPISpecClient = &http.Client{Timeout: 30 * time.Second}

// maxOpenAPISpecBytes 响应正文上限:OpenAPI 文档正常在几十到几百 KB,
// 不设上限就等于让远端决定内存占用。
const maxOpenAPISpecBytes = 10 << 20

// LoadOpenAPISpec 加载 OpenAPI 3.0 规格(本地文件路径或 http(s) URL)。
// CLI(frontend 组)与 MCP(generate_frontend)共用同一实现,
// 保证两条入口的拉取语义(超时/大小上限)一致。
func LoadOpenAPISpec(ref string) (*frontendgen.Spec, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, fmt.Errorf("必须指定 OpenAPI 规范(文件路径或 http(s) URL)")
	}

	var data []byte
	if strings.HasPrefix(strings.ToLower(ref), "http://") || strings.HasPrefix(strings.ToLower(ref), "https://") {
		resp, err := openAPISpecClient.Get(ref)
		if err != nil {
			return nil, fmt.Errorf("拉取 OpenAPI 失败: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("拉取 OpenAPI 失败: HTTP %d", resp.StatusCode)
		}
		// 多读 1 字节用来分辨"正好等于上限"和"被截断"。
		data, err = io.ReadAll(io.LimitReader(resp.Body, maxOpenAPISpecBytes+1))
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 响应失败: %w", err)
		}
		if len(data) > maxOpenAPISpecBytes {
			return nil, fmt.Errorf("OpenAPI 文档超过 %d MiB 上限,拒绝读入内存", maxOpenAPISpecBytes>>20)
		}
	} else {
		var err error
		data, err = os.ReadFile(ref)
		if err != nil {
			return nil, fmt.Errorf("读取 OpenAPI 文件失败: %w", err)
		}
	}

	return frontendgen.ParseOpenAPIYAML(data)
}
