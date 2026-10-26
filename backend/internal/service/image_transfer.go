package service

import (
	"log"
	"os"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"
)

type ImageTransferResult struct {
	ImageURL    string
	Transferred bool
	Updated     int64
}

func IsExternalImageURL(imageURL string) bool {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" || strings.HasPrefix(imageURL, "data:") || strings.HasPrefix(imageURL, "/") {
		return false
	}
	if !(strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://")) {
		return false
	}
	if domain := strings.TrimSpace(storage.OSSDomain); domain != "" && hasURLPrefix(imageURL, domain) {
		return false
	}
	bucket := strings.TrimSpace(os.Getenv("OSS_BUCKET"))
	endpoint := strings.TrimSpace(os.Getenv("OSS_ENDPOINT"))
	if bucket != "" && endpoint != "" {
		endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
		endpoint = strings.TrimRight(endpoint, "/")
		ossHost := bucket + "." + endpoint
		if hasURLPrefix(imageURL, "https://"+ossHost) ||
			hasURLPrefix(imageURL, "http://"+ossHost) {
			return false
		}
	}
	return true
}

func hasURLPrefix(imageURL string, prefix string) bool {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
	return imageURL == prefix || strings.HasPrefix(imageURL, prefix+"/")
}

func EnsureUserImageOSS(userID uint, imageURL string) (ImageTransferResult, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" || !IsExternalImageURL(imageURL) {
		return ImageTransferResult{ImageURL: imageURL}, nil
	}
	if !UserOwnsImageURL(userID, imageURL) {
		return ImageTransferResult{}, ErrImageNotOwned
	}
	return transferAndReplace(imageURL, func(oldURL, newURL string) int64 {
		return UpdateOwnedImageURL(userID, oldURL, newURL)
	})
}

func TransferExternalImageURL(imageURL string) (ImageTransferResult, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" || !IsExternalImageURL(imageURL) {
		return ImageTransferResult{ImageURL: imageURL}, nil
	}
	return transferAndReplace(imageURL, UpdateAllImageURL)
}

func transferAndReplace(imageURL string, replace func(string, string) int64) (ImageTransferResult, error) {
	ossURL, _, err := storage.SaveImageFromURL(imageURL)
	if err != nil {
		return ImageTransferResult{}, err
	}
	updated := replace(imageURL, ossURL)
	return ImageTransferResult{ImageURL: ossURL, Transferred: true, Updated: updated}, nil
}

var ErrImageNotOwned = errImageNotOwned{}

type errImageNotOwned struct{}

func (errImageNotOwned) Error() string { return "image url is not owned by user" }

func UserOwnsImageURL(userID uint, imageURL string) bool {
	if imageURL == "" {
		return false
	}

	var count int64
	database.DB.Model(&model.UsageLog{}).Where("user_id = ? AND image_url = ?", userID, imageURL).Count(&count)
	if count > 0 {
		return true
	}
	database.DB.Model(&model.ImageGenerationTask{}).Where("user_id = ? AND image_url = ?", userID, imageURL).Count(&count)
	if count > 0 {
		return true
	}
	database.DB.Model(&model.ImageBatchItem{}).
		Joins("JOIN image_batches ON image_batches.id = image_batch_items.batch_id").
		Where("image_batches.user_id = ? AND image_batch_items.image_url = ?", userID, imageURL).
		Count(&count)
	if count > 0 {
		return true
	}
	database.DB.Model(&model.SceneGenerationImage{}).
		Joins("JOIN scene_generation_tasks ON scene_generation_tasks.task_id = scene_generation_images.task_id_ref").
		Where("scene_generation_tasks.user_id = ? AND (scene_generation_images.image_url = ? OR scene_generation_images.candidate_image_url = ?)", userID, imageURL, imageURL).
		Count(&count)
	if count > 0 {
		return true
	}
	database.DB.Model(&model.ProductReplaceImage{}).
		Joins("JOIN product_replace_tasks ON product_replace_tasks.task_id = product_replace_images.task_id_ref").
		Where("product_replace_tasks.user_id = ? AND product_replace_images.image_url = ?", userID, imageURL).
		Count(&count)
	if count > 0 {
		return true
	}
	database.DB.Model(&model.EcomSuiteImage{}).
		Joins("JOIN ecom_suite_tasks ON ecom_suite_tasks.task_id = ecom_suite_images.task_id").
		Where("ecom_suite_tasks.user_id = ? AND ecom_suite_images.image_url = ?", userID, imageURL).
		Count(&count)
	return count > 0
}

func UpdateOwnedImageURL(userID uint, oldURL string, newURL string) int64 {
	return updateImageURL(oldURL, newURL, userID, true)
}

func UpdateAllImageURL(oldURL string, newURL string) int64 {
	return updateImageURL(oldURL, newURL, 0, false)
}

func updateImageURL(oldURL string, newURL string, userID uint, ownedOnly bool) int64 {
	now := time.Now()
	var updated int64

	res := database.DB.Model(&model.UsageLog{}).Where("image_url = ?", oldURL)
	if ownedOnly {
		res = res.Where("user_id = ?", userID)
	}
	updated += res.Updates(map[string]interface{}{"image_url": newURL, "updated_at": now}).RowsAffected

	res = database.DB.Model(&model.ImageGenerationTask{}).Where("image_url = ?", oldURL)
	if ownedOnly {
		res = res.Where("user_id = ?", userID)
	}
	updated += res.Updates(map[string]interface{}{"image_url": newURL, "updated_at": now}).RowsAffected

	updated += updateImageBatchItems(oldURL, newURL, now, userID, ownedOnly)
	updated += updateSceneImages("image_url", oldURL, newURL, now, userID, ownedOnly)
	updated += updateSceneImages("candidate_image_url", oldURL, newURL, now, userID, ownedOnly)
	updated += updateProductReplaceImages(oldURL, newURL, now, userID, ownedOnly)
	updated += updateEcomSuiteImages(oldURL, newURL, now, userID, ownedOnly)

	return updated
}

func updateImageBatchItems(oldURL string, newURL string, now time.Time, userID uint, ownedOnly bool) int64 {
	query := database.DB.Model(&model.ImageBatchItem{}).Where("image_url = ?", oldURL)
	if ownedOnly {
		query = query.Where("batch_id IN (?)", database.DB.Model(&model.ImageBatch{}).Select("id").Where("user_id = ?", userID))
	}
	return query.Updates(map[string]interface{}{"image_url": newURL, "updated_at": now}).RowsAffected
}

func updateSceneImages(column string, oldURL string, newURL string, now time.Time, userID uint, ownedOnly bool) int64 {
	query := database.DB.Model(&model.SceneGenerationImage{}).Where(column+" = ?", oldURL)
	if ownedOnly {
		query = query.Where("task_id_ref IN (?)", database.DB.Model(&model.SceneGenerationTask{}).Select("task_id").Where("user_id = ?", userID))
	}
	return query.Updates(map[string]interface{}{column: newURL, "updated_at": now}).RowsAffected
}

func updateProductReplaceImages(oldURL string, newURL string, now time.Time, userID uint, ownedOnly bool) int64 {
	query := database.DB.Model(&model.ProductReplaceImage{}).Where("image_url = ?", oldURL)
	if ownedOnly {
		query = query.Where("task_id_ref IN (?)", database.DB.Model(&model.ProductReplaceTask{}).Select("task_id").Where("user_id = ?", userID))
	}
	return query.Updates(map[string]interface{}{"image_url": newURL, "updated_at": now}).RowsAffected
}

func updateEcomSuiteImages(oldURL string, newURL string, now time.Time, userID uint, ownedOnly bool) int64 {
	query := database.DB.Model(&model.EcomSuiteImage{}).Where("image_url = ?", oldURL)
	if ownedOnly {
		query = query.Where("task_id IN (?)", database.DB.Model(&model.EcomSuiteTask{}).Select("task_id").Where("user_id = ?", userID))
	}
	return query.Updates(map[string]interface{}{"image_url": newURL, "updated_at": now}).RowsAffected
}

func LogTransferFailure(prefix string, imageURL string, err error) {
	if err != nil {
		log.Printf("[image-transfer] %s url=%s err=%v", prefix, imageURL, err)
	}
}
