package handler

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestTikTokSelectedSheet(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "说明")
	f.SetCellValue("说明", "A1", "不要导入")
	f.NewSheet("申报明细")
	f.SetCellValue("申报明细", "A1", "TikTok 涉税申报明细")
	header := make([]interface{}, len(tikTokTaxHeaders))
	for i, name := range tikTokTaxHeaders {
		header[i] = name
	}
	f.SetSheetRow("申报明细", "A2", &header)
	values := []interface{}{"2026-10-01", "order-123", "USD", "100", "5", "2"}
	f.SetSheetRow("申报明细", "A3", &values)
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseTikTokTaxFile(buf.Bytes(), ".xlsx", "申报明细")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Data[tikTokTaxHeaders[1]] != "order-123" {
		t.Fatalf("wrong rows: %#v", rows)
	}
	if _, err := parseTikTokTaxFile(buf.Bytes(), ".xlsx", "不存在"); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("expected missing sheet error, got %v", err)
	}
	if _, err := parseTikTokTaxFile(buf.Bytes(), ".xlsx", ""); err == nil || !strings.Contains(err.Error(), "指定") {
		t.Fatal("must reject worksheet selection by position")
	}
	f.DeleteSheet("申报明细")
	singleSheet, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseTikTokTaxFile(singleSheet.Bytes(), ".xlsx", ""); err == nil || !strings.Contains(err.Error(), "指定") {
		t.Fatalf("must reject missing named worksheet, got %v", err)
	}
	if _, err := parseTikTokTaxFile([]byte("test"), ".csv", "申报明细"); err == nil {
		t.Fatal("CSV must reject sheet selection")
	}
}

func TestTikTokTransactionSummaryUsesNamedWorksheetAndDeduplicatesQuarter(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "申报明细")
	f.NewSheet("申报汇总")
	f.SetCellValue("申报汇总", "A1", "年份-季度")
	f.SetCellValue("申报汇总", "H1", "交易笔数")
	f.SetSheetRow("申报汇总", "A2", &[]interface{}{"2026-Q1", "", "", "", "", "", "", "4"})
	f.SetSheetRow("申报汇总", "A3", &[]interface{}{"2026-Q1", "", "", "", "", "", "", "9"})
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseTikTokTransactionSummaryWithSheet(buf.Bytes(), ".xlsx", "申报汇总")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Quarter != "2026-Q1" || rows[0].TransactionCount != 9 {
		t.Fatalf("unexpected transaction summary: %#v", rows)
	}
}

func TestNormalizeTikTokQuarter(t *testing.T) {
	for input, want := range map[string]string{
		"2026年第2季度（4-6月）": "2026-Q2",
		"2026-Q3":         "2026-Q3",
		"2026 q1":         "2026-Q1",
	} {
		if got := normalizeTikTokQuarter(input); got != want {
			t.Fatalf("normalizeTikTokQuarter(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTikTokPostedAtFormats(t *testing.T) {
	for _, value := range []string{"2024-01-01", "2024-01-01 12:30:00", "2024/1/1", "2024/1/1 12:30", "1/1/2024 12:30:00", "2024-1-1", "2024年1月1日", "2024-01-01T12:30:00Z", "45292.5"} {
		t.Run(value, func(t *testing.T) {
			got := parseTikTokPostedAt(value)
			if got.IsZero() || got.Format("2006-01-02") != "2024-01-01" {
				t.Fatalf("incorrect date for %q: %v", value, got)
			}
		})
	}
	for _, value := range []string{"", "合计", "2024-02-30", "not-a-date"} {
		if got := parseTikTokPostedAt(value); !got.IsZero() {
			t.Fatalf("invalid input accepted: %q -> %v", value, got)
		}
	}
}
