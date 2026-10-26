package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"advertising-image/backend/internal/cache"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func isSupportedModelExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".stp", ".step", ".obj", ".fbx":
		return true
	default:
		return false
	}
}

func redisQueuePosition(ctx context.Context, queueName string, taskID string) (int, int) {
	if cache.RDB == nil || taskID == "" {
		return 0, 0
	}
	items, err := cache.RDB.LRange(ctx, queueName, 0, -1).Result()
	if err != nil {
		log.Printf("[STP] queue position lookup failed queue=%s task=%s err=%v", queueName, taskID, err)
		return 0, 0
	}
	for index, item := range items {
		if item == taskID {
			return index + 1, len(items)
		}
	}
	return 0, len(items)
}

// UploadStpFile 上传 STP 文件并触发渲染
func UploadStpFile(c *gin.Context) {
	productID := c.Param("id")

	log.Printf("[STP] UploadStpFile called for product: %s", productID)

	// 解析 product ID
	var id uint
	if _, err := fmt.Sscanf(productID, "%d", &id); err != nil {
		log.Printf("[STP] Invalid product ID: %s, err: %v", productID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的产品ID"})
		return
	}

	// 查找产品
	var product model.ProductPrompt
	if err := database.DB.First(&product, id).Error; err != nil {
		log.Printf("[STP] Product not found: %d, err: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "产品不存在"})
		return
	}

	// 获取当前用户
	userID := c.GetUint("userID")
	log.Printf("[STP] userID from context: %d", userID)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	// 接收文件
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("[STP] FormFile error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 STP 文件"})
		return
	}
	log.Printf("[STP] Received file: %s, size: %d", file.Filename, file.Size)

	// 校验文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isSupportedModelExt(ext) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .stp、.step、.obj 和 .fbx 文件"})
		return
	}

	// 校验文件大小 (< 50MB)
	if file.Size > 50*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过 50 MB"})
		return
	}

	// 保存到本地临时目录
	tmpDir := "./uploads/stp"
	os.MkdirAll(tmpDir, 0755)
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext))
	log.Printf("[STP] Saving to: %s", tmpPath)

	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		log.Printf("[STP] SaveUploadedFile failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}
	log.Printf("[STP] File saved successfully")

	// 上传到 OSS
	ossDir := "products/stp"
	log.Printf("[STP] Uploading to OSS, dir: %s", ossDir)
	url, err := storage.MoveToOSS(tmpPath, ossDir)
	if err != nil {
		log.Printf("[STP] MoveToOSS failed: %v", err)
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传到 OSS 失败"})
		return
	}
	log.Printf("[STP] Uploaded to OSS: %s", url)

	// OSS 上传成功后删除本地临时文件
	os.Remove(tmpPath)

	// 生成任务 ID
	taskID := uuid.New().String()
	log.Printf("[STP] Generated taskID: %s", taskID)

	// 写入 Redis Hash（任务状态）和队列
	if cache.RDB == nil {
		log.Printf("[STP] Redis not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置，无法处理渲染任务"})
		// 清理已上传的 OSS 文件
		os.Remove(tmpPath)
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_render_task:%s", taskID)
	taskData := map[string]interface{}{
		"task_id":      taskID,
		"product_id":   fmt.Sprintf("%d", id),
		"stp_file_url": url,
		"user_id":      fmt.Sprintf("%d", userID),
		"status":       "pending",
		"created_at":   time.Now().Format(time.RFC3339),
	}
	// go-redis v9 HSet 不兼容 Redis 3.0，使用 HMSet 替代
	if err := cache.RDB.HMSet(ctx, taskKey, taskData).Err(); err != nil {
		log.Printf("[STP] HMSet failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入任务状态失败"})
		os.Remove(tmpPath)
		return
	}
	log.Printf("[STP] HSet succeeded, key: %s", taskKey)

	// 设置 24 小时过期
	if err := cache.RDB.Expire(ctx, taskKey, 24*time.Hour).Err(); err != nil {
		log.Printf("[STP] Expire failed: %v", err)
	}

	// 写入队列
	if err := cache.RDB.LPush(ctx, "stp_render_queue", taskID).Err(); err != nil {
		log.Printf("[STP] LPush failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入渲染队列失败"})
		os.Remove(tmpPath)
		return
	}
	log.Printf("[STP] LPush succeeded")

	// 更新产品数据库
	log.Printf("[STP] Updating product %d with render_status=processing", id)
	if err := database.DB.Model(&product).Updates(map[string]interface{}{
		"stp_file_url":  url,
		"render_status": "processing",
		"render_error":  "",
	}).Error; err != nil {
		log.Printf("[STP] DB Update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新产品状态失败"})
		return
	}
	log.Printf("[STP] DB Update succeeded")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"task_id":       taskID,
			"stp_file_url":  url,
			"render_status": "processing",
		},
	})
	log.Printf("[STP] UploadStpFile completed successfully")
}

// UploadStp 上传 STP 文件（白底图页面专用）
func UploadStp(c *gin.Context) {
	log.Printf("[STP] UploadStp called")

	userID := c.GetUint("userID")
	if userID == 0 {
		log.Printf("[STP] UploadStp: unauthorized - userID is 0")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}
	log.Printf("[STP] UploadStp: userID=%d", userID)

	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("[STP] UploadStp: FormFile error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 STP 文件"})
		return
	}
	log.Printf("[STP] UploadStp: received file: %s, size: %d", file.Filename, file.Size)

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isSupportedModelExt(ext) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .stp、.step、.obj 和 .fbx 文件"})
		return
	}

	if file.Size > 50*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过 50 MB"})
		return
	}

	tmpDir := "./uploads/stp"
	os.MkdirAll(tmpDir, 0755)
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext))

	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	log.Printf("[STP] UploadStp: uploading to OSS")
	url, err := storage.MoveToOSS(tmpPath, "products/stp")
	if err != nil {
		log.Printf("[STP] UploadStp: MoveToOSS failed: %v", err)
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传到 OSS 失败"})
		return
	}
	log.Printf("[STP] UploadStp: uploaded to OSS: %s", url)

	os.Remove(tmpPath)

	// 提取 mesh 数据用于前端 3D 预览
	meshData, meshError := extractMeshFromStp(url)
	if meshError != "" {
		// mesh 提取失败不影响上传成功，只是没有预览数据
		log.Printf("[STP] UploadStp: mesh extraction warning: %s", meshError)
	}

	log.Printf("[STP] UploadStp: SUCCESS, returning URL: %s", url)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"url":       url,
			"mesh_data": meshData,
		},
	})
}

// MeshDataRequest 解析 STP 并返回 mesh 数据
type MeshDataResponse struct {
	Vertices    [][]float64 `json:"vertices"`
	Triangles   [][]int     `json:"triangles"`
	BoundingBox struct {
		Center []float64 `json:"center"`
		Size   []float64 `json:"size"`
	} `json:"bounding_box"`
}

// extractMeshFromStp 从 STP 文件提取 mesh 数据
func extractMeshFromStp(ossUrl string) (map[string]interface{}, string) {
	// 下载 STP 文件到临时目录
	tmpDir := "./uploads/stp_tmp"
	os.MkdirAll(tmpDir, 0755)
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("mesh_%d.stp", time.Now().UnixNano()))

	// 从 OSS 下载
	// 简化处理：返回 nil 让前端使用占位模型
	// 完整实现需要 OSS SDK 下载文件后再调用 Python 脚本提取 mesh
	_ = ossUrl
	_ = tmpPath

	// 由于这里无法直接运行 Python，返回 nil
	// 前端会使用占位模型，但上传仍然成功
	return nil, "mesh extraction not available in Go handler"
}

// GetStpMeshPreview 获取 STP 文件的 mesh 预览数据（通过 Python worker）
func GetStpMeshPreview(c *gin.Context) {
	stpUrl := c.Query("stp_url")
	if stpUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 stp_url 参数"})
		return
	}

	// 创建任务来提取 mesh 数据
	taskID := uuid.New().String()

	if cache.RDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置"})
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_mesh_task:%s", taskID)
	taskData := map[string]interface{}{
		"task_id":      taskID,
		"stp_file_url": stpUrl,
		"type":         "mesh_preview",
		"status":       "pending",
		"created_at":   time.Now().Format(time.RFC3339),
	}

	if err := cache.RDB.HMSet(ctx, taskKey, taskData).Err(); err != nil {
		log.Printf("[STP] GetStpMeshPreview HMSet failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入任务状态失败"})
		return
	}

	cache.RDB.Expire(ctx, taskKey, 24*time.Hour)
	cache.RDB.RPush(ctx, "stp_mesh_queue", taskID)
	queuePosition, queueTotal := redisQueuePosition(ctx, "stp_mesh_queue", taskID)

	if async := strings.ToLower(c.Query("async")); async == "1" || async == "true" {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"task_id":        taskID,
				"status":         "pending",
				"queue_position": queuePosition,
				"queue_total":    queueTotal,
			},
		})
		return
	}

	// 轮询结果：大 STP 文件网格化可能明显超过 15 秒。
	maxAttempts := 240
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(500 * time.Millisecond)
		taskInfo, err := cache.RDB.HGetAll(ctx, taskKey).Result()
		if err != nil || len(taskInfo) == 0 {
			continue
		}

		if taskInfo["status"] == "completed" {
			// 返回 mesh 数据
			meshDataStr := taskInfo["mesh_data"]
			if meshDataStr != "" {
				var meshData map[string]interface{}
				if json.Unmarshal([]byte(meshDataStr), &meshData) == nil {
					c.JSON(http.StatusOK, gin.H{"code": 0, "data": meshData})
					return
				}
			}
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
			return
		}

		if taskInfo["status"] == "failed" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": taskInfo["error_message"]})
			return
		}
	}

	c.JSON(http.StatusGatewayTimeout, gin.H{"error": "mesh 提取超时"})
}

// GetStpMeshPreviewTask 查询 STP mesh 预览任务状态
func GetStpMeshPreviewTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 task_id 参数"})
		return
	}

	if cache.RDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置"})
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_mesh_task:%s", taskID)
	taskInfo, err := cache.RDB.HGetAll(ctx, taskKey).Result()
	if err != nil || len(taskInfo) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	resp := gin.H{
		"task_id":    taskID,
		"status":     taskInfo["status"],
		"error_msg":  taskInfo["error_message"],
		"created_at": taskInfo["created_at"],
	}
	if taskInfo["status"] == "pending" {
		position, total := redisQueuePosition(ctx, "stp_mesh_queue", taskID)
		resp["queue_position"] = position
		resp["queue_total"] = total
	}

	if taskInfo["status"] == "completed" {
		meshDataStr := taskInfo["mesh_data"]
		if meshDataStr != "" {
			var meshData map[string]interface{}
			if err := json.Unmarshal([]byte(meshDataStr), &meshData); err == nil {
				resp["mesh_data"] = meshData
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// GetRenderStatus 查询渲染状态
func GetRenderStatus(c *gin.Context) {
	productID := c.Param("id")

	var id uint
	if _, err := fmt.Sscanf(productID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的产品ID"})
		return
	}

	var product model.ProductPrompt
	if err := database.DB.First(&product, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "产品不存在"})
		return
	}

	// 从 reference_images 中筛选六面图
	sixViews := make([]model.ReferenceImage, 0)
	for _, img := range product.ReferenceImages {
		if strings.HasPrefix(img.Description, "[六面-") {
			sixViews = append(sixViews, img)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"stp_file_url":  product.StpFileURL,
			"render_status": product.RenderStatus,
			"render_error":  product.RenderError,
			"six_views":     sixViews,
		},
	})
}

// UpdateRenderResult Python Worker 调用此接口更新渲染结果
func UpdateRenderResult(c *gin.Context) {
	log.Printf("[STP] UpdateRenderResult: received request")
	log.Printf("[STP] Request headers: %v", c.Request.Header)
	log.Printf("[STP] Request body: %s", c.GetHeader("X-Body")) // Will be empty, we need to read it

	var req struct {
		TaskID   string                 `json:"task_id"`
		Status   string                 `json:"status"` // completed / failed
		SixViews []model.ReferenceImage `json:"six_views"`
		ErrorMsg string                 `json:"error_message"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[STP] UpdateRenderResult: JSON bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	log.Printf("[STP] UpdateRenderResult: taskID=%s, status=%s, sixViewsCount=%d, errorMsg=%s",
		req.TaskID, req.Status, len(req.SixViews), req.ErrorMsg)

	// 从 Redis 获取任务信息
	if cache.RDB == nil {
		log.Printf("[STP] UpdateRenderResult: Redis client is nil!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis 未配置"})
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_render_task:%s", req.TaskID)
	log.Printf("[STP] UpdateRenderResult: taskKey=%s", taskKey)
	productIDStr, err := cache.RDB.HGet(ctx, taskKey, "product_id").Result()
	if err != nil {
		log.Printf("[STP] UpdateRenderResult: HGet product_id failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	log.Printf("[STP] UpdateRenderResult: productIDStr=%s", productIDStr)

	var productID uint
	fmt.Sscanf(productIDStr, "%d", &productID)

	// 更新任务状态 - Redis 3.0 使用 HMSET
	statusData := map[string]interface{}{
		"status":        req.Status,
		"error_message": req.ErrorMsg,
		"completed_at":  time.Now().Format(time.RFC3339),
	}
	if err := cache.RDB.HMSet(ctx, taskKey, statusData).Err(); err != nil {
		log.Printf("[STP] UpdateRenderResult HMSet failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务状态失败"})
		return
	}

	// 更新产品数据库
	updates := map[string]interface{}{
		"render_status": req.Status,
		"render_error":  req.ErrorMsg,
	}

	if req.Status == "completed" && len(req.SixViews) > 0 {
		// 获取当前 reference_images，追加六面图
		var product model.ProductPrompt
		if err := database.DB.First(&product, productID).Error; err == nil {
			existingImgs := product.ReferenceImages
			// 移除旧六面图
			filtered := make([]model.ReferenceImage, 0)
			for _, img := range existingImgs {
				if !strings.HasPrefix(img.Description, "[六面-") {
					filtered = append(filtered, img)
				}
			}
			// 添加新六面图
			newImgs := append(filtered, req.SixViews...)
			// JSONB 列需要序列化为 JSON 字符串
			jsonBytes, _ := json.Marshal(newImgs)
			updates["reference_images"] = string(jsonBytes)
			log.Printf("[STP] UpdateRenderResult: productID=%d, existingImgs=%d, filtered=%d, newSixViews=%d, total=%d",
				productID, len(existingImgs), len(filtered), len(req.SixViews), len(newImgs))
		} else {
			log.Printf("[STP] UpdateRenderResult: productID=%d, First error: %v", productID, err)
		}
	}

	log.Printf("[STP] UpdateRenderResult: productID=%d, updates=%+v", productID, updates)
	result := database.DB.Model(&model.ProductPrompt{}).Where("id = ?", productID).Updates(updates)
	log.Printf("[STP] UpdateRenderResult: rows affected=%d, error=%v", result.RowsAffected, result.Error)

	if result.Error != nil {
		log.Printf("[STP] UpdateRenderResult: Database update FAILED: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新数据库失败"})
		return
	}

	log.Printf("[STP] UpdateRenderResult: SUCCESS, sending OK response")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// stpRenderSingleRequest 单张渲染请求
type stpRenderSingleRequest struct {
	StpFileURL   string          `json:"stp_file_url" binding:"required"`
	RenderMode   string          `json:"render_mode"` // "photo" | "wireframe" | "depth" | "silhouette"
	CameraParams json.RawMessage `json:"camera_params"`
	CameraAngle  struct {
		Azimuth   float64 `json:"azimuth"`
		Elevation float64 `json:"elevation"`
		Distance  float64 `json:"distance"`
	} `json:"camera_angle"`
	Resolution int `json:"resolution"`
}

// CreateStpRenderTask 创建 STP 单张渲染任务
func CreateStpRenderTask(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
		return
	}

	var req stpRenderSingleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StpFileURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "STP 文件 URL 不能为空"})
		return
	}

	resolution := req.Resolution
	if resolution == 0 {
		resolution = 1024
	}
	renderMode := req.RenderMode
	if renderMode == "" {
		renderMode = "photo"
	}

	taskID := uuid.New().String()
	log.Printf("[STP] CreateStpRenderTask: user=%d mode=%s resolution=%d azimuth=%.2f elevation=%.2f distance=%.2f url=%s",
		userID, renderMode, resolution, req.CameraAngle.Azimuth, req.CameraAngle.Elevation, req.CameraAngle.Distance, req.StpFileURL)
	cameraParams := string(req.CameraParams)

	if cache.RDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置"})
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_single_render_task:%s", taskID)
	taskData := map[string]interface{}{
		"task_id":       taskID,
		"stp_file_url":  req.StpFileURL,
		"user_id":       fmt.Sprintf("%d", userID),
		"azimuth":       req.CameraAngle.Azimuth,
		"elevation":     req.CameraAngle.Elevation,
		"distance":      req.CameraAngle.Distance,
		"camera_params": cameraParams,
		"resolution":    resolution,
		"render_mode":   renderMode,
		"status":        "pending",
		"created_at":    time.Now().Format(time.RFC3339),
	}

	if err := cache.RDB.HMSet(ctx, taskKey, taskData).Err(); err != nil {
		log.Printf("[STP] HMSet failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入任务状态失败"})
		return
	}

	cache.RDB.Expire(ctx, taskKey, 24*time.Hour)

	if err := cache.RDB.RPush(ctx, "stp_single_render_queue", taskID).Err(); err != nil {
		log.Printf("[STP] CreateStpRenderTask: RPush failed task=%s err=%v", taskID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入渲染队列失败"})
		return
	}
	queuePosition, queueTotal := redisQueuePosition(ctx, "stp_single_render_queue", taskID)
	log.Printf("[STP] CreateStpRenderTask: queued task=%s mode=%s", taskID, renderMode)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"task_id":        taskID,
			"status":         "pending",
			"queue_position": queuePosition,
			"queue_total":    queueTotal,
		},
	})
}

// GetStpRenderTask 查询 STP 单张渲染任务状态
func GetStpRenderTask(c *gin.Context) {
	taskID := c.Param("task_id")

	if cache.RDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置"})
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_single_render_task:%s", taskID)
	taskInfo, err := cache.RDB.HGetAll(ctx, taskKey).Result()
	if err != nil || len(taskInfo) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	resp := gin.H{
		"task_id":    taskID,
		"status":     taskInfo["status"],
		"error_msg":  taskInfo["error_message"],
		"created_at": taskInfo["created_at"],
	}
	if taskInfo["status"] == "pending" {
		position, total := redisQueuePosition(ctx, "stp_single_render_queue", taskID)
		resp["queue_position"] = position
		resp["queue_total"] = total
	}
	log.Printf("[STP] GetStpRenderTask: task=%s status=%s error=%s image=%s",
		taskID, taskInfo["status"], taskInfo["error_message"], taskInfo["image_url"])

	if taskInfo["status"] == "completed" {
		resp["image_url"] = taskInfo["image_url"]
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// UpdateStpSingleRenderResult Python Worker 调用此接口更新单张渲染结果
func UpdateStpSingleRenderResult(c *gin.Context) {
	var req struct {
		TaskID   string `json:"task_id"`
		Status   string `json:"status"`
		ImageURL string `json:"image_url"`
		ErrorMsg string `json:"error_message"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	if cache.RDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis 未配置"})
		return
	}

	ctx := c.Request.Context()
	taskKey := fmt.Sprintf("stp_single_render_task:%s", req.TaskID)

	updates := map[string]interface{}{
		"status":        req.Status,
		"error_message": req.ErrorMsg,
		"image_url":     req.ImageURL,
		"completed_at":  time.Now().Format(time.RFC3339),
	}

	cache.RDB.HMSet(ctx, taskKey, updates)
	log.Printf("[STP] UpdateStpSingleRenderResult: task=%s status=%s image=%s error=%s",
		req.TaskID, req.Status, req.ImageURL, req.ErrorMsg)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}
