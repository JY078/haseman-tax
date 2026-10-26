package handler

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomImagePreservesExactOutput(t *testing.T) {
	var input bytes.Buffer
	if err := png.Encode(&input, image.NewNRGBA(image.Rect(0, 0, 2928, 1200))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(input.Bytes())
	}))
	defer server.Close()
	result, err := resizeCustomImageResult(imageGenerationAdapterResult{ImageURL: server.URL, MimeType: "image/png"}, 2928, 1200)
	if err != nil {
		t.Fatal(err)
	}
	if result.ImageURL != server.URL || result.ImageURLs[0] != server.URL || result.MimeType != "image/png" {
		t.Fatalf("native result modified: %+v", result)
	}
}

func TestValidateCustomImageSize(t *testing.T) {
	for _, size := range [][2]int{{0, 0}, {2928, 1200}, {8192, 64}} {
		if err := validateCustomImageSize(size[0], size[1]); err != nil {
			t.Fatal(err)
		}
	}
	for _, size := range [][2]int{{0, 1200}, {63, 1200}, {-1, 1200}, {8193, 64}, {8192, 8192}} {
		if validateCustomImageSize(size[0], size[1]) == nil {
			t.Fatalf("accepted invalid size %v", size)
		}
	}
}

func TestResizeCustomImage(t *testing.T) {
	for _, target := range [][2]int{{2928, 1200}, {1200, 2928}, {128, 128}} {
		src := image.NewNRGBA(image.Rect(0, 0, 210, 90))
		// Coordinate colors expose off-center cropping and nonuniform scaling.
		for y := 0; y < 90; y++ {
			for x := 0; x < 210; x++ {
				src.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 0, 255})
			}
		}
		var input bytes.Buffer
		if err := png.Encode(&input, src); err != nil {
			t.Fatal(err)
		}
		output, err := resizeCustomImage(input.Bytes(), target[0], target[1])
		if err != nil {
			t.Fatal(err)
		}
		got, err := png.Decode(bytes.NewReader(output))
		if err != nil {
			t.Fatal(err)
		}
		if got.Bounds().Dx() != target[0] || got.Bounds().Dy() != target[1] {
			t.Fatalf("wrong size: %v", got.Bounds())
		}
		center := color.NRGBAModel.Convert(got.At(target[0]/2, target[1]/2)).(color.NRGBA)
		if center.R < 103 || center.R > 106 || center.G < 43 || center.G > 46 {
			t.Fatalf("crop not centered: %v", center)
		}
		left := color.NRGBAModel.Convert(got.At(target[0]/2-30, target[1]/2)).(color.NRGBA)
		top := color.NRGBAModel.Convert(got.At(target[0]/2, target[1]/2-30)).(color.NRGBA)
		delta := int(center.R) - int(left.R) - (int(center.G) - int(top.G))
		if delta < -1 || delta > 1 {
			t.Fatalf("unequal axis scales: %d", delta)
		}
	}
	if _, err := resizeCustomImage([]byte("invalid"), 2928, 1200); err == nil {
		t.Fatal("accepted corrupt image")
	}
}
