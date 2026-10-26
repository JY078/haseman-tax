package model

import "time"

type Menu struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ParentID  uint      `gorm:"default:0" json:"parent_id"`
	Title     string    `gorm:"size:50;not null" json:"title"`
	Path      string    `gorm:"size:100;not null" json:"path"`
	Icon      string    `gorm:"size:50" json:"icon"`
	Sort      int       `gorm:"default:0" json:"sort"`
	Children  []*Menu   `gorm:"-" json:"children,omitempty"` // For tree structure response
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
