package handler

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
)

func TestConversionUploadPreservesFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("payload", `{"platform":"TEMU","items":[{"order_id":"123","sku":"SKU","quantity":2}]}`)
	source := []byte("订单号,数量\r\n123,2\r\n")
	mcf := []byte{0xff, 0xfe, 'A', 0, '\t', 0, '2', 0}
	file, _ := writer.CreateFormFile("source_file", "原订单.csv")
	file.Write(source)
	file, _ = writer.CreateFormFile("mcf_file", "MCF.txt")
	file.Write(mcf)
	writer.Close()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	var input orderTrackingImportInput
	record, err := readConversionUpload(c, &input)
	if err != nil {
		t.Fatal(err)
	}
	if record.SourceName != "原订单.csv" || record.MCFName != "MCF.txt" || !bytes.Equal(source, record.SourceData) || !bytes.Equal(mcf, record.MCFData) {
		t.Fatalf("uploaded bytes or names changed: %#v", record)
	}
	if input.Platform != "TEMU" || len(input.Items) != 1 || input.Items[0].Quantity != 2 {
		t.Fatalf("unexpected payload: %#v", input)
	}
}

func TestConversionFingerprintRetryAndIsolation(t *testing.T) {
	record := model.OrderConversion{ImportedBy: 7, SourceData: []byte("original"), MCFData: []byte("output")}
	input := orderTrackingImportInput{Platform: "TEMU"}
	original := conversionFingerprint(&record, input)
	record.MCFName = "different-timestamp.txt"
	if conversionFingerprint(&record, input) != original {
		t.Fatal("retry with a new filename must deduplicate")
	}
	record.ImportedBy = 8
	if conversionFingerprint(&record, input) == original {
		t.Fatal("different operators must remain separate")
	}
	record.ImportedBy = 7
	record.MCFData = []byte("changed output")
	if conversionFingerprint(&record, input) == original {
		t.Fatal("different conversion output must remain separate")
	}
}

func TestConversionUploadRejectsMissingFile(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("payload", `{"platform":"TEMU"}`)
	w.Close()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	var input orderTrackingImportInput
	if _, err := readConversionUpload(c, &input); err == nil {
		t.Fatal("missing source and output must not create history")
	}
}
