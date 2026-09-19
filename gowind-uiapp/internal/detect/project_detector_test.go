package detect

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestProjectDetector_Detect_PathSeparator 回归:Detect 不能把正斜杠无条件替换成
// 反斜杠。macOS/Linux 上反斜杠是合法文件名字符,一旦被注入 Root,后端代码生成会
// 把产物写进错误的相对目录。
func TestProjectDetector_Detect_PathSeparator(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}
	projectRoot := wd
	for {
		if _, err = os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("未找到 go.mod 文件")
		}
		projectRoot = parent
	}

	// 用正斜杠形式传入,模拟前端/GUI 传来的路径。
	forward := filepath.ToSlash(projectRoot)
	info, err := NewProjectDetector().Detect(forward)
	if err != nil {
		t.Fatalf("检测项目失败: %v", err)
	}

	want := filepath.Clean(projectRoot)
	if info.Root != want {
		t.Fatalf("Root 未按平台归一化: got %q, want %q", info.Root, want)
	}
	if runtime.GOOS != "windows" && strings.Contains(info.Root, `\`) {
		t.Fatalf("非 Windows 平台的 Root 不应含反斜杠, got %q", info.Root)
	}
}

func TestProjectDetector_Detect(t *testing.T) {
	// 获取当前工作目录作为测试项目路径
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}
	// 向上查找到项目根目录(包含 go.mod 的目录)
	projectRoot := wd
	for {
		if _, err = os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("未找到 go.mod 文件")
		}
		projectRoot = parent
	}

	detector := NewProjectDetector()
	info, err := detector.Detect(projectRoot)
	if err != nil {
		t.Fatalf("检测项目失败: %v", err)
	}

	t.Logf("项目根目录: %s", info.Root)
	t.Logf("模块路径: %s", info.ModPath)
	t.Logf("Go 版本要求: %s", info.GoVersion)
	t.Logf("是否为主模块: %v", info.Main)
	t.Logf("模块版本: %s", info.Version)
	if info.Replace != nil {
		t.Logf("替换信息: %s => %s@%s", info.ModPath, info.Replace.Path, info.Replace.Version)
	}
	t.Logf("依赖列表:")
	for _, dep := range info.Dependencies {
		t.Logf(" - %s@%s", dep.Path, dep.Version)
	}
	t.Logf("检测到的服务: %v", info.Services)
}

func TestProjectDetector_collectServices(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir, err := os.MkdirTemp("", "project_detector_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 app 目录和一些子目录
	appDir := filepath.Join(tmpDir, "app")
	if err := os.Mkdir(appDir, 0755); err != nil {
		t.Fatalf("创建 app 目录失败: %v", err)
	}

	services := []string{"user", "order", "payment"}
	for _, svc := range services {
		svcDir := filepath.Join(appDir, svc)
		if err := os.Mkdir(svcDir, 0755); err != nil {
			t.Fatalf("创建服务目录失败: %v", err)
		}
	}

	// 创建一个文件（不应该被统计）
	testFile := filepath.Join(appDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试 collectServices
	detector := NewProjectDetector()
	result, err := detector.collectServices(tmpDir)
	if err != nil {
		t.Fatalf("collectServices 失败: %v", err)
	}

	if len(result) != len(services) {
		t.Errorf("服务数量不匹配: got %d, want %d", len(result), len(services))
	}

	// 验证所有服务都被检测到
	serviceMap := make(map[string]bool)
	for _, svc := range result {
		serviceMap[svc] = true
	}

	for _, expectedSvc := range services {
		if !serviceMap[expectedSvc] {
			t.Errorf("服务 %s 未被检测到", expectedSvc)
		}
	}

	t.Logf("检测到的服务: %v", result)
}

func TestProjectDetector_collectServices_NoAppDir(t *testing.T) {
	// 创建临时测试目录（不包含 app 目录）
	tmpDir, err := os.MkdirTemp("", "project_detector_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detector := NewProjectDetector()
	_, err = detector.collectServices(tmpDir)
	if err == nil {
		t.Error("期望返回错误，但没有错误")
	}
}

func TestProjectDetector_collectServices_EmptyAppDir(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir, err := os.MkdirTemp("", "project_detector_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建空的 app 目录
	appDir := filepath.Join(tmpDir, "app")
	if err := os.Mkdir(appDir, 0755); err != nil {
		t.Fatalf("创建 app 目录失败: %v", err)
	}

	detector := NewProjectDetector()
	result, err := detector.collectServices(tmpDir)
	if err != nil {
		t.Fatalf("collectServices 失败: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("期望空服务列表，但得到: %v", result)
	}
}

func TestNewProjectDetector(t *testing.T) {
	detector := NewProjectDetector()
	if detector == nil {
		t.Fatal("NewProjectDetector 返回 nil")
	}
}
