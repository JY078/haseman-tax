package model

import "time"

type User struct {
	ID                       uint      `gorm:"primaryKey" json:"id"`
	Username                 string    `gorm:"size:50;not null" json:"username"`
	Name                     string    `gorm:"size:100" json:"name"`
	Platform                 string    `gorm:"size:50" json:"platform"`
	Department               string    `gorm:"size:100" json:"department"`
	SubDepartment            string    `gorm:"size:100" json:"sub_department"`
	Password                 string    `gorm:"not null" json:"-"`
	Email                    string    `gorm:"size:100;not null" json:"email"`
	UsageCount               int       `gorm:"default:0" json:"usage_count"`
	LongLogin                bool      `gorm:"not null;default:false" json:"long_login"`
	Enabled                  bool      `gorm:"not null;default:true" json:"enabled"`
	Online                   bool      `gorm:"-" json:"online"`
	Roles                    []Role    `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	OrderPlatforms           []string  `gorm:"-" json:"order_platforms"`
	OrderPlatformsConfigured bool      `gorm:"not null;default:false" json:"-"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type UserOrderPlatform struct {
	UserID   uint   `gorm:"primaryKey" json:"user_id"`
	Platform string `gorm:"primaryKey;size:20" json:"platform"`
}
