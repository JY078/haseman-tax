package model

import "time"

type DailyCall struct {
	Kind                      string     `gorm:"size:20;not null;default:inbound;index" json:"kind"`
	InformationRequested      string     `gorm:"type:text;not null;default:''" json:"information_requested"`
	Result                    string     `gorm:"type:text;not null;default:''" json:"result"`
	ID                        uint       `gorm:"primaryKey" json:"id"`
	Name                      string     `gorm:"size:200" json:"name"`
	OrderNumber               string     `gorm:"size:200" json:"order_number"`
	Email                     string     `gorm:"size:254" json:"email"`
	PhoneNo                   string     `gorm:"size:100" json:"phone_no"`
	BoughtIn                  string     `gorm:"size:200" json:"bought_in"`
	Country                   string     `gorm:"size:100" json:"country"`
	BriefOfIssue              string     `gorm:"type:text" json:"brief_of_issue"`
	QualityResult             string     `gorm:"type:text;default:''" json:"quality_result"`
	Reason                    string     `gorm:"type:text;default:''" json:"reason"`
	SKU                       string     `gorm:"size:200" json:"sku"`
	ASIN                      string     `gorm:"size:200" json:"asin"`
	IssueType                 string     `gorm:"size:200" json:"issue_type"`
	CreatedByID               uint       `gorm:"not null" json:"created_by_id"`
	CreatedByName             string     `gorm:"size:100;not null" json:"created_by_name"`
	RegistrationUpdatedByID   uint       `gorm:"not null" json:"registration_updated_by_id"`
	RegistrationUpdatedByName string     `gorm:"size:100;not null" json:"registration_updated_by_name"`
	RegistrationUpdatedAt     time.Time  `gorm:"not null" json:"registration_updated_at"`
	ReviewedByID              *uint      `json:"reviewed_by_id"`
	ReviewedByName            string     `gorm:"size:100" json:"reviewed_by_name"`
	ReviewedAt                *time.Time `json:"reviewed_at"`
	CreatedAt                 time.Time  `gorm:"not null;index" json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}
