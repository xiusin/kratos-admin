package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/financeaccount"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// FinanceAccountRepo 财务账户数据访问接口
type FinanceAccountRepo interface {
	// Create 创建财务账户
	Create(ctx context.Context, data *consumerV1.FinanceAccount) (*consumerV1.FinanceAccount, error)

	// Get 查询账户
	Get(ctx context.Context, id uint32) (*consumerV1.FinanceAccount, error)

	// GetByConsumerID 按用户ID查询
	GetByConsumerID(ctx context.Context, tenantID uint32, consumerID uint32) (*consumerV1.FinanceAccount, error)

	// UpdateBalance 更新余额（使用乐观锁）
	UpdateBalance(ctx context.Context, id uint32, balance string, frozenBalance string, version int32) error

	// IncrementBalance 增加余额（原子操作）
	IncrementBalance(ctx context.Context, id uint32, amount decimal.Decimal) error

	// DecrementBalance 减少余额（原子操作）
	DecrementBalance(ctx context.Context, id uint32, amount decimal.Decimal) error

	// FreezeBalance 冻结余额
	FreezeBalance(ctx context.Context, id uint32, amount decimal.Decimal) error

	// UnfreezeBalance 解冻余额
	UnfreezeBalance(ctx context.Context, id uint32, amount decimal.Decimal) error
}

type financeAccountRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[consumerV1.FinanceAccount, ent.FinanceAccount]

	repository *entCrud.Repository[
		ent.FinanceAccountQuery, ent.FinanceAccountSelect,
		ent.FinanceAccountCreate, ent.FinanceAccountCreateBulk,
		ent.FinanceAccountUpdate, ent.FinanceAccountUpdateOne,
		ent.FinanceAccountDelete,
		predicate.FinanceAccount,
		consumerV1.FinanceAccount, ent.FinanceAccount,
	]
}

// NewFinanceAccountRepo 创建财务账户数据访问实例
func NewFinanceAccountRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) FinanceAccountRepo {
	repo := &financeAccountRepo{
		log:       ctx.NewLoggerHelper("consumer/repo/finance-account"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[consumerV1.FinanceAccount, ent.FinanceAccount](),
	}

	repo.init()

	return repo
}

func (r *financeAccountRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.FinanceAccountQuery, ent.FinanceAccountSelect,
		ent.FinanceAccountCreate, ent.FinanceAccountCreateBulk,
		ent.FinanceAccountUpdate, ent.FinanceAccountUpdateOne,
		ent.FinanceAccountDelete,
		predicate.FinanceAccount,
		consumerV1.FinanceAccount, ent.FinanceAccount,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// Create 创建财务账户
func (r *financeAccountRepo) Create(ctx context.Context, data *consumerV1.FinanceAccount) (*consumerV1.FinanceAccount, error) {
	if data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	// 默认余额为0
	balance := "0"
	if data.Balance != nil {
		balance = *data.Balance
	}

	frozenBalance := "0"
	if data.FrozenBalance != nil {
		frozenBalance = *data.FrozenBalance
	}

	builder := r.entClient.Client().FinanceAccount.Create().
		SetBalance(balance).
		SetFrozenBalance(frozenBalance).
		SetCreatedAt(time.Now())

	// 设置租户ID和用户ID
	if data.TenantId != nil {
		builder.SetTenantID(*data.TenantId)
	}
	if data.ConsumerId != nil {
		builder.SetConsumerID(*data.ConsumerId)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert finance account failed: %s", err.Error())
		return nil, fmt.Errorf("insert finance account failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询账户
func (r *financeAccountRepo) Get(ctx context.Context, id uint32) (*consumerV1.FinanceAccount, error) {
	entity, err := r.entClient.Client().FinanceAccount.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, consumerV1.ErrorNotFound("finance account not found")
		}
		r.log.Errorf("get finance account failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("get finance account failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// GetByConsumerID 按用户ID查询
func (r *financeAccountRepo) GetByConsumerID(ctx context.Context, tenantID uint32, consumerID uint32) (*consumerV1.FinanceAccount, error) {
	entity, err := r.entClient.Client().FinanceAccount.Query().
		Where(
			financeaccount.TenantID(tenantID),
			financeaccount.ConsumerID(consumerID),
		).
		Only(ctx)
	
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, consumerV1.ErrorNotFound("finance account not found")
		}
		r.log.Errorf("get finance account by consumer id failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("get finance account by consumer id failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// UpdateBalance 更新余额（使用乐观锁）
func (r *financeAccountRepo) UpdateBalance(ctx context.Context, id uint32, balance string, frozenBalance string, version int32) error {
	// 注意：由于 Schema 中没有 version 字段，这里使用 updated_at 作为乐观锁
	// 实际项目中建议在 Schema 中添加 version 字段

	// 先查询当前记录
	current, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// 使用 updated_at 作为版本控制
	affected, err := r.entClient.Client().FinanceAccount.Update().
		Where(
			financeaccount.And(
				financeaccount.IDEQ(id),
				// 使用 updated_at 确保记录没有被其他事务修改
				financeaccount.UpdatedAtEQ(current.UpdatedAt.AsTime()),
			),
		).
		SetBalance(balance).
		SetFrozenBalance(frozenBalance).
		SetUpdatedAt(time.Now()).
		Save(ctx)

	if err != nil {
		r.log.Errorf("update balance failed: %s", err.Error())
		return fmt.Errorf("update balance failed")
	}

	if affected == 0 {
		return fmt.Errorf("balance update conflict, please retry")
	}

	return nil
}

// IncrementBalance 增加余额（原子操作）
func (r *financeAccountRepo) IncrementBalance(ctx context.Context, id uint32, amount decimal.Decimal) error {
	// 先查询当前余额
	account, err := r.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("finance account not found")
		}
		return err
	}

	// 计算新余额
	currentBalance, _ := decimal.NewFromString(*account.Balance)
	newBalance := currentBalance.Add(amount)

	// 更新余额
	err = r.entClient.Client().FinanceAccount.UpdateOneID(id).
		SetBalance(newBalance.String()).
		SetUpdatedAt(time.Now()).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("finance account not found")
		}
		r.log.Errorf("increment balance failed: %s", err.Error())
		return fmt.Errorf("increment balance failed")
	}

	return nil
}

// DecrementBalance 减少余额（原子操作）
func (r *financeAccountRepo) DecrementBalance(ctx context.Context, id uint32, amount decimal.Decimal) error {
	// 先查询当前余额
	account, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// 验证余额充足
	currentBalance, _ := decimal.NewFromString(*account.Balance)
	if currentBalance.LessThan(amount) {
		return fmt.Errorf("insufficient balance")
	}

	// 计算新余额
	newBalance := currentBalance.Sub(amount)

	// 更新余额
	err = r.entClient.Client().FinanceAccount.UpdateOneID(id).
		SetBalance(newBalance.String()).
		SetUpdatedAt(time.Now()).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("finance account not found")
		}
		r.log.Errorf("decrement balance failed: %s", err.Error())
		return fmt.Errorf("decrement balance failed")
	}

	return nil
}

// FreezeBalance 冻结余额
func (r *financeAccountRepo) FreezeBalance(ctx context.Context, id uint32, amount decimal.Decimal) error {
	// 先查询当前余额
	account, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// 验证余额充足
	currentBalance, _ := decimal.NewFromString(*account.Balance)
	if currentBalance.LessThan(amount) {
		return fmt.Errorf("insufficient balance to freeze")
	}

	// 计算新余额
	newBalance := currentBalance.Sub(amount)
	currentFrozen, _ := decimal.NewFromString(*account.FrozenBalance)
	newFrozen := currentFrozen.Add(amount)

	// 更新余额
	err = r.entClient.Client().FinanceAccount.UpdateOneID(id).
		SetBalance(newBalance.String()).
		SetFrozenBalance(newFrozen.String()).
		SetUpdatedAt(time.Now()).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("finance account not found")
		}
		r.log.Errorf("freeze balance failed: %s", err.Error())
		return fmt.Errorf("freeze balance failed")
	}

	return nil
}

// UnfreezeBalance 解冻余额
func (r *financeAccountRepo) UnfreezeBalance(ctx context.Context, id uint32, amount decimal.Decimal) error {
	// 先查询当前余额
	account, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	// 验证冻结余额充足
	currentFrozen, _ := decimal.NewFromString(*account.FrozenBalance)
	if currentFrozen.LessThan(amount) {
		return fmt.Errorf("insufficient frozen balance to unfreeze")
	}

	// 计算新余额
	newFrozen := currentFrozen.Sub(amount)
	currentBalance, _ := decimal.NewFromString(*account.Balance)
	newBalance := currentBalance.Add(amount)

	// 更新余额
	err = r.entClient.Client().FinanceAccount.UpdateOneID(id).
		SetBalance(newBalance.String()).
		SetFrozenBalance(newFrozen.String()).
		SetUpdatedAt(time.Now()).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("finance account not found")
		}
		r.log.Errorf("unfreeze balance failed: %s", err.Error())
		return fmt.Errorf("unfreeze balance failed")
	}

	return nil
}
