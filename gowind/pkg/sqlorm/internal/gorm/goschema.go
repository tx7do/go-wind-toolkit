package gorm

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"ariga.io/atlas/sql/schema"
	"golang.org/x/mod/modfile"
	"gorm.io/gen"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/schemasource"
)

// gensrcDirName 是 DAO 回转生成期间的临时目录名,生成结束后删除。
// gorm gen 必须从模型包推导 import 路径,直接指向用户已有 models 目录会
// 覆盖手写模型,因此在 daoPath 下开临时模型包,再把 DAO 移动并改写 import。
const gensrcDirName = ".gow_gensrc"

// ImporterFromGoSchema 从 gorm model 源码目录(dsn 形如 gorm://<dir>)生成
// DAO:解析模型 → 还原 MySQL DDL → gorm.io/rawsql 内存库 → gorm gen。
// 模型文件只在目标不存在时补齐,绝不覆盖已有用户模型。
func ImporterFromGoSchema(_ context.Context, dsn string, schemaPath, daoPath *string, includeTables []string) error {
	if schemaPath == nil || daoPath == nil {
		return fmt.Errorf("gormimport: schema/dao path is nil")
	}
	dir := strings.TrimPrefix(dsn, "gorm://")
	ps, err := schemasource.ParseGormSchemaDir(dir)
	if err != nil {
		return fmt.Errorf("gormimport: parse gorm schema dir: %w", err)
	}

	tables := filterParsedTables(ps.Tables, includeTables)
	if len(tables) == 0 {
		return fmt.Errorf("gormimport: no tables to generate (dir: %s)", dir)
	}
	ps.Tables = tables

	ddl := strings.Join(schemasource.BuildMySQLDDL(ps), ";\n")
	db, err := NewRawSqlClient(ddl)
	if err != nil {
		return fmt.Errorf("gormimport: rawsql from parsed schema failed: %w", err)
	}

	moduleRoot, modulePath, err := locateGoModule(*daoPath)
	if err != nil {
		return fmt.Errorf("gormimport: %w (schema-source DAO generation requires the target inside a Go module)", err)
	}

	tmpRoot := filepath.Join(*daoPath, gensrcDirName)
	tmpModels := filepath.Join(tmpRoot, filepath.Base(*schemaPath))
	tmpDao := filepath.Join(tmpRoot, "dao")
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	g := gen.NewGenerator(gen.Config{
		OutPath:      tmpDao,
		ModelPkgPath: tmpModels,

		Mode: gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,

		WithUnitTest:      false,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	g.UseDB(db)

	var models []any
	for _, t := range tables {
		models = append(models, g.GenerateModel(t.Name))
	}
	g.ApplyBasic(models...)
	g.Execute()

	tmpImport := importPathOf(moduleRoot, modulePath, tmpModels)
	realImport := importPathOf(moduleRoot, modulePath, *schemaPath)
	if tmpImport == "" || realImport == "" {
		return fmt.Errorf("gormimport: cannot resolve model package import path (tmp=%q real=%q)", tmpImport, realImport)
	}

	if err := moveGeneratedGoFiles(tmpDao, *daoPath, tmpImport, realImport, true); err != nil {
		return err
	}
	// 模型仅补缺:用户手写模型(含自定义类型、方法、tag)不被覆盖。
	return moveGeneratedGoFiles(tmpModels, *schemaPath, "", "", false)
}

// filterParsedTables 按 include 名单过滤;名单为空时全量返回。
func filterParsedTables(tables []*schema.Table, include []string) []*schema.Table {
	if len(include) == 0 {
		return tables
	}
	want := make(map[string]bool, len(include))
	for _, t := range include {
		want[t] = true
	}
	out := make([]*schema.Table, 0, len(include))
	for _, t := range tables {
		if want[t.Name] {
			out = append(out, t)
		}
	}
	return out
}

// moveGeneratedGoFiles 把 src 目录里的 .go 文件移动到 dst:
// overwrite=true 覆盖同名文件(DAO 是生成物);false 仅补齐缺失的模型——
// 目标包已声明同名类型(如手写 user.go 里的 User)时跳过,避免包内类型重复声明。
// from!=to 时对内容做 import 路径替换。
func moveGeneratedGoFiles(src, dst, from, to string, overwrite bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("gormimport: read generated dir: %w", err)
	}
	if err = os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	var dstTypes map[string]bool
	if !overwrite {
		dstTypes = packageTypeNames(dst)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		srcFile := filepath.Join(src, e.Name())
		dstFile := filepath.Join(dst, e.Name())
		if !overwrite {
			if _, statErr := os.Stat(dstFile); statErr == nil {
				continue
			}
			if declaresAnyOf(srcFile, dstTypes) {
				continue
			}
		}
		content, err := os.ReadFile(srcFile)
		if err != nil {
			return err
		}
		if from != "" && to != "" && from != to {
			content = []byte(strings.ReplaceAll(string(content), from, to))
		}
		if err := os.WriteFile(dstFile, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// packageTypeNames 收集目录内已有 Go 文件声明的类型名。
func packageTypeNames(dir string) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		for name := range fileTypeNames(filepath.Join(dir, e.Name())) {
			out[name] = true
		}
	}
	return out
}

func declaresAnyOf(path string, existing map[string]bool) bool {
	if len(existing) == 0 {
		return false
	}
	for name := range fileTypeNames(path) {
		if existing[name] {
			return true
		}
	}
	return false
}

func fileTypeNames(path string) map[string]bool {
	out := map[string]bool{}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return out
	}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				out[ts.Name.Name] = true
			}
		}
	}
	return out
}

// locateGoModule 从 dir 向上找 go.mod,返回 module 根与 module path。
func locateGoModule(dir string) (string, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	for {
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", "", fmt.Errorf("go.mod not found above %q", dir)
		}
		abs = parent
		data, err := os.ReadFile(filepath.Join(abs, "go.mod"))
		if err != nil {
			continue
		}
		f, err := modfile.Parse(filepath.Join(abs, "go.mod"), data, nil)
		if err != nil || f.Module == nil || f.Module.Mod.Path == "" {
			return "", "", fmt.Errorf("malformed go.mod at %s", abs)
		}
		return abs, f.Module.Mod.Path, nil
	}
}

// importPathOf 计算目录在 module 内的 import 路径。
func importPathOf(moduleRoot, modulePath, dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(moduleRoot, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return modulePath + "/" + filepath.ToSlash(rel)
}
