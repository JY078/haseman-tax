package handler

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

var testPNG = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xAE, 0x42, 0x60, 0x82}

func TestBuildSyntheticPerformerArchive(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "example.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(testPNG); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatal(err)
	}
	archive, err := buildSyntheticPerformerArchive(req.MultipartForm.File["files"])
	if err != nil {
		t.Fatalf("build archive: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "example.png" {
		t.Fatalf("unexpected archive entries: %#v", zr.File)
	}
	entry, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(entry)
	entry.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("contains-synthetic-performer")) {
		t.Fatal("archive image is missing the required keyword")
	}
}

func TestAddSyntheticPerformerMetadataImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "example.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(testPNG); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/", &body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	AddSyntheticPerformerMetadataImage(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("contains-synthetic-performer")) {
		t.Fatal("annotated image is missing the required keyword")
	}
}

func TestWriteDownloadedImageOnlyAnnotatesWhenRequested(t *testing.T) {
	gin.SetMode(gin.TestMode)
	imagePath := filepath.Join(t.TempDir(), "example.png")
	if err := os.WriteFile(imagePath, testPNG, 0644); err != nil {
		t.Fatal(err)
	}

	plainRecorder := httptest.NewRecorder()
	plainContext, _ := gin.CreateTestContext(plainRecorder)
	if err := writeDownloadedImage(plainContext, imagePath, "uploads/images/example.png", "example.png", false); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plainRecorder.Body.Bytes(), testPNG) {
		t.Fatal("download without annotation must preserve the original image bytes")
	}

	annotatedRecorder := httptest.NewRecorder()
	annotatedContext, _ := gin.CreateTestContext(annotatedRecorder)
	if err := writeDownloadedImage(annotatedContext, imagePath, "uploads/images/example.png", "example.png", true); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(annotatedRecorder.Body.Bytes(), []byte("contains-synthetic-performer")) {
		t.Fatal("download with annotation is missing the required keyword")
	}
}

func TestCopyDownloadToNASWritesAnnotatedImage(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, testPNG, 0644); err != nil {
		t.Fatal(err)
	}
	nasDir := filepath.Join(t.TempDir(), "nas")
	if err := copyDownloadToNAS("", sourcePath, nasDir, "download.png", true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(nasDir, "download.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("contains-synthetic-performer")) {
		t.Fatal("NAS copy is missing the required keyword")
	}
}

func TestWriteDownloadDataToNASPreservesAnnotatedBytes(t *testing.T) {
	annotated, err := storage.EnsureSyntheticPerformerXMP(testPNG, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	nasDir := filepath.Join(t.TempDir(), "nas")
	if err := writeDownloadDataToNAS(annotated, nasDir, "download.png"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(nasDir, "download.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, annotated) {
		t.Fatal("NAS copy does not match the annotated download bytes")
	}
}
