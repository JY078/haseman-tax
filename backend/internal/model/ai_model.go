package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type StringArray []string

func (a *StringArray) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed")
	}
	return json.Unmarshal(bytes, a)
}

func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

type AiModel struct {
	ID                   uint        `gorm:"primaryKey" json:"id"`
	Name                 string      `gorm:"type:varchar(255);unique;not null" json:"name"`
	Types                StringArray `gorm:"type:text" json:"types"`
	IsActive             bool        `gorm:"default:true" json:"is_active"`
	IsDefault            bool        `gorm:"default:false" json:"is_default"`
	UsageCount           float64     `gorm:"type:decimal(12,6);default:0" json:"usage_count"`
	SupportedResolutions StringArray `gorm:"type:text" json:"supported_resolutions"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

func (AiModel) TableName() string {
	return "ai_models"
}
