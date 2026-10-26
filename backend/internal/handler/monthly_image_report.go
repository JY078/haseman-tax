package handler

import (
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func reportMonthQuery(c *gin.Context) string {
	month := strings.TrimSpace(c.Query("month"))
	if month == "" {
		month = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	return month
}

// GetMonthlyImageReportPDF renders a report for admin preview/download.
func GetMonthlyImageReportPDF(c *gin.Context) {
	report, err := service.BuildMonthlyImageReport(reportMonthQuery(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pdf, err := service.RenderMonthlyImageReportPDF(report)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// SendMonthlyImageReport sends the selected month report to a Feishu chat.
func SendMonthlyImageReport(c *gin.Context) {
	month := reportMonthQuery(c)
	chatID := strings.TrimSpace(c.Query("chat_id"))
	if err := service.GenerateAndSendMonthlyImageReport(month, chatID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"month": month, "message": "图片统计月报已发送"})
}
