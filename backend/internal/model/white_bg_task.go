package model

import "time"

// WhiteBgTask 白底图生成任务
type WhiteBgTask struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"index;not null" json:"user_id"`
	TaskID          string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Status          string     `gorm:"type:varchar(20);default:'queued'" json:"status"` // queued/running/done/failed
	ErrorMessage    string     `gorm:"type:text;default:''" json:"error_message"`
	Subject         string     `gorm:"type:varchar(255);default:''" json:"subject"`
	AspectRatio     string     `gorm:"type:varchar(20);default:'1:1'" json:"aspect_ratio"`
	Resolution      string     `gorm:"type:varchar(20);default:'2K'" json:"resolution"`
	PreviewImageURL string     `gorm:"type:varchar(500);default:''" json:"preview_image_url"`
	ResultImageURL  string     `gorm:"type:varchar(500);default:''" json:"result_image_url"`
	IsSavedToDesign bool       `gorm:"default:false" json:"is_saved_to_design"`
	SavedAt         *time.Time `json:"saved_at"`
	Model           string     `gorm:"type:varchar(100);default:''" json:"model"`
	Prompt          string     `gorm:"type:text;default:''" json:"prompt"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (WhiteBgTask) TableName() string {
	return "white_bg_tasks"
}
