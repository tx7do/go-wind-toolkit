package internal

// Exporter 远程配置导入器
type Exporter interface {
	// Export 导入所有的配置
	Export() error

	// ExportOneService 导入单个配置
	ExportOneService(app string) error
}
