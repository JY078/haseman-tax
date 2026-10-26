package model

import "time"

type ProductSPUAttachment struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	SPUID             uint      `gorm:"column:spu_id;index;not null;default:0" json:"spu_id"`
	Label             string    `gorm:"type:varchar(120);not null;default:''" json:"label"`
	CertificationType string    `gorm:"type:varchar(32);not null;default:'';index" json:"certification_type"`
	CertificationDate string    `gorm:"type:varchar(10);not null;default:'';index" json:"certification_date"`
	FileURL           string    `gorm:"type:varchar(500);not null;default:''" json:"file_url"`
	FileName          string    `gorm:"type:varchar(255);not null;default:''" json:"file_name"`
	FileType          string    `gorm:"type:varchar(50);not null;default:''" json:"file_type"`
	TestReportURL     string    `gorm:"type:varchar(500);not null;default:''" json:"test_report_url"`
	TestReportName    string    `gorm:"type:varchar(255);not null;default:''" json:"test_report_name"`
	TestReportType    string    `gorm:"type:varchar(50);not null;default:''" json:"test_report_type"`
	Sort              int       `gorm:"default:0" json:"sort"`
	UpdatedBy         uint      `gorm:"not null;default:0" json:"updated_by"`
	UpdatedByName     string    `gorm:"-" json:"updated_by_name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (ProductSPUAttachment) TableName() string {
	return "product_spu_attachments"
}
