package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/loginlog"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// LoginLogRepo 登录日志数据访问接口
type LoginLogRepo interface {
	// Create 记录登录日志
	Create(ctx context.Context, data *consumerV1.LoginLog) error

	// List 分页查询登录日志
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLoginLogsResponse, error)
}

type loginLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper             *mapper.CopierMapper[consumerV1.LoginLog, ent.LoginLog]
	loginTypeConverter *mapper.EnumTypeConverter[consumerV1.LoginLog_LoginType, loginlog.LoginType]

	repository *entCrud.Repository[
		ent.LoginLogQuery, ent.LoginLogSelect,
		ent.LoginLogCreate, ent.LoginLogCreateBulk,
		ent.LoginLogUpdate, ent.LoginLogUpdateOne,
		ent.LoginLogDelete,
		predicate.LoginLog,
		consumerV1.LoginLog, ent.LoginLog,
	]
}

// NewLoginLogRepo 创建登录日志数据访问实例
func NewLoginLogRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) LoginLogRepo {
	repo := &loginLogRepo{
		log:                ctx.NewLoggerHelper("login-log/repo/consumer-service"),
		entClient:          entClient,
		mapper:             mapper.NewCopierMapper[consumerV1.LoginLog, ent.LoginLog](),
		loginTypeConverter: mapper.NewEnumTypeConverter[consumerV1.LoginLog_LoginType, loginlog.LoginType](consumerV1.LoginLog_LoginType_name, consumerV1.LoginLog_LoginType_value),
	}

	repo.init()

	return repo
}

func (r *loginLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LoginLogQuery, ent.LoginLogSelect,
		ent.LoginLogCreate, ent.LoginLogCreateBulk,
		ent.LoginLogUpdate, ent.LoginLogUpdateOne,
		ent.LoginLogDelete,
		predicate.LoginLog,
		consumerV1.LoginLog, ent.LoginLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.loginTypeConverter.NewConverterPair())
}

// Create 记录登录日志
func (r *loginLogRepo) Create(ctx context.Context, data *consumerV1.LoginLog) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginLog.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableSuccess(data.Success).
		SetNillableFailReason(data.FailReason).
		SetNillableUserAgent(data.UserAgent).
		SetNillableDeviceType(data.DeviceType)

	// 设置必填字段 consumer_id
	if data.ConsumerId != nil {
		builder.SetConsumerID(*data.ConsumerId)
	}
	
	// 设置必填字段 phone
	if data.Phone != nil {
		builder.SetPhone(*data.Phone)
	}
	
	// 设置必填字段 ip_address
	if data.IpAddress != nil {
		builder.SetIPAddress(*data.IpAddress)
	}
	
	// 设置必填字段 login_type
	if data.LoginType != nil {
		loginType := r.loginTypeConverter.ToEntity(data.LoginType)
		if loginType != nil {
			builder.SetLoginType(*loginType)
		}
	}

	// 设置登录时间
	if data.LoginAt != nil {
		builder.SetLoginAt(data.LoginAt.AsTime())
	} else {
		builder.SetLoginAt(time.Now())
	}

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert login log failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("insert login log failed")
	}

	return nil
}

// List 分页查询登录日志
func (r *loginLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLoginLogsResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginLog.Query().
		Order(ent.Desc(loginlog.FieldLoginAt))

	// 计算总数
	count, err := builder.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count login logs failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("count login logs failed")
	}

	// 分页查询
	if req.GetPage() > 0 && req.GetPageSize() > 0 {
		offset := int(req.GetPage()-1) * int(req.GetPageSize())
		builder.Offset(offset).Limit(int(req.GetPageSize()))
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("list login logs failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("list login logs failed")
	}

	items := make([]*consumerV1.LoginLog, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		
		// 敏感数据脱敏
		// 手机号脱敏：保留前3位和后4位
		if dto.Phone != nil && len(*dto.Phone) >= 11 {
			phone := *dto.Phone
			masked := phone[:3] + "****" + phone[len(phone)-4:]
			dto.Phone = &masked
		}

		// IP地址脱敏：只保留前两段
		if dto.IpAddress != nil && len(*dto.IpAddress) > 0 {
			ip := *dto.IpAddress
			masked := maskIPAddress(ip)
			dto.IpAddress = &masked
		}
		
		items = append(items, dto)
	}

	return &consumerV1.ListLoginLogsResponse{
		Total: uint64(count),
		Items: items,
	}, nil
}

// maskIPAddress IP地址脱敏
func maskIPAddress(ip string) string {
	// 简单实现：保留前两段
	// 例如：192.168.1.100 -> 192.168.*.*
	parts := []rune(ip)
	dotCount := 0
	for i, ch := range parts {
		if ch == '.' {
			dotCount++
			if dotCount >= 2 {
				// 从第二个点之后开始替换
				for j := i + 1; j < len(parts); j++ {
					if parts[j] != '.' {
						parts[j] = '*'
					}
				}
				break
			}
		}
	}
	return string(parts)
}
