package internal

import (
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto/internal/mux"
)

// TestNewTextConvert tests the creation of a Text converter
func TestNewTextConvert(t *testing.T) {
	opts := &ConvertOptions{
		schemaPath: "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100))",
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Errorf("failed to create Text converter: %v", err)
	}

	if converter == nil {
		t.Error("converter should not be nil")
	}
}
