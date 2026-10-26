package model

import "time"

// Prompt status constants
const (
	PromptStatusApproved = "approved" // 已上架（公开可见）
	PromptStatusPending  = "pending"  // 待审核（用户提交）
	PromptStatusOffline  = "offline"  // 已下架

	PromptSourceAdmin = "admin" // 管理员添加，默认直接上架
	PromptSourceUser  = "user"  // 用户提交，需审核
)

type PromptGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	Sort        int       `gorm:"default:0" json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Prompts     []Prompt  `gorm:"foreignKey:GroupID" json:"prompts,omitempty"`
}

func (PromptGroup) TableName() string {
	return "prompt_groups"
}

type Prompt struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	GroupID   uint        `gorm:"index;default:0" json:"group_id"`
	Shape     string      `gorm:"type:varchar(20);not null;default:'square';index" json:"shape"`
	Title     string      `gorm:"type:varchar(255);not null" json:"title"`
	Content   string      `gorm:"type:text;not null" json:"content"`
	Images    StringArray `gorm:"type:text;default:'[]'" json:"images"`
	Status    string      `gorm:"type:varchar(20);not null;default:'approved'" json:"status"`
	Source    string      `gorm:"type:varchar(20);not null;default:'admin'" json:"source"`
	Sort      int         `gorm:"default:0" json:"sort"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (Prompt) TableName() string {
	return "prompts"
}
