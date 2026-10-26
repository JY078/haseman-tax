package model

import "time"

const (
	ReferenceGalleryStatusEnabled  = "enabled"
	ReferenceGalleryStatusDisabled = "disabled"
)

type ReferenceGalleryAsset struct {
	Key         string                   `json:"key"`
	URL         string                   `json:"url"`
	Description string                   `json:"description"`
	Regions     []ReferenceGalleryRegion `json:"regions,omitempty"`
}

type ReferenceGalleryRegion struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	Fit  string  `json:"fit"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

type ReferenceGallery struct {
	ID          uint                    `gorm:"primaryKey" json:"id"`
	Name        string                  `gorm:"type:varchar(120);not null" json:"name"`
	Category    string                  `gorm:"type:varchar(80);not null;default:''" json:"category"`
	Version     string                  `gorm:"type:varchar(80);not null;default:''" json:"version"`
	Description string                  `gorm:"type:varchar(500);not null;default:''" json:"description"`
	Status      string                  `gorm:"type:varchar(20);not null;default:'enabled';index" json:"status"`
	Assets      []ReferenceGalleryAsset `gorm:"type:json;serializer:json;default:'[]'" json:"assets"`
	Sort        int                     `gorm:"default:0" json:"sort"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

func (ReferenceGallery) TableName() string {
	return "reference_galleries"
}
