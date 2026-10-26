package model

type TaxCatalogEntry struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Platform string `gorm:"size:20;uniqueIndex:tax_catalog_identity;not null" json:"platform"`
	Kind     string `gorm:"size:10;uniqueIndex:tax_catalog_identity;not null" json:"kind"`
	Value    string `gorm:"size:100;uniqueIndex:tax_catalog_identity;not null" json:"value"`
	Label    string `gorm:"size:100;not null" json:"label"`
	Currency string `gorm:"size:3" json:"currency"`
}
