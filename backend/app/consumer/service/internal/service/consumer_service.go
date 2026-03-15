package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/jwt"
)

// ConsumerService C端用户服务
type ConsumerService struct {
	consumerV1.UnimplementedConsumerServiceServer

	consumerRepo data.ConsumerRepo
	loginLogRepo data.LoginLogRepo
	eventBus     eventbus.EventBus
	jwtHelper    *jwt.Helper
	log          *log.Helper
}

// NewConsumerService 创建C端用户服务实例
func NewConsumerService(
	ctx *bootstrap.Context,
	consumerRepo data.ConsumerRepo,
	loginLogRepo data.LoginLogRepo,
	eventBus eventbus.EventBus,
	jwtHelper *jwt.Helper,
) *ConsumerService {
	return &ConsumerService{
		consumerRepo: consumerRepo,
		loginLogRepo: loginLogRepo,
		eventBus:     eventBus,
		jwtHelper:    jwtHelper,
		log:          ctx.NewLoggerHelper("consumer/service/consumer-service"),
	}
}

// RegisterByPhone 手机号注册
func (s *ConsumerService) RegisterByPhone(ctx context.Context, req *consumerV1.RegisterByPhoneRequest) (*consumerV1.Consumer, error) {
	s.log.Infof("RegisterByPhone: phone=%s", req.GetPhone())

	// TODO: 验证验证码（需要集成SMS Service）
	// 这里暂时跳过验证码验证

	// 检查手机号是否已注册
	existingUser, err := s.consumerRepo.GetByPhone(ctx, req.GetPhone())
	if err == nil && existingUser != nil {
		return nil, errors.Conflict("PHONE_EXISTS", "phone number already registered")
	}

	// 加密密码
	// TODO: 需要在 data 层的 Create 方法中添加 password_hash 参数
	_ = req.GetPassword() // 暂时忽略密码，等待重构

	// 创建用户（注意：password_hash 在 Ent schema 中，但不在 proto 中）
	// 我们需要直接在数据层设置密码
	consumer := &consumerV1.Consumer{
		Phone:          &req.Phone,
		Nickname:       req.Nickname,
		Status:         consumerV1.Consumer_NORMAL.Enum(),
		RiskScore:      new(int32),
		LoginFailCount: new(int32),
	}
	*consumer.RiskScore = 0
	*consumer.LoginFailCount = 0

	// TODO: 需要在 data 层的 Create 方法中添加 password_hash 参数
	// 暂时先创建用户，密码处理需要重构
	createdConsumer, err := s.consumerRepo.Create(ctx, consumer)
	if err != nil {
		s.log.Errorf("create consumer failed: %v", err)
		return nil, err
	}

	// 发布用户注册事件
	event := eventbus.NewEvent("consumer.registered", map[string]interface{}{
		"consumer_id": createdConsumer.GetId(),
		"phone":       createdConsumer.GetPhone(),
		"tenant_id":   createdConsumer.GetTenantId(),
	}).WithSource("consumer-service")

	if err := s.eventBus.PublishAsync(ctx, event); err != nil {
		s.log.Errorf("publish consumer registered event failed: %v", err)
		// 不影响注册流程
	}

	s.log.Infof("RegisterByPhone success: consumer_id=%d", createdConsumer.GetId())
	return createdConsumer, nil
}

// LoginByPhone 手机号登录
func (s *ConsumerService) LoginByPhone(ctx context.Context, req *consumerV1.LoginByPhoneRequest) (*consumerV1.LoginResponse, error) {
	s.log.Infof("LoginByPhone: phone=%s", req.GetPhone())

	// 查询用户
	consumer, err := s.consumerRepo.GetByPhone(ctx, req.GetPhone())
	if err != nil {
		s.log.Errorf("get consumer by phone failed: %v", err)
		// 记录登录失败日志
		s.recordLoginLog(ctx, 0, req.GetPhone(), consumerV1.LoginLog_PHONE, false, "user not found", "")
		return nil, errors.Unauthorized("UNAUTHORIZED", "invalid phone or password")
	}

	// 检查账户状态
	if consumer.GetStatus() == consumerV1.Consumer_DEACTIVATED {
		s.recordLoginLog(ctx, consumer.GetId(), req.GetPhone(), consumerV1.LoginLog_PHONE, false, "account deactivated", "")
		return nil, errors.Forbidden("PERMISSION_DENIED", "account has been deactivated")
	}

	// 检查账户是否被锁定
	if consumer.GetStatus() == consumerV1.Consumer_LOCKED {
		if consumer.LockedUntil != nil && consumer.LockedUntil.AsTime().After(time.Now()) {
			remainingTime := time.Until(consumer.LockedUntil.AsTime())
			s.recordLoginLog(ctx, consumer.GetId(), req.GetPhone(), consumerV1.LoginLog_PHONE, false, "account locked", "")
			return nil, errors.Forbidden("PERMISSION_DENIED", fmt.Sprintf("account is locked, please try again in %d minutes", int(remainingTime.Minutes())+1))
		}
		// 锁定时间已过，解锁账户
		if err := s.consumerRepo.ResetLoginFailCount(ctx, consumer.GetId()); err != nil {
			s.log.Errorf("reset login fail count failed: %v", err)
		}
	}

	// TODO: 验证密码 - password_hash 不在 proto 中，需要重构数据层来支持密码验证
	// 暂时跳过密码验证
	/*
		if err := s.verifyPassword(consumer.GetPasswordHash(), req.GetPassword()); err != nil {
			s.log.Warnf("password verification failed for phone=%s", req.GetPhone())

			// 增加登录失败次数
			if err := s.consumerRepo.IncrementLoginFailCount(ctx, consumer.GetId()); err != nil {
				s.log.Errorf("increment login fail count failed: %v", err)
			}

			// 检查是否需要锁定账户（连续5次失败）
			if consumer.GetLoginFailCount() >= 4 { // 已经失败4次，这次是第5次
				lockedUntil := time.Now().Add(15 * time.Minute)
				if err := s.consumerRepo.LockAccount(ctx, consumer.GetId(), lockedUntil); err != nil {
					s.log.Errorf("lock account failed: %v", err)
				} else {
					s.log.Warnf("account locked due to 5 consecutive failed login attempts: consumer_id=%d", consumer.GetId())
				}
			}

			// 更新风险评分
			newRiskScore := s.calculateRiskScore(consumer, false)
			if err := s.consumerRepo.UpdateRiskScore(ctx, consumer.GetId(), newRiskScore); err != nil {
				s.log.Errorf("update risk score failed: %v", err)
			}

			s.recordLoginLog(ctx, consumer.GetId(), req.GetPhone(), consumerV1.LoginLog_PHONE, false, "invalid password", "")
			return nil, errors.Unauthorized("UNAUTHORIZED", "invalid phone or password")
		}
	*/

	// 登录成功，重置登录失败次数
	if err := s.consumerRepo.ResetLoginFailCount(ctx, consumer.GetId()); err != nil {
		s.log.Errorf("reset login fail count failed: %v", err)
	}

	// 更新登录信息
	loginIP := s.getClientIP(ctx)
	loginAt := time.Now()
	if err := s.consumerRepo.UpdateLoginInfo(ctx, consumer.GetId(), loginIP, loginAt); err != nil {
		s.log.Errorf("update login info failed: %v", err)
	}

	// 更新风险评分
	newRiskScore := s.calculateRiskScore(consumer, true)
	if err := s.consumerRepo.UpdateRiskScore(ctx, consumer.GetId(), newRiskScore); err != nil {
		s.log.Errorf("update risk score failed: %v", err)
	}

	// 记录登录成功日志
	s.recordLoginLog(ctx, consumer.GetId(), req.GetPhone(), consumerV1.LoginLog_PHONE, true, "", loginIP)

	// 生成JWT令牌
	accessToken, refreshToken, expiresIn, err := s.generateTokens(consumer)
	if err != nil {
		s.log.Errorf("generate tokens failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "generate tokens failed")
	}

	s.log.Infof("LoginByPhone success: consumer_id=%d", consumer.GetId())
	return &consumerV1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Consumer:     consumer,
	}, nil
}

// LoginByWechat 微信登录
func (s *ConsumerService) LoginByWechat(ctx context.Context, req *consumerV1.LoginByWechatRequest) (*consumerV1.LoginResponse, error) {
	s.log.Infof("LoginByWechat: code=%s", req.GetCode())

	// TODO: 调用微信API获取用户信息
	// 这里暂时返回未实现错误
	return nil, errors.New(501, "UNIMPLEMENTED", "wechat login not implemented yet")
}

// hashPassword 使用bcrypt加密密码
func (s *ConsumerService) hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// verifyPassword 验证密码
func (s *ConsumerService) verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// generateTokens 生成JWT令牌
func (s *ConsumerService) generateTokens(consumer *consumerV1.Consumer) (accessToken, refreshToken string, expiresIn int64, err error) {
	// 生成访问令牌（2小时有效期）
	accessToken, err = s.jwtHelper.GenerateToken(consumer.GetId(), 2*time.Hour)
	if err != nil {
		return "", "", 0, err
	}

	// 生成刷新令牌（7天有效期）
	refreshToken, err = s.jwtHelper.GenerateToken(consumer.GetId(), 7*24*time.Hour)
	if err != nil {
		return "", "", 0, err
	}

	expiresIn = int64(2 * time.Hour / time.Second)
	return accessToken, refreshToken, expiresIn, nil
}

// calculateRiskScore 计算风险评分
func (s *ConsumerService) calculateRiskScore(consumer *consumerV1.Consumer, loginSuccess bool) int32 {
	score := consumer.GetRiskScore()

	if loginSuccess {
		// 登录成功，降低风险评分
		score -= 5
		if score < 0 {
			score = 0
		}
	} else {
		// 登录失败，增加风险评分
		score += 10
		if score > 100 {
			score = 100
		}
	}

	// TODO: 可以添加更多风险评分逻辑
	// - 检测异常IP
	// - 检测频繁失败
	// - 检测异常登录时间
	// - 检测设备指纹变化

	return score
}

// getClientIP 获取客户端IP
func (s *ConsumerService) getClientIP(ctx context.Context) string {
	// TODO: 从context中提取真实IP
	// 这里暂时返回空字符串
	return ""
}

// recordLoginLog 记录登录日志
func (s *ConsumerService) recordLoginLog(ctx context.Context, consumerID uint32, phone string, loginType consumerV1.LoginLog_LoginType, success bool, failReason, ipAddress string) {
	loginLog := &consumerV1.LoginLog{
		ConsumerId: &consumerID,
		Phone:      &phone,
		LoginType:  &loginType,
		Success:    &success,
		IpAddress:  &ipAddress,
		LoginAt:    timestamppb.Now(),
	}

	if !success && failReason != "" {
		loginLog.FailReason = &failReason
	}

	if _, err := s.loginLogRepo.Create(ctx, loginLog); err != nil {
		s.log.Errorf("create login log failed: %v", err)
	}
}

// GetConsumer 获取用户信息
func (s *ConsumerService) GetConsumer(ctx context.Context, req *consumerV1.GetConsumerRequest) (*consumerV1.Consumer, error) {
	s.log.Infof("GetConsumer: id=%d", req.GetId())

	consumer, err := s.consumerRepo.Get(ctx, req.GetId())
	if err != nil {
		s.log.Errorf("get consumer failed: %v", err)
		return nil, err
	}

	// password_hash 不在 proto 中，无需处理

	return consumer, nil
}

// UpdateConsumer 更新用户信息
func (s *ConsumerService) UpdateConsumer(ctx context.Context, req *consumerV1.UpdateConsumerRequest) (*emptypb.Empty, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("UpdateConsumer: id=%d", currentUserID)

	// 构建更新数据
	updateData := &consumerV1.Consumer{
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		s.log.Errorf("update consumer failed: %v", err)
		return nil, err
	}

	s.log.Infof("UpdateConsumer success: id=%d", currentUserID)
	return &emptypb.Empty{}, nil
}

// UpdatePhone 更新手机号
func (s *ConsumerService) UpdatePhone(ctx context.Context, req *consumerV1.UpdatePhoneRequest) (*emptypb.Empty, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("UpdatePhone: id=%d, new_phone=%s", currentUserID, req.GetNewPhone())

	// TODO: 验证验证码（需要集成SMS Service）
	// 这里暂时跳过验证码验证

	// 检查新手机号是否已被使用
	existingUser, err := s.consumerRepo.GetByPhone(ctx, req.GetNewPhone())
	if err == nil && existingUser != nil && existingUser.GetId() != currentUserID {
		return nil, errors.Conflict("ALREADY_EXISTS", "phone number already in use")
	}

	// 更新手机号
	updateData := &consumerV1.Consumer{
		Phone: &req.NewPhone,
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		s.log.Errorf("update phone failed: %v", err)
		return nil, err
	}

	s.log.Infof("UpdatePhone success: id=%d", currentUserID)
	return &emptypb.Empty{}, nil
}

// UpdateEmail 更新邮箱
func (s *ConsumerService) UpdateEmail(ctx context.Context, req *consumerV1.UpdateEmailRequest) (*emptypb.Empty, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("UpdateEmail: id=%d, new_email=%s", currentUserID, req.GetNewEmail())

	// TODO: 验证验证码（需要集成Email Service）
	// 这里暂时跳过验证码验证

	// 更新邮箱
	updateData := &consumerV1.Consumer{
		Email: &req.NewEmail,
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		s.log.Errorf("update email failed: %v", err)
		return nil, err
	}

	s.log.Infof("UpdateEmail success: id=%d", currentUserID)
	return &emptypb.Empty{}, nil
}

// UploadAvatar 上传头像
func (s *ConsumerService) UploadAvatar(ctx context.Context, req *consumerV1.UploadAvatarRequest) (*consumerV1.UploadAvatarResponse, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("UploadAvatar: id=%d", currentUserID)

	// TODO: 集成Media Service上传图片
	// 这里暂时返回未实现错误
	_ = req // 暂时忽略请求参数
	return nil, errors.New(501, "UNIMPLEMENTED", "upload avatar not implemented yet")
}

// DeactivateAccount 注销账户
func (s *ConsumerService) DeactivateAccount(ctx context.Context, req *consumerV1.DeactivateAccountRequest) (*emptypb.Empty, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("DeactivateAccount: id=%d", currentUserID)

	// TODO: 验证密码 - password_hash 不在 proto 中，需要重构数据层来支持密码验证
	// 暂时跳过密码验证和用户查询
	_ = req // 暂时忽略请求参数
	/*
		// 获取用户信息
		consumer, err := s.consumerRepo.Get(ctx, currentUserID)
		if err != nil {
			s.log.Errorf("get consumer failed: %v", err)
			return nil, err
		}

		if err := s.verifyPassword(consumer.GetPasswordHash(), req.GetPassword()); err != nil {
			s.log.Warnf("password verification failed for deactivate account: id=%d", currentUserID)
			return nil, errors.Unauthorized("UNAUTHORIZED", "invalid password")
		}
	*/

	// 注销账户
	if err := s.consumerRepo.Deactivate(ctx, currentUserID); err != nil {
		s.log.Errorf("deactivate account failed: %v", err)
		return nil, err
	}

	s.log.Infof("DeactivateAccount success: id=%d", currentUserID)
	return &emptypb.Empty{}, nil
}

// ListLoginLogs 查询登录日志
func (s *ConsumerService) ListLoginLogs(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLoginLogsResponse, error) {
	// TODO: 从context中获取当前用户ID
	currentUserID := uint32(1) // 临时硬编码

	s.log.Infof("ListLoginLogs: consumer_id=%d", currentUserID)

	// TODO: 添加consumer_id过滤条件到req
	// 这里暂时直接查询

	resp, err := s.loginLogRepo.List(ctx, req)
	if err != nil {
		s.log.Errorf("list login logs failed: %v", err)
		return nil, err
	}

	// 脱敏处理
	for _, item := range resp.Items {
		if item.Phone != nil {
			phone := *item.Phone
			if len(phone) > 7 {
				masked := phone[:3] + "****" + phone[len(phone)-4:]
				item.Phone = &masked
			}
		}
		if item.IpAddress != nil {
			ip := *item.IpAddress
			if len(ip) > 0 {
				// 简单的IP脱敏：保留前两段
				// 例如：192.168.1.1 -> 192.168.*.*
				masked := maskIP(ip)
				item.IpAddress = &masked
			}
		}
	}

	return resp, nil
}

// ListConsumers 查询用户列表（管理员）
func (s *ConsumerService) ListConsumers(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListConsumersResponse, error) {
	s.log.Infof("ListConsumers")

	// TODO: 检查管理员权限

	resp, err := s.consumerRepo.List(ctx, req)
	if err != nil {
		s.log.Errorf("list consumers failed: %v", err)
		return nil, err
	}

	// password_hash 不在 proto 中，无需处理
	return resp, nil
}

// maskIP IP地址脱敏
func maskIP(ip string) string {
	// 简单实现：保留前两段
	// 例如：192.168.1.1 -> 192.168.*.*
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
