package model

import "time"

// ImageGenerationTask 图像生成异步任务
type ImageGenerationTask struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	TaskID        string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	BatchKey      string    `gorm:"type:varchar(64);index;default:''" json:"batch_key"`
	BatchItemID   uint      `gorm:"index;default:0" json:"batch_item_id"`
	ImageCount    int       `gorm:"default:1" json:"image_count"`
	Status        string    `gorm:"type:varchar(20);default:'queued'" json:"status"` // queued, running, done, failed
	ErrorMessage  string    `gorm:"type:text;default:''" json:"error_message"`
	Prompt        string    `gorm:"type:text" json:"prompt"`
	AspectRatio   string    `gorm:"type:varchar(20);default:'1:1'" json:"aspect_ratio"`
	Resolution    string    `gorm:"type:varchar(20);default:''" json:"resolution"`
	Model         string    `gorm:"type:varchar(100);default:''" json:"model"`
	RefImagesData string    `gorm:"type:text;default:''" json:"-"` // Deprecated: do not store base64 reference images.
	ImageURL      string    `gorm:"type:varchar(500);default:''" json:"image_url"`
	ImageURLsData string    `gorm:"type:text;default:''" json:"-"`
	MimeType      string    `gorm:"type:varchar(50);default:'image/png'" json:"mime_type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ImageGenerationTask) TableName() string {
	return "image_generation_tasks"
}
