package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
	"go-wind-admin/app/consumer/service/internal/data/ent/smslog"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// SMSLogRepo 短信日志数据访问接口
type SMSLogRepo interface {
	// Create 创建短信日志
	Create(ctx context.Context, data *consumerV1.SMSLog) (*consumerV1.SMSLog, error)

	// List 分页查询短信日志
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListSMSLogsResponse, error)
}

type smsLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper           *mapper.CopierMapper[consumerV1.SMSLog, ent.SMSLog]
	smsTypeConverter *mapper.EnumTypeConverter[consumerV1.SMSLog_SMSType, smslog.SmsType]
	channelConverter *mapper.EnumTypeConverter[consumerV1.SMSLog_Channel, smslog.Channel]
	statusConverter  *mapper.EnumTypeConverter[consumerV1.SMSLog_Status, smslog.Status]

	repository *entCrud.Repository[
		ent.SMSLogQuery, ent.SMSLogSelect,
		ent.SMSLogCreate, ent.SMSLogCreateBulk,
		ent.SMSLogUpdate, ent.SMSLogUpdateOne,
		ent.SMSLogDelete,
		predicate.SMSLog,
		consumerV1.SMSLog, ent.SMSLog,
	]
}

// NewSMSLogRepo 创建短信日志数据访问实例
func NewSMSLogRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) SMSLogRepo {
	repo := &smsLogRepo{
		log:              ctx.NewLoggerHelper("consumer/repo/sms-log"),
		entClient:        entClient,
		mapper:           mapper.NewCopierMapper[consumerV1.SMSLog, ent.SMSLog](),
		smsTypeConverter: mapper.NewEnumTypeConverter[consumerV1.SMSLog_SMSType, smslog.SmsType](consumerV1.SMSLog_SMSType_name, consumerV1.SMSLog_SMSType_value),
		channelConverter: mapper.NewEnumTypeConverter[consumerV1.SMSLog_Channel, smslog.Channel](consumerV1.SMSLog_Channel_name, consumerV1.SMSLog_Channel_value),
		statusConverter:  mapper.NewEnumTypeConverter[consumerV1.SMSLog_Status, smslog.Status](consumerV1.SMSLog_Status_name, consumerV1.SMSLog_Status_value),
	}

	repo.init()

	return repo
}

func (r *smsLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.SMSLogQuery, ent.SMSLogSelect,
		ent.SMSLogCreate, ent.SMSLogCreateBulk,
		ent.SMSLogUpdate, ent.SMSLogUpdateOne,
		ent.SMSLogDelete,
		predicate.SMSLog,
		consumerV1.SMSLog, ent.SMSLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.smsTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.channelConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// Create 创建短信日志
func (r *smsLogRepo) Create(ctx context.Context, data *consumerV1.SMSLog) (*consumerV1.SMSLog, error) {
	if data == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().SMSLog.Create().
		SetNillableTenantID(data.TenantId).
		SetNillablePhone(data.Phone).
		SetNillableSmsType(r.smsTypeConverter.ToEntity(data.SmsType)).
		SetNillableContent(data.Content).
		SetNillableChannel(r.channelConverter.ToEntity(data.Channel)).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status))

	// 设置可选字段
	if data.Code != nil {
		builder.SetCode(*data.Code)
	}
	if data.ErrorMessage != nil {
		builder.SetErrorMessage(*data.ErrorMessage)
	}
	if data.SentAt != nil {
		builder.SetSentAt(data.SentAt.AsTime())
	} else {
		builder.SetSentAt(time.Now())
	}
	if data.ExpiresAt != nil {
		builder.SetExpiresAt(data.ExpiresAt.AsTime())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert sms log failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert sms log failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// List 分页查询短信日志
func (r *smsLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListSMSLogsResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().SMSLog.Query()

	// 多租户过滤：自动添加 tenant_id 过滤条件
	// 注意：这里假设 tenant_id 已经通过中间件注入到 context 中
	// 实际实现中需要从 context 中提取 tenant_id
	// 这里先保留基础查询，后续在 service 层添加租户过滤

	// 按发送时间倒序排列
	builder = builder.Order(ent.Desc(smslog.FieldSentAt))

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListSMSLogsResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListSMSLogsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}
