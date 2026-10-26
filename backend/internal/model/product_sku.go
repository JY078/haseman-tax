package model

import "time"

type ProductSKU struct {
	ImageURL        string    `gorm:"type:text;not null;default:''" json:"image_url"`
	AmazonPrice     *string    `gorm:"type:numeric(10,2)" json:"amazon_price"`
	AmazonSalePrice *string    `gorm:"type:numeric(10,2)" json:"amazon_sale_price"`
	AmazonBestDeal  *string    `gorm:"type:numeric(10,2)" json:"amazon_best_deal"`
	PriceUpdatedAt  *time.Time `json:"price_updated_at"`

	ID            uint      `gorm:"primaryKey" json:"id"`
	SPUID         uint      `gorm:"column:spu_id;index;not null;default:0" json:"spu_id"`
	SKU           string    `gorm:"type:varchar(120);not null;uniqueIndex" json:"sku"`
	ASIN          string    `gorm:"type:varchar(20);not null;default:'';index" json:"asin"`
	Name          string    `gorm:"type:varchar(255);not null;default:''" json:"name"`
	SPU           string    `gorm:"type:varchar(120);not null;default:''" json:"spu"`
	StyleName     string    `gorm:"type:varchar(255);not null;default:''" json:"style_name"`
	Color         string    `gorm:"type:varchar(120);not null;default:''" json:"color"`
	Size          string    `gorm:"type:varchar(120);not null;default:''" json:"size"`
	Feature       string    `gorm:"type:varchar(255);not null;default:''" json:"feature"`
	PackageQty    string    `gorm:"type:varchar(120);not null;default:''" json:"package_qty"`
	Material      string    `gorm:"type:varchar(255);not null;default:''" json:"material"`
	Category1     string    `gorm:"type:varchar(255);not null;default:''" json:"category1"`
	Category2     string    `gorm:"type:varchar(255);not null;default:''" json:"category2"`
	Category3     string    `gorm:"type:varchar(255);not null;default:''" json:"category3"`
	ListedAt      string    `gorm:"type:varchar(32);not null;default:''" json:"listed_at"`
	PackageLength string    `gorm:"type:varchar(64);not null;default:''" json:"package_length"`
	PackageWidth  string    `gorm:"type:varchar(64);not null;default:''" json:"package_width"`
	PackageHeight string    `gorm:"type:varchar(64);not null;default:''" json:"package_height"`
	PackageWeight string    `gorm:"type:varchar(64);not null;default:''" json:"package_weight"`
	PurchasePrice string    `gorm:"type:varchar(64);not null;default:''" json:"purchase_price"`
	StpModelURL   string    `gorm:"type:varchar(500);not null;default:''" json:"stp_model_url"`
	AuthDocURL    string    `gorm:"type:varchar(500);not null;default:''" json:"auth_doc_url"`
	UpdatedBy     uint      `gorm:"not null;default:0" json:"updated_by"`
	PlatformSKUs  map[string][]string `gorm:"-" json:"platform_skus"`
	UpdatedByName string    `gorm:"-" json:"updated_by_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ProductSKU) TableName() string {
	return "product_skus"
}
