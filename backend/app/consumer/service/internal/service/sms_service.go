package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/sms"
)

const (
	// 验证码相关常量
	verificationCodeLength = 6               // 验证码长度
	verificationCodeTTL    = 5 * time.Minute // 验证码有效期 5分钟

	// 频率限制常量
	rateLimitPerMinute = 1  // 每分钟最多发送1条
	rateLimitPerDay    = 10 // 每天最多发送10条

	// Redis Key 前缀
	redisKeyVerificationCode = "sms:verification:%s"     // 验证码存储 key
	redisKeyRateLimitMinute  = "sms:ratelimit:minute:%s" // 分钟级限流 key
	redisKeyRateLimitDay     = "sms:ratelimit:day:%s"    // 天级限流 key
)

// SMSClients SMS客户端集合
type SMSClients struct {
	Aliyun  sms.Client
	Tencent sms.Client
}

// SMSService 短信服务
type SMSService struct {
	consumerV1.UnimplementedSMSServiceServer

	smsLogRepo    data.SMSLogRepo
	rdb           *redis.Client
	aliyunClient  sms.Client
	tencentClient sms.Client
	log           *log.Helper
}

// NewSMSService 创建短信服务实例
func NewSMSService(
	ctx *bootstrap.Context,
	smsLogRepo data.SMSLogRepo,
	rdb *redis.Client,
	smsClients *SMSClients,
) *SMSService {
	return &SMSService{
		smsLogRepo:    smsLogRepo,
		rdb:           rdb,
		aliyunClient:  smsClients.Aliyun,
		tencentClient: smsClients.Tencent,
		log:           ctx.NewLoggerHelper("sms/service/consumer-service"),
	}
}

// SendVerificationCode 发送验证码
func (s *SMSService) SendVerificationCode(ctx context.Context, req *consumerV1.SendVerificationCodeRequest) (*emptypb.Empty, error) {
	s.log.Infof("SendVerificationCode: phone=%s, scene=%s", req.GetPhone(), req.GetScene())

	phone := req.GetPhone()

	// 1. 检查频率限制
	if err := s.checkRateLimit(ctx, phone); err != nil {
		return nil, err
	}

	// 2. 生成6位数字验证码
	code, err := s.generateVerificationCode()
	if err != nil {
		s.log.Errorf("generate verification code failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "generate verification code failed")
	}

	// 3. 存储验证码到Redis (5分钟过期)
	if err := s.storeVerificationCode(ctx, phone, code); err != nil {
		s.log.Errorf("store verification code failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "store verification code failed")
	}

	// 4. 发送短信 (带故障转移)
	channel, err := s.sendSMSWithFailover(ctx, phone, code)
	if err != nil {
		s.log.Errorf("send sms failed: %v", err)
		// 记录失败日志
		s.recordSMSLog(ctx, phone, consumerV1.SMSLog_VERIFICATION, code, channel, consumerV1.SMSLog_FAILED, err.Error())
		return nil, errors.InternalServer("INTERNAL_ERROR", "send sms failed")
	}

	// 5. 记录成功日志
	s.recordSMSLog(ctx, phone, consumerV1.SMSLog_VERIFICATION, code, channel, consumerV1.SMSLog_SUCCESS, "")

	// 6. 更新频率限制计数
	s.incrementRateLimit(ctx, phone)

	s.log.Infof("SendVerificationCode success: phone=%s, code=%s, channel=%s", phone, code, channel)
	return &emptypb.Empty{}, nil
}

// VerifyCode 验证验证码
func (s *SMSService) VerifyCode(ctx context.Context, req *consumerV1.VerifyCodeRequest) (*consumerV1.VerifyCodeResponse, error) {
	s.log.Infof("VerifyCode: phone=%s", req.GetPhone())

	phone := req.GetPhone()
	code := req.GetCode()

	// 1. 从Redis获取验证码
	storedCode, err := s.getVerificationCode(ctx, phone)
	if err != nil {
		if err == redis.Nil {
			return &consumerV1.VerifyCodeResponse{
				Valid:   false,
				Message: stringPtr("verification code expired or not found"),
			}, nil
		}
		s.log.Errorf("get verification code failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "get verification code failed")
	}

	// 2. 验证验证码
	if storedCode != code {
		return &consumerV1.VerifyCodeResponse{
			Valid:   false,
			Message: stringPtr("invalid verification code"),
		}, nil
	}

	// 3. 验证成功后立即删除验证码 (一次性使用)
	if err := s.deleteVerificationCode(ctx, phone); err != nil {
		s.log.Errorf("delete verification code failed: %v", err)
		// 不影响验证结果
	}

	s.log.Infof("VerifyCode success: phone=%s", phone)
	return &consumerV1.VerifyCodeResponse{
		Valid:   true,
		Message: stringPtr("verification code is valid"),
	}, nil
}

// SendNotification 发送通知短信
func (s *SMSService) SendNotification(ctx context.Context, req *consumerV1.SendNotificationRequest) (*emptypb.Empty, error) {
	s.log.Infof("SendNotification: phone=%s, template=%s", req.GetPhone(), req.GetTemplateCode())

	phone := req.GetPhone()
	templateCode := req.GetTemplateCode()
	params := req.GetParams()

	// 1. 发送短信 (带故障转移)
	channel, err := s.sendNotificationWithFailover(ctx, phone, templateCode, params)
	if err != nil {
		s.log.Errorf("send notification failed: %v", err)
		// 记录失败日志
		content := fmt.Sprintf("template=%s, params=%v", templateCode, params)
		s.recordSMSLog(ctx, phone, consumerV1.SMSLog_NOTIFICATION, content, channel, consumerV1.SMSLog_FAILED, err.Error())
		return nil, errors.InternalServer("INTERNAL_ERROR", "send notification failed")
	}

	// 2. 记录成功日志
	content := fmt.Sprintf("template=%s, params=%v", templateCode, params)
	s.recordSMSLog(ctx, phone, consumerV1.SMSLog_NOTIFICATION, content, channel, consumerV1.SMSLog_SUCCESS, "")

	s.log.Infof("SendNotification success: phone=%s, template=%s, channel=%s", phone, templateCode, channel)
	return &emptypb.Empty{}, nil
}

// ListSMSLogs 查询短信日志
func (s *SMSService) ListSMSLogs(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListSMSLogsResponse, error) {
	s.log.Infof("ListSMSLogs: page=%d, pageSize=%d", req.GetPage(), req.GetPageSize())

	return s.smsLogRepo.List(ctx, req)
}

// generateVerificationCode 生成6位数字验证码
func (s *SMSService) generateVerificationCode() (string, error) {
	// 生成 100000 到 999999 之间的随机数
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	code := n.Int64() + 100000
	return fmt.Sprintf("%06d", code), nil
}

// storeVerificationCode 存储验证码到Redis
func (s *SMSService) storeVerificationCode(ctx context.Context, phone string, code string) error {
	key := fmt.Sprintf(redisKeyVerificationCode, phone)
	return s.rdb.Set(ctx, key, code, verificationCodeTTL).Err()
}

// getVerificationCode 从Redis获取验证码
func (s *SMSService) getVerificationCode(ctx context.Context, phone string) (string, error) {
	key := fmt.Sprintf(redisKeyVerificationCode, phone)
	return s.rdb.Get(ctx, key).Result()
}

// deleteVerificationCode 从Redis删除验证码
func (s *SMSService) deleteVerificationCode(ctx context.Context, phone string) error {
	key := fmt.Sprintf(redisKeyVerificationCode, phone)
	return s.rdb.Del(ctx, key).Err()
}

// checkRateLimit 检查频率限制
func (s *SMSService) checkRateLimit(ctx context.Context, phone string) error {
	// 检查每分钟限制
	minuteKey := fmt.Sprintf(redisKeyRateLimitMinute, phone)
	minuteCount, err := s.rdb.Get(ctx, minuteKey).Int()
	if err != nil && err != redis.Nil {
		return err
	}
	if minuteCount >= rateLimitPerMinute {
		return errors.New(429, "TOO_MANY_REQUESTS", "too many requests per minute, please try again later")
	}

	// 检查每日限制
	dayKey := fmt.Sprintf(redisKeyRateLimitDay, phone)
	dayCount, err := s.rdb.Get(ctx, dayKey).Int()
	if err != nil && err != redis.Nil {
		return err
	}
	if dayCount >= rateLimitPerDay {
		return errors.New(429, "TOO_MANY_REQUESTS", "daily sms limit exceeded")
	}

	return nil
}

// incrementRateLimit 增加频率限制计数
func (s *SMSService) incrementRateLimit(ctx context.Context, phone string) {
	// 增加每分钟计数
	minuteKey := fmt.Sprintf(redisKeyRateLimitMinute, phone)
	s.rdb.Incr(ctx, minuteKey)
	s.rdb.Expire(ctx, minuteKey, time.Minute)

	// 增加每日计数
	dayKey := fmt.Sprintf(redisKeyRateLimitDay, phone)
	s.rdb.Incr(ctx, dayKey)
	// 设置过期时间为当天结束
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	s.rdb.Expire(ctx, dayKey, time.Until(endOfDay))
}

// sendSMSWithFailover 发送短信 (带故障转移)
func (s *SMSService) sendSMSWithFailover(ctx context.Context, phone string, code string) (consumerV1.SMSLog_Channel, error) {
	// 优先使用阿里云
	if s.aliyunClient != nil {
		err := s.aliyunClient.SendVerificationCode(ctx, phone, code)
		if err == nil {
			return consumerV1.SMSLog_ALIYUN, nil
		}
		s.log.Warnf("aliyun sms failed, fallback to tencent: %v", err)
	}

	// 故障转移到腾讯云
	if s.tencentClient != nil {
		err := s.tencentClient.SendVerificationCode(ctx, phone, code)
		if err == nil {
			return consumerV1.SMSLog_TENCENT, nil
		}
		s.log.Errorf("tencent sms also failed: %v", err)
		return consumerV1.SMSLog_TENCENT, err
	}

	return consumerV1.SMSLog_ALIYUN, fmt.Errorf("no sms client available")
}

// sendNotificationWithFailover 发送通知短信 (带故障转移)
func (s *SMSService) sendNotificationWithFailover(ctx context.Context, phone string, templateCode string, params map[string]string) (consumerV1.SMSLog_Channel, error) {
	// 优先使用阿里云
	if s.aliyunClient != nil {
		err := s.aliyunClient.Send(ctx, phone, templateCode, params)
		if err == nil {
			return consumerV1.SMSLog_ALIYUN, nil
		}
		s.log.Warnf("aliyun sms failed, fallback to tencent: %v", err)
	}

	// 故障转移到腾讯云
	if s.tencentClient != nil {
		err := s.tencentClient.Send(ctx, phone, templateCode, params)
		if err == nil {
			return consumerV1.SMSLog_TENCENT, nil
		}
		s.log.Errorf("tencent sms also failed: %v", err)
		return consumerV1.SMSLog_TENCENT, err
	}

	return consumerV1.SMSLog_ALIYUN, fmt.Errorf("no sms client available")
}

// recordSMSLog 记录短信日志
func (s *SMSService) recordSMSLog(ctx context.Context, phone string, smsType consumerV1.SMSLog_SMSType, code string, channel consumerV1.SMSLog_Channel, status consumerV1.SMSLog_Status, errorMsg string) {
	now := time.Now()
	smsLog := &consumerV1.SMSLog{
		Phone:   &phone,
		SmsType: &smsType,
		Content: stringPtr(fmt.Sprintf("code=%s", code)),
		Code:    stringPtr(code),
		Channel: &channel,
		Status:  &status,
		SentAt:  timestamppb.New(now),
	}

	if smsType == consumerV1.SMSLog_VERIFICATION {
		expiresAt := now.Add(verificationCodeTTL)
		smsLog.ExpiresAt = timestamppb.New(expiresAt)
	}

	if errorMsg != "" {
		smsLog.ErrorMessage = &errorMsg
	}

	_, err := s.smsLogRepo.Create(ctx, smsLog)
	if err != nil {
		s.log.Errorf("record sms log failed: %v", err)
	}
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}
