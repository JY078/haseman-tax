package model

import "time"

type MultiAngleTask struct {
	ID                uint               `gorm:"primaryKey" json:"id"`
	UserID            uint               `gorm:"index;not null" json:"user_id"`
	TaskID            string             `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Status            string             `gorm:"type:varchar(20);default:'queued'" json:"status"`
	ErrorMessage      string             `gorm:"type:text;default:''" json:"error_message"`
	Model             string             `gorm:"type:varchar(100);default:''" json:"model"`
	Quality           string             `gorm:"type:varchar(20);default:'2K'" json:"quality"`
	AspectRatio       string             `gorm:"type:varchar(20);default:'auto'" json:"aspect_ratio"`
	ViewKeys          string             `gorm:"type:text;default:''" json:"view_keys"`
	ReferenceImageURL string             `gorm:"type:varchar(1000);default:''" json:"reference_image_url"`
	MimeType          string             `gorm:"type:varchar(50);default:'image/png'" json:"mime_type"`
	FirstImageURL     string             `gorm:"type:varchar(1000);default:''" json:"first_image_url"`
	Results           []MultiAngleResult `gorm:"foreignKey:TaskIDRef;references:TaskID" json:"results,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

func (MultiAngleTask) TableName() string {
	return "multi_angle_tasks"
}

type MultiAngleResult struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TaskIDRef    string    `gorm:"type:varchar(64);index;not null" json:"task_id"`
	View         string    `gorm:"type:varchar(30);index;not null" json:"view"`
	Label        string    `gorm:"type:varchar(100);default:''" json:"label"`
	Status       string    `gorm:"type:varchar(20);default:'queued'" json:"status"`
	Prompt       string    `gorm:"type:text;default:''" json:"prompt"`
	ImageURL     string    `gorm:"type:varchar(1000);default:''" json:"image_url"`
	MimeType     string    `gorm:"type:varchar(50);default:'image/png'" json:"mime_type"`
	ErrorMessage string    `gorm:"type:text;default:''" json:"error_message"`
	UsageLogID   uint      `gorm:"index;default:0" json:"usage_log_id"`
	Sort         int       `gorm:"default:0" json:"sort"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (MultiAngleResult) TableName() string {
	return "multi_angle_results"
}
