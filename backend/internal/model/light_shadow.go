package model

import "time"

type LightShadowTask struct {
	ID           uint               `gorm:"primaryKey" json:"id"`
	UserID       uint               `gorm:"index;not null" json:"user_id"`
	TaskID       string             `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Status       string             `gorm:"type:varchar(20);default:'queued'" json:"status"`
	ErrorMessage string             `gorm:"type:text;default:''" json:"error_message"`
	AspectRatio  string             `gorm:"type:varchar(20);default:'1:1'" json:"aspect_ratio"`
	Quality      string             `gorm:"type:varchar(20);default:'2K'" json:"quality"`
	ImageCount   int                `gorm:"default:1" json:"image_count"`
	Images       []LightShadowImage `gorm:"foreignKey:TaskIDRef;references:TaskID" json:"images,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

func (LightShadowTask) TableName() string {
	return "light_shadow_tasks"
}

type LightShadowImage struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	TaskIDRef         string    `gorm:"type:varchar(64);index;not null" json:"task_id"`
	Sort              int       `gorm:"default:0" json:"sort"`
	ReferenceImageURL string    `gorm:"type:varchar(1000);default:''" json:"reference_image_url"`
	ImageURL          string    `gorm:"type:varchar(1000);default:''" json:"image_url"`
	ErrorMsg          string    `gorm:"type:text;default:''" json:"error_message"`
	UsageLogID        uint      `gorm:"default:0" json:"usage_log_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (LightShadowImage) TableName() string {
	return "light_shadow_images"
}
