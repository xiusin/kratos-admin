package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
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
	GetByConsumerID(ctx context.Context, consumerID uint32) (*consumerV1.FinanceAccount, error)

	// UpdateBalance 更新余额(使用乐观锁)
	UpdateBalance(ctx context.Context, id uint32, balance, frozenBalance string) error
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
		log:       ctx.NewLoggerHelper("finance-account/repo/consumer-service"),
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
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().FinanceAccount.Create().
		SetNillableTenantID(data.TenantId).
		SetConsumerID(data.GetConsumerId()).
		SetBalance(data.GetBalance()).
		SetFrozenBalance(data.GetFrozenBalance())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert finance account failed: %s", err.Error())
		return nil, errors.InternalServer("INSERT_FAILED", "insert finance account failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询账户
func (r *financeAccountRepo) Get(ctx context.Context, id uint32) (*consumerV1.FinanceAccount, error) {
	builder := r.entClient.Client().FinanceAccount.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(financeaccount.FieldID, id))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetByConsumerID 按用户ID查询
func (r *financeAccountRepo) GetByConsumerID(ctx context.Context, consumerID uint32) (*consumerV1.FinanceAccount, error) {
	builder := r.entClient.Client().FinanceAccount.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(financeaccount.FieldConsumerID, consumerID))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// UpdateBalance 更新余额(使用乐观锁)
func (r *financeAccountRepo) UpdateBalance(ctx context.Context, id uint32, balance, frozenBalance string) error {
	// 使用事务和乐观锁更新余额
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("begin transaction failed: %s", err.Error())
		return errors.InternalServer("TRANSACTION_FAILED", "begin transaction failed")
	}

	// 先查询当前记录(加锁)
	account, err := tx.FinanceAccount.Query().
		Where(financeaccount.IDEQ(id)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return errors.NotFound("ACCOUNT_NOT_FOUND", "finance account not found")
		}
		r.log.Errorf("query finance account failed: %s", err.Error())
		return errors.InternalServer("QUERY_FAILED", "query finance account failed")
	}

	// 更新余额
	if err := tx.FinanceAccount.UpdateOne(account).
		SetBalance(balance).
		SetFrozenBalance(frozenBalance).
		SetUpdatedAt(time.Now()).
		Exec(ctx); err != nil {
		_ = tx.Rollback()
		r.log.Errorf("update balance failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "update balance failed")
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		r.log.Errorf("commit transaction failed: %s", err.Error())
		return errors.InternalServer("COMMIT_FAILED", "commit transaction failed")
	}

	return nil
}
