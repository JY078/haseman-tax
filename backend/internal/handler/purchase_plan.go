package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type purchasePlanInput struct {
	ProductSKUID        uint   `json:"product_sku_id"`
	PlannedQuantity     int    `json:"planned_quantity"`
	UnitsPerCarton      int    `json:"units_per_carton"`
	CartonCount         int    `json:"carton_count"`
	ExpectedArrivalDate string `json:"expected_arrival_date"`
	Remark              string `json:"remark"`
}

type normalizedPurchasePlanInput struct {
	ProductSKUID        uint
	PlannedQuantity     int
	UnitsPerCarton      int
	CartonCount         int
	ExpectedArrivalDate time.Time
	Remark              string
}

type purchasePlanProduct struct {
	ID          uint   `json:"id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
}

func CreatePurchasePlan(c *gin.Context) {
	applicantID := currentUserID(c)
	if applicantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	var input purchasePlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请完整填写采购计划"})
		return
	}
	normalized, err := validatePurchasePlanInput(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var sku model.ProductSKU
	if err := database.DB.First(&sku, normalized.ProductSKUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "所选产品不存在，请重新选择"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取产品信息失败"})
		return
	}

	var spu model.ProductSPU
	if sku.SPUID > 0 {
		_ = database.DB.First(&spu, sku.SPUID).Error
	}
	plan := model.PurchasePlan{
		ProductSKUID:        sku.ID,
		ProductName:         purchasePlanProductName(sku, spu),
		SKU:                 sku.SKU,
		PlannedQuantity:     normalized.PlannedQuantity,
		UnitsPerCarton:      normalized.UnitsPerCarton,
		CartonCount:         normalized.CartonCount,
		ExpectedArrivalDate: normalized.ExpectedArrivalDate,
		Remark:              normalized.Remark,
		ApplicantID:         applicantID,
	}
	if err := database.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交采购计划失败"})
		return
	}
	hydratePurchasePlanApplicant(&plan)
	c.JSON(http.StatusCreated, plan)
}

func ListPurchasePlans(c *gin.Context) {
	page := positiveQueryInt(c, "page", 1)
	pageSize := positiveQueryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}

	query := database.DB.Model(&model.PurchasePlan{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取采购计划失败"})
		return
	}
	var plans []model.PurchasePlan
	if err := query.Preload("Applicant").Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取采购计划失败"})
		return
	}
	for i := range plans {
		setPurchasePlanApplicantName(&plans[i])
	}
	c.JSON(http.StatusOK, gin.H{"data": plans, "total": total, "page": page, "page_size": pageSize})
}

func SearchPurchasePlanProducts(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	query := database.DB.Table("product_skus AS sku").
		Select(`sku.id,
			COALESCE(NULLIF(spu.name, ''), NULLIF(sku.name, ''), NULLIF(sku.style_name, ''), sku.sku) AS product_name,
			sku.sku`).
		Joins("LEFT JOIN product_spus AS spu ON spu.id = sku.spu_id OR spu.spu = sku.spu").
		Where("spu.id IS NOT NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("sku.sku ILIKE ? OR sku.name ILIKE ? OR sku.style_name ILIKE ? OR spu.name ILIKE ?", like, like, like, like)
	}
	var products []purchasePlanProduct
	if err := query.Order("product_name ASC, sku.sku ASC").Limit(30).Scan(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索产品失败"})
		return
	}
	c.JSON(http.StatusOK, products)
}

func validatePurchasePlanInput(input purchasePlanInput) (normalizedPurchasePlanInput, error) {
	if input.ProductSKUID == 0 {
		return normalizedPurchasePlanInput{}, fmt.Errorf("请选择产品")
	}
	if input.PlannedQuantity <= 0 {
		return normalizedPurchasePlanInput{}, fmt.Errorf("计划采购量必须为正整数")
	}
	if input.UnitsPerCarton <= 0 {
		return normalizedPurchasePlanInput{}, fmt.Errorf("单箱数量必须为正整数")
	}
	if input.CartonCount <= 0 {
		return normalizedPurchasePlanInput{}, fmt.Errorf("箱数必须为正整数")
	}
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(input.ExpectedArrivalDate), time.Local)
	if err != nil {
		return normalizedPurchasePlanInput{}, fmt.Errorf("请选择有效的期望到货时间")
	}
	return normalizedPurchasePlanInput{
		ProductSKUID:        input.ProductSKUID,
		PlannedQuantity:     input.PlannedQuantity,
		UnitsPerCarton:      input.UnitsPerCarton,
		CartonCount:         input.CartonCount,
		ExpectedArrivalDate: date,
		Remark:              strings.TrimSpace(input.Remark),
	}, nil
}

func purchasePlanProductName(sku model.ProductSKU, spu model.ProductSPU) string {
	for _, value := range []string{spu.Name, sku.Name, sku.StyleName, sku.SKU} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "未命名产品"
}

func hydratePurchasePlanApplicant(plan *model.PurchasePlan) {
	_ = database.DB.Preload("Applicant").First(plan, plan.ID).Error
	setPurchasePlanApplicantName(plan)
}

func setPurchasePlanApplicantName(plan *model.PurchasePlan) {
	if plan.Applicant == nil {
		plan.ApplicantName = fmt.Sprintf("用户 #%d", plan.ApplicantID)
		return
	}
	plan.ApplicantName = strings.TrimSpace(plan.Applicant.Name)
	if plan.ApplicantName == "" {
		plan.ApplicantName = plan.Applicant.Username
	}
}
