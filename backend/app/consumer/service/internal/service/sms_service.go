package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/sms"
)

const (
	// 验证码相关常量
	verificationCodeLength = 6                   // 验证码长度
	verificationCodeTTL    = 5 * time.Minute     // 验证码有效期：5分钟
	verificationCodePrefix = "sms:verification:" // Redis key前缀

	// 频率限制常量
	rateLimitMinutePrefix = "sms:ratelimit:minute:" // 每分钟限制前缀
	rateLimitDayPrefix    = "sms:ratelimit:day:"    // 每日限制前缀
	rateLimitMinuteTTL    = 1 * time.Minute         // 每分钟限制TTL
	rateLimitDayTTL       = 24 * time.Hour          // 每日限制TTL
	rateLimitMinuteMax    = 1                       // 每分钟最多发送1条
	rateLimitDayMax       = 10                      // 每天最多发送10条
)

// SMSService 短信服务
type SMSService struct {
	consumerV1.UnimplementedSMSServiceServer

	log        *log.Helper
	smsManager *sms.Manager
	smsLogRepo data.SMSLogRepo
	redis      *redis.Client
}

// NewSMSService 创建短信服务实例
func NewSMSService(
	ctx *bootstrap.Context,
	smsManager *sms.Manager,
	smsLogRepo data.SMSLogRepo,
	redis *redis.Client,
) *SMSService {
	return &SMSService{
		log:        ctx.NewLoggerHelper("consumer/service/sms-service"),
		smsManager: smsManager,
		smsLogRepo: smsLogRepo,
		redis:      redis,
	}
}

// SendVerificationCode 发送验证码
func (s *SMSService) SendVerificationCode(ctx context.Context, req *consumerV1.SendVerificationCodeRequest) (*emptypb.Empty, error) {
	if req == nil || req.Phone == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	phone := *req.Phone

	// 1. 检查频率限制
	if err := s.checkRateLimit(ctx, phone); err != nil {
		return nil, err
	}

	// 2. 生成6位数字验证码
	code, err := s.generateVerificationCode()
	if err != nil {
		s.log.Errorf("generate verification code failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("generate verification code failed")
	}

	// 3. 发送短信（支持故障转移）
	channel, err := s.sendSMS(ctx, phone, code)
	if err != nil {
		// 记录失败日志
		s.recordSMSLog(ctx, phone, code, channel, consumerV1.SMSLog_FAILED, err.Error())
		return nil, err
	}

	// 4. 存储验证码到Redis（5分钟过期）
	if err := s.storeVerificationCode(ctx, phone, code); err != nil {
		s.log.Errorf("store verification code failed: %v", err)
		// 即使存储失败，短信已发送，不返回错误
	}

	// 5. 记录成功日志
	expiresAt := time.Now().Add(verificationCodeTTL)
	s.recordSMSLog(ctx, phone, code, channel, consumerV1.SMSLog_SUCCESS, "")
	s.recordSMSLogWithExpiry(ctx, phone, code, channel, consumerV1.SMSLog_SUCCESS, "", &expiresAt)

	// 6. 增加频率限制计数
	s.incrementRateLimit(ctx, phone)

	return &emptypb.Empty{}, nil
}

// VerifyCode 验证验证码
func (s *SMSService) VerifyCode(ctx context.Context, req *consumerV1.VerifyCodeRequest) (*consumerV1.VerifyCodeResponse, error) {
	if req == nil || req.Phone == nil || req.Code == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	phone := *req.Phone
	code := *req.Code

	// 1. 从Redis获取验证码
	storedCode, err := s.getVerificationCode(ctx, phone)
	if err != nil {
		if err == redis.Nil {
			return &consumerV1.VerifyCodeResponse{
				Valid:   false,
				Message: strPtr("验证码不存在或已过期"),
			}, nil
		}
		s.log.Errorf("get verification code failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("verify code failed")
	}

	// 2. 验证验证码
	if storedCode != code {
		return &consumerV1.VerifyCodeResponse{
			Valid:   false,
			Message: strPtr("验证码错误"),
		}, nil
	}

	// 3. 验证成功，立即删除验证码（一次性使用）
	if err := s.deleteVerificationCode(ctx, phone); err != nil {
		s.log.Warnf("delete verification code failed: %v", err)
		// 删除失败不影响验证结果
	}

	return &consumerV1.VerifyCodeResponse{
		Valid:   true,
		Message: strPtr("验证成功"),
	}, nil
}

// SendNotification 发送通知短信
func (s *SMSService) SendNotification(ctx context.Context, req *consumerV1.SendNotificationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Phone == nil || req.TemplateCode == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	phone := *req.Phone
	templateCode := *req.TemplateCode
	params := req.Params

	// 1. 发送短信（支持故障转移）
	channel, err := s.sendNotificationSMS(ctx, phone, templateCode, params)
	if err != nil {
		// 记录失败日志
		content := fmt.Sprintf("Template: %s, Params: %v", templateCode, params)
		s.recordNotificationLog(ctx, phone, content, channel, consumerV1.SMSLog_FAILED, err.Error())
		return nil, err
	}

	// 2. 记录成功日志
	content := fmt.Sprintf("Template: %s, Params: %v", templateCode, params)
	s.recordNotificationLog(ctx, phone, content, channel, consumerV1.SMSLog_SUCCESS, "")

	return &emptypb.Empty{}, nil
}

// ListSMSLogs 查询短信日志
func (s *SMSService) ListSMSLogs(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListSMSLogsResponse, error) {
	return s.smsLogRepo.List(ctx, req)
}

// ========== 私有辅助方法 ==========

// generateVerificationCode 生成6位数字验证码
func (s *SMSService) generateVerificationCode() (string, error) {
	// 生成范围：100000-999999
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	code := n.Int64() + 100000
	return fmt.Sprintf("%06d", code), nil
}

// sendSMS 发送验证码短信（支持故障转移）
func (s *SMSService) sendSMS(ctx context.Context, phone string, code string) (consumerV1.SMSLog_Channel, error) {
	err := s.smsManager.SendVerificationCode(ctx, phone, code)
	if err != nil {
		return s.getChannelFromProvider(s.smsManager.GetPrimaryProvider()),
			consumerV1.ErrorSMSServiceUnavailable("短信发送失败: %v", err)
	}

	// 返回实际使用的通道
	// 注意：这里简化处理，实际应该从 Manager 返回使用的通道
	return s.getChannelFromProvider(s.smsManager.GetPrimaryProvider()), nil
}

// sendNotificationSMS 发送通知短信（支持故障转移）
func (s *SMSService) sendNotificationSMS(ctx context.Context, phone string, templateCode string, params map[string]string) (consumerV1.SMSLog_Channel, error) {
	err := s.smsManager.Send(ctx, phone, templateCode, params)
	if err != nil {
		return s.getChannelFromProvider(s.smsManager.GetPrimaryProvider()),
			consumerV1.ErrorSMSServiceUnavailable("短信发送失败: %v", err)
	}

	return s.getChannelFromProvider(s.smsManager.GetPrimaryProvider()), nil
}

// getChannelFromProvider 从提供商类型转换为通道枚举
func (s *SMSService) getChannelFromProvider(provider sms.Provider) consumerV1.SMSLog_Channel {
	switch provider {
	case sms.ProviderAliyun:
		return consumerV1.SMSLog_ALIYUN
	case sms.ProviderTencent:
		return consumerV1.SMSLog_TENCENT
	default:
		return consumerV1.SMSLog_ALIYUN
	}
}

// storeVerificationCode 存储验证码到Redis
func (s *SMSService) storeVerificationCode(ctx context.Context, phone string, code string) error {
	key := verificationCodePrefix + phone
	return s.redis.Set(ctx, key, code, verificationCodeTTL).Err()
}

// getVerificationCode 从Redis获取验证码
func (s *SMSService) getVerificationCode(ctx context.Context, phone string) (string, error) {
	key := verificationCodePrefix + phone
	return s.redis.Get(ctx, key).Result()
}

// deleteVerificationCode 删除验证码（一次性使用）
func (s *SMSService) deleteVerificationCode(ctx context.Context, phone string) error {
	key := verificationCodePrefix + phone
	return s.redis.Del(ctx, key).Err()
}

// checkRateLimit 检查频率限制
func (s *SMSService) checkRateLimit(ctx context.Context, phone string) error {
	// 1. 检查每分钟限制
	minuteKey := rateLimitMinutePrefix + phone
	minuteCount, err := s.redis.Get(ctx, minuteKey).Int()
	if err != nil && err != redis.Nil {
		s.log.Warnf("get minute rate limit failed: %v", err)
	}
	if minuteCount >= rateLimitMinuteMax {
		return consumerV1.ErrorTooManyRequests("发送过于频繁，请1分钟后再试")
	}

	// 2. 检查每日限制
	dayKey := rateLimitDayPrefix + phone
	dayCount, err := s.redis.Get(ctx, dayKey).Int()
	if err != nil && err != redis.Nil {
		s.log.Warnf("get day rate limit failed: %v", err)
	}
	if dayCount >= rateLimitDayMax {
		return consumerV1.ErrorTooManyRequests("今日发送次数已达上限，请明天再试")
	}

	return nil
}

// incrementRateLimit 增加频率限制计数
func (s *SMSService) incrementRateLimit(ctx context.Context, phone string) {
	// 1. 增加每分钟计数
	minuteKey := rateLimitMinutePrefix + phone
	count, err := s.redis.Incr(ctx, minuteKey).Result()
	if err != nil {
		s.log.Warnf("increment minute rate limit failed: %v", err)
	}
	if count == 1 {
		// 第一次设置过期时间
		s.redis.Expire(ctx, minuteKey, rateLimitMinuteTTL)
	}

	// 2. 增加每日计数
	dayKey := rateLimitDayPrefix + phone
	count, err = s.redis.Incr(ctx, dayKey).Result()
	if err != nil {
		s.log.Warnf("increment day rate limit failed: %v", err)
	}
	if count == 1 {
		// 第一次设置过期时间
		s.redis.Expire(ctx, dayKey, rateLimitDayTTL)
	}
}

// recordSMSLog 记录短信日志（验证码）
func (s *SMSService) recordSMSLog(ctx context.Context, phone string, code string, channel consumerV1.SMSLog_Channel, status consumerV1.SMSLog_Status, errorMsg string) {
	s.recordSMSLogWithExpiry(ctx, phone, code, channel, status, errorMsg, nil)
}

// recordSMSLogWithExpiry 记录短信日志（验证码，带过期时间）
func (s *SMSService) recordSMSLogWithExpiry(ctx context.Context, phone string, code string, channel consumerV1.SMSLog_Channel, status consumerV1.SMSLog_Status, errorMsg string, expiresAt *time.Time) {
	logData := &consumerV1.SMSLog{
		Phone:   &phone,
		SmsType: consumerV1.SMSLog_VERIFICATION.Enum(),
		Content: strPtr(fmt.Sprintf("验证码: %s", code)),
		Code:    &code,
		Channel: &channel,
		Status:  &status,
		SentAt:  timestamppb.Now(),
	}

	if errorMsg != "" {
		logData.ErrorMessage = &errorMsg
	}

	if expiresAt != nil {
		logData.ExpiresAt = timestamppb.New(*expiresAt)
	}

	_, err := s.smsLogRepo.Create(ctx, logData)
	if err != nil {
		s.log.Errorf("record sms log failed: %v", err)
	}
}

// recordNotificationLog 记录通知短信日志
func (s *SMSService) recordNotificationLog(ctx context.Context, phone string, content string, channel consumerV1.SMSLog_Channel, status consumerV1.SMSLog_Status, errorMsg string) {
	logData := &consumerV1.SMSLog{
		Phone:   &phone,
		SmsType: consumerV1.SMSLog_NOTIFICATION.Enum(),
		Content: &content,
		Channel: &channel,
		Status:  &status,
		SentAt:  timestamppb.Now(),
	}

	if errorMsg != "" {
		logData.ErrorMessage = &errorMsg
	}

	_, err := s.smsLogRepo.Create(ctx, logData)
	if err != nil {
		s.log.Errorf("record notification log failed: %v", err)
	}
}

// strPtr 字符串指针辅助函数
func strPtr(s string) *string {
	return &s
}
