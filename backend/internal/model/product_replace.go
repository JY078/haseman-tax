package model

import "time"

type ProductReplaceTask struct {
	ID           uint                   `gorm:"primaryKey" json:"id"`
	UserID       uint                   `gorm:"index;not null" json:"user_id"`
	TaskID       string                 `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Status       string                 `gorm:"type:varchar(20);default:'queued'" json:"status"`
	ErrorMessage string                 `gorm:"type:text;default:''" json:"error_message"`
	IsSavedToDesign bool                `gorm:"default:false" json:"is_saved_to_design"`
	SavedAt      *time.Time            `json:"saved_at"`
	Mode         string                 `gorm:"type:varchar(20);default:'water'" json:"mode"`
	AspectRatio  string                 `gorm:"type:varchar(20);default:'1:1'" json:"aspect_ratio"`
	ImageCount   int                    `gorm:"default:1" json:"image_count"`
	Images       []ProductReplaceImage  `gorm:"foreignKey:TaskIDRef;references:TaskID" json:"images,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

func (ProductReplaceTask) TableName() string {
	return "product_replace_tasks"
}

type ProductReplaceImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskIDRef string    `gorm:"type:varchar(64);index;not null" json:"task_id"`
	Sort      int       `gorm:"default:0" json:"sort"`
	ImageURL  string    `gorm:"type:varchar(500);not null" json:"image_url"`
	ErrorMsg  string    `gorm:"type:text;default:''" json:"error_message"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ProductReplaceImage) TableName() string {
	return "product_replace_images"
}
