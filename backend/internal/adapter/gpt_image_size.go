package adapter

import (
	"fmt"
	"math"
	"strings"
)

// GPTImageNativeSize follows the official model size constraints, not the preset ratio labels.
// https://developers.openai.com/api/docs/guides/image-generation
func GPTImageNativeSize(model string, width, height int) string {
	name := strings.ToLower(strings.TrimSpace(model))
	parts := strings.Split(name, "/")
	name = parts[len(parts)-1]
	if width <= 0 || height <= 0 {
		return ""
	}
	if name == "gpt-image-2" || strings.HasPrefix(name, "gpt-image-2-") {
		if width > 3840 || height > 3840 || width%16 != 0 || height%16 != 0 || width > height*3 || height > width*3 || width*height < 655360 || width*height > 8294400 {
			return ""
		}
	} else if name == "gpt-image-1" || name == "gpt-image-1-mini" || name == "gpt-image-1.5" {
		if !((width == 1024 && height == 1024) || (width == 1536 && height == 1024) || (width == 1024 && height == 1536)) {
			return ""
		}
	} else {
		return ""
	}
	return fmt.Sprintf("%dx%d", width, height)
}

func (a *GPTImageAdapter) imageParams(model, ratio, resolution string) GPTImageParams {
	params := MapGPTImageParams(ratio, resolution)
	if width, height := MatchGPTImageSize(model, a.OutputWidth, a.OutputHeight); width > 0 {
		params.Size = fmt.Sprintf("%dx%d", width, height)
	}
	return params
}

// Keep the search and scoring aligned with web/src/utils/customImageSize.js.
func MatchGPTImageSize(model string, width, height int) (int, int) {
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	if GPTImageNativeSize(model, width, height) != "" {
		return width, height
	}
	parts := strings.Split(strings.ToLower(strings.TrimSpace(model)), "/")
	name := parts[len(parts)-1]
	long, short := max(width, height), min(width, height)
	bestW, bestH, bestScore := 0, 0, math.Inf(1)
	consider := func(w, h int) {
		if width == height && w != h {
			return
		}
		dw, dh := math.Log(float64(w)/float64(long)), math.Log(float64(h)/float64(short))
		score := dw*dw + dh*dh
		if score < bestScore {
			bestW, bestH, bestScore = w, h, score
		}
	}
	if name == "gpt-image-2" || strings.HasPrefix(name, "gpt-image-2-") {
		for w := 16; w <= 3840; w += 16 {
			minH := int(math.Ceil(math.Max(float64(w)/3, 655360/float64(w))/16)) * 16
			for h := minH; h <= w && w*h <= 8294400; h += 16 {
				consider(w, h)
			}
		}
	} else if name == "gpt-image-1" || name == "gpt-image-1-mini" || name == "gpt-image-1.5" {
		consider(1024, 1024)
		consider(1536, 1024)
	}
	if width < height {
		return bestH, bestW
	}
	return bestW, bestH
}
