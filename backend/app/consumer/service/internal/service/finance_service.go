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

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

const (
	// 提现金额限制
	minWithdrawAmount = 10.0   // 最小提现金额（元）
	maxWithdrawAmount = 5000.0 // 单日最大提现金额（元）

	// 事件类型
	eventTypeUserRegistered = "consumer.user.registered"
	eventTypePaymentSuccess = "consumer.payment.success"
)

type FinanceService struct {
	consumerV1.UnimplementedFinanceServiceServer

	financeAccountRepo     data.FinanceAccountRepo
	financeTransactionRepo data.FinanceTransactionRepo
	eventBus               eventbus.EventBus

	log *log.Helper
}

func NewFinanceService(
	ctx *bootstrap.Context,
	financeAccountRepo data.FinanceAccountRepo,
	financeTransactionRepo data.FinanceTransactionRepo,
	eventBus eventbus.EventBus,
) *FinanceService {
	svc := &FinanceService{
		log:                    ctx.NewLoggerHelper("consumer/service/finance-service"),
		financeAccountRepo:     financeAccountRepo,
		financeTransactionRepo: financeTransactionRepo,
		eventBus:               eventBus,
	}

	// 订阅事件
	svc.subscribeEvents()

	return svc
}

// subscribeEvents 订阅事件
func (s *FinanceService) subscribeEvents() {
	if s.eventBus == nil {
		s.log.Warn("event bus not initialized, skip subscribing events")
		return
	}

	// 订阅用户注册事件，自动创建财务账户
	s.eventBus.Subscribe(eventTypeUserRegistered, s.handleUserRegisteredEvent)

	// 订阅支付成功事件，自动充值
	s.eventBus.Subscribe(eventTypePaymentSuccess, s.handlePaymentSuccessEvent)

	s.log.Info("finance service events subscribed")
}

// handleUserRegisteredEvent 处理用户注册事件
func (s *FinanceService) handleUserRegisteredEvent(ctx context.Context, event interface{}) error {
	eventData, ok := event.(map[string]interface{})
	if !ok {
		s.log.Error("invalid user registered event data")
		return fmt.Errorf("invalid event data")
	}

	tenantID, _ := eventData["tenant_id"].(uint32)
	consumerID, _ := eventData["consumer_id"].(uint32)

	if tenantID == 0 || consumerID == 0 {
		s.log.Error("invalid tenant_id or consumer_id in event")
		return fmt.Errorf("invalid event data")
	}

	// 创建财务账户
	account := &consumerV1.FinanceAccount{
		TenantId:   &tenantID,
		ConsumerId: &consumerID,
	}

	_, err := s.financeAccountRepo.Create(ctx, account)
	if err != nil {
		s.log.Errorf("create finance account failed: %v", err)
		return err
	}

	s.log.Infof("finance account created for consumer: %d", consumerID)
	return nil
}

// handlePaymentSuccessEvent 处理支付成功事件
func (s *FinanceService) handlePaymentSuccessEvent(ctx context.Context, event interface{}) error {
	eventData, ok := event.(map[string]interface{})
	if !ok {
		s.log.Error("invalid payment success event data")
		return fmt.Errorf("invalid event data")
	}

	tenantID, _ := eventData["tenant_id"].(uint32)
	consumerID, _ := eventData["consumer_id"].(uint32)
	orderNo, _ := eventData["order_no"].(string)
	amountStr, _ := eventData["amount"].(string)

	if tenantID == 0 || consumerID == 0 || orderNo == "" || amountStr == "" {
		s.log.Error("invalid event data")
		return fmt.Errorf("invalid event data")
	}

	// 解析金额
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		s.log.Errorf("invalid amount: %s", amountStr)
		return err
	}

	// 自动充值
	req := &consumerV1.RechargeRequest{
		TenantId:       &tenantID,
		ConsumerId:     &consumerID,
		Amount:         &amountStr,
		RelatedOrderNo: &orderNo,
		Description:    stringPtr("支付成功自动充值"),
	}

	if err := s.rechargeInternal(ctx, req); err != nil {
		s.log.Errorf("auto recharge failed: %v", err)
		return err
	}

	s.log.Infof("auto recharge success: consumer=%d, amount=%s, order=%s", consumerID, amountStr, orderNo)
	return nil
}

// GetAccount 获取账户余额
func (s *FinanceService) GetAccount(ctx context.Context, req *consumerV1.GetAccountRequest) (*consumerV1.FinanceAccount, error) {
	// 1. 验证输入
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取租户ID和用户ID
	tenantID := middleware.GetTenantID(ctx)
	consumerID := middleware.GetUserID(ctx)

	// 如果请求中指定了用户ID，使用请求中的（管理员查询）
	if req.ConsumerId != nil {
		consumerID = *req.ConsumerId
	}

	// 3. 查询账户
	account, err := s.financeAccountRepo.GetByConsumerID(ctx, tenantID, consumerID)
	if err != nil {
		// 如果账户不存在，自动创建
		if err.Error() == "record not found" || err.Error() == "finance account not found" {
			account = &consumerV1.FinanceAccount{
				TenantId:   &tenantID,
				ConsumerId: &consumerID,
			}
			account, err = s.financeAccountRepo.Create(ctx, account)
			if err != nil {
				s.log.Errorf("create finance account failed: %v", err)
				return nil, err
			}
			s.log.Infof("finance account auto-created for consumer: %d", consumerID)
		} else {
			return nil, err
		}
	}

	return account, nil
}

// Recharge 充值
func (s *FinanceService) Recharge(ctx context.Context, req *consumerV1.RechargeRequest) (*emptypb.Empty, error) {
	// 1. 验证输入
	if req == nil || req.Amount == nil || *req.Amount == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取租户ID和用户ID
	tenantID := middleware.GetTenantID(ctx)
	consumerID := middleware.GetUserID(ctx)

	// 设置租户ID和用户ID
	req.TenantId = &tenantID
	req.ConsumerId = &consumerID

	// 3. 执行充值
	if err := s.rechargeInternal(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// rechargeInternal 充值内部实现
func (s *FinanceService) rechargeInternal(ctx context.Context, req *consumerV1.RechargeRequest) error {
	// 1. 验证金额
	amount, err := decimal.NewFromString(*req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return consumerV1.ErrorBadRequest("invalid amount")
	}

	// 2. 查询账户
	account, err := s.financeAccountRepo.GetByConsumerID(ctx, *req.TenantId, *req.ConsumerId)
	if err != nil {
		return err
	}

	// 3. 增加余额（原子操作）
	if err := s.financeAccountRepo.IncrementBalance(ctx, *account.Id, amount); err != nil {
		s.log.Errorf("increment balance failed: %v", err)
		return err
	}

	// 4. 重新查询账户获取最新余额
	account, err = s.financeAccountRepo.Get(ctx, *account.Id)
	if err != nil {
		s.log.Errorf("get account after recharge failed: %v", err)
		return err
	}

	// 5. 记录财务流水
	balanceBefore := subtractDecimal(*account.Balance, *req.Amount)
	transaction := &consumerV1.FinanceTransaction{
		TenantId:        req.TenantId,
		ConsumerId:      req.ConsumerId,
		TransactionNo:   stringPtr(s.generateTransactionNo("RCH")),
		TransactionType: consumerV1.FinanceTransaction_TRANSACTION_TYPE_RECHARGE.Enum(),
		Amount:          req.Amount,
		BalanceBefore:   &balanceBefore,
		BalanceAfter:    account.Balance,
		Description:     req.Description,
		RelatedOrderNo:  req.RelatedOrderNo,
		OperatorId:      req.OperatorId,
	}

	if _, err := s.financeTransactionRepo.Create(ctx, transaction); err != nil {
		s.log.Errorf("create transaction failed: %v", err)
		// 流水记录失败不影响充值结果
	}

	s.log.Infof("recharge success: consumer=%d, amount=%s, balance=%s", *req.ConsumerId, *req.Amount, *account.Balance)
	return nil
}

// Withdraw 申请提现
func (s *FinanceService) Withdraw(ctx context.Context, req *consumerV1.WithdrawRequest) (*consumerV1.WithdrawResponse, error) {
	// 1. 验证输入
	if req == nil || req.Amount == nil || *req.Amount == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 验证提现金额
	amount, err := decimal.NewFromString(*req.Amount)
	if err != nil {
		return nil, consumerV1.ErrorBadRequest("invalid amount")
	}

	minAmount := decimal.NewFromFloat(minWithdrawAmount)
	maxAmount := decimal.NewFromFloat(maxWithdrawAmount)

	if amount.LessThan(minAmount) {
		return nil, consumerV1.ErrorBadRequest(fmt.Sprintf("withdraw amount must be at least %.2f", minWithdrawAmount))
	}

	if amount.GreaterThan(maxAmount) {
		return nil, consumerV1.ErrorBadRequest(fmt.Sprintf("withdraw amount cannot exceed %.2f per day", maxWithdrawAmount))
	}

	// 3. 获取租户ID和用户ID
	tenantID := middleware.GetTenantID(ctx)
	consumerID := middleware.GetUserID(ctx)

	// 4. 查询账户
	account, err := s.financeAccountRepo.GetByConsumerID(ctx, tenantID, consumerID)
	if err != nil {
		return nil, err
	}

	// 5. 冻结余额
	if err := s.financeAccountRepo.FreezeBalance(ctx, *account.Id, amount); err != nil {
		s.log.Errorf("freeze balance failed: %v", err)
		return nil, err
	}

	// 6. 重新查询账户获取最新余额
	account, err = s.financeAccountRepo.Get(ctx, *account.Id)
	if err != nil {
		s.log.Errorf("get account after freeze failed: %v", err)
		return nil, err
	}

	// 7. 记录财务流水（提现申请）
	balanceBefore := addDecimal(*account.Balance, *req.Amount)
	transactionNo := s.generateTransactionNo("WDR")
	transaction := &consumerV1.FinanceTransaction{
		TenantId:        &tenantID,
		ConsumerId:      &consumerID,
		TransactionNo:   &transactionNo,
		TransactionType: consumerV1.FinanceTransaction_TRANSACTION_TYPE_WITHDRAW.Enum(),
		Amount:          req.Amount,
		BalanceBefore:   &balanceBefore,
		BalanceAfter:    account.Balance,
		Description:     stringPtr("提现申请（待审核）"),
	}

	if _, err := s.financeTransactionRepo.Create(ctx, transaction); err != nil {
		s.log.Errorf("create transaction failed: %v", err)
	}

	s.log.Infof("withdraw request created: consumer=%d, amount=%s, transaction=%s", consumerID, *req.Amount, transactionNo)

	return &consumerV1.WithdrawResponse{
		TransactionNo: transactionNo,
		Status:        "pending",
	}, nil
}

// ApproveWithdraw 审核提现
func (s *FinanceService) ApproveWithdraw(ctx context.Context, req *consumerV1.ApproveWithdrawRequest) (*emptypb.Empty, error) {
	// 1. 验证输入
	if req == nil || req.TransactionNo == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取租户ID
	tenantID := middleware.GetTenantID(ctx)
	operatorID := middleware.GetUserID(ctx)

	// 3. 查询提现流水
	// 注意：这里需要根据交易号查询流水，简化处理
	// 实际项目中应该有专门的查询方法

	// 4. 查询账户
	// 简化处理，假设从请求中获取用户ID
	if req.ConsumerId == nil {
		return nil, consumerV1.ErrorBadRequest("consumer_id is required")
	}

	account, err := s.financeAccountRepo.GetByConsumerID(ctx, tenantID, *req.ConsumerId)
	if err != nil {
		return nil, err
	}

	// 5. 解析提现金额（从交易号或请求中获取）
	if req.Amount == nil {
		return nil, consumerV1.ErrorBadRequest("amount is required")
	}

	amount, err := decimal.NewFromString(*req.Amount)
	if err != nil {
		return nil, consumerV1.ErrorBadRequest("invalid amount")
	}

	// 6. 根据审核结果处理
	if req.Approved {
		// 审核通过：从冻结余额中扣除（实际打款）
		currentFrozen, _ := decimal.NewFromString(*account.FrozenBalance)
		if currentFrozen.LessThan(amount) {
			return nil, fmt.Errorf("insufficient frozen balance")
		}

		newFrozen := currentFrozen.Sub(amount)

		// 更新冻结余额
		err = s.financeAccountRepo.UpdateBalance(ctx, *account.Id, *account.Balance, newFrozen.String(), 0)
		if err != nil {
			s.log.Errorf("deduct frozen balance failed: %v", err)
			return nil, err
		}

		// 记录流水（提现成功）
		account, _ = s.financeAccountRepo.Get(ctx, *account.Id)
		balanceBefore := addDecimal(*account.Balance, amount.String())
		transaction := &consumerV1.FinanceTransaction{
			TenantId:        &tenantID,
			ConsumerId:      req.ConsumerId,
			TransactionNo:   stringPtr(s.generateTransactionNo("WDS")),
			TransactionType: consumerV1.FinanceTransaction_TRANSACTION_TYPE_WITHDRAW.Enum(),
			Amount:          req.Amount,
			BalanceBefore:   &balanceBefore,
			BalanceAfter:    account.Balance,
			Description:     stringPtr("提现审核通过"),
			RelatedOrderNo:  &req.TransactionNo,
			OperatorId:      &operatorID,
		}

		if _, err := s.financeTransactionRepo.Create(ctx, transaction); err != nil {
			s.log.Errorf("create transaction failed: %v", err)
		}

		s.log.Infof("withdraw approved: consumer=%d, amount=%s, transaction=%s", *req.ConsumerId, *req.Amount, req.TransactionNo)
	} else {
		// 审核拒绝：解冻余额
		if err := s.financeAccountRepo.UnfreezeBalance(ctx, *account.Id, amount); err != nil {
			s.log.Errorf("unfreeze balance failed: %v", err)
			return nil, err
		}

		// 记录流水（提现拒绝）
		account, _ = s.financeAccountRepo.Get(ctx, *account.Id)
		transaction := &consumerV1.FinanceTransaction{
			TenantId:        &tenantID,
			ConsumerId:      req.ConsumerId,
			TransactionNo:   stringPtr(s.generateTransactionNo("WDR")),
			TransactionType: consumerV1.FinanceTransaction_TRANSACTION_TYPE_WITHDRAW.Enum(),
			Amount:          stringPtr("0"),
			BalanceBefore:   account.Balance,
			BalanceAfter:    account.Balance,
			Description:     stringPtr(fmt.Sprintf("提现审核拒绝: %s", req.RejectReason)),
			RelatedOrderNo:  &req.TransactionNo,
			OperatorId:      &operatorID,
		}

		if _, err := s.financeTransactionRepo.Create(ctx, transaction); err != nil {
			s.log.Errorf("create transaction failed: %v", err)
		}

		s.log.Infof("withdraw rejected: consumer=%d, transaction=%s, reason=%s", *req.ConsumerId, req.TransactionNo, req.RejectReason)
	}

	return &emptypb.Empty{}, nil
}

// ListTransactions 查询财务流水
func (s *FinanceService) ListTransactions(ctx context.Context, req *consumerV1.ListTransactionsRequest) (*consumerV1.ListTransactionsResponse, error) {
	// 1. 验证输入
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取租户ID
	tenantID := middleware.GetTenantID(ctx)
	req.TenantId = &tenantID

	// 3. 如果没有指定用户ID，使用当前用户ID
	if req.ConsumerId == nil {
		consumerID := middleware.GetUserID(ctx)
		req.ConsumerId = &consumerID
	}

	// 4. 查询流水
	resp, err := s.financeTransactionRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ExportTransactions 导出财务流水
func (s *FinanceService) ExportTransactions(ctx context.Context, req *consumerV1.ExportTransactionsRequest) (*consumerV1.ExportTransactionsResponse, error) {
	// 1. 验证输入
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取租户ID
	tenantID := middleware.GetTenantID(ctx)
	req.TenantId = &tenantID

	// 3. 如果没有指定用户ID，使用当前用户ID
	if req.ConsumerId == nil {
		consumerID := middleware.GetUserID(ctx)
		req.ConsumerId = &consumerID
	}

	// 4. 导出流水
	filepath, err := s.financeTransactionRepo.Export(ctx, req)
	if err != nil {
		return nil, err
	}

	s.log.Infof("transactions exported: consumer=%d, file=%s", *req.ConsumerId, filepath)

	return &consumerV1.ExportTransactionsResponse{
		FilePath: filepath,
	}, nil
}

// generateTransactionNo 生成交易流水号
// 格式: {prefix}{timestamp}{uuid前8位}
// prefix: RCH=充值, WDR=提现, CSM=消费, RFD=退款
func (s *FinanceService) generateTransactionNo(prefix string) string {
	timestamp := time.Now().Format("20060102150405")
	uuidStr := uuid.New().String()
	uuidShort := uuidStr[:8]
	return fmt.Sprintf("%s%s%s", prefix, timestamp, uuidShort)
}

// 辅助函数：字符串指针
func stringPtr(s string) *string {
	return &s
}

// 辅助函数：decimal 加法
func addDecimal(a, b string) string {
	aDecimal, _ := decimal.NewFromString(a)
	bDecimal, _ := decimal.NewFromString(b)
	return aDecimal.Add(bDecimal).String()
}

// 辅助函数：decimal 减法
func subtractDecimal(a, b string) string {
	aDecimal, _ := decimal.NewFromString(a)
	bDecimal, _ := decimal.NewFromString(b)
	return aDecimal.Sub(bDecimal).String()
}
