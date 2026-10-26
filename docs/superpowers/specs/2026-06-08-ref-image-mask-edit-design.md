---
name: ref-image-mask-edit-design
description: 参考图片上传后添加画笔蒙版编辑功能
metadata:
  type: feature
  created: 2026-06-08
  status: draft
---

# 参考图片蒙版编辑功能设计

## 1. 概述

### 1.1 功能目标
在上传参考图片后，通过悬停显示的"编辑"按钮弹出画笔编辑界面，用户可以在原图上绘制蒙版区域。编辑完成后，将画笔轨迹作为蒙版数据保存，与原图配对提交。

### 1.2 用户场景
1. 用户上传参考图片
2. 鼠标悬停于缩略图，显示"编辑"按钮
3. 点击"编辑"，弹出 ImageEditModal
4. 用户在图片上绘制修改区域（画笔蒙版）
5. 点击"重新生成"，蒙版数据与原图配对保存
6. 提交生成时，原图+蒙版一起发送给后端

## 2. 数据结构

### 2.1 refImages 扩展
```typescript
interface RefImage {
  data: string        // base64 数据（无前缀）
  mimeType: string   // MIME 类型
  url: string         // 完整 data URL
  mask?: string       // 蒙版 base64 数据（无前缀），可选
}
```

### 2.2 ImageEditModal 提交数据
```typescript
interface EditSubmitData {
  prompt: string
  imageData: string       // 编辑后的完整图片 base64
  mimeType: string
  quality: string
  // 新增：蒙版数据（仅 ref 模式）
  maskData?: string
}
```

## 3. 组件改动

### 3.1 ImageEditModal.vue

#### Props 变更
| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `visible` | `Boolean` | `false` | 弹窗可见性 |
| `imageUrl` | `String` | `''` | 图片 URL |
| `originalPrompt` | `String` | `''` | 原始提示词 |
| `mode` | `'ref' \| 'result'` | `'result'` | 编辑模式 |

#### 行为变更
- **mode = 'result'`（默认）**：保持现有逻辑不变
- **mode = 'ref'`**：
  - 提交时在 emit('submit') 数据中增加 `maskData` 字段
  - `maskData` 为蒙版 canvas 的 base64 数据

### 3.2 GenerateView.vue

#### UI 改动
在参考图缩略图上添加编辑按钮：
- 悬停时显示（opacity 过渡）
- 红色编辑图标 + "编辑蒙版" 文字
- 点击后打开 ImageEditModal

```vue
<!-- 缩略图容器 -->
<div class="group relative overflow-hidden rounded-xl ...">
  <img :src="img.url" class="w-full h-full object-contain bg-white p-1.5" />
  
  <!-- 编辑按钮（悬停显示） -->
  <button
    class="absolute right-2 top-2 ... opacity-0 group-hover:opacity-100 ..."
    @click="openRefImageEdit(idx)"
  >
    <PencilIcon class="w-3.5 h-3.5" />
    编辑蒙版
  </button>
</div>
```

#### 新增状态
```typescript
// 参考图编辑弹窗
const refEditModalVisible = ref(false)
const refEditIndex = ref(-1)  // 当前编辑的 refImages 索引

// ImageEditModal 需要的数据
const refEditImageUrl = ref('')
```

#### 新增方法
```typescript
// 打开参考图编辑
function openRefImageEdit(idx: number) {
  refEditIndex.value = idx
  refEditImageUrl.value = refImages.value[idx].url
  refEditModalVisible.value = true
}

// 处理参考图编辑提交
async function handleRefEditSubmit({ prompt, imageData, mimeType, maskData }) {
  if (refEditIndex.value === -1) return
  
  // 保存蒙版数据到 refImages
  refImages.value[refEditIndex.value].mask = maskData
  
  // 关闭弹窗
  refEditModalVisible.value = false
  refEditIndex.value = -1
  refEditImageUrl.value = ''
  
  toast.success('蒙版已保存')
}
```

### 3.3 生成提交逻辑改动

#### generate() 函数改动
在构建 refPayload 时，检查每个 refImage 是否有 mask：

```typescript
const refPayload = refImages.value.map(img => {
  const payload: any = { data: img.data, mime_type: img.mimeType }
  
  // 如果有蒙版数据，合并原图+蒙版
  if (img.mask) {
    // 使用 canvas 合并原图和蒙版
    payload.data = mergeImageAndMask(img.url, img.mask)
    payload.mime_type = 'image/png'
  }
  
  return payload
})
```

#### mergeImageAndMask 函数
```typescript
async function mergeImageAndMask(imageUrl: string, maskBase64: string): Promise<string> {
  return new Promise(async (resolve, reject) => {
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    
    // 加载原图
    const img = await loadImage(imageUrl)
    canvas.width = img.width
    canvas.height = img.height
    
    // 绘制原图
    ctx.drawImage(img, 0, 0)
    
    // 加载蒙版
    const maskImg = await loadImage(`data:image/png;base64,${maskBase64}`)
    
    // 混合蒙版（保留透明区域为蒙版指示）
    ctx.globalCompositeOperation = 'destination-over'
    ctx.drawImage(maskImg, 0, 0)
    
    resolve(canvas.toDataURL('image/png').split(',')[1])
  })
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = src
  })
}
```

## 4. 交互流程

```
┌─────────────────────────────────────────────────────────────┐
│  用户上传参考图片                                           │
│       │                                                   │
│       ▼                                                   │
│  ┌─────────────┐                                         │
│  │ 显示缩略图   │                                         │
│  └─────────────┘                                         │
│       │                                                   │
│       ▼ (鼠标悬停)                                         │
│  ┌─────────────────┐                                      │
│  │ 显示"编辑蒙版"按钮│                                     │
│  └─────────────────┘                                      │
│       │                                                   │
│       ▼ (点击)                                             │
│  ┌─────────────────────┐                                 │
│  │ ImageEditModal 弹窗   │                                 │
│  │ - 显示原图            │                                 │
│  │ - 画笔工具绘制蒙版     │                                 │
│  │ - 输入提示词（可选）   │                                 │
│  └─────────────────────┘                                 │
│       │                                                   │
│       ▼ (提交)                                             │
│  refImages[idx].mask = 蒙版base64                          │
│       │                                                   │
│       ▼ (开始生成)                                         │
│  原图 + 蒙版 → 合并 → 发送给后端                           │
└─────────────────────────────────────────────────────────────┘
```

## 5. API 兼容性

### 5.1 蒙版格式
- 使用 PNG 格式的透明蒙版
- 蒙版图片中非透明区域表示需要关注的区域
- 提交时将原图和蒙版合并为一个 RGBA 图片

### 5.2 向后兼容
- 如果 refImages 中某项没有 mask 字段，按原有逻辑处理
- 后端无需改动，接收到的仍是原图+蒙版合并后的数据

## 6. 改动范围

| 文件 | 改动类型 | 预估行数 |
|------|----------|----------|
| `ImageEditModal.vue` | 修改 Props + emit 数据 | ~10 行 |
| `GenerateView.vue` | UI 改动 + 逻辑 | ~40 行 |
| 合计 | - | ~50 行 |

## 7. 测试要点

1. **上传流程**：上传参考图 → 显示缩略图
2. **编辑按钮**：悬停显示，点击弹出弹窗
3. **画笔功能**：绘制蒙版、撤销、清除
4. **蒙版保存**：提交后 refImages 对应项有 mask 字段
5. **生成提交**：有蒙版的图片正确合并发送
6. **向后兼容**：无蒙版的图片按原有逻辑处理

## 8. 注意事项

1. **蒙版大小**：蒙版 canvas 尺寸需与显示尺寸一致，提交时注意比例
2. **性能考虑**：蒙版 base64 数据可能较大，建议限制画布尺寸
3. **用户体验**：编辑按钮位置和样式需要与现有"删除"按钮保持一致