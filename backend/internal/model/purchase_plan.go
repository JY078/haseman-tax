package model

import "time"

type PurchasePlan struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	ProductSKUID        uint      `gorm:"column:product_sku_id;not null;index" json:"product_sku_id"`
	ProductName         string    `gorm:"type:varchar(255);not null" json:"product_name"`
	SKU                 string    `gorm:"type:varchar(120);not null;index" json:"sku"`
	PlannedQuantity     int       `gorm:"not null" json:"planned_quantity"`
	UnitsPerCarton      int       `gorm:"not null" json:"units_per_carton"`
	CartonCount         int       `gorm:"not null" json:"carton_count"`
	ExpectedArrivalDate time.Time `gorm:"type:date;not null" json:"expected_arrival_date"`
	Remark              string    `gorm:"type:text;not null;default:''" json:"remark"`
	ApplicantID         uint      `gorm:"not null;index" json:"applicant_id"`
	Applicant           *User     `gorm:"foreignKey:ApplicantID" json:"-"`
	ApplicantName       string    `gorm:"-" json:"applicant_name"`
	CreatedAt           time.Time `json:"created_at"`
}

func (PurchasePlan) TableName() string {
	return "purchase_plans"
}
