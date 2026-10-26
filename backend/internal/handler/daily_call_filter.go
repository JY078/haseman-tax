package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

func dailyCallKind(c *gin.Context) string {
	if c.Query("kind") == "outbound" {
		return "outbound"
	}
	return "inbound"
}

func dailyCallQuery(c *gin.Context) *gorm.DB {
	return database.DB.Model(&model.DailyCall{}).Where("kind = ?", dailyCallKind(c))
}

// Calendar filters use Shanghai time and half-open bounds, including leap days.
func dailyCallDateBounds(value string) (time.Time, time.Time, error) {
	layout := map[int]string{4: "2006", 7: "2006-01", 10: "2006-01-02"}[len(value)]
	if layout == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date")
	}
	start, err := time.ParseInLocation(layout, value, time.FixedZone("Asia/Shanghai", 8*60*60))
	if err != nil {
		return start, start, err
	}
	end := start.AddDate(0, 0, 1)
	if len(value) == 4 {
		end = start.AddDate(1, 0, 0)
	}
	if len(value) == 7 {
		end = start.AddDate(0, 1, 0)
	}
	return start, end, nil
}

func filterDailyCallDate(query *gorm.DB, value string) (*gorm.DB, error) {
	if value == "" {
		return query, nil
	}
	start, end, err := dailyCallDateBounds(value)
	if err != nil {
		return query, err
	}
	return query.Where("created_at >= ? AND created_at < ?", start, end), nil
}
