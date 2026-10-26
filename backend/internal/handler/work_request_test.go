package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestValidateWorkRequestInput(t *testing.T) {
	tests := []struct {
		name    string
		input   workRequestInput
		wantErr string
	}{
		{
			name: "video shot durations must match total",
			input: workRequestInput{Type: "video", Title: "短视频", Content: mustJSON(t, videoRequestContent{
				DurationSeconds: 15,
				Shots:           []videoRequestShot{{DurationSeconds: 6, DisplayRequirement: "开场"}, {DurationSeconds: 8, DisplayRequirement: "产品"}},
			})},
			wantErr: "画面时长合计",
		},
		{
			name: "grouped storyboard is valid",
			input: workRequestInput{Type: "video", Title: "新品视频", Content: mustJSON(t, videoRequestContent{
				DurationSeconds: 15,
				ShotGroups: []videoRequestShotGroup{
					{Title: "产品开场", Shots: []videoRequestShot{{DurationSeconds: 5, Scene: "产品进入画面", Subtitle: "新品上市"}}},
					{Title: "卖点展示", Note: "保持节奏紧凑", Shots: []videoRequestShot{{DurationSeconds: 10, Scene: "依次展示三个卖点"}}},
				},
			})},
		},
		{
			name: "group must contain a small shot",
			input: workRequestInput{Type: "video", Title: "新品视频", Content: mustJSON(t, videoRequestContent{
				DurationSeconds: 15,
				ShotGroups:      []videoRequestShotGroup{{Title: "产品开场"}},
			})},
			wantErr: "至少需要一个画面",
		},
		{
			name: "small shot scene is required",
			input: workRequestInput{Type: "video", Title: "新品视频", Content: mustJSON(t, videoRequestContent{
				DurationSeconds: 15,
				ShotGroups:      []videoRequestShotGroup{{Title: "产品开场", Shots: []videoRequestShot{{DurationSeconds: 15}}}},
			})},
			wantErr: "画面 1",
		},
		{
			name:    "other request types are rejected",
			input:   workRequestInput{Type: "image", Title: "图片需求", Content: mustJSON(t, videoRequestContent{})},
			wantErr: "仅支持视频需求",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateWorkRequestInput(tt.input)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestWorkRequestNumbering(t *testing.T) {
	when := time.Date(2026, 8, 12, 9, 0, 0, 0, time.Local)
	if got := formatWorkRequestNo(when, workRequestNextSequence("REQ-20260812-009")); got != "REQ-20260812-010" {
		t.Fatalf("got %q", got)
	}
	if got := workRequestNextSequence(""); got != 1 {
		t.Fatalf("empty sequence = %d", got)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
