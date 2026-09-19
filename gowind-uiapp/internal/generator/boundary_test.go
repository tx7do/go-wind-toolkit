package generator

import (
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"

	"github.com/stretchr/testify/assert"
)

// TestGenerator_OptionIDAssignment 测试选项 ID 分配
func TestGenerator_OptionIDAssignment(t *testing.T) {
	g := NewGenerator()

	// 添加多个选项，验证 ID 递增
	g.AddOption(&Option{TableName: "users", Service: "core"})
	g.AddOption(&Option{TableName: "orders", Service: "core"})
	g.AddOption(&Option{TableName: "products", Service: "core"})

	opts := g.GetOptions()
	assert.Equal(t, uint32(1), opts[0].ID)
	assert.Equal(t, uint32(2), opts[1].ID)
	assert.Equal(t, uint32(3), opts[2].ID)
}

// TestGenerator_ExportedFieldAccess 测试直接访问导出字段（并发安全）
func TestGenerator_ExportedFieldAccess(t *testing.T) {
	g := NewGenerator()

	// 直接修改内部状态（模拟并发场景）
	g.mu.Lock()
	g.options = GeneratorOptions{
		{TableName: "test", Service: "core"},
	}
	g.mu.Unlock()

	// 通过公开方法读取
	opts := g.GetOptions()
	assert.Len(t, opts, 1)
}

// TestGenerator_MultipleServices 测试多服务场景
func TestGenerator_MultipleServices(t *testing.T) {
	g := NewGenerator()

	// 添加不同服务的表
	g.AddOption(&Option{TableName: "users", Service: "auth"})
	g.AddOption(&Option{TableName: "orders", Service: "commerce"})
	g.AddOption(&Option{TableName: "products", Service: "commerce"})
	g.AddOption(&Option{TableName: "roles", Service: "auth"})

	validOpts := g.GetValidateOptions()
	assert.Len(t, validOpts, 4)

	// 验证服务分布
	authCount := 0
	commerceCount := 0
	for _, opt := range validOpts {
		if opt.Service == "auth" {
			authCount++
		} else if opt.Service == "commerce" {
			commerceCount++
		}
	}
	assert.Equal(t, 2, authCount)
	assert.Equal(t, 2, commerceCount)
}

// TestGenerator_ExcludedTables 测试排除表的过滤
func TestGenerator_ExcludedTables(t *testing.T) {
	g := NewGenerator()

	g.AddOption(&Option{TableName: "users", Service: "core", Exclude: false})
	g.AddOption(&Option{TableName: "temp_table", Service: "core", Exclude: true})
	g.AddOption(&Option{TableName: "orders", Service: "core", Exclude: false})
	g.AddOption(&Option{TableName: "logs", Service: "core", Exclude: true})

	validOpts := g.GetValidateOptions()
	assert.Len(t, validOpts, 2) // 只有 users 和 orders
}

// TestGenerator_ServiceGrouping 测试按服务分组
func TestGenerator_ServiceGrouping(t *testing.T) {
	g := NewGenerator()

	// 添加混合服务的表
	services := []string{"auth", "commerce", "notification", "auth"}
	tables := []string{"users", "orders", "emails", "roles"}

	for i, svc := range services {
		g.AddOption(&Option{
			TableName: tables[i],
			Service:   svc,
		})
	}

	// 验证 GetValidateOptions 保持插入顺序
	validOpts := g.GetValidateOptions()
	assert.Len(t, validOpts, 4)
	assert.Equal(t, "users", validOpts[0].TableName)
	assert.Equal(t, "orders", validOpts[1].TableName)
	assert.Equal(t, "emails", validOpts[2].TableName)
	assert.Equal(t, "roles", validOpts[3].TableName)
}

// TestGenerator_ProtoPackageCustom 测试自定义 proto package
func TestGenerator_ProtoPackageCustom(t *testing.T) {
	g := NewGenerator()

	g.AddOption(&Option{
		TableName:    "users",
		Service:      "core",
		ProtoPackage: "custom.proto.v1",
	})

	opts := g.GetOptions()
	assert.Equal(t, "custom.proto.v1", opts[0].ProtoPackage)
}

// TestGenerator_EmptyServiceName 测试空服务名被过滤
func TestGenerator_EmptyServiceName(t *testing.T) {
	g := NewGenerator()

	g.AddOption(&Option{TableName: "users", Service: "core"})
	g.AddOption(&Option{TableName: "temp", Service: ""}) // 空服务名
	g.AddOption(&Option{TableName: "orders", Service: "admin"})

	validOpts := g.GetValidateOptions()
	// 空服务名的表会被 ValidateOptions 拒绝，不会出现在有效列表中
	assert.Len(t, validOpts, 2)
}

// TestGenerator_SetOptions_CopyBehavior 测试 SetOptions 的副本行为
func TestGenerator_SetOptions_CopyBehavior(t *testing.T) {
	g := NewGenerator()

	// 设置初始选项
	initialOpts := GeneratorOptions{
		{TableName: "users", Service: "core"},
	}
	g.SetOptions(initialOpts)

	// 修改原始切片（不应该影响内部状态）
	initialOpts[0].TableName = "modified"

	// 验证内部状态未被影响
	opts := g.GetOptions()
	assert.Equal(t, "users", opts[0].TableName)
}

// TestGenerator_LargeDataset 测试大数据集性能
func TestGenerator_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("skip performance test")
	}

	g := NewGenerator()

	// 添加大量表
	numTables := 100
	for i := 0; i < numTables; i++ {
		g.AddOption(&Option{
			TableName: "table_" + string(rune('0'+i%10)),
			Service:   "core",
		})
	}

	opts := g.GetOptions()
	assert.Len(t, opts, numTables)
}

// TestGenerator_ConcurrentAddAndRead 测试并发添加和读取
func TestGenerator_ConcurrentAddAndRead(t *testing.T) {
	g := NewGenerator()

	done := make(chan bool, 20)

	// 10 个 goroutine 添加，10 个 goroutine 读取
	for i := 0; i < 10; i++ {
		go func(id int) {
			g.AddOption(&Option{TableName: "table"+string(rune(id+'0')), Service: "core"})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func() {
			_ = g.GetOptions()
			done <- true
		}()
	}

	// 等待所有操作完成
	for i := 0; i < 20; i++ {
		<-done
	}

	// 验证最终状态
	opts := g.GetOptions()
	assert.Len(t, opts, 10)
}

// TestGenerator_CleanAfterAdd 测试添加后清空
func TestGenerator_CleanAfterAdd(t *testing.T) {
	g := NewGenerator()

	// 添加一些选项
	g.AddOption(&Option{TableName: "users", Service: "core"})
	g.AddOption(&Option{TableName: "orders", Service: "core"})

	// 清空
	g.CleanOptions()

	// 再次添加应该正常工作
	g.AddOption(&Option{TableName: "products", Service: "admin"})
	opts := g.GetOptions()
	assert.Len(t, opts, 1)
	assert.Equal(t, "products", opts[0].TableName)
}

// TestGenerator_InvalidProtoPackage 测试无效 proto package 格式
func TestGenerator_InvalidProtoPackage(t *testing.T) {
	g := NewGenerator()

	// 各种无效的 proto package 格式
	testCases := []string{
		"",                    // 空
		"/invalid",           // 以 / 开头
		"no.dots.here",       // 没有点
		"UPPERCASE",          // 全大写
		"mixedCase123",       // 混合大小写和数字
	}

	for _, pkg := range testCases {
		g.AddOption(&Option{
			TableName:    "test",
			Service:      "core",
			ProtoPackage: pkg,
		})
	}

	opts := g.GetOptions()
	assert.Len(t, opts, 5) // 所有选项都应该被添加（validation 不检查 proto package 格式）
}

// TestGenerator_GenerateWithEmptyDBConfig 测试空 DBConfig 生成
func TestGenerator_GenerateWithEmptyDBConfig(t *testing.T) {
	g := NewGenerator()

	// 添加一个有效选项
	g.AddOption(&Option{TableName: "users", Service: "core"})

	// 使用空的 DBConfig
	err := g.GenerateGrpcCode(
		nil, /* context */
		database.DBConfig{}, // 空配置
		"ent",
		"per-table",
		"/tmp",
		"test",
	)
	// 预期会失败（因为没有有效的数据源）
	assert.Error(t, err)
}

// TestGenerator_DuplicateTableName 测试重复表名的处理
func TestGenerator_DuplicateTableName(t *testing.T) {
	g := NewGenerator()

	// 添加相同表名但不同服务的选项
	g.AddOption(&Option{TableName: "users", Service: "auth"})
	g.AddOption(&Option{TableName: "users", Service: "commerce"})

	opts := g.GetOptions()
	assert.Len(t, opts, 2) // 两个选项都应该被添加

	// 验证它们是不同的实例
	assert.Equal(t, "auth", opts[0].Service)
	assert.Equal(t, "commerce", opts[1].Service)
}

// TestGenerator_EdgeCase_EmptyStringFields 测试空字符串字段
func TestGenerator_EdgeCase_EmptyStringFields(t *testing.T) {
	g := NewGenerator()

	// 表名和服务名都是空字符串的情况
	g.AddOption(&Option{TableName: "", Service: ""})

	// 空表名会被 AddOption 过滤
	opts := g.GetOptions()
	assert.Empty(t, opts)
}

// TestGenerator_IDResetAfterClean 测试清空后 ID 重置
func TestGenerator_IDResetAfterClean(t *testing.T) {
	g := NewGenerator()

	// 第一轮添加
	g.AddOption(&Option{TableName: "first", Service: "core"})
	g.AddOption(&Option{TableName: "second", Service: "core"})

	// 清空
	g.CleanOptions()

	// 第二轮添加，ID 应该从 1 开始
	g.AddOption(&Option{TableName: "third", Service: "core"})
	g.AddOption(&Option{TableName: "fourth", Service: "core"})

	opts := g.GetOptions()
	assert.Equal(t, uint32(1), opts[0].ID)
	assert.Equal(t, uint32(2), opts[1].ID)
}
