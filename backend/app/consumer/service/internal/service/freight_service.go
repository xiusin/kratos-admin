package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/freighttemplate"
)

// FreightService 运费计算服务
type FreightService struct {
	consumerV1.UnimplementedFreightServiceServer

	freightTemplateRepo data.FreightTemplateRepo
	log                 *log.Helper
}

// NewFreightService 创建运费计算服务实例
func NewFreightService(
	ctx *bootstrap.Context,
	freightTemplateRepo data.FreightTemplateRepo,
) *FreightService {
	return &FreightService{
		freightTemplateRepo: freightTemplateRepo,
		log:                 ctx.NewLoggerHelper("consumer/service/freight-service"),
	}
}

// CalculateFreight 计算运费
func (s *FreightService) CalculateFreight(ctx context.Context, req *consumerV1.CalculateFreightRequest) (*consumerV1.CalculateFreightResponse, error) {
	s.log.Infof("CalculateFreight: template_id=%d, weight=%s, to=%s-%s",
		req.GetTemplateId(), req.GetWeight(), req.ToProvince, req.ToCity)

	// 1. 获取运费模板
	var template *ent.FreightTemplate
	var err error

	if req.GetTemplateId() > 0 {
		// 使用指定模板
		template, err = s.freightTemplateRepo.Get(ctx, req.GetTemplateId())
		if err != nil {
			return nil, errors.BadRequest("TEMPLATE_NOT_FOUND", "运费模板不存在")
		}
	} else {
		// 使用默认模板（查询第一个启用的模板）
		templates, _, err := s.freightTemplateRepo.List(ctx, 1, 1,
			freighttemplate.IsActive(true),
		)
		if err != nil || len(templates) == 0 {
			return nil, errors.BadRequest("NO_DEFAULT_TEMPLATE", "未找到默认运费模板")
		}
		template = templates[0]
	}

	// 2. 检查包邮规则
	isFreeShipping, reason := s.checkFreeShipping(template, req)
	if isFreeShipping {
		return &consumerV1.CalculateFreightResponse{
			Freight:            "0.00",
			IsFreeShipping:     true,
			FreeShippingReason: &reason,
		}, nil
	}

	// 3. 根据计算方式计算运费
	var freight decimal.Decimal
	var calculationDetail string

	switch template.CalculationType {
	case freighttemplate.CalculationTypeByWeight:
		freight, calculationDetail, err = s.calculateByWeight(template, req)
	case freighttemplate.CalculationTypeByDistance:
		freight, calculationDetail, err = s.calculateByDistance(template, req)
	default:
		return nil, errors.BadRequest("INVALID_CALCULATION_TYPE", "不支持的计算方式")
	}

	if err != nil {
		return nil, err
	}

	// 4. 返回结果
	return &consumerV1.CalculateFreightResponse{
		Freight:           freight.StringFixed(2),
		IsFreeShipping:    false,
		CalculationDetail: &calculationDetail,
	}, nil
}

// checkFreeShipping 检查是否包邮
func (s *FreightService) checkFreeShipping(template *ent.FreightTemplate, req *consumerV1.CalculateFreightRequest) (bool, string) {
	if template.FreeShippingRules == nil || len(template.FreeShippingRules) == 0 {
		return false, ""
	}

	// 检查满额包邮
	if req.GetOrderAmount() != "" {
		orderAmount, err := decimal.NewFromString(req.GetOrderAmount())
		if err == nil {
			for _, rule := range template.FreeShippingRules {
				if ruleType, ok := rule["type"].(string); ok && ruleType == "amount" {
					if minAmount, ok := rule["min_amount"].(string); ok {
						minAmountDec, err := decimal.NewFromString(minAmount)
						if err == nil && orderAmount.GreaterThanOrEqual(minAmountDec) {
							return true, fmt.Sprintf("订单满%s元包邮", minAmount)
						}
					}
				}
			}
		}
	}

	// 检查地区包邮
	for _, rule := range template.FreeShippingRules {
		if ruleType, ok := rule["type"].(string); ok && ruleType == "region" {
			if provinces, ok := rule["provinces"].([]interface{}); ok {
				for _, p := range provinces {
					if province, ok := p.(string); ok && province == req.ToProvince {
						return true, fmt.Sprintf("%s地区包邮", req.ToProvince)
					}
				}
			}
		}
	}

	return false, ""
}

// calculateByWeight 按重量计算运费
func (s *FreightService) calculateByWeight(template *ent.FreightTemplate, req *consumerV1.CalculateFreightRequest) (decimal.Decimal, string, error) {
	// 解析重量
	weight, err := decimal.NewFromString(req.GetWeight())
	if err != nil {
		return decimal.Zero, "", errors.BadRequest("INVALID_WEIGHT", "重量格式错误")
	}

	// 解析首重和首重价格
	firstWeight, err := decimal.NewFromString(*template.FirstWeight)
	if err != nil {
		return decimal.Zero, "", errors.InternalServer("INVALID_FIRST_WEIGHT", "首重配置错误")
	}

	firstPrice, err := decimal.NewFromString(*template.FirstPrice)
	if err != nil {
		return decimal.Zero, "", errors.InternalServer("INVALID_FIRST_PRICE", "首重价格配置错误")
	}

	// 如果重量小于等于首重，直接返回首重价格
	if weight.LessThanOrEqual(firstWeight) {
		detail := fmt.Sprintf("重量%.2fkg ≤ 首重%.2fkg，运费=%.2f元",
			weight, firstWeight, firstPrice)
		return firstPrice, detail, nil
	}

	// 计算续重
	additionalWeight, err := decimal.NewFromString(*template.AdditionalWeight)
	if err != nil {
		return decimal.Zero, "", errors.InternalServer("INVALID_ADDITIONAL_WEIGHT", "续重配置错误")
	}

	additionalPrice, err := decimal.NewFromString(*template.AdditionalPrice)
	if err != nil {
		return decimal.Zero, "", errors.InternalServer("INVALID_ADDITIONAL_PRICE", "续重价格配置错误")
	}

	// 计算超出首重的重量
	extraWeight := weight.Sub(firstWeight)

	// 计算续重份数（向上取整）
	additionalCount := extraWeight.Div(additionalWeight).Ceil()

	// 计算总运费 = 首重价格 + 续重份数 * 续重价格
	totalFreight := firstPrice.Add(additionalCount.Mul(additionalPrice))

	detail := fmt.Sprintf("首重%.2fkg=%.2f元，续重%.2fkg，每%.2fkg=%.2f元，共%.0f份，总运费=%.2f元",
		firstWeight, firstPrice, extraWeight, additionalWeight, additionalPrice, additionalCount, totalFreight)

	return totalFreight, detail, nil
}

// calculateByDistance 按距离计算运费
func (s *FreightService) calculateByDistance(template *ent.FreightTemplate, req *consumerV1.CalculateFreightRequest) (decimal.Decimal, string, error) {
	// 检查地区规则
	if template.RegionRules == nil || len(template.RegionRules) == 0 {
		return decimal.Zero, "", errors.InternalServer("NO_REGION_RULES", "未配置地区规则")
	}

	// 查找匹配的地区规则
	for _, rule := range template.RegionRules {
		if provinces, ok := rule["provinces"].([]interface{}); ok {
			for _, p := range provinces {
				if province, ok := p.(string); ok && province == req.ToProvince {
					// 找到匹配的地区规则
					priceStr, ok := rule["price"].(string)
					if !ok {
						continue
					}

					price, err := decimal.NewFromString(priceStr)
					if err != nil {
						continue
					}

					detail := fmt.Sprintf("目的地%s，运费=%.2f元", req.ToProvince, price)
					return price, detail, nil
				}
			}
		}
	}

	// 使用默认价格
	if template.FirstPrice != nil {
		price, err := decimal.NewFromString(*template.FirstPrice)
		if err != nil {
			return decimal.Zero, "", errors.InternalServer("INVALID_DEFAULT_PRICE", "默认价格配置错误")
		}

		detail := fmt.Sprintf("目的地%s，使用默认运费=%.2f元", req.ToProvince, price)
		return price, detail, nil
	}

	return decimal.Zero, "", errors.BadRequest("NO_MATCHING_REGION", "未找到匹配的地区规则")
}

// CreateFreightTemplate 创建运费模板
func (s *FreightService) CreateFreightTemplate(ctx context.Context, req *consumerV1.CreateFreightTemplateRequest) (*consumerV1.FreightTemplate, error) {
	s.log.Infof("CreateFreightTemplate: name=%s, calculation_type=%s", req.Name, req.CalculationType)

	// 构建 Ent 实体
	template := &ent.FreightTemplate{
		Name:            req.Name,
		CalculationType: s.convertCalculationType(req.CalculationType),
		IsActive:        true,
	}

	// 设置首重和首重价格
	if req.GetFirstWeight() != "" {
		template.FirstWeight = req.FirstWeight
	}
	if req.GetFirstPrice() != "" {
		template.FirstPrice = req.FirstPrice
	}

	// 设置续重和续重价格
	if req.GetAdditionalWeight() != "" {
		template.AdditionalWeight = req.AdditionalWeight
	}
	if req.GetAdditionalPrice() != "" {
		template.AdditionalPrice = req.AdditionalPrice
	}

	// 设置地区规则
	if len(req.RegionRules) > 0 {
		template.RegionRules = s.convertStructsToMaps(req.RegionRules)
	}

	// 设置包邮规则
	if len(req.FreeShippingRules) > 0 {
		template.FreeShippingRules = s.convertStructsToMaps(req.FreeShippingRules)
	}

	// 创建模板
	created, err := s.freightTemplateRepo.Create(ctx, template)
	if err != nil {
		return nil, errors.InternalServer("CREATE_FAILED", "创建运费模板失败")
	}

	return s.toProtoFreightTemplate(created), nil
}

// UpdateFreightTemplate 更新运费模板
func (s *FreightService) UpdateFreightTemplate(ctx context.Context, req *consumerV1.UpdateFreightTemplateRequest) (*emptypb.Empty, error) {
	s.log.Infof("UpdateFreightTemplate: id=%d", req.Id)

	// 构建更新实体
	template := &ent.FreightTemplate{}

	if req.GetName() != "" {
		template.Name = req.GetName()
	}

	if req.CalculationType != nil {
		template.CalculationType = s.convertCalculationType(*req.CalculationType)
	}

	if req.GetFirstWeight() != "" {
		template.FirstWeight = req.FirstWeight
	}
	if req.GetFirstPrice() != "" {
		template.FirstPrice = req.FirstPrice
	}

	if req.GetAdditionalWeight() != "" {
		template.AdditionalWeight = req.AdditionalWeight
	}
	if req.GetAdditionalPrice() != "" {
		template.AdditionalPrice = req.AdditionalPrice
	}

	if len(req.RegionRules) > 0 {
		template.RegionRules = s.convertStructsToMaps(req.RegionRules)
	}

	if len(req.FreeShippingRules) > 0 {
		template.FreeShippingRules = s.convertStructsToMaps(req.FreeShippingRules)
	}

	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	// 更新模板
	if err := s.freightTemplateRepo.Update(ctx, req.Id, template); err != nil {
		return nil, errors.InternalServer("UPDATE_FAILED", "更新运费模板失败")
	}

	return &emptypb.Empty{}, nil
}

// GetFreightTemplate 查询运费模板
func (s *FreightService) GetFreightTemplate(ctx context.Context, req *consumerV1.GetFreightTemplateRequest) (*consumerV1.FreightTemplate, error) {
	s.log.Infof("GetFreightTemplate: id=%d", req.Id)

	template, err := s.freightTemplateRepo.Get(ctx, req.Id)
	if err != nil {
		return nil, errors.BadRequest("TEMPLATE_NOT_FOUND", "运费模板不存在")
	}

	return s.toProtoFreightTemplate(template), nil
}

// ListFreightTemplates 查询运费模板列表
func (s *FreightService) ListFreightTemplates(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListFreightTemplatesResponse, error) {
	s.log.Infof("ListFreightTemplates: page=%d, pageSize=%d", req.GetPage(), req.GetPageSize())

	page := int(req.GetPage())
	if page <= 0 {
		page = 1
	}

	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}

	templates, total, err := s.freightTemplateRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, errors.InternalServer("LIST_FAILED", "查询运费模板列表失败")
	}

	items := make([]*consumerV1.FreightTemplate, 0, len(templates))
	for _, template := range templates {
		items = append(items, s.toProtoFreightTemplate(template))
	}

	return &consumerV1.ListFreightTemplatesResponse{
		Items: items,
		Total: uint64(total),
	}, nil
}

// convertCalculationType 转换计算方式
func (s *FreightService) convertCalculationType(pbType consumerV1.FreightTemplate_CalculationType) freighttemplate.CalculationType {
	switch pbType {
	case consumerV1.FreightTemplate_BY_WEIGHT:
		return freighttemplate.CalculationTypeByWeight
	case consumerV1.FreightTemplate_BY_DISTANCE:
		return freighttemplate.CalculationTypeByDistance
	default:
		return freighttemplate.CalculationTypeByWeight
	}
}

// convertStructsToMaps 转换 Struct 列表为 map 列表
func (s *FreightService) convertStructsToMaps(structs []*structpb.Struct) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(structs))
	for _, st := range structs {
		result = append(result, st.AsMap())
	}
	return result
}

// toProtoFreightTemplate 转换为 Proto FreightTemplate
func (s *FreightService) toProtoFreightTemplate(template *ent.FreightTemplate) *consumerV1.FreightTemplate {
	result := &consumerV1.FreightTemplate{
		Id:   func() *uint32 { v := template.ID; return &v }(),
		Name: &template.Name,
		CalculationType: func() *consumerV1.FreightTemplate_CalculationType {
			var ct consumerV1.FreightTemplate_CalculationType
			switch template.CalculationType {
			case freighttemplate.CalculationTypeByWeight:
				ct = consumerV1.FreightTemplate_BY_WEIGHT
			case freighttemplate.CalculationTypeByDistance:
				ct = consumerV1.FreightTemplate_BY_DISTANCE
			}
			return &ct
		}(),
		IsActive: &template.IsActive,
	}

	// 设置租户ID
	if template.TenantID != nil {
		result.TenantId = template.TenantID
	}

	// 设置首重和首重价格
	if template.FirstWeight != nil {
		result.FirstWeight = template.FirstWeight
	}
	if template.FirstPrice != nil {
		result.FirstPrice = template.FirstPrice
	}

	// 设置续重和续重价格
	if template.AdditionalWeight != nil {
		result.AdditionalWeight = template.AdditionalWeight
	}
	if template.AdditionalPrice != nil {
		result.AdditionalPrice = template.AdditionalPrice
	}

	// 设置地区规则
	if template.RegionRules != nil {
		regionRules := make([]*structpb.Struct, 0, len(template.RegionRules))
		for _, rule := range template.RegionRules {
			if st, err := structpb.NewStruct(rule); err == nil {
				regionRules = append(regionRules, st)
			}
		}
		result.RegionRules = regionRules
	}

	// 设置包邮规则
	if template.FreeShippingRules != nil {
		freeShippingRules := make([]*structpb.Struct, 0, len(template.FreeShippingRules))
		for _, rule := range template.FreeShippingRules {
			if st, err := structpb.NewStruct(rule); err == nil {
				freeShippingRules = append(freeShippingRules, st)
			}
		}
		result.FreeShippingRules = freeShippingRules
	}

	// 设置创建者和更新者
	if template.CreatedBy != nil {
		result.CreatedBy = template.CreatedBy
	}
	if template.UpdatedBy != nil {
		result.UpdatedBy = template.UpdatedBy
	}

	// 设置时间戳（铁律12：检查指针）
	if template.CreatedAt != nil {
		result.CreatedAt = timestamppb.New(*template.CreatedAt)
	}
	if template.UpdatedAt != nil {
		result.UpdatedAt = timestamppb.New(*template.UpdatedAt)
	}

	return result
}
