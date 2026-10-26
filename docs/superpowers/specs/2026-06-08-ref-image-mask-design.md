# 参考图画笔编辑功能设计

**日期**: 2026-06-08  
**状态**: 已批准  
**功能**: 在上传参考图片后，通过画笔标记蒙版区域，实现局部修改

---

## 1. 功能目标

在上传参考图片后，用户可以通过悬停显示的"编辑"按钮，弹出画笔编辑界面，在图片上使用画笔标记需要修改的区域。编辑完成后，画笔轨迹作为蒙版数据保存，与原图配对提交给后端进行局部生成。

---

## 2. 核心改动

### 2.1 数据结构扩展

参考图片数组 `refImages` 新增 `mask` 字段：

```typescript
// 现有结构
refImages = Array<{ data: string; mimeType: string; url: string }>

// 新增 mask 字段后
refImages = Array<{
  data: string        // 原图 base64 数据
  mimeType: string   // MIME 类型
  url: string         // 显示用 URL
  mask?: string      // 新增：蒙版 base64 数据（可选）
}>
```

### 2.2 ImageEditModal 改动

**Props 扩展**：
```typescript
// 新增 mode prop
const props = defineProps({
  // ... 现有 props
  mode: { type: String, default: 'result' }  // 'ref' | 'result'
})
```

**行为差异**：

| 模式 | 触发方式 | 提交返回 |
|------|----------|----------|
| `'result'` | 生成结果卡片"编辑"按钮 | `{ prompt, imageData, mimeType, quality }` |
| `'ref'` | 参考图缩略图"编辑"按钮 | `{ imageUrl, maskData, originalPrompt }` |

**提交逻辑调整**：
```typescript
async function submit() {
  // ... 现有合并逻辑

  if (props.mode === 'ref') {
    emit('submit', {
      imageUrl: props.imageUrl,
      maskData: drawCanvasRef.value.toDataURL('image/png').split(',')[1],
      originalPrompt: props.originalPrompt
    })
  } else {
    // 保持现有逻辑
    emit('submit', { prompt, imageData, mimeType, quality })
  }
}
```

### 2.3 GenerateView 改动

**缩略图编辑按钮**（第 58-69 行附近）：

```vue
<!-- 在现有删除按钮旁添加编辑按钮 -->
<button
  class="absolute left-2 top-2 ..."
  @click="removeRefImage(idx)">
  <XMarkIcon class="w-3.5 h-3.5" />
</button>
<!-- 新增：编辑按钮（悬停显示） -->
<button
  class="absolute left-2 top-2 ..."
  @click="editRefImage(idx)">
  <PencilIcon class="w-3.5 h-3.5" />
</button>
```

**新增状态**：
```typescript
const editingRefIndex = ref<number | null>(null)
```

**新增方法**：
```typescript
function editRefImage(idx: number) {
  editingRefIndex.value = idx
  editingRefItem.value = refImages.value[idx]
  refEditModalVisible.value = true
}

function handleRefEditSubmit({ imageUrl, maskData, originalPrompt }) {
  if (editingRefIndex.value !== null) {
    // 更新对应项的 mask
    refImages.value[editingRefIndex.value].mask = maskData
  }
  refEditModalVisible.value = false
  editingRefIndex.value = null
  toast.success('蒙版已保存')
}
```

**提交逻辑调整**（generate 函数）：
```typescript
async function generate() {
  // ... 现有逻辑

  const refPayload = refImages.value.map(img => {
    const payload: any = { data: img.data, mime_type: img.mimeType }
    // 如果有蒙版，合并到原图中一起发送
    if (img.mask) {
      // 蒙版数据会在提交时与原图合并
      payload.mask = img.mask
    }
    return payload
  })

  // ... 后续逻辑
}
```

### 2.4 蒙版数据发送

蒙版数据与原图配对发送，后端通过 `mask` 字段识别蒙版区域：

```typescript
// 提交时构造 payload
const payload = {
  prompt: promptText,
  aspect_ratio: aspectRatio.value,
  model: selectedModel.value,
  ref_images: refImages.value.map(img => ({
    data: img.data,
    mime_type: img.mimeType,
    mask: img.mask || undefined  // 可选蒙版数据
  }))
}
```

---

## 3. UI 改动

### 3.1 参考图缩略图区域

```
┌─────────────────────────────────────────┐
│  参考图片                    3/6        │
│  ┌───────┐ ┌───────┐ ┌───────┐        │
│  │       │ │       │ │       │        │
│  │  [1]  │ │  [2]  │ │  [3]  │        │
│  │       │ │       │ │       │        │
│  └───────┘ └───────┘ └───────┘        │
│                           ✏️ 编辑      │  ← 悬停时显示
└─────────────────────────────────────────┘
```

**交互说明**：
- 缩略图悬停时显示：
  - 右上角：删除按钮（现有）
  - 左上角：编号徽章（现有）
  - 右下角："编辑"按钮（新增）
- 点击"编辑"按钮弹出编辑弹窗

### 3.2 编辑弹窗

复用现有 ImageEditModal，mode 为 `'ref'` 时：
- 标题改为"编辑参考图"
- 提示文字改为"在图片上涂抹标记需要修改的区域"
- 提交按钮文字改为"保存蒙版"
- 提交后关闭弹窗，保存蒙版数据到 refImages

---

## 4. 后端接口要求

后端 `/image/generate-async` 和 `/image/generate` 接口需要支持 `ref_images` 中的 `mask` 字段：

```json
{
  "prompt": "...",
  "ref_images": [
    {
      "data": "base64原图",
      "mime_type": "image/png",
      "mask": "base64蒙版"  // 可选
    }
  ]
}
```

---

## 5. 文件改动清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `ImageEditModal.vue` | 修改 | 新增 `mode` prop 和 ref 模式提交逻辑 |
| `GenerateView.vue` | 修改 | 缩略图添加编辑按钮、新增编辑方法、调整提交逻辑 |

---

## 6. 风险与注意事项

1. **蒙版数据大小**：蒙版 base64 可能较大，需要确保不超过后端限制
2. **Canvas 尺寸**：ImageEditModal 中的 Canvas 会有缩放，提交时需要考虑尺寸一致性
3. **兼容性**：复用现有 ImageEditModal，确保不影响现有编辑功能