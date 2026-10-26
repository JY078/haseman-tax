package model

import (
	"encoding/json"
	"time"
)

type WorkRequest struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	WorkNo               string          `gorm:"size:32;not null;uniqueIndex" json:"work_no"`
	Type                 string          `gorm:"size:20;not null;default:'video';index" json:"type"`
	Title                string          `gorm:"size:200;not null" json:"title"`
	Status               string          `gorm:"size:30;not null;default:'pending';index" json:"status"`
	CreatorID            uint            `gorm:"not null;index" json:"creator_id"`
	Creator              *User           `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	AssigneeID           *uint           `gorm:"index" json:"assignee_id"`
	Assignee             *User           `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Progress             int             `gorm:"not null;default:0" json:"progress"`
	ProgressBeforeReview int             `gorm:"not null;default:10" json:"-"`
	Content              json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"content"`
	DeliveryNote         string          `gorm:"type:text;not null;default:''" json:"delivery_note"`
	DeliveryPaths        json.RawMessage `gorm:"type:jsonb;not null;default:'[]'" json:"delivery_paths"`
	RejectionReason      string          `gorm:"type:text;not null;default:''" json:"rejection_reason"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

func (WorkRequest) TableName() string {
	return "work_requests"
}
