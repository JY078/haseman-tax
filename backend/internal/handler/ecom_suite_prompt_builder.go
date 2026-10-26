package handler

import (
	"fmt"
	"strings"

	"advertising-image/backend/internal/model"
)

// joinNonEmpty 过滤空字符串后拼接
func joinNonEmpty(sep string, parts ...string) string {
	var filtered []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, sep)
}

// buildSceneDescription 从结构化字段构建场景描述
// 核心逻辑：以 visual_strategy + layout_instruction + global_style 为输入，
// 程序化组装适合生图的场景描述
func buildSceneDescription(imageType string, visualStrategy string, layoutInstruction string, globalStyle model.EcomSuiteGlobalStyle, outputRatio string) string {
	var parts []string

	// 从 visual_strategy 提取场景元素（核心）
	if visualStrategy != "" {
		parts = append(parts, formatVisualStrategyForPrompt(visualStrategy, imageType))
	}

	// 从 layout_instruction 提取构图元素
	if layoutInstruction != "" {
		parts = append(parts, formatLayoutInstructionForPrompt(layoutInstruction, imageType))
	}

	// 从 global_style 补充环境基调
	if globalStyle.PrimaryColor != "" || globalStyle.BackgroundTone != "" {
		parts = append(parts, formatGlobalStyleForPrompt(globalStyle, imageType))
	}

	// 从场景主色和环境辅助色补充颜色信息
	if planColor := extractImagePlanColor(imageType, globalStyle); planColor != "" {
		parts = append(parts, planColor)
	}

	// 类型特定的场景模板（当所有结构化字段都为空时的回退）
	if len(parts) == 0 {
		parts = append(parts, getSceneDescriptionTemplate(imageType, outputRatio))
	}

	return strings.Join(parts, " ")
}

// extractImagePlanColor 从 global_style 提取当前图片的颜色信息
func extractImagePlanColor(imageType string, globalStyle model.EcomSuiteGlobalStyle) string {
	// global_style 的 primary_color 和 secondary_color 是整套图的场景基调，
	// 不是产品自身的颜色，也不是品牌固定色。
	var colors []string

	if globalStyle.PrimaryColor != "" {
		colors = append(colors, globalStyle.PrimaryColor)
	}
	if globalStyle.SecondaryColor != "" {
		colors = append(colors, globalStyle.SecondaryColor)
	}

	if len(colors) == 0 {
		return ""
	}

	return fmt.Sprintf("The category-led scene palette features %s tones; use these for environmental surfaces, accents and typography while keeping the uploaded product's original appearance unchanged.", strings.Join(colors, " and "))
}

// formatVisualStrategyForPrompt 将 visual_strategy 从自然语言转换为生图 Prompt 风格
// 核心：把命令式描述转换为描述性段落
func formatVisualStrategyForPrompt(strategy string, imageType string) string {
	// 清理输入
	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		return ""
	}

	// 检测并转换命令式前缀
	strategy = convertCommandToDescriptive(strategy)

	// 确保以描述性语句开头（不是命令）
	strategy = ensureDescriptiveSentence(strategy)

	// 根据图片类型添加特定后缀
	strategy = addImageTypeSuffix(strategy, imageType)

	return strategy
}

// convertCommandToDescriptive 将命令式描述转换为描述性
// 例如："Use modern kitchen" -> "A modern kitchen scene with"
// "Create a luxury bathroom" -> "A luxury bathroom setting featuring"
func convertCommandToDescriptive(s string) string {
	lower := strings.ToLower(s)

	// 命令词到描述词映射
	conversions := map[string]string{
		"use ":       "featuring ",
		"add ":       "with ",
		"include ":   "featuring ",
		"apply ":     "with ",
		"create a ":  "a ",
		"create an ": "an ",
		"create ":    "featuring ",
		"make ":      "with ",
		"place ":     "with ",
		"show ":      "showing ",
		"display ":   "featuring ",
	}

	for cmd, replacement := range conversions {
		if strings.HasPrefix(lower, cmd) {
			return replacement + s[len(cmd):]
		}
	}

	return s
}

// ensureDescriptiveSentence 确保字符串以描述性语句开头
func ensureDescriptiveSentence(s string) string {
	// 如果以动词祈使句开头，添加适当的前缀使其成为描述性语句
	lower := strings.ToLower(s)

	imperativePrefixes := []string{"must ", "should ", "need to ", "ensure ", "make sure "}
	for _, prefix := range imperativePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return "The scene features " + s[len(prefix):]
		}
	}

	// 如果是完整的句子（以大写开头或已有介词开头），保持原样
	firstChar := s[0]
	if firstChar >= 'A' && firstChar <= 'Z' {
		return s
	}

	// 如果已经是介词短语或名词短语，直接返回
	if strings.HasPrefix(lower, "with ") ||
		strings.HasPrefix(lower, "featuring ") ||
		strings.HasPrefix(lower, "showing ") ||
		strings.HasPrefix(lower, "a ") ||
		strings.HasPrefix(lower, "an ") ||
		strings.HasPrefix(lower, "the ") ||
		strings.HasPrefix(lower, "in ") ||
		strings.HasPrefix(lower, "on ") ||
		strings.HasPrefix(lower, "for ") {
		return s
	}

	// 其他情况添加 "featuring " 前缀
	return "featuring " + s
}

// addImageTypeSuffix 根据图片类型添加场景后缀
func addImageTypeSuffix(scene string, imageType string) string {
	// 检查是否已经包含特定类型的后缀
	lowerScene := strings.ToLower(scene)

	switch imageType {
	case "a_plus":
		if !strings.Contains(lowerScene, "banner") && !strings.Contains(lowerScene, "lifestyle") {
			return scene + ", creating an elevated lifestyle photography feel with premium composition"
		}
	case "feature":
		if !strings.Contains(lowerScene, "feature") && !strings.Contains(lowerScene, "product") {
			return scene + ", showcasing the product feature prominently"
		}
	case "detail":
		if !strings.Contains(lowerScene, "detail") && !strings.Contains(lowerScene, "close-up") && !strings.Contains(lowerScene, "macro") {
			return scene + ", with detailed texture and material clarity"
		}
	case "scene":
		if !strings.Contains(lowerScene, "lifestyle") && !strings.Contains(lowerScene, "setting") {
			return scene + ", creating an aspirational lifestyle atmosphere"
		}
	}

	return scene
}

// formatLayoutInstructionForPrompt 将 layout_instruction 转换为生图 Prompt 风格
// 核心：从排版说明中提取构图元素
func formatLayoutInstructionForPrompt(layout string, imageType string) string {
	layout = strings.TrimSpace(layout)
	if layout == "" {
		return ""
	}

	var parts []string

	// 提取产品位置
	productPlacement := extractProductPlacement(layout)
	if productPlacement != "" {
		parts = append(parts, productPlacement)
	}

	// 提取构图结构
	composition := extractComposition(layout)
	if composition != "" {
		parts = append(parts, composition)
	}

	// 提取文案区域位置
	textArea := extractTextArea(layout)
	if textArea != "" {
		parts = append(parts, textArea)
	}

	// 提取空间分布
	spaceDist := extractSpaceDistribution(layout)
	if spaceDist != "" {
		parts = append(parts, spaceDist)
	}

	if len(parts) == 0 {
		// 如果没有提取到特定元素，直接使用 layout 描述（转为描述性）
		return convertCommandToDescriptive(layout)
	}

	return strings.Join(parts, ". ") + "."
}

// extractProductPlacement 从 layout_instruction 中提取产品位置描述
func extractProductPlacement(layout string) string {
	lower := strings.ToLower(layout)

	// 检测产品位置关键词
	positionIndicators := []struct {
		keywords []string
		pattern  string
	}{
		{
			keywords: []string{"左侧", "左边", "left side", "left-hand", "left portion"},
			pattern:  "Product positioned on the left side",
		},
		{
			keywords: []string{"右侧", "右边", "right side", "right-hand", "right portion"},
			pattern:  "Product positioned on the right side",
		},
		{
			keywords: []string{"中间", "中央", "居中", "center", "middle", "centre"},
			pattern:  "Product centered in the frame",
		},
		{
			keywords: []string{"上方", "上部", "top", "upper"},
			pattern:  "Product positioned in the upper area",
		},
		{
			keywords: []string{"下方", "下部", "底部", "bottom", "lower"},
			pattern:  "Product positioned in the lower area",
		},
		{
			keywords: []string{"1/3", "三分之一", "one-third"},
			pattern:  "Product positioned at the golden ratio point",
		},
		{
			keywords: []string{"前景", "foreground", "front"},
			pattern:  "Product in the foreground with clear visibility",
		},
		{
			keywords: []string{"背景", "background"},
			pattern:  "Product positioned in the background layer",
		},
	}

	for _, ind := range positionIndicators {
		for _, kw := range ind.keywords {
			if strings.Contains(lower, kw) {
				return ind.pattern
			}
		}
	}

	return ""
}

// extractComposition 从 layout_instruction 中提取构图结构描述
func extractComposition(layout string) string {
	lower := strings.ToLower(layout)

	compositionIndicators := []struct {
		keywords []string
		pattern  string
	}{
		{
			keywords: []string{"三分法", "rule of thirds", "thirds"},
			pattern:  "Rule of thirds composition",
		},
		{
			keywords: []string{"对称", "symmetric", "balanced"},
			pattern:  "Balanced symmetrical composition",
		},
		{
			keywords: []string{"对角线", "diagonal"},
			pattern:  "Diagonal composition",
		},
		{
			keywords: []string{"三角形", "triangular", "triangle"},
			pattern:  "Triangular composition",
		},
		{
			keywords: []string{"前景", "foreground", "背景", "background", "depth"},
			pattern:  "Strong sense of depth with foreground and background layers",
		},
		{
			keywords: []string{"景深", "depth of field", "bokeh"},
			pattern:  "Beautiful depth of field with soft background blur",
		},
		{
			keywords: []string{"留白", "negative space", "minimal"},
			pattern:  "Generous negative space for elegant composition",
		},
		{
			keywords: []string{"填充", "fill", "full frame", "edge-to-edge"},
			pattern:  "Full frame composition with product filling the space",
		},
	}

	for _, ind := range compositionIndicators {
		for _, kw := range ind.keywords {
			if strings.Contains(lower, kw) {
				return ind.pattern
			}
		}
	}

	return ""
}

// extractTextArea 从 layout_instruction 中提取文案区域描述
func extractTextArea(layout string) string {
	lower := strings.ToLower(layout)

	textAreaIndicators := []struct {
		keywords []string
		pattern  string
	}{
		{
			keywords: []string{"标题上方", "title top", "heading top"},
			pattern:  "Title positioned at the top",
		},
		{
			keywords: []string{"标题下方", "subtitle below", "副标题在"},
			pattern:  "Subtitle positioned below the main title",
		},
		{
			keywords: []string{"文字左侧", "text left"},
			pattern:  "Text area on the left",
		},
		{
			keywords: []string{"文字右侧", "text right"},
			pattern:  "Text area on the right",
		},
		{
			keywords: []string{"角标", "corner", "badge", "label"},
			pattern:  "Corner labels or badges included",
		},
	}

	for _, ind := range textAreaIndicators {
		for _, kw := range ind.keywords {
			if strings.Contains(lower, kw) {
				return ind.pattern
			}
		}
	}

	return ""
}

// extractSpaceDistribution 从 layout_instruction 中提取空间分布描述
func extractSpaceDistribution(layout string) string {
	lower := strings.ToLower(layout)

	spaceIndicators := []struct {
		keywords []string
		pattern  string
	}{
		{
			keywords: []string{"开阔", "spacious", "airy", "open"},
			pattern:  "Spacious and airy composition",
		},
		{
			keywords: []string{"紧凑", "compact", "dense", "tight"},
			pattern:  "Compact and efficient space usage",
		},
		{
			keywords: []string{"分层", "layer", "tier"},
			pattern:  "Multi-layered spatial layout",
		},
		{
			keywords: []string{"聚焦", "focus", "spotlight", "spotlight on"},
			pattern:  "Product as the focal point with surrounding space",
		},
	}

	for _, ind := range spaceIndicators {
		for _, kw := range ind.keywords {
			if strings.Contains(lower, kw) {
				return ind.pattern
			}
		}
	}

	return ""
}

// formatGlobalStyleForPrompt 将 global_style 格式化为生图 Prompt
func formatGlobalStyleForPrompt(globalStyle model.EcomSuiteGlobalStyle, imageType string) string {
	var parts []string

	// 背景基调
	if globalStyle.BackgroundTone != "" {
		parts = append(parts, fmt.Sprintf("Background: %s.", globalStyle.BackgroundTone))
	}

	// 光线风格
	if globalStyle.Lighting != "" {
		parts = append(parts, fmt.Sprintf("Lighting: %s.", globalStyle.Lighting))
	}

	// 颜色基调
	if globalStyle.PrimaryColor != "" {
		colorDesc := fmt.Sprintf("Scene primary color tones: %s", globalStyle.PrimaryColor)
		if globalStyle.SecondaryColor != "" {
			colorDesc += fmt.Sprintf(" with %s accents", globalStyle.SecondaryColor)
		}
		parts = append(parts, colorDesc+".")
	}

	return strings.Join(parts, " ")
}

// getSceneDescriptionTemplate 根据图片类型获取场景描述模板
// 作为当结构化字段为空时的回退
func getSceneDescriptionTemplate(imageType string, outputRatio string) string {
	switch imageType {
	case "main_image":
		return "Create a premium ecommerce main product image with clean studio lighting on a white or neutral background. Professional product photography quality with sharp focus and polished commercial appeal."

	case "a_plus":
		return buildAPlusTemplate(outputRatio)

	case "feature":
		return "Create an attractive feature showcase image with the product as the hero. Commercial product photography style with dynamic composition and clean background or lifestyle setting."

	case "detail":
		return "Create a detailed close-up product image showcasing material texture and craftsmanship. High-end studio photography with clear lighting that reveals surface details and quality finish."

	case "scene":
		return "Create a lifestyle scene image showing the product in an aspirational real-world setting. Premium lifestyle photography with natural light, beautiful composition, and aspirational atmosphere."

	case "comparison":
		return "Create a clean comparison visualization showing product characteristics. Professional illustration or photography style with clear visual hierarchy."

	case "spec_dimension":
		return "Create a clean specification diagram with clear dimensions and measurements. Technical illustration style with professional presentation."

	case "installation_steps":
		// 安装步骤图必须有明确的步骤展示
		switch outputRatio {
		case "21:9":
			return "Create a wide installation instruction banner showing multiple steps in a horizontal grid layout. Include step numbers, each step action with supporting product visual, and clear visual flow from left to right. The layout should accommodate at least 4-6 steps visibly. Professional instructional photography style."
		case "4:3":
			return "Create an installation guidance image with multiple steps displayed in a clear 2x3 or 3x2 grid layout. Each step cell shows step number, action description, and corresponding product visual. All steps from the installation manual must be visible. Clean instructional design style."
		case "4:5", "1:1":
			return "Create a step-by-step installation guide image with clear visual grid showing all installation steps. Include numbered steps, action descriptions, and product context for each step. Multi-step grid layout is required - all steps must be visible. Clean instructional photography style."
		default:
			return "Create an installation guidance image with clear multi-step visual instructions. Use a grid layout showing all steps from the product manual with step numbers and action descriptions. Professional instructional design."
		}

	case "manufacturing_process":
		return "Create a manufacturing quality showcase image highlighting production excellence. Professional industrial photography with emphasis on quality and precision."

	default:
		return "Create a professional ecommerce product image with commercial photography quality."
	}
}

// buildAPlusTemplate 构建 A+ 图场景模板
func buildAPlusTemplate(outputRatio string) string {
	base := "Create a premium Amazon A+ lifestyle banner photograph featuring "

	switch outputRatio {
	case "21:9":
		base += "the product naturally integrated into an elegant lifestyle setting. Wide panoramic composition with the product on the left third, lifestyle environment filling the right, and clean text zone for branding elements."
	case "4:3":
		base += "the product prominently in a sophisticated lifestyle context. Well-balanced composition with product focus and elegant environmental storytelling."
	case "4:5", "1:1":
		base += "the product as the hero in a visually appealing lifestyle scene. Centered or offset composition with natural lighting and premium atmosphere."
	default:
		base += "the product in an elevated lifestyle context with careful attention to composition and visual appeal."
	}

	return base + " High-end commercial photography quality with refined aesthetic."
}

// buildTypographyBlock 构建排版约束块
// 从 typography_instruction 提取关键排版要求
func buildTypographyBlock(imageType string, typographyInstruction string, outputRatio string) string {
	var parts []string

	// 基础排版要求（按图片类型）
	switch imageType {
	case "a_plus":
		switch outputRatio {
		case "21:9":
			parts = append(parts, "Typography: Bold large title on the left side with smaller subtitle below. Visual hierarchy with title as dominant element. Generous whitespace around text areas.")
		case "4:3":
			parts = append(parts, "Typography: Bold centered or slightly offset title with supporting subtitle. Compact but clear hierarchy suitable for the 4:3 frame.")
		default:
			parts = append(parts, "Typography: Clear title with supporting subtitle, properly sized for readability at various viewing sizes.")
		}
	case "feature":
		parts = append(parts, "Typography: Feature label or short callout text, sized appropriately (title approximately 4-8% of image height). Clean hierarchy without overpowering the product visual.")
	case "detail":
		parts = append(parts, "Typography: Text labels for detail areas, minimal and elegant. Should feel like professional annotation rather than busy labeling.")
	case "scene":
		parts = append(parts, "Typography: Subtle text overlay or corner badges if needed. Should enhance the lifestyle aesthetic without dominating the visual.")
	default:
		parts = append(parts, "Typography: Clear text hierarchy with properly sized elements for readability.")
	}

	// 从 typography_instruction 提取额外要求
	if typographyInstruction != "" {
		extraReq := extractTypographyRequirements(typographyInstruction)
		if extraReq != "" {
			parts = append(parts, extraReq)
		}
	}

	return strings.Join(parts, " ")
}

// extractTypographyRequirements 从 typography_instruction 提取具体排版要求
func extractTypographyRequirements(instruction string) string {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return ""
	}

	// 将中文排版指令转为英文生图约束
	// 这里主要是从指令中提取关键排版特征

	lower := strings.ToLower(instruction)

	var requirements []string

	// 字重
	if strings.Contains(lower, "粗体") || strings.Contains(lower, "bold") || strings.Contains(lower, "heavy") {
		requirements = append(requirements, "Bold typography weight for primary text")
	}
	if strings.Contains(lower, "细体") || strings.Contains(lower, "light") || strings.Contains(lower, "纤细") {
		requirements = append(requirements, "Light typography weight for secondary text")
	}

	// 字号
	if strings.Contains(lower, "大") && (strings.Contains(lower, "标题") || strings.Contains(lower, "title")) {
		requirements = append(requirements, "Large title typography as focal point")
	}
	if strings.Contains(lower, "小") && (strings.Contains(lower, "副标题") || strings.Contains(lower, "subtitle")) {
		requirements = append(requirements, "Smaller subtitle text")
	}

	// 间距
	if strings.Contains(lower, "间距") || strings.Contains(lower, "spacing") || strings.Contains(lower, "whitespace") {
		requirements = append(requirements, "Generous spacing around text elements for clean appearance")
	}

	// 层级
	if strings.Contains(lower, "层次") || strings.Contains(lower, "hierarchy") || strings.Contains(lower, "层级") {
		requirements = append(requirements, "Clear visual hierarchy between title, subtitle, and other text")
	}

	if len(requirements) == 0 {
		// 如果没有匹配到特定模式，添加通用要求
		return "Typography should follow professional ecommerce design standards."
	}

	return strings.Join(requirements, ". ") + "."
}
