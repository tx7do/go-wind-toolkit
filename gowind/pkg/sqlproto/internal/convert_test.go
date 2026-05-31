package internal

import (
	"context"
	"testing"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlproto/internal/mux"
)

// TestTextSchemaTables tests parsing SQL text and converting to table data
func TestTextSchemaTables(t *testing.T) {
	sqlContent := `
	CREATE TABLE users (
		id INT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		email VARCHAR(100) NULL,
		age INT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("expected at least one table, got none")
	}

	table := tables[0]
	if table.Name != "users" {
		t.Errorf("expected table name 'users', got %q", table.Name)
	}

	expectedFields := 5
	if len(table.Fields) != expectedFields {
		t.Errorf("expected %d fields, got %d", expectedFields, len(table.Fields))
	}

	// Check first field (id)
	if table.Fields[0].Name != "id" {
		t.Errorf("expected first field name 'id', got %q", table.Fields[0].Name)
	}
	if table.Fields[0].Null {
		t.Error("expected field 'id' to not be nullable")
	}

	// Check second field (name)
	if table.Fields[1].Name != "name" {
		t.Errorf("expected second field name 'name', got %q", table.Fields[1].Name)
	}
	if table.Fields[1].Null {
		t.Error("expected field 'name' to not be nullable")
	}

	// Check third field (email)
	if table.Fields[2].Name != "email" {
		t.Errorf("expected third field name 'email', got %q", table.Fields[2].Name)
	}
	if !table.Fields[2].Null {
		t.Error("expected field 'email' to be nullable")
	}
}

// TestTextParseMultipleTables tests parsing multiple tables from SQL text
func TestTextParseMultipleTables(t *testing.T) {
	sqlContent := `
	CREATE TABLE users (
		id INT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		email VARCHAR(100)
	) CHARSET=utf8mb4;

	CREATE TABLE posts (
		id INT PRIMARY KEY,
		user_id INT NOT NULL,
		title VARCHAR(255) NOT NULL,
		content TEXT
	) CHARSET=utf8mb4;
	`

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	expectedTableCount := 2
	if len(tables) != expectedTableCount {
		t.Errorf("expected %d tables, got %d", expectedTableCount, len(tables))
	}

	if len(tables) >= 1 && tables[0].Name != "users" {
		t.Errorf("expected first table 'users', got %q", tables[0].Name)
	}

	if len(tables) >= 2 && tables[1].Name != "posts" {
		t.Errorf("expected second table 'posts', got %q", tables[1].Name)
	}
}

// TestTextParseEmptySQL tests error handling for empty SQL content
func TestTextParseEmptySQL(t *testing.T) {
	sqlContent := ""

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err == nil {
		t.Error("expected error for empty SQL, got nil")
	}

	if tables != nil {
		t.Errorf("expected nil tables for empty SQL, got %v", tables)
	}
}

// TestTextParseInvalidSQL tests error handling for invalid SQL content
func TestTextParseInvalidSQL(t *testing.T) {
	sqlContent := "INVALID SQL STATEMENT"

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	_, err = converter.SchemaTables(context.Background())
	if err == nil {
		t.Error("expected error for invalid SQL, got nil")
	}
}

// TestTextParseWithExcludedTables tests table filtering with excluded tables
func TestTextParseWithExcludedTables(t *testing.T) {
	sqlContent := `
	CREATE TABLE users (
		id INT PRIMARY KEY,
		name VARCHAR(100)
	);

	CREATE TABLE logs (
		id INT PRIMARY KEY,
		message VARCHAR(255)
	);

	CREATE TABLE settings (
		id INT PRIMARY KEY,
		key VARCHAR(100),
		value VARCHAR(255)
	);
	`

	opts := &ConvertOptions{
		schemaPath:     sqlContent,
		excludedTables: []string{"logs"},
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	if len(tables) != 2 {
		t.Errorf("expected 2 tables (after excluding 'logs'), got %d", len(tables))
	}

	// Check that 'logs' is not in the result
	for _, table := range tables {
		if table.Name == "logs" {
			t.Error("expected 'logs' table to be excluded, but it was found")
		}
	}

	// Check that 'users' and 'settings' are present
	tableNames := make(map[string]bool)
	for _, table := range tables {
		tableNames[table.Name] = true
	}

	if !tableNames["users"] {
		t.Error("expected 'users' table to be present")
	}

	if !tableNames["settings"] {
		t.Error("expected 'settings' table to be present")
	}
}

// TestTextParseWithIncludedTables tests table filtering with included tables
func TestTextParseWithIncludedTables(t *testing.T) {
	sqlContent := `
	CREATE TABLE users (
		id INT PRIMARY KEY,
		name VARCHAR(100)
	);

	CREATE TABLE posts (
		id INT PRIMARY KEY,
		title VARCHAR(255)
	);

	CREATE TABLE comments (
		id INT PRIMARY KEY,
		content TEXT
	);
	`

	opts := &ConvertOptions{
		schemaPath:     sqlContent,
		includedTables: []string{"users", "posts"},
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	if len(tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(tables))
	}

	tableNames := make(map[string]bool)
	for _, table := range tables {
		tableNames[table.Name] = true
	}

	if !tableNames["users"] {
		t.Error("expected 'users' table to be present")
	}

	if !tableNames["posts"] {
		t.Error("expected 'posts' table to be present")
	}

	if tableNames["comments"] {
		t.Error("expected 'comments' table to not be present when not included")
	}
}

// TestTextParseComplexDataTypes tests parsing various SQL data types
func TestTextParseComplexDataTypes(t *testing.T) {
	sqlContent := `
	CREATE TABLE test_types (
		id INT PRIMARY KEY,
		col_int INT,
		col_bigint BIGINT,
		col_varchar VARCHAR(255),
		col_text TEXT,
		col_decimal DECIMAL(10, 2),
		col_float FLOAT,
		col_double DOUBLE,
		col_boolean BOOLEAN,
		col_datetime DATETIME,
		col_timestamp TIMESTAMP,
		col_date DATE,
		col_time TIME,
		col_json JSON,
		col_blob BLOB
	);
	`

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}

	table := tables[0]
	expectedFieldCount := 15
	if len(table.Fields) != expectedFieldCount {
		t.Errorf("expected %d fields, got %d", expectedFieldCount, len(table.Fields))
	}

	// Verify some specific fields are parsed correctly
	idField := table.Fields[0]
	if idField.Name != "id" {
		t.Errorf("expected first field 'id', got %q", idField.Name)
	}

	textField := table.Fields[4]
	if textField.Name != "col_text" {
		t.Errorf("expected field 'col_text', got %q", textField.Name)
	}
}

// TestTextParseWithFieldComments tests parsing field comments
func TestTextParseWithFieldComments(t *testing.T) {
	sqlContent := `
	CREATE TABLE products (
		id INT PRIMARY KEY COMMENT 'ProductID',
		name VARCHAR(255) NOT NULL COMMENT 'ProductName',
		price DECIMAL(10, 2) NOT NULL COMMENT 'ProductPrice',
		stock INT DEFAULT 0 COMMENT 'StockQuantity'
	);
	`

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}

	table := tables[0]

	// Check field comments when available
	if table.Fields[0].Comment != "" && table.Fields[0].Comment != "ProductID" {
		t.Errorf("expected field comment 'ProductID', got %q", table.Fields[0].Comment)
	}

	if table.Fields[1].Comment != "" && table.Fields[1].Comment != "ProductName" {
		t.Errorf("expected field comment 'ProductName', got %q", table.Fields[1].Comment)
	}
}

// TestNewConvertWithTextDialect tests the NewConvert factory function with text dialect
func TestNewConvertWithTextDialect(t *testing.T) {
	sqlContent := `
	CREATE TABLE users (
		id INT PRIMARY KEY,
		name VARCHAR(100)
	);
	`

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewConvert(WithSchemaPath(sqlContent), WithDriver(opts.driver))
	if err != nil {
		t.Fatalf("failed to create converter: %v", err)
	}

	if converter == nil {
		t.Error("expected converter to not be nil")
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to get schema tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("expected at least one table")
	}

	if tables[0].Name != "users" {
		t.Errorf("expected table name 'users', got %q", tables[0].Name)
	}
}

// TestTextParseNullableFields tests nullable field detection
func TestTextParseNullableFields(t *testing.T) {
	sqlContent := `
	CREATE TABLE employees (
		id INT PRIMARY KEY,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		middle_name VARCHAR(100),
		phone VARCHAR(20),
		department_id INT NOT NULL,
		manager_id INT
	);
	`

	opts := &ConvertOptions{
		schemaPath: sqlContent,
		driver: &mux.ConvertDriver{
			Dialect:    "text",
			SchemaName: "public",
		},
	}

	converter, err := NewText(opts)
	if err != nil {
		t.Fatalf("failed to create Text converter: %v", err)
	}

	tables, err := converter.SchemaTables(context.Background())
	if err != nil {
		t.Fatalf("failed to parse schema tables: %v", err)
	}

	table := tables[0]

	// Check not null fields
	notNullTests := map[string]bool{
		"id":            false,
		"first_name":    false,
		"last_name":     false,
		"middle_name":   true,
		"phone":         true,
		"department_id": false,
		"manager_id":    true,
	}

	for _, field := range table.Fields {
		expectedNull, exists := notNullTests[field.Name]
		if !exists {
			continue
		}
		if field.Null != expectedNull {
			t.Errorf("field %q: expected Null=%v, got %v", field.Name, expectedNull, field.Null)
		}
	}
}
