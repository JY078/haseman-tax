package model

import "time"

type Department struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Platform      string    `gorm:"type:varchar(50);not null;default:''" json:"platform"`
	Name          string    `gorm:"type:varchar(100);not null" json:"name"`
	SubName       string    `gorm:"type:varchar(100);not null;default:''" json:"sub_name"`
	NASRegionPath string    `gorm:"type:varchar(200);not null;default:''" json:"nas_region_path"`
	Sort          int       `gorm:"default:0" json:"sort"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Department) TableName() string {
	return "departments"
}
