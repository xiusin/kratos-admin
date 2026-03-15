package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/loginlog"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// LoginLogRepo 登录日志数据访问接口
type LoginLogRepo interface {
	// Create 记录登录日志
	Create(ctx context.Context, data *consumerV1.LoginLog) (*consumerV1.LoginLog, error)

	// List 分页查询登录日志
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLoginLogsResponse, error)
}

type loginLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper             *mapper.CopierMapper[consumerV1.LoginLog, ent.LoginLog]
	loginTypeConverter *mapper.EnumTypeConverter[consumerV1.LoginLog_LoginType, loginlog.LoginType]

	repository *entCrud.Repository[
		ent.LoginLogQuery, ent.LoginLogSelect,
		ent.LoginLogCreate, ent.LoginLogCreateBulk,
		ent.LoginLogUpdate, ent.LoginLogUpdateOne,
		ent.LoginLogDelete,
		predicate.LoginLog,
		consumerV1.LoginLog, ent.LoginLog,
	]
}

// NewLoginLogRepo 创建登录日志数据访问实例
func NewLoginLogRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) LoginLogRepo {
	repo := &loginLogRepo{
		log:                ctx.NewLoggerHelper("login-log/repo/consumer-service"),
		entClient:          entClient,
		mapper:             mapper.NewCopierMapper[consumerV1.LoginLog, ent.LoginLog](),
		loginTypeConverter: mapper.NewEnumTypeConverter[consumerV1.LoginLog_LoginType, loginlog.LoginType](consumerV1.LoginLog_LoginType_name, consumerV1.LoginLog_LoginType_value),
	}

	repo.init()

	return repo
}

func (r *loginLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LoginLogQuery, ent.LoginLogSelect,
		ent.LoginLogCreate, ent.LoginLogCreateBulk,
		ent.LoginLogUpdate, ent.LoginLogUpdateOne,
		ent.LoginLogDelete,
		predicate.LoginLog,
		consumerV1.LoginLog, ent.LoginLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.loginTypeConverter.NewConverterPair())
}

// Create 记录登录日志
func (r *loginLogRepo) Create(ctx context.Context, data *consumerV1.LoginLog) (*consumerV1.LoginLog, error) {
	if data == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().LoginLog.Create().
		SetNillableTenantID(data.TenantId).
		SetConsumerID(data.GetConsumerId()).
		SetPhone(data.GetPhone()).
		SetSuccess(data.GetSuccess()).
		SetNillableFailReason(data.FailReason).
		SetIPAddress(data.GetIpAddress()).
		SetNillableUserAgent(data.UserAgent).
		SetNillableDeviceType(data.DeviceType)

	// 设置 login_type
	if loginType := r.loginTypeConverter.ToEntity(data.LoginType); loginType != nil {
		builder.SetLoginType(*loginType)
	}

	if data.LoginAt != nil {
		builder.SetLoginAt(data.LoginAt.AsTime())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert login log failed: %s", err.Error())
		return nil, errors.InternalServer("INSERT_FAILED", "insert login log failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// List 分页查询登录日志
func (r *loginLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLoginLogsResponse, error) {
	if req == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().LoginLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListLoginLogsResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListLoginLogsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}
