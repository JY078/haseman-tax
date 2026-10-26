# 生图页面模型选择功能设计

**日期**: 2026-05-22

## 1. 需求背景

当前生图页面 (`GenerateView.vue`) 使用后台管理设置的默认绘图模型进行图片生成。用户希望能够自主选择不同的绘图模型。

## 2. 设计方案

### 2.1 前端改动 (GenerateView.vue)

**新增模型选择下拉框**

- 位置：放在"生成数量"之前、"输出分辨率"之后
- 显示范围：仅显示 `is_active=true` 且 `types` 包含 `drawing` 的活跃模型
- 默认选中：跟随系统默认（即 `is_default=true` 的模型）

**模型列表数据结构**

```typescript
interface DrawingModel {
  id: number
  name: string              // 模型名称（如 "gemini-2.0-flash-exp"）
  types: string[]           // 类型数组，如 ["chat", "drawing"]
  is_default: boolean       // 是否为系统默认
  is_active: boolean        // 是否活跃
  supported_resolutions?: string[]  // 支持的分辨率
}
```

**新增状态**

```typescript
const drawingModels = ref<DrawingModel[]>([])
const selectedModel = ref<string>('')
```

**获取模型列表**

```typescript
async function loadDrawingModels() {
  try {
    const res = await request.get('/models', { params: { type: 'drawing' } })
    // 过滤仅活跃的绘图模型
    drawingModels.value = (res.data || []).filter((m: any) =>
      m.is_active && m.types?.includes('drawing')
    )
    // 设置默认选中
    const defaultModel = drawingModels.value.find((m: any) => m.is_default)
    selectedModel.value = defaultModel?.name || drawingModels.value[0]?.name || ''
    // 更新支持的分辨率
    updateSupportedResolutions()
  } catch {
    drawingModels.value = []
  }
}
```

**分辨率联动**

- 切换模型时，更新 `supportedResolutions` 为选中模型支持的分辨率
- 如果当前选中的分辨率不在新模型的支撑列表中，重置为第一个

**请求参数传递**

- `generate()` 和 `handleEditSubmit()` 在提交时传递 `model` 参数
- 异步任务 `doGenerateSingle()` 也需传递模型参数

### 2.2 后端改动

**无需后端改动**

后端 API (`GenerateImage`, `CreateImageGenerationTask`) 已支持接收 `model` 参数：

```go
type ImageGenerateRequest struct {
    Prompt       string         `json:"prompt" binding:"required"`
    Model        string         `json:"model"`      // ← 已支持
    AspectRatio  string         `json:"aspect_ratio"`
    Resolution   string         `json:"resolution"`
    RefImages    []RefImageItem `json:"ref_images"`
    ...
}
```

当 `model` 为空时，后端自动使用 `getDefaultModelByType("drawing", "")` 获取默认模型。

## 3. UI 设计

```
┌──────────────────────────────────────────┐
│  参考图片                                 │
│  [拖拽/上传区域]                           │
├──────────────────────────────────────────┤
│  提示词 *                                 │
│  [多行文本框]                              │
├──────────────────────────────────────────┤
│  图片比例                                 │
│  [1:1] [4:3] [3:4] [16:9] [9:16] [21:9]  │
├──────────────────────────────────────────┤
│  生成数量                                 │
│  [1] [2] [3] [4]                         │
├──────────────────────────────────────────┤
│  选择模型 ▼                               │  ← 新增下拉框
│  [gemini-2.0-flash-exp      ▼]           │
├──────────────────────────────────────────┤
│  输出分辨率 (仅当模型支持时显示)            │
│  [512] [1K] [2K] [4K]                    │
├──────────────────────────────────────────┤
│  [开始生成]                               │
└──────────────────────────────────────────┘
```

## 4. 实现计划

### 步骤 1: 后端确认
- [x] 确认 `GET /models` 接口返回活跃绘图模型（已确认）
- [x] 确认 `POST /image/generate` 支持 `model` 参数（已确认）
- [x] 确认 `POST /image/generate-async` 支持 `model` 参数（已确认）

### 步骤 2: 前端模型选择下拉框
- 添加模型选择下拉框 UI 组件
- 添加 `drawingModels` 和 `selectedModel` 状态
- 实现 `loadDrawingModels()` 获取模型列表
- 在 `onMounted` 中调用 `loadDrawingModels()`

### 步骤 3: 分辨率联动
- 切换模型时更新 `supportedResolutions`
- 如果当前分辨率不在支持列表中，重置为第一个

### 步骤 4: 生成请求参数
- 修改 `generate()` 传递 `model` 参数
- 修改 `doGenerateSingle()` 传递 `model` 参数
- 修改 `handleEditSubmit()` 传递 `model` 参数

## 5. 风险与注意事项

1. **后端兼容性**：后端已支持 `model` 参数，无需修改
2. **默认值处理**：如果数据库中没有默认模型，前端默认选中第一个活跃模型
3. **分辨率重置**：切换模型时，如果当前分辨率不在新模型支持列表中，自动重置
4. **向后兼容**：不传递 `model` 参数时，后端自动使用默认模型

## 6. 验收标准

- [ ] 生图页面显示模型选择下拉框
- [ ] 下拉框列出所有活跃的绘图模型
- [ ] 默认选中系统默认的绘图模型
- [ ] 切换模型时，分辨率选项相应更新
- [ ] 生成图片时使用选中的模型
