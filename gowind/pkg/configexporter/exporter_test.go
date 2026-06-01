package configexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==============================
// NewExporter — nil 返回值测试（Apollo/Kubernetes/Polaris 未实现，返回 nil）
// ==============================

func TestNewExporter_Apollo(t *testing.T) {
	result := NewExporter("apollo", "localhost:8080", "test", ".", "", "", "", false)
	assert.Nil(t, result)
}

func TestNewExporter_Kubernetes(t *testing.T) {
	result := NewExporter("kubernetes", "localhost:6443", "test", ".", "", "", "", false)
	assert.Nil(t, result)
}

func TestNewExporter_Polaris(t *testing.T) {
	result := NewExporter("polaris", "localhost:8091", "test", ".", "", "", "", false)
	assert.Nil(t, result)
}

// ==============================
// Export — nil exporter 应报错
// ==============================

func TestExport_NilExporter(t *testing.T) {
	err := Export("apollo", "localhost:8080", "test", ".", "", "", "", false)
	assert.NotNil(t, err)
	assert.Equal(t, "exporter is nil", err.Error())
}
