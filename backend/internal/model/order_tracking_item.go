package model

import "time"

type OrderTrackingItem struct {
	ID                    uint                   `gorm:"primaryKey" json:"id"`
	Platform              string                 `gorm:"type:varchar(20);not null;uniqueIndex:idx_order_tracking_platform_order_sku_package;index" json:"platform"`
	OrderID               string                 `gorm:"type:varchar(160);not null;uniqueIndex:idx_order_tracking_platform_order_sku_package;index" json:"order_id"`
	OriginalOrderIDs      []string               `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"original_order_ids"`
	OriginalItems         []OrderTrackingSource  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"original_items"`
	IsMergedOrder         bool                   `gorm:"not null;default:false;index" json:"is_merged_order"`
	PlatformSKU           string                 `gorm:"type:varchar(160);not null;uniqueIndex:idx_order_tracking_platform_order_sku_package;index" json:"sku"`
	PackageKey            string                 `gorm:"type:varchar(200);not null;default:'';uniqueIndex:idx_order_tracking_platform_order_sku_package;index" json:"package_key"`
	AmazonSKU             string                 `gorm:"-" json:"amazon_sku"`
	Quantity              int                    `gorm:"not null;default:0" json:"quantity"`
	Recipient             string                 `gorm:"type:varchar(200);not null;default:''" json:"recipient"`
	Phone                 string                 `gorm:"type:varchar(80);not null;default:''" json:"phone"`
	Address               string                 `gorm:"type:text;not null;default:''" json:"address"`
	TrackingNumber        string                 `gorm:"type:varchar(160);not null;default:'';index" json:"tracking_number"`
	AmazonOrderID         string                 `gorm:"type:varchar(160);not null;default:''" json:"amazon_order_id"`
	Status                string                 `gorm:"type:varchar(24);not null;default:'pending_shipment';index" json:"status"`
	Remark                string                 `gorm:"type:text;not null;default:''" json:"remark"`
	ImportedBy            uint                   `gorm:"not null;default:0" json:"imported_by"`
	ImportedByName        string                 `gorm:"-" json:"imported_by_name"`
	Carrier               string                 `gorm:"-" json:"carrier"`
	TrackingUpdatedBy     uint                   `gorm:"not null;default:0" json:"tracking_updated_by"`
	TrackingUpdatedByName string                 `gorm:"-" json:"tracking_updated_by_name"`
	TrackingUpdatedAt     *time.Time             `json:"tracking_updated_at,omitempty"`
	ExportedAt            *time.Time             `json:"exported_at,omitempty"`
	CanEditTracking       bool                   `gorm:"-" json:"can_edit_tracking"`
	CanRequestCancel      bool                   `gorm:"-" json:"can_request_cancel"`
	CanCancel             bool                   `gorm:"-" json:"can_cancel"`
	CanRevokeCancel       bool                   `gorm:"-" json:"can_revoke_cancel"`
	Details               []OrderTrackingDetail  `gorm:"-" json:"details,omitempty"`
	Packages              []OrderTrackingPackage `gorm:"-" json:"packages,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

type OrderTrackingDetail struct {
	OriginalOrderID string `json:"original_order_id"`
	AmazonSKU       string `json:"amazon_sku"`
	Quantity        int    `json:"quantity"`
}

type OrderTrackingPackage struct {
	ID               uint                  `json:"id"`
	PackageKey       string                `json:"package_key"`
	Quantity         int                   `json:"quantity"`
	TrackingNumber   string                `json:"tracking_number"`
	AmazonOrderID    string                `json:"amazon_order_id"`
	Carrier          string                `json:"carrier"`
	Status           string                `json:"status"`
	Remark           string                `json:"remark"`
	Exported         bool                  `json:"exported"`
	CanEditTracking  bool                  `json:"can_edit_tracking"`
	CanRequestCancel bool                  `json:"can_request_cancel"`
	CanCancel        bool                  `json:"can_cancel"`
	CanRevokeCancel  bool                  `json:"can_revoke_cancel"`
	Details          []OrderTrackingDetail `json:"details,omitempty"`
}

type OrderTrackingSource struct {
	OriginalOrderID string `json:"original_order_id"`
	PlatformSKU     string `json:"sku"`
	AmazonSKU       string `json:"amazon_sku"`
	Quantity        int    `json:"quantity"`
}

func (OrderTrackingItem) TableName() string {
	return "order_tracking_items"
}
