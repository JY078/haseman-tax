package model

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type TaxRate struct {
	ProviderTimestamp int64           `json:"provider_timestamp,omitempty"`
	Rate              decimal.Decimal `json:"rate"`
	Source            string          `json:"source"`
	Date              string          `json:"date"`
	FetchedAt         time.Time       `json:"fetched_at"`
}

type TaxExchangeRate struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Date      string          `gorm:"size:10;uniqueIndex:tax_exchange_rate_date_currency;not null" json:"date"`
	Currency  string          `gorm:"size:3;uniqueIndex:tax_exchange_rate_date_currency;not null" json:"currency"`
	Rate      decimal.Decimal `gorm:"type:numeric(24,12);not null" json:"rate"`
	Source    string          `gorm:"size:50" json:"source"`
	FetchedAt time.Time       `json:"fetched_at"`
}

type TaxDataBatch struct {
	ID            uint               `gorm:"primaryKey" json:"id"`
	StoreID       string             `gorm:"size:100;not null;uniqueIndex:tax_batch_file" json:"store_id"`
	SiteID        string             `gorm:"size:20;uniqueIndex:tax_batch_file" json:"site_id"`
	FileHash      string             `gorm:"size:64;not null;uniqueIndex:tax_batch_file" json:"file_hash"`
	FileName      string             `gorm:"size:255;not null" json:"file_name"`
	FileContent   []byte             `gorm:"type:bytea" json:"-"`
	RawRows       int                `json:"raw_rows"`
	UnmatchedRows int                `json:"unmatched_rows"`
	Rates         map[string]TaxRate `gorm:"type:jsonb;serializer:json" json:"rates"`
	Status        string             `gorm:"size:30;index;not null" json:"status"`
	CreatedAt     time.Time          `json:"created_at"`
}

type TaxRawRow struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	BatchID     uint              `gorm:"index;not null" json:"batch_id"`
	RowNumber   int               `json:"row_number"`
	StoreID     string            `gorm:"size:100;not null;uniqueIndex:tax_raw_identity" json:"store_id"`
	SiteID      string            `gorm:"size:20;uniqueIndex:tax_raw_identity" json:"site_id"`
	Fingerprint string            `gorm:"size:64;not null;uniqueIndex:tax_raw_identity" json:"-"`
	Data        map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"data"`
}

type TaxProcessedRow struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	BatchID        uint            `gorm:"index;not null" json:"batch_id"`
	RawRowID       uint            `gorm:"uniqueIndex;not null" json:"raw_row_id"`
	Month          string          `gorm:"size:7;index" json:"month"`
	Quarter        string          `gorm:"size:7;index" json:"quarter"`
	FeeType        string          `gorm:"size:50;index" json:"fee_type"`
	Fee            decimal.Decimal `gorm:"type:numeric(24,2)" json:"fee"`
	Amount         decimal.Decimal `gorm:"type:numeric(24,8)" json:"amount"`
	ShipQty        decimal.Decimal `gorm:"type:numeric(24,8)" json:"ship_qty"`
	Rate           decimal.Decimal `gorm:"type:numeric(24,12)" json:"rate"`
	Currency       string          `gorm:"size:3" json:"currency"`
	RuleSnapshot   TaxFeeRule      `gorm:"type:jsonb;serializer:json" json:"rule_snapshot"`
	TransactionKey string          `gorm:"type:text" json:"transaction_key"`
}

type TaxQuarter struct {
	ID            uint                       `gorm:"primaryKey" json:"id"`
	BatchID       uint                       `gorm:"uniqueIndex:tax_batch_quarter;not null" json:"batch_id"`
	Quarter       string                     `gorm:"size:7;uniqueIndex:tax_batch_quarter;not null" json:"quarter"`
	Totals        map[string]decimal.Decimal `gorm:"type:jsonb;serializer:json" json:"totals"`
	Total         decimal.Decimal            `gorm:"type:numeric(24,2)" json:"total"`
	TaxableIncome decimal.Decimal            `gorm:"type:numeric(24,2)" json:"taxable_income"`
}

type TaxFeeRule struct {
	ID                     uint   `gorm:"primaryKey" json:"id"`
	MatchKey               string `gorm:"size:64;uniqueIndex;not null" json:"match_key"`
	TransactionType        string `gorm:"size:100" json:"transaction_type"`
	TransactionDescription string `gorm:"size:255" json:"transaction_description"`
	AmountType             string `gorm:"size:150" json:"amount_type"`
	CustomFeeType          string `gorm:"size:50;not null" json:"custom_fee_type"`
	Allocation             string `gorm:"size:50" json:"allocation"`
	Note                   string `gorm:"size:255" json:"note"`
}

func TaxMatchKey(a, b, c string) string {
	normalize := func(s string) string { return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "\"")) }
	data, _ := json.Marshal([]string{normalize(a), normalize(b), normalize(c)})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

//go:embed tax_fee_rules.json
var taxRuleData []byte

func DefaultTaxFeeRules() ([]TaxFeeRule, error) {
	var rules []TaxFeeRule
	if err := json.Unmarshal(taxRuleData, &rules); err != nil {
		return nil, err
	}
	for i := range rules {
		r := &rules[i]
		r.MatchKey = TaxMatchKey(r.TransactionType, r.TransactionDescription, r.AmountType)
	}
	return rules, nil
}

// Importer is an RPA-reported label, not an authenticated human identity.
type TaxImportLog struct {
	Platform     string    `gorm:"size:20;index;not null;default:walmart" json:"platform"`
	ReportType   string    `gorm:"size:30" json:"report_type"`
	ID           uint      `gorm:"primaryKey" json:"id"`
	BatchID      uint      `gorm:"index" json:"batch_id"`
	StoreID      string    `gorm:"size:100;index" json:"store_id"`
	SiteID       string    `gorm:"size:20;index" json:"site_id"`
	Importer     string    `gorm:"size:150" json:"importer"`
	FileName     string    `gorm:"size:255" json:"file_name"`
	FileContent  []byte    `gorm:"type:bytea" json:"-"`
	SourceRows   int       `json:"source_rows"`
	ImportedRows int       `json:"imported_rows"`
	SkippedRows  int       `json:"skipped_rows"`
	Status       string    `gorm:"size:30;index" json:"status"`
	Message      string    `gorm:"type:text" json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

type TaxDownloadLog struct {
	Platform    string    `gorm:"size:20;index;not null;default:walmart" json:"platform"`
	SiteID      string    `gorm:"size:20;index" json:"site_id"`
	Month       string    `gorm:"type:text" json:"month"`
	Currency    string    `gorm:"size:3" json:"currency"`
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index" json:"user_id"`
	Username    string    `gorm:"size:150" json:"username"`
	ReportType  string    `gorm:"size:30" json:"report_type"`
	Quarter     string    `gorm:"type:text" json:"quarter"`
	FileName    string    `gorm:"size:255" json:"file_name"`
	FileContent []byte    `gorm:"type:bytea" json:"-"`
	ContentType string    `gorm:"size:100" json:"-"`
	RowCount    int       `json:"row_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type TikTokTaxBatch struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	StoreID     string    `gorm:"size:100;index;uniqueIndex:tiktok_tax_batch_site_file" json:"store_id"`
	SiteID      string    `gorm:"size:20;index;uniqueIndex:tiktok_tax_batch_site_file" json:"site_id"`
	FileHash    string    `gorm:"size:64;not null;uniqueIndex:tiktok_tax_batch_site_file" json:"file_hash"`
	FileName    string    `gorm:"size:255;not null" json:"file_name"`
	FileContent []byte    `gorm:"type:bytea" json:"-"`
	RawRows     int       `json:"raw_rows"`
	CreatedAt   time.Time `json:"created_at"`
}

type TikTokTaxTransactionSummary struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	BatchID          uint   `gorm:"index;not null" json:"batch_id"`
	StoreID          string `gorm:"size:100;index;uniqueIndex:tiktok_transaction_summary_store" json:"store_id"`
	SiteID           string `gorm:"size:20;index;uniqueIndex:tiktok_transaction_summary_store" json:"site_id"`
	Quarter          string `gorm:"size:7;uniqueIndex:tiktok_transaction_summary_store" json:"quarter"`
	TransactionCount int64  `json:"transaction_count"`
}

type TikTokTaxRawRow struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	BatchID     uint              `gorm:"index;not null" json:"batch_id"`
	StoreID     string            `gorm:"size:100;index;uniqueIndex:tiktok_tax_raw_site_identity" json:"store_id"`
	SiteID      string            `gorm:"size:20;index;uniqueIndex:tiktok_tax_raw_site_identity" json:"site_id"`
	Fingerprint string            `gorm:"size:64;uniqueIndex:tiktok_tax_raw_site_identity;not null" json:"-"`
	Data        map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"data"`
}

type TikTokTaxDetail struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	BatchID    uint            `gorm:"index;not null" json:"batch_id"`
	StoreID    string          `gorm:"size:100;index" json:"store_id"`
	SiteID     string          `gorm:"size:20;index" json:"site_id"`
	RawRowID   uint            `gorm:"uniqueIndex;not null" json:"raw_row_id"`
	Month      string          `gorm:"size:7;index" json:"month"`
	Quarter    string          `gorm:"size:7;index" json:"quarter"`
	Fee        decimal.Decimal `gorm:"type:numeric(24,2)" json:"fee"`
	PostedAt   time.Time       `gorm:"index:idx_tiktok_detail_identity,priority:1;index" json:"posted_at"`
	BusinessID string          `gorm:"size:255;index:idx_tiktok_detail_identity,priority:2;index" json:"business_id"`
	Income     decimal.Decimal `gorm:"type:numeric(24,8)" json:"income"`
	Refund     decimal.Decimal `gorm:"type:numeric(24,8)" json:"refund"`
	Commission decimal.Decimal `gorm:"type:numeric(24,8)" json:"commission"`
	Rate       decimal.Decimal `gorm:"type:numeric(24,12)" json:"rate"`
}

type TikTokTaxQuarter struct {
	ID            uint            `gorm:"primaryKey" json:"id"`
	Quarter       string          `gorm:"size:7;uniqueIndex" json:"quarter"`
	Income        decimal.Decimal `gorm:"type:numeric(24,2)" json:"income"`
	Refund        decimal.Decimal `gorm:"type:numeric(24,2)" json:"refund"`
	Commission    decimal.Decimal `gorm:"type:numeric(24,2)" json:"commission"`
	TaxableIncome decimal.Decimal `gorm:"type:numeric(24,2)" json:"taxable_income"`
}

// TikTokTaxAdBatch preserves the original RPA file and prevents duplicate uploads.
type TikTokTaxAdBatch struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	StoreID     string    `gorm:"size:100;index;uniqueIndex:tiktok_tax_ad_batch_site_file" json:"store_id"`
	SiteID      string    `gorm:"size:20;index;uniqueIndex:tiktok_tax_ad_batch_site_file" json:"site_id"`
	FileHash    string    `gorm:"size:64;uniqueIndex:tiktok_tax_ad_batch_site_file;not null" json:"-"`
	FileName    string    `gorm:"size:255;not null" json:"file_name"`
	FileContent []byte    `gorm:"type:bytea" json:"-"`
	RowCount    int       `json:"row_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type TikTokTaxAd struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	BatchID      uint            `gorm:"index;not null" json:"batch_id"`
	StoreID      string          `gorm:"size:100;index" json:"store_id"`
	SiteID       string          `gorm:"size:20;index" json:"site_id"`
	Month        string          `gorm:"size:7;index;not null" json:"month"`
	Quarter      string          `gorm:"size:7;index;not null" json:"quarter"`
	Date         string          `gorm:"size:10;index;index:idx_tiktok_ad_identity,priority:1;not null" json:"date"`
	CampaignName string          `gorm:"type:text" json:"campaign_name"`
	CampaignID   string          `gorm:"type:text;index:idx_tiktok_ad_identity,priority:2" json:"campaign_id"`
	CashSpend    decimal.Decimal `gorm:"type:numeric(24,8);not null" json:"cash_spend"`
	CreditSpend  decimal.Decimal `gorm:"type:numeric(24,8);not null" json:"credit_spend"`
	GrantSpend   decimal.Decimal `gorm:"type:numeric(24,8);not null" json:"grant_spend"`
	Amount       decimal.Decimal `gorm:"type:numeric(24,8);not null" json:"amount"`
	Currency     string          `gorm:"size:3;not null" json:"currency"`
	Type         string          `gorm:"type:text" json:"type"`
	Rate         decimal.Decimal `gorm:"type:numeric(24,12);not null" json:"rate"`
	Fee          decimal.Decimal `gorm:"type:numeric(24,2);not null" json:"fee"`
}

type TikTokTaxFBT struct {
	ID                             uint            `gorm:"primaryKey" json:"id"`
	BatchID                        uint            `gorm:"index;not null" json:"batch_id"`
	StoreID                        string          `gorm:"size:100;index" json:"store_id"`
	SiteID                         string          `gorm:"size:20;index" json:"site_id"`
	Month                          string          `gorm:"size:7;index;not null" json:"month"`
	Quarter                        string          `gorm:"size:7;index;not null" json:"quarter"`
	GoodsID                        string          `json:"goodsId"`
	SKUID                          string          `json:"skuId"`
	ProductID                      string          `json:"productId"`
	ProductName                    string          `json:"productName"`
	ShopID                         string          `json:"shopId"`
	ShopName                       string          `json:"shopName"`
	CostRate                       decimal.Decimal `gorm:"type:numeric(24,8)" json:"costRate"`
	GMV                            decimal.Decimal `gorm:"type:numeric(24,8)" json:"gmv"`
	TotalLogisticsCost             decimal.Decimal `gorm:"type:numeric(24,8)" json:"totalLogisticsCost"`
	FBTFulfillmentFee              decimal.Decimal `gorm:"type:numeric(24,8)" json:"fbtFulfillmentFee"`
	FBTFulfillmentFeeReimbursement decimal.Decimal `gorm:"type:numeric(24,8)" json:"fbtFulfillmentFeeReimbursement"`
	ShippingFeeDiscount            decimal.Decimal `gorm:"type:numeric(24,8)" json:"shippingFeeDiscount"`
	HubPlacementFee                decimal.Decimal `gorm:"type:numeric(24,8)" json:"hubPlacementFee"`
	StorageFee                     decimal.Decimal `gorm:"type:numeric(24,8)" json:"storageFee"`
	InboundShippingFee             decimal.Decimal `gorm:"type:numeric(24,8)" json:"inboundShippingFee"`
	InboundIncidentsFee            decimal.Decimal `gorm:"type:numeric(24,8)" json:"inboundIncidentsFee"`
	InboundBookingNonComplianceFee decimal.Decimal `gorm:"type:numeric(24,8)" json:"inboundBookingNonComplianceFee"`
	RoutingNonComplianceFee        decimal.Decimal `gorm:"type:numeric(24,8)" json:"routingNonComplianceFee"`
	OutboundAppointmentNoShowFee   decimal.Decimal `gorm:"type:numeric(24,8)" json:"outboundAppointmentNoShowFee"`
	DelayedResponseFee             decimal.Decimal `gorm:"type:numeric(24,8)" json:"delayedResponseFee"`
	DisposalFee                    decimal.Decimal `gorm:"type:numeric(24,8)" json:"disposalFee"`
	ReturnToSellerShippingFee      decimal.Decimal `gorm:"type:numeric(24,8)" json:"returnToSellerShippingFee"`
	ReturnToSellerHandlingFee      decimal.Decimal `gorm:"type:numeric(24,8)" json:"returnToSellerHandlingFee"`
	InboundReturnOperationFee      decimal.Decimal `gorm:"type:numeric(24,8)" json:"inboundReturnOperationFee"`
}

type TikTokTaxFBTBatch struct {
	ID          uint   `gorm:"primaryKey"`
	StoreID     string `gorm:"size:100;index;uniqueIndex:tiktok_tax_fbt_batch_site_file"`
	SiteID      string `gorm:"size:20;index;uniqueIndex:tiktok_tax_fbt_batch_site_file"`
	FileHash    string `gorm:"size:64;uniqueIndex:tiktok_tax_fbt_batch_site_file;not null"`
	FileName    string `gorm:"size:255;not null"`
	FileContent []byte `gorm:"type:bytea"`
	RowCount    int
	CreatedAt   time.Time
}

func (TikTokTaxFBT) TableName() string      { return "tik_tok_tax_fbts" }
func (TikTokTaxFBTBatch) TableName() string { return "tik_tok_tax_fbt_batches" }
