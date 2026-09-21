package devtools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestServiceNameGates_RejectPathEscape 三个会用 serviceName 拼目录的入口:
// RunEntGenerate 把目录当 go 的工作目录和 ent 的参数,RunWire 在目录下改写
// wire_gen.go,AddService 直接按这个路径建目录写文件。
//
// 每一例都把"被逃到的目标"真的建出来,这样红的时候只能是被校验挡住,而不是
// 顺带靠 os.Stat 失败蒙对。
func TestServiceNameGates_RejectPathEscape(t *testing.T) {
	tests := []struct {
		name string
		// dirs 是逃出去之后需要存在的目录,相对临时根
		dirs []string
		// absent 是校验失效后才会冒出来的目录
		absent []string
		invoke func(root string) *CommandResult
	}{
		{
			name: "RunEntGenerate",
			dirs: []string{"evil/service/internal/data/ent/schema"},
			invoke: func(root string) *CommandResult {
				return RunEntGenerate(root, "../evil")
			},
		},
		{
			name: "RunWire",
			dirs: []string{"evil/service/cmd/server"},
			invoke: func(root string) *CommandResult {
				return RunWire(root, `..\evil`)
			},
		},
		{
			name:   "AddService",
			dirs:   []string{"app"},
			absent: []string{"evil"},
			invoke: func(root string) *CommandResult {
				return AddService(root, AddServiceOptions{ServiceName: "../evil"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, d := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", d, err)
				}
			}

			result := tt.invoke(root)
			if result.Success {
				t.Fatal("用 ../ 逃出 app/ 的服务名不能被判成成功")
			}
			if !strings.Contains(result.Error, "路径分隔符") {
				t.Errorf("应当因服务名校验而失败,实际错误: %q", result.Error)
			}
			for _, d := range tt.absent {
				if _, err := os.Stat(filepath.Join(root, d)); err == nil {
					t.Errorf("%s 越过了校验,在 app/ 外建出了 %s", tt.name, d)
				}
			}
		})
	}
}

// TestAddService_RejectsEmptyName 空名字原本由这里自己挡,现在由同一个校验挡,
// 错误文案变了但语义不变:不能建出 app//service。
func TestAddService_RejectsEmptyName(t *testing.T) {
	result := AddService(t.TempDir(), AddServiceOptions{})
	if result.Success {
		t.Fatal("空服务名不应成功")
	}
	if !strings.Contains(result.Error, "服务名为空") {
		t.Errorf("实际错误: %q", result.Error)
	}
}
