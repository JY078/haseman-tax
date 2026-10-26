package handler

import (
	"net/http"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type buyerReviewInput struct {
	Brand        string  `json:"brand"`
	Country      string  `json:"country"`
	ParentASIN   string  `json:"parent_asin"`
	ChildASIN    string  `json:"child_asin"`
	Rating       float64 `json:"rating"`
	Name         string  `json:"name"`
	Date         string  `json:"date"`
	Title        string  `json:"title"`
	Content      string  `json:"content"`
	Link         string  `json:"link"`
	Image        string  `json:"image"`
	ReviewImages *string `json:"review_images"`
	Video        bool    `json:"video"`
	VideoUpload  string  `json:"video_upload"`
}

func ListBuyerReviews(c *gin.Context) {
	var reviews []model.BuyerReview
	if err := database.DB.Order("review_date DESC, id DESC").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "评论加载失败"})
		return
	}
	keys := make([]string, 0, len(reviews))
	seen := make(map[string]bool)
	for _, review := range reviews {
		key := strings.ToUpper(strings.TrimSpace(review.ChildASIN))
		if key != "" && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	var products []model.ProductSKU
	for start := 0; start < len(keys); start += 1000 {
		end := min(start+1000, len(keys))
		var batch []model.ProductSKU
		if err := database.DB.Select("asin, image_url").Where("UPPER(TRIM(asin)) IN ?", keys[start:end]).Find(&batch).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "产品图片加载失败"})
			return
		}
		products = append(products, batch...)
	}
	matchBuyerReviewProductImages(reviews, products)
	c.JSON(http.StatusOK, gin.H{"data": reviews})
}

func matchBuyerReviewProductImages(reviews []model.BuyerReview, products []model.ProductSKU) {
	asinImages := make(map[string]string)
	add := func(images map[string]string, key, image string) {
		key = strings.ToUpper(strings.TrimSpace(key))
		if key == "" {
			return
		}
		image = strings.TrimSpace(image)
		if previous, exists := images[key]; exists && previous != image {
			// Conflicting catalog matches must not show an arbitrary product image.
			images[key] = ""
		} else {
			images[key] = image
		}
	}
	for _, product := range products {
		add(asinImages, product.ASIN, product.ImageURL)
	}
	for index := range reviews {
		key := strings.ToUpper(strings.TrimSpace(reviews[index].ChildASIN))
		reviews[index].ProductImageURL = asinImages[key]
	}
}

func ImportBuyerReviews(c *gin.Context) {
	var input []buyerReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论数据格式无效"})
		return
	}
	if len(input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可导入的评论"})
		return
	}
	if len(input) > 50000 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "单次最多导入 50000 条评论"})
		return
	}
	for index := range input {
		row := &input[index]
		if row.ReviewImages != nil {
			links, err := reviewMediaLinks(*row.ReviewImages)
			if err != nil {
				c.JSON(400, gin.H{"error": "图片链接列必须填写有效链接"})
				return
			}
			row.ReviewImages = &links
		}
		links, err := reviewMediaLinks(row.VideoUpload)
		if err != nil {
			c.JSON(400, gin.H{"error": "视频附件必须是链接；嵌入视频请使用 XLSX 模板导入"})
			return
		}
		row.VideoUpload = links
		row.Video = row.Video || links != ""
	}
	if err := saveBuyerReviews(input); err != nil {
		c.JSON(500, gin.H{"error": "评论导入失败"})
		return
	}
	c.JSON(200, gin.H{"imported": len(input)})
}

func saveBuyerReviews(input []buyerReviewInput) error {
	reviews := make([]model.BuyerReview, 0, len(input))
	reviewsWithImages := make([]model.BuyerReview, 0, len(input))
	seen := make(map[string]bool, len(input))
	for _, row := range input {
		review := model.BuyerReview{Fingerprint: model.BuyerReviewFingerprint(row.Link, row.ChildASIN, row.Date, row.Name, row.Title, row.Content), Brand: row.Brand, Country: row.Country, ParentASIN: row.ParentASIN, ChildASIN: row.ChildASIN, Rating: row.Rating, Name: row.Name, ReviewDate: row.Date, Title: row.Title, Content: row.Content, Link: row.Link, Image: row.Image, Video: row.Video, VideoUpload: row.VideoUpload}
		if row.ReviewImages != nil {
			review.ReviewImages = *row.ReviewImages
		}
		// A normalized link can make two rows in one import collide; replace
		// the earlier row so ON CONFLICT never targets one fingerprint twice.
		// Sequential upserts never clear review_images, so inherit it when the
		// replacement row has none of its own.
		if seen[review.Fingerprint] {
			var previous model.BuyerReview
			for index := range reviews {
				if reviews[index].Fingerprint == review.Fingerprint {
					previous = reviews[index]
					reviews = append(reviews[:index], reviews[index+1:]...)
					break
				}
			}
			for index := range reviewsWithImages {
				if reviewsWithImages[index].Fingerprint == review.Fingerprint {
					previous = reviewsWithImages[index]
					reviewsWithImages = append(reviewsWithImages[:index], reviewsWithImages[index+1:]...)
					break
				}
			}
			if row.ReviewImages == nil && previous.ReviewImages != "" {
				review.ReviewImages = previous.ReviewImages
			}
		}
		seen[review.Fingerprint] = true
		if row.ReviewImages != nil || review.ReviewImages != "" {
			reviewsWithImages = append(reviewsWithImages, review)
		} else {
			reviews = append(reviews, review)
		}
	}
	updates := []string{"brand", "country", "parent_asin", "child_asin", "rating", "name", "review_date", "title", "content", "link", "image", "video", "video_upload", "updated_at"}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		for _, batch := range []struct {
			rows    []model.BuyerReview
			columns []string
		}{{reviews, updates}, {reviewsWithImages, append(append([]string{}, updates...), "review_images")}} {
			if len(batch.rows) == 0 {
				continue
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "fingerprint"}}, DoUpdates: clause.AssignmentColumns(batch.columns)}).CreateInBatches(&batch.rows, 500).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func UpdateBuyerReviewVideoLink(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论编号无效"})
		return
	}
	var input struct {
		VideoLink string `json:"video_link" binding:"max=5000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "视频链接无效"})
		return
	}
	result := database.DB.Model(&model.BuyerReview{}).Where("id = ?", id).Update("video_link", strings.TrimSpace(input.VideoLink))
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "视频链接保存失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
		return
	}
	c.Status(http.StatusNoContent)
}
