package model

import (
	"encoding/json"
	"testing"
)

func TestProductPromptRenderStatus(t *testing.T) {
	p := ProductPrompt{
		Name:        "Test Product",
		StpFileURL:  "oss://products/stp/123.stp",
		RenderStatus: "pending",
		ReferenceImages: []ReferenceImage{
			{URL: "oss://products/6view_front.png", Description: "[六面-正面]"},
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal product prompt: %v", err)
	}

	var p2 ProductPrompt
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("unmarshal product prompt: %v", err)
	}

	if p2.StpFileURL != p.StpFileURL {
		t.Errorf("StpFileURL mismatch: got %s, want %s", p2.StpFileURL, p.StpFileURL)
	}
	if p2.RenderStatus != p.RenderStatus {
		t.Errorf("RenderStatus mismatch: got %s, want %s", p2.RenderStatus, p.RenderStatus)
	}
}
