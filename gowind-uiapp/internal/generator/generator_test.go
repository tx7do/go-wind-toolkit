package generator

import (
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"

	"github.com/stretchr/testify/assert"
)

// TestGenerator_CleanOptions 测试清空选项
func TestGenerator_CleanOptions(t *testing.T) {
	g := NewGenerator()

	// 添加一些选项
	g.AddOption(&Option{TableName: "users", Service: "core"})
	g.AddOption(&Option{TableName: "orders", Service: "core"})
	assert.Len(t, g.GetOptions(), 2)

	// 清空
	g.CleanOptions()
	assert.Empty(t, g.GetOptions())
}

// TestGenerator_AddOption_EmptyTable 测试空表名的处理
func TestGenerator_AddOption_EmptyTable(t *testing.T) {
	g := NewGenerator()

	// 空表名应该被忽略
	g.AddOption(&Option{TableName: "", Service: "core"})
	assert.Empty(t, g.GetOptions())

	// 有效表名应该被添加
	g.AddOption(&Option{TableName: "users", Service: "core"})
	assert.Len(t, g.GetOptions(), 1)
}

// TestGenerator_EditOption 测试编辑选项
func TestGenerator_EditOption(t *testing.T) {
	g := NewGenerator()

	// 添加初始选项
	g.AddOption(&Option{TableName: "users", Service: "core", Exclude: false})
	assert.Len(t, g.GetOptions(), 1)

	// 编辑选项
	g.EditOption(&Option{TableName: "users", Service: "admin"})
	opts := g.GetOptions()
	assert.Equal(t, "admin", opts[0].Service)
}

// TestGenerator_EditOption_NotFound 测试编辑不存在的选项
func TestGenerator_EditOption_NotFound(t *testing.T) {
	g := NewGenerator()

	// 尝试编辑不存在的选项（不应该报错）
	g.EditOption(&Option{TableName: "nonexistent", Service: "core"})

	// 选项列表应该保持不变
	assert.Empty(t, g.GetOptions())
}

// TestGenerator_ValidateOptions_Empty 测试空选项验证
func TestGenerator_ValidateOptions_Empty(t *testing.T) {
	g := NewGenerator()

	err := g.ValidateOptions()
	assert.Equal(t, "no tables selected", err)
}

// TestGenerator_ValidateOptions_Invalid 测试无效选项验证
func TestGenerator_ValidateOptions_Invalid(t *testing.T) {
	g := NewGenerator()

	// 直接操作内部 options 来模拟无效状态（AddOption 会过滤空表名）
	g.mu.Lock()
	g.options = GeneratorOptions{
		{TableName: "", Service: "core"}, // 空表名
	}
	g.mu.Unlock()

	err := g.ValidateOptions()
	assert.Equal(t, "table name cannot be empty", err)

	// 清空后添加空服务名的选项
	g.CleanOptions()
	g.mu.Lock()
	g.options = GeneratorOptions{
		{TableName: "users", Service: ""}, // 空服务名
	}
	g.mu.Unlock()

	err = g.ValidateOptions()
	assert.Equal(t, "service name cannot be empty", err)
}

// TestGenerator_GetValidateOptions 测试获取验证后的选项
func TestGenerator_GetValidateOptions(t *testing.T) {
	g := NewGenerator()

	// 添加混合选项
	g.AddOption(&Option{TableName: "users", Service: "core", Exclude: false})
	g.AddOption(&Option{TableName: "orders", Service: "core", Exclude: true}) // 排除
	g.AddOption(&Option{TableName: "products", Service: "admin", Exclude: false})
	g.AddOption(&Option{TableName: "", Service: "core", Exclude: false}) // 无效

	validOpts := g.GetValidateOptions()
	assert.Len(t, validOpts, 2) // 只有 users 和 products
}

// TestGenerator_ConcurrentAccess 测试并发访问安全
func TestGenerator_ConcurrentAccess(t *testing.T) {
	g := NewGenerator()

	done := make(chan bool, 10)

	// 启动多个 goroutine 同时读写
	for i := 0; i < 10; i++ {
		go func(id int) {
			g.AddOption(&Option{TableName: "table"+string(rune(id+'0')), Service: "core"})
			_ = g.GetOptions()
			g.CleanOptions()
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 不应该有 panic 或数据竞争
	assert.NotNil(t, g)
}

// TestGenerator_SetLogger_Nil 测试设置 nil logger
func TestGenerator_SetLogger_Nil(t *testing.T) {
	g := NewGenerator()

	// 设置 nil logger 不应该 panic
	g.SetLogger(nil)

	// 应该使用 noopLogger
	assert.NotNil(t, g)
}

// TestGenerator_GenerateGrpcCode_NoOptions 测试无选项时的生成
func TestGenerator_GenerateGrpcCode_NoOptions(t *testing.T) {
	g := NewGenerator()

	// 无选项时应该返回错误
	err := g.GenerateGrpcCode(
		nil, /* context */
		database.DBConfig{},
		"ent",
		"per-table",
		"/tmp",
		"test",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "没有可用的表选项")
}

// TestGenerator_GenerateRestCode_NoOptions 测试无选项时的 REST 生成
func TestGenerator_GenerateRestCode_NoOptions(t *testing.T) {
	g := NewGenerator()

	err := g.GenerateRestCode(
		nil, /* context */
		"admin",
		"ent",
		"per-table",
		database.DBConfig{},
		"/tmp",
		"test",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "没有可用的表选项")
}
