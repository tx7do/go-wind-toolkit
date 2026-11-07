package schemasource

import (
	"ariga.io/atlas/sql/schema"
)

// Note: at this moment ent doesn't support fields on m2m relations.
func IsJoinTable(table *schema.Table) bool {
	if table.PrimaryKey == nil || len(table.PrimaryKey.Parts) != 2 || len(table.ForeignKeys) != 2 {
		return false
	}
	// Make sure that the foreign key columns exactly match primary key column.
	for _, fk := range table.ForeignKeys {
		if len(fk.Columns) != 1 {
			return false
		}
		if fk.Columns[0] != table.PrimaryKey.Parts[0].C && fk.Columns[0] != table.PrimaryKey.Parts[1].C {
			return false
		}
	}
	return true
}

// FilterTables 按 include/exclude 过滤表清单(表名精确匹配):include 非空 =
// 白名单;exclude 非空 = 黑名单;两者同时给定时先白后黑。原始顺序保持,
// include/exclude 为空时原样返回。
//
// 供两条生成管线的方言适配器在 InspectSchema 之后统一过滤。注意 mysql/
// postgres 走 atlas InspectOptions.Tables 在 inspect 期过滤,与这里的
// 精确匹配语义一致,叠加调用幂等;但**不要**把同一份 include 同时交给
// atlas 与本函数以外的大小写不敏感实现,以免语义漂移。
func FilterTables(tables []*schema.Table, include, exclude []string) []*schema.Table {
	if len(include) == 0 && len(exclude) == 0 {
		return tables
	}
	included := make(map[string]bool, len(include))
	for _, n := range include {
		included[n] = true
	}
	excluded := make(map[string]bool, len(exclude))
	for _, n := range exclude {
		excluded[n] = true
	}
	filtered := make([]*schema.Table, 0, len(tables))
	for _, t := range tables {
		if len(included) > 0 && !included[t.Name] {
			continue
		}
		if excluded[t.Name] {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}
