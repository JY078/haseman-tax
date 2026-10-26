package handler

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/storage"
	"github.com/extrame/ole2"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

var buyerReviewTemplateHeaders = []string{"品牌", "国家", "父ASIN", "子ASIN", "评分", "姓名", "评论时间", "评论标题", "评论内容", "评论链接", "图片链接", "视频附件"}

// buyerReviewHeaderScanRows bounds how many leading rows of each worksheet are
// inspected when looking for the review list header row.
const buyerReviewHeaderScanRows = 10

type reviewEmbeddedVideo struct {
	Row  int
	Name string
	Data []byte
}

func reviewMediaLinks(value string) (string, error) {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '，' || r == '\n' || r == '\r' || r == ';' || r == '；' })
	links := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		u, err := url.Parse(part)
		if err != nil || !((u.Scheme == "https" || u.Scheme == "http") && u.Host != "" || strings.HasPrefix(part, "/uploads/")) {
			return "", errors.New("附件必须是 HTTP/HTTPS 链接或嵌入的视频文件")
		}
		links = append(links, part)
	}
	return strings.Join(links, ","), nil
}

func reviewZipRead(files map[string]*zip.File, name string, limit int64) ([]byte, error) {
	file := files[path.Clean(name)]
	if file == nil {
		return nil, fmt.Errorf("工作簿缺少 %s", name)
	}
	if file.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("工作簿附件过大")
	}
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if int64(len(data)) > limit {
		return nil, errors.New("工作簿附件过大")
	}
	return data, err
}

type reviewRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
	Mode   string `xml:"TargetMode,attr"`
}

func reviewRelationships(files map[string]*zip.File, source string) (map[string]string, error) {
	name := path.Join(path.Dir(source), "_rels", path.Base(source)+".rels")
	result := map[string]string{}
	if files[name] == nil {
		return result, nil
	}
	data, err := reviewZipRead(files, name, 4<<20)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Items []reviewRelationship `xml:"Relationship"`
	}
	if err = xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	for _, item := range doc.Items {
		if item.Mode != "External" {
			if strings.HasPrefix(item.Target, "/") {
				result[item.ID] = strings.TrimPrefix(item.Target, "/")
			} else {
				result[item.ID] = path.Clean(path.Join(path.Dir(source), item.Target))
			}
		}
	}
	return result, nil
}

// OLE packages are data only: extract Ole10Native, never execute/open the attachment.
func reviewOLEVideo(data []byte) (name string, content []byte, err error) {
	defer func() {
		if recover() != nil {
			name = ""
			content = nil
			err = errors.New("无法读取嵌入视频")
		}
	}()
	ole, err := ole2.Open(bytes.NewReader(data), "")
	if err != nil {
		return "", nil, err
	}
	entries, err := ole.ListDir()
	if err != nil {
		return "", nil, err
	}
	var root, native *ole2.File
	for _, entry := range entries {
		if entry.Type == ole2.ROOT {
			root = entry
		}
		if entry.Bsize >= 2 && entry.Bsize <= 64 && entry.Name() == "\x01Ole10Native" {
			native = entry
		}
	}
	if root == nil || native == nil || native.Size > maxReviewVideoSize+(1<<20) {
		return "", nil, errors.New("不支持的嵌入附件")
	}
	raw, err := io.ReadAll(io.LimitReader(ole.OpenFile(native, root), int64(native.Size)))
	if err != nil {
		return "", nil, err
	}
	if len(raw) < 6 {
		return "", nil, errors.New("视频附件损坏")
	}
	offset := 6
	readString := func() (string, error) {
		if offset >= len(raw) {
			return "", io.ErrUnexpectedEOF
		}
		end := bytes.IndexByte(raw[offset:], 0)
		if end < 0 {
			return "", io.ErrUnexpectedEOF
		}
		s := string(raw[offset : offset+end])
		offset += end + 1
		return s, nil
	}
	name, err = readString()
	if err != nil {
		return "", nil, err
	}
	if _, err = readString(); err != nil {
		return "", nil, err
	}
	offset += 8
	if _, err = readString(); err != nil {
		return "", nil, err
	}
	if offset+4 > len(raw) {
		return "", nil, io.ErrUnexpectedEOF
	}
	size := int(binary.LittleEndian.Uint32(raw[offset : offset+4]))
	offset += 4
	if size <= 0 || size > maxReviewVideoSize || size > len(raw)-offset {
		return "", nil, errors.New("视频附件为空、损坏或超过 100 MB")
	}
	content = raw[offset : offset+size]
	if !reviewVideoTypeAllowed(name, mimetype.Detect(content).String()) {
		return "", nil, errors.New("嵌入附件只支持 MP4、MOV、WebM 视频")
	}
	return name, content, nil
}

func parseBuyerReviewWorkbook(data []byte) ([]buyerReviewInput, []reviewEmbeddedVideo, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, errors.New("请选择有效的 XLSX 文件")
	}
	files := map[string]*zip.File{}
	var total uint64
	for _, file := range archive.File {
		total += file.UncompressedSize64
		files[file.Name] = file
	}
	if total > 512<<20 {
		return nil, nil, errors.New("工作簿解压后不能超过 512 MB")
	}
	book, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 512 << 20, UnzipXMLSizeLimit: 16 << 20})
	if err != nil {
		return nil, nil, err
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, errors.New("工作簿没有工作表")
	}
	// The review list is not always the first sheet: scan every worksheet for
	// the header row carrying the full template, and allow a few leading
	// title rows above the header.
	sheet := ""
	headerIndex := 0
	positions := map[string]int{}
	var rows [][]string
	for _, name := range sheets {
		candidate, err := book.GetRows(name)
		if err != nil {
			continue
		}
		for i, row := range candidate {
			if i >= buyerReviewHeaderScanRows {
				break
			}
			columns := map[string]int{}
			for j, h := range row {
				columns[strings.TrimSpace(h)] = j
			}
			complete := true
			for _, h := range buyerReviewTemplateHeaders {
				if _, ok := columns[h]; !ok {
					complete = false
					break
				}
			}
			if complete {
				sheet, headerIndex, positions, rows = name, i, columns, candidate
				break
			}
		}
		if sheet != "" {
			break
		}
	}
	if sheet == "" {
		return nil, nil, fmt.Errorf("工作簿中没有评论列表工作表，需要列：%s", strings.Join(buyerReviewTemplateHeaders, "、"))
	}
	if len(rows) < headerIndex+2 {
		return nil, nil, errors.New("没有可导入的评论")
	}
	inputs := []buyerReviewInput{}
	rowIndex := map[int]int{}
	for index, row := range rows[headerIndex+1:] {
		excelRow := headerIndex + index + 2
		get := func(header string) string {
			col := positions[header]
			if col >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[col])
		}
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		images, e := reviewMediaLinks(get("图片链接"))
		if e != nil {
			return nil, nil, fmt.Errorf("第 %d 行图片链接：%w", excelRow, e)
		}
		video, e := reviewMediaLinks(get("视频附件"))
		if e != nil {
			return nil, nil, fmt.Errorf("第 %d 行视频附件：%w", excelRow, e)
		}
		date := get("评论时间")
		cell, _ := excelize.CoordinatesToCellName(positions["评论时间"]+1, excelRow)
		raw, _ := book.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
		if serial, e := strconv.ParseFloat(raw, 64); e == nil && serial > 0 && serial < 100000 {
			props, _ := book.GetWorkbookProps()
			use1904 := props.Date1904 != nil && *props.Date1904
			d, e := excelize.ExcelDateToTime(serial, use1904)
			if e == nil {
				date = d.Format(time.DateOnly)
			}
		}
		var rating float64
		fmt.Sscan(get("评分"), &rating)
		rowIndex[excelRow] = len(inputs)
		inputs = append(inputs, buyerReviewInput{Brand: get("品牌"), Country: get("国家"), ParentASIN: get("父ASIN"), ChildASIN: get("子ASIN"), Rating: rating, Name: get("姓名"), Date: date, Title: get("评论标题"), Content: get("评论内容"), Link: get("评论链接"), ReviewImages: &images, Video: video != "", VideoUpload: video})
	}
	workbookXML, err := reviewZipRead(files, "xl/workbook.xml", 4<<20)
	if err != nil {
		return nil, nil, err
	}
	var workbook struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			ID   string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err = xml.Unmarshal(workbookXML, &workbook); err != nil {
		return nil, nil, err
	}
	rels, err := reviewRelationships(files, "xl/workbook.xml")
	if err != nil {
		return nil, nil, err
	}
	source := ""
	for _, s := range workbook.Sheets {
		if s.Name == sheet {
			source = rels[s.ID]
		}
	}
	sheetXML, err := reviewZipRead(files, source, 32<<20)
	if err != nil {
		return nil, nil, err
	}
	rels, err = reviewRelationships(files, source)
	if err != nil {
		return nil, nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(sheetXML))
	videos := []reviewEmbeddedVideo{}
	seen := map[string]bool{}
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, nil, e
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "oleObject" {
			continue
		}
		var object struct {
			ID     string `xml:"id,attr"`
			Anchor *struct {
				From struct {
					Row int `xml:"row"`
					Col int `xml:"col"`
				} `xml:"from"`
			} `xml:"objectPr>anchor"`
		}
		if err = decoder.DecodeElement(&object, &start); err != nil {
			return nil, nil, err
		}
		if seen[object.ID] {
			continue
		}
		if object.Anchor == nil {
			return nil, nil, errors.New("无法确定嵌入视频所在行，请将附件放入视频附件列后重新保存")
		}
		seen[object.ID] = true
		row := object.Anchor.From.Row + 1
		if object.Anchor.From.Col != positions["视频附件"] {
			return nil, nil, fmt.Errorf("第 %d 行附件不在视频附件列", row)
		}
		if _, ok := rowIndex[row]; !ok {
			return nil, nil, fmt.Errorf("第 %d 行视频没有对应评论", row)
		}
		raw, e := reviewZipRead(files, rels[object.ID], maxReviewVideoSize+(1<<20))
		if e != nil {
			return nil, nil, e
		}
		name, video, e := reviewOLEVideo(raw)
		if e != nil {
			return nil, nil, fmt.Errorf("第 %d 行：%w", row, e)
		}
		videos = append(videos, reviewEmbeddedVideo{Row: rowIndex[row], Name: name, Data: video})
	}
	return inputs, videos, nil
}

func ImportBuyerReviewWorkbook(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<20)
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
		c.JSON(400, gin.H{"error": "工作簿读取失败，文件不能超过 256 MB"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "请选择评论工作簿"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(400, gin.H{"error": "文件读取失败"})
		return
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, 256<<20))
	if err != nil {
		c.JSON(400, gin.H{"error": "文件读取失败"})
		return
	}
	input, videos, err := parseBuyerReviewWorkbook(data)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if len(input) == 0 || len(input) > 50000 {
		c.JSON(400, gin.H{"error": "单次请导入 1 至 50000 条评论"})
		return
	}
	localFiles, ossKeys := []string{}, []string{}
	committed := false
	defer func() {
		for _, p := range localFiles {
			if !committed || storage.IsOSSEnabled() {
				_ = os.Remove(p)
			}
		}
		if !committed {
			for _, key := range ossKeys {
				_ = storage.DeleteFromOSS(key)
			}
		}
	}()
	for _, video := range videos {
		dir := filepath.Join("uploads", "review-videos")
		if err = os.MkdirAll(dir, 0755); err != nil {
			break
		}
		p := filepath.Join(dir, uuid.NewString()+strings.ToLower(filepath.Ext(video.Name)))
		localFiles = append(localFiles, p)
		if err = os.WriteFile(p, video.Data, 0600); err != nil {
			break
		}
		link := "/" + filepath.ToSlash(p)
		if storage.IsOSSEnabled() {
			key := filepath.ToSlash(p)
			link, err = storage.UploadToOSS(p, key)
			if err != nil {
				break
			}
			ossKeys = append(ossKeys, key)
		}
		row := &input[video.Row]
		if row.VideoUpload != "" {
			row.VideoUpload += ","
		}
		row.VideoUpload += link
		row.Video = true
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "视频附件上传失败，评论未导入，请重试"})
		return
	}
	if err = saveBuyerReviews(input); err != nil {
		c.JSON(500, gin.H{"error": "评论导入失败，请重试"})
		return
	}
	committed = true
	c.JSON(200, gin.H{"imported": len(input), "videos": len(videos)})
}
