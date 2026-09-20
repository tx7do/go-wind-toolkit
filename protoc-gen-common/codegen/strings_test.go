package codegen

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestIndent(t *testing.T) {
	for _, tt := range []struct {
		n    int
		want string
	}{
		{n: 0, want: ""},
		{n: 1, want: "  "},
		{n: 3, want: "      "},
	} {
		if got := Indent(tt.n); got != tt.want {
			t.Fatalf("Indent(%d) = %q (len %d); want %q (len %d)", tt.n, got, len(got), tt.want, len(tt.want))
		}
	}
}

func TestLowerFirst(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "a", want: "a"},
		{in: "Abc", want: "abc"},
		{in: "ABC", want: "aBC"},
		{in: "already", want: "already"},
		{in: "1num", want: "1num"},
	} {
		if got := LowerFirst(tt.in); got != tt.want {
			t.Fatalf("LowerFirst(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestLocaleCompare(t *testing.T) {
	// 排序键仿真 JS localeCompare(UCA):标点 < 数字 < 字母,字母不区分大小写。
	for _, tt := range []struct {
		a, b string
		want bool // a 排在 b 之前
	}{
		{a: "a", b: "b", want: true},
		{a: "b", b: "a", want: false},
		{a: "a", b: "a", want: false}, // 严格小于,相等为 false
		{a: "apple", b: "Zebra", want: true},
		{a: "Zebra", b: "apple", want: false},
		{a: "_id", b: "id", want: true},          // 下划线先于字母
		{a: "userId", b: "user_id", want: false}, // 下划线先于字母:user_id 在前
		{a: "v1", b: "va", want: true},           // 数字先于字母
		{a: "a1", b: "aa", want: true},           // 位内数字先于字母
	} {
		if got := LocaleCompare(tt.a, tt.b); got != tt.want {
			t.Errorf("LocaleCompare(%q, %q) = %v; want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// proto 的 LeadingComments 里段落之间是一个空串元素。两侧生成器原先直接 continue
// 掉它,段落分隔就此丢失(生成的 dart/TS 文档把两段并成一段);而更早的基线又写成
// "/// " 带尾随空格。目标形态:空行保留成裸标记,且任何一行都不带尾随空格。
func TestCommentLines(t *testing.T) {
	for _, tt := range []struct {
		name   string
		in     string
		marker string
		want   []string
	}{
		{
			name:   "interior blank line keeps the paragraph break",
			in:     "Get a single log entry.\n\nUnary request/response.\n",
			marker: "//",
			want:   []string{"// Get a single log entry.", "//", "// Unary request/response."},
		},
		{
			name:   "trailing blank line from protoc is dropped",
			in:     "Only one paragraph.\n",
			marker: "///",
			want:   []string{"/// Only one paragraph."},
		},
		{
			name:   "leading blank lines are dropped",
			in:     "\nText.\n",
			marker: "//",
			want:   []string{"// Text."},
		},
		{
			name:   "whitespace-only line counts as blank",
			in:     "A.\n \nB.\n",
			marker: "//",
			want:   []string{"// A.", "//", "// B."},
		},
		{
			name:   "all blank yields no lines",
			in:     "\n\n",
			marker: "//",
			want:   []string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := CommentLines(strings.Split(tt.in, "\n"), tt.marker)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommentLines(%q, %q) = %q; want %q", tt.in, tt.marker, got, tt.want)
			}
			for _, line := range got {
				if strings.TrimRight(line, " ") != line {
					t.Errorf("CommentLines emitted a trailing-space line: %q", line)
				}
			}
		})
	}
}

func TestSortKeyOrdering(t *testing.T) {
	// 一组混合标识符按 sortKey 升序应得到标点 < 数字 < 字母、大小写不敏感
	// 的确定顺序——这是两侧生成器字段/枚举排序稳定性的依据。
	got := []string{"User", "_meta", "id2", "ID1", "zz"}
	sort.SliceStable(got, func(i, j int) bool { return LocaleCompare(got[i], got[j]) })
	want := []string{"_meta", "ID1", "id2", "User", "zz"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted = %v; want %v", got, want)
		}
	}
}
