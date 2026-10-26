package handler

import "testing"

func TestNormalizeImageToPromptImages(t *testing.T) {
	t.Run("legacy single image", func(t *testing.T) {
		images, err := normalizeImageToPromptImages(ImageToPromptRequest{ImageData: "one"})
		if err != nil {
			t.Fatal(err)
		}
		if len(images) != 1 || images[0].Data != "one" || images[0].MimeType != "image/png" {
			t.Fatalf("unexpected images: %#v", images)
		}
	})

	t.Run("multiple images", func(t *testing.T) {
		images, err := normalizeImageToPromptImages(ImageToPromptRequest{Images: []ImageToPromptImage{
			{ImageData: "one", MimeType: "image/jpeg"},
			{ImageData: "two", MimeType: "image/webp"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if len(images) != 2 || images[1].Data != "two" {
			t.Fatalf("unexpected images: %#v", images)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if _, err := normalizeImageToPromptImages(ImageToPromptRequest{}); err == nil {
			t.Fatal("expected empty input error")
		}
	})

	t.Run("too many images", func(t *testing.T) {
		inputs := make([]ImageToPromptImage, maxImageToPromptImages+1)
		for i := range inputs {
			inputs[i].ImageData = "image"
		}
		if _, err := normalizeImageToPromptImages(ImageToPromptRequest{Images: inputs}); err == nil {
			t.Fatal("expected image limit error")
		}
	})
}
