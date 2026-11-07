package schemasource

import (
	"testing"

	"ariga.io/atlas/sql/schema"
)

func newTable(name string) *schema.Table {
	t := schema.NewTable(name)
	return t
}

func TestFilterTables(t *testing.T) {
	tables := []*schema.Table{newTable("users"), newTable("roles"), newTable("user_roles"), newTable("permissions")}

	tests := []struct {
		name    string
		include []string
		exclude []string
		want    []string
	}{
		{"空参原样返回(同一切片)", nil, nil, []string{"users", "roles", "user_roles", "permissions"}},
		{"include 白名单保序", []string{"roles", "users"}, nil, []string{"users", "roles"}},
		{"include 含未知表名忽略", []string{"users", "ghost"}, nil, []string{"users"}},
		{"exclude 黑名单", nil, []string{"user_roles", "permissions"}, []string{"users", "roles"}},
		{"先白后黑", []string{"users", "roles", "user_roles"}, []string{"user_roles"}, []string{"users", "roles"}},
		{"空 include 切片等同未给", []string{}, nil, []string{"users", "roles", "user_roles", "permissions"}},
		{"exclude 全部", nil, []string{"users", "roles", "user_roles", "permissions"}, []string{}},
		{"精确匹配不做大小写折叠", []string{"Users"}, nil, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterTables(tables, tt.include, tt.exclude)
			names := make([]string, 0, len(got))
			for _, tb := range got {
				names = append(names, tb.Name)
			}
			if len(names) == 0 && len(tt.want) == 0 {
				return
			}
			if len(names) != len(tt.want) {
				t.Fatalf("过滤结果 = %v, want %v", names, tt.want)
			}
			for i := range names {
				if names[i] != tt.want[i] {
					t.Fatalf("过滤结果 = %v, want %v", names, tt.want)
				}
			}
		})
	}
}
