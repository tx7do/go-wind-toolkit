package codegen

import "strings"

// Indent returns n levels of indentation for generated code.
func Indent(n int) string {
	return strings.Repeat("  ", n)
}

// CommentLines renders proto doc-comment source lines as commented lines.
// Interior blank lines become a bare marker so paragraph structure survives;
// leading and trailing ones are dropped (LeadingComments ends with "\n").
// No returned line carries trailing whitespace.
func CommentLines(lines []string, marker string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if text := strings.TrimSpace(line); text != "" {
			out = append(out, marker+" "+text)
			continue
		}
		out = append(out, marker)
	}
	return out
}

// LowerFirst returns s with its first character lowercased.
func LowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	b := []byte(s)
	if b[0] >= 'A' && b[0] <= 'Z' {
		b[0] += 32
	}
	return string(b)
}

// LocaleCompare compares two strings in a way that matches JavaScript's
// String.prototype.localeCompare for the common cases encountered in generated
// code. The Unicode Collation Algorithm (used by localeCompare) sorts
// punctuation/symbols before digits before letters, and is case-insensitive.
// Go's native string comparison treats punctuation (e.g. underscore 95, '{'
// 123) as greater than letters (65-122), which conflicts with the ordering the
// generated output is normalized to. sortKey transforms each character so that
// Go byte comparison matches the UCA ordering.
func LocaleCompare(a, b string) bool {
	return sortKey(a) < sortKey(b)
}

// sortKey transforms a string into a comparison key where:
//   - Non-alphanumeric characters (punctuation, symbols) sort before alphanumerics
//   - Digits sort after punctuation but before letters
//   - Letters are compared case-insensitively (lowercase form used)
func sortKey(s string) string {
	var sb strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z':
			sb.WriteByte(2)
			sb.WriteByte(c)
		case c >= 'A' && c <= 'Z':
			sb.WriteByte(2)
			sb.WriteByte(c + 32)
		case c >= '0' && c <= '9':
			sb.WriteByte(1)
			sb.WriteByte(c)
		default:
			sb.WriteByte(0)
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
