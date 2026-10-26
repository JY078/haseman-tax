package model

import (
	"time"
)

type Setting struct {
	Key         string    `gorm:"primaryKey;type:varchar(255)" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Setting) TableName() string {
	return "settings"
}
