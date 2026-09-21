package frontendgen

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 与 gowind-uiapp TS 生成器（golden-dump.mts）使用完全相同的固定选项生成黄金样本。
var goldenModulePathMap = map[string]string{
	"role":         "permission/role",
	"dict-type":    "system/dict",
	"org-unit":     "opm/org_unit",
	"api-audit-log": "log/api_audit_log",
}

var goldenRouterModules = []RouterModuleConfig{
	{ModuleKey: "permission", ModuleDisplayName: "权限管理", ServiceTags: []string{"RoleService"}},
	{ModuleKey: "system", ModuleDisplayName: "系统管理", ServiceTags: []string{"DictTypeService"}},
	{ModuleKey: "opm", ModuleDisplayName: "组织人员", ServiceTags: []string{"OrgUnitService"}},
	{ModuleKey: "log", ModuleDisplayName: "日志审计", ServiceTags: []string{"ApiAuditLogService"}},
}

func goldenOptions(fw Framework) Options {
	return Options{
		Framework:     fw,
		ServiceName:   "admin",
		ModulePathMap: goldenModulePathMap,
		RouterModules: goldenRouterModules,
	}
}

func loadTestSpec(t *testing.T) *Spec {
	t.Helper()
	data, err := os.ReadFile("testdata/openapi.yaml")
	if err != nil {
		t.Fatalf("读取测试规格失败: %v", err)
	}
	spec, err := ParseOpenAPIYAML(data)
	if err != nil {
		t.Fatalf("解析测试规格失败: %v", err)
	}
	return spec
}

// TestGolden 三框架产物与 TS 生成器固化的黄金样本逐文件比对
func TestGolden(t *testing.T) {
	spec := loadTestSpec(t)

	for _, fw := range []Framework{FrameworkVueVben, FrameworkVueElement, FrameworkReactAntd} {
		t.Run(string(fw), func(t *testing.T) {
			opts := goldenOptions(fw)
			opts.Spec = spec
			files, err := Generate(opts)
			if err != nil {
				t.Fatalf("生成失败: %v", err)
			}
			if len(files) == 0 {
				t.Fatal("没有生成任何文件")
			}

			generated := map[string]bool{}
			for _, file := range files {
				generated[file.Path] = true
				goldenPath := filepath.Join("testdata", "golden", string(fw), filepath.FromSlash(file.Path))
				golden, err := os.ReadFile(goldenPath)
				if err != nil {
					t.Errorf("缺少黄金样本 %s: %v", file.Path, err)
					continue
				}

				got := normalizeNewlines(file.Content)
				want := normalizeNewlines(string(golden))
				if got != want {
					t.Errorf("文件 %s 与黄金样本不一致\n--- got ---\n%s\n--- want ---\n%s",
						file.Path, truncate(got), truncate(want))
					continue
				}

				// JSON 文件再做语义级比对（键序无关）
				if strings.HasSuffix(file.Path, ".json") && !strings.HasPrefix(strings.TrimSpace(file.Content), `"`) {
					assertJSONEqual(t, file.Path, got, want)
				}
			}

			// 反向检查：黄金样本中没有多余文件
			goldenRoot := filepath.Join("testdata", "golden", string(fw))
			err = filepath.WalkDir(goldenRoot, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, err := filepath.Rel(goldenRoot, path)
				if err != nil {
					return err
				}
				relSlash := filepath.ToSlash(rel)
				if !generated[relSlash] {
					t.Errorf("黄金样本 %s 未被生成（多余样本）", relSlash)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("遍历黄金样本失败: %v", err)
			}
		})
	}
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func truncate(s string) string {
	if len(s) > 3000 {
		return s[:3000] + "\n... (截断)"
	}
	return s
}

// assertJSONEqual 语义级 JSON 比对（忽略键序与格式）
func assertJSONEqual(t *testing.T, path, got, want string) {
	t.Helper()
	var gotV, wantV any
	if err := json.Unmarshal([]byte(got), &gotV); err != nil {
		t.Errorf("文件 %s 生成内容不是合法 JSON: %v", path, err)
		return
	}
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		// 黄金样本可能是合并式片段（无外层大括号），跳过语义比对
		return
	}
	gotJSON, _ := json.Marshal(gotV)
	wantJSON, _ := json.Marshal(wantV)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Errorf("文件 %s JSON 语义不一致", path)
	}
}

// TestParseAndExtract 验证服务提取的关键推导（类型前缀、getter、文件名、字段序）
func TestParseAndExtract(t *testing.T) {
	spec := loadTestSpec(t)
	services := ExtractServices(spec)

	if len(services) != 4 {
		t.Fatalf("期望 4 个服务，实际 %d", len(services))
	}

	expect := []struct {
		tag, prefix, getter, kebab string
	}{
		{"ApiAuditLogService", "auditservicev1", "apiAuditLogService", "api-audit-log"},
		{"DictTypeService", "dictservicev1", "dictTypeService", "dict-type"},
		{"OrgUnitService", "identityservicev1", "orgUnitService", "org-unit"},
		{"RoleService", "permissionservicev1", "roleService", "role"},
	}
	for i, e := range expect {
		svc := services[i]
		if svc.TagName != e.tag {
			t.Errorf("服务 %d 期望 %s 实际 %s", i, e.tag, svc.TagName)
			continue
		}
		if svc.TypePrefix != e.prefix {
			t.Errorf("%s TypePrefix 期望 %s 实际 %s", e.tag, e.prefix, svc.TypePrefix)
		}
		if svc.ClientGetterName != e.getter {
			t.Errorf("%s ClientGetterName 期望 %s 实际 %s", e.tag, e.getter, svc.ClientGetterName)
		}
		if svc.KebabName != e.kebab {
			t.Errorf("%s KebabName 期望 %s 实际 %s", e.tag, e.kebab, svc.KebabName)
		}
	}

	// Role 服务字段按文档序（字典序书写）
	role := services[3]
	var fieldNames []string
	for _, f := range role.Fields {
		fieldNames = append(fieldNames, f.Name)
	}
	wantFields := "code,description,isProtected,name,permissions,sortOrder,status,tenantName"
	if strings.Join(fieldNames, ",") != wantFields {
		t.Errorf("Role 字段序期望 %s 实际 %s", wantFields, strings.Join(fieldNames, ","))
	}

	// ApiAuditLog 仅 List/Get
	audit := services[0]
	paths := GetCrudPaths(audit)
	if paths.List == nil || paths.Get == nil || paths.Create != nil || paths.Update != nil || paths.Delete != nil {
		t.Errorf("ApiAuditLogService CRUD 检测错误: %+v", paths)
	}
}

// TestDetectRouterModules 验证路由模块自动检测
func TestDetectRouterModules(t *testing.T) {
	spec := loadTestSpec(t)
	services := ExtractServices(spec)
	modules := DetectRouterModules(services)

	var keys []string
	for _, m := range modules {
		keys = append(keys, m.ModuleKey)
	}
	// api-audit-logs -> api；dict-types -> system；org-units -> opm；roles -> roles（启发式）
	want := "api,system,opm,roles"
	if strings.Join(keys, ",") != want {
		t.Errorf("模块分组期望 %s 实际 %s", want, strings.Join(keys, ","))
	}
}

// TestWriteFilesMerge 验证 vben 国际化片段合并语义
func TestWriteFilesMerge(t *testing.T) {
	dir := t.TempDir()

	// 预置已有 page.json（含 user 键与既有 role 键）
	existing := "{\n  \"user\": {\n    \"moduleName\": \"用户\"\n  },\n  \"role\": {\n    \"moduleName\": \"旧角色\"\n  }\n}\n"
	localeDir := filepath.Join(dir, "locales", "langs", "zh-CN")
	if err := os.MkdirAll(localeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localeDir, "page.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	fragment := "  \"role\": {\n    \"moduleName\": \"角色\"\n  }"
	files := []GeneratedFile{
		{Path: "locales/langs/zh-CN/page.role.json", Content: fragment},
		{Path: "locales/langs/zh-CN/menu.permission.json", Content: "  \"permission\": {\n    \"moduleName\": \"权限管理\"\n  }"},
		{Path: "api/composables/role.ts", Content: "export {};\n"},
	}

	results, err := WriteFiles(files, dir)
	if err != nil {
		t.Fatalf("写盘失败: %v", err)
	}

	byAction := map[string]string{}
	for _, r := range results {
		byAction[r.Action] = r.Path
	}
	if _, ok := byAction[ActionMerged]; !ok {
		t.Errorf("期望存在 merged 动作，实际 %+v", results)
	}
	if _, ok := byAction[ActionMergedNew]; !ok {
		t.Errorf("期望存在 merged-new 动作，实际 %+v", results)
	}
	if _, ok := byAction[ActionCreated]; !ok {
		t.Errorf("期望存在 created 动作，实际 %+v", results)
	}

	// 合并后的 page.json 必须是合法 JSON，且 role 键被替换
	merged, err := os.ReadFile(filepath.Join(localeDir, "page.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(merged, &m); err != nil {
		t.Fatalf("合并后 page.json 非法 JSON: %v\n%s", err, merged)
	}
	role, _ := m["role"].(map[string]any)
	if role["moduleName"] != "角色" {
		t.Errorf("role 键未替换为片段值: %v", role)
	}
	if _, ok := m["user"]; !ok {
		t.Error("合并丢失了既有的 user 键")
	}

	// menu 片段新建为独立合法 JSON
	menuNew, err := os.ReadFile(filepath.Join(localeDir, "menu.permission.json"))
	if err != nil {
		t.Fatal(err)
	}
	var mv map[string]any
	if err := json.Unmarshal(menuNew, &mv); err != nil {
		t.Fatalf("新建 menu 片段非法 JSON: %v\n%s", err, menuNew)
	}
}

// TestOrderedMapRendering 验证有序 JSON 渲染与 JS JSON.stringify 一致
func TestOrderedMapRendering(t *testing.T) {
	m := NewOrderedMap().
		Set("pageTitle", "字典类型管理管理").
		Set("moduleName", "字典类型管理").
		Set("serial", "序号").
		Set("statusMap", NewOrderedMap().Set("ON", "启用").Set("OFF", "禁用"))

	want := `{
  "pageTitle": "字典类型管理管理",
  "moduleName": "字典类型管理",
  "serial": "序号",
  "statusMap": {
    "ON": "启用",
    "OFF": "禁用"
  }
}`
	if m.String() != want {
		t.Errorf("渲染不一致:\ngot:\n%s\nwant:\n%s", m.String(), want)
	}
}

// 生成产物里对枚举选项常量的引用形态:JSX 的 options={<ident>(…)}、
// Vue 模板的 v-for="item in <ident>"。
var (
	reOptionsRef = regexp.MustCompile(`options[=:]\s*\{?\s*([A-Za-z_$][\w$]*)\s*[(,}]`)
	reVueForRef  = regexp.MustCompile(`v-for="[^"]*\bin\s+([A-Za-z_$][\w$]*)`)

	reImportBraces = regexp.MustCompile(`(?s)import\s+(?:type\s+)?\{([^}]*)\}`)
	reDeclStmt     = regexp.MustCompile(`(?:export\s+)?(?:const|let|var|function|class)\s+([A-Za-z_$][\w$]*)`)
	reAnyIdent     = regexp.MustCompile(`[A-Za-z_$][\w$]*`)
)

// fileDeclaredIdents 一个文件里可用(已声明或已导入)的标识符集合
func fileDeclaredIdents(src string) map[string]bool {
	declared := map[string]bool{}
	for _, group := range reImportBraces.FindAllStringSubmatch(src, -1) {
		for _, id := range reAnyIdent.FindAllString(group[1], -1) {
			declared[id] = true
		}
	}
	for _, group := range reDeclStmt.FindAllStringSubmatch(src, -1) {
		declared[group[1]] = true
	}
	return declared
}

// TestGeneratedEnumSymbolsAreDeclared 三框架产物中,options / v-for 引用到的枚举常量
// 必须在同一文件里被声明或导入。此前非 status 枚举(requestMethod)只写了引用没写声明,
// 抽屉一渲染就 ReferenceError,而逐字比对的黄金样本把这份坏产物当成了正确基线。
func TestGeneratedEnumSymbolsAreDeclared(t *testing.T) {
	refs := 0
	eachGeneratedFile(t, func(fw Framework, f GeneratedFile) {
		if !isSourceFile(f.Path) {
			return
		}
		declared := fileDeclaredIdents(f.Content)
		refsInFile := append(
			reOptionsRef.FindAllStringSubmatch(f.Content, -1),
			reVueForRef.FindAllStringSubmatch(f.Content, -1)...,
		)
		for _, group := range refsInFile {
			id := group[1]
			refs++
			if !declared[id] {
				t.Errorf("%s/%s 引用了未声明的标识符 %q", fw, f.Path, id)
			}
		}
	})

	// 夹具必须真的覆盖到这些腿,否则本用例退化成空跑
	if refs < 3 {
		t.Errorf("仅检查到 %d 处枚举选项引用,夹具可能已不再覆盖 select/v-for 腿", refs)
	}
}

// TestEnumSymbolNamingIsShared 模板引用与声明必须取自同一个命名函数,
// 否则「引用了没人声明的常量」这类缺陷会再次出现。
func TestEnumSymbolNamingIsShared(t *testing.T) {
	field := &ParsedField{Name: "requestMethod", IsEnum: true, EnumValues: []string{"GET", "POST", "PUT", "DELETE"}}

	if got, want := elementEnumListVar(field), "requestMethodList"; got != want {
		t.Errorf("elementEnumListVar = %q, 期望 %q", got, want)
	}
	if got, want := reactEnumOptionsFn(field), "getRequestMethodOptions"; got != want {
		t.Errorf("reactEnumOptionsFn = %q, 期望 %q", got, want)
	}

	// element 抽屉:模板里的 v-for 与脚本里的 const 声明必须同名
	drawer := elementDrawerCode(mustServiceOf(t, "ApiAuditLogService"))
	for _, want := range []string{`v-for="item in requestMethodList"`, "const requestMethodList = ["} {
		if !strings.Contains(drawer, want) {
			t.Errorf("抽屉产物缺少 %q", want)
		}
	}
}

// TestElementStatusEnumGate 只有 status 枚举才走 statusToColor/statusToName 标签渲染:
// 判据放宽到任意枚举,列表页会引用一个只为 status 导入的符号,且把 GET/POST 译成「启用/禁用」。
func TestElementStatusEnumGate(t *testing.T) {
	status := &ParsedField{Name: "status", IsEnum: true, EnumValues: []string{"ON", "OFF"}}
	method := &ParsedField{Name: "requestMethod", IsEnum: true, EnumValues: []string{"GET", "POST", "PUT", "DELETE"}}

	if !isElementStatusEnum(status) {
		t.Error("status 枚举应走标签渲染")
	}
	if isElementStatusEnum(method) {
		t.Error("非 status 枚举不应套用 statusToName/statusToColor")
	}

	// 列表页:没有 status 枚举时不得出现 statusToName 引用
	page := elementPageCode(mustServiceOf(t, "ApiAuditLogService"), "admin", "log/api_audit_log")
	if strings.Contains(page, "statusToName(") {
		t.Errorf("api-audit-log 列表页仍在渲染 requestMethod 的状态文案:\n%s", truncate(page))
	}
	if strings.Contains(page, "statusList") {
		t.Error("列表页导入了未被使用的 statusList")
	}
}

func mustServiceOf(t *testing.T, tag string) *ParsedService {
	t.Helper()
	for _, s := range ExtractServices(loadTestSpec(t)) {
		if s.TagName == tag {
			return s
		}
	}
	t.Fatalf("测试规格里没有 %s 服务", tag)
	return nil
}

// isSourceFile 产物里需要做静态体检的代码文件（json/scss 等不在其列）
func isSourceFile(path string) bool {
	return strings.HasSuffix(path, ".vue") || strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")
}

// eachGeneratedFile 用与黄金样本完全相同的选项跑三框架,逐个产物回调
func eachGeneratedFile(t *testing.T, fn func(fw Framework, f GeneratedFile)) {
	t.Helper()
	spec := loadTestSpec(t)
	for _, fw := range []Framework{FrameworkVueVben, FrameworkVueElement, FrameworkReactAntd} {
		opts := goldenOptions(fw)
		opts.Spec = spec
		files, err := Generate(opts)
		if err != nil {
			t.Fatalf("%s 生成失败: %v", fw, err)
		}
		for _, f := range files {
			fn(fw, f)
		}
	}
}

var (
	reBracedImport = regexp.MustCompile(`(?s)import\s+(?:type\s+)?\{([^}]*)\}`)
	reSpecifier    = regexp.MustCompile(`^(?:type\s+)?[A-Za-z_$][\w$]*(?:\s+as\s+[A-Za-z_$][\w$]*)?$`)
)

// TestGeneratedImportsAreWellFormed 生成的 import 花括号必须非空、且每项都是合法标识符。
// 两条曾同时踩中过:只读服务(仅 List/Get)留下 `import {\n  ,\n}` 的空壳,
// 复合组件被写成 `import { ProFormRadio.Group }`——两种都是一行都过不了解析的语法错误,
// 而逐字比对的黄金样本对此毫无反应。
func TestGeneratedImportsAreWellFormed(t *testing.T) {
	checked := 0
	eachGeneratedFile(t, func(fw Framework, f GeneratedFile) {
		if !isSourceFile(f.Path) {
			return
		}
		for _, group := range reBracedImport.FindAllStringSubmatch(f.Content, -1) {
			checked++
			valid := 0
			for _, spec := range strings.Split(group[1], ",") {
				spec = strings.TrimSpace(spec)
				if spec == "" {
					continue // 生成风格里的尾随逗号
				}
				if !reSpecifier.MatchString(spec) {
					t.Errorf("%s/%s: 非法的 import 项 %q", fw, f.Path, spec)
				}
				valid++
			}
			if valid == 0 {
				t.Errorf("%s/%s: import 花括号里没有任何标识符", fw, f.Path)
			}
		}
	})

	// 夹具必须真的覆盖到 import 语句,否则本用例退化成空跑
	if checked < 20 {
		t.Errorf("仅检查到 %d 条 import,夹具覆盖面可能已缩小", checked)
	}
}
