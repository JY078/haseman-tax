package handler

import (
	"github.com/xuri/excelize/v2"
	"os"
	"strings"
	"testing"
)

func TestBuyerReviewWorkbookTextDatesAndLinks(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	headers := make([]any, len(buyerReviewTemplateHeaders))
	for i, h := range buyerReviewTemplateHeaders {
		headers[i] = h
	}
	book.SetSheetRow("Sheet1", "A1", &headers)
	row := []any{"品牌", "美国", "PARENT", "B0TEST", 4, "买家", "2026-10-23", "标题", "内容", "https://example.com/review", "https://example.com/a.jpg,https://example.com/b.jpg", "https://example.com/a.mp4,/uploads/review-videos/b.mp4"}
	book.SetSheetRow("Sheet1", "A2", &row)
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, videos, err := parseBuyerReviewWorkbook(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(videos) != 0 || rows[0].Date != "2026-10-23" || !rows[0].Video || rows[0].VideoUpload != row[11] {
		t.Fatalf("unexpected row: %+v", rows)
	}
	if *rows[0].ReviewImages != row[10] || rows[0].Image != "" {
		t.Fatal("wrong image mapping")
	}
	book.SetCellValue("Sheet1", "L1", "错误列名")
	buffer, _ = book.WriteToBuffer()
	if _, _, err = parseBuyerReviewWorkbook(buffer.Bytes()); err == nil {
		t.Fatal("missing video attachment column accepted")
	}
}

func TestBuyerReviewWorkbookFindsReviewSheet(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	// First sheet is a cover page without review headers.
	book.SetSheetName("Sheet1", "说明")
	book.SetCellValue("说明", "A1", "评论导入模板使用说明")
	headers := make([]any, len(buyerReviewTemplateHeaders))
	for i, h := range buyerReviewTemplateHeaders {
		headers[i] = h
	}
	// Review list lives on a later sheet, with a title row above the header.
	if _, err := book.NewSheet("评论列表"); err != nil {
		t.Fatal(err)
	}
	book.SetSheetRow("评论列表", "A2", &headers)
	row := []any{"品牌", "美国", "PARENT", "B0TEST", 5, "买家", "2026-10-23", "标题", "内容", "https://example.com/review", "https://example.com/a.jpg", ""}
	book.SetSheetRow("评论列表", "A3", &row)
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, videos, err := parseBuyerReviewWorkbook(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(videos) != 0 || rows[0].Link != "https://example.com/review" || *rows[0].ReviewImages != "https://example.com/a.jpg" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestBuyerReviewWorkbookRejectsWorkbookWithoutReviewHeaders(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	book.SetSheetName("Sheet1", "说明")
	book.SetCellValue("说明", "A1", "评论导入模板使用说明")
	if _, err := book.NewSheet("数据"); err != nil {
		t.Fatal(err)
	}
	book.SetSheetRow("数据", "A1", &[]any{"任意", "表头"})
	book.SetSheetRow("数据", "A2", &[]any{"任意", "内容"})
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = parseBuyerReviewWorkbook(buffer.Bytes()); err == nil {
		t.Fatal("workbook without review headers accepted")
	}
}

func TestBuyerReviewSuppliedWorkbook(t *testing.T) {
	p := os.Getenv("REVIEW_TEMPLATE_TEST_FILE")
	if p == "" {
		t.Skip("set REVIEW_TEMPLATE_TEST_FILE")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	rows, videos, err := parseBuyerReviewWorkbook(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 25 || len(videos) != 2 {
		t.Fatalf("rows=%d videos=%d", len(rows), len(videos))
	}
	if videos[0].Row != 7 || videos[1].Row != 23 {
		t.Fatalf("wrong attachment rows: %d, %d", videos[0].Row, videos[1].Row)
	}
	if len(strings.Split(*rows[14].ReviewImages, ",")) != 3 {
		t.Fatal("row 16 must have three images")
	}
	for _, video := range videos {
		if len(video.Data) < 1000000 || !strings.HasSuffix(video.Name, ".mp4") {
			t.Fatal("video extraction failed")
		}
	}
	for _, row := range rows {
		if row.Image != "" || len(row.Date) != 10 {
			t.Fatalf("incorrect field mapping or date: %s", row.Date)
		}
	}
}
func TestReviewMediaLinks(t *testing.T) {
	got, err := reviewMediaLinks(" https://example.com/1.jpg,https://example.com/2.jpg ")
	if err != nil || got != "https://example.com/1.jpg,https://example.com/2.jpg" {
		t.Fatal(got, err)
	}
	for _, bad := range []string{"video.mp4", "C:\\video.mp4", "data:video/mp4;base64,abcd", "javascript:alert(1)"} {
		if _, err := reviewMediaLinks(bad); err == nil {
			t.Fatalf("accepted non-link %q", bad)
		}
	}
}
