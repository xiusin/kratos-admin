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
	GetByOrderNo(ctx context.Context, orderNo string) (*consumerV1.PaymentOrder, error)

	// Update 更新订单状态
	Update(ctx context.Context, id uint64, data *consumerV1.PaymentOrder) error

	// List 分页查询支付流水
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListPaymentsResponse, error)

	// CloseExpiredOrders 关闭超时订单
	CloseExpiredOrders(ctx context.Context) (int, error)
}

type paymentOrderRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                 *mapper.CopierMapper[consumerV1.PaymentOrder, ent.PaymentOrder]
	paymentMethodConverter *mapper.EnumTypeConverter[consumerV1.PaymentOrder_PaymentMethod, paymentorder.PaymentMethod]
	paymentTypeConverter   *mapper.EnumTypeConverter[consumerV1.PaymentOrder_PaymentType, paymentorder.PaymentType]
	statusConverter        *mapper.EnumTypeConverter[consumerV1.PaymentOrder_Status, paymentorder.Status]

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
		log:                    ctx.NewLoggerHelper("payment-order/repo/consumer-service"),
		entClient:              entClient,
		mapper:                 mapper.NewCopierMapper[consumerV1.PaymentOrder, ent.PaymentOrder](),
		paymentMethodConverter: mapper.NewEnumTypeConverter[consumerV1.PaymentOrder_PaymentMethod, paymentorder.PaymentMethod](consumerV1.PaymentOrder_PaymentMethod_name, consumerV1.PaymentOrder_PaymentMethod_value),
		paymentTypeConverter:   mapper.NewEnumTypeConverter[consumerV1.PaymentOrder_PaymentType, paymentorder.PaymentType](consumerV1.PaymentOrder_PaymentType_name, consumerV1.PaymentOrder_PaymentType_value),
		statusConverter:        mapper.NewEnumTypeConverter[consumerV1.PaymentOrder_Status, paymentorder.Status](consumerV1.PaymentOrder_Status_name, consumerV1.PaymentOrder_Status_value),
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
	r.mapper.AppendConverters(r.paymentMethodConverter.NewConverterPair())
	r.mapper.AppendConverters(r.paymentTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// Create 创建支付订单
func (r *paymentOrderRepo) Create(ctx context.Context, data *consumerV1.PaymentOrder) (*consumerV1.PaymentOrder, error) {
	if data == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().PaymentOrder.Create().
		SetNillableTenantID(data.TenantId).
		SetOrderNo(data.GetOrderNo()).
		SetConsumerID(data.GetConsumerId()).
		SetAmount(data.GetAmount()).
		SetExpiresAt(data.ExpiresAt.AsTime())

	// 设置 payment_method
	if paymentMethod := r.paymentMethodConverter.ToEntity(data.PaymentMethod); paymentMethod != nil {
		builder.SetPaymentMethod(*paymentMethod)
	}

	// 设置 payment_type
	if paymentType := r.paymentTypeConverter.ToEntity(data.PaymentType); paymentType != nil {
		builder.SetPaymentType(*paymentType)
	}

	// 设置 status
	if status := r.statusConverter.ToEntity(data.Status); status != nil {
		builder.SetStatus(*status)
	}

	if data.TransactionId != nil {
		builder.SetTransactionID(*data.TransactionId)
	}

	if data.CallbackData != nil {
		builder.SetCallbackData(*data.CallbackData)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert payment order failed: %s", err.Error())
		return nil, errors.InternalServer("INSERT_FAILED", "insert payment order failed")
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
func (r *paymentOrderRepo) GetByOrderNo(ctx context.Context, orderNo string) (*consumerV1.PaymentOrder, error) {
	builder := r.entClient.Client().PaymentOrder.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(paymentorder.FieldOrderNo, orderNo))
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
		return errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().PaymentOrder.UpdateOneID(uint32(id)).
		SetUpdatedAt(time.Now())

	if data.Status != nil {
		status := r.statusConverter.ToEntity(data.Status)
		if status != nil {
			builder.SetStatus(*status)
		}
	}

	if data.TransactionId != nil {
		builder.SetTransactionID(*data.TransactionId)
	}

	if data.CallbackData != nil {
		builder.SetCallbackData(*data.CallbackData)
	}

	if data.PaidAt != nil {
		builder.SetPaidAt(data.PaidAt.AsTime())
	}

	if data.ClosedAt != nil {
		builder.SetClosedAt(data.ClosedAt.AsTime())
	}

	if _, err := builder.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("PAYMENT_ORDER_NOT_FOUND", "payment order not found")
		}
		r.log.Errorf("update payment order failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "update payment order failed")
	}

	return nil
}

// List 分页查询支付流水
func (r *paymentOrderRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListPaymentsResponse, error) {
	if req == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
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
func (r *paymentOrderRepo) CloseExpiredOrders(ctx context.Context) (int, error) {
	now := time.Now()

	// 查询所有超时且状态为PENDING的订单
	count, err := r.entClient.Client().PaymentOrder.Update().
		Where(
			paymentorder.StatusEQ(paymentorder.StatusPending),
			paymentorder.ExpiresAtLT(now),
		).
		SetStatus(paymentorder.StatusClosed).
		SetClosedAt(now).
		SetUpdatedAt(now).
		Save(ctx)

	if err != nil {
		r.log.Errorf("close expired orders failed: %s", err.Error())
		return 0, errors.InternalServer("CLOSE_FAILED", "close expired orders failed")
	}

	if count > 0 {
		r.log.Infof("closed %d expired payment orders", count)
	}

	return count, nil
}
