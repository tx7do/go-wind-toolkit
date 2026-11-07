package internal

// NewGoSchema 返回 Go 源码数据源(ent:// 与 gorm://)的转换器:
// schemasource 的内存 Inspector 已把 ent schema / gorm model 目录还原为
// atlas 表,类型文本为 MySQL 风格,先 MySQL 后 PostgreSQL 回退。
func NewGoSchema(i *ConvertOptions) (SchemaConverter, error) {
	return &converterFunc{opts: i, fnc: goSchemaFieldType}, nil
}

// goSchemaFieldType 依次尝试 MySQL/PostgreSQL 映射;源码解析产物的类型文本为 MySQL 风格。
func goSchemaFieldType(sqlType string) string {
	if f := MySQLFieldType(sqlType); f != "" {
		return f
	}
	return PostgresFieldType(sqlType)
}
