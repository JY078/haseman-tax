package model

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
)

type BuyerReview struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	Fingerprint     string    `gorm:"size:64;not null;uniqueIndex" json:"fingerprint"`
	Brand           string    `gorm:"type:text" json:"brand"`
	Country         string    `gorm:"type:text" json:"country"`
	ParentASIN      string    `gorm:"type:text" json:"parent_asin"`
	ChildASIN       string    `gorm:"type:text" json:"child_asin"`
	Rating          float64   `json:"rating"`
	Name            string    `gorm:"type:text" json:"name"`
	ReviewDate      string    `gorm:"type:text" json:"date"`
	Title           string    `gorm:"type:text" json:"title"`
	Content         string    `gorm:"type:text" json:"content"`
	Link            string    `gorm:"type:text" json:"link"`
	Image           string    `gorm:"type:text" json:"image"`
	ProductImageURL string    `gorm:"-" json:"product_image_url"`
	ReviewImages    string    `gorm:"type:text" json:"review_images"`
	Video           bool      `json:"video"`
	VideoUpload     string    `gorm:"type:text" json:"video_upload"`
	VideoLink       string    `gorm:"type:text" json:"video_link"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (BuyerReview) TableName() string { return "buyer_reviews" }

// NormalizeReviewLink canonicalizes a buyer review link so the same review
// imported with different link spellings still deduplicates: query parameters
// and fragments are dropped, scheme/host are lowercased, http is upgraded to
// https, and a trailing path slash is removed. Values that do not parse as a
// URL fall back to trimmed lowercase, matching the legacy fingerprint input.
func NormalizeReviewLink(link string) string {
	trimmed := strings.TrimSpace(link)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.ToLower(trimmed)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "http" {
		scheme = "https"
	}
	return scheme + "://" + strings.ToLower(parsed.Host) + strings.TrimSuffix(parsed.EscapedPath(), "/")
}

// BuyerReviewFingerprint returns the deduplication key of a buyer review: the
// normalized review link when present, otherwise the legacy fallback of
// child ASIN + date + name + title + content.
func BuyerReviewFingerprint(link, childASIN, reviewDate, name, title, content string) string {
	identity := NormalizeReviewLink(link)
	if identity == "" {
		identity = strings.Join([]string{childASIN, reviewDate, name, title, content}, "\x00")
	}
	sum := sha256.Sum256([]byte(strings.ToLower(identity)))
	return hex.EncodeToString(sum[:])
}
