package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
{{- if .HasTimeConversionField}}
	"github.com/tx7do/go-utils/timeutil"
{{- end}}
{{- if .HasStringNumConversionField}}
	"github.com/tx7do/go-utils/stringutil"
{{- end}}

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"{{.Module}}/app/{{lower .Service}}/service/internal/data/ent"
	"{{.Module}}/app/{{lower .Service}}/service/internal/data/ent/predicate"
	"{{.Module}}/app/{{lower .Service}}/service/internal/data/ent/{{lower .Model}}"

	{{.ApiPackage}} "{{.Module}}/api/gen/go/{{lower .Service}}/service/{{.ApiPackageVersion}}"
)

type {{.ClassName}} struct {
	entClient *entCrud.EntClient[*ent.Client]
	log          *log.Helper

	mapper     *mapper.CopierMapper[{{.ApiPackage}}.{{pascal .Model}}, ent.{{pascal .Model}}]
	repository *entCrud.Repository[
		ent.{{pascal .Model}}Query, ent.{{pascal .Model}}Select,
		ent.{{pascal .Model}}Create, ent.{{pascal .Model}}CreateBulk,
		ent.{{pascal .Model}}Update, ent.{{pascal .Model}}UpdateOne,
		ent.{{pascal .Model}}Delete,
		predicate.{{pascal .Model}},
		{{.ApiPackage}}.{{pascal .Model}}, ent.{{pascal .Model}},
	]
}

func New{{.ClassName}}(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *{{.ClassName}} {
	repo := &{{.ClassName}}{
		log:  ctx.NewLoggerHelper("{{lower .Model}}/repo/{{lower .Service}}-service"),
		entClient: entClient,
		mapper: mapper.NewCopierMapper[{{.ApiPackage}}.{{pascal .Model}}, ent.{{pascal .Model}}](),
	}

	repo.init()

	return repo
}

func (r *{{.ClassName}}) init() {
	r.repository = entCrud.NewRepository[
		ent.{{pascal .Model}}Query, ent.{{pascal .Model}}Select,
		ent.{{pascal .Model}}Create, ent.{{pascal .Model}}CreateBulk,
		ent.{{pascal .Model}}Update, ent.{{pascal .Model}}UpdateOne,
		ent.{{pascal .Model}}Delete,
		predicate.{{pascal .Model}},
		{{.ApiPackage}}.{{pascal .Model}}, ent.{{pascal .Model}},
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *{{.ClassName}}) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	builder := r.entClient.Client().{{pascal .Model}}.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query {{lower .Model}} count failed: %s", err.Error())
		return 0, {{.ApiPackage}}.ErrorInternalServerError("query {{lower .Model}} count failed")
	}

	return count, nil
}

func (r *{{.ClassName}}) List(ctx context.Context, req *paginationV1.PagingRequest) (*{{.ApiPackage}}.List{{pascal .Model}}Response, error) {
	if req == nil {
		return nil, {{.ApiPackage}}.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().{{pascal .Model}}.Query()

    ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
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

func (r *{{.ClassName}}) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().{{pascal .Model}}.Query().
		Where({{lower .Model}}.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query {{lower .Model}} exist failed: %s", err.Error())
		return false, {{.ApiPackage}}.ErrorInternalServerError("query {{lower .Model}} exist failed")
	}
	return exist, nil
}

func (r *{{.ClassName}}) Get(ctx context.Context, req *{{.ApiPackage}}.Get{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil {
		return nil, {{.ApiPackage}}.ErrorBadRequest("invalid parameter")
	}

    var whereCond []func(s *sql.Selector)
    switch req.QueryBy.(type) {
    case *{{.ApiPackage}}.Get{{pascal .Model}}Request_Id:
        whereCond = append(whereCond, {{lower .Model}}.IDEQ(req.GetId()))
    default:
        whereCond = append(whereCond, {{lower .Model}}.IDEQ(req.GetId()))
    }

    builder := r.entClient.Client().{{pascal .Model}}.Query()
	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *{{.ClassName}}) Create(ctx context.Context, req *{{.ApiPackage}}.Create{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil || req.Data == nil {
		return nil, {{.ApiPackage}}.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().{{pascal .Model}}.Create()
{{- range .Fields}}
{{- if not .IsPrimaryKey}}
	builder.{{.EntCreateSetFunc}}
{{- end}}
{{- end}}

	if ret, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert one data failed: %s", err.Error())
		return nil, {{.ApiPackage}}.ErrorInternalServerError("insert data failed")
	} else {
		return r.mapper.ToDTO(ret), nil
	}
}

func (r *{{.ClassName}}) Update(ctx context.Context, req *{{.ApiPackage}}.Update{{pascal .Model}}Request) (*{{.ApiPackage}}.{{pascal .Model}}, error) {
	if req == nil || req.Data == nil {
		return nil, {{.ApiPackage}}.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetData().GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &{{.ApiPackage}}.Create{{pascal .Model}}Request{Data: req.Data}
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().{{pascal .Model}}.UpdateOneID(req.Data.GetId())
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *{{.ApiPackage}}.{{pascal .Model}}) {
{{- range .Fields}}
{{- if not .IsPrimaryKey}}
			builder.{{.EntSetNillableFunc}}
{{- end}}
{{- end}}
		},
	)

	return result, err
}

func (r *{{.ClassName}}) Delete(ctx context.Context, req *{{.ApiPackage}}.Delete{{pascal .Model}}Request) error {
	if req == nil {
		return {{.ApiPackage}}.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().{{pascal .Model}}.Delete()
	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ({{lower .Model}}.FieldID, req.GetId()))
	})

	return err
}
