package schemasource

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ariga.io/atlas/sql/schema"
)

func TestNormalizeDSN(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "dump.sql")
	if err := os.WriteFile(existing, []byte("CREATE TABLE a (id INT);\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name string
		dsn  string
		want string
	}{
		{name: "scheme passthrough", dsn: "mysql://user:pass@tcp(localhost:3306)/db", want: "mysql://user:pass@tcp(localhost:3306)/db"},
		{name: "existing file gains file scheme", dsn: existing, want: "file://" + existing},
		{name: "inline sql text gains text scheme", dsn: "CREATE TABLE a (id INT);", want: "text://CREATE TABLE a (id INT);"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeDSN(tt.dsn); got != tt.want {
				t.Fatalf("NormalizeDSN(%q) = %q; want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

func TestLoadSQLFromFile(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "dump.sql")
	const content = "CREATE TABLE a (id INT);\n"
	if err := os.WriteFile(existing, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name string
		path string
		want string
	}{
		{name: "existing file yields its content", path: existing, want: content},
		// 生产路径给的是 NormalizeDSN 之后的值:裸文件路径会被补成 file://,
		// 所以判文件之前必须先剥 scheme,否则路径本身会被当成 SQL 文本。
		{name: "file scheme yields the file content", path: "file://" + existing, want: content},
		{name: "text scheme is stripped", path: "text://CREATE TABLE a (id INT);", want: "CREATE TABLE a (id INT);"},
		{name: "inline sql without scheme is returned as-is", path: "CREATE TABLE a (id INT);", want: "CREATE TABLE a (id INT);"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := LoadSQLFromFile(tt.path); got != tt.want {
				t.Fatalf("LoadSQLFromFile(%q) = %q; want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestParseType(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want any
	}{
		{raw: "int", want: &schema.IntegerType{}},
		{raw: "varchar(100)", want: &schema.StringType{}},
		{raw: "wtf$%$#", want: &schema.UnsupportedType{}},
	} {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseType(tt.raw)
			if err != nil {
				t.Fatalf("ParseType(%q): %v", tt.raw, err)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Fatalf("ParseType(%q) = %T; want %T", tt.raw, got, tt.want)
			}
		})
	}
}
