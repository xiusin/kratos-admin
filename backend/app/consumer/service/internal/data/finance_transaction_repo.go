package data

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/financetransaction"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// FinanceTransactionRepo 财务流水数据访问接口
type FinanceTransactionRepo interface {
	// Create 记录财务流水
	Create(ctx context.Context, data *consumerV1.FinanceTransaction) (*consumerV1.FinanceTransaction, error)

	// List 分页查询流水
	List(ctx context.Context, req *consumerV1.ListTransactionsRequest) (*consumerV1.ListTransactionsResponse, error)

	// Export 导出流水为CSV
	Export(ctx context.Context, req *consumerV1.ExportTransactionsRequest) (string, error)
}

type financeTransactionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                   *mapper.CopierMapper[consumerV1.FinanceTransaction, ent.FinanceTransaction]
	transactionTypeConverter *mapper.EnumTypeConverter[consumerV1.FinanceTransaction_TransactionType, financetransaction.TransactionType]

	repository *entCrud.Repository[
		ent.FinanceTransactionQuery, ent.FinanceTransactionSelect,
		ent.FinanceTransactionCreate, ent.FinanceTransactionCreateBulk,
		ent.FinanceTransactionUpdate, ent.FinanceTransactionUpdateOne,
		ent.FinanceTransactionDelete,
		predicate.FinanceTransaction,
		consumerV1.FinanceTransaction, ent.FinanceTransaction,
	]
}

// NewFinanceTransactionRepo 创建财务流水数据访问实例
func NewFinanceTransactionRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) FinanceTransactionRepo {
	repo := &financeTransactionRepo{
		log:                      ctx.NewLoggerHelper("consumer/repo/finance-transaction"),
		entClient:                entClient,
		mapper:                   mapper.NewCopierMapper[consumerV1.FinanceTransaction, ent.FinanceTransaction](),
		transactionTypeConverter: mapper.NewEnumTypeConverter[consumerV1.FinanceTransaction_TransactionType, financetransaction.TransactionType](consumerV1.FinanceTransaction_TransactionType_name, consumerV1.FinanceTransaction_TransactionType_value),
	}

	repo.init()

	return repo
}

func (r *financeTransactionRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.FinanceTransactionQuery, ent.FinanceTransactionSelect,
		ent.FinanceTransactionCreate, ent.FinanceTransactionCreateBulk,
		ent.FinanceTransactionUpdate, ent.FinanceTransactionUpdateOne,
		ent.FinanceTransactionDelete,
		predicate.FinanceTransaction,
		consumerV1.FinanceTransaction, ent.FinanceTransaction,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.transactionTypeConverter.NewConverterPair())
}

// Create 记录财务流水
func (r *financeTransactionRepo) Create(ctx context.Context, data *consumerV1.FinanceTransaction) (*consumerV1.FinanceTransaction, error) {
	if data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	builder := r.entClient.Client().FinanceTransaction.Create().
		SetCreatedAt(time.Now())

	// 设置字段
	if data.TenantId != nil {
		builder.SetTenantID(*data.TenantId)
	}
	if data.ConsumerId != nil {
		builder.SetConsumerID(*data.ConsumerId)
	}
	if data.TransactionNo != nil {
		builder.SetTransactionNo(*data.TransactionNo)
	}
	if data.TransactionType != nil {
		builder.SetTransactionType(r.transactionTypeConverter.ToEntity(data.TransactionType))
	}
	if data.Amount != nil {
		builder.SetAmount(*data.Amount)
	}
	if data.BalanceBefore != nil {
		builder.SetBalanceBefore(*data.BalanceBefore)
	}
	if data.BalanceAfter != nil {
		builder.SetBalanceAfter(*data.BalanceAfter)
	}
	if data.Description != nil {
		builder.SetDescription(*data.Description)
	}
	if data.RelatedOrderNo != nil {
		builder.SetRelatedOrderNo(*data.RelatedOrderNo)
	}
	if data.OperatorId != nil {
		builder.SetOperatorID(*data.OperatorId)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert finance transaction failed: %s", err.Error())
		return nil, fmt.Errorf("insert finance transaction failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// List 分页查询流水
func (r *financeTransactionRepo) List(ctx context.Context, req *consumerV1.ListTransactionsRequest) (*consumerV1.ListTransactionsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	builder := r.entClient.Client().FinanceTransaction.Query()

	// 应用筛选条件
	predicates := []predicate.FinanceTransaction{}

	// 租户ID筛选
	if req.TenantId != nil {
		predicates = append(predicates, financetransaction.TenantIDEQ(*req.TenantId))
	}

	// 用户ID筛选
	if req.ConsumerId != nil {
		predicates = append(predicates, financetransaction.ConsumerIDEQ(*req.ConsumerId))
	}

	// 交易类型筛选
	if req.TransactionType != nil {
		predicates = append(predicates, financetransaction.TransactionTypeEQ(r.transactionTypeConverter.ToEntity(req.TransactionType)))
	}

	// 时间范围筛选
	if req.StartTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtGTE(req.StartTime.AsTime()))
	}
	if req.EndTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtLTE(req.EndTime.AsTime()))
	}

	if len(predicates) > 0 {
		builder = builder.Where(financetransaction.And(predicates...))
	}

	// 按创建时间倒序排列
	builder = builder.Order(ent.Desc(financetransaction.FieldCreatedAt))

	// 分页查询
	pagingReq := &paginationV1.PagingRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), pagingReq)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListTransactionsResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListTransactionsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// Export 导出流水为CSV
func (r *financeTransactionRepo) Export(ctx context.Context, req *consumerV1.ExportTransactionsRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("invalid parameter")
	}

	// 查询所有符合条件的流水（不分页）
	builder := r.entClient.Client().FinanceTransaction.Query()

	// 应用筛选条件（与List方法相同）
	predicates := []predicate.FinanceTransaction{}

	if req.TenantId != nil {
		predicates = append(predicates, financetransaction.TenantIDEQ(*req.TenantId))
	}

	if req.ConsumerId != nil {
		predicates = append(predicates, financetransaction.ConsumerIDEQ(*req.ConsumerId))
	}

	if req.TransactionType != nil {
		predicates = append(predicates, financetransaction.TransactionTypeEQ(r.transactionTypeConverter.ToEntity(req.TransactionType)))
	}

	if req.StartTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtGTE(req.StartTime.AsTime()))
	}
	if req.EndTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtLTE(req.EndTime.AsTime()))
	}

	if len(predicates) > 0 {
		builder = builder.Where(financetransaction.And(predicates...))
	}

	// 按创建时间倒序排列
	builder = builder.Order(ent.Desc(financetransaction.FieldCreatedAt))

	// 查询所有记录
	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query transactions for export failed: %s", err.Error())
		return "", fmt.Errorf("query transactions failed")
	}

	// 创建临时CSV文件
	tmpDir := os.TempDir()
	filename := fmt.Sprintf("finance_transactions_%d.csv", time.Now().Unix())
	filepath := filepath.Join(tmpDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		r.log.Errorf("create csv file failed: %s", err.Error())
		return "", fmt.Errorf("create csv file failed")
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头部
	headers := []string{
		"流水号", "交易类型", "金额", "交易前余额", "交易后余额",
		"描述", "关联订单号", "操作人ID", "创建时间",
	}
	if err := writer.Write(headers); err != nil {
		r.log.Errorf("write csv header failed: %s", err.Error())
		return "", fmt.Errorf("write csv failed")
	}

	// 写入数据行
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		row := []string{
			getStringValue(dto.TransactionNo),
			dto.TransactionType.String(),
			getStringValue(dto.Amount),
			getStringValue(dto.BalanceBefore),
			getStringValue(dto.BalanceAfter),
			getStringValue(dto.Description),
			getStringValue(dto.RelatedOrderNo),
			fmt.Sprintf("%d", getUint32Value(dto.OperatorId)),
			dto.CreatedAt.AsTime().Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			r.log.Errorf("write csv row failed: %s", err.Error())
			return "", fmt.Errorf("write csv failed")
		}
	}

	r.log.Infof("exported %d transactions to %s", len(entities), filepath)

	return filepath, nil
}

// 辅助函数：获取字符串指针的值
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// 辅助函数：获取uint32指针的值
func getUint32Value(u *uint32) uint32 {
	if u == nil {
		return 0
	}
	return *u
}
