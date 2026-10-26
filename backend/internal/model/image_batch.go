package model

import "time"

// ImageBatch AI图像生成批次
type ImageBatch struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	UserID          uint             `gorm:"index;not null" json:"user_id"`
	BatchKey        string           `gorm:"type:varchar(64);uniqueIndex;not null" json:"batch_key"`
	Title           string           `gorm:"type:text;default:''" json:"title"`
	Subtitle        string           `gorm:"type:text;default:''" json:"subtitle"`
	Cover           string           `gorm:"type:text;default:''" json:"cover"`
	ReferenceImages StringSlice      `gorm:"type:jsonb;serializer:json;default:'[]'" json:"reference_images"`
	TimeText        string           `gorm:"type:varchar(50);default:''" json:"time_text"`
	Status          string           `gorm:"type:varchar(20);default:'running'" json:"status"` // running, done, error
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Items           []ImageBatchItem `gorm:"foreignKey:BatchID" json:"items"`
}

func (ImageBatch) TableName() string {
	return "image_batches"
}

// ImageBatchItem AI图像生成批次中的单个任务
type ImageBatchItem struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	BatchID       uint      `gorm:"index;not null" json:"batch_id"`
	AsyncTaskID   string    `gorm:"type:varchar(64);index;default:''" json:"async_task_id"`
	Prompt        string    `gorm:"type:text" json:"prompt"`
	ImageURL      string    `gorm:"type:varchar(500);default:''" json:"image_url"`
	MimeType      string    `gorm:"type:varchar(50);default:'image/png'" json:"mime_type"`
	Status        string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, generating, done, error
	Error         string    `gorm:"type:text;default:''" json:"error"`
	SavedToDesign bool      `gorm:"default:false" json:"saved_to_design"`
	ImageID       *uint     `gorm:"default:null" json:"image_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ImageBatchItem) TableName() string {
	return "image_batch_items"
}
