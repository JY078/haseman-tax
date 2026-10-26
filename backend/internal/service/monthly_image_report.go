package service

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"advertising-image/backend/internal/database"

	fontpkg "golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"gorm.io/gorm"
)

const monthlyImageReportChatID = "oc_42710bfbbe1657cc251c762b50b81610"

var monthlyImageReportLocation = func() *time.Location {
	location, err := time.LoadLocation(monthlyImageReportTimeZone)
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return location
}()

var reportFontState struct {
	sync.Once
	font *sfnt.Font
	data []byte
	err  error
}

func loadReportFont() (*sfnt.Font, []byte, error) {
	reportFontState.Do(func() {
		paths := []string{
			"ReportFont.ttf",
			filepath.Join("internal", "service", "ReportFont.ttf"),
			filepath.Join("fonts", "ReportFont.ttf"),
			filepath.Join("backend", "internal", "service", "ReportFont.ttf"),
			filepath.Join("backend", "fonts", "ReportFont.ttf"),
			`/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc`,
			`/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc`,
			`/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc`,
			`C:\Windows\Fonts\msyh.ttc`,
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			font, err := sfnt.Parse(data)
			if err != nil {
				continue
			}
			reportFontState.font = font
			reportFontState.data = data
			return
		}
		reportFontState.err = fmt.Errorf("ReportFont.ttf not found; copy it to backend/internal/service/ReportFont.ttf or install a CJK system font")
	})
	return reportFontState.font, reportFontState.data, reportFontState.err
}

type MonthlyImageReportRow struct {
	UserID            uint
	Username          string
	Name              string
	CurrentGenerated  int64
	CurrentDownloads  int64
	CurrentRate       float64
	PreviousGenerated int64
	PreviousDownloads int64
	PreviousRate      float64
	RateChange        float64
}

type MonthlyImageReport struct {
	Month         string
	PreviousMonth string
	Rows          []MonthlyImageReportRow
	ChartRows     []MonthlyImageReportRow
	Generated     int64
	Downloads     int64
	Rate          float64
	PreviousRate  float64
}

// BuildMonthlyImageReport loads all users for a month. Charts only use users
// with previous-month generation, while the detail table keeps every user.
func BuildMonthlyImageReport(month string) (*MonthlyImageReport, error) {
	month = strings.TrimSpace(month)
	start, err := time.ParseInLocation("2006-01", month, monthlyImageReportLocation)
	if err != nil {
		return nil, fmt.Errorf("month must use YYYY-MM format")
	}
	previous := start.AddDate(0, -1, 0)
	report := &MonthlyImageReport{Month: month, PreviousMonth: previous.Format("2006-01")}
	if err := queryMonthlyImageReport(database.DB, start, previous, report); err != nil {
		return nil, err
	}
	return report, nil
}

func queryMonthlyImageReport(db *gorm.DB, current, previous time.Time, report *MonthlyImageReport) error {
	currentEnd := current.AddDate(0, 1, 0)
	previousEnd := previous.AddDate(0, 1, 0)
	rows := make([]MonthlyImageReportRow, 0)
	err := db.Raw(`
SELECT
  u.id AS user_id,
  COALESCE(u.username, '') AS username,
  COALESCE(u.name, '') AS name,
  COALESCE(cur.generated_count, 0) AS current_generated,
  COALESCE(cur.download_count, 0) AS current_downloads,
  COALESCE(cur.download_rate, 0) AS current_rate,
  COALESCE(prev.generated_count, 0) AS previous_generated,
  COALESCE(prev.download_count, 0) AS previous_downloads,
  COALESCE(prev.download_rate, 0) AS previous_rate
FROM users u
LEFT JOIN (
  SELECT user_id, COUNT(*) AS generated_count,
         COALESCE(SUM(download_count), 0) AS download_count,
         CASE WHEN COUNT(*) > 0 THEN COALESCE(SUM(download_count), 0)::float / COUNT(*) ELSE 0 END AS download_rate
  FROM usage_logs
  WHERE image_url <> '' AND status = 'success' AND usage_type NOT IN ('retry', 'download')
    AND created_at >= ? AND created_at < ?
  GROUP BY user_id
) cur ON cur.user_id = u.id
LEFT JOIN (
  SELECT user_id, COUNT(*) AS generated_count,
         COALESCE(SUM(download_count), 0) AS download_count,
         CASE WHEN COUNT(*) > 0 THEN COALESCE(SUM(download_count), 0)::float / COUNT(*) ELSE 0 END AS download_rate
  FROM usage_logs
  WHERE image_url <> '' AND status = 'success' AND usage_type NOT IN ('retry', 'download')
    AND created_at >= ? AND created_at < ?
  GROUP BY user_id
) prev ON prev.user_id = u.id
ORDER BY current_generated DESC, current_downloads DESC, u.id ASC`, current, currentEnd, previous, previousEnd).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("query monthly image report: %w", err)
	}
	for i := range rows {
		if rows[i].CurrentGenerated > 0 && rows[i].CurrentRate == 0 {
			rows[i].CurrentRate = float64(rows[i].CurrentDownloads) / float64(rows[i].CurrentGenerated)
		}
		if rows[i].PreviousGenerated > 0 && rows[i].PreviousRate == 0 {
			rows[i].PreviousRate = float64(rows[i].PreviousDownloads) / float64(rows[i].PreviousGenerated)
		}
		if rows[i].PreviousGenerated > 0 {
			rows[i].RateChange = rows[i].CurrentRate - rows[i].PreviousRate
		}
		report.Generated += rows[i].CurrentGenerated
		report.Downloads += rows[i].CurrentDownloads
	}
	report.Rows = rows
	if report.Generated > 0 {
		report.Rate = float64(report.Downloads) / float64(report.Generated)
	}
	var previousGenerated, previousDownloads int64
	for _, row := range rows {
		previousGenerated += row.PreviousGenerated
		previousDownloads += row.PreviousDownloads
		if row.PreviousGenerated > 0 {
			report.ChartRows = append(report.ChartRows, row)
		}
	}
	if previousGenerated > 0 {
		report.PreviousRate = float64(previousDownloads) / float64(previousGenerated)
	}
	return nil
}

func GenerateAndSendMonthlyImageReport(month, chatID string) error {
	report, err := BuildMonthlyImageReport(month)
	if err != nil {
		return err
	}
	pdf, err := renderMonthlyImageReportPDF(report)
	if err != nil {
		return err
	}
	if strings.TrimSpace(chatID) == "" {
		chatID = monthlyImageReportChatID
	}
	return SendPDFToFeishuChat(chatID, fmt.Sprintf("图片统计月报-%s.pdf", report.Month), pdf)
}

func RenderMonthlyImageReportPDF(report *MonthlyImageReport) ([]byte, error) {
	if report == nil {
		return nil, fmt.Errorf("report is nil")
	}
	return renderMonthlyImageReportPDF(report)
}

func renderMonthlyImageReportPDF(report *MonthlyImageReport) ([]byte, error) {
	if _, _, err := loadReportFont(); err != nil {
		return nil, err
	}
	pages := []string{renderReportOverviewPage(report)}
	for offset := 0; offset < len(report.Rows) || offset == 0; offset += 24 {
		end := offset + 24
		if end > len(report.Rows) {
			end = len(report.Rows)
		}
		pages = append(pages, renderReportTablePage(report, report.Rows[offset:end], offset == 0))
		if end == len(report.Rows) {
			break
		}
	}
	return buildSimplePDF(pages), nil
}

func renderReportOverviewPage(report *MonthlyImageReport) string {
	var b strings.Builder
	b.WriteString(pdfTextAt(42, 800, 22, "图片统计月报"))
	b.WriteString(pdfTextAt(42, 775, 11, fmt.Sprintf("统计月份: %s    对比月份: %s", report.Month, report.PreviousMonth)))
	box := func(x float64, label, value string) {
		b.WriteString(pdfRect(x, 680, 155, 58, 0.95, 0.97, 1.0, true))
		b.WriteString(pdfTextAt(x+12, 715, 10, label))
		b.WriteString(pdfTextAt(x+12, 690, 20, value))
	}
	box(42, "本月生图数量", strconv.FormatInt(report.Generated, 10))
	box(220, "本月下载数量", strconv.FormatInt(report.Downloads, 10))
	box(398, "本月下载率", formatReportRate(report.Rate))
	b.WriteString(pdfTextAt(42, 645, 13, "下载率对比"))
	b.WriteString(pdfTextAt(42, 625, 10, fmt.Sprintf("本月 %s    上月 %s    变化 %s", formatReportRate(report.Rate), formatReportRate(report.PreviousRate), formatRateChange(report.Rate-report.PreviousRate))))
	b.WriteString(pdfTextAt(42, 575, 13, "图表: 上月有生图记录的用户"))
	b.WriteString(renderCountChart(report.ChartRows))
	b.WriteString(pdfTextAt(42, 265, 13, "下载率变化图"))
	b.WriteString(renderRateChart(report.ChartRows))
	return b.String()
}

func renderCountChart(rows []MonthlyImageReportRow) string {
	var b strings.Builder
	maxRows := rows
	if len(maxRows) > 12 {
		maxRows = maxRows[:12]
	}
	if len(maxRows) == 0 {
		return pdfTextAt(42, 525, 10, "暂无上月有生图记录的用户")
	}
	max := int64(1)
	for _, row := range maxRows {
		if row.CurrentGenerated > max {
			max = row.CurrentGenerated
		}
		if row.PreviousGenerated > max {
			max = row.PreviousGenerated
		}
	}
	startY := 535.0
	for i, row := range maxRows {
		y := startY - float64(i*22)
		name := reportUserName(row)
		if len([]rune(name)) > 8 {
			name = string([]rune(name)[:8])
		}
		b.WriteString(pdfTextAt(42, y, 8, name))
		b.WriteString(pdfRect(105, y-2, 300*float64(row.PreviousGenerated)/float64(max), 7, 0.66, 0.73, 0.84, true))
		b.WriteString(pdfRect(105, y+7, 300*float64(row.CurrentGenerated)/float64(max), 7, 0.18, 0.46, 0.78, true))
	}
	b.WriteString(pdfRect(420, 535, 8, 8, 0.18, 0.46, 0.78, true))
	b.WriteString(pdfTextAt(433, 535, 8, "本月生图"))
	b.WriteString(pdfRect(420, 515, 8, 8, 0.66, 0.73, 0.84, true))
	b.WriteString(pdfTextAt(433, 515, 8, "上月生图"))
	return b.String()
}

func renderRateChart(rows []MonthlyImageReportRow) string {
	var b strings.Builder
	maxRows := rows
	if len(maxRows) > 12 {
		maxRows = maxRows[:12]
	}
	if len(maxRows) == 0 {
		return pdfTextAt(42, 220, 10, "暂无可比较的下载率")
	}
	startY := 225.0
	for i, row := range maxRows {
		y := startY - float64(i*20)
		name := reportUserName(row)
		if len([]rune(name)) > 8 {
			name = string([]rune(name)[:8])
		}
		b.WriteString(pdfTextAt(42, y, 8, name))
		b.WriteString(pdfRect(105, y-2, 260*clampRate(row.PreviousRate), 7, 0.73, 0.78, 0.86, true))
		b.WriteString(pdfRect(105, y+7, 260*clampRate(row.CurrentRate), 7, 0.12, 0.62, 0.43, true))
	}
	b.WriteString(pdfRect(380, 225, 8, 8, 0.12, 0.62, 0.43, true))
	b.WriteString(pdfTextAt(393, 225, 8, "本月下载率"))
	b.WriteString(pdfRect(380, 205, 8, 8, 0.73, 0.78, 0.86, true))
	b.WriteString(pdfTextAt(393, 205, 8, "上月下载率"))
	return b.String()
}

func renderReportTablePage(report *MonthlyImageReport, rows []MonthlyImageReportRow, first bool) string {
	var b strings.Builder
	if first {
		b.WriteString(pdfTextAt(42, 800, 18, "用户明细"))
	} else {
		b.WriteString(pdfTextAt(42, 800, 18, "用户明细（续）"))
	}
	b.WriteString(pdfTextAt(42, 775, 9, "明细显示全部用户；下载率环比仅对上月有生图记录的用户计算。"))
	columns := []struct {
		x     float64
		title string
	}{{42, "用户"}, {145, "本月生图"}, {205, "本月下载"}, {270, "本月下载率"}, {350, "上月下载率"}, {430, "下载率变化"}}
	for _, col := range columns {
		b.WriteString(pdfTextAt(col.x, 742, 9, col.title))
	}
	b.WriteString(pdfLine(42, 734, 553, 734, 0.7, 0.74, 0.8))
	for i, row := range rows {
		y := 715 - float64(i*26)
		b.WriteString(pdfTextAt(42, y, 8, reportUserName(row)))
		b.WriteString(pdfTextAt(145, y, 8, strconv.FormatInt(row.CurrentGenerated, 10)))
		b.WriteString(pdfTextAt(205, y, 8, strconv.FormatInt(row.CurrentDownloads, 10)))
		b.WriteString(pdfTextAt(270, y, 8, formatReportRate(row.CurrentRate)))
		if row.PreviousGenerated == 0 {
			b.WriteString(pdfTextAt(350, y, 8, "-"))
			b.WriteString(pdfTextAt(430, y, 8, "-"))
		} else {
			b.WriteString(pdfTextAt(350, y, 8, formatReportRate(row.PreviousRate)))
			b.WriteString(pdfTextAt(430, y, 8, formatRateChange(row.RateChange)))
		}
		b.WriteString(pdfLine(42, y-8, 553, y-8, 0.9, 0.92, 0.95))
	}
	return b.String()
}

func reportUserName(row MonthlyImageReportRow) string {
	if strings.TrimSpace(row.Name) != "" {
		return strings.TrimSpace(row.Name)
	}
	if strings.TrimSpace(row.Username) != "" {
		return strings.TrimSpace(row.Username)
	}
	return fmt.Sprintf("用户%d", row.UserID)
}

func formatReportRate(rate float64) string { return fmt.Sprintf("%.1f%%", rate*100) }

func formatRateChange(change float64) string {
	if change > -0.0005 && change < 0.0005 {
		return "持平"
	}
	if change > 0 {
		return fmt.Sprintf("上升 %.1f 个百分点", change*100)
	}
	return fmt.Sprintf("下降 %.1f 个百分点", -change*100)
}

func clampRate(rate float64) float64 {
	if rate < 0 {
		return 0
	}
	if rate > 1 {
		return 1
	}
	return rate
}

func pdfTextAt(x, y float64, size int, text string) string {
	return fmt.Sprintf("0 0 0 rg BT /F1 %d Tf 1 0 0 1 %.2f %.2f Tm <%s> Tj ET\n", size, x, y, utf16Hex(text))
}

func utf16Hex(text string) string {
	loadedFont, _, err := loadReportFont()
	if err != nil {
		return ""
	}
	var fontBuffer sfnt.Buffer
	var buf bytes.Buffer
	for _, r := range text {
		glyph, glyphErr := loadedFont.GlyphIndex(&fontBuffer, r)
		if glyphErr != nil {
			glyph = 0
		}
		var encoded [2]byte
		binary.BigEndian.PutUint16(encoded[:], uint16(glyph))
		buf.Write(encoded[:2])
	}
	return hex.EncodeToString(buf.Bytes())
}

func pdfRect(x, y, w, h, r, g, bl float64, fill bool) string {
	op := "S"
	if fill {
		op = "f"
	}
	return fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re %s\n", r, g, bl, x, y, w, h, op)
}

func pdfLine(x1, y1, x2, y2, r, g, bl float64) string {
	return fmt.Sprintf("%.3f %.3f %.3f RG %.2f %.2f m %.2f %.2f l S\n", r, g, bl, x1, y1, x2, y2)
}

type simplePDF struct{ pages []string }

func buildSimplePDF(pages []string) []byte {
	_, fontData, _ := loadReportFont()
	fontWidths := reportFontWidths()
	objects := make([]string, 7)
	objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	objects[1] = ""
	objects[2] = "<< /Type /Font /Subtype /Type0 /BaseFont /NotoSansSC-Regular /Encoding /Identity-H /DescendantFonts [4 0 R] >>"
	objects[3] = fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType2 /BaseFont /NotoSansSC-Regular /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor 5 0 R /CIDToGIDMap /Identity /DW 1000 /W %s >>", fontWidths)
	objects[4] = "<< /Type /FontDescriptor /FontName /NotoSansSC-Regular /Flags 4 /FontBBox [0 -300 1200 1100] /ItalicAngle 0 /Ascent 1000 /Descent -300 /CapHeight 880 /StemV 80 /FontFile2 6 0 R >>"
	objects[5] = fmt.Sprintf("<< /Length %d /Length1 %d >>\nstream\n%s\nendstream", len(fontData), len(fontData), string(fontData))
	objects[6] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"
	kids := make([]string, 0, len(pages))
	for _, page := range pages {
		contentID := len(objects) + 1
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(page), page))
		pageID := len(objects) + 1
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R /F2 7 0 R >> >> /Contents %d 0 R >>", contentID))
		kids = append(kids, fmt.Sprintf("%d 0 R", pageID))
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages))
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func reportFontWidths() string {
	loadedFont, _, err := loadReportFont()
	if err != nil {
		return "[]"
	}
	var buffer sfnt.Buffer
	var b strings.Builder
	b.WriteString("[")
	for glyph := 0; glyph < loadedFont.NumGlyphs(); glyph++ {
		advance, err := loadedFont.GlyphAdvance(&buffer, sfnt.GlyphIndex(glyph), fixed.I(1000), fontpkg.HintingNone)
		width := 1000
		if err == nil {
			width = advance.Round()
			if width < 0 {
				width = 0
			}
		}
		fmt.Fprintf(&b, "%d [%d] ", glyph, width)
	}
	b.WriteString("]")
	return b.String()
}
