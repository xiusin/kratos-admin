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
	"go-wind-admin/app/consumer/service/internal/data/ent/consumer"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// ConsumerRepo C端用户数据访问接口
type ConsumerRepo interface {
	// Create 创建用户
	Create(ctx context.Context, data *consumerV1.Consumer) (*consumerV1.Consumer, error)

	// Get 查询用户
	Get(ctx context.Context, id uint32) (*consumerV1.Consumer, error)

	// GetByPhone 按手机号查询用户
	GetByPhone(ctx context.Context, tenantID uint32, phone string) (*consumerV1.Consumer, error)

	// GetByWechatOpenID 按微信OpenID查询用户
	GetByWechatOpenID(ctx context.Context, tenantID uint32, openID string) (*consumerV1.Consumer, error)

	// Update 更新用户信息
	Update(ctx context.Context, id uint32, data *consumerV1.Consumer) error

	// List 分页查询用户列表
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListConsumersResponse, error)

	// Deactivate 注销账户
	Deactivate(ctx context.Context, id uint32) error

	// UpdateLoginInfo 更新登录信息
	UpdateLoginInfo(ctx context.Context, id uint32, ip string, failCount int32, lockedUntil *time.Time) error

	// ResetLoginFailCount 重置登录失败次数
	ResetLoginFailCount(ctx context.Context, id uint32) error
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

// NewConsumerRepo 创建C端用户数据访问实例
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
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Consumer.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableEmail(data.Email).
		SetNillableNickname(data.Nickname).
		SetNillableAvatar(data.Avatar).
		SetNillableWechatOpenid(data.WechatOpenid).
		SetNillableWechatUnionid(data.WechatUnionid).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetCreatedAt(time.Now())

	// 设置必填字段 phone
	if data.Phone != nil {
		builder.SetPhone(*data.Phone)
	}

	// 设置密码哈希（必填字段）- Consumer message 中没有 password 字段
	// 密码应该在创建前已经哈希处理
	builder.SetPasswordHash("") // 默认空密码，实际使用时需要在 service 层设置

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert consumer failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert consumer failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询用户
func (r *consumerRepo) Get(ctx context.Context, id uint32) (*consumerV1.Consumer, error) {
	entity, err := r.entClient.Client().Consumer.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("get consumer failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("get consumer failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// GetByPhone 按手机号查询用户
func (r *consumerRepo) GetByPhone(ctx context.Context, tenantID uint32, phone string) (*consumerV1.Consumer, error) {
	entity, err := r.entClient.Client().Consumer.Query().
		Where(
			consumer.TenantID(tenantID),
			consumer.Phone(phone),
		).
		Only(ctx)
	
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("get consumer by phone failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("get consumer by phone failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// GetByWechatOpenID 按微信OpenID查询用户
func (r *consumerRepo) GetByWechatOpenID(ctx context.Context, tenantID uint32, openID string) (*consumerV1.Consumer, error) {
	entity, err := r.entClient.Client().Consumer.Query().
		Where(
			consumer.TenantID(tenantID),
			consumer.WechatOpenid(openID),
		).
		Only(ctx)
	
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("get consumer by wechat openid failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("get consumer by wechat openid failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Update 更新用户信息
func (r *consumerRepo) Update(ctx context.Context, id uint32, data *consumerV1.Consumer) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Consumer.UpdateOneID(id).
		SetNillableNickname(data.Nickname).
		SetNillableAvatar(data.Avatar).
		SetNillableEmail(data.Email).
		SetNillableWechatOpenid(data.WechatOpenid).
		SetNillableWechatUnionid(data.WechatUnionid).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetUpdatedAt(time.Now())

	// phone 是必填字段，不能用 Nillable
	if data.Phone != nil {
		builder.SetPhone(*data.Phone)
	}

	if err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("update consumer failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("update consumer failed")
	}

	return nil
}

// List 分页查询用户列表
func (r *consumerRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListConsumersResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	query := r.entClient.Client().Consumer.Query()

	// 计算总数
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count consumers failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("count consumers failed")
	}

	// 分页查询
	var offset, limit int
	if req.Page != nil && req.PageSize != nil {
		offset = int((*req.Page - 1) * *req.PageSize)
		limit = int(*req.PageSize)
	} else {
		offset = 0
		limit = 10 // 默认每页10条
	}
	
	entities, err := query.
		Offset(offset).
		Limit(limit).
		Order(ent.Desc(consumer.FieldCreatedAt)).
		All(ctx)
	
	if err != nil {
		r.log.Errorf("list consumers failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("list consumers failed")
	}

	// 转换为 DTO
	items := make([]*consumerV1.Consumer, 0, len(entities))
	for _, entity := range entities {
		items = append(items, r.mapper.ToDTO(entity))
	}

	return &consumerV1.ListConsumersResponse{
		Total: uint64(total),
		Items: items,
	}, nil
}

// Deactivate 注销账户
func (r *consumerRepo) Deactivate(ctx context.Context, id uint32) error {
	now := time.Now()

	err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetStatus(consumer.StatusDeactivated).
		SetDeactivatedAt(now).
		SetUpdatedAt(now).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("deactivate consumer failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("deactivate consumer failed")
	}

	return nil
}

// UpdateLoginInfo 更新登录信息
func (r *consumerRepo) UpdateLoginInfo(ctx context.Context, id uint32, ip string, failCount int32, lockedUntil *time.Time) error {
	builder := r.entClient.Client().Consumer.UpdateOneID(id).
		SetLastLoginIP(ip).
		SetLoginFailCount(int(failCount)).
		SetUpdatedAt(time.Now())

	if lockedUntil != nil {
		builder.SetLockedUntil(*lockedUntil)
	} else {
		builder.ClearLockedUntil()
	}

	if err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("update login info failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("update login info failed")
	}

	return nil
}

// ResetLoginFailCount 重置登录失败次数
func (r *consumerRepo) ResetLoginFailCount(ctx context.Context, id uint32) error {
	now := time.Now()

	err := r.entClient.Client().Consumer.UpdateOneID(id).
		SetLoginFailCount(0).
		ClearLockedUntil().
		SetLastLoginAt(now).
		SetUpdatedAt(now).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("consumer not found")
		}
		r.log.Errorf("reset login fail count failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("reset login fail count failed")
	}

	return nil
}
