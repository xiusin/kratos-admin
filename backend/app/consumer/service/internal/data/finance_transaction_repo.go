package data

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
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
		log:                      ctx.NewLoggerHelper("finance-transaction/repo/consumer-service"),
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
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().FinanceTransaction.Create().
		SetNillableTenantID(data.TenantId).
		SetConsumerID(data.GetConsumerId()).
		SetTransactionNo(data.GetTransactionNo()).
		SetAmount(data.GetAmount()).
		SetBalanceBefore(data.GetBalanceBefore()).
		SetBalanceAfter(data.GetBalanceAfter())

	// 设置 transaction_type
	if transactionType := r.transactionTypeConverter.ToEntity(data.TransactionType); transactionType != nil {
		builder.SetTransactionType(*transactionType)
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
		return nil, errors.InternalServer("INSERT_FAILED", "insert finance transaction failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// List 分页查询流水
func (r *financeTransactionRepo) List(ctx context.Context, req *consumerV1.ListTransactionsRequest) (*consumerV1.ListTransactionsResponse, error) {
	if req == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().FinanceTransaction.Query()

	// 构建查询条件
	var predicates []predicate.FinanceTransaction

	// 按用户ID筛选
	if req.ConsumerId != nil {
		predicates = append(predicates, financetransaction.ConsumerIDEQ(*req.ConsumerId))
	}

	// 按交易类型筛选
	if req.TransactionType != nil {
		if transactionType := r.transactionTypeConverter.ToEntity(req.TransactionType); transactionType != nil {
			predicates = append(predicates, financetransaction.TransactionTypeEQ(*transactionType))
		}
	}

	// 按时间范围筛选
	if req.StartTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtGTE(req.StartTime.AsTime()))
	}
	if req.EndTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtLTE(req.EndTime.AsTime()))
	}

	if len(predicates) > 0 {
		builder.Where(predicates...)
	}

	// 按创建时间倒序排序
	builder.Order(ent.Desc(financetransaction.FieldCreatedAt))

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
		return "", errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().FinanceTransaction.Query()

	// 构建查询条件(与List相同)
	var predicates []predicate.FinanceTransaction

	if req.ConsumerId != nil {
		predicates = append(predicates, financetransaction.ConsumerIDEQ(*req.ConsumerId))
	}

	if req.TransactionType != nil {
		if transactionType := r.transactionTypeConverter.ToEntity(req.TransactionType); transactionType != nil {
			predicates = append(predicates, financetransaction.TransactionTypeEQ(*transactionType))
		}
	}

	if req.StartTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtGTE(req.StartTime.AsTime()))
	}
	if req.EndTime != nil {
		predicates = append(predicates, financetransaction.CreatedAtLTE(req.EndTime.AsTime()))
	}

	if len(predicates) > 0 {
		builder.Where(predicates...)
	}

	// 按创建时间倒序排序
	builder.Order(ent.Desc(financetransaction.FieldCreatedAt))

	// 查询所有记录(限制最多10000条)
	entities, err := builder.Limit(10000).All(ctx)
	if err != nil {
		r.log.Errorf("query finance transactions failed: %s", err.Error())
		return "", errors.InternalServer("QUERY_FAILED", "query finance transactions failed")
	}

	// 生成CSV内容
	var csvBuilder strings.Builder
	writer := csv.NewWriter(&csvBuilder)

	// 写入表头
	header := []string{
		"流水号", "交易类型", "交易金额", "交易前余额", "交易后余额",
		"交易描述", "关联订单号", "创建时间",
	}
	if err := writer.Write(header); err != nil {
		r.log.Errorf("write csv header failed: %s", err.Error())
		return "", errors.InternalServer("EXPORT_FAILED", "export failed")
	}

	// 写入数据行
	for _, entity := range entities {
		row := []string{
			entity.TransactionNo,
			string(entity.TransactionType),
			entity.Amount,
			entity.BalanceBefore,
			entity.BalanceAfter,
			func() string {
				if entity.Description != nil {
					return *entity.Description
				}
				return ""
			}(),
			func() string {
				if entity.RelatedOrderNo != nil {
					return *entity.RelatedOrderNo
				}
				return ""
			}(),
			entity.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			r.log.Errorf("write csv row failed: %s", err.Error())
			return "", errors.InternalServer("EXPORT_FAILED", "export failed")
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		r.log.Errorf("flush csv writer failed: %s", err.Error())
		return "", errors.InternalServer("EXPORT_FAILED", "export failed")
	}

	// 返回CSV内容(实际应用中应该上传到OSS并返回URL)
	// csvContent := csvBuilder.String()
	// TODO: 上传到OSS并返回URL
	// 这里暂时返回一个模拟的URL
	fileURL := fmt.Sprintf("/exports/finance_transactions_%s.csv", time.Now().Format("20060102150405"))

	r.log.Infof("exported %d finance transactions to %s", len(entities), fileURL)

	return fileURL, nil
}
