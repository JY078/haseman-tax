package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatRateChange(t *testing.T) {
	tests := []struct {
		change float64
		want   string
	}{
		{0, "持平"},
		{0.125, "上升 12.5 个百分点"},
		{-0.082, "下降 8.2 个百分点"},
	}
	for _, tt := range tests {
		if got := formatRateChange(tt.change); got != tt.want {
			t.Fatalf("formatRateChange(%v) = %q, want %q", tt.change, got, tt.want)
		}
	}
}

func TestRenderMonthlyImageReportPDF(t *testing.T) {
	report := &MonthlyImageReport{
		Month: "2026-07", PreviousMonth: "2026-06",
		Generated: 180, Downloads: 96, Rate: 96.0 / 180.0, PreviousRate: 0.42,
		Rows: []MonthlyImageReportRow{
			{Name: "张三", CurrentGenerated: 100, CurrentDownloads: 60, CurrentRate: 0.6, PreviousGenerated: 80, PreviousRate: 0.5, RateChange: 0.1},
			{Name: "新用户", CurrentGenerated: 80, CurrentDownloads: 36, CurrentRate: 0.45},
		},
		ChartRows: []MonthlyImageReportRow{
			{Name: "张三", CurrentGenerated: 100, PreviousGenerated: 80, CurrentRate: 0.6, PreviousRate: 0.5},
		},
	}
	pdf, err := RenderMonthlyImageReportPDF(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF-1.4") {
		t.Fatalf("unexpected PDF header: %q", string(pdf[:minInt(len(pdf), 12)]))
	}
	path := filepath.Join("tmp", "pdfs", "monthly-image-report-test.pdf")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pdf, 0644); err != nil {
		t.Fatal(err)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
