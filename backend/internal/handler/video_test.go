package handler

import "testing"

func TestArkVideoModel(t *testing.T) {
	modelID, slug, ok := arkVideoModel("seedance-2-fast")
	if ok || modelID != "" || slug != "" {
		t.Fatalf("legacy model should be rejected: %q %q %v", modelID, slug, ok)
	}
}

func TestArkVideoModelSeedance25(t *testing.T) {
	modelID, slug, ok := arkVideoModel("sed2.5")
	if !ok || modelID != "doubao-seedance-2-5-260628" || slug != "seedance-2-5" {
		t.Fatalf("unexpected Seedance 2.5 mapping: %q %q %v", modelID, slug, ok)
	}
}

func TestArkVideoRatio(t *testing.T) {
	ratio, ok := arkVideoRatio("widescreen_16_9")
	if !ok || ratio != "16:9" {
		t.Fatalf("unexpected legacy ratio mapping: %q %v", ratio, ok)
	}
}

func TestArkVideoImageContent(t *testing.T) {
	content := arkVideoImageContent("abc", "first_frame")
	if content == nil || content.ImageURL == nil || content.ImageURL.URL != "data:image/png;base64,abc" || content.Role != "first_frame" {
		t.Fatalf("unexpected image content: %#v", content)
	}
}

func TestArkVideoReferenceImageContent(t *testing.T) {
	content := arkVideoImageContent("data:image/jpeg;base64,abc", "reference_image")
	if content == nil || content.Role != "reference_image" || content.ImageURL.URL != "data:image/jpeg;base64,abc" {
		t.Fatalf("unexpected reference image content: %#v", content)
	}
}

func TestArkVideoMediaContent(t *testing.T) {
	video := arkVideoMediaContent("data:video/mp4;base64,abc", "video_url", "reference_video")
	if video == nil || video.VideoURL == nil || video.VideoURL.URL != "data:video/mp4;base64,abc" || video.Role != "reference_video" {
		t.Fatalf("unexpected video content: %#v", video)
	}
	audio := arkVideoMediaContent("https://example.com/reference.mp3", "audio_url", "reference_audio")
	if audio == nil || audio.AudioURL == nil || audio.AudioURL.URL != "https://example.com/reference.mp3" || audio.Role != "reference_audio" {
		t.Fatalf("unexpected audio content: %#v", audio)
	}
}
