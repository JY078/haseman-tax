# 生图页面模型选择功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在生图页面添加模型选择下拉框，允许用户选择绘图模型而非强制使用系统默认

**Architecture:** 前端从 `/api/models?type=drawing` 获取活跃绘图模型列表，渲染为下拉框。切换模型时联动更新支持的分辨率选项。生成请求时传递 `model` 参数，后端已有支持。

**Tech Stack:** Vue 3 Composition API, TypeScript, Tailwind CSS

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `web/src/views/GenerateView.vue` | 添加模型选择下拉框，状态管理，API 调用，参数传递 |

---

## Task 1: 添加模型选择状态

**Files:**
- Modify: `web/src/views/GenerateView.vue:80-90`

- [ ] **Step 1: 在 `ref` 声明区域添加模型相关状态**

在 `const count = ref(1)` 附近添加：

```typescript
// 模型选择
const drawingModels = ref<any[]>([])
const selectedModel = ref<string>('')
```

- [ ] **Step 2: 添加 `loadDrawingModels` 函数**

在 `loadDefaultDrawingModel` 函数附近添加：

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

- [ ] **Step 3: 修改 `updateSupportedResolutions` 函数**

找到现有函数，修改为支持从模型获取分辨率：

```typescript
function updateSupportedResolutions() {
  // 如果当前选中的模型有 supported_resolutions，使用它
  const currentModel = drawingModels.value.find(m => m.name === selectedModel.value)
  if (currentModel?.supported_resolutions?.length) {
    supportedResolutions.value = currentModel.supported_resolutions
  } else {
    // 否则使用默认分辨率列表
    supportedResolutions.value = ['512', '1K', '2K', '4K']
  }
  // 如果当前选中的分辨率不在支持列表中，重置为第一个
  if (!supportedResolutions.value.includes(resolution.value)) {
    resolution.value = supportedResolutions.value[0] || '1K'
  }
}
```

- [ ] **Step 4: 在 `onMounted` 中调用 `loadDrawingModels`**

在 `onMounted` 回调中添加：

```typescript
onMounted(async () => {
  // ... 现有代码 ...
  await loadDrawingModels()  // 添加这行
})
```

- [ ] **Step 5: 提交**

```bash
git add web/src/views/GenerateView.vue
git commit -m "feat(generate): 添加模型选择状态和加载函数"
```

---

## Task 2: 添加模型选择下拉框 UI

**Files:**
- Modify: `web/src/views/GenerateView.vue:400-450`

- [ ] **Step 1: 在生成数量和输出分辨率之间添加模型选择下拉框**

找到输出分辨率的代码块，在其上方添加模型选择：

```html
<!-- 模型选择 -->
<div class="mb-4">
  <label class="block text-sm font-medium text-gray-700 mb-1">选择模型</label>
  <select
    v-model="selectedModel"
    class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
  >
    <option v-for="model in drawingModels" :key="model.id" :value="model.name">
      {{ model.name }}
    </option>
  </select>
</div>
```

- [ ] **Step 2: 提交**

```bash
git add web/src/views/GenerateView.vue
git commit -m "feat(generate): 添加模型选择下拉框UI"
```

---

## Task 3: 模型切换联动分辨率更新

**Files:**
- Modify: `web/src/views/GenerateView.vue:200-210`

- [ ] **Step 1: 监听 selectedModel 变化**

在 `watch` 区域添加：

```typescript
watch(selectedModel, () => {
  updateSupportedResolutions()
})
```

- [ ] **Step 2: 提交**

```bash
git add web/src/views/GenerateView.vue
git commit -m "feat(generate): 模型切换联动分辨率更新"
```

---

## Task 4: 生成请求传递 model 参数

**Files:**
- Modify: `web/src/views/GenerateView.vue:500-550`

- [ ] **Step 1: 修改 `generate` 函数中的请求参数**

找到 `const payload = {` 部分，添加 `model` 字段：

```typescript
const payload = {
  prompt: prompt.value,
  aspect_ratio: aspectRatio.value,
  model: selectedModel.value,  // 添加这行
  // ... 其他字段
}
```

- [ ] **Step 2: 修改 `doGenerateSingle` 函数中的请求参数**

找到该函数中的 payload，添加 `model` 字段：

```typescript
const payload = {
  prompt: prompt.value,
  aspect_ratio: aspectRatio.value,
  model: selectedModel.value,  // 添加这行
  // ... 其他字段
}
```

- [ ] **Step 3: 检查并修改 `handleEditSubmit` 函数**

找到该函数，确保传递 `model` 参数：

```typescript
const payload = {
  prompt: editPrompt.value,
  aspect_ratio: editAspectRatio.value,
  model: selectedModel.value,  // 添加这行
  // ... 其他字段
}
```

- [ ] **Step 4: 提交**

```bash
git add web/src/views/GenerateView.vue
git commit -m "feat(generate): 生成请求传递model参数"
```

---

## Task 5: 验证与测试

**Files:**
- Test: 手动测试

- [ ] **Step 1: 验证模型列表加载**

打开生图页面，确认下拉框显示所有活跃绘图模型

- [ ] **Step 2: 验证默认选中**

确认默认选中 `is_default=true` 的模型

- [ ] **Step 3: 验证分辨率联动**

切换不同模型，确认分辨率选项相应变化

- [ ] **Step 4: 验证图片生成**

选择不同模型生成图片，确认后端正确使用选中的模型

---

## 验收标准

- [ ] 生图页面显示模型选择下拉框
- [ ] 下拉框列出所有活跃的绘图模型
- [ ] 默认选中系统默认的绘图模型
- [ ] 切换模型时，分辨率选项相应更新
- [ ] 生成图片时使用选中的模型