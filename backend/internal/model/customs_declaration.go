package model

import "time"

// CustomsDeclaration retains the original PDF and exact extracted decimal strings for audit.
type CustomsDeclaration struct {
	ID                uint                `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time           `json:"created_at"`
	UserID            uint                `json:"user_id"`
	FileName          string              `gorm:"size:255;not null" json:"file_name"`
	FileHash          string              `gorm:"size:64;uniqueIndex;not null" json:"-"`
	FileContent       []byte              `gorm:"type:bytea;not null" json:"-"`
	DeclarationNumber string              `gorm:"size:32;uniqueIndex;not null" json:"declaration_number"`
	DeclarationDate   string              `gorm:"size:10;index" json:"declaration_date"`
	Fields            map[string]string   `gorm:"type:jsonb;serializer:json;not null" json:"fields"`
	Items             []map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"items"`
	Warnings          []string            `gorm:"type:jsonb;serializer:json;not null" json:"warnings"`
	RecognitionMethod string              `gorm:"size:20" json:"recognition_method"`
	RawText           string              `gorm:"type:text" json:"raw_text,omitempty"`
}
