package model

import "time"

type Feedback struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	ContactInfo string    `gorm:"type:varchar(255)" json:"contact_info"`
	Status      string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending / reviewed
	AdminNote   string    `gorm:"type:text" json:"admin_note"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Feedback) TableName() string {
	return "feedbacks"
}
