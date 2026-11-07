package service

import (
	"context"
	"testing"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
{{if not .UseRepo}}
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
{{end}}

	{{.SourceApiPackage}} "{{.Module}}/api/gen/go/{{lower .SourceApiPackageName}}/service/{{lower .SourceApiPackageVersion}}"
)

{{if .UseRepo}}
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
		log: bLogger.NewHelper(bLogger.NewStdLogger()),
		{{.DataSourceVar}}: repo,
	}
}
{{else}}
// fake{{pascal .Model}}ServiceClient 通过嵌入完整 client 接口满足类型:
// 仅覆写 CRUD 五方法(测试只触碰这些),其余方法调用会 panic——
// 这是有意的:骨架测试旨在暴露越界调用。
type fake{{pascal .Model}}ServiceClient struct {
	{{.DataSourceType}}
	item    *{{.SourceApiPackage}}.{{pascal .Model}}
	deleted bool
}

func (f *fake{{pascal .Model}}ServiceClient) List(ctx context.Context, req *paginationV1.PagingRequest, opts ...grpc.CallOption) (*{{.SourceApiPackage}}.List{{pascal .Model}}Response, error) {
	return &{{.SourceApiPackage}}.List{{pascal .Model}}Response{}, nil
}

func (f *fake{{pascal .Model}}ServiceClient) Get(ctx context.Context, req *{{.SourceApiPackage}}.Get{{pascal .Model}}Request, opts ...grpc.CallOption) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	return f.item, nil
}

func (f *fake{{pascal .Model}}ServiceClient) Create(ctx context.Context, req *{{.SourceApiPackage}}.Create{{pascal .Model}}Request, opts ...grpc.CallOption) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	return f.item, nil
}

func (f *fake{{pascal .Model}}ServiceClient) Update(ctx context.Context, req *{{.SourceApiPackage}}.Update{{pascal .Model}}Request, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fake{{pascal .Model}}ServiceClient) Delete(ctx context.Context, req *{{.SourceApiPackage}}.Delete{{pascal .Model}}Request, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	f.deleted = true
	return &emptypb.Empty{}, nil
}

// newTest{{.ClassName}} 以 fake client 构造被测 service,不经过 bootstrap 装配。
func newTest{{.ClassName}}(cli {{.DataSourceType}}) *{{.ClassName}} {
	return &{{.ClassName}}{
		log: bLogger.NewHelper(bLogger.NewStdLogger()),
		{{.DataSourceVar}}: cli,
	}
}
{{end}}

func Test{{pascal .Model}}Service_List(t *testing.T) {
{{if .UseRepo}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
	resp, err := svc.List(context.Background(), &paginationV1.PagingRequest{})
{{else}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}ServiceClient{})
	resp, err := svc.List(context.Background(), &paginationV1.PagingRequest{})
{{end}}
	if err != nil {
		t.Fatalf("List 不应报错: %v", err)
	}
	if resp == nil {
		t.Fatal("List 应返回响应")
	}
}

func Test{{pascal .Model}}Service_Get(t *testing.T) {
{{if .UseRepo}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
{{else}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}ServiceClient{})
{{end}}
	if _, err := svc.Get(context.Background(), &{{.SourceApiPackage}}.Get{{pascal .Model}}Request{}); err != nil {
		t.Fatalf("Get 不应报错: %v", err)
	}
}

func Test{{pascal .Model}}Service_Create_NilRequest(t *testing.T) {
{{if .UseRepo}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
{{else}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}ServiceClient{})
{{end}}
	if _, err := svc.Create(context.Background(), nil); err == nil {
		t.Fatal("Create(nil) 应返回参数错误")
	}
}

func Test{{pascal .Model}}Service_Update_NilRequest(t *testing.T) {
{{if .UseRepo}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
{{else}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}ServiceClient{})
{{end}}
	if _, err := svc.Update(context.Background(), nil); err == nil {
		t.Fatal("Update(nil) 应返回参数错误")
	}
}

func Test{{pascal .Model}}Service_Delete(t *testing.T) {
{{if .UseRepo}}
	repo := &fake{{pascal .Model}}Repo{}
	svc := newTest{{.ClassName}}(repo)
	if _, err := svc.Delete(context.Background(), &{{.SourceApiPackage}}.Delete{{pascal .Model}}Request{}); err != nil {
		t.Fatalf("Delete 不应报错: %v", err)
	}
	if !repo.deleted {
		t.Fatal("Delete 应到达仓储层")
	}
{{else}}
	cli := &fake{{pascal .Model}}ServiceClient{}
	svc := newTest{{.ClassName}}(cli)
	if _, err := svc.Delete(context.Background(), &{{.SourceApiPackage}}.Delete{{pascal .Model}}Request{}); err != nil {
		t.Fatalf("Delete 不应报错: %v", err)
	}
	if !cli.deleted {
		t.Fatal("Delete 应到达 client 层")
	}
{{end}}
}

func Test{{pascal .Model}}Service_Delete_NilRequest(t *testing.T) {
{{if .UseRepo}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}Repo{})
{{else}}
	svc := newTest{{.ClassName}}(&fake{{pascal .Model}}ServiceClient{})
{{end}}
	if _, err := svc.Delete(context.Background(), nil); err == nil {
		t.Fatal("Delete(nil) 应返回参数错误")
	}
}
