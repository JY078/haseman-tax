package model

import "time"

type ImageViolationExample struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Platform    string    `gorm:"size:30;not null" json:"platform"`
	ProblemType string    `gorm:"size:100;not null" json:"problem_type"`
	Description string    `gorm:"type:text;not null" json:"description"`
	ImageURL    string    `gorm:"type:text;not null" json:"image_url"`
	UserID      uint      `gorm:"not null" json:"user_id"`
	Submitter   string    `gorm:"size:100;not null" json:"submitter"`
	CreatedAt   time.Time `gorm:"not null;index" json:"created_at"`
}
