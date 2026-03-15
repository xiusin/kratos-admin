package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/consumer"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// ConsumerRepo 用户数据访问接口
type ConsumerRepo interface {
	// Create 创建用户
	Create(ctx context.Context, data *consumerV1.Consumer) (*consumerV1.Consumer, error)

	// Get 查询用户
	Get(ctx context.Context, id uint32) (*consumerV1.Consumer, error)

	// GetByPhone 按手机号查询用户
	GetByPhone(ctx context.Context, phone string) (*consumerV1.Consumer, error)

	// Update 更新用户信息
	Update(ctx context.Context, id uint32, data *consumerV1.Consumer) error

	// List 分页查询用户列表
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListConsumersResponse, error)

	// Deactivate 注销账户
	Deactivate(ctx context.Context, id uint32) error

	// UpdateLoginInfo 更新登录信息
	UpdateLoginInfo(ctx context.Context, id uint32, loginIP string, loginAt time.Time) error

	// IncrementLoginFailCount 增加登录失败次数
	IncrementLoginFailCount(ctx context.Context, id uint32) error

	// ResetLoginFailCount 重置登录失败次数
	ResetLoginFailCount(ctx context.Context, id uint32) error

	// LockAccount 锁定账户
	LockAccount(ctx context.Context, id uint32, lockedUntil time.Time) error

	// UpdateRiskScore 更新风险评分
	UpdateRiskScore(ctx context.Context, id uint32, score int32) error
}

type consumerRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[consumerV1.Consumer, ent.Consumer]
	statusConverter *mapper.EnumTypeConverter[consumerV1.Consumer_Status, consumer.Status]

	repository *entCrud.Repository[
		ent.ConsumerQuery, ent.ConsumerSelect,
		ent.ConsumerCreate, ent.ConsumerCreateBulk,
		ent.ConsumerUpdate, ent.ConsumerUpdateOne,
		ent.ConsumerDelete,
		predicate.Consumer,
		consumerV1.Consumer, ent.Consumer,
	]
}

// NewConsumerRepo 创建用户数据访问实例
func NewConsumerRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) ConsumerRepo {
	repo := &consumerRepo{
		log:             ctx.NewLoggerHelper("consumer/repo/consumer-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[consumerV1.Consumer, ent.Consumer](),
		statusConverter: mapper.NewEnumTypeConverter[consumerV1.Consumer_Status, consumer.Status](consumerV1.Consumer_Status_name, consumerV1.Consumer_Status_value),
	}

	repo.init()

	return repo
}

func (r *consumerRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.ConsumerQuery, ent.ConsumerSelect,
		ent.ConsumerCreate, ent.ConsumerCreateBulk,
		ent.ConsumerUpdate, ent.ConsumerUpdateOne,
		ent.ConsumerDelete,
		predicate.Consumer,
		consumerV1.Consumer, ent.Consumer,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// Create 创建用户
func (r *consumerRepo) Create(ctx context.Context, data *consumerV1.Consumer) (*consumerV1.Consumer, error) {
	if data == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().Consumer.Create().
		SetNillableTenantID(data.TenantId).
		SetPhone(data.GetPhone()).
		SetNillableEmail(data.Email).
		SetNillableNickname(data.Nickname).
		SetNillableAvatar(data.Avatar).
		SetNillableWechatOpenid(data.WechatOpenid).
		SetNillableWechatUnionid(data.WechatUnionid).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetCreatedAt(time.Now())

	// 设置 risk_score (int32 -> int)
	if data.RiskScore != nil {
		builder.SetRiskScore(int(*data.RiskScore))
	}

	// 设置 login_fail_count (int32 -> int)
	if data.LoginFailCount != nil {
		builder.SetLoginFailCount(int(*data.LoginFailCount))
	}

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert consumer failed: %s", err.Error())
		return nil, errors.InternalServer("INSERT_FAILED", "insert consumer failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询用户
func (r *consumerRepo) Get(ctx context.Context, id uint32) (*consumerV1.Consumer, error) {
	builder := r.entClient.Client().Consumer.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(consumer.FieldID, id))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetByPhone 按手机号查询用户
func (r *consumerRepo) GetByPhone(ctx context.Context, phone string) (*consumerV1.Consumer, error) {
	builder := r.entClient.Client().Consumer.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(consumer.FieldPhone, phone))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// Update 更新用户信息
func (r *consumerRepo) Update(ctx context.Context, id uint32, data *consumerV1.Consumer) error {
	if data == nil {
		return errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().Consumer.UpdateOneID(id).
		SetNillableNickname(data.Nickname).
		SetNillableAvatar(data.Avatar).
		SetNillableEmail(data.Email).
		SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		r.log.Errorf("update consumer failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "update consumer failed")
	}

	return nil
}

// List 分页查询用户列表
func (r *consumerRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListConsumersResponse, error) {
	if req == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().Consumer.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListConsumersResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListConsumersResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// Deactivate 注销账户
func (r *consumerRepo) Deactivate(ctx context.Context, id uint32) error {
	now := time.Now()
	_, err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetStatus(consumer.StatusDeactivated).
		SetDeactivatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		r.log.Errorf("deactivate consumer failed: %s", err.Error())
		return errors.InternalServer("DEACTIVATE_FAILED", "deactivate consumer failed")
	}

	return nil
}

// UpdateLoginInfo 更新登录信息
func (r *consumerRepo) UpdateLoginInfo(ctx context.Context, id uint32, loginIP string, loginAt time.Time) error {
	_, err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetLastLoginAt(loginAt).
		SetLastLoginIP(loginIP).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		r.log.Errorf("update login info failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "update login info failed")
	}

	return nil
}

// IncrementLoginFailCount 增加登录失败次数
func (r *consumerRepo) IncrementLoginFailCount(ctx context.Context, id uint32) error {
	entity, err := r.entClient.Client().Consumer.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		return errors.InternalServer("GET_FAILED", "get consumer failed")
	}

	_, err = r.entClient.Client().Consumer.UpdateOneID(id).
		SetLoginFailCount(entity.LoginFailCount + 1).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		r.log.Errorf("increment login fail count failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "increment login fail count failed")
	}

	return nil
}

// ResetLoginFailCount 重置登录失败次数
func (r *consumerRepo) ResetLoginFailCount(ctx context.Context, id uint32) error {
	_, err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetLoginFailCount(0).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		r.log.Errorf("reset login fail count failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "reset login fail count failed")
	}

	return nil
}

// LockAccount 锁定账户
func (r *consumerRepo) LockAccount(ctx context.Context, id uint32, lockedUntil time.Time) error {
	_, err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetStatus(consumer.StatusLocked).
		SetLockedUntil(lockedUntil).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		r.log.Errorf("lock account failed: %s", err.Error())
		return errors.InternalServer("LOCK_FAILED", "lock account failed")
	}

	return nil
}

// UpdateRiskScore 更新风险评分
func (r *consumerRepo) UpdateRiskScore(ctx context.Context, id uint32, score int32) error {
	_, err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetRiskScore(int(score)).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("CONSUMER_NOT_FOUND", "consumer not found")
		}
		r.log.Errorf("update risk score failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "update risk score failed")
	}

	return nil
}
