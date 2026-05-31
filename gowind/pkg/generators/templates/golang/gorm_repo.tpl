package data

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"gorm.io/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCurd "github.com/tx7do/go-crud/gorm"

	"{{.Module}}/app/{{lower .Service}}/service/internal/data/gorm/models"

	{{.ApiPackage}} "{{.Module}}/api/gen/go/{{lower .Service}}/service/{{.ApiPackageVersion}}"
)

type {{.ClassName}} struct {
	gormClient *gormCrud.Client
	log  *log.Helper

	mapper     *mapper.CopierMapper[{{.ApiPackage}}.{{pascal .Model}}, models.{{pascal .Model}}]
	repository *gormCurd.Repository[{{.ApiPackage}}.{{pascal .Model}}, models.{{pascal .Model}}]
}

func New{{.ClassName}}(ctx *bootstrap.Context, gormClient *gormCrud.Client) *{{.ClassName}} {
	repo := &{{.ClassName}}{
		gormClient:   gormClient,
		log:    ctx.NewLoggerHelper("{{lower .Model}}/repo/{{lower .Service}}-service"),
		mapper: mapper.NewCopierMapper[{{.ApiPackage}}.{{pascal .Model}}, models.{{pascal .Model}}](),
	}

	repo.init()

	return repo
}

func (r *{{.ClassName}}) init() {
    r.repository = gormCurd.NewRepository[{{.ApiPackage}}.{{pascal .Model}}, models.{{pascal .Model}}](
        r.mapper,
    )

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *{{.ClassName}}) List(ctx context.Context, req *pagination.PagingRequest) (*{{.ApiPackage}}.List{{pascal .Model}}Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.gormClient, req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &{{.ApiPackage}}.List{{pascal .Model}}Response{Total: 0, Items: nil}, nil
	}

	return &{{.ApiPackage}}.List{{pascal .Model}}Response{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *{{.ClassName}}) Get(ctx context.Context, req *{{.ApiPackage}}.Get{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	var whereCond *gorm.DB
	switch req.QueryBy.(type) {
	case *{{.ApiPackage}}.Get{{pascal .Model}}Request_Id:
		whereCond = r.gormClient.Where("id = ?", req.GetId())
	default:
		whereCond = r.gormClient.Where("id = ?", req.GetId())
	}

	dto, err := r.repository.Get(ctx, whereCond, req.GetViewMask())
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *{{.ClassName}}) Create(ctx context.Context, req *{{.ApiPackage}}.Create{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New("request is nil")
	}

	result, err := r.repository.Create(ctx, r.gormClient, req.Data, nil)

	return result, err
}

func (r *{{.ClassName}}) Update(ctx context.Context, req *{{.ApiPackage}}.Update{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New("request is nil")
	}

	result, err := r.repository.Update(ctx,
		r.gormClient.Where("id = ?", req.Data.GetId()),
		req.Data,
		req.GetUpdateMask(),
	)

	return result, err
}

func (r *{{.ClassName}}) Upsert(ctx context.Context, req *{{.ApiPackage}}.Update{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New("request is nil")
	}

	var err error

	result, err := r.repository.Upsert(ctx, r.gormClient, req.Data, req.GetUpdateMask())

	return result, err
}

func (r *{{.ClassName}}) Delete(ctx context.Context, req *{{.ApiPackage}}.Delete{{pascal .Model}}Request) (bool, error) {
	if req == nil {
		return false, errors.New("request is nil")
	}

	result, err := r.repository.Delete(
		ctx,
		r.gormClient.Where("id = ?", req.GetId()),
		true,
	)

	return result > 0, err
}
