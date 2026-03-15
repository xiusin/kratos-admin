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
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
	"go-wind-admin/app/consumer/service/internal/data/ent/smslog"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// SMSLogRepo 短信日志数据访问接口
type SMSLogRepo interface {
	// Create 记录短信日志
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
		log:              ctx.NewLoggerHelper("sms-log/repo/consumer-service"),
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

// Create 记录短信日志
func (r *smsLogRepo) Create(ctx context.Context, data *consumerV1.SMSLog) (*consumerV1.SMSLog, error) {
	if data == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().SMSLog.Create().
		SetNillableTenantID(data.TenantId).
		SetPhone(data.GetPhone()).
		SetContent(data.GetContent()).
		SetNillableCode(data.Code).
		SetNillableErrorMessage(data.ErrorMessage)

	// 设置 sms_type
	if smsType := r.smsTypeConverter.ToEntity(data.SmsType); smsType != nil {
		builder.SetSmsType(*smsType)
	}

	// 设置 channel
	if channel := r.channelConverter.ToEntity(data.Channel); channel != nil {
		builder.SetChannel(*channel)
	}

	// 设置 status
	if status := r.statusConverter.ToEntity(data.Status); status != nil {
		builder.SetStatus(*status)
	}

	if data.SentAt != nil {
		builder.SetSentAt(data.SentAt.AsTime())
	}

	if data.ExpiresAt != nil {
		builder.SetExpiresAt(data.ExpiresAt.AsTime())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert sms log failed: %s", err.Error())
		return nil, errors.InternalServer("INSERT_FAILED", "insert sms log failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// List 分页查询短信日志
func (r *smsLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListSMSLogsResponse, error) {
	if req == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().SMSLog.Query()

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
