package handler

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"advertising-image/backend/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

//go:embed amazon_summary.py
var amazonSummaryParser string

type amazonSummary struct {
	DisplayName string          `json:"display_name"`
	Start       string          `json:"start"`
	End         string          `json:"end"`
	Currency    string          `json:"currency"`
	Tax         decimal.Decimal `json:"tax"`
	Storage     decimal.Decimal `json:"storage"`
	Delivery    decimal.Decimal `json:"delivery"`
}

func sameAmazonSummaryIdentity(a, b amazonSummary) bool {
	return strings.EqualFold(strings.TrimSpace(a.DisplayName), strings.TrimSpace(b.DisplayName)) &&
		strings.EqualFold(strings.TrimSpace(a.Currency), strings.TrimSpace(b.Currency)) &&
		a.Start == b.Start && a.End == b.End
}

func parseAmazonSummary(ctx context.Context, content []byte) (amazonSummary, error) {
	var result amazonSummary
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		return result, fmt.Errorf("summary_file 必须为有效 PDF")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, customsPythonExecutable(), "-c", amazonSummaryParser)
	cmd.Stdin = bytes.NewReader(content)
	output, err := cmd.Output()
	if err != nil {
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(output, &failure) == nil && failure.Error != "" {
			return result, fmt.Errorf("汇总 PDF 解析失败：%s", failure.Error)
		}
		return result, fmt.Errorf("汇总 PDF 解析失败，请检查 CUSTOMS_PYTHON、pdfplumber 或文件是否损坏")
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return result, fmt.Errorf("汇总 PDF 解析结果无效")
	}
	return result, nil
}

func validateAmazonSummary(db *gorm.DB, rows []model.AmazonTaxShipment, summary amazonSummary) error {
	start, err := time.Parse("2006-01-02", summary.Start)
	if err != nil {
		return fmt.Errorf("汇总开始日期无效")
	}
	end, err := time.Parse("2006-01-02", summary.End)
	if err != nil || end.Before(start) || len(strings.TrimSpace(summary.Currency)) != 3 {
		return fmt.Errorf("汇总日期范围或币种无效")
	}
	_ = db
	return nil
}
