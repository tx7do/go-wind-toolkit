package service

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	{{.SourceApiPackage}} "{{.Module}}/api/gen/go/{{lower .SourceApiPackageName}}/service/{{lower .SourceApiPackageVersion}}"
)

// fake{{pascal .Model}}Repo 是 {{pascal .Model}}Repo 的内存实现:
// 骨架测试不依赖数据库,后续在此扩展夹具与用例。
type fake{{pascal .Model}}Repo struct {
	item    *{{.SourceApiPackage}}.{{pascal .Model}}
	deleted bool
}

func (f *fake{{pascal .Model}}Repo) List(ctx context.Context, req *paginationV1.PagingRequest) (*{{.SourceApiPackage}}.List{{pascal .Model}}Response, error) {
	return &{{.SourceApiPackage}}.List{{pascal .Model}}Response{}, nil
}

func (f *fake{{pascal .Model}}Repo) Get(ctx context.Context, req *{{.SourceApiPackage}}.Get{{pascal .Model}}Request) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	return f.item, nil
}

func (f *fake{{pascal .Model}}Repo) Create(ctx context.Context, req *{{.SourceApiPackage}}.Create{{pascal .Model}}Request) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	return f.item, nil
}

func (f *fake{{pascal .Model}}Repo) Update(ctx context.Context, req *{{.SourceApiPackage}}.Update{{pascal .Model}}Request) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	return f.item, nil
}

{{- if eq .OrmType "gorm"}}
func (f *fake{{pascal .Model}}Repo) Delete(ctx context.Context, req *{{.SourceApiPackage}}.Delete{{pascal .Model}}Request) (bool, error) {
	f.deleted = true
	return true, nil
}
{{- else}}
func (f *fake{{pascal .Model}}Repo) Delete(ctx context.Context, req *{{.SourceApiPackage}}.Delete{{pascal .Model}}Request) error {
	f.deleted = true
	return nil
}
{{- end}}

// newTest{{.ClassName}} 以 fake 仓储构造被测 service,不经过 bootstrap 装配。
func newTest{{.ClassName}}(repo {{pascal .Model}}Repo) *{{.ClassName}} {
	return &{{.ClassName}}{
		log: log.NewHelper(log.NewStdLogger(io.Discard)),
		{{.DataSourceVar}}: repo,
	}
}

func Test{{pascal .Model}}Service_List(t *testing.T) {
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
	resp, err := svc.List(context.Background(), &paginationV1.PagingRequest{})
	if err != nil {
		t.Fatalf("List 不应报错: %v", err)
	}
	if resp == nil {
		t.Fatal("List 应返回响应")
	}
}

func Test{{pascal .Model}}Service_Get(t *testing.T) {
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
	if _, err := svc.Get(context.Background(), &{{.SourceApiPackage}}.Get{{pascal .Model}}Request{}); err != nil {
		t.Fatalf("Get 不应报错: %v", err)
	}
}

func Test{{pascal .Model}}Service_Create_NilRequest(t *testing.T) {
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
	if _, err := svc.Create(context.Background(), nil); err == nil {
		t.Fatal("Create(nil) 应返回参数错误")
	}
}

func Test{{pascal .Model}}Service_Update_NilRequest(t *testing.T) {
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
	if _, err := svc.Update(context.Background(), nil); err == nil {
		t.Fatal("Update(nil) 应返回参数错误")
	}
}

func Test{{pascal .Model}}Service_Delete(t *testing.T) {
	repo := &fake{{pascal .Model}}Repo{}
	svc := newTest{{.ClassName}}(repo)
	if _, err := svc.Delete(context.Background(), &{{.SourceApiPackage}}.Delete{{pascal .Model}}Request{}); err != nil {
		t.Fatalf("Delete 不应报错: %v", err)
	}
	if !repo.deleted {
		t.Fatal("Delete 应到达仓储层")
	}
}

func Test{{pascal .Model}}Service_Delete_NilRequest(t *testing.T) {
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
	if _, err := svc.Delete(context.Background(), nil); err == nil {
		t.Fatal("Delete(nil) 应返回参数错误")
	}
}
