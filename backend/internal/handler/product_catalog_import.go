package handler

import (
	"fmt"
	"net/http"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxProductCatalogImportRows = 5000

type productCatalogImportRequest struct {
	Rows []productCatalogImportRow `json:"rows"`
}

type productCatalogImportRow struct {
	ImageURL      string `json:"image_url"`
	RowNumber     int    `json:"row_number"`
	SPU           string `json:"spu"`
	SKU           string `json:"sku"`
	ASIN          string `json:"asin"`
	SPUName       string `json:"spu_name"`
	Brand         string `json:"brand"`
	Model         string `json:"model"`
	SPUMaterial   string `json:"spu_material"`
	SPUCategory1  string `json:"spu_category1"`
	SPUCategory2  string `json:"spu_category2"`
	SKUName       string `json:"sku_name"`
	StyleName     string `json:"style_name"`
	Color         string `json:"color"`
	Size          string `json:"size"`
	Feature       string `json:"feature"`
	PackageQty    string `json:"package_qty"`
	Material      string `json:"material"`
	Category1     string `json:"category1"`
	Category2     string `json:"category2"`
	Category3     string `json:"category3"`
	ListedAt      string `json:"listed_at"`
	PackageLength string `json:"package_length"`
	PackageWidth  string `json:"package_width"`
	PackageHeight string `json:"package_height"`
	PackageWeight string `json:"package_weight"`
	PurchasePrice string `json:"purchase_price"`
}

func ImportProductCatalog(c *gin.Context) {
	var request productCatalogImportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导入数据格式错误"})
		return
	}
	if len(request.Rows) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导入文件中没有产品数据"})
		return
	}
	if len(request.Rows) > maxProductCatalogImportRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("单次最多导入 %d 行", maxProductCatalogImportRows)})
		return
	}

	seenSKUs := make(map[string]int, len(request.Rows))
	for index := range request.Rows {
		row := &request.Rows[index]
		trimProductCatalogImportRow(row)
		if row.RowNumber <= 0 {
			row.RowNumber = index + 2
		}
		if row.SPU == "" || row.SKU == "" || row.ASIN == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("第 %d 行的 SPU、SKU、ASIN 均为必填项", row.RowNumber)})
			return
		}
		if firstRow, exists := seenSKUs[row.SKU]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("第 %d 行与第 %d 行的 SKU 重复：%s", row.RowNumber, firstRow, row.SKU)})
			return
		}
		seenSKUs[row.SKU] = row.RowNumber
	}

	createdSPUs, createdSKUs, updatedSKUs := 0, 0, 0
	userID := currentUserID(c)
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		spus := make(map[string]model.ProductSPU)
		for _, row := range request.Rows {
			spu, exists := spus[row.SPU]
			if !exists {
				result := tx.Where("spu = ?", row.SPU).First(&spu)
				if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
					return result.Error
				}
				if result.Error == gorm.ErrRecordNotFound {
					spu = model.ProductSPU{SPU: row.SPU, UpdatedBy: userID}
					createdSPUs++
				}
				applyImportedSPUFields(&spu, row)
				spu.UpdatedBy = userID
				if spu.ID == 0 {
					if err := tx.Create(&spu).Error; err != nil {
						return err
					}
				} else if err := tx.Save(&spu).Error; err != nil {
					return err
				}
				spus[row.SPU] = spu
			}

			var sku model.ProductSKU
			result := tx.Where("sku = ?", row.SKU).First(&sku)
			if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
				return result.Error
			}
			if result.Error == gorm.ErrRecordNotFound {
				sku.SKU = row.SKU
				createdSKUs++
			} else {
				updatedSKUs++
			}
			applyImportedSKUFields(&sku, spu, row, userID)
			if sku.ID == 0 {
				if err := tx.Create(&sku).Error; err != nil {
					return err
				}
			} else if err := tx.Save(&sku).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "产品导入失败：" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"created_spus": createdSPUs,
		"created_skus": createdSKUs,
		"updated_skus": updatedSKUs,
		"total":        len(request.Rows),
	})
}

func trimProductCatalogImportRow(row *productCatalogImportRow) {
	row.ImageURL = strings.TrimSpace(row.ImageURL)
	row.SPU = strings.TrimSpace(row.SPU)
	row.SKU = strings.TrimSpace(row.SKU)
	row.ASIN = strings.TrimSpace(row.ASIN)
	row.SPUName = strings.TrimSpace(row.SPUName)
	row.Brand = strings.TrimSpace(row.Brand)
	row.Model = strings.TrimSpace(row.Model)
	row.SPUMaterial = strings.TrimSpace(row.SPUMaterial)
	row.SPUCategory1 = strings.TrimSpace(row.SPUCategory1)
	row.SPUCategory2 = strings.TrimSpace(row.SPUCategory2)
	row.SKUName = strings.TrimSpace(row.SKUName)
	row.StyleName = strings.TrimSpace(row.StyleName)
	row.Color = strings.TrimSpace(row.Color)
	row.Size = strings.TrimSpace(row.Size)
	row.Feature = strings.TrimSpace(row.Feature)
	row.PackageQty = strings.TrimSpace(row.PackageQty)
	row.Material = strings.TrimSpace(row.Material)
	row.Category1 = strings.TrimSpace(row.Category1)
	row.Category2 = strings.TrimSpace(row.Category2)
	row.Category3 = strings.TrimSpace(row.Category3)
	row.ListedAt = strings.TrimSpace(row.ListedAt)
	row.PackageLength = strings.TrimSpace(row.PackageLength)
	row.PackageWidth = strings.TrimSpace(row.PackageWidth)
	row.PackageHeight = strings.TrimSpace(row.PackageHeight)
	row.PackageWeight = strings.TrimSpace(row.PackageWeight)
	row.PurchasePrice = strings.TrimSpace(row.PurchasePrice)
}

func applyImportedSPUFields(spu *model.ProductSPU, row productCatalogImportRow) {
	spu.Name = row.SPUName
	spu.Brand = row.Brand
	spu.Model = row.Model
	spu.Material = row.SPUMaterial
	spu.Category1 = row.SPUCategory1
	spu.Category2 = row.SPUCategory2
}

func applyImportedSKUFields(sku *model.ProductSKU, spu model.ProductSPU, row productCatalogImportRow, userID uint) {
	// Blank images in older templates must not erase existing product photos.
	if row.ImageURL != "" {
		sku.ImageURL = row.ImageURL
	}
	sku.SPUID = spu.ID
	sku.SPU = spu.SPU
	sku.ASIN = row.ASIN
	sku.Name = row.SKUName
	sku.StyleName = row.StyleName
	sku.Color = row.Color
	sku.Size = row.Size
	sku.Feature = row.Feature
	sku.PackageQty = row.PackageQty
	sku.Material = row.Material
	sku.Category1 = row.Category1
	sku.Category2 = row.Category2
	sku.Category3 = row.Category3
	sku.ListedAt = row.ListedAt
	sku.PackageLength = row.PackageLength
	sku.PackageWidth = row.PackageWidth
	sku.PackageHeight = row.PackageHeight
	sku.PackageWeight = row.PackageWeight
	sku.PurchasePrice = row.PurchasePrice
	sku.UpdatedBy = userID
}
