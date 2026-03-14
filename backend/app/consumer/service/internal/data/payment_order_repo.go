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
	"go-wind-admin/app/consumer/service/internal/data/ent/paymentorder"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// PaymentOrderRepo 支付订单数据访问接口
type PaymentOrderRepo interface {
	// Create 创建支付订单
	Create(ctx context.Context, data *consumerV1.PaymentOrder) (*consumerV1.PaymentOrder, error)

	// Get 查询支付订单
	Get(ctx context.Context, id uint64) (*consumerV1.PaymentOrder, error)

	// GetByOrderNo 按订单号查询
	GetByOrderNo(ctx context.Context, tenantID uint32, orderNo string) (*consumerV1.PaymentOrder, error)

	// Update 更新订单状态
	Update(ctx context.Context, id uint64, data *consumerV1.PaymentOrder) error

	// List 分页查询支付流水
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListPaymentsResponse, error)

	// CloseExpiredOrders 关闭超时订单
	CloseExpiredOrders(ctx context.Context, tenantID uint32) (int, error)
}

type paymentOrderRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                 *mapper.CopierMapper[consumerV1.PaymentOrder, ent.PaymentOrder]
	statusConverter        *mapper.EnumTypeConverter[consumerV1.PaymentOrder_Status, paymentorder.Status]
	paymentMethodConverter *mapper.EnumTypeConverter[consumerV1.PaymentOrder_PaymentMethod, paymentorder.PaymentMethod]
	paymentTypeConverter   *mapper.EnumTypeConverter[consumerV1.PaymentOrder_PaymentType, paymentorder.PaymentType]

	repository *entCrud.Repository[
		ent.PaymentOrderQuery, ent.PaymentOrderSelect,
		ent.PaymentOrderCreate, ent.PaymentOrderCreateBulk,
		ent.PaymentOrderUpdate, ent.PaymentOrderUpdateOne,
		ent.PaymentOrderDelete,
		predicate.PaymentOrder,
		consumerV1.PaymentOrder, ent.PaymentOrder,
	]
}

// NewPaymentOrderRepo 创建支付订单数据访问实例
func NewPaymentOrderRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) PaymentOrderRepo {
	repo := &paymentOrderRepo{
		log:                    ctx.NewLoggerHelper("consumer/repo/payment-order"),
		entClient:              entClient,
		mapper:                 mapper.NewCopierMapper[consumerV1.PaymentOrder, ent.PaymentOrder](),
		statusConverter:        mapper.NewEnumTypeConverter[consumerV1.PaymentOrder_Status, paymentorder.Status](consumerV1.PaymentOrder_Status_name, consumerV1.PaymentOrder_Status_value),
		paymentMethodConverter: mapper.NewEnumTypeConverter[consumerV1.PaymentOrder_PaymentMethod, paymentorder.PaymentMethod](consumerV1.PaymentOrder_PaymentMethod_name, consumerV1.PaymentOrder_PaymentMethod_value),
		paymentTypeConverter:   mapper.NewEnumTypeConverter[consumerV1.PaymentOrder_PaymentType, paymentorder.PaymentType](consumerV1.PaymentOrder_PaymentType_name, consumerV1.PaymentOrder_PaymentType_value),
	}

	repo.init()

	return repo
}

func (r *paymentOrderRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PaymentOrderQuery, ent.PaymentOrderSelect,
		ent.PaymentOrderCreate, ent.PaymentOrderCreateBulk,
		ent.PaymentOrderUpdate, ent.PaymentOrderUpdateOne,
		ent.PaymentOrderDelete,
		predicate.PaymentOrder,
		consumerV1.PaymentOrder, ent.PaymentOrder,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.paymentMethodConverter.NewConverterPair())
	r.mapper.AppendConverters(r.paymentTypeConverter.NewConverterPair())
}

// Create 创建支付订单
func (r *paymentOrderRepo) Create(ctx context.Context, data *consumerV1.PaymentOrder) (*consumerV1.PaymentOrder, error) {
	if data == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PaymentOrder.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableOrderNo(data.OrderNo).
		SetNillableConsumerID(data.ConsumerId).
		SetNillablePaymentMethod(r.paymentMethodConverter.ToEntity(data.PaymentMethod)).
		SetNillablePaymentType(r.paymentTypeConverter.ToEntity(data.PaymentType)).
		SetNillableAmount(data.Amount).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetCreatedAt(time.Now())

	// 设置过期时间（如果提供）
	if data.ExpiresAt != nil {
		builder.SetExpiresAt(data.ExpiresAt.AsTime())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert payment order failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert payment order failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询支付订单
func (r *paymentOrderRepo) Get(ctx context.Context, id uint64) (*consumerV1.PaymentOrder, error) {
	builder := r.entClient.Client().PaymentOrder.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(paymentorder.FieldID, id))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetByOrderNo 按订单号查询
func (r *paymentOrderRepo) GetByOrderNo(ctx context.Context, tenantID uint32, orderNo string) (*consumerV1.PaymentOrder, error) {
	builder := r.entClient.Client().PaymentOrder.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(paymentorder.FieldTenantID, tenantID),
				sql.EQ(paymentorder.FieldOrderNo, orderNo),
			))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// Update 更新订单状态
func (r *paymentOrderRepo) Update(ctx context.Context, id uint64, data *consumerV1.PaymentOrder) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PaymentOrder.UpdateOneID(id).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableTransactionID(data.TransactionId).
		SetNillableCallbackData(data.CallbackData).
		SetUpdatedAt(time.Now())

	// 设置支付时间（如果提供）
	if data.PaidAt != nil {
		builder.SetPaidAt(data.PaidAt.AsTime())
	}

	// 设置关闭时间（如果提供）
	if data.ClosedAt != nil {
		builder.SetClosedAt(data.ClosedAt.AsTime())
	}

	if err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("payment order not found")
		}
		r.log.Errorf("update payment order failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("update payment order failed")
	}

	return nil
}

// List 分页查询支付流水
func (r *paymentOrderRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListPaymentsResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PaymentOrder.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListPaymentsResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListPaymentsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// CloseExpiredOrders 关闭超时订单
func (r *paymentOrderRepo) CloseExpiredOrders(ctx context.Context, tenantID uint32) (int, error) {
	now := time.Now()

	// 查询所有超时且状态为PENDING的订单
	affected, err := r.entClient.Client().PaymentOrder.Update().
		Where(
			paymentorder.And(
				paymentorder.TenantIDEQ(tenantID),
				paymentorder.StatusEQ(paymentorder.StatusPENDING),
				paymentorder.ExpiresAtLT(now),
			),
		).
		SetStatus(paymentorder.StatusCLOSED).
		SetClosedAt(now).
		SetUpdatedAt(now).
		Save(ctx)

	if err != nil {
		r.log.Errorf("close expired orders failed: %s", err.Error())
		return 0, consumerV1.ErrorInternalServerError("close expired orders failed")
	}

	r.log.Infof("closed %d expired orders for tenant %d", affected, tenantID)
	return affected, nil
}
