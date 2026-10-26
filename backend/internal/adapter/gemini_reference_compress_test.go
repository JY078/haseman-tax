package adapter

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"math/rand"
	"testing"
)

func noisyReference(t *testing.T, transparent bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256)), 255
		if transparent {
			img.Pix[i+3] = 128
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCompressGeminiReference(t *testing.T) {
	for _, transparent := range []bool{false, true} {
		input := noisyReference(t, transparent)
		for _, limit := range []int{len(input), 100_000, 10_000} {
			output, mimeType, err := compressGeminiReference(context.Background(), input, "image/png", limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(output) > limit {
				t.Fatalf("size %d exceeds %d", len(output), limit)
			}
			decoded, format, err := image.Decode(bytes.NewReader(output))
			if err != nil {
				t.Fatal(err)
			}
			if limit == len(input) {
				if !bytes.Equal(input, output) || mimeType != "image/png" {
					t.Fatal("small originals must remain unchanged")
				}
			} else if transparent {
				_, _, _, alpha := decoded.At(0, 0).RGBA()
				if format != "png" || mimeType != "image/png" || alpha == 65535 {
					t.Fatal("transparency lost")
				}
			} else if format != "jpeg" || mimeType != "image/jpeg" {
				t.Fatal("opaque oversized images should use JPEG")
			}
			if !transparent && limit == 100_000 && decoded.Bounds().Dx() != 256 {
				t.Fatal("quality reduction should preserve dimensions when sufficient")
			}
			if limit == 10_000 && decoded.Bounds().Dx() >= 256 {
				t.Fatal("expected resize when quality reduction is insufficient")
			}
		}
	}
}

func TestCompressGeminiReferenceRejectsInvalidAndCancelled(t *testing.T) {
	if _, _, err := compressGeminiReference(context.Background(), []byte("invalid image"), "image/png", 1); err == nil {
		t.Fatal("expected decode error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := compressGeminiReference(ctx, []byte("image"), "image/png", 100); err != context.Canceled {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
