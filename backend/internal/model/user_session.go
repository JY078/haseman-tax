package model

import "time"

type UserSession struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index" json:"user_id"`
	TokenHash    string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"token_hash"`
	ClientType   string     `gorm:"type:varchar(20);default:'web'" json:"client_type"` // "web" | "admin"
	UserAgent    string     `gorm:"type:text" json:"user_agent"`
	IP           string     `gorm:"type:varchar(64)" json:"ip"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"expires_at"`
	LastActiveAt *time.Time `gorm:"index" json:"last_active_at"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}
