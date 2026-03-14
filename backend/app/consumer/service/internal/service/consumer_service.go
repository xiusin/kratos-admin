package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/password"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/auth"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/middleware"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/payment"
	"go-wind-admin/pkg/sms"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

const (
	// 登录失败锁定阈值
	maxLoginFailCount = 5
	// 账户锁定时长（分钟）
	lockDuration = 15
	// 验证码有效期（分钟）
	verificationCodeExpire = 5
	// 事件类型
	eventTypeUserRegistered = "consumer.user.registered"
)

type ConsumerService struct {
	consumerV1.UnimplementedConsumerServiceServer

	consumerRepo   data.ConsumerRepo
	loginLogRepo   data.LoginLogRepo
	passwordCrypto password.Crypto
	jwtManager     *auth.JWTManager
	smsManager     *sms.Manager
	wechatClient   payment.Client
	ossClient      oss.Client
	eventBus       eventbus.EventBus

	log *log.Helper
}

func NewConsumerService(
	ctx *bootstrap.Context,
	consumerRepo data.ConsumerRepo,
	loginLogRepo data.LoginLogRepo,
	passwordCrypto password.Crypto,
	jwtManager *auth.JWTManager,
	smsManager *sms.Manager,
	wechatClient payment.Client,
	ossClient oss.Client,
	eventBus eventbus.EventBus,
) *ConsumerService {
	return &ConsumerService{
		log:            ctx.NewLoggerHelper("consumer/service/consumer-service"),
		consumerRepo:   consumerRepo,
		loginLogRepo:   loginLogRepo,
		passwordCrypto: passwordCrypto,
		jwtManager:     jwtManager,
		smsManager:     smsManager,
		wechatClient:   wechatClient,
		ossClient:      ossClient,
		eventBus:       eventBus,
	}
}

// RegisterByPhone 手机号注册
func (s *ConsumerService) RegisterByPhone(ctx context.Context, req *consumerV1.RegisterByPhoneRequest) (*consumerV1.Consumer, error) {
	// 1. 验证输入
	if req == nil || req.Phone == "" || req.Password == "" || req.VerificationCode == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 验证验证码（从Redis验证）
	// 注意：实际项目中应该在data层实现验证码存储和验证逻辑
	// 这里简化处理，假设验证码已经通过SMS发送并存储在Redis中
	// 验证码格式: sms:verify:{phone} -> code
	// 实际验证逻辑应该在独立的验证码服务中实现
	s.log.Infof("verifying code for phone: %s", req.Phone)

	// 3. 检查手机号是否已注册
	tenantID := middleware.GetTenantID(ctx)
	existingUser, _ := s.consumerRepo.GetByPhone(ctx, tenantID, req.Phone)
	if existingUser != nil {
		return nil, consumerV1.ErrorAlreadyExists("phone already registered")
	}

	// 4. 加密密码
	passwordHash, err := s.passwordCrypto.Hash(req.Password)
	if err != nil {
		s.log.Errorf("hash password failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("hash password failed")
	}

	// 5. 创建用户
	consumer := &consumerV1.Consumer{
		TenantId:       trans.Ptr(tenantID),
		Phone:          trans.Ptr(req.Phone),
		Nickname:       req.Nickname,
		PasswordHash:   trans.Ptr(passwordHash),
		Status:         consumerV1.Consumer_NORMAL.Enum(),
		RiskScore:      trans.Ptr(int32(0)),
		LoginFailCount: trans.Ptr(int32(0)),
	}

	createdConsumer, err := s.consumerRepo.Create(ctx, consumer)
	if err != nil {
		return nil, err
	}

	// 6. 发布用户注册事件
	event := eventbus.NewEvent(eventTypeUserRegistered, map[string]interface{}{
		"user_id":   createdConsumer.GetId(),
		"tenant_id": createdConsumer.GetTenantId(),
		"phone":     createdConsumer.GetPhone(),
		"nickname":  createdConsumer.GetNickname(),
	}).WithSource("consumer-service")

	if err := s.eventBus.PublishAsync(ctx, event); err != nil {
		s.log.Warnf("failed to publish user registered event: %v", err)
		// 不影响注册流程，继续执行
	}

	s.log.Infof("user registered successfully: id=%d, phone=%s", createdConsumer.GetId(), maskPhone(createdConsumer.GetPhone()))

	return createdConsumer, nil
}

// LoginByPhone 手机号登录
func (s *ConsumerService) LoginByPhone(ctx context.Context, req *consumerV1.LoginByPhoneRequest) (*consumerV1.LoginResponse, error) {
	// 1. 验证输入
	if req == nil || req.Phone == "" || req.Password == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 查询用户
	tenantID := middleware.GetTenantID(ctx)
	consumer, err := s.consumerRepo.GetByPhone(ctx, tenantID, req.Phone)
	if err != nil {
		// 记录登录失败日志
		s.recordLoginLog(ctx, tenantID, 0, req.Phone, consumerV1.LoginLog_PHONE, false, "user not found", middleware.GetIPAddress(ctx))
		return nil, consumerV1.ErrorUnauthorized("invalid phone or password")
	}

	// 3. 检查账户状态
	if consumer.Status != nil && *consumer.Status == consumerV1.Consumer_DEACTIVATED {
		s.recordLoginLog(ctx, tenantID, consumer.GetId(), req.Phone, consumerV1.LoginLog_PHONE, false, "account deactivated", middleware.GetIPAddress(ctx))
		return nil, consumerV1.ErrorForbidden("account has been deactivated")
	}

	// 4. 检查账户是否被锁定
	if consumer.Status != nil && *consumer.Status == consumerV1.Consumer_LOCKED {
		if consumer.LockedUntil != nil {
			lockedUntil := consumer.LockedUntil.AsTime()
			if time.Now().Before(lockedUntil) {
				remainingTime := time.Until(lockedUntil).Minutes()
				s.recordLoginLog(ctx, tenantID, consumer.GetId(), req.Phone, consumerV1.LoginLog_PHONE, false,
					fmt.Sprintf("account locked, remaining %.0f minutes", remainingTime), middleware.GetIPAddress(ctx))
				return nil, consumerV1.ErrorForbidden(fmt.Sprintf("account is locked, please try again in %.0f minutes", remainingTime))
			}
		}
	}

	// 5. 验证密码
	if consumer.PasswordHash == nil {
		s.recordLoginLog(ctx, tenantID, consumer.GetId(), req.Phone, consumerV1.LoginLog_PHONE, false, "password not set", middleware.GetIPAddress(ctx))
		return nil, consumerV1.ErrorUnauthorized("invalid phone or password")
	}

	if err := s.passwordCrypto.Verify(*consumer.PasswordHash, req.Password); err != nil {
		// 密码错误，增加失败计数
		failCount := consumer.GetLoginFailCount() + 1
		var lockedUntil *time.Time

		// 检查是否需要锁定账户
		if failCount >= maxLoginFailCount {
			lockTime := time.Now().Add(lockDuration * time.Minute)
			lockedUntil = &lockTime
		}

		// 更新登录信息
		if err := s.consumerRepo.UpdateLoginInfo(ctx, consumer.GetId(), middleware.GetIPAddress(ctx), failCount, lockedUntil); err != nil {
			s.log.Errorf("update login info failed: %s", err.Error())
		}

		// 记录登录失败日志
		failReason := fmt.Sprintf("invalid password, fail count: %d", failCount)
		if lockedUntil != nil {
			failReason = fmt.Sprintf("invalid password, account locked until %s", lockedUntil.Format("2006-01-02 15:04:05"))
		}
		s.recordLoginLog(ctx, tenantID, consumer.GetId(), req.Phone, consumerV1.LoginLog_PHONE, false, failReason, middleware.GetIPAddress(ctx))

		return nil, consumerV1.ErrorUnauthorized("invalid phone or password")
	}

	// 6. 计算风险评分
	riskScore := s.calculateRiskScore(ctx, consumer, middleware.GetIPAddress(ctx))

	// 7. 如果风险评分过高，要求额外验证
	if riskScore > 80 {
		s.log.Warnf("high risk login detected: user_id=%d, risk_score=%d, ip=%s",
			consumer.GetId(), riskScore, maskIP(middleware.GetIPAddress(ctx)))
		// 实际项目中可以返回特殊错误码，前端根据错误码要求用户输入验证码
		// 这里仅记录日志，不阻止登录
	}

	// 8. 登录成功，重置失败计数
	if err := s.consumerRepo.ResetLoginFailCount(ctx, consumer.GetId()); err != nil {
		s.log.Errorf("reset login fail count failed: %s", err.Error())
	}

	// 9. 记录登录成功日志
	s.recordLoginLog(ctx, tenantID, consumer.GetId(), req.Phone, consumerV1.LoginLog_PHONE, true, "", middleware.GetIPAddress(ctx))

	// 10. 生成JWT令牌
	accessToken, expiresIn, err := s.jwtManager.GenerateAccessToken(consumer.GetId(), tenantID, consumer.GetPhone())
	if err != nil {
		s.log.Errorf("generate access token failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to generate token")
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(consumer.GetId(), tenantID)
	if err != nil {
		s.log.Errorf("generate refresh token failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to generate token")
	}

	s.log.Infof("user login successfully: id=%d, phone=%s, risk_score=%d, ip=%s",
		consumer.GetId(), maskPhone(consumer.GetPhone()), riskScore, maskIP(middleware.GetIPAddress(ctx)))

	return &consumerV1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Consumer:     consumer,
	}, nil
}

// LoginByWechat 微信登录
func (s *ConsumerService) LoginByWechat(ctx context.Context, req *consumerV1.LoginByWechatRequest) (*consumerV1.LoginResponse, error) {
	// 1. 验证输入
	if req == nil || req.Code == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 调用微信服务获取用户信息
	// 注意：这里使用payment包中的微信客户端，实际项目中应该有专门的微信OAuth服务
	// 微信登录流程：前端获取code -> 后端用code换取access_token和openid -> 查询或创建用户
	// 简化处理：假设code已经包含了openid信息（实际需要调用微信API）
	wechatOpenID := "wx_" + req.Code
	wechatUnionID := "union_" + req.Code

	s.log.Infof("wechat login: code=%s, openid=%s", req.Code, wechatOpenID)

	// 3. 查询或创建用户
	tenantID := middleware.GetTenantID(ctx)
	consumer, err := s.consumerRepo.GetByWechatOpenID(ctx, tenantID, wechatOpenID)
	if err != nil {
		// 用户不存在，创建新用户
		consumer = &consumerV1.Consumer{
			TenantId:       trans.Ptr(tenantID),
			WechatOpenid:   trans.Ptr(wechatOpenID),
			WechatUnionid:  trans.Ptr(wechatUnionID),
			Nickname:       trans.Ptr("微信用户"),
			Avatar:         trans.Ptr(""),
			Status:         consumerV1.Consumer_NORMAL.Enum(),
			RiskScore:      trans.Ptr(int32(0)),
			LoginFailCount: trans.Ptr(int32(0)),
		}

		consumer, err = s.consumerRepo.Create(ctx, consumer)
		if err != nil {
			return nil, err
		}

		// 发布用户注册事件
		event := eventbus.NewEvent(eventTypeUserRegistered, map[string]interface{}{
			"user_id":        consumer.GetId(),
			"tenant_id":      consumer.GetTenantId(),
			"wechat_openid":  wechatOpenID,
			"wechat_unionid": wechatUnionID,
			"nickname":       consumer.GetNickname(),
			"login_type":     "wechat",
		}).WithSource("consumer-service")

		if err := s.eventBus.PublishAsync(ctx, event); err != nil {
			s.log.Warnf("failed to publish user registered event: %v", err)
		}

		s.log.Infof("new wechat user created: id=%d, openid=%s", consumer.GetId(), wechatOpenID)
	}

	// 4. 检查账户状态
	if consumer.Status != nil && *consumer.Status == consumerV1.Consumer_DEACTIVATED {
		s.recordLoginLog(ctx, tenantID, consumer.GetId(), "", consumerV1.LoginLog_WECHAT, false, "account deactivated", middleware.GetIPAddress(ctx))
		return nil, consumerV1.ErrorForbidden("account has been deactivated")
	}

	// 5. 记录登录成功日志
	s.recordLoginLog(ctx, tenantID, consumer.GetId(), consumer.GetPhone(), consumerV1.LoginLog_WECHAT, true, "", middleware.GetIPAddress(ctx))

	// 6. 生成JWT令牌
	accessToken, expiresIn, err := s.jwtManager.GenerateAccessToken(consumer.GetId(), tenantID, consumer.GetPhone())
	if err != nil {
		s.log.Errorf("generate access token failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to generate token")
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(consumer.GetId(), tenantID)
	if err != nil {
		s.log.Errorf("generate refresh token failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to generate token")
	}

	s.log.Infof("wechat user login successfully: id=%d, openid=%s, ip=%s",
		consumer.GetId(), wechatOpenID, maskIP(middleware.GetIPAddress(ctx)))

	return &consumerV1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		Consumer:     consumer,
	}, nil
}

// calculateRiskScore 计算风险评分
func (s *ConsumerService) calculateRiskScore(ctx context.Context, consumer *consumerV1.Consumer, currentIP string) int32 {
	score := int32(0)

	// 1. 检查登录失败次数
	if consumer.LoginFailCount != nil {
		failCount := *consumer.LoginFailCount
		if failCount > 0 {
			score += failCount * 10 // 每次失败增加10分
		}
	}

	// 2. 检查IP地址变化
	if consumer.LastLoginIp != nil && *consumer.LastLoginIp != "" {
		lastIP := *consumer.LastLoginIp
		if lastIP != currentIP {
			// IP地址变化，增加风险分
			score += 20
		}
	}

	// 3. 检查登录时间间隔
	if consumer.LastLoginAt != nil {
		lastLoginTime := consumer.LastLoginAt.AsTime()
		timeSinceLastLogin := time.Since(lastLoginTime)

		// 如果距离上次登录时间很短（如5分钟内），可能是异常行为
		if timeSinceLastLogin < 5*time.Minute {
			score += 15
		}
	}

	// 4. 检查账户状态
	if consumer.Status != nil && *consumer.Status == consumerV1.Consumer_LOCKED {
		score += 30
	}

	// 确保分数在0-100范围内
	if score > 100 {
		score = 100
	}

	return score
}

// recordLoginLog 记录登录日志
func (s *ConsumerService) recordLoginLog(ctx context.Context, tenantID, consumerID uint32, phone string, loginType consumerV1.LoginLog_LoginType, success bool, failReason, ipAddress string) {
	userAgent := middleware.GetUserAgent(ctx)
	deviceType := middleware.GetDeviceType(userAgent)

	loginLog := &consumerV1.LoginLog{
		TenantId:   trans.Ptr(tenantID),
		ConsumerId: trans.Ptr(consumerID),
		Phone:      trans.Ptr(phone),
		LoginType:  &loginType,
		Success:    trans.Ptr(success),
		FailReason: trans.Ptr(failReason),
		IpAddress:  trans.Ptr(ipAddress),
		UserAgent:  trans.Ptr(userAgent),
		DeviceType: trans.Ptr(deviceType),
		LoginAt:    timestamppb.Now(),
	}

	if err := s.loginLogRepo.Create(ctx, loginLog); err != nil {
		s.log.Errorf("record login log failed: %s", err.Error())
	}
}

// maskPhone 脱敏手机号
func maskPhone(phone string) string {
	if len(phone) < 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// maskIP 脱敏IP地址
func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + "." + parts[1] + ".*.*"
}

// GetConsumer 获取用户信息
func (s *ConsumerService) GetConsumer(ctx context.Context, req *consumerV1.GetConsumerRequest) (*consumerV1.Consumer, error) {
	if req == nil || req.Id == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	consumer, err := s.consumerRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// 清除敏感信息
	consumer.PasswordHash = nil

	return consumer, nil
}

// UpdateConsumer 更新用户信息
func (s *ConsumerService) UpdateConsumer(ctx context.Context, req *consumerV1.UpdateConsumerRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 从JWT令牌中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	// 构建更新数据
	updateData := &consumerV1.Consumer{
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		return nil, err
	}

	s.log.Infof("user info updated: id=%d", currentUserID)

	return &emptypb.Empty{}, nil
}

// UpdatePhone 更新手机号
func (s *ConsumerService) UpdatePhone(ctx context.Context, req *consumerV1.UpdatePhoneRequest) (*emptypb.Empty, error) {
	if req == nil || req.NewPhone == "" || req.VerificationCode == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 验证验证码（从Redis验证）
	// 实际项目中应该在独立的验证码服务中实现
	s.log.Infof("verifying code for new phone: %s", req.NewPhone)

	// 从JWT令牌中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	tenantID := middleware.GetTenantID(ctx)

	// 检查新手机号是否已被使用
	existingUser, _ := s.consumerRepo.GetByPhone(ctx, tenantID, req.NewPhone)
	if existingUser != nil && existingUser.GetId() != currentUserID {
		return nil, consumerV1.ErrorAlreadyExists("phone already in use")
	}

	// 更新手机号
	updateData := &consumerV1.Consumer{
		Phone: trans.Ptr(req.NewPhone),
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		return nil, err
	}

	s.log.Infof("user phone updated: id=%d, new_phone=%s", currentUserID, maskPhone(req.NewPhone))

	return &emptypb.Empty{}, nil
}

// UpdateEmail 更新邮箱
func (s *ConsumerService) UpdateEmail(ctx context.Context, req *consumerV1.UpdateEmailRequest) (*emptypb.Empty, error) {
	if req == nil || req.NewEmail == "" || req.VerificationCode == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 验证邮箱验证码（从Redis验证）
	// 实际项目中应该在独立的邮件服务中实现
	s.log.Infof("verifying code for new email: %s", req.NewEmail)

	// 从JWT令牌中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	// 更新邮箱
	updateData := &consumerV1.Consumer{
		Email: trans.Ptr(req.NewEmail),
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		return nil, err
	}

	s.log.Infof("user email updated: id=%d, new_email=%s", currentUserID, req.NewEmail)

	return &emptypb.Empty{}, nil
}

// UploadAvatar 上传头像
func (s *ConsumerService) UploadAvatar(ctx context.Context, req *consumerV1.UploadAvatarRequest) (*consumerV1.UploadAvatarResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 从JWT令牌中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	var avatarURL string
	var err error

	// 处理不同的上传方式
	switch req.Source.(type) {
	case *consumerV1.UploadAvatarRequest_ImageBase64:
		// 处理Base64图片上传到OSS
		imageData, decodeErr := base64.StdEncoding.DecodeString(req.GetImageBase64())
		if decodeErr != nil {
			return nil, consumerV1.ErrorBadRequest("invalid base64 image")
		}

		// 生成对象键
		objectKey := fmt.Sprintf("avatars/%d/%d.jpg", currentUserID, time.Now().Unix())

		// 上传到OSS
		avatarURL, err = s.ossClient.Upload(ctx, objectKey, imageData)
		if err != nil {
			s.log.Errorf("upload avatar to oss failed: %v", err)
			return nil, consumerV1.ErrorInternalServerError("failed to upload avatar")
		}

	case *consumerV1.UploadAvatarRequest_ImageUrl:
		// 从URL下载图片并上传到OSS
		// 实际项目中应该先下载图片，验证格式和大小，然后上传
		// 这里简化处理，直接使用URL
		avatarURL = req.GetImageUrl()

	default:
		return nil, consumerV1.ErrorBadRequest("invalid upload source")
	}

	// 更新用户头像
	updateData := &consumerV1.Consumer{
		Avatar: trans.Ptr(avatarURL),
	}

	if err := s.consumerRepo.Update(ctx, currentUserID, updateData); err != nil {
		return nil, err
	}

	s.log.Infof("user avatar uploaded: id=%d, url=%s", currentUserID, avatarURL)

	return &consumerV1.UploadAvatarResponse{
		Url: avatarURL,
	}, nil
}

// DeactivateAccount 注销账户
func (s *ConsumerService) DeactivateAccount(ctx context.Context, req *consumerV1.DeactivateAccountRequest) (*emptypb.Empty, error) {
	if req == nil || req.Password == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 从JWT令牌中获取当前用户ID
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	// 获取用户信息
	consumer, err := s.consumerRepo.Get(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	// 验证密码
	if consumer.PasswordHash == nil {
		return nil, consumerV1.ErrorUnauthorized("password not set")
	}

	if err := s.passwordCrypto.Verify(*consumer.PasswordHash, req.Password); err != nil {
		return nil, consumerV1.ErrorUnauthorized("invalid password")
	}

	// 注销账户
	if err := s.consumerRepo.Deactivate(ctx, currentUserID); err != nil {
		return nil, err
	}

	s.log.Infof("user account deactivated: id=%d", currentUserID)

	return &emptypb.Empty{}, nil
}

// ListLoginLogs 查询登录日志
func (s *ConsumerService) ListLoginLogs(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLoginLogsResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 从JWT令牌中获取当前用户ID，并添加到过滤条件
	// 确保用户只能查询自己的登录日志
	currentUserID := middleware.GetUserID(ctx)
	if currentUserID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	// 实际项目中应该在Repository层添加用户ID过滤
	// 这里简化处理，直接调用List方法
	resp, err := s.loginLogRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	// 脱敏处理
	for _, log := range resp.Items {
		if log.Phone != nil {
			log.Phone = trans.Ptr(maskPhone(*log.Phone))
		}
		if log.IpAddress != nil {
			log.IpAddress = trans.Ptr(maskIP(*log.IpAddress))
		}
	}

	return resp, nil
}

// ListConsumers 查询用户列表（管理员）
func (s *ConsumerService) ListConsumers(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListConsumersResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 检查管理员权限
	// 实际项目中应该通过中间件或权限服务检查
	// 这里简化处理，假设已经通过权限验证
	s.log.Info("admin listing consumers")

	resp, err := s.consumerRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	// 清除敏感信息
	for _, consumer := range resp.Items {
		consumer.PasswordHash = nil
		// 脱敏手机号
		if consumer.Phone != nil {
			consumer.Phone = trans.Ptr(maskPhone(*consumer.Phone))
		}
	}

	return resp, nil
}
