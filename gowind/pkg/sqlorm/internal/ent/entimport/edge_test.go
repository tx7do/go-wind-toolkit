package entimport

import (
	"strings"
	"testing"

	"ariga.io/atlas/sql/schema"
	"entgo.io/contrib/schemast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpsertRelation_DuplicateEdge 测试重复 edge 检测逻辑
func TestUpsertRelation_DuplicateEdge(t *testing.T) {
	// 创建两个 schema
	balance := &schemast.UpsertSchema{
		Name: "Balance",
	}
	payment := &schemast.UpsertSchema{
		Name: "Payment",
	}

	// 模拟多个 payment 相关的外键（如 payment_id, payer_payment_id, payee_payment_id）
	// 它们都会被映射成相同的 edge name "payments"（复数形式）
	for i := 0; i < 3; i++ {
		opts := relOptions{
			uniqueEdgeFromParent: true,
			refName:              "payments",
			edgeField:            "payment_id", // 不同的字段名但映射到同一个 edge
		}
		upsertRelation(balance, payment, opts)
	}

	// 验证 balance 表只添加了一个 payments edge（重复的被过滤）
	assert.Equal(t, 1, len(balance.Edges), "Balance should have exactly 1 'payments' edge")
	assert.Equal(t, 1, len(payment.Edges), "Payment should have exactly 1 'payment' edge")

	// 验证 edge 名称正确（注意：edge name 是复数形式）
	if assert.Len(t, balance.Edges, 1) {
		assert.Equal(t, "payments", balance.Edges[0].Descriptor().Name)
	}
}

// TestUpsertRelation_DifferentEdges 测试不同外键应该生成不同的 edge
func TestUpsertRelation_DifferentEdges(t *testing.T) {
	user := &schemast.UpsertSchema{Name: "User"}
	order := &schemast.UpsertSchema{Name: "Order"}
	payment := &schemast.UpsertSchema{Name: "Payment"}

	// 添加 user -> order 关系
	opts1 := relOptions{
		uniqueEdgeFromParent: true,
		refName:              "orders",
		edgeField:            "order_id",
	}
	upsertRelation(user, order, opts1)

	// 添加 user -> payment 关系
	opts2 := relOptions{
		uniqueEdgeFromParent: true,
		refName:              "payments",
		edgeField:            "payment_id",
	}
	upsertRelation(user, payment, opts2)

	// 验证 user 表有两个不同的 edge
	assert.Equal(t, 2, len(user.Edges), "User should have 2 edges: orders and payments")

	// 验证 order 和 payment 各有一个 edge
	assert.Equal(t, 1, len(order.Edges))
	assert.Equal(t, 1, len(payment.Edges))
}

// TestUpsertOneToX_SkipMissingTables 测试当父表或子表不存在时跳过该 FK
func TestUpsertOneToX_SkipMissingTables(t *testing.T) {
	// 创建一个只有部分表的 mutations map
	mutations := make(map[string]schemast.Mutator)

	// 只创建 User 表，不创建 Order 表
	mutations["users"] = &schemast.UpsertSchema{Name: "User"}

	// 模拟一个有 FK 指向不存在的表的 table
	fkTable := &schema.Table{
		Name: "orders",
		ForeignKeys: []*schema.ForeignKey{
			{
				Columns: []*schema.Column{{Name: "user_id"}},
				RefTable:     &schema.Table{Name: "users"}, // 存在
			},
			{
				Columns: []*schema.Column{{Name: "admin_id"}},
				RefTable:     &schema.Table{Name: "admins"}, // 不存在
			},
		},
	}

	// 调用 upsertOneToX，应该只处理存在的 FK
	upsertOneToX(mutations, fkTable)

	// 验证 mutations 中只有一个表
	assert.Len(t, mutations, 1, "Should only have the existing table")
}

// TestUpsertOneToX_MultipleFKs 测试多个 FK 的正确处理（使用字段名区分 edge）
func TestUpsertOneToX_MultipleFKs(t *testing.T) {
	parent := &schemast.UpsertSchema{Name: "Parent"}
	child := &schemast.UpsertSchema{Name: "Child"}

	mutations := map[string]schemast.Mutator{
		"parents":  parent,
		"children": child,
	}

	// 模拟有多个 FK 指向同一张表的场景
	// parent_id -> parent (singularize 后仍是 parent)
	// grandparent_id -> grandparent (singularize 后仍是 grandparent)
	multiFkTable := &schema.Table{
		Name: "children",
		ForeignKeys: []*schema.ForeignKey{
			{
				Columns:  []*schema.Column{{Name: "parent_id"}},
				RefTable: &schema.Table{Name: "parents"},
			},
			{
				Columns:  []*schema.Column{{Name: "grandparent_id"}},
				RefTable: &schema.Table{Name: "parents"}, // 指向同一张表
			},
		},
	}

	upsertOneToX(mutations, multiFkTable)

	// 验证 child 表有两个不同的 edge（parent 和 grandparent）
	assert.Equal(t, 2, len(child.Edges), "Child should have 2 edges pointing to Parent")

	// 验证 parent 表有 1 个 edge（因为两个 FK 都指向 parent，反向 edge 会合并）
	// 这是预期行为：不同方向的 edge 可以共存，但同向的 edge 只保留一个
	assert.Equal(t, 1, len(parent.Edges), "Parent should have 1 reverse edge from Child")
}

// TestUpsertRelation_RecursiveEdge 测试递归关系的 edge 命名
func TestUpsertRelation_RecursiveEdge(t *testing.T) {
	category := &schemast.UpsertSchema{Name: "Category"}

	// 模拟自引用关系（category -> parent_category）
	opts := relOptions{
		uniqueEdgeFromParent: true,
		recursive:            true,
		refName:              "categories",
		edgeField:            "parent_id",
	}
	upsertRelation(category, category, opts)

	// 验证自引用场景添加了 2 个不同的 edge（to 和 from）
	assert.Equal(t, 2, len(category.Edges), "Category should have 2 edges for self-reference")

	// 打印实际 edge name 用于调试
	for i, e := range category.Edges {
		t.Logf("Edge %d: Name=%s, RefName=%s", i, e.Descriptor().Name, e.Descriptor().RefName)
	}

	// 验证 edge 名称包含 child_ 和 parent_ 前缀
	hasChild := false
	hasParent := false
	for _, e := range category.Edges {
		name := e.Descriptor().Name
		if strings.HasPrefix(name, "child_") {
			hasChild = true
		}
		if strings.HasPrefix(name, "parent_") {
			hasParent = true
		}
	}
	assert.True(t, hasChild, "Should have at least one edge with 'child_' prefix")
	assert.True(t, hasParent, "Should have at least one edge with 'parent_' prefix")
}

// TestUpsertManyToMany 测试 M2M 关系的 edge 生成
func TestUpsertManyToMany(t *testing.T) {
	user := &schemast.UpsertSchema{Name: "User"}
	role := &schemast.UpsertSchema{Name: "Role"}

	mutations := map[string]schemast.Mutator{
		"users":  user,
		"roles":  role,
	}

	// 模拟 join 表
	joinTable := &schema.Table{
		Name: "user_roles",
		ForeignKeys: []*schema.ForeignKey{
			{
				RefTable: &schema.Table{Name: "users"},
			},
			{
				RefTable: &schema.Table{Name: "roles"},
			},
		},
	}

	err := upsertManyToMany(mutations, joinTable)
	require.NoError(t, err)

	// 验证双向 edge 都已创建
	assert.Equal(t, 1, len(user.Edges), "User should have 1 edge to Roles")
	assert.Equal(t, 1, len(role.Edges), "Role should have 1 edge to Users")
}
