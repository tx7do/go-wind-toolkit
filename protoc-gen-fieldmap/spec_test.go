package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempSpec(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "s.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const validOptionSpec = `
output:
  file: x/m.gen.go
  package: x
imports:
  up: example.com/up
message:
  proto: example/v1/store.proto
  type: Store.SQL
  func: Map
  target_type: up.Option
  fields:
    driver: { target: WithDriver }
`

func TestLoadSpecsValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(string) string
		wantErr string // 空 = 期望通过
	}{
		{"valid", func(s string) string { return s }, ""},
		{"missing output.file", func(s string) string {
			return strings.Replace(s, "  file: x/m.gen.go\n", "", 1)
		}, "output.file is required"},
		{"bad file suffix", func(s string) string {
			return strings.Replace(s, "m.gen.go", "m.go.txt", 1)
		}, "must end with .go"},
		{"missing package", func(s string) string {
			return strings.Replace(s, "  package: x\n", "", 1)
		}, "output.package is required"},
		{"missing func", func(s string) string {
			return strings.Replace(s, "  func: Map\n", "", 1)
		}, "message.func is required"},
		{"bad style", func(s string) string {
			return strings.Replace(s, "  target_type: up.Option", "  style: weird\n  target_type: up.Option", 1)
		}, `message.style "weird"`},
		{"undeclared type qualifier", func(s string) string {
			return strings.Replace(s, "target_type: up.Option", "target_type: up2.Option", 1)
		}, "not declared in imports"},
		{"unknown guard", func(s string) string {
			return strings.Replace(s, "target: WithDriver }", "target: WithDriver, guard: weird }", 1)
		}, `guard "weird"`},
		{"unknown cast", func(s string) string {
			return strings.Replace(s, "target: WithDriver }", "target: WithDriver, cast: pointer }", 1)
		}, `cast "pointer": unsupported`},
		{"undeclared field target qualifier", func(s string) string {
			return strings.Replace(s, "target: WithDriver }", "target: up2.WithDriver }", 1)
		}, "not declared in imports"},
		{"struct style with guard", func(s string) string {
			s = strings.Replace(s, "  target_type: up.Option", "  style: struct\n  target_type: up.Cfg", 1)
			return strings.Replace(s, "driver: { target: WithDriver }", "driver: { target: Driver, guard: empty }", 1)
		}, "struct style does not support guard"},
		{"struct style default target ok", func(s string) string {
			s = strings.Replace(s, "  target_type: up.Option", "  style: struct\n  target_type: up.Cfg", 1)
			return strings.Replace(s, "driver: { target: WithDriver }", "driver: {}", 1)
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTempSpec(t, tc.mutate(validOptionSpec))
			_, err := LoadSpecs([]string{dir})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want ok, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want error %q, got ok", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestCamelTarget(t *testing.T) {
	cases := map[string]string{
		"driver":               "Driver",
		"max_idle_connections": "MaxIdleConnections",
		"dial_timeout_ms":      "DialTimeoutMs",
		"_leading":             "Leading",
		"":                     "",
	}
	for in, want := range cases {
		if got := camelTarget(in); got != want {
			t.Errorf("camelTarget(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultGuardDerivation(t *testing.T) {
	cases := []struct {
		name  string
		shape string
		calls [5]bool // isString, isBool, isNumeric, isEnum, isMessage
		want  string
	}{
		{"string", "singular", [5]bool{true, false, false, false, false}, guardEmpty},
		{"bool", "singular", [5]bool{false, true, false, false, false}, guardNone},
		{"numeric", "singular", [5]bool{false, false, true, false, false}, guardZero},
		{"enum", "singular", [5]bool{false, false, false, true, false}, guardZero},
		{"message", "singular", [5]bool{false, false, false, false, true}, guardNotNil},
		{"list", "list", [5]bool{}, guardLen},
		{"map", "map", [5]bool{}, guardLen},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultGuard(tc.shape, tc.calls[0], tc.calls[1], tc.calls[2], tc.calls[3], tc.calls[4])
			if got != tc.want {
				t.Errorf("defaultGuard(%q,...) = %q, want %q", tc.shape, got, tc.want)
			}
		})
	}
}
