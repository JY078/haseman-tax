package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"advertising-image/backend/internal/cache"
	"advertising-image/backend/internal/config"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/handler"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/router"
	"advertising-image/backend/internal/seeder"
	"advertising-image/backend/internal/service"
	"advertising-image/backend/internal/storage"
	"advertising-image/backend/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	logKeepDays                     = 5
	sceneTaskRunningCleanupInterval = 1 * time.Minute
	sceneTaskRunningCleanupTimeout  = 15 * time.Minute
	aiImageTaskCleanupInterval      = 1 * time.Minute
	aiImageTaskCleanupTimeout       = 45 * time.Minute
)

type dailyLogger struct {
	mu     sync.Mutex
	file   *os.File
	logDir string
}

var globalLogger *dailyLogger

func (d *dailyLogger) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

func (d *dailyLogger) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.file != nil {
		return d.file.Write(p)
	}
	return len(p), nil
}

func (d *dailyLogger) rotate() {
	d.mu.Lock()
	defer d.mu.Unlock()

	cleanOldLogs(d.logDir, logKeepDays)

	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(d.logDir, "app-"+today+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("WARNING: cannot open log file %s: %v", logPath, err)
		return
	}
	old := d.file
	d.file = f
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("Log file: %s", logPath)
	if old != nil {
		old.Close()
	}
}

func cleanOldLogs(logDir string, keepDays int) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "app-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		dateStr := strings.TrimSuffix(strings.TrimPrefix(name, "app-"), ".log")
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(logDir, name))
			log.Printf("Removed old log: %s", name)
		}
	}
}

func initLogger() io.Closer {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("WARNING: cannot create logs dir: %v", err)
		return nil
	}
	dl := &dailyLogger{logDir: logDir}
	dl.rotate()
	globalLogger = dl
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 1, 0, now.Location())
			time.Sleep(time.Until(next))
			dl.rotate()
		}
	}()
	return dl
}

// cleanupExpiredRunningSceneTasks 清理场景图生成任务
func cleanupExpiredRunningSceneTasks() {
	cutoff := time.Now().Add(-sceneTaskRunningCleanupTimeout)

	// 情况1: running 状态超过15分钟的任务 → 清理整个任务
	var runningTasks []model.SceneGenerationTask
	if err := database.DB.Where("status = ? AND updated_at < ?", "running", cutoff).Find(&runningTasks).Error; err != nil {
		log.Printf("[cleanup] 查询超时任务失败: %v", err)
		return
	}

	// 情况2: failed 状态的任务 → 清理整个任务
	var failedTasks []model.SceneGenerationTask
	if err := database.DB.Where("status = ?", "failed").Find(&failedTasks).Error; err != nil {
		log.Printf("[cleanup] 查询失败任务失败: %v", err)
		return
	}

	// 清理情况1的整个任务
	for _, task := range runningTasks {
		tx := database.DB.Begin()
		if err := tx.Where("task_id_ref = ?", task.TaskID).Delete(&model.SceneGenerationImage{}).Error; err != nil {
			tx.Rollback()
			log.Printf("[cleanup] 删除任务 %s 的关联图片失败: %v", task.TaskID, err)
			continue
		}
		if err := tx.Delete(&task).Error; err != nil {
			tx.Rollback()
			log.Printf("[cleanup] 删除任务 %s 失败: %v", task.TaskID, err)
			continue
		}
		tx.Commit()
		log.Printf("[cleanup] 已清理超时任务: %s (status=%s)", task.TaskID, task.Status)
	}

	// 清理情况2的整个任务
	for _, task := range failedTasks {
		tx := database.DB.Begin()
		if err := tx.Where("task_id_ref = ?", task.TaskID).Delete(&model.SceneGenerationImage{}).Error; err != nil {
			tx.Rollback()
			log.Printf("[cleanup] 删除任务 %s 的关联图片失败: %v", task.TaskID, err)
			continue
		}
		if err := tx.Delete(&task).Error; err != nil {
			tx.Rollback()
			log.Printf("[cleanup] 删除任务 %s 失败: %v", task.TaskID, err)
			continue
		}
		tx.Commit()
		log.Printf("[cleanup] 已清理失败任务: %s (status=%s)", task.TaskID, task.Status)
	}

	// 情况3: 精细清理 - 只清理失败的图片，保留成功的（适用于其他状态的任务）
	cleanupFailedSceneImages(cutoff)
}

// cleanupFailedSceneImages 清理失败的场景图片，保留成功的
func cleanupFailedSceneImages(cutoff time.Time) {
	// 找到所有有图片的任务
	var tasks []model.SceneGenerationTask
	if err := database.DB.Find(&tasks).Error; err != nil {
		return
	}

	for _, task := range tasks {
		// 跳过已清理的情况1和情况2的任务
		if task.Status == "running" && task.UpdatedAt.Before(cutoff) {
			continue
		}
		if task.Status == "failed" {
			continue
		}

		// 获取该任务的所有图片
		var images []model.SceneGenerationImage
		if err := database.DB.Where("task_id_ref = ?", task.TaskID).Find(&images).Error; err != nil {
			continue
		}

		var toDeleteIDs []uint
		var successCount, failCount int

		for _, img := range images {
			// 成功的图片：ImageURL 不为空，或者有 ready 的候选图片
			isSuccess := img.ImageURL != ""
			isReadyCandidate := img.CandidateStatus == "ready" && img.CandidateImageURL != ""

			if isSuccess || isReadyCandidate {
				successCount++
				continue
			}

			// 失败的图片：ImageURL 为空 且 没有 ready 的候选图片
			// 或者有错误信息
			// 或者 generating 超时
			if img.CandidateErrorMessage != "" {
				failCount++
				toDeleteIDs = append(toDeleteIDs, img.ID)
			} else if img.CandidateStatus == "generating" && img.UpdatedAt.Before(cutoff) {
				failCount++
				toDeleteIDs = append(toDeleteIDs, img.ID)
			} else if img.ImageURL == "" && img.CandidateStatus != "ready" {
				failCount++
				toDeleteIDs = append(toDeleteIDs, img.ID)
			}
		}

		if len(toDeleteIDs) > 0 {
			if len(toDeleteIDs) == len(images) {
				// 所有图片都失败了，删除整个任务
				tx := database.DB.Begin()
				if err := tx.Where("task_id_ref = ?", task.TaskID).Delete(&model.SceneGenerationImage{}).Error; err != nil {
					tx.Rollback()
					continue
				}
				if err := tx.Delete(&task).Error; err != nil {
					tx.Rollback()
					continue
				}
				tx.Commit()
				log.Printf("[cleanup] 已清理全失败场景任务: %s (图片数=%d)", task.TaskID, len(images))
			} else {
				// 只删除失败的图片
				result := database.DB.Where("id IN ?", toDeleteIDs).Delete(&model.SceneGenerationImage{})
				if result.Error != nil {
					log.Printf("[cleanup] 删除场景 %s 的失败图片失败: %v", task.TaskID, result.Error)
				} else {
					log.Printf("[cleanup] 已清理场景 %s 的失败图片: %d 个失败，保留 %d 个成功", task.TaskID, failCount, successCount)
				}
			}
		}
	}
}

// startSceneTaskRunningCleanup 启动场景任务超时清理定时任务
func startSceneTaskRunningCleanup() {
	// 延迟5分钟启动，让应用先正常启动
	time.AfterFunc(5*time.Minute, func() {
		cleanupExpiredRunningSceneTasks()
	})

	// 定时执行（每5分钟检查一次）
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			cleanupExpiredRunningSceneTasks()
		}
	}()
	log.Printf("场景任务超时清理已启动（延迟5分钟后首次清理，之后每5分钟检查一次）")
}

// cleanupExpiredImageGenerationTasks 清理超过15分钟仍处于 running 状态的任务，failed 状态的任务，以及 done 但 image_url 为空的任务
func cleanupExpiredImageGenerationTasks() {
	cutoff := time.Now().Add(-aiImageTaskCleanupTimeout)

	// 清理 running 状态且超时的任务
	var runningTasks []model.ImageGenerationTask
	if err := database.DB.Where("status = ? AND updated_at < ?", "running", cutoff).Find(&runningTasks).Error; err != nil {
		log.Printf("[image-task-cleanup] 查询超时任务失败: %v", err)
		return
	}

	// 清理所有 failed 状态的任务（不限制时间）
	var failedTasks []model.ImageGenerationTask
	if err := database.DB.Where("status = ?", "failed").Find(&failedTasks).Error; err != nil {
		log.Printf("[image-task-cleanup] 查询失败任务失败: %v", err)
		return
	}

	// 清理 done 状态但 image_url 为空的任务（数据不一致，需要重试）
	var emptyImageTasks []model.ImageGenerationTask
	if err := database.DB.Where("status = ? AND image_url = ?", "done", "").Find(&emptyImageTasks).Error; err != nil {
		log.Printf("[image-task-cleanup] 查询空图片任务失败: %v", err)
		return
	}

	var allTasks []model.ImageGenerationTask
	allTasks = append(allTasks, runningTasks...)
	allTasks = append(allTasks, failedTasks...)
	allTasks = append(allTasks, emptyImageTasks...)

	if len(allTasks) == 0 {
		return
	}

	log.Printf("[image-task-cleanup] 发现 %d 个超时任务，%d 个失败任务，%d 个空图片任务，开始清理...", len(runningTasks), len(failedTasks), len(emptyImageTasks))

	for _, task := range allTasks {
		tx := database.DB.Begin()

		// 删除任务
		if err := tx.Delete(&task).Error; err != nil {
			tx.Rollback()
			log.Printf("[image-task-cleanup] 删除任务 %s 失败: %v", task.TaskID, err)
			continue
		}

		tx.Commit()
		var reason string
		if task.Status == "running" {
			reason = "超时"
		} else if task.Status == "failed" {
			reason = "失败"
		} else {
			reason = "空图片"
		}
		log.Printf("[image-task-cleanup] 已清理任务: %s (status=%s, 原因=%s)", task.TaskID, task.Status, reason)
	}

	log.Printf("[image-task-cleanup] 本次清理完成，共清理 %d 个任务（超时=%d, 失败=%d, 空图片=%d）",
		len(allTasks), len(runningTasks), len(failedTasks), len(emptyImageTasks))
}

// cleanupExpiredImageBatches 清理 ImageBatch
func cleanupExpiredImageBatches() {
	cutoff := time.Now().Add(-aiImageTaskCleanupTimeout)

	// 情况1: running 状态超过15分钟的批次 → 清理整个批次
	var runningBatches []model.ImageBatch
	if err := database.DB.Where("status = ? AND updated_at < ?", "running", cutoff).Find(&runningBatches).Error; err != nil {
		log.Printf("[image-batch-cleanup] 查询超时批次失败: %v", err)
		return
	}

	// 情况2: done 状态且有失败或超时 items 的批次 → 精细清理
	var doneBatches []model.ImageBatch
	if err := database.DB.Where("status = ?", "done").Find(&doneBatches).Error; err != nil {
		log.Printf("[image-batch-cleanup] 查询已完成批次失败: %v", err)
		return
	}

	// 清理情况1的批次
	for _, batch := range runningBatches {
		tx := database.DB.Begin()
		if err := tx.Where("batch_id = ?", batch.ID).Delete(&model.ImageBatchItem{}).Error; err != nil {
			tx.Rollback()
			log.Printf("[image-batch-cleanup] 删除批次 %d 的关联项目失败: %v", batch.ID, err)
			continue
		}
		if err := tx.Delete(&batch).Error; err != nil {
			tx.Rollback()
			log.Printf("[image-batch-cleanup] 删除批次 %d 失败: %v", batch.ID, err)
			continue
		}
		tx.Commit()
		log.Printf("[image-batch-cleanup] 已清理超时批次: %s (id=%d)", batch.BatchKey, batch.ID)
	}

	// 清理情况2的批次：精细处理
	for _, batch := range doneBatches {
		var totalCount, doneCount, errorCount, generatingCount int64
		database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ?", batch.ID).Count(&totalCount)
		database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ? AND status = ?", batch.ID, "done").Count(&doneCount)
		database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ? AND status = ?", batch.ID, "error").Count(&errorCount)
		database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ? AND status = ?", batch.ID, "generating").Count(&generatingCount)

		if totalCount == 0 {
			continue
		}

		if errorCount == totalCount {
			// 所有 items 都是失败的 → 清理整个批次
			tx := database.DB.Begin()
			if err := tx.Where("batch_id = ?", batch.ID).Delete(&model.ImageBatchItem{}).Error; err != nil {
				tx.Rollback()
				log.Printf("[image-batch-cleanup] 删除批次 %d 的关联项目失败: %v", batch.ID, err)
				continue
			}
			if err := tx.Delete(&batch).Error; err != nil {
				tx.Rollback()
				log.Printf("[image-batch-cleanup] 删除批次 %d 失败: %v", batch.ID, err)
				continue
			}
			tx.Commit()
			log.Printf("[image-batch-cleanup] 已清理全失败批次: %s (id=%d, total=%d)", batch.BatchKey, batch.ID, totalCount)
		} else {
			// 有成功的或有生成中的 → 清理失败的 + 超时的 generating + done 但无图片的
			cleanCount := int64(0)

			// 清理失败的 items
			if errorCount > 0 {
				result := database.DB.Where("batch_id = ? AND status = ?", batch.ID, "error").Delete(&model.ImageBatchItem{})
				if result.Error != nil {
					log.Printf("[image-batch-cleanup] 删除批次 %d 的失败 items 失败: %v", batch.ID, result.Error)
				} else {
					cleanCount += result.RowsAffected
				}
			}

			// 清理超时的 generating items（超过15分钟）
			if generatingCount > 0 {
				result := database.DB.Where("batch_id = ? AND status = ? AND updated_at < ?", batch.ID, "generating", cutoff).Delete(&model.ImageBatchItem{})
				if result.Error != nil {
					log.Printf("[image-batch-cleanup] 删除批次 %d 的超时 generating items 失败: %v", batch.ID, result.Error)
				} else {
					cleanCount += result.RowsAffected
				}
			}

			// 清理 done 状态但 image_url 为空的项目（数据不一致，包括空字符串和 NULL）
			var emptyImageCount int64
			database.DB.Model(&model.ImageBatchItem{}).Where("batch_id = ? AND status = ? AND (image_url = '' OR image_url IS NULL)", batch.ID, "done").Count(&emptyImageCount)
			if emptyImageCount > 0 {
				result := database.DB.Where("batch_id = ? AND status = ? AND (image_url = '' OR image_url IS NULL)", batch.ID, "done").Delete(&model.ImageBatchItem{})
				if result.Error != nil {
					log.Printf("[image-batch-cleanup] 删除批次 %d 的空图片 items 失败: %v", batch.ID, result.Error)
				} else {
					cleanCount += result.RowsAffected
					log.Printf("[image-batch-cleanup] 已清理批次 %s 的空图片 items: %d 个", batch.BatchKey, result.RowsAffected)
				}
			}

			if cleanCount > 0 {
				log.Printf("[image-batch-cleanup] 已清理批次 %s 的失败/超时 items: %d 个（done=%d, error=%d, generating=%d）", batch.BatchKey, cleanCount, doneCount, errorCount, generatingCount)
			}
		}
	}
}

// startImageGenerationTaskCleanup 启动 AI 图像生成任务超时清理定时任务
func startImageGenerationTaskCleanup() {
	// 延迟5分钟启动，让应用先正常启动
	time.AfterFunc(5*time.Minute, func() {
		cleanupExpiredImageGenerationTasks()
		cleanupExpiredImageBatches()
	})

	// 定时执行（每5分钟检查一次）
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			cleanupExpiredImageGenerationTasks()
			cleanupExpiredImageBatches()
		}
	}()
	log.Printf("AI图像生成任务超时清理已启动（延迟5分钟后首次清理，之后每5分钟检查一次）")
}

// migrateExternalImageURLsToOSS 每分钟检测任务中外部 CDN 图片，转存到 OSS
func migrateAllExternalImageURLsToOSS() {
	urlSet := make(map[string]struct{})
	addURL := func(url string) {
		url = strings.TrimSpace(url)
		if service.IsExternalImageURL(url) {
			urlSet[url] = struct{}{}
		}
	}

	var usageLogs []model.UsageLog
	database.DB.Where("image_url != ''").Find(&usageLogs)
	for _, item := range usageLogs {
		addURL(item.ImageURL)
	}

	var imageTasks []model.ImageGenerationTask
	database.DB.Where("image_url != ''").Find(&imageTasks)
	for _, item := range imageTasks {
		addURL(item.ImageURL)
	}

	var batchItems []model.ImageBatchItem
	database.DB.Where("image_url != ''").Find(&batchItems)
	for _, item := range batchItems {
		addURL(item.ImageURL)
	}

	var sceneImages []model.SceneGenerationImage
	database.DB.Where("image_url != '' OR candidate_image_url != ''").Find(&sceneImages)
	for _, item := range sceneImages {
		addURL(item.ImageURL)
		addURL(item.CandidateImageURL)
	}

	var replaceImages []model.ProductReplaceImage
	database.DB.Where("image_url != ''").Find(&replaceImages)
	for _, item := range replaceImages {
		addURL(item.ImageURL)
	}

	var ecomImages []model.EcomSuiteImage
	database.DB.Where("image_url != ''").Find(&ecomImages)
	for _, item := range ecomImages {
		addURL(item.ImageURL)
	}

	log.Printf("[migrate-image] 发现 %d 个外部图片 URL 待迁移", len(urlSet))
	successCount := 0
	failedCount := 0
	for imageURL := range urlSet {
		result, err := service.TransferExternalImageURL(imageURL)
		if err != nil {
			failedCount++
			service.LogTransferFailure("nightly migrate", imageURL, err)
			continue
		}
		if result.Transferred {
			successCount++
			log.Printf("[migrate-image] 迁移成功 updated=%d: %s -> %s", result.Updated, imageURL, result.ImageURL)
		}
	}
	log.Printf("[migrate-image] 夜间迁移完成: 成功 %d, 失败 %d", successCount, failedCount)
}

func migrateExternalImageURLsToOSS() {
	// 防止并发执行导致内存/网络过载
	if !migrateRunning.CompareAndSwap(false, true) {
		log.Printf("[migrate-image] 上一次迁移任务尚未完成，跳过本次执行")
		return
	}
	defer migrateRunning.Store(false)

	migrateAllExternalImageURLsToOSS()
	return

	const cdnPrefix = "https://cdn-hk.51sux.com"
	// 只转存指定日期及之后的图片，避免处理过多历史数据
	migrateCutoff := time.Date(2026, 6, 9, 0, 0, 0, 0, time.Local)

	// 1. 批量迁移 image_batch_items
	batchMigrateImageURLs[model.ImageBatchItem](cdnPrefix,
		func() []model.ImageBatchItem {
			var items []model.ImageBatchItem
			database.DB.Where("status = ? AND image_url LIKE ? AND updated_at >= ?", "done", cdnPrefix+"%", migrateCutoff).Find(&items)
			return items
		},
		func(item model.ImageBatchItem) string { return item.ImageURL },
		func(id uint, url string) {
			database.DB.Model(&model.ImageBatchItem{}).Where("id = ?", id).
				Updates(map[string]interface{}{"image_url": url, "updated_at": time.Now()})
		},
		"image_batch_items",
	)

	// 2. 批量迁移 scene_generation_images 的 image_url
	batchMigrateImageURLs[model.SceneGenerationImage](cdnPrefix,
		func() []model.SceneGenerationImage {
			var items []model.SceneGenerationImage
			database.DB.Where("image_url LIKE ? AND updated_at >= ?", cdnPrefix+"%", migrateCutoff).Find(&items)
			return items
		},
		func(item model.SceneGenerationImage) string { return item.ImageURL },
		func(id uint, url string) {
			database.DB.Model(&model.SceneGenerationImage{}).Where("id = ?", id).
				Updates(map[string]interface{}{"image_url": url, "updated_at": time.Now()})
		},
		"scene_generation_images",
	)

	// 3. 批量迁移 scene_generation_images 的 candidate_image_url
	batchMigrateCandidateImageURLs(cdnPrefix, migrateCutoff)

	// 4. 批量迁移 usage_log 的 image_url
	batchMigrateImageURLs[model.UsageLog](cdnPrefix,
		func() []model.UsageLog {
			var items []model.UsageLog
			database.DB.Where("image_url LIKE ? AND image_url != '' AND created_at >= ?", cdnPrefix+"%", migrateCutoff).Find(&items)
			return items
		},
		func(item model.UsageLog) string { return item.ImageURL },
		func(id uint, url string) {
			database.DB.Model(&model.UsageLog{}).Where("id = ?", id).
				Updates(map[string]interface{}{"image_url": url, "updated_at": time.Now()})
		},
		"usage_log",
	)
}

// 并发配置
const (
	migrateDownloadConcurrency = 10
	migrateUploadConcurrency   = 10
	migrateMaxRetries          = 3
	migrateBatchSize           = 50
)

// migrateRunning 标记迁移任务是否正在运行，防止并发执行导致内存/网络过载
var migrateRunning atomic.Bool

// downloadResult 下载结果
type downloadResult struct {
	id       uint
	url      string
	data     []byte
	mimeType string
	err      error
}

// uploadResult 上传结果
type uploadResult struct {
	id     uint
	url    string
	ossURL string
	err    error
}

// batchMigrateImageURLs 批量迁移表格图片URL（优化版：分离下载/上传管道，增大并发，添加重试）
func batchMigrateImageURLs[T any](cdnPrefix string, query func() []T, getURL func(T) string, update func(uint, string), tableName string) {
	items := query()
	if len(items) == 0 {
		return
	}

	log.Printf("[migrate-image] 发现 %d 个外部CDN图片待迁移到OSS (%s)...", len(items), tableName)

	// 收集需要迁移的项
	type migrationItem struct {
		id  uint
		url string
	}
	toMigrate := make([]migrationItem, 0, len(items))

	for _, item := range items {
		imageURL := getURL(item)
		if imageURL == "" || !strings.HasPrefix(imageURL, cdnPrefix) {
			continue
		}
		var id uint
		switch v := any(item).(type) {
		case model.ImageBatchItem:
			id = v.ID
		case model.SceneGenerationImage:
			id = v.ID
		case model.UsageLog:
			id = v.ID
		}
		toMigrate = append(toMigrate, migrationItem{id: id, url: imageURL})
	}

	if len(toMigrate) == 0 {
		return
	}

	log.Printf("[migrate-image] %s 实际待迁移: %d 个", tableName, len(toMigrate))

	// 下载阶段
	downloadSem := make(chan struct{}, migrateDownloadConcurrency)
	downloadWg := sync.WaitGroup{}
	downloadResults := make(chan downloadResult, len(toMigrate))

	for _, m := range toMigrate {
		downloadWg.Add(1)
		downloadSem <- struct{}{}
		go func(item migrationItem) {
			defer downloadWg.Done()
			defer func() { <-downloadSem }()

			var data []byte
			var mimeType string
			var dlErr error

			// 下载（带重试）
			for attempt := 1; attempt <= migrateMaxRetries; attempt++ {
				data, mimeType, dlErr = storage.DownloadImageToMemory(item.url)
				if dlErr == nil {
					break
				}
				if attempt < migrateMaxRetries {
					log.Printf("[migrate-image] %s ID=%d 下载重试 %d/%d: %v", tableName, item.id, attempt, migrateMaxRetries, dlErr)
				}
			}

			downloadResults <- downloadResult{
				id:       item.id,
				url:      item.url,
				data:     data,
				mimeType: mimeType,
				err:      dlErr,
			}
		}(m)
	}

	go func() {
		downloadWg.Wait()
		close(downloadResults)
	}()

	// 上传阶段
	uploadSem := make(chan struct{}, migrateUploadConcurrency)
	uploadWg := sync.WaitGroup{}
	uploadResults := make(chan uploadResult, len(toMigrate))
	uploadTasks := make(chan downloadResult, len(toMigrate))

	// 启动上传 worker
	for i := 0; i < migrateUploadConcurrency; i++ {
		uploadWg.Add(1)
		go func() {
			defer uploadWg.Done()
			for dl := range uploadTasks {
				if dl.err != nil {
					log.Printf("[migrate-image] %s ID=%d 下载失败: %v", tableName, dl.id, dl.err)
					uploadResults <- uploadResult{id: dl.id, url: dl.url, err: dl.err}
					continue
				}

				uploadSem <- struct{}{}
				go func(d downloadResult) {
					defer func() { <-uploadSem }()

					// 上传 OSS（带重试）
					var ossURL string
					var upErr error
					for attempt := 1; attempt <= migrateMaxRetries; attempt++ {
						ossURL, upErr = storage.UploadImageDataToOSS(d.data, d.mimeType, storage.UploadDir)
						if upErr == nil {
							break
						}
						if attempt < migrateMaxRetries {
							log.Printf("[migrate-image] %s ID=%d 上传OSS重试 %d/%d: %v", tableName, d.id, attempt, migrateMaxRetries, upErr)
						}
					}

					if upErr != nil {
						log.Printf("[migrate-image] %s ID=%d 上传OSS失败: %v", tableName, d.id, upErr)
						uploadResults <- uploadResult{id: d.id, url: d.url, err: upErr}
						return
					}

					uploadResults <- uploadResult{id: d.id, url: d.url, ossURL: ossURL}
				}(dl)
			}
		}()
	}

	go func() {
		uploadWg.Wait()
		close(uploadResults)
		close(uploadTasks)
	}()

	// 发送下载结果到上传任务
	go func() {
		for dl := range downloadResults {
			uploadTasks <- dl
		}
	}()

	// 收集结果并批量更新数据库
	type updateItem struct {
		id  uint
		url string
	}
	successItems := make([]updateItem, 0)
	failedCount := 0
	for up := range uploadResults {
		if up.err == nil {
			successItems = append(successItems, updateItem{id: up.id, url: up.ossURL})
		} else {
			failedCount++
		}
	}

	// 批量更新数据库（每 migrateBatchSize 条一批）
	for i := 0; i < len(successItems); i += migrateBatchSize {
		end := i + migrateBatchSize
		if end > len(successItems) {
			end = len(successItems)
		}
		batch := successItems[i:end]
		for _, item := range batch {
			update(item.id, item.url)
		}
	}

	// 统计成功日志
	successCount := len(successItems)
	for _, item := range successItems {
		for _, m := range toMigrate {
			if m.id == item.id {
				log.Printf("[migrate-image] %s ID=%d 迁移成功: %s -> %s", tableName, item.id, m.url, item.url)
				break
			}
		}
	}

	log.Printf("[migrate-image] %s 批量迁移完成: 成功 %d, 失败 %d", tableName, successCount, failedCount)
}

// batchMigrateCandidateImageURLs 批量迁移 scene_generation_images 的候选图片URL（优化版）
func batchMigrateCandidateImageURLs(cdnPrefix string, migrateCutoff time.Time) {
	var items []model.SceneGenerationImage
	database.DB.Where("candidate_image_url LIKE ? AND updated_at >= ?", cdnPrefix+"%", migrateCutoff).Find(&items)
	if len(items) == 0 {
		return
	}

	log.Printf("[migrate-image] 发现 %d 个外部CDN候选图片待迁移到OSS (scene_generation_images)...", len(items))

	type candidateItem struct {
		id  uint
		url string
	}
	toMigrate := make([]candidateItem, 0, len(items))
	for _, item := range items {
		if item.CandidateImageURL != "" && strings.HasPrefix(item.CandidateImageURL, cdnPrefix) {
			toMigrate = append(toMigrate, candidateItem{id: item.ID, url: item.CandidateImageURL})
		}
	}

	if len(toMigrate) == 0 {
		return
	}

	log.Printf("[migrate-image] scene_generation_images candidate 实际待迁移: %d 个", len(toMigrate))

	// 下载阶段
	downloadSem := make(chan struct{}, migrateDownloadConcurrency)
	downloadWg := sync.WaitGroup{}
	downloadResults := make(chan downloadResult, len(toMigrate))

	for _, m := range toMigrate {
		downloadWg.Add(1)
		downloadSem <- struct{}{}
		go func(item candidateItem) {
			defer downloadWg.Done()
			defer func() { <-downloadSem }()

			var data []byte
			var mimeType string
			var dlErr error

			for attempt := 1; attempt <= migrateMaxRetries; attempt++ {
				data, mimeType, dlErr = storage.DownloadImageToMemory(item.url)
				if dlErr == nil {
					break
				}
				if attempt < migrateMaxRetries {
					log.Printf("[migrate-image] scene_generation_images ID=%d candidate 下载重试 %d/%d: %v", item.id, attempt, migrateMaxRetries, dlErr)
				}
			}

			downloadResults <- downloadResult{
				id:       item.id,
				url:      item.url,
				data:     data,
				mimeType: mimeType,
				err:      dlErr,
			}
		}(m)
	}

	go func() {
		downloadWg.Wait()
		close(downloadResults)
	}()

	// 上传阶段
	uploadSem := make(chan struct{}, migrateUploadConcurrency)
	uploadWg := sync.WaitGroup{}
	uploadResults := make(chan uploadResult, len(toMigrate))
	uploadTasks := make(chan downloadResult, len(toMigrate))

	for i := 0; i < migrateUploadConcurrency; i++ {
		uploadWg.Add(1)
		go func() {
			defer uploadWg.Done()
			for dl := range uploadTasks {
				if dl.err != nil {
					log.Printf("[migrate-image] scene_generation_images ID=%d candidate 下载失败: %v", dl.id, dl.err)
					uploadResults <- uploadResult{id: dl.id, url: dl.url, err: dl.err}
					continue
				}

				uploadSem <- struct{}{}
				go func(d downloadResult) {
					defer func() { <-uploadSem }()

					var ossURL string
					var upErr error
					for attempt := 1; attempt <= migrateMaxRetries; attempt++ {
						ossURL, upErr = storage.UploadImageDataToOSS(d.data, d.mimeType, storage.UploadDir)
						if upErr == nil {
							break
						}
						if attempt < migrateMaxRetries {
							log.Printf("[migrate-image] scene_generation_images ID=%d candidate 上传OSS重试 %d/%d: %v", d.id, attempt, migrateMaxRetries, upErr)
						}
					}

					if upErr != nil {
						log.Printf("[migrate-image] scene_generation_images ID=%d candidate 上传OSS失败: %v", d.id, upErr)
						uploadResults <- uploadResult{id: d.id, url: d.url, err: upErr}
						return
					}

					uploadResults <- uploadResult{id: d.id, url: d.url, ossURL: ossURL}
				}(dl)
			}
		}()
	}

	go func() {
		uploadWg.Wait()
		close(uploadResults)
		close(uploadTasks)
	}()

	go func() {
		for dl := range downloadResults {
			uploadTasks <- dl
		}
	}()

	// 收集结果并批量更新
	type updateItem struct {
		id  uint
		url string
	}
	successItems := make([]updateItem, 0)
	failedCount := 0

	for up := range uploadResults {
		if up.err == nil {
			successItems = append(successItems, updateItem{id: up.id, url: up.ossURL})
		} else {
			failedCount++
		}
	}

	// 批量更新数据库
	for i := 0; i < len(successItems); i += migrateBatchSize {
		end := i + migrateBatchSize
		if end > len(successItems) {
			end = len(successItems)
		}
		batch := successItems[i:end]
		ids := make([]uint, 0, len(batch))
		urls := make([]string, 0, len(batch))
		for _, item := range batch {
			ids = append(ids, item.id)
			urls = append(urls, item.url)
		}
		database.DB.Model(&model.SceneGenerationImage{}).Where("id IN ?", ids).
			Updates(map[string]interface{}{"candidate_image_url": urls, "updated_at": time.Now()})
	}

	// 统计成功日志
	successCount := len(successItems)
	for _, item := range successItems {
		for _, m := range toMigrate {
			if m.id == item.id {
				log.Printf("[migrate-image] scene_generation_images ID=%d candidate 迁移成功: %s -> %s", item.id, m.url, item.url)
				break
			}
		}
	}

	log.Printf("[migrate-image] scene_generation_images candidate 批量迁移完成: 成功 %d, 失败 %d", successCount, failedCount)
}

// startImageURLMigration 启动外部 CDN 图片迁移到 OSS 的定时任务（已停用）
func startImageURLMigration() {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 21, 0, 0, 0, time.Local)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			timer := time.NewTimer(time.Until(next))
			<-timer.C
			migrateExternalImageURLsToOSS()
		}
	}()
	log.Printf("[migrate-image] 外部图片夜间迁移已启动：每天 21:00 执行")
	return
	log.Printf("[migrate-image] 外部CDN图片迁移功能已停用")
	// TODO: 如需重新启用，取消注释以下代码
	// time.AfterFunc(1*time.Minute, func() {
	// 	migrateExternalImageURLsToOSS()
	// })
	// ticker := time.NewTicker(1 * time.Minute)
	// go func() {
	// 	for range ticker.C {
	// 		migrateExternalImageURLsToOSS()
	// 	}
	// }()
}

var stpWorkerProcess *exec.Cmd

func startSTPWorker() {
	stpWorkerPath := config.AppConfig.Server.STPWorkerPath
	if stpWorkerPath == "" {
		wd, _ := os.Getwd()
		stpWorkerPath = filepath.Join(wd, "stp-renderer")
	}

	pythonPath := config.AppConfig.Server.STPWorkerPython
	if pythonPath == "" {
		if runtime.GOOS == "windows" {
			pythonPath = "py"
		} else {
			pythonPath = "python3"
		}
	}
	log.Printf("STP Worker config: python=%s, path=%s", pythonPath, stpWorkerPath)

	mainScript := filepath.Join(stpWorkerPath, "main.py")
	if _, err := os.Stat(mainScript); err != nil {
		log.Printf("STP Worker not found at %s, skipping", mainScript)
		return
	}

	cmd := exec.Command(pythonPath, mainScript)
	cmd.Dir = stpWorkerPath
	// Redirect STP Worker output to the same daily log file as Go backend
	if globalLogger != nil && globalLogger.file != nil {
		cmd.Stdout = globalLogger
		cmd.Stderr = globalLogger
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// Pass Redis, OSS, and Go API config env vars to Python Worker
	goBackendPort := config.AppConfig.Server.Port
	if goBackendPort == "" {
		goBackendPort = ":8080"
	} else if goBackendPort[0] != ':' {
		goBackendPort = ":" + goBackendPort
	}
	cmd.Env = append(os.Environ(),
		"REDIS_HOST="+config.AppConfig.Redis.Addr,
		"REDIS_PORT="+getRedisPort(),
		"REDIS_DB="+fmt.Sprintf("%d", config.AppConfig.Redis.DB),
		"GO_API_BASE_URL=http://localhost"+goBackendPort,
		"INTERNAL_API_SECRET="+config.AppConfig.Server.InternalSecret,
		"OSS_ACCESS_KEY="+config.AppConfig.Aliyun.AccessKeyID,
		"OSS_ACCESS_SECRET="+config.AppConfig.Aliyun.AccessKeySecret,
		"OSS_ENDPOINT="+config.AppConfig.OSS.Endpoint,
		"OSS_BUCKET="+config.AppConfig.OSS.Bucket,
		"OSS_DOMAIN="+config.AppConfig.OSS.Domain,
		"STP_RENDER_LOCK_KEY="+os.Getenv("STP_RENDER_LOCK_KEY"),
		"STP_RENDER_LOCK_TTL_SECONDS="+os.Getenv("STP_RENDER_LOCK_TTL_SECONDS"),
		"STP_RENDER_MEMORY_LIMIT_PERCENT="+os.Getenv("STP_RENDER_MEMORY_LIMIT_PERCENT"),
		"STP_RENDER_MEMORY_WAIT_SECONDS="+os.Getenv("STP_RENDER_MEMORY_WAIT_SECONDS"),
		"STP_RENDER_MEMORY_WAIT_MAX_SECONDS="+os.Getenv("STP_RENDER_MEMORY_WAIT_MAX_SECONDS"),
	)

	if err := cmd.Start(); err != nil {
		log.Printf("WARNING: STP Worker failed to start: %v", err)
		return
	}

	stpWorkerProcess = cmd
	log.Printf("STP Worker started (PID: %d, path: %s)", cmd.Process.Pid, stpWorkerPath)
}

func stopSTPWorker() {
	if stpWorkerProcess != nil && stpWorkerProcess.Process != nil {
		log.Printf("Stopping STP Worker (PID: %d)...", stpWorkerProcess.Process.Pid)
		stpWorkerProcess.Process.Kill()
		stpWorkerProcess.Wait()
		log.Println("STP Worker stopped")
	}
}

// getRedisPort extracts port from REDIS_ADDR (format: host:port or :port)
func getRedisPort() string {
	addr := config.AppConfig.Redis.Addr
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[idx+1:]
	}
	return "6379"
}

const serverWriteTimeout = 25 * time.Minute

func main() {
	// 1. Load Configuration
	config.LoadConfig()

	if err := utils.InitJWTFromEnv(); err != nil {
		log.Fatalf("JWT init failed: %v", err)
	}

	if logCloser := initLogger(); logCloser != nil {
		defer logCloser.Close()
	}

	// Set Gin mode: release suppresses debug logs and speeds up routing
	mode := config.AppConfig.Server.Mode
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	// 2. Connect to Database
	database.Connect()
	database.RunMigrations()

	// // 2.5. 启动外部CDN图片迁移到OSS定时任务
	// startImageURLMigration()

	// 3. Seed Data (Roles & Admin)
	seeder.SeedRolesAndAdmin()

	// 4. Connect to Redis
	cache.Connect()

	// 4.5. Start STP Renderer Worker
	startSTPWorker()

	// 4.5. Initialize OSS
	if err := storage.InitOSS(); err != nil {
		log.Fatalf("OSS init failed: %v", err)
	}

	// 4.6. Start scene task running cleanup
	startSceneTaskRunningCleanup()

	// 4.7. Start AI image generation task cleanup
	startImageGenerationTaskCleanup()

	// 4.8. Start monthly image statistics report scheduler
	service.StartMonthlyImageReportScheduler()
	handler.StartTaxExchangeRateScheduler()

	// 5. Start background image cleanup task (runs every 6 hours)
	go func() {
		for {
			if err := storage.CleanExpiredImages(); err != nil {
				log.Printf("Image cleanup error: %v", err)
			}
			time.Sleep(6 * time.Hour)
		}
	}()

	// 6. Setup Router and HTTP server with explicit timeouts.
	// WriteTimeout must exceed the longest AI response time (20 min).
	r := router.SetupRouter()
	port := config.AppConfig.Server.Port
	if port == "" {
		port = ":8080"
	} else if port[0] != ':' {
		port = ":" + port
	}
	srv := &http.Server{
		Addr:    port,
		Handler: r,
		// Guard against slow/malicious clients holding connections open
		ReadHeaderTimeout: 15 * time.Second,
		// Large enough for base64 image uploads over a slow link
		ReadTimeout: 2 * time.Minute,
		// Must cover 20-minute AI responses plus response transfer buffer
		WriteTimeout: serverWriteTimeout,
		// Recycle idle keep-alive connections to free file descriptors
		IdleTimeout: 90 * time.Second,
	}

	// 8. Graceful shutdown: wait for in-flight requests before exiting
	go func() {
		log.Printf("Server starting on port %s (mode=%s)", port, mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	stopSTPWorker()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}
