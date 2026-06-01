package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
{{if not .IsGrpc}}
	"github.com/tx7do/go-utils/trans"
{{end}}
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"google.golang.org/protobuf/types/known/emptypb"
{{if .UseRepo}}
	"{{.Module}}/app/{{.Service}}/service/internal/data"
{{end}}
	{{.TargetApiPackage}} "{{.Module}}/api/gen/go/{{lower .TargetApiPackageName}}/service/{{lower .TargetApiPackageVersion}}"
{{- if not .IsSameApi}}
	{{.SourceApiPackage}} "{{.Module}}/api/gen/go/{{lower .SourceApiPackageName}}/service/{{lower .SourceApiPackageVersion}}"
{{end -}}

{{if not .IsGrpc}}
	"{{.Module}}/pkg/middleware/auth"
{{- end}}
)

type {{.ClassName}} struct {
	{{.ServiceInterface}}

	log *log.Helper

	{{.DataSourceVar}} {{.DataSourceType}}
}

func New{{.ClassName}}(
    ctx *bootstrap.Context,
    {{.DataSourceVar}} {{.DataSourceType}},
) *{{.ClassName}} {
	return &{{.ClassName}}{
		log:  ctx.NewLoggerHelper("{{lower .Model}}/service/{{lower .Service}}-service"),
		{{.DataSourceVar}}:  {{.DataSourceVar}},
	}
}

func (s *{{.ClassName}}) List(ctx context.Context, req *paginationV1.PagingRequest) (*{{.SourceApiPackage}}.List{{pascal .Model}}Response, error) {
	return s.{{.DataSourceVar}}.List(ctx, req)
}

func (s *{{.ClassName}}) Get(ctx context.Context, req *{{.SourceApiPackage}}.Get{{pascal .Model}}Request) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	return s.{{.DataSourceVar}}.Get(ctx, req)
}

func (s *{{.ClassName}}) Create(ctx context.Context, req *{{.SourceApiPackage}}.Create{{pascal .Model}}Request) (*{{.SourceApiPackage}}.{{pascal .Model}}, error) {
	if req == nil || req.Data == nil {
		return nil, {{.SourceApiPackage}}.ErrorBadRequest("invalid parameter")
	}
{{if not .IsGrpc}}
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.GetUserId())
{{end -}}

{{- if not .UseRepo}}
	if result, err := s.{{.DataSourceVar}}.Create(ctx, req); err != nil {
		return nil, err
	} else {
	    return result, nil
	}
{{else}}
    return s.{{.DataSourceVar}}.Create(ctx, req)
{{end -}}
}

func (s *{{.ClassName}}) Update(ctx context.Context, req *{{.SourceApiPackage}}.Update{{pascal .Model}}Request) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, {{.SourceApiPackage}}.ErrorBadRequest("invalid parameter")
	}
{{if not .IsGrpc}}
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
{{end -}}

{{if not .UseRepo}}
	if _, err := s.{{.DataSourceVar}}.Update(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
{{else}}
	if _, err := s.{{.DataSourceVar}}.Update(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
{{end -}}
}

func (s *{{.ClassName}}) Delete(ctx context.Context, req *{{.SourceApiPackage}}.Delete{{pascal .Model}}Request) (*emptypb.Empty, error) {
    if req == nil {
        return nil, {{.SourceApiPackage}}.ErrorBadRequest("invalid parameter")
    }
{{if .UseRepo}}
	if err := s.{{.DataSourceVar}}.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
{{else}}
    return s.{{.DataSourceVar}}.Delete(ctx, req)
{{end -}}
}
