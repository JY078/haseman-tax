package handler

import (
	"fmt"
	"net/http"
	"path/filepath"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func beginTikTokImport(kind, name, store, site, importer string, content []byte) (*model.TaxImportLog, error) {
	if store == "" {
		store = "Forious"
	}
	audit := &model.TaxImportLog{Platform: "tiktok", ReportType: kind, StoreID: store, SiteID: taxSiteID(site), Importer: importer, FileName: filepath.Base(name), FileContent: content, Status: "processing"}
	if err := database.DB.Create(audit).Error; err != nil {
		return nil, fmt.Errorf("创建导入日志失败：%w", err)
	}
	return audit, nil
}

func finishTikTokImport(db *gorm.DB, audit *model.TaxImportLog, batchID uint, imported, skipped int, status string) error {
	audit.BatchID, audit.ImportedRows, audit.SkippedRows, audit.Status = batchID, imported, skipped, status
	return db.Model(audit).Updates(map[string]interface{}{"batch_id": batchID, "source_rows": audit.SourceRows, "imported_rows": imported, "skipped_rows": skipped, "status": status}).Error
}

func respondTikTokImport(c *gin.Context, audit *model.TaxImportLog, code int, ok bool, msg string) {
	if audit != nil && (!ok || audit.Status == "processing") {
		status, batchID, skipped := "failed", uint(0), 0
		if ok {
			status, batchID, skipped = "duplicate", audit.BatchID, audit.SourceRows
		}
		err := database.DB.Model(audit).Updates(map[string]interface{}{"batch_id": batchID, "source_rows": audit.SourceRows, "imported_rows": 0, "skipped_rows": skipped, "status": status, "message": msg}).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": false, "msg": "保存导入日志失败"})
			return
		}
	}
	c.JSON(code, gin.H{"status": ok, "msg": msg})
}
