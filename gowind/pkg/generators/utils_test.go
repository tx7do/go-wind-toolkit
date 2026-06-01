package generators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple snake case",
			input:    "user_name",
			expected: "UserName",
		},
		{
			name:     "single word",
			input:    "user",
			expected: "User",
		},
		{
			name:     "multiple underscores",
			input:    "user_first_name",
			expected: "UserFirstName",
		},
		{
			name:     "with numbers",
			input:    "user_id",
			expected: "UserId",
		},
		{
			name:     "already pascal case without underscore",
			input:    "UserName",
			expected: "UserName",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SnakeToPascal(tt.input)
			if result != tt.expected {
				t.Errorf("SnakeToPascal(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSnakeToPascalPlus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple snake case",
			input:    "user_name",
			expected: "UserName",
		},
		{
			name:     "with id suffix",
			input:    "user_id",
			expected: "UserID",
		},
		{
			name:     "with id in middle",
			input:    "user_id_name",
			expected: "UserIDName",
		},
		{
			name:     "only id",
			input:    "id",
			expected: "ID",
		},
		{
			name:     "multiple ids",
			input:    "user_id_parent_id",
			expected: "UserIDParentID",
		},
		{
			name:     "no id",
			input:    "user_name",
			expected: "UserName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SnakeToPascalPlus(tt.input)
			if result != tt.expected {
				t.Errorf("SnakeToPascalPlus(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMakeEntSetNillableFunc(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "simple field name",
			fieldName: "user_name",
			expected:  "SetNillableUserName(req.Data.UserName)",
		},
		{
			name:      "field with id",
			fieldName: "user_id",
			expected:  "SetNillableUserID(req.Data.UserId)",
		},
		{
			name:      "single word",
			fieldName: "name",
			expected:  "SetNillableName(req.Data.Name)",
		},
		{
			name:      "complex field name",
			fieldName: "parent_user_id",
			expected:  "SetNillableParentUserID(req.Data.ParentUserId)",
		},
		{
			name:      "only id",
			fieldName: "id",
			expected:  "SetNillableID(req.Data.Id)",
		},
		{
			name:      "multiple underscores",
			fieldName: "user_first_name",
			expected:  "SetNillableUserFirstName(req.Data.UserFirstName)",
		},
		{
			name:      "with multiple ids",
			fieldName: "user_id_parent_id",
			expected:  "SetNillableUserIDParentID(req.Data.UserIdParentId)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MakeEntSetNillableFunc(tt.fieldName)
			if result != tt.expected {
				t.Errorf("MakeEntSetNillableFunc(%q) = %q, expected %q", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestMakeEntSetNillableFuncWithTransfer(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		transFunc string
		expected  string
	}{
		{
			name:      "with pointer conversion",
			fieldName: "user_name",
			transFunc: "stringToPointer",
			expected:  "SetNillableUserName(stringToPointer(req.Data.UserName))",
		},
		{
			name:      "with int conversion",
			fieldName: "user_id",
			transFunc: "int32ToInt64",
			expected:  "SetNillableUserID(int32ToInt64(req.Data.UserId))",
		},
		{
			name:      "with timestamp conversion",
			fieldName: "created_at",
			transFunc: "timestampToTime",
			expected:  "SetNillableCreatedAt(timestampToTime(req.Data.CreatedAt))",
		},
		{
			name:      "with enum conversion",
			fieldName: "status",
			transFunc: "toEnum",
			expected:  "SetNillableStatus(toEnum(req.Data.Status))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MakeEntSetNillableFuncWithTransfer(tt.fieldName, tt.transFunc)
			if result != tt.expected {
				t.Errorf("MakeEntSetNillableFuncWithTransfer(%q, %q) = %q, expected %q",
					tt.fieldName, tt.transFunc, result, tt.expected)
			}
		})
	}
}

func TestMakeEntSetFunc(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "simple field name",
			fieldName: "user_name",
			expected:  "SetUserName(req.Data.GetUserName())",
		},
		{
			name:      "field with id",
			fieldName: "user_id",
			expected:  "SetUserID(req.Data.GetUserId())",
		},
		{
			name:      "single word",
			fieldName: "name",
			expected:  "SetName(req.Data.GetName())",
		},
		{
			name:      "complex field name",
			fieldName: "parent_user_id",
			expected:  "SetParentUserID(req.Data.GetParentUserId())",
		},
		{
			name:      "only id",
			fieldName: "id",
			expected:  "SetID(req.Data.GetId())",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MakeEntSetFunc(tt.fieldName)
			if result != tt.expected {
				t.Errorf("MakeEntSetFunc(%q) = %q, expected %q", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestRemoveTableCommentSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with Chinese suffix",
			input:    "用户表",
			expected: "用户",
		},
		{
			name:     "with English suffix",
			input:    "user table",
			expected: "user ",
		},
		{
			name:     "no suffix",
			input:    "用户信息",
			expected: "用户信息",
		},
		{
			name:     "suffix in middle",
			input:    "表格数据",
			expected: "表格数据",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only suffix Chinese",
			input:    "表",
			expected: "",
		},
		{
			name:     "only suffix English",
			input:    "table",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveTableCommentSuffix(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveTableCommentSuffix(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestServerImportPaths(t *testing.T) {
	tests := []struct {
		name     string
		servers  []string
		expected []string
	}{
		{
			name:     "grpc only",
			servers:  []string{"grpc"},
			expected: []string{"github.com/go-kratos/kratos/v2/transport/grpc"},
		},
		{
			name:     "rest only",
			servers:  []string{"rest"},
			expected: []string{"github.com/go-kratos/kratos/v2/transport/http"},
		},
		{
			name:    "grpc and rest",
			servers: []string{"grpc", "rest"},
			expected: []string{
				"github.com/go-kratos/kratos/v2/transport/grpc",
				"github.com/go-kratos/kratos/v2/transport/http",
			},
		},
		{
			name:    "duplicate servers",
			servers: []string{"grpc", "grpc", "rest"},
			expected: []string{
				"github.com/go-kratos/kratos/v2/transport/grpc",
				"github.com/go-kratos/kratos/v2/transport/http",
			},
		},
		{
			name:     "kafka",
			servers:  []string{"kafka"},
			expected: []string{"github.com/tx7do/kratos-transport/transport/kafka"},
		},
		{
			name:     "mqtt",
			servers:  []string{"mqtt"},
			expected: []string{"github.com/tx7do/kratos-transport/transport/mqtt"},
		},
		{
			name:    "mixed transports",
			servers: []string{"grpc", "kafka", "mqtt"},
			expected: []string{
				"github.com/go-kratos/kratos/v2/transport/grpc",
				"github.com/tx7do/kratos-transport/transport/kafka",
				"github.com/tx7do/kratos-transport/transport/mqtt",
			},
		},
		{
			name:    "case insensitive",
			servers: []string{"GRPC", "Rest", "KafKa"},
			expected: []string{
				"github.com/go-kratos/kratos/v2/transport/grpc",
				"github.com/go-kratos/kratos/v2/transport/http",
				"github.com/tx7do/kratos-transport/transport/kafka",
			},
		},
		{
			name:     "empty list",
			servers:  []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ServerImportPaths(tt.servers)
			if len(result) != len(tt.expected) {
				t.Errorf("ServerImportPaths(%v) returned %d paths, expected %d",
					tt.servers, len(result), len(tt.expected))
				return
			}
			for i, path := range result {
				if path != tt.expected[i] {
					t.Errorf("ServerImportPaths(%v)[%d] = %q, expected %q",
						tt.servers, i, path, tt.expected[i])
				}
			}
		})
	}
}

func TestServerFormalParameters(t *testing.T) {
	tests := []struct {
		name     string
		servers  []string
		expected []string
	}{
		{
			name:     "grpc only",
			servers:  []string{"grpc"},
			expected: []string{"gs *grpc.Server"},
		},
		{
			name:     "rest only",
			servers:  []string{"rest"},
			expected: []string{"hs *http.Server"},
		},
		{
			name:     "grpc and rest",
			servers:  []string{"grpc", "rest"},
			expected: []string{"gs *grpc.Server", "hs *http.Server"},
		},
		{
			name:     "kafka",
			servers:  []string{"kafka"},
			expected: []string{"ks *kafka.Server"},
		},
		{
			name:     "duplicate servers",
			servers:  []string{"grpc", "grpc"},
			expected: []string{"gs *grpc.Server"},
		},
		{
			name:     "empty list",
			servers:  []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ServerFormalParameters(tt.servers)
			if len(result) != len(tt.expected) {
				t.Errorf("ServerFormalParameters(%v) returned %d params, expected %d",
					tt.servers, len(result), len(tt.expected))
				return
			}
			for i, param := range result {
				if param != tt.expected[i] {
					t.Errorf("ServerFormalParameters(%v)[%d] = %q, expected %q",
						tt.servers, i, param, tt.expected[i])
				}
			}
		})
	}
}

func TestServerTransferParameters(t *testing.T) {
	tests := []struct {
		name     string
		servers  []string
		expected []string
	}{
		{
			name:     "grpc only",
			servers:  []string{"grpc"},
			expected: []string{"gs"},
		},
		{
			name:     "rest only",
			servers:  []string{"rest"},
			expected: []string{"hs"},
		},
		{
			name:     "grpc and rest",
			servers:  []string{"grpc", "rest"},
			expected: []string{"gs", "hs"},
		},
		{
			name:     "kafka",
			servers:  []string{"kafka"},
			expected: []string{"ks"},
		},
		{
			name:     "mqtt",
			servers:  []string{"mqtt"},
			expected: []string{"ms"},
		},
		{
			name:     "duplicate servers",
			servers:  []string{"grpc", "grpc"},
			expected: []string{"gs"},
		},
		{
			name:     "empty list",
			servers:  []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ServerTransferParameters(tt.servers)
			if len(result) != len(tt.expected) {
				t.Errorf("ServerTransferParameters(%v) returned %d params, expected %d",
					tt.servers, len(result), len(tt.expected))
				return
			}
			for i, param := range result {
				if param != tt.expected[i] {
					t.Errorf("ServerTransferParameters(%v)[%d] = %q, expected %q",
						tt.servers, i, param, tt.expected[i])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkMakeEntSetNillableFunc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MakeEntSetNillableFunc("user_name")
	}
}

func BenchmarkSnakeToPascal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SnakeToPascal("user_first_name")
	}
}

func BenchmarkSnakeToPascalPlus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SnakeToPascalPlus("user_id")
	}
}

func BenchmarkServerImportPaths(b *testing.B) {
	servers := []string{"grpc", "rest", "kafka", "mqtt"}
	for i := 0; i < b.N; i++ {
		ServerImportPaths(servers)
	}
}

func TestApiPackageAlias(t *testing.T) {
	tests := []struct {
		name     string
		pkgName  string
		version  string
		expected string
	}{
		{
			name:     "user + v1",
			pkgName:  "user",
			version:  "v1",
			expected: "userV1",
		},
		{
			name:     "admin + v1",
			pkgName:  "admin",
			version:  "v1",
			expected: "adminV1",
		},
		{
			name:     "user_service + v2",
			pkgName:  "user_service",
			version:  "v2",
			expected: "userServiceV2",
		},
		{
			name:     "empty version",
			pkgName:  "user",
			version:  "",
			expected: "user",
		},
		{
			name:     "single char name + v1",
			pkgName:  "a",
			version:  "v1",
			expected: "aV1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := apiPackageAlias(tt.pkgName, tt.version)
			if result != tt.expected {
				t.Errorf("apiPackageAlias(%q, %q) = %q, expected %q", tt.pkgName, tt.version, result, tt.expected)
			}
		})
	}
}

// ==============================
// renderImports
// ==============================

func TestRenderImports_Nil(t *testing.T) {
	assert.Equal(t, "", renderImports(nil))
}

func TestRenderImports_String(t *testing.T) {
	result := renderImports("github.com/example/pkg")
	assert.Equal(t, "\tgithub.com/example/pkg\n", result)
}

func TestRenderImports_EmptyString(t *testing.T) {
	assert.Equal(t, "", renderImports(""))
}

func TestRenderImports_StringSlice(t *testing.T) {
	result := renderImports([]string{"github.com/example/pkg1", "github.com/example/pkg2"})
	assert.Contains(t, result, "github.com/example/pkg1")
	assert.Contains(t, result, "github.com/example/pkg2")
}

func TestRenderImports_StringSliceWithEmpty(t *testing.T) {
	result := renderImports([]string{"", "github.com/example/pkg1", ""})
	assert.Contains(t, result, "github.com/example/pkg1")
	assert.NotContains(t, result, "\t\"\"\n")
}

func TestRenderImports_UnsupportedType(t *testing.T) {
	assert.Equal(t, "", renderImports(42))
}

// ==============================
// renderFormalParameters
// ==============================

func TestRenderFormalParameters_Nil(t *testing.T) {
	assert.Equal(t, "", renderFormalParameters(nil))
}

func TestRenderFormalParameters_String(t *testing.T) {
	result := renderFormalParameters("hs *http.Server")
	assert.Contains(t, result, "hs *http.Server")
	assert.Contains(t, result, ",")
}

func TestRenderFormalParameters_EmptyString(t *testing.T) {
	assert.Equal(t, "", renderFormalParameters(""))
}

func TestRenderFormalParameters_StringSlice(t *testing.T) {
	result := renderFormalParameters([]string{"hs *http.Server", "gs *grpc.Server"})
	assert.Contains(t, result, "hs *http.Server")
	assert.Contains(t, result, "gs *grpc.Server")
}

func TestRenderFormalParameters_AnySlice(t *testing.T) {
	result := renderFormalParameters([]any{"hs *http.Server", 123})
	assert.Contains(t, result, "hs *http.Server")
	assert.Contains(t, result, "123")
}

func TestRenderFormalParameters_CustomTabs(t *testing.T) {
	result := renderFormalParameters("hs *http.Server", 2)
	assert.Contains(t, result, "hs *http.Server")
}

func TestRenderFormalParameters_UnsupportedType(t *testing.T) {
	assert.Equal(t, "", renderFormalParameters(42))
}

// ==============================
// renderInParameters
// ==============================

func TestRenderInParameters_Nil(t *testing.T) {
	assert.Equal(t, "", renderInParameters(nil))
}

func TestRenderInParameters_String(t *testing.T) {
	result := renderInParameters("hs")
	assert.Contains(t, result, "hs")
	assert.Contains(t, result, ",")
}

func TestRenderInParameters_EmptyString(t *testing.T) {
	assert.Equal(t, "", renderInParameters(""))
}

func TestRenderInParameters_StringSlice(t *testing.T) {
	result := renderInParameters([]string{"hs", "gs"})
	assert.Contains(t, result, "hs,")
	assert.Contains(t, result, "gs,")
}

func TestRenderInParameters_AnySlice(t *testing.T) {
	result := renderInParameters([]any{"hs", 42})
	assert.Contains(t, result, "hs")
	assert.Contains(t, result, "42")
}

func TestRenderInParameters_UnsupportedType(t *testing.T) {
	assert.Equal(t, "", renderInParameters(42))
}

// ==============================
// renderServiceName / renderRepoName / renderServerName
// ==============================

func TestRenderServiceName(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"user", "UserService"},
		{"user_role", "UserRoleService"},
		{"", "Service"},
		{nil, ""},
		{42, ""},
	}
	for _, tt := range tests {
		result := renderServiceName(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestRenderRepoName(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"user", "UserRepo"},
		{"user_role", "UserRoleRepo"},
		{"", "Repo"},
		{nil, ""},
		{42, ""},
	}
	for _, tt := range tests {
		result := renderRepoName(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestRenderServerName(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"rest", "RestServer"},
		{"grpc", "GrpcServer"},
		{"", "Server"},
		{nil, ""},
		{42, ""},
	}
	for _, tt := range tests {
		result := renderServerName(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
