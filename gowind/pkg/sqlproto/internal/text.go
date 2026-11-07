package internal



// NewText 返回 DDL 文本驱动 sink 的转换器。
func NewText(i *ConvertOptions) (SchemaConverter, error) {
	return &converterFunc{opts: i, fnc: textFieldType}, nil
}

// textFieldType 依次尝试 MySQL/PostgreSQL 映射;DDL 文本可能来自任一方言。
func textFieldType(sqlType string) (f string) {
	if f = MySQLFieldType(sqlType); f != "" {
		return
	}

	f = PostgresFieldType(sqlType)

	return
}
