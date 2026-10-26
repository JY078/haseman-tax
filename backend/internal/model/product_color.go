package model

import "time"

type ProductColorFamily struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	ImageURL  string    `gorm:"type:varchar(500);default:''" json:"image_url"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ProductColorFamily) TableName() string {
	return "product_color_families"
}
