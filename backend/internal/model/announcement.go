package model

import (
	"time"
)

type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(255);not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Status    string    `gorm:"type:varchar(20);default:'draft'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Announcement) TableName() string {
	return "announcements"
}

type AnnouncementRead struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index:idx_announcement_read_user" json:"user_id"`
	AnnouncementID uint      `gorm:"not null;index:idx_announcement_read_user" json:"announcement_id"`
	ReadAt         time.Time `gorm:"not null" json:"read_at"`
}

func (AnnouncementRead) TableName() string {
	return "announcement_reads"
}
