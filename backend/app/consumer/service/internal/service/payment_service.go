package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/payment"
)

// PaymentService 支付服务
type PaymentService struct {
	consumerV1.UnimplementedPaymentServiceServer

	paymentOrderRepo data.PaymentOrderRepo
	paymentClient    payment.Client
	eventBus         eventbus.EventBus
	log              *log.Helper
}

// NewPaymentService 创建支付服务实例
func NewPaymentService(
	ctx *bootstrap.Context,
	paymentOrderRepo data.PaymentOrderRepo,
	paymentClient payment.Client,
	eventBus eventbus.EventBus,
) *PaymentService {
	return &PaymentService{
		paymentOrderRepo: paymentOrderRepo,
		paymentClient:    paymentClient,
		eventBus:         eventBus,
		log:              ctx.NewLoggerHelper("consumer/service/payment-service"),
	}
}

// CreatePayment 创建支付订单
func (s *PaymentService) CreatePayment(ctx context.Context, req *consumerV1.CreatePaymentRequest) (*consumerV1.CreatePaymentResponse, error) {
	s.log.Infof("CreatePayment: payment_method=%s, amount=%s", req.GetPaymentMethod(), req.GetAmount())

	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	// 生成订单号（全局唯一）
	orderNo := s.generateOrderNo()

	// 解析金额（元转分）
	amountYuan, err := strconv.ParseFloat(req.GetAmount(), 64)
	if err != nil {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "invalid amount format")
	}
	amountFen := int64(amountYuan * 100)

	// 设置订单过期时间（30分钟）
	expiresAt := time.Now().Add(30 * time.Minute)

	// 创建支付订单记录
	paymentOrder := &consumerV1.PaymentOrder{
		OrderNo:       &orderNo,
		ConsumerId:    &currentUserID,
		PaymentMethod: &req.PaymentMethod,
		PaymentType:   &req.PaymentType,
		Amount:        &req.Amount,
		Status:        consumerV1.PaymentOrder_PENDING.Enum(),
		ExpiresAt:     timestamppb.New(expiresAt),
	}

	_, err = s.paymentOrderRepo.Create(ctx, paymentOrder)
	if err != nil {
		s.log.Errorf("create payment order failed: %v", err)
		return nil, err
	}

	// 调用第三方支付接口创建订单
	paymentType := s.convertPaymentType(req.GetPaymentType())
	paymentReq := &payment.CreateOrderRequest{
		OrderNo:     orderNo,
		Amount:      amountFen,
		Currency:    "CNY",
		Subject:     req.GetSubject(),
		Description: req.GetBody(),
		PaymentType: paymentType,
		ExpireTime:  expiresAt,
		Extra: map[string]string{
			"notify_url": req.GetNotifyUrl(),
			"return_url": req.GetReturnUrl(),
		},
	}

	paymentResp, err := s.paymentClient.CreateOrder(ctx, paymentReq)
	if err != nil {
		s.log.Errorf("create payment order in third-party failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "create payment order failed")
	}

	// 构建支付参数
	payInfo := s.buildPayInfo(paymentResp, req.GetPaymentType())

	s.log.Infof("CreatePayment success: order_no=%s", orderNo)
	return &consumerV1.CreatePaymentResponse{
		OrderNo: orderNo,
		PayInfo: payInfo,
	}, nil
}

// GetPayment 查询支付订单
func (s *PaymentService) GetPayment(ctx context.Context, req *consumerV1.GetPaymentRequest) (*consumerV1.PaymentOrder, error) {
	s.log.Infof("GetPayment: order_no=%s", req.GetOrderNo())

	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		s.log.Errorf("get payment order failed: %v", err)
		return nil, err
	}

	return order, nil
}

// QueryPaymentStatus 查询支付结果
func (s *PaymentService) QueryPaymentStatus(ctx context.Context, req *consumerV1.QueryPaymentStatusRequest) (*consumerV1.PaymentStatusResponse, error) {
	s.log.Infof("QueryPaymentStatus: order_no=%s", req.GetOrderNo())

	// 查询本地订单
	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		s.log.Errorf("get payment order failed: %v", err)
		return nil, err
	}

	// 如果订单已经是终态，直接返回
	if order.GetStatus() == consumerV1.PaymentOrder_SUCCESS ||
		order.GetStatus() == consumerV1.PaymentOrder_FAILED ||
		order.GetStatus() == consumerV1.PaymentOrder_CLOSED ||
		order.GetStatus() == consumerV1.PaymentOrder_REFUNDED {
		return &consumerV1.PaymentStatusResponse{
			Status:        order.GetStatus(),
			TransactionId: order.TransactionId,
			PaidAt:        order.PaidAt,
		}, nil
	}

	// 查询第三方支付状态
	queryResp, err := s.paymentClient.QueryOrder(ctx, req.GetOrderNo())
	if err != nil {
		s.log.Errorf("query payment status from third-party failed: %v", err)
		// 返回本地状态
		return &consumerV1.PaymentStatusResponse{
			Status:        order.GetStatus(),
			TransactionId: order.TransactionId,
			PaidAt:        order.PaidAt,
		}, nil
	}

	// 更新本地订单状态
	if queryResp.Status == payment.OrderStatusSuccess {
		updateData := &consumerV1.PaymentOrder{
			Status:        consumerV1.PaymentOrder_SUCCESS.Enum(),
			TransactionId: &queryResp.TransactionID,
		}
		if queryResp.PaidAt != nil {
			updateData.PaidAt = timestamppb.New(*queryResp.PaidAt)
		}

		if err := s.paymentOrderRepo.Update(ctx, order.GetId(), updateData); err != nil {
			s.log.Errorf("update payment order status failed: %v", err)
		} else {
			// 发布支付成功事件
			s.publishPaymentSuccessEvent(ctx, order)
		}

		return &consumerV1.PaymentStatusResponse{
			Status:        consumerV1.PaymentOrder_SUCCESS.Enum(),
			TransactionId: &queryResp.TransactionID,
			PaidAt:        timestamppb.New(*queryResp.PaidAt),
		}, nil
	}

	return &consumerV1.PaymentStatusResponse{
		Status:        order.GetStatus(),
		TransactionId: order.TransactionId,
		PaidAt:        order.PaidAt,
	}, nil
}

// Refund 申请退款
func (s *PaymentService) Refund(ctx context.Context, req *consumerV1.RefundRequest) (*consumerV1.RefundResponse, error) {
	s.log.Infof("Refund: order_no=%s, refund_amount=%s", req.GetOrderNo(), req.GetRefundAmount())

	// 查询订单
	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, req.GetOrderNo())
	if err != nil {
		s.log.Errorf("get payment order failed: %v", err)
		return nil, err
	}

	// 检查订单状态
	if order.GetStatus() != consumerV1.PaymentOrder_SUCCESS {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "order is not paid")
	}

	// 解析退款金额
	refundAmountYuan, err := strconv.ParseFloat(req.GetRefundAmount(), 64)
	if err != nil {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "invalid refund amount format")
	}
	refundAmountFen := int64(refundAmountYuan * 100)

	// 解析订单金额
	orderAmountYuan, err := strconv.ParseFloat(order.GetAmount(), 64)
	if err != nil {
		return nil, errors.InternalServer("INTERNAL_ERROR", "invalid order amount format")
	}
	orderAmountFen := int64(orderAmountYuan * 100)

	// 检查退款金额
	if refundAmountFen > orderAmountFen {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "refund amount exceeds order amount")
	}

	// 生成退款单号
	refundNo := s.generateRefundNo()

	// 调用第三方支付接口申请退款
	refundReq := &payment.RefundRequest{
		OrderNo:     req.GetOrderNo(),
		RefundNo:    refundNo,
		Amount:      refundAmountFen,
		TotalAmount: orderAmountFen,
		Reason:      req.GetReason(),
	}

	refundResp, err := s.paymentClient.Refund(ctx, refundReq)
	if err != nil {
		s.log.Errorf("refund in third-party failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "refund failed")
	}

	// 更新订单状态为已退款
	if refundResp.Status == payment.RefundStatusSuccess {
		updateData := &consumerV1.PaymentOrder{
			Status: consumerV1.PaymentOrder_REFUNDED.Enum(),
		}
		if err := s.paymentOrderRepo.Update(ctx, order.GetId(), updateData); err != nil {
			s.log.Errorf("update payment order status failed: %v", err)
		}
	}

	// TODO: 记录退款流水到FinanceTransaction

	s.log.Infof("Refund success: refund_no=%s", refundNo)
	return &consumerV1.RefundResponse{
		RefundNo: refundNo,
		Status:   string(refundResp.Status),
	}, nil
}

// QueryRefundStatus 查询退款状态
func (s *PaymentService) QueryRefundStatus(ctx context.Context, req *consumerV1.QueryRefundStatusRequest) (*consumerV1.RefundStatusResponse, error) {
	s.log.Infof("QueryRefundStatus: refund_no=%s", req.GetRefundNo())

	// 查询第三方退款状态
	queryResp, err := s.paymentClient.QueryRefund(ctx, req.GetRefundNo())
	if err != nil {
		s.log.Errorf("query refund status from third-party failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "query refund status failed")
	}

	resp := &consumerV1.RefundStatusResponse{
		Status:       string(queryResp.Status),
		RefundAmount: nil,
		RefundedAt:   nil,
	}

	if queryResp.Amount > 0 {
		amountYuan := fmt.Sprintf("%.2f", float64(queryResp.Amount)/100)
		resp.RefundAmount = &amountYuan
	}

	if queryResp.SuccessTime != nil {
		resp.RefundedAt = timestamppb.New(*queryResp.SuccessTime)
	}

	return resp, nil
}

// ListPayments 查询支付流水
func (s *PaymentService) ListPayments(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListPaymentsResponse, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("ListPayments: consumer_id=%d", currentUserID)

	// TODO: 添加consumer_id过滤条件到req
	// 这里暂时直接查询

	resp, err := s.paymentOrderRepo.List(ctx, req)
	if err != nil {
		s.log.Errorf("list payments failed: %v", err)
		return nil, err
	}

	return resp, nil
}

// generateOrderNo 生成订单号（全局唯一）
func (s *PaymentService) generateOrderNo() string {
	// 格式：PAY + 时间戳(14位) + 随机数(6位)
	// 例如：PAY20260315143025123456
	now := time.Now()
	timestamp := now.Format("20060102150405")
	random := now.UnixNano() % 1000000
	return fmt.Sprintf("PAY%s%06d", timestamp, random)
}

// generateRefundNo 生成退款单号
func (s *PaymentService) generateRefundNo() string {
	// 格式：REF + 时间戳(14位) + 随机数(6位)
	// 例如：REF20260315143025123456
	now := time.Now()
	timestamp := now.Format("20060102150405")
	random := now.UnixNano() % 1000000
	return fmt.Sprintf("REF%s%06d", timestamp, random)
}

// convertPaymentType 转换支付类型
func (s *PaymentService) convertPaymentType(pbType consumerV1.PaymentOrder_PaymentType) payment.PaymentType {
	switch pbType {
	case consumerV1.PaymentOrder_APP:
		return payment.PaymentTypeApp
	case consumerV1.PaymentOrder_H5:
		return payment.PaymentTypeH5
	case consumerV1.PaymentOrder_MINI:
		return payment.PaymentTypeMini
	case consumerV1.PaymentOrder_QRCODE:
		return payment.PaymentTypeQRCode
	default:
		return payment.PaymentTypeApp
	}
}

// buildPayInfo 构建支付参数
func (s *PaymentService) buildPayInfo(resp *payment.CreateOrderResponse, paymentType consumerV1.PaymentOrder_PaymentType) string {
	var payInfo map[string]interface{}

	switch paymentType {
	case consumerV1.PaymentOrder_APP:
		payInfo = map[string]interface{}{
			"type": "app",
			"data": resp.AppPayData,
		}
	case consumerV1.PaymentOrder_H5:
		payInfo = map[string]interface{}{
			"type": "h5",
			"url":  resp.H5URL,
		}
	case consumerV1.PaymentOrder_MINI:
		payInfo = map[string]interface{}{
			"type": "mini",
			"data": resp.MiniPayData,
		}
	case consumerV1.PaymentOrder_QRCODE:
		payInfo = map[string]interface{}{
			"type":     "qrcode",
			"code_url": resp.CodeURL,
		}
	default:
		payInfo = map[string]interface{}{
			"type": "unknown",
		}
	}

	jsonBytes, _ := json.Marshal(payInfo)
	return string(jsonBytes)
}

// publishPaymentSuccessEvent 发布支付成功事件
func (s *PaymentService) publishPaymentSuccessEvent(ctx context.Context, order *consumerV1.PaymentOrder) {
	event := eventbus.NewEvent("payment.success", map[string]interface{}{
		"order_no":    order.GetOrderNo(),
		"consumer_id": order.GetConsumerId(),
		"amount":      order.GetAmount(),
		"tenant_id":   order.GetTenantId(),
	}).WithSource("payment-service")

	if err := s.eventBus.PublishAsync(ctx, event); err != nil {
		s.log.Errorf("publish payment success event failed: %v", err)
	}
}

// HandlePaymentCallback 处理支付回调（内部方法，由HTTP回调接口调用）
func (s *PaymentService) HandlePaymentCallback(ctx context.Context, callbackData map[string]string) error {
	s.log.Infof("HandlePaymentCallback: data=%v", callbackData)

	// 验证签名
	valid, err := s.paymentClient.VerifyCallback(ctx, callbackData)
	if err != nil {
		s.log.Errorf("verify callback signature failed: %v", err)
		return errors.InternalServer("INTERNAL_ERROR", "verify callback signature failed")
	}

	if !valid {
		s.log.Warnf("invalid callback signature")
		return errors.BadRequest("INVALID_ARGUMENT", "invalid callback signature")
	}

	// 提取订单号
	orderNo, ok := callbackData["order_no"]
	if !ok {
		return errors.BadRequest("INVALID_ARGUMENT", "missing order_no in callback data")
	}

	// 查询订单
	order, err := s.paymentOrderRepo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		s.log.Errorf("get payment order failed: %v", err)
		return err
	}

	// 如果订单已经是终态，直接返回
	if order.GetStatus() != consumerV1.PaymentOrder_PENDING {
		s.log.Infof("order already in final status: order_no=%s, status=%s", orderNo, order.GetStatus())
		return nil
	}

	// 提取支付状态
	status, ok := callbackData["status"]
	if !ok {
		return errors.BadRequest("INVALID_ARGUMENT", "missing status in callback data")
	}

	// 更新订单状态
	updateData := &consumerV1.PaymentOrder{}

	if status == "success" {
		updateData.Status = consumerV1.PaymentOrder_SUCCESS.Enum()
		if transactionID, ok := callbackData["transaction_id"]; ok {
			updateData.TransactionId = &transactionID
		}
		updateData.PaidAt = timestamppb.Now()

		// 保存回调数据
		callbackJSON, _ := json.Marshal(callbackData)
		callbackStr := string(callbackJSON)
		updateData.CallbackData = &callbackStr

		if err := s.paymentOrderRepo.Update(ctx, order.GetId(), updateData); err != nil {
			s.log.Errorf("update payment order status failed: %v", err)
			return err
		}

		// 发布支付成功事件
		s.publishPaymentSuccessEvent(ctx, order)

		s.log.Infof("payment callback success: order_no=%s", orderNo)
	} else if status == "failed" {
		updateData.Status = consumerV1.PaymentOrder_FAILED.Enum()

		if err := s.paymentOrderRepo.Update(ctx, order.GetId(), updateData); err != nil {
			s.log.Errorf("update payment order status failed: %v", err)
			return err
		}

		s.log.Infof("payment callback failed: order_no=%s", orderNo)
	}

	return nil
}

// CloseExpiredOrders 关闭超时订单（定时任务调用）
func (s *PaymentService) CloseExpiredOrders(ctx context.Context) error {
	s.log.Infof("CloseExpiredOrders: start")

	count, err := s.paymentOrderRepo.CloseExpiredOrders(ctx)
	if err != nil {
		s.log.Errorf("close expired orders failed: %v", err)
		return err
	}

	s.log.Infof("CloseExpiredOrders: closed %d orders", count)
	return nil
}
