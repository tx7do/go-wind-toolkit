package generators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertProviderSetFunction(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "wire.go")

	// 初始文件内容
	initialContent := `package server

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	server.NewRestMiddleware,
	server.NewRestServer,

	server.NewGrpcMiddleware,
	server.NewGrpcServer,
)
`

	// 写入初始内容
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	g := NewGoGenerator()

	// 测试添加新函数
	t.Run("Add new function", func(t *testing.T) {
		err := g.UpsertProviderSetFunction(testFile, "server.NewRedisClient")
		if err != nil {
			t.Fatalf("UpsertProviderSetFunction failed: %v", err)
		}

		// 验证函数已添加
		exists, err := g.CheckProviderSetFunctionExists(testFile, "server.NewRedisClient")
		if err != nil {
			t.Fatalf("CheckProviderSetFunctionExists failed: %v", err)
		}
		if !exists {
			t.Error("expected server.NewRedisClient to exist in ProviderSet")
		}
	})

	// 测试添加已存在的函数（应该不报错，保持幂等性）
	t.Run("Add existing function", func(t *testing.T) {
		err := g.UpsertProviderSetFunction(testFile, "server.NewRestServer")
		if err != nil {
			t.Fatalf("UpsertProviderSetFunction failed: %v", err)
		}

		// 读取文件内容，验证没有重复
		content, _ := os.ReadFile(testFile)
		contentStr := string(content)

		// 简单计数检查（应该只出现一次）
		count := 0
		start := 0
		searchStr := "server.NewRestServer"
		for {
			idx := indexOf(contentStr[start:], searchStr)
			if idx == -1 {
				break
			}
			count++
			start += idx + len(searchStr)
		}

		if count != 1 {
			t.Errorf("expected server.NewRestServer to appear once, got %d times", count)
		}
	})

	// 测试批量添加
	t.Run("Add multiple functions", func(t *testing.T) {
		functions := []string{
			"server.NewKafkaProducer",
			"server.NewKafkaConsumer",
		}

		err := g.UpsertProviderSetFunctions(testFile, functions)
		if err != nil {
			t.Fatalf("UpsertProviderSetFunctions failed: %v", err)
		}

		// 验证所有函数都已添加
		for _, fn := range functions {
			exists, err := g.CheckProviderSetFunctionExists(testFile, fn)
			if err != nil {
				t.Fatalf("CheckProviderSetFunctionExists failed for %s: %v", fn, err)
			}
			if !exists {
				t.Errorf("expected %s to exist in ProviderSet", fn)
			}
		}
	})

	// 打印最终文件内容（用于调试）
	t.Run("Print final content", func(t *testing.T) {
		content, _ := os.ReadFile(testFile)
		t.Logf("Final wire.go content:\n%s", string(content))
	})
}

func TestUpsertProviderSetFunctionEmptySet(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "wire.go")

	// 空的 ProviderSet
	initialContent := `package server

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet()
`

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	g := NewGoGenerator()

	// 向空的 NewSet 添加函数
	err = g.UpsertProviderSetFunction(testFile, "server.NewRestServer")
	if err != nil {
		t.Fatalf("UpsertProviderSetFunction failed: %v", err)
	}

	// 验证
	exists, err := g.CheckProviderSetFunctionExists(testFile, "server.NewRestServer")
	if err != nil {
		t.Fatalf("CheckProviderSetFunctionExists failed: %v", err)
	}
	if !exists {
		t.Error("expected server.NewRestServer to exist in ProviderSet")
	}

	content, _ := os.ReadFile(testFile)
	t.Logf("Final content:\n%s", string(content))
}

// TestUpsertProviderSetFunctionWithTrailingComma tests the bug fix for double commas
func TestUpsertProviderSetFunctionWithTrailingComma(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "wire.go")

	// 模拟已有尾部逗号的情况
	initialContent := `package data

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	data.NewRedisClient,

	data.NewGreeterRepo,
)
`

	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	g := NewGoGenerator()

	// 添加新函数
	err = g.UpsertProviderSetFunction(testFile, "data.NewUserRepo")
	if err != nil {
		t.Fatalf("UpsertProviderSetFunction failed: %v", err)
	}

	// 再添加一个
	err = g.UpsertProviderSetFunction(testFile, "data.NewPostRepo")
	if err != nil {
		t.Fatalf("UpsertProviderSetFunction failed: %v", err)
	}

	// 读取并检查结果
	content, _ := os.ReadFile(testFile)
	contentStr := string(content)

	t.Logf("Result content:\n%s", contentStr)

	// 检查不应该有双逗号
	if strings.Contains(contentStr, ",,") {
		t.Error("found double comma ',,' in output - bug not fixed!")
	}

	// 检查所有函数都存在
	expectedFuncs := []string{
		"data.NewRedisClient",
		"data.NewGreeterRepo",
		"data.NewUserRepo",
		"data.NewPostRepo",
	}

	for _, fn := range expectedFuncs {
		if !strings.Contains(contentStr, fn) {
			t.Errorf("expected function %s not found in output", fn)
		}
	}

	// 检查格式正确（每个函数后应该有逗号）
	for _, fn := range expectedFuncs {
		if !strings.Contains(contentStr, fn+",") {
			t.Errorf("expected function %s to have trailing comma", fn)
		}
	}
}

func TestCheckProviderSetFunctionExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "wire.go")

	content := `package server

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	server.NewRestMiddleware,
	server.NewRestServer,
)
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	g := NewGoGenerator()

	// 测试存在的函数
	exists, err := g.CheckProviderSetFunctionExists(testFile, "server.NewRestServer")
	if err != nil {
		t.Fatalf("CheckProviderSetFunctionExists failed: %v", err)
	}
	if !exists {
		t.Error("expected server.NewRestServer to exist")
	}

	// 测试不存在的函数
	exists, err = g.CheckProviderSetFunctionExists(testFile, "server.NewRedisClient")
	if err != nil {
		t.Fatalf("CheckProviderSetFunctionExists failed: %v", err)
	}
	if exists {
		t.Error("expected server.NewRedisClient to not exist")
	}
}

// 辅助函数：字符串查找
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
