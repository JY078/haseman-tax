package model

import "time"

// Files are stored with the conversion so backups and transactions cover both.
type OrderConversion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Platform    string    `gorm:"type:varchar(20);not null;index" json:"platform"`
	ImportedBy  uint      `json:"imported_by"`
	SourceName  string    `json:"source_name"`
	MCFName     string    `json:"mcf_name"`
	SourceData  []byte    `gorm:"type:bytea;not null" json:"-"`
	MCFData     []byte    `gorm:"type:bytea;not null" json:"-"`
	Fingerprint string    `gorm:"size:64;not null;uniqueIndex" json:"-"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}
