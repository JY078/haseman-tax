package handler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/config"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DownloadUploadFile 代理下载 uploads 目录下的文件。
// 通过 Gin 响应链设置 Content-Disposition: attachment 强制浏览器触发下载。
func DownloadUploadFile(c *gin.Context) {
	rawPath := c.Query("path")
	if rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	clean := filepath.ToSlash(filepath.Clean(rawPath))
	if !strings.HasPrefix(clean, "/uploads/") || strings.Contains(clean, "..") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	sourcePath := filepath.Clean(strings.TrimPrefix(clean, "/"))
	sourceRef := sourcePath
	if sourceURL := trustedDownloadSourceURL(c.Query("source_url"), clean); sourceURL != "" {
		sourceRef = sourceURL
	}
	localSourcePath, localErr := resolveLocalUploadPath(sourcePath)
	user := getDownloadUser(c)
	filename := buildDownloadFilename(user, downloadFilenameParts{
		Brand:             c.Query("brand"),
		SKU:               c.Query("sku"),
		RequestedFilename: c.Query("filename"),
		SourcePath:        sourcePath,
	})
	annotate := c.Query("annotate") == "1"

	if nasDir := strings.TrimSpace(config.GetEnv("DOWNLOAD_NAS_DIR", "")); nasDir != "" && shouldSaveDownloadToNAS(user) {
		targetDir := resolveDownloadNASDir(nasDir, user)
		if err := copyDownloadToNAS(sourceRef, localSourcePath, targetDir, filename, annotate); err != nil {
			log.Printf("Warning: failed to copy download to NAS: path=%q source_ref=%q local_err=%v err=%v", clean, sourceRef, localErr, err)
		}
	}
	incrementDownloadCount(clean)

	c.Header("Content-Disposition", contentDisposition(filename))
	if localErr == nil {
		if !annotate {
			c.File(localSourcePath)
			return
		}
		if err := writeDownloadedImage(c, localSourcePath, sourcePath, filename, true); err != nil {
			log.Printf("Warning: failed to annotate downloaded image path=%q: %v", clean, err)
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "failed to annotate image metadata"})
		}
		return
	}
	if err := proxyDownloadStoredImage(c, sourceRef, filename, annotate); err != nil {
		log.Printf("Warning: failed to download stored image path=%q source_ref=%q local_err=%v err=%v", clean, sourceRef, localErr, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
	}
}

func contentDisposition(filename string) string {
	asciiName := sanitizeFilenamePart(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiName, url.PathEscape(filename))
}

func getDownloadUser(c *gin.Context) model.User {
	var user model.User
	if userID, exists := c.Get("userID"); exists {
		database.DB.Select("id", "username", "name", "platform", "department", "sub_department").First(&user, userID)
	}
	return user
}

func shouldSaveDownloadToNAS(user model.User) bool {
	return strings.TrimSpace(user.Platform) != "" && strings.TrimSpace(user.Department) != ""
}

func trustedDownloadSourceURL(rawURL string, cleanPath string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	if filepath.ToSlash(filepath.Clean(parsed.Path)) != cleanPath {
		return ""
	}
	if isPrivateHost(parsed.Hostname()) {
		return ""
	}
	return rawURL
}

type downloadFilenameParts struct {
	Brand             string
	SKU               string
	RequestedFilename string
	SourcePath        string
}

func buildDownloadFilename(user model.User, parts downloadFilenameParts) string {
	ext := safeExtension(filepath.Ext(parts.RequestedFilename))
	if ext == "" {
		ext = safeExtension(filepath.Ext(parts.SourcePath))
	}
	if ext == "" {
		ext = ".png"
	}

	subDepartment := strings.TrimSpace(user.SubDepartment)
	if subDepartment == "" {
		subDepartment = strings.TrimSpace(user.Department)
	}
	if subDepartment == "" {
		subDepartment = "未分部门"
	}
	brand := strings.TrimSpace(parts.Brand)
	sku := strings.TrimSpace(parts.SKU)
	if brand != "" && !isDownloadBrandAllowed(brand) {
		brand = ""
	}
	if brand == "" || sku == "" {
		productBrand, productSKU := lookupDownloadProductInfo(parts.SourcePath)
		if brand == "" && isDownloadBrandAllowed(productBrand) {
			brand = productBrand
		}
		if sku == "" {
			sku = productSKU
		}
	}
	if brand == "" {
		brand = "unknown"
	}
	if sku == "" {
		sku = "unknown"
	}
	remark := strings.TrimSpace(user.Name)
	if remark == "" {
		remark = strings.TrimSpace(user.Username)
	}
	if remark == "" {
		remark = "用户"
	}

	timestamp := strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "")
	fields := []string{subDepartment, brand, sku, remark, timestamp}
	for i, value := range fields {
		fields[i] = sanitizeFilenamePart(value)
	}
	return fmt.Sprintf("%s%s", strings.Join(fields, "_"), ext)
}

func lookupDownloadProductInfo(sourcePath string) (string, string) {
	candidates := downloadImageURLCandidates(sourcePath)
	if len(candidates) == 0 {
		return "", ""
	}

	var task struct {
		ProductID uint64
	}
	if err := database.DB.Table("ecom_suite_images AS images").
		Select("tasks.product_id").
		Joins("JOIN ecom_suite_tasks AS tasks ON tasks.task_id = images.task_id").
		Where("images.image_url IN ?", candidates).
		Order("images.created_at DESC").
		Limit(1).
		Scan(&task).Error; err != nil || task.ProductID == 0 {
		return "", ""
	}

	var product model.ProductPrompt
	if err := database.DB.Select("brand", "sku").First(&product, task.ProductID).Error; err != nil {
		return "", ""
	}
	return strings.TrimSpace(product.Brand), strings.TrimSpace(product.SKU)
}

func downloadImageURLCandidates(sourcePath string) []string {
	raw := strings.TrimSpace(sourcePath)
	if raw == "" {
		return nil
	}

	values := []string{raw}
	if parsed, err := url.Parse(raw); err == nil && parsed.Path != "" {
		values = append(values, parsed.Path)
	}

	result := make([]string, 0, len(values)*2)
	seen := make(map[string]bool, len(values)*2)
	for _, value := range values {
		clean := filepath.ToSlash(strings.TrimSpace(value))
		if clean == "" {
			continue
		}
		for _, candidate := range []string{clean, "/" + strings.TrimPrefix(clean, "/")} {
			if !seen[candidate] {
				seen[candidate] = true
				result = append(result, candidate)
			}
		}
	}
	return result
}

func safeExtension(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" || len(ext) > 10 || !strings.HasPrefix(ext, ".") {
		return ""
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return ext
}

func sanitizeFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(value)
}

func copyDownloadToNAS(sourcePath, localSourcePath, nasDir, filename string, annotate bool) error {
	var src io.ReadCloser
	if localSourcePath != "" {
		file, err := os.Open(localSourcePath)
		if err != nil {
			return err
		}
		src = file
	} else {
		body, _, err := openStoredImage(sourcePath)
		if err != nil {
			return err
		}
		src = body
	}
	defer src.Close()

	if !annotate {
		if err := os.MkdirAll(nasDir, 0755); err != nil {
			return err
		}
		destPath := uniqueDestinationPath(nasDir, filename)
		dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	data, err = storage.EnsureSyntheticPerformerXMP(data, http.DetectContentType(data))
	if err != nil {
		return err
	}
	return writeDownloadDataToNAS(data, nasDir, filename)
}

func writeDownloadDataToNAS(data []byte, nasDir, filename string) error {
	if err := os.MkdirAll(nasDir, 0755); err != nil {
		return err
	}
	destPath := uniqueDestinationPath(nasDir, filename)
	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = dst.Write(data)
	return err
}

func resolveDownloadNASDir(rootDir string, user model.User) string {
	if department, ok := findDepartmentByUser(user); ok {
		platformDir := nasPlatformDir(department.Platform)
		if platformDir == "" {
			return rootDir
		}
		if regionPath := cleanNASSubDir(department.NASRegionPath); regionPath != "" && normalizePlatform(department.Platform) == "amazon" {
			return filepath.Join(rootDir, filepath.FromSlash(platformDir), filepath.FromSlash(regionPath))
		}
		return filepath.Join(rootDir, filepath.FromSlash(platformDir))
	}

	if platformDir := nasPlatformDir(user.Platform); platformDir != "" {
		return filepath.Join(rootDir, filepath.FromSlash(platformDir))
	}

	subDir := lookupDepartmentNASSubDir(user)
	if subDir == "" {
		return rootDir
	}
	return filepath.Join(rootDir, filepath.FromSlash(subDir))
}

func lookupDepartmentNASSubDir(user model.User) string {
	department := strings.TrimSpace(user.Department)
	subDepartment := strings.TrimSpace(user.SubDepartment)
	fullDepartment := ""
	if department != "" && subDepartment != "" {
		fullDepartment = department + "/" + subDepartment
	}

	if fullDepartment != "" {
		if subDir := lookupDepartmentNASSubDirValue(fullDepartment); subDir != "" {
			return subDir
		}
	}
	return lookupDepartmentNASSubDirValue(department)
}

func lookupDepartmentNASSubDirValue(department string) string {
	department = strings.TrimSpace(department)
	if department == "" {
		return ""
	}
	raw := config.GetEnv("DOWNLOAD_NAS_DEPARTMENT_DIRS", "")
	for _, item := range strings.Split(raw, ",") {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != department {
			continue
		}
		return cleanNASSubDir(parts[1])
	}
	return ""
}

func cleanNASSubDir(value string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == ".." || clean == "" || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, ":") {
		return ""
	}
	return clean
}

func nasPlatformDir(platform string) string {
	switch normalizePlatform(platform) {
	case "amazon":
		return "Amazon"
	case "fbm":
		return "FBM"
	case "data":
		return "Data"
	case "designer":
		return "Designer"
	case "homedepot":
		return "HomeDepot"
	case "lowes":
		return "Lowe's"
	case "temu":
		return "TEMU"
	case "tiktok":
		return "TikTok"
	case "walmart":
		return "Walmart"
	case "wayfair":
		return "Wayfair"
	default:
		if cleaned := cleanNASSubDir(platform); cleaned != "" {
			return cleaned
		}
		return platform
	}
}

func resolveLocalUploadPath(sourcePath string) (string, error) {
	clean := filepath.Clean(strings.TrimPrefix(sourcePath, string(filepath.Separator)))
	if filepath.IsAbs(clean) || strings.Contains(filepath.ToSlash(clean), "..") {
		return "", fmt.Errorf("forbidden upload path")
	}

	candidates := make([]string, 0, 6)
	appendCandidate := func(path string) {
		if path == "" {
			return
		}
		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	appendCandidate(clean)
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		appendCandidate(filepath.Join(cwd, clean))
		appendCandidate(filepath.Join(cwd, "backend", clean))
		appendCandidate(filepath.Join(filepath.Dir(cwd), "backend", clean))
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		appendCandidate(filepath.Join(exeDir, clean))
		appendCandidate(filepath.Join(filepath.Dir(exeDir), clean))
		appendCandidate(filepath.Join(filepath.Dir(exeDir), "backend", clean))
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", os.ErrNotExist
}

func proxyDownloadStoredImage(c *gin.Context, imagePath string, filename string, annotate bool) error {
	body, contentType, err := openStoredImage(imagePath)
	if err != nil {
		return err
	}
	defer body.Close()

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition(filename))
	if !annotate {
		_, err = io.Copy(c.Writer, body)
		return err
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	data, err = storage.EnsureSyntheticPerformerXMP(data, contentType)
	if err != nil {
		return err
	}
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
	return nil
}

func writeDownloadedImage(c *gin.Context, localPath, imagePath, filename string, annotate bool) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	contentType := http.DetectContentType(data)
	if annotate {
		data, err = storage.EnsureSyntheticPerformerXMP(data, contentType)
		if err != nil {
			return err
		}
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition(filename))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, contentType, data)
	return nil
}

func openStoredImage(imagePath string) (io.ReadCloser, string, error) {
	objectKey := storedImageObjectKey(imagePath)
	if objectKey == "" || strings.Contains(filepath.ToSlash(objectKey), "..") {
		return nil, "", fmt.Errorf("invalid image path")
	}
	if storage.IsOSSEnabled() {
		body, contentType, err := storage.FetchFromOSS(objectKey)
		if err == nil {
			return body, contentType, nil
		}
		if !isPublicHTTPURL(imagePath) {
			return nil, "", fmt.Errorf("fetch oss object %q: %w", objectKey, err)
		}
	}
	if isPublicHTTPURL(imagePath) {
		return openRemoteImage(imagePath)
	}
	return nil, "", os.ErrNotExist
}

func storedImageObjectKey(imagePath string) string {
	if parsed, err := url.Parse(imagePath); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return strings.TrimPrefix(filepath.ToSlash(parsed.Path), "/")
	}
	return strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(imagePath)), "/")
}

func isPublicHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return false
	}
	return !isPrivateHost(parsed.Hostname())
}

func openRemoteImage(rawURL string) (io.ReadCloser, string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Haseman/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("remote image responded %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return resp.Body, contentType, nil
}

func uniqueDestinationPath(dir, filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	path := filepath.Join(dir, filename)
	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
	}
}

func incrementDownloadCount(imageURL string) {
	var logItem model.UsageLog
	err := database.DB.Select("id").
		Where("status = ? AND (image_url = ? OR image_url LIKE ?)", "success", imageURL, "%"+imageURL).
		Order("created_at DESC").
		First(&logItem).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Warning: failed to find usage log for download count: %v", err)
		}
		return
	}

	if err := database.DB.Model(&model.UsageLog{}).
		Where("id = ?", logItem.ID).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error; err != nil {
		log.Printf("Warning: failed to update usage log download count: %v", err)
	}
}
