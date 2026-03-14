package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/middleware"
	"go-wind-admin/pkg/payment"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

const (
	// 支付订单超时时间（分钟）
	paymentOrderTimeout = 30
	// 事件类型
	eventTypePaymentSuccess = "consumer.payment.success"
)

type PaymentService struct {
	consumerV1.UnimplementedPaymentServiceServer

	paymentOrderRepo data.PaymentOrderRepo
	paymentClients   map[string]payment.Client // 支付客户端映射: wechat, alipay, yeepay
	eventBus         eventbus.EventBus

	log *log.Helper
}

func NewPaymentService(
	ctx *bootstrap.Context,
	paymentOrderRepo data.PaymentOrderRepo,
	wechatClient payment.Client,
	alipayClient payment.Client,
	yeepayClient payment.Client,
	eventBus eventbus.EventBus,
) *PaymentService {
	// 初始化支付客户端映射
	paymentClients := make(map[string]payment.Client)
	if wechatClient != nil {
		paymentClients["wechat"] = wechatClient
	}
	if alipayClient != nil {
		paymentClients["alipay"] = alipayClient
	}
	if yeepayClient != nil {
		paymentClients["yeepay"] = yeepayClient
	}

	return &PaymentService{
		log:              ctx.NewLoggerHelper("consumer/service/payment-service"),
		paymentOrderRepo: paymentOrderRepo,
		paymentClients:   paymentClients,
		eventBus:         eventBus,
	}
}

// generateOrderNo 生成全局唯一订单号
// 格式: PAY{timestamp}{uuid前8位}
func (s *PaymentService) generateOrderNo() string {
	timestamp := time.Now().Format("20060102150405")
	uuidStr := uuid.New().String()
	uuidShort := uuidStr[:8]
	return fmt.Sprintf("PAY%s%s", timestamp, uuidShort)
}

// CreatePayment 创建支付订单
func (s *PaymentService) CreatePayment(ctx context.Context, req *consumerV1.CreatePaymentRequest) (*consumerV1.CreatePaymentResponse, error) {
	// 1. 验证输入
	if req == nil || req.Amount == "" || req.PaymentMethod == consumerV1.PaymentOrder_PAYMENT_METHOD_UNSPECIFIED {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 验证金额格式
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return nil, consumerV1.ErrorBadRequest("invalid amount")
	}

	// 3. 获取租户ID和用户ID
	tenantID := middleware.GetTenantID(ctx)
	consumerID := middleware.GetUserID(ctx)

	// 4. 生成订单号
	orderNo := s.generateOrderNo()

	// 5. 计算过期时间（30分钟后）
	expiresAt := time.Now().Add(paymentOrderTimeout * time.Minute)

	// 6. 创建支付订单
	order := &consumerV1.PaymentOrder{
		TenantId:      &tenantID,
		OrderNo:       &orderNo,
		ConsumerId:    &consumerID,
		PaymentMethod: &req.PaymentMethod,
		PaymentType:   &req.PaymentType,
		Amount:        &req.Amount,
		Status:        consumerV1.PaymentOrder_STATUS_PENDING.Enum(),
		ExpiresAt:     timestamppb.New(expiresAt),
	}

	createdOrder, err := s.paymentOrderRepo.Create(ctx, order)
	if err != nil {
		s.log.Errorf("create payment order failed: %v", err)
		return nil, err
	}

	// 7. 调用第三方支付接口
	paymentMethodStr := req.PaymentMethod.String()
	paymentClient, ok := s.paymentClients[paymentMethodStr]
	if !ok {
		s.log.Errorf("payment client not found for method: %s", paymentMethodStr)
		return nil, consumerV1.ErrorInternalServerError("payment method not supported")
	}

	// 8. 创建支付请求
	paymentReq := &payment.CreateOrderRequest{
		OrderNo:     orderNo,
		Amount:      req.Amount,
		Description: req.Description,
		NotifyURL:   req.NotifyUrl,
		ReturnURL:   req.ReturnUrl,
	}

	paymentResp, err := paymentClient.CreateOrder(ctx, paymentReq)
	if err != nil {
		s.log.Errorf("create payment failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("create payment failed")
	}

	s.log.Infof("payment order created: %s, amount: %s", orderNo, req.Amount)

	return &consumerV1.CreatePaymentResponse{
		Order:      createdOrder,
		PaymentUrl: paymentResp.PaymentURL,
		QrCode:     paymentResp.QRCode,
	}, nil
}

// GetPayment 查询支付订单
func (s *PaymentService) GetPayment(ctx context.Context, req *consumerV1.GetPaymentRequest) (*consumerV1.PaymentOrder, error) {
	// 1. 验证输入
	if req == nil || req.Id == 0 {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 查询订单
	order, err := s.paymentOrderRepo.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// 3. 验证租户隔离
	tenantID := middleware.GetTenantID(ctx)
	if order.TenantId != nil && *order.TenantId != tenantID {
		return nil, consumerV1.ErrorForbidden("access denied")
	}

	return order, nil
}

// QueryPaymentStatus 查询支付结果
func (s *PaymentService) QueryPaymentStatus(ctx context.Context, req *consumerV1.QueryPaymentStatusRequest) (*consumerV1.PaymentStatusResponse, error) {
	// 1. 验证输入
	if req == nil || req.OrderNo == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 查询订单
	tenantID := middleware.GetTenantID(ctx)
	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, tenantID, req.OrderNo)
	if err != nil {
		return nil, err
	}

	// 3. 如果订单已经是终态，直接返回
	if order.Status != nil && (*order.Status == consumerV1.PaymentOrder_STATUS_SUCCESS ||
		*order.Status == consumerV1.PaymentOrder_STATUS_FAILED ||
		*order.Status == consumerV1.PaymentOrder_STATUS_CLOSED ||
		*order.Status == consumerV1.PaymentOrder_STATUS_REFUNDED) {
		return &consumerV1.PaymentStatusResponse{
			Order:  order,
			Status: *order.Status,
		}, nil
	}

	// 4. 调用第三方支付接口查询状态
	paymentMethodStr := order.PaymentMethod.String()
	paymentClient, ok := s.paymentClients[paymentMethodStr]
	if !ok {
		s.log.Errorf("payment client not found for method: %s", paymentMethodStr)
		return nil, consumerV1.ErrorInternalServerError("payment method not supported")
	}

	queryResp, err := paymentClient.QueryOrder(ctx, &payment.QueryOrderRequest{
		OrderNo: req.OrderNo,
	})
	if err != nil {
		s.log.Errorf("query payment status failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("query payment status failed")
	}

	// 5. 更新订单状态
	if queryResp.Status == payment.OrderStatusSuccess {
		now := timestamppb.Now()
		order.Status = consumerV1.PaymentOrder_STATUS_SUCCESS.Enum()
		order.PaidAt = now
		order.TransactionId = &queryResp.TransactionID

		if err := s.paymentOrderRepo.Update(ctx, *order.Id, order); err != nil {
			s.log.Errorf("update payment order failed: %v", err)
		}

		// 发布支付成功事件
		s.publishPaymentSuccessEvent(ctx, order)
	} else if queryResp.Status == payment.OrderStatusFailed {
		order.Status = consumerV1.PaymentOrder_STATUS_FAILED.Enum()
		if err := s.paymentOrderRepo.Update(ctx, *order.Id, order); err != nil {
			s.log.Errorf("update payment order failed: %v", err)
		}
	}

	return &consumerV1.PaymentStatusResponse{
		Order:  order,
		Status: *order.Status,
	}, nil
}

// Refund 申请退款
func (s *PaymentService) Refund(ctx context.Context, req *consumerV1.RefundRequest) (*consumerV1.RefundResponse, error) {
	// 1. 验证输入
	if req == nil || req.OrderNo == "" || req.RefundAmount == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 验证退款金额
	refundAmount, err := decimal.NewFromString(req.RefundAmount)
	if err != nil || refundAmount.LessThanOrEqual(decimal.Zero) {
		return nil, consumerV1.ErrorBadRequest("invalid refund amount")
	}

	// 3. 查询订单
	tenantID := middleware.GetTenantID(ctx)
	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, tenantID, req.OrderNo)
	if err != nil {
		return nil, err
	}

	// 4. 验证订单状态（只有支付成功的订单才能退款）
	if order.Status == nil || *order.Status != consumerV1.PaymentOrder_STATUS_SUCCESS {
		return nil, consumerV1.ErrorBadRequest("order cannot be refunded")
	}

	// 5. 验证退款金额不超过订单金额
	orderAmount, _ := decimal.NewFromString(*order.Amount)
	if refundAmount.GreaterThan(orderAmount) {
		return nil, consumerV1.ErrorBadRequest("refund amount exceeds order amount")
	}

	// 6. 调用第三方支付接口申请退款
	paymentMethodStr := order.PaymentMethod.String()
	paymentClient, ok := s.paymentClients[paymentMethodStr]
	if !ok {
		s.log.Errorf("payment client not found for method: %s", paymentMethodStr)
		return nil, consumerV1.ErrorInternalServerError("payment method not supported")
	}

	refundReq := &payment.RefundRequest{
		OrderNo:      req.OrderNo,
		RefundAmount: req.RefundAmount,
		RefundReason: req.RefundReason,
	}

	refundResp, err := paymentClient.Refund(ctx, refundReq)
	if err != nil {
		s.log.Errorf("refund failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("refund failed")
	}

	// 7. 更新订单状态为已退款
	order.Status = consumerV1.PaymentOrder_STATUS_REFUNDED.Enum()
	if err := s.paymentOrderRepo.Update(ctx, *order.Id, order); err != nil {
		s.log.Errorf("update payment order failed: %v", err)
	}

	s.log.Infof("refund success: order=%s, amount=%s", req.OrderNo, req.RefundAmount)

	return &consumerV1.RefundResponse{
		RefundNo: refundResp.RefundNo,
		Status:   refundResp.Status,
	}, nil
}

// QueryRefundStatus 查询退款状态
func (s *PaymentService) QueryRefundStatus(ctx context.Context, req *consumerV1.QueryRefundStatusRequest) (*consumerV1.RefundStatusResponse, error) {
	// 1. 验证输入
	if req == nil || req.RefundNo == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 调用第三方支付接口查询退款状态
	// 注意：这里需要根据退款单号找到对应的支付方式
	// 简化处理，假设退款单号包含支付方式信息或从订单中获取
	// 实际项目中应该维护退款单号与订单的映射关系

	return &consumerV1.RefundStatusResponse{
		RefundNo: req.RefundNo,
		Status:   "processing", // 实际应该从第三方接口查询
	}, nil
}

// ListPayments 查询支付流水
func (s *PaymentService) ListPayments(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListPaymentsResponse, error) {
	// 1. 验证输入
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 查询支付流水
	resp, err := s.paymentOrderRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// HandlePaymentCallback 处理支付回调（内部方法）
func (s *PaymentService) HandlePaymentCallback(ctx context.Context, paymentMethod string, callbackData map[string]string) error {
	// 1. 获取支付客户端
	paymentClient, ok := s.paymentClients[paymentMethod]
	if !ok {
		s.log.Errorf("payment client not found for method: %s", paymentMethod)
		return consumerV1.ErrorInternalServerError("payment method not supported")
	}

	// 2. 验证回调签名
	if err := paymentClient.VerifyCallback(ctx, callbackData); err != nil {
		s.log.Errorf("verify callback signature failed: %v", err)
		return consumerV1.ErrorBadRequest("invalid signature")
	}

	// 3. 提取订单号和交易号
	orderNo := callbackData["order_no"]
	transactionID := callbackData["transaction_id"]
	status := callbackData["status"]

	// 4. 查询订单（需要从回调数据中获取租户ID，或者通过订单号查询）
	// 简化处理，假设可以从订单号中提取租户ID
	// 实际项目中应该有更完善的租户识别机制
	tenantID := uint32(1) // 临时处理
	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, tenantID, orderNo)
	if err != nil {
		s.log.Errorf("order not found: %s", orderNo)
		return err
	}

	// 5. 更新订单状态
	if status == "success" {
		now := timestamppb.Now()
		order.Status = consumerV1.PaymentOrder_STATUS_SUCCESS.Enum()
		order.PaidAt = now
		order.TransactionId = &transactionID

		if err := s.paymentOrderRepo.Update(ctx, *order.Id, order); err != nil {
			s.log.Errorf("update payment order failed: %v", err)
			return err
		}

		// 发布支付成功事件
		s.publishPaymentSuccessEvent(ctx, order)

		s.log.Infof("payment callback processed: order=%s, transaction=%s", orderNo, transactionID)
	} else if status == "failed" {
		order.Status = consumerV1.PaymentOrder_STATUS_FAILED.Enum()
		if err := s.paymentOrderRepo.Update(ctx, *order.Id, order); err != nil {
			s.log.Errorf("update payment order failed: %v", err)
			return err
		}
	}

	return nil
}

// CloseExpiredOrders 关闭超时订单（定时任务调用）
func (s *PaymentService) CloseExpiredOrders(ctx context.Context, tenantID uint32) error {
	affected, err := s.paymentOrderRepo.CloseExpiredOrders(ctx, tenantID)
	if err != nil {
		s.log.Errorf("close expired orders failed: %v", err)
		return err
	}

	s.log.Infof("closed %d expired orders for tenant %d", affected, tenantID)
	return nil
}

// publishPaymentSuccessEvent 发布支付成功事件
func (s *PaymentService) publishPaymentSuccessEvent(ctx context.Context, order *consumerV1.PaymentOrder) {
	if s.eventBus == nil {
		s.log.Warn("event bus not initialized, skip publishing event")
		return
	}

	event := map[string]interface{}{
		"event_type":     eventTypePaymentSuccess,
		"tenant_id":      order.TenantId,
		"consumer_id":    order.ConsumerId,
		"order_no":       order.OrderNo,
		"amount":         order.Amount,
		"payment_method": order.PaymentMethod.String(),
		"transaction_id": order.TransactionId,
		"paid_at":        order.PaidAt,
	}

	if err := s.eventBus.Publish(ctx, eventTypePaymentSuccess, event); err != nil {
		s.log.Errorf("publish payment success event failed: %v", err)
	} else {
		s.log.Infof("payment success event published: order=%s", *order.OrderNo)
	}
}
