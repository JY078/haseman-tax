package handler

import "strings"

func getSceneVariationText(sceneType string, productDescriptor string, variationIndex int) string {
	if productDescriptor == "" {
		productDescriptor = "target product"
	}
	category := strings.ToLower(strings.TrimSpace(productDescriptor))
	var variationRules []string
	switch {
	case strings.Contains(category, "厨房岛台") || strings.Contains(category, "岛台") || strings.Contains(category, "kitchen island"):
		variationRules = []string{
			"变化：强调岛台水槽、连续台面和开放式厨房背景关系，底座固定在岛台台面孔位。",
			"变化：展示更多岛台台面和周围开放空间，水龙头仍锚定岛台水槽区。",
			"变化：镜头靠近岛台水槽安装区，但保留连续台面、底座接触面和水槽内缘，不改变水龙头尺寸和岛台几何。",
			"变化：更换少量岛台杯具或食材道具，光线和金属高光保持一致。",
		}
	case strings.Contains(category, "厨房水槽") || strings.Contains(category, "厨房水龙头") || strings.Contains(category, "kitchen sink") || strings.Contains(category, "kitchen faucet"):
		variationRules = []string{
			"变化：强调水槽、台面和防溅墙关系，底座固定在台面孔位。",
			"变化：展示更多橱柜和台面，水龙头仍锚定水槽区。",
			"变化：镜头靠近水槽安装区，但保留连续台面、底座接触面和水槽内缘，不改变水龙头尺寸。",
			"变化：更换少量厨房道具，光线和金属高光保持一致。",
		}
	case strings.Contains(category, "浴室台盆") || strings.Contains(category, "浴室水龙头") || strings.Contains(category, "bathroom vanity") || strings.Contains(category, "bathroom faucet"):
		variationRules = []string{
			"变化：强调台盆、台面和镜墙关系，底座固定在台面孔位。",
			"变化：展示更多镜子、墙砖和柜体，水龙头仍锚定台盆区。",
			"变化：镜头靠近台盆安装区，但保留连续台面、底座接触面和台盆内缘，不改变水龙头尺寸。",
			"变化：更换少量洗漱道具或毛巾，光线和金属高光保持一致。",
		}
	case strings.Contains(category, "房车") || strings.Contains(category, "rv") || strings.Contains(category, "camper"):
		variationRules = []string{
			"变化：强调现代房车厨房的小型水槽、连续紧凑台面、轻量柜体和车内墙板关系，产品仍锚定厨房水槽区。",
			"变化：展示更多房车窗、明亮温馨光线、现代柜体和紧凑厨房空间，不能变成普通住宅厨房。",
			"变化：镜头靠近房车厨房水槽安装区，但保留紧凑台面、底座接触面和水槽内缘，不改变产品尺寸和房车空间比例。",
			"变化：更换少量杯具、餐具、咖啡杯或旅行用品，整体保持明亮温馨的现代房车厨房氛围。",
		}
	default:
		return ""
	}
	return productDescriptor + ". " + variationRules[variationIndex%len(variationRules)] + " 变体保护：只允许改变镜头距离、少量场景道具、光线或背景空间关系；目标产品外形、出水口形状、把手数量、把手位置、底座数量、颜色分区和安装位置必须完全保持参考图一致。水槽或台盆内口宽度必须明显大于水龙头整体宽度和出水口投影宽度，不能因为近景裁切让水槽或台盆看起来比水龙头窄。禁止新增背景水龙头、第二套龙头、镜中重复龙头或把产品改成常见款式。"
}
