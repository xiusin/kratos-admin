package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/eventbus"
)

// FinanceService 财务服务
type FinanceService struct {
	consumerV1.UnimplementedFinanceServiceServer

	financeAccountRepo     data.FinanceAccountRepo
	financeTransactionRepo data.FinanceTransactionRepo
	eventBus               eventbus.EventBus
	log                    *log.Helper
}

// NewFinanceService 创建财务服务实例
func NewFinanceService(
	ctx *bootstrap.Context,
	financeAccountRepo data.FinanceAccountRepo,
	financeTransactionRepo data.FinanceTransactionRepo,
	eventBus eventbus.EventBus,
) *FinanceService {
	svc := &FinanceService{
		financeAccountRepo:     financeAccountRepo,
		financeTransactionRepo: financeTransactionRepo,
		eventBus:               eventBus,
		log:                    ctx.NewLoggerHelper("consumer/service/finance-service"),
	}

	// 订阅用户注册事件
	svc.subscribeUserRegisteredEvent()

	// 订阅支付成功事件
	svc.subscribePaymentSuccessEvent()

	return svc
}

// GetAccount 获取账户余额
func (s *FinanceService) GetAccount(ctx context.Context, req *consumerV1.GetAccountRequest) (*consumerV1.FinanceAccount, error) {
	s.log.Infof("GetAccount: consumer_id=%v", req.GetConsumerId())

	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	// 如果请求中没有指定用户ID，则查询当前用户
	consumerID := req.GetConsumerId()
	if consumerID == 0 {
		consumerID = currentUserID
	}

	// 查询账户
	account, err := s.financeAccountRepo.GetByConsumerID(ctx, consumerID)
	if err != nil {
		s.log.Errorf("get finance account failed: %v", err)
		return nil, err
	}

	return account, nil
}

// Recharge 充值
func (s *FinanceService) Recharge(ctx context.Context, req *consumerV1.RechargeRequest) (*emptypb.Empty, error) {
	s.log.Infof("Recharge: amount=%s, payment_order_no=%s", req.GetAmount(), req.GetPaymentOrderNo())

	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	// 验证充值金额
	amount, err := decimal.NewFromString(req.GetAmount())
	if err != nil {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "invalid amount format")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "amount must be greater than 0")
	}

	// 查询账户
	account, err := s.financeAccountRepo.GetByConsumerID(ctx, currentUserID)
	if err != nil {
		s.log.Errorf("get finance account failed: %v", err)
		return nil, err
	}

	// 计算新余额
	balanceBefore, _ := decimal.NewFromString(account.GetBalance())
	balanceAfter := balanceBefore.Add(amount)

	// 更新余额
	if err := s.financeAccountRepo.UpdateBalance(ctx, account.GetId(), balanceAfter.String(), account.GetFrozenBalance()); err != nil {
		s.log.Errorf("update balance failed: %v", err)
		return nil, err
	}

	// 记录财务流水
	transactionNo := s.generateTransactionNo()
	transaction := &consumerV1.FinanceTransaction{
		TenantId:        account.TenantId,
		ConsumerId:      &currentUserID,
		TransactionNo:   &transactionNo,
		TransactionType: consumerV1.FinanceTransaction_RECHARGE.Enum(),
		Amount:          &req.Amount,
		BalanceBefore:   &account.Balance,
		BalanceAfter:    func() *string { s := balanceAfter.String(); return &s }(),
		Description:     func() *string { s := "充值"; return &s }(),
		RelatedOrderNo:  &req.PaymentOrderNo,
	}

	if _, err := s.financeTransactionRepo.Create(ctx, transaction); err != nil {
		s.log.Errorf("create finance transaction failed: %v", err)
		// 不返回错误，因为余额已经更新成功
	}

	s.log.Infof("recharge success: consumer_id=%d, amount=%s, balance_after=%s", currentUserID, req.GetAmount(), balanceAfter.String())

	return &emptypb.Empty{}, nil
}

// Withdraw 申请提现
func (s *FinanceService) Withdraw(ctx context.Context, req *consumerV1.WithdrawRequest) (*consumerV1.WithdrawResponse, error) {
	s.log.Infof("Withdraw: amount=%s", req.GetAmount())

	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	// 验证提现金额
	amount, err := decimal.NewFromString(req.GetAmount())
	if err != nil {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "invalid amount format")
	}

	// 提现金额限制：10-5000元
	minAmount := decimal.NewFromInt(10)
	maxAmount := decimal.NewFromInt(5000)
	if amount.LessThan(minAmount) {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "withdraw amount must be at least 10 yuan")
	}
	if amount.GreaterThan(maxAmount) {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "withdraw amount must not exceed 5000 yuan")
	}

	// 查询账户
	account, err := s.financeAccountRepo.GetByConsumerID(ctx, currentUserID)
	if err != nil {
		s.log.Errorf("get finance account failed: %v", err)
		return nil, err
	}

	// 检查余额是否充足
	balance, _ := decimal.NewFromString(account.GetBalance())
	if balance.LessThan(amount) {
		return nil, errors.BadRequest("INSUFFICIENT_BALANCE", "insufficient balance")
	}

	// 冻结提现金额
	frozenBalance, _ := decimal.NewFromString(account.GetFrozenBalance())
	newFrozenBalance := frozenBalance.Add(amount)

	// 更新余额（冻结）
	if err := s.financeAccountRepo.UpdateBalance(ctx, account.GetId(), account.GetBalance(), newFrozenBalance.String()); err != nil {
		s.log.Errorf("update balance failed: %v", err)
		return nil, err
	}

	// 生成提现单号
	withdrawNo := s.generateWithdrawNo()

	// 记录财务流水
	transactionNo := s.generateTransactionNo()
	transaction := &consumerV1.FinanceTransaction{
		TenantId:        account.TenantId,
		ConsumerId:      &currentUserID,
		TransactionNo:   &transactionNo,
		TransactionType: consumerV1.FinanceTransaction_WITHDRAW.Enum(),
		Amount:          &req.Amount,
		BalanceBefore:   &account.Balance,
		BalanceAfter:    &account.Balance, // 余额未变，只是冻结
		Description:     func() *string { s := fmt.Sprintf("提现申请：%s", withdrawNo); return &s }(),
		RelatedOrderNo:  &withdrawNo,
	}

	if _, err := s.financeTransactionRepo.Create(ctx, transaction); err != nil {
		s.log.Errorf("create finance transaction failed: %v", err)
		// 不返回错误，因为余额已经冻结成功
	}

	s.log.Infof("withdraw request created: consumer_id=%d, amount=%s, withdraw_no=%s", currentUserID, req.GetAmount(), withdrawNo)

	return &consumerV1.WithdrawResponse{
		WithdrawNo: withdrawNo,
		Status:     "pending",
	}, nil
}

// ApproveWithdraw 审核提现
func (s *FinanceService) ApproveWithdraw(ctx context.Context, req *consumerV1.ApproveWithdrawRequest) (*emptypb.Empty, error) {
	s.log.Infof("ApproveWithdraw: withdraw_no=%s, approved=%v", req.GetWithdrawNo(), req.GetApproved())

	// TODO: 验证管理员权限

	// TODO: 查询提现申请记录（这里简化处理，实际应该有提现申请表）
	// 这里假设从财务流水中查询

	// 如果审核通过
	if req.GetApproved() {
		// TODO: 扣减余额，解冻金额
		// TODO: 发起打款
		s.log.Infof("withdraw approved: withdraw_no=%s", req.GetWithdrawNo())
	} else {
		// 如果审核拒绝，解冻金额
		// TODO: 解冻金额
		s.log.Infof("withdraw rejected: withdraw_no=%s, reason=%s", req.GetWithdrawNo(), req.GetRejectReason())
	}

	return &emptypb.Empty{}, nil
}

// ListTransactions 查询财务流水
func (s *FinanceService) ListTransactions(ctx context.Context, req *consumerV1.ListTransactionsRequest) (*consumerV1.ListTransactionsResponse, error) {
	s.log.Infof("ListTransactions: consumer_id=%v, transaction_type=%v", req.GetConsumerId(), req.GetTransactionType())

	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	// 如果请求中没有指定用户ID，则查询当前用户
	if req.ConsumerId == nil {
		req.ConsumerId = &currentUserID
	}

	// 查询财务流水
	resp, err := s.financeTransactionRepo.List(ctx, req)
	if err != nil {
		s.log.Errorf("list finance transactions failed: %v", err)
		return nil, err
	}

	return resp, nil
}

// ExportTransactions 导出财务流水
func (s *FinanceService) ExportTransactions(ctx context.Context, req *consumerV1.ExportTransactionsRequest) (*consumerV1.ExportTransactionsResponse, error) {
	s.log.Infof("ExportTransactions: consumer_id=%v", req.GetConsumerId())

	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	// 如果请求中没有指定用户ID，则导出当前用户
	if req.ConsumerId == nil {
		req.ConsumerId = &currentUserID
	}

	// 导出财务流水
	fileURL, err := s.financeTransactionRepo.Export(ctx, req)
	if err != nil {
		s.log.Errorf("export finance transactions failed: %v", err)
		return nil, err
	}

	return &consumerV1.ExportTransactionsResponse{
		FileUrl: fileURL,
	}, nil
}

// subscribeUserRegisteredEvent 订阅用户注册事件
func (s *FinanceService) subscribeUserRegisteredEvent() {
	s.eventBus.Subscribe(eventbus.TopicUserRegistered, func(ctx context.Context, event eventbus.Event) error {
		s.log.Infof("received UserRegisteredEvent: %+v", event)

		// 解析事件数据
		userRegisteredEvent, ok := event.Data.(eventbus.UserRegisteredEvent)
		if !ok {
			s.log.Errorf("invalid event data type")
			return errors.InternalServer("INVALID_EVENT_DATA", "invalid event data type")
		}

		// 自动创建财务账户
		account := &consumerV1.FinanceAccount{
			TenantId:      &userRegisteredEvent.TenantID,
			ConsumerId:    &userRegisteredEvent.UserID,
			Balance:       func() *string { s := "0"; return &s }(),
			FrozenBalance: func() *string { s := "0"; return &s }(),
			CreatedAt:     timestamppb.Now(),
			UpdatedAt:     timestamppb.Now(),
		}

		if _, err := s.financeAccountRepo.Create(ctx, account); err != nil {
			s.log.Errorf("create finance account failed: %v", err)
			return err
		}

		s.log.Infof("finance account created for user: user_id=%d", userRegisteredEvent.UserID)

		return nil
	})
}

// subscribePaymentSuccessEvent 订阅支付成功事件
func (s *FinanceService) subscribePaymentSuccessEvent() {
	s.eventBus.Subscribe(eventbus.TopicPaymentSuccess, func(ctx context.Context, event eventbus.Event) error {
		s.log.Infof("received PaymentSuccessEvent: %+v", event)

		// 解析事件数据
		paymentSuccessEvent, ok := event.Data.(eventbus.PaymentSuccessEvent)
		if !ok {
			s.log.Errorf("invalid event data type")
			return errors.InternalServer("INVALID_EVENT_DATA", "invalid event data type")
		}

		// 自动充值
		amount := fmt.Sprintf("%.2f", float64(paymentSuccessEvent.Amount)/100.0) // 分转元
		rechargeReq := &consumerV1.RechargeRequest{
			Amount:         amount,
			PaymentOrderNo: paymentSuccessEvent.OrderNo,
		}

		if _, err := s.Recharge(ctx, rechargeReq); err != nil {
			s.log.Errorf("auto recharge failed: %v", err)
			return err
		}

		s.log.Infof("auto recharge success: order_no=%s, amount=%s", paymentSuccessEvent.OrderNo, amount)

		return nil
	})
}

// generateTransactionNo 生成流水号
func (s *FinanceService) generateTransactionNo() string {
	return fmt.Sprintf("TXN%s%06d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%1000000)
}

// generateWithdrawNo 生成提现单号
func (s *FinanceService) generateWithdrawNo() string {
	return fmt.Sprintf("WD%s%06d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%1000000)
}
