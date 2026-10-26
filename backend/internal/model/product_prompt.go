package model

import "time"

// ReferenceImage 用户自定义参考图，每张图可以有多个标签
type ReferenceImage struct {
	ID          uint   `json:"id"`
	URL         string `json:"url"`          // 图片URL
	Description string `json:"description"` // 用户手动输入的描述，如"水龙头抽拉图"、"水柱模式流水图"
}

type ProductPrompt struct {
	ID                   uint             `gorm:"primaryKey" json:"id"`
	Name                 string           `gorm:"type:varchar(255);not null" json:"name"`
	Brand                string           `gorm:"type:varchar(120);default:''" json:"brand"`
	Category             string           `gorm:"type:varchar(120);default:''" json:"category"`
	SKU                  string           `gorm:"type:varchar(120);default:''" json:"sku"`
	Color                string           `gorm:"type:varchar(120);default:''" json:"color"`
	Material             string           `gorm:"type:varchar(255);default:''" json:"material"`
	ImageURL             string           `gorm:"type:varchar(500);default:''" json:"image_url"`
	// WhiteBgImageURL 产品白底图URL，用于AI生图参考
	WhiteBgImageURL      string           `gorm:"type:varchar(500);default:''" json:"white_bg_image_url"`
	SellingPoints        string           `gorm:"type:text;not null;default:''" json:"selling_points"`
	ProductParams        string           `gorm:"type:text;not null;default:''" json:"product_params"`
	UsageScenarios       string           `gorm:"type:text;not null;default:''" json:"usage_scenarios"`
	PackageContents      string           `gorm:"type:text;not null;default:''" json:"package_contents"`
	InstallationSteps    string           `gorm:"type:text;not null;default:''" json:"installation_steps"`
	InstallationNotes    string           `gorm:"type:text;not null;default:''" json:"installation_notes"`
	ForbiddenFabrication string           `gorm:"type:text;not null;default:''" json:"forbidden_fabrication"`
	// 自定义参考图：用户上传图片并输入描述（如"水龙头抽拉图"、"水柱模式流水图"）
	ReferenceImages      []ReferenceImage `gorm:"type:json;serializer:json;default:'[]'" json:"reference_images"`
	// StpFileURL 产品 STP 文件的 OSS URL
	StpFileURL string `gorm:"type:varchar(500);default:''" json:"stp_file_url"`
	// RenderStatus 渲染状态：pending/processing/completed/failed
	RenderStatus string `gorm:"type:varchar(20);default:'pending'" json:"render_status"`
	// RenderError 渲染错误信息
	RenderError string `gorm:"type:text;default:''" json:"render_error"`
	MainImagePrompt      string           `gorm:"type:text;not null;default:''" json:"main_image_prompt"`
	DetailPagePrompt     string           `gorm:"type:text;not null;default:''" json:"detail_page_prompt"`
	CustomPromptSuffix   string           `gorm:"type:text;not null;default:''" json:"custom_prompt_suffix"`
	Sort                 int              `gorm:"default:0" json:"sort"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

func (ProductPrompt) TableName() string {
	return "product_prompts"
}
