# 参考图画笔编辑功能实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在参考图片上传后，支持通过悬停显示的"编辑"按钮弹出画笔编辑界面，将画笔轨迹作为蒙版数据保存，与原图配对提交生成。

**Architecture:** 复用现有 ImageEditModal 组件，通过 `mode` prop 区分 `'ref'`（参考图编辑）和 `'result'`（生成结果编辑）两种模式。refImages 数据结构增加 `mask` 字段存储蒙版 base64。

**Tech Stack:** Vue 3 Composition API + Canvas API + TypeScript

---

## 文件结构

| 文件 | 改动类型 | 职责 |
|------|----------|------|
| `web/src/components/ui/ImageEditModal.vue` | 修改 | 新增 `mode` prop，调整提交返回格式 |
| `web/src/views/GenerateView.vue` | 修改 | 缩略图添加编辑按钮，处理编辑回调，更新提交逻辑 |

---

## Task 1: ImageEditModal 新增 mode prop

**Files:**
- Modify: `web/src/components/ui/ImageEditModal.vue:280-286`
- Modify: `web/src/components/ui/ImageEditModal.vue:31-33` (Header)
- Modify: `web/src/components/ui/ImageEditModal.vue:253-257` (Submit button)
- Modify: `web/src/components/ui/ImageEditModal.vue:524-549` (submit function)

- [ ] **Step 1: 在 props 中新增 mode 属性**

文件路径: `web/src/components/ui/ImageEditModal.vue:280-286`

将:
```typescript
const props = defineProps({
  visible: { type: Boolean, default: false },
  imageUrl: { type: String, default: '' },
  originalPrompt: { type: String, default: '' },
})
```

改为:
```typescript
const props = defineProps({
  visible: { type: Boolean, default: false },
  imageUrl: { type: String, default: '' },
  originalPrompt: { type: String, default: '' },
  mode: { type: String, default: 'result' },  // 'ref' | 'result'
})
```

- [ ] **Step 2: Header 区域根据 mode 调整文案**

文件路径: `web/src/components/ui/ImageEditModal.vue:31-33`

将:
```vue
<h3 class="font-semibold text-neutral-900">编辑图片</h3>
<p class="text-xs text-neutral-400 mt-0.5">用画笔标记修改区域，输入新的描述词后重新生成</p>
```

改为:
```vue
<h3 class="font-semibold text-neutral-900">{{ mode === 'ref' ? '编辑参考图' : '编辑图片' }}</h3>
<p class="text-xs text-neutral-400 mt-0.5">
  {{ mode === 'ref' ? '在图片上涂抹标记需要修改的区域' : '用画笔标记修改区域，输入新的描述词后重新生成' }}
</p>
```

- [ ] **Step 3: 提交按钮文案根据 mode 调整**

文件路径: `web/src/components/ui/ImageEditModal.vue:253-257`

将:
```vue
<template v-else>
  <SparklesIcon class="w-4 h-4" />
  重新生成
</template>
```

改为:
```vue
<template v-else>
  <SparklesIcon class="w-4 h-4" />
  {{ mode === 'ref' ? '保存蒙版' : '重新生成' }}
</template>
```

- [ ] **Step 4: submit 函数根据 mode 调整 emit 数据**

文件路径: `web/src/components/ui/ImageEditModal.vue:524-549`

将:
```typescript
async function submit() {
  if (isSubmitting.value) return
  isSubmitting.value = true

  try {
    /** Merge image layer + drawing layer into one canvas */
    const mergeCanvas = document.createElement('canvas')
    mergeCanvas.width = displayW.value
    mergeCanvas.height = displayH.value
    const ctx = mergeCanvas.getContext('2d')
    ctx.drawImage(imageCanvasRef.value, 0, 0)
    ctx.drawImage(drawCanvasRef.value, 0, 0)

    const dataUrl = mergeCanvas.toDataURL('image/png')
    const base64 = dataUrl.split(',')[1]

    emit('submit', {
      prompt: editPrompt.value.trim() || props.originalPrompt,
      imageData: base64,
      mimeType: 'image/png',
      quality: quality.value,
    })
  } finally {
    isSubmitting.value = false
  }
}
```

改为:
```typescript
async function submit() {
  if (isSubmitting.value) return
  isSubmitting.value = true

  try {
    if (props.mode === 'ref') {
      // ref 模式：只发送蒙版数据（原图保留）
      const maskData = drawCanvasRef.value.toDataURL('image/png').split(',')[1]
      emit('submit', {
        imageUrl: props.imageUrl,
        maskData: maskData,
        originalPrompt: props.originalPrompt,
      })
    } else {
      // result 模式：保持现有逻辑
      /** Merge image layer + drawing layer into one canvas */
      const mergeCanvas = document.createElement('canvas')
      mergeCanvas.width = displayW.value
      mergeCanvas.height = displayH.value
      const ctx = mergeCanvas.getContext('2d')
      ctx.drawImage(imageCanvasRef.value, 0, 0)
      ctx.drawImage(drawCanvasRef.value, 0, 0)

      const dataUrl = mergeCanvas.toDataURL('image/png')
      const base64 = dataUrl.split(',')[1]

      emit('submit', {
        prompt: editPrompt.value.trim() || props.originalPrompt,
        imageData: base64,
        mimeType: 'image/png',
        quality: quality.value,
      })
    }
  } finally {
    isSubmitting.value = false
  }
}
```

- [ ] **Step 5: 提交代码**

```bash
git add web/src/components/ui/ImageEditModal.vue
git commit -m "feat(ImageEditModal): 新增 mode prop 支持参考图编辑模式"
```

---

## Task 2: GenerateView 缩略图添加编辑按钮

**Files:**
- Modify: `web/src/views/GenerateView.vue:56-72` (缩略图模板)
- Modify: `web/src/views/GenerateView.vue:642` (refImages 类型)
- Modify: `web/src/views/GenerateView.vue:1032-1034` (新增状态)

- [ ] **Step 1: refImages 类型增加 mask 字段**

文件路径: `web/src/views/GenerateView.vue:642`

将:
```typescript
const refImages = ref<Array<{ data: string; mimeType: string; url: string }>>([])
```

改为:
```typescript
const refImages = ref<Array<{ data: string; mimeType: string; url: string; mask?: string }>>([])
```

- [ ] **Step 2: 新增参考图编辑弹窗状态**

文件路径: `web/src/views/GenerateView.vue:1032-1034` 附近（在 `// ---------- Edit modal state ----------` 后）

将:
```typescript
// ---------- Edit modal state ----------
const editModalVisible = ref(false)
const editingItem = ref(null)
```

改为:
```typescript
// ---------- Edit modal state ----------
const editModalVisible = ref(false)
const editingItem = ref(null)

// ---------- Reference image edit modal state ----------
const refEditModalVisible = ref(false)
const refEditIndex = ref(-1)  // 当前编辑的 refImages 索引
const refEditImageUrl = ref('')
```

- [ ] **Step 3: 缩略图添加编辑按钮**

文件路径: `web/src/views/GenerateView.vue:56-72`

将:
```vue
<div v-if="refImages.length > 0" class="mt-4">
  <div class="grid grid-cols-3 gap-3">
    <div v-for="(img, idx) in refImages" :key="idx"
      class="group relative overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 aspect-[4/3] transition-all hover:border-neutral-300 hover:shadow-sm">
      <img :src="img.url" class="w-full h-full object-contain bg-white p-1.5" />
      <div
        class="absolute left-2 top-2 inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-black/60 px-1.5 text-[10px] font-bold text-white">
        {{ idx + 1 }}
      </div>
      <button
        class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md opacity-0 transition-all group-hover:opacity-100 hover:bg-white"
        @click="removeRefImage(idx)">
        <XMarkIcon class="w-3.5 h-3.5" />
      </button>
    </div>
  </div>
</div>
```

改为:
```vue
<div v-if="refImages.length > 0" class="mt-4">
  <div class="grid grid-cols-3 gap-3">
    <div v-for="(img, idx) in refImages" :key="idx"
      class="group relative overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 aspect-[4/3] transition-all hover:border-neutral-300 hover:shadow-sm">
      <img :src="img.url" class="w-full h-full object-contain bg-white p-1.5" />
      <div
        class="absolute left-2 top-2 inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-black/60 px-1.5 text-[10px] font-bold text-white">
        {{ idx + 1 }}
      </div>
      <!-- 编辑按钮 -->
      <button
        class="absolute right-10 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md opacity-0 transition-all group-hover:opacity-100 hover:bg-white"
        title="编辑蒙版"
        @click="openRefImageEdit(idx)">
        <PencilIcon class="w-3.5 h-3.5" />
      </button>
      <!-- 删除按钮 -->
      <button
        class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md opacity-0 transition-all group-hover:opacity-100 hover:bg-white"
        @click="removeRefImage(idx)">
        <XMarkIcon class="w-3.5 h-3.5" />
      </button>
      <!-- 蒙版标记 -->
      <div v-if="img.mask"
        class="absolute left-2 bottom-2 px-1.5 py-0.5 rounded bg-emerald-500/90 text-[9px] font-bold text-white">
        已编辑
      </div>
    </div>
  </div>
</div>
```

- [ ] **Step 4: 导入 PencilIcon**

文件路径: `web/src/views/GenerateView.vue:617-620`

将:
```typescript
import {
  SparklesIcon, PhotoIcon, XMarkIcon,
  ExclamationCircleIcon, ArrowDownTrayIcon,
} from '@heroicons/vue/24/outline'
```

改为:
```typescript
import {
  SparklesIcon, PhotoIcon, XMarkIcon,
  ExclamationCircleIcon, ArrowDownTrayIcon,
  PencilIcon,
} from '@heroicons/vue/24/outline'
```

- [ ] **Step 5: 新增参考图编辑相关方法**

文件路径: `web/src/views/GenerateView.vue:1176` 附近（在 `removeRefImage` 函数后）

添加:
```typescript
// 打开参考图编辑弹窗
function openRefImageEdit(idx: number) {
  refEditIndex.value = idx
  refEditImageUrl.value = refImages.value[idx].url
  refEditModalVisible.value = true
}

// 处理参考图编辑提交
function handleRefEditSubmit({ imageUrl, maskData }: { imageUrl: string; maskData: string }) {
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

- [ ] **Step 6: 添加参考图编辑弹窗组件**

文件路径: `web/src/views/GenerateView.vue:317-319` 附近

在 `ImageEditModal` 后添加:
```vue
<!-- Reference Image Edit Modal -->
<ImageEditModal
  :visible="refEditModalVisible"
  :imageUrl="refEditImageUrl"
  :originalPrompt="''"
  mode="ref"
  @close="refEditModalVisible = false"
  @submit="handleRefEditSubmit"
/>
```

- [ ] **Step 7: 提交代码**

```bash
git add web/src/views/GenerateView.vue
git commit -m "feat(GenerateView): 添加参考图编辑按钮和蒙版保存逻辑"
```

---

## Task 3: 调整生成提交逻辑（处理带蒙版的参考图）

**Files:**
- Modify: `web/src/views/GenerateView.vue:1353-1355` (generate 函数中的 refPayload)

- [ ] **Step 1: 更新 generate 函数中的 refPayload 逻辑**

文件路径: `web/src/views/GenerateView.vue:1353-1355`

将:
```typescript
const refPayload = refImages.value.length > 0
  ? refImages.value.map(img => ({ data: img.data, mime_type: img.mimeType }))
  : null
```

改为:
```typescript
const refPayload = refImages.value.length > 0
  ? refImages.value.map(img => {
      const payload: any = { data: img.data, mime_type: img.mimeType }
      // 如果有蒙版数据，合并原图和蒙版
      if (img.mask) {
        // 使用 canvas 合并原图和蒙版
        payload.data = mergeImageAndMask(img.data, img.mask)
        payload.mime_type = 'image/png'
      }
      return payload
    })
  : null
```

- [ ] **Step 2: 添加 mergeImageAndMask 函数**

文件路径: `web/src/views/GenerateView.vue:1345` 附近

在 `async function generate()` 之前添加:
```typescript
// 合并原图和蒙版，返回合并后的 base64（无前缀）
async function mergeImageAndMask(imageBase64: string, maskBase64: string): Promise<string> {
  return new Promise(async (resolve, reject) => {
    try {
      const canvas = document.createElement('canvas')
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        resolve(imageBase64)
        return
      }

      // 加载原图
      const img = await loadImageFromBase64(imageBase64)
      canvas.width = img.width
      canvas.height = img.height

      // 绘制原图
      ctx.drawImage(img, 0, 0)

      // 加载蒙版
      const maskImg = await loadImageFromBase64(maskBase64)

      // 混合蒙版（使用 destination-over 将蒙版叠加在原图上）
      ctx.globalCompositeOperation = 'destination-over'
      ctx.drawImage(maskImg, 0, 0)

      resolve(canvas.toDataURL('image/png').split(',')[1])
    } catch (err) {
      console.error('合并蒙版失败:', err)
      resolve(imageBase64)  // 失败时返回原图
    }
  })
}

// 从 base64 加载图片
function loadImageFromBase64(base64: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = reject
    // 添加 data:image/png;base64, 前缀（如果没有）
    img.src = base64.includes(',') ? base64 : `data:image/png;base64,${base64}`
  })
}
```

- [ ] **Step 3: 提交代码**

```bash
git add web/src/views/GenerateView.vue
git commit -m "feat(GenerateView): 支持带蒙版的参考图合并提交"
```

---

## Task 4: 清理删除参考图时的蒙版数据

**Files:**
- Modify: `web/src/views/GenerateView.vue:1176` (removeRefImage 函数)

- [ ] **Step 1: 确保 removeRefImage 清理蒙版**

`removeRefImage` 函数使用 `splice` 删除数组元素，会自动清理 mask 字段，无需额外处理。确认现有代码:
```typescript
function removeRefImage(idx) { refImages.value.splice(idx, 1) }
```

此步骤已自动完成，无需修改。

---

## 自检清单

### Spec 覆盖检查
| 设计需求 | 对应任务 |
|---------|----------|
| refImages 增加 mask 字段 | Task 2 Step 1 |
| ImageEditModal 新增 mode prop | Task 1 Step 1 |
| mode='ref' 返回 { imageUrl, maskData } | Task 1 Step 4 |
| 缩略图悬停显示编辑按钮 | Task 2 Step 3 |
| 编辑完成保存蒙版到 refImages | Task 2 Step 5 |
| generate() 处理带 mask 的参考图 | Task 3 Step 1 |
| 原图+蒙版合并发送 | Task 3 Step 2 |

### 占位符检查
- [x] 无 TBD/TODO
- [x] 所有代码块完整
- [x] 文件路径精确到行号

### 类型一致性检查
- [x] `refImages.value[idx].mask` - mask 是 string 类型
- [x] `handleRefEditSubmit` 参数 `{ imageUrl, maskData }` 与 ImageEditModal emit 一致
- [x] `mergeImageAndMask` 返回 `Promise<string>` 与 payload.data 赋值类型一致
