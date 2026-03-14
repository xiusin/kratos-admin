package service

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/middleware"
)

// FreightService 运费计算服务
type FreightService struct {
	consumerV1.UnimplementedFreightServiceServer

	log                 *log.Helper
	freightTemplateRepo data.FreightTemplateRepo
}

// NewFreightService 创建运费计算服务实例
func NewFreightService(
	ctx *bootstrap.Context,
	freightTemplateRepo data.FreightTemplateRepo,
) *FreightService {
	return &FreightService{
		log:                 ctx.NewLoggerHelper("consumer/service/freight-service"),
		freightTemplateRepo: freightTemplateRepo,
	}
}

// CalculateFreight 计算运费
func (s *FreightService) CalculateFreight(ctx context.Context, req *consumerV1.CalculateFreightRequest) (*consumerV1.CalculateFreightResponse, error) {
	// 1. 验证输入
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取运费模板
	var template *consumerV1.FreightTemplate
	var err error

	if req.TemplateId != nil && *req.TemplateId > 0 {
		// 使用指定模板
		template, err = s.freightTemplateRepo.Get(ctx, *req.TemplateId)
		if err != nil {
			s.log.Errorf("get freight template failed: %v", err)
			return nil, consumerV1.ErrorNotFound("freight template not found")
		}
	} else {
		// 使用默认模板（查询第一个启用的模板）
		pagingReq := &paginationV1.PagingRequest{
			Page:     trans.Ptr(uint32(1)),
			PageSize: trans.Ptr(uint32(1)),
		}
		resp, err := s.freightTemplateRepo.List(ctx, pagingReq)
		if err != nil || resp == nil || len(resp.Items) == 0 {
			s.log.Errorf("get default freight template failed: %v", err)
			return nil, consumerV1.ErrorNotFound("no freight template available")
		}
		template = resp.Items[0]
	}

	// 3. 检查包邮规则
	isFreeShipping, freeShippingReason := s.checkFreeShipping(req, template)
	if isFreeShipping {
		return &consumerV1.CalculateFreightResponse{
			Freight:            "0.00",
			IsFreeShipping:     true,
			FreeShippingReason: trans.Ptr(freeShippingReason),
			CalculationDetail:  trans.Ptr("满足包邮条件"),
		}, nil
	}

	// 4. 根据计算方式计算运费
	var freight decimal.Decimal
	var calculationDetail string

	switch template.GetCalculationType() {
	case consumerV1.FreightTemplate_BY_WEIGHT:
		freight, calculationDetail, err = s.calculateByWeight(req, template)
	case consumerV1.FreightTemplate_BY_DISTANCE:
		freight, calculationDetail, err = s.calculateByDistance(req, template)
	default:
		return nil, consumerV1.ErrorBadRequest("unsupported calculation type")
	}

	if err != nil {
		s.log.Errorf("calculate freight failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to calculate freight")
	}

	// 5. 确保运费精度（保留两位小数）
	freight = freight.Round(2)

	s.log.Infof("freight calculated: template_id=%d, freight=%s, to=%s-%s",
		template.GetId(), freight.String(), req.ToProvince, req.ToCity)

	return &consumerV1.CalculateFreightResponse{
		Freight:           freight.String(),
		IsFreeShipping:    false,
		CalculationDetail: trans.Ptr(calculationDetail),
	}, nil
}

// CreateFreightTemplate 创建运费模板
func (s *FreightService) CreateFreightTemplate(ctx context.Context, req *consumerV1.CreateFreightTemplateRequest) (*consumerV1.FreightTemplate, error) {
	// 1. 验证输入
	if req == nil || req.Name == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取租户ID
	tenantID := middleware.GetTenantID(ctx)

	// 3. 构建运费模板
	template := &consumerV1.FreightTemplate{
		TenantId:          trans.Ptr(tenantID),
		Name:              trans.Ptr(req.Name),
		CalculationType:   &req.CalculationType,
		FirstWeight:       req.FirstWeight,
		FirstPrice:        req.FirstPrice,
		AdditionalWeight:  req.AdditionalWeight,
		AdditionalPrice:   req.AdditionalPrice,
		RegionRules:       req.RegionRules,
		FreeShippingRules: req.FreeShippingRules,
		IsActive:          trans.Ptr(true),
	}

	// 4. 创建运费模板
	created, err := s.freightTemplateRepo.Create(ctx, template)
	if err != nil {
		s.log.Errorf("create freight template failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to create freight template")
	}

	s.log.Infof("freight template created: id=%d, name=%s", created.GetId(), created.GetName())

	return created, nil
}

// UpdateFreightTemplate 更新运费模板
func (s *FreightService) UpdateFreightTemplate(ctx context.Context, req *consumerV1.UpdateFreightTemplateRequest) (*emptypb.Empty, error) {
	// 1. 验证输入
	if req == nil || req.Id == 0 {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 检查模板是否存在
	existing, err := s.freightTemplateRepo.Get(ctx, req.Id)
	if err != nil {
		return nil, consumerV1.ErrorNotFound("freight template not found")
	}

	// 3. 构建更新数据
	template := &consumerV1.FreightTemplate{
		Name:              req.Name,
		CalculationType:   req.CalculationType,
		FirstWeight:       req.FirstWeight,
		FirstPrice:        req.FirstPrice,
		AdditionalWeight:  req.AdditionalWeight,
		AdditionalPrice:   req.AdditionalPrice,
		RegionRules:       req.RegionRules,
		FreeShippingRules: req.FreeShippingRules,
		IsActive:          req.IsActive,
	}

	// 4. 更新运费模板
	if err := s.freightTemplateRepo.Update(ctx, req.Id, template); err != nil {
		s.log.Errorf("update freight template failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to update freight template")
	}

	s.log.Infof("freight template updated: id=%d, name=%s", req.Id, existing.GetName())

	return &emptypb.Empty{}, nil
}

// GetFreightTemplate 查询运费模板
func (s *FreightService) GetFreightTemplate(ctx context.Context, req *consumerV1.GetFreightTemplateRequest) (*consumerV1.FreightTemplate, error) {
	// 1. 验证输入
	if req == nil || req.Id == 0 {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 查询运费模板
	template, err := s.freightTemplateRepo.Get(ctx, req.Id)
	if err != nil {
		return nil, consumerV1.ErrorNotFound("freight template not found")
	}

	return template, nil
}

// ListFreightTemplates 查询运费模板列表
func (s *FreightService) ListFreightTemplates(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListFreightTemplatesResponse, error) {
	// 1. 验证输入
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 查询运费模板列表
	resp, err := s.freightTemplateRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ========== 私有辅助方法 ==========

// checkFreeShipping 检查包邮规则
func (s *FreightService) checkFreeShipping(req *consumerV1.CalculateFreightRequest, template *consumerV1.FreightTemplate) (bool, string) {
	if len(template.FreeShippingRules) == 0 {
		return false, ""
	}

	for _, rule := range template.FreeShippingRules {
		ruleMap := rule.AsMap()

		// 检查满额包邮
		if minAmount, ok := ruleMap["min_amount"].(string); ok && minAmount != "" {
			if req.OrderAmount != nil && *req.OrderAmount != "" {
				orderAmount, err := decimal.NewFromString(*req.OrderAmount)
				if err == nil {
					minAmountDec, err := decimal.NewFromString(minAmount)
					if err == nil && orderAmount.GreaterThanOrEqual(minAmountDec) {
						return true, fmt.Sprintf("订单金额满%s元包邮", minAmount)
					}
				}
			}
		}

		// 检查地区包邮
		if provinces, ok := ruleMap["provinces"].([]interface{}); ok && len(provinces) > 0 {
			for _, p := range provinces {
				if province, ok := p.(string); ok && province == req.ToProvince {
					return true, fmt.Sprintf("%s地区包邮", req.ToProvince)
				}
			}
		}
	}

	return false, ""
}

// calculateByWeight 按重量计算运费
func (s *FreightService) calculateByWeight(req *consumerV1.CalculateFreightRequest, template *consumerV1.FreightTemplate) (decimal.Decimal, string, error) {
	// 1. 验证必要参数
	if req.Weight == nil || *req.Weight == "" {
		return decimal.Zero, "", fmt.Errorf("weight is required for BY_WEIGHT calculation")
	}

	if template.FirstWeight == nil || template.FirstPrice == nil {
		return decimal.Zero, "", fmt.Errorf("template missing first_weight or first_price")
	}

	// 2. 解析重量
	weight, err := decimal.NewFromString(*req.Weight)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("invalid weight format")
	}

	// 3. 解析首重和首重价格
	firstWeight, err := decimal.NewFromString(*template.FirstWeight)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("invalid first_weight format")
	}

	firstPrice, err := decimal.NewFromString(*template.FirstPrice)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("invalid first_price format")
	}

	// 4. 如果重量小于等于首重，直接返回首重价格
	if weight.LessThanOrEqual(firstWeight) {
		detail := fmt.Sprintf("重量%.2fkg ≤ 首重%.2fkg，运费=%.2f元",
			weight.InexactFloat64(), firstWeight.InexactFloat64(), firstPrice.InexactFloat64())
		return firstPrice, detail, nil
	}

	// 5. 计算续重部分
	if template.AdditionalWeight == nil || template.AdditionalPrice == nil {
		// 如果没有续重配置，只收首重价格
		detail := fmt.Sprintf("重量%.2fkg > 首重%.2fkg，但无续重配置，运费=%.2f元",
			weight.InexactFloat64(), firstWeight.InexactFloat64(), firstPrice.InexactFloat64())
		return firstPrice, detail, nil
	}

	additionalWeight, err := decimal.NewFromString(*template.AdditionalWeight)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("invalid additional_weight format")
	}

	additionalPrice, err := decimal.NewFromString(*template.AdditionalPrice)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("invalid additional_price format")
	}

	// 6. 计算超出首重的重量
	excessWeight := weight.Sub(firstWeight)

	// 7. 计算需要多少个续重单位（向上取整）
	additionalUnits := excessWeight.Div(additionalWeight).Ceil()

	// 8. 计算总运费 = 首重价格 + 续重单位数 * 续重价格
	totalFreight := firstPrice.Add(additionalUnits.Mul(additionalPrice))

	detail := fmt.Sprintf("重量%.2fkg，首重%.2fkg=%.2f元，续重%.2fkg=%.2f元，共%.0f个续重单位，运费=%.2f+%.0f*%.2f=%.2f元",
		weight.InexactFloat64(),
		firstWeight.InexactFloat64(), firstPrice.InexactFloat64(),
		additionalWeight.InexactFloat64(), additionalPrice.InexactFloat64(),
		additionalUnits.InexactFloat64(),
		firstPrice.InexactFloat64(), additionalUnits.InexactFloat64(), additionalPrice.InexactFloat64(),
		totalFreight.InexactFloat64())

	return totalFreight, detail, nil
}

// calculateByDistance 按距离计算运费
func (s *FreightService) calculateByDistance(req *consumerV1.CalculateFreightRequest, template *consumerV1.FreightTemplate) (decimal.Decimal, string, error) {
	// 1. 验证必要参数
	if req.ToProvince == "" || req.ToCity == "" {
		return decimal.Zero, "", fmt.Errorf("to_province and to_city are required for BY_DISTANCE calculation")
	}

	// 2. 查找地区规则
	if len(template.RegionRules) == 0 {
		return decimal.Zero, "", fmt.Errorf("template missing region_rules")
	}

	// 3. 匹配地区规则
	for _, rule := range template.RegionRules {
		ruleMap := rule.AsMap()

		// 检查省份匹配
		if provinces, ok := ruleMap["provinces"].([]interface{}); ok {
			for _, p := range provinces {
				if province, ok := p.(string); ok && province == req.ToProvince {
					// 找到匹配的地区规则
					if priceStr, ok := ruleMap["price"].(string); ok && priceStr != "" {
						price, err := decimal.NewFromString(priceStr)
						if err != nil {
							return decimal.Zero, "", fmt.Errorf("invalid price format in region rule")
						}

						detail := fmt.Sprintf("目的地%s-%s，地区运费=%.2f元",
							req.ToProvince, req.ToCity, price.InexactFloat64())
						return price, detail, nil
					}
				}
			}
		}
	}

	// 4. 如果没有匹配的地区规则，使用默认价格
	if template.FirstPrice != nil && *template.FirstPrice != "" {
		price, err := decimal.NewFromString(*template.FirstPrice)
		if err != nil {
			return decimal.Zero, "", fmt.Errorf("invalid first_price format")
		}

		detail := fmt.Sprintf("目的地%s-%s，使用默认运费=%.2f元",
			req.ToProvince, req.ToCity, price.InexactFloat64())
		return price, detail, nil
	}

	return decimal.Zero, "", fmt.Errorf("no matching region rule and no default price")
}
