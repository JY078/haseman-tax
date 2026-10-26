package model

import "time"

type OrderFieldMapping struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Platform      string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_order_field_mapping_platform_key" json:"platform"`
	FieldKey      string    `gorm:"type:varchar(80);not null;uniqueIndex:idx_order_field_mapping_platform_key" json:"field_key"`
	SourceHeaders string    `gorm:"type:text;not null" json:"-"`
	UpdatedBy     uint      `gorm:"not null;default:0" json:"updated_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (OrderFieldMapping) TableName() string {
	return "order_field_mappings"
}
