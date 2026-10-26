package model

import "time"

// UsageLog records AI model usage
// UsageType: 'chat' | 'drawing'
// Source identifies the feature that initiated the request.
type UsageLog struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	UserID             uint       `gorm:"not null" json:"user_id"`
	UserName           string     `gorm:"-" json:"user_name"`
	UsageType          string     `gorm:"type:varchar(20);not null" json:"usage_type"`
	Source             string     `gorm:"type:varchar(50);default:''" json:"source"`
	Model              string     `gorm:"type:varchar(100);not null" json:"model"`
	Prompt             string     `gorm:"type:text" json:"prompt"`
	Result             string     `gorm:"type:text" json:"result"`
	ImageURL           string     `gorm:"type:varchar(500)" json:"image_url"`
	ReferenceImageURL  string     `gorm:"type:varchar(1000);default:''" json:"reference_image_url"`
	ReferenceImageURLs string     `gorm:"type:text;not null;default:'[]'" json:"reference_image_urls"`
	RequestBody        string     `gorm:"type:text" json:"request_body"`
	ResponseBody       string     `gorm:"type:text" json:"response_body"`
	PromptTokens       int        `gorm:"default:0" json:"prompt_tokens"`
	CompletionTokens   int        `gorm:"default:0" json:"completion_tokens"`
	TotalTokens        int        `gorm:"default:0" json:"total_tokens"`
	Resolution         string     `gorm:"type:varchar(20);default:''" json:"resolution"`
	Duration           int        `gorm:"default:0" json:"duration"` // 耗时，单位毫秒
	Status             string     `gorm:"type:varchar(20);default:'success'" json:"status"`
	DownloadCount      int        `gorm:"not null;default:0" json:"download_count"`
	IsSavedToDesign    bool       `gorm:"not null;default:false" json:"is_saved_to_design"`
	SavedAt            *time.Time `json:"saved_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (UsageLog) TableName() string {
	return "usage_logs"
}
