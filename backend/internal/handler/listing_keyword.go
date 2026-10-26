package handler

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var listingKeywordStages = map[string]bool{
	"expanded": true,
	"filtered": true,
	"tagged":   true,
	"library":  true,
	"rejected": true,
}

func normalizeListingPlatform(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "_", "-", "_", "'", "", "’", "")
	normalized = replacer.Replace(normalized)
	switch normalized {
	case "tiktok", "tiktok_shop", "tiktokshop":
		return "tiktok"
	case "home_depot", "homedepot":
		return "home_depot"
	case "lowes", "lowe", "lowe_s":
		return "lowes"
	case "wayfair":
		return "wayfair"
	case "amazon", "amazon_com":
		return "amazon"
	default:
		return strings.Trim(normalized, "_")
	}
}

func normalizeListingKeyword(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func splitListingKeywords(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';' || r == '，' || r == '；'
	})
	seen := make(map[string]struct{}, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		term := normalizeListingKeyword(part)
		key := term
		if term == "" || len([]rune(term)) > 255 {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, term)
	}
	return result
}

func activeListingKeywordRecords(platform, countryCode, category string, limit int) []model.ListingKeyword {
	if database.DB == nil {
		return nil
	}
	platform = normalizeListingPlatform(platform)
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	category = strings.TrimSpace(category)
	if platform == "" {
		return nil
	}
	if limit < 1 || limit > 200 {
		limit = 80
	}
	var records []model.ListingKeyword
	query := database.DB.Where("platform = ? AND stage = ? AND status = ?", platform, "library", "active").
		Where("(country_code = '' OR country_code = ?)", countryCode).
		Where("(category = '' OR LOWER(category) = LOWER(?))", category).
		Order("CASE WHEN country_code = '' THEN 1 ELSE 0 END ASC").
		Order("CASE WHEN category = '' THEN 1 ELSE 0 END ASC").
		Order("usage_count DESC, updated_at DESC").
		Limit(limit)
	if err := query.Find(&records).Error; err != nil {
		return nil
	}
	return records
}

func activeListingKeywordLibrary(platform, countryCode, category string, limit int) string {
	records := activeListingKeywordRecords(platform, countryCode, category, limit)
	terms := make([]string, 0, len(records))
	for _, record := range records {
		terms = append(terms, record.Term)
	}
	return strings.Join(terms, ", ")
}

func listingKeywordLibraryWithInput(platform, countryCode, category, input string) string {
	terms := append([]string{}, splitListingKeywords(input)...)
	terms = append(terms, splitListingKeywords(activeListingKeywordLibrary(platform, countryCode, category, 80))...)
	return strings.Join(splitListingKeywords(strings.Join(terms, ", ")), ", ")
}

func recordListingKeywordCandidates(platform, countryCode, category, source string, values []string) int {
	candidates := make([]model.EcomSuiteKeywordCandidate, 0, len(values))
	for _, value := range values {
		candidates = append(candidates, model.EcomSuiteKeywordCandidate{Term: value})
	}
	return recordListingKeywordCandidatesWithMetadata(platform, countryCode, category, source, candidates)
}

func recordListingKeywordCandidatesWithMetadata(platform, countryCode, category, source string, values []model.EcomSuiteKeywordCandidate) int {
	if database.DB == nil {
		return 0
	}
	platform = normalizeListingPlatform(platform)
	if platform == "" {
		return 0
	}
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	category = strings.TrimSpace(category)
	if strings.TrimSpace(source) == "" {
		source = "agent"
	}
	if len([]rune(source)) > 40 {
		source = string([]rune(source)[:40])
	}
	created := 0
	for _, value := range values {
		term := normalizeListingKeyword(value.Term)
		if term == "" || len([]rune(term)) > 255 {
			continue
		}
		tags := make([]string, 0, len(value.Tags))
		seenTags := make(map[string]struct{}, len(value.Tags))
		for _, rawTag := range value.Tags {
			tag := normalizeListingKeyword(rawTag)
			if tag == "" || len([]rune(tag)) > 80 {
				continue
			}
			if _, exists := seenTags[tag]; exists {
				continue
			}
			seenTags[tag] = struct{}{}
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		record := model.ListingKeyword{Platform: platform, CountryCode: countryCode, Category: category, Term: term, Stage: "expanded", Status: "candidate", Source: source, Tags: model.StringSlice(tags), Notes: strings.TrimSpace(value.Reason), SearchVolume: value.SearchVolume, Competition: value.Competition, PlatformFrequency: value.PlatformFrequency, SEOScore: value.SEOScore, FactIDs: model.StringSlice(value.FactIDs), EvidenceURLs: model.StringSlice(value.EvidenceURLs)}
		result := database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "platform"}, {Name: "country_code"}, {Name: "category"}, {Name: "term"}},
			DoNothing: true,
		}).Create(&record)
		if result.Error == nil && result.RowsAffected > 0 {
			created++
		}
	}
	return created
}

func markListingKeywordUsage(records []model.ListingKeyword, listingText string) {
	if database.DB == nil || len(records) == 0 {
		return
	}
	lowerText := strings.ToLower(listingText)
	now := time.Now()
	ids := make([]uint64, 0, len(records))
	for _, record := range records {
		if strings.Contains(lowerText, strings.ToLower(record.Term)) {
			ids = append(ids, record.ID)
		}
	}
	if len(ids) > 0 {
		_ = database.DB.Model(&model.ListingKeyword{}).Where("id IN ?", ids).
			Updates(map[string]interface{}{"usage_count": gorm.Expr("usage_count + 1"), "last_used_at": now, "updated_at": now}).Error
	}
}

func ListAdminListingKeywords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	query := database.DB.Model(&model.ListingKeyword{})
	if platform := normalizeListingPlatform(c.Query("platform")); platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if stage := strings.TrimSpace(c.Query("stage")); stage != "" {
		query = query.Where("stage = ?", stage)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("LOWER(term) LIKE ?", "%"+strings.ToLower(keyword)+"%")
	}
	var total int64
	query.Count(&total)
	var records []model.ListingKeyword
	if err := query.Order("updated_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "listing_keyword_query_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": records, "total": total, "page": page, "page_size": pageSize})
}

func ExpandAdminListingKeywords(c *gin.Context) {
	var req struct {
		Platform    string   `json:"platform"`
		CountryCode string   `json:"country_code"`
		Category    string   `json:"category"`
		Terms       []string `json:"terms"`
		Text        string   `json:"text"`
		Source      string   `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_keyword_request"})
		return
	}
	platform := normalizeListingPlatform(req.Platform)
	terms := append([]string(nil), req.Terms...)
	terms = append(terms, splitListingKeywords(req.Text)...)
	terms = splitListingKeywords(strings.Join(terms, "\n"))
	if platform == "" || len(terms) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform_and_terms_required"})
		return
	}
	created := recordListingKeywordCandidates(platform, req.CountryCode, req.Category, strings.TrimSpace(req.Source), terms)
	c.JSON(http.StatusOK, gin.H{"created": created, "received": len(terms), "stage": "expanded"})
}

func UpdateAdminListingKeyword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_keyword_id"})
		return
	}
	var req struct {
		Term        *string   `json:"term"`
		CountryCode *string   `json:"country_code"`
		Category    *string   `json:"category"`
		Stage       *string   `json:"stage"`
		Tags        *[]string `json:"tags"`
		Notes       *string   `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_keyword_update"})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Term != nil {
		term := strings.ToLower(normalizeListingKeyword(*req.Term))
		if term == "" || len([]rune(term)) > 255 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_keyword_term"})
			return
		}
		updates["term"] = term
	}
	if req.CountryCode != nil {
		updates["country_code"] = strings.ToUpper(strings.TrimSpace(*req.CountryCode))
	}
	if req.Category != nil {
		updates["category"] = strings.TrimSpace(*req.Category)
	}
	if req.Tags != nil {
		tags := make([]string, 0, len(*req.Tags))
		seen := map[string]struct{}{}
		for _, value := range *req.Tags {
			tag := normalizeListingKeyword(value)
			if tag == "" {
				continue
			}
			key := strings.ToLower(tag)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		updates["tags"] = model.StringSlice(tags)
	}
	if req.Notes != nil {
		updates["notes"] = strings.TrimSpace(*req.Notes)
	}
	if req.Stage != nil {
		stage := strings.ToLower(strings.TrimSpace(*req.Stage))
		if !listingKeywordStages[stage] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_keyword_stage"})
			return
		}
		updates["stage"] = stage
		switch stage {
		case "library":
			updates["status"] = "active"
		case "rejected":
			updates["status"] = "rejected"
		default:
			updates["status"] = "candidate"
		}
	}
	result := database.DB.Model(&model.ListingKeyword{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "listing_keyword_update_failed"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing_keyword_not_found"})
		return
	}
	var record model.ListingKeyword
	database.DB.First(&record, id)
	c.JSON(http.StatusOK, record)
}

func DeleteAdminListingKeyword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_keyword_id"})
		return
	}
	result := database.DB.Delete(&model.ListingKeyword{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "listing_keyword_delete_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected > 0})
}
