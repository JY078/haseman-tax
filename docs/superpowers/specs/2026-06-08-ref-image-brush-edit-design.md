# 参考图片画笔编辑功能设计

## 1. 功能目标

在上传参考图片后，通过悬停显示的"编辑"按钮，弹出画笔编辑界面，编辑完成后将画笔轨迹作为蒙版数据保存，与原图配对提交生成。

## 2. 核心改动

### 2.1 数据结构扩展

```typescript
// refImages 现有结构
refImages = Array<{ data: string; mimeType: string; url: string }>

// 扩展后结构
refImages = Array<{
  data: string;
  mimeType: string;
  url: string;
  mask?: string;  // 新增：蒙版 base64 数据
  originalUrl?: string;  // 新增：原始图片 URL（编辑后用于显示）
}>
```

### 2.2 ImageEditModal 改动

**新增 Props:**
- `mode: 'ref' | 'result'`（默认 `'result'`）
  - `'ref'`：参考图编辑模式
  - `'result'`：生成结果编辑模式（现有逻辑）

**Event 调整:**
- `mode === 'result'` 时：保持现有 `submit` 事件，返回 `{ prompt, imageData, mimeType, quality }`
- `mode === 'ref'` 时：提交时返回 `{ imageUrl, maskData, originalPrompt }`

### 2.3 GenerateView 改动

**缩略图 UI:**
- 悬停时在右上角显示"编辑"按钮
- 按钮样式与删除按钮一致（圆形背景，图标）

**编辑触发:**
- 点击编辑按钮时，打开 ImageEditModal（mode='ref'）
- 传入 `imageUrl` 为原始图片 URL
- `originalPrompt` 置空（参考图无提示词）

**编辑完成回调:**
```typescript
function handleRefImageEdit({ imageUrl, maskData }) {
  // 1. 更新 refImages 中对应项的 mask 和 originalUrl
  // 2. 如果蒙版不为空，更新 url 为合并后的图片（可选预览）
}
```

**生成提交时:**
- 遍历 refImages，对于有 mask 的图片：
  - 将原图 base64 和蒙版 base64 合并发送
  - 或使用 canvas 合并后发送

### 2.4 蒙版合并逻辑

提交生成时，对每张参考图处理：

```typescript
function prepareRefImageWithMask(img) {
  if (!img.mask) {
    return { data: img.data, mime_type: img.mimeType };
  }
  // 蒙版存在时，合并原图和蒙版
  // 方案 A：将蒙版叠加在原图上，使用 canvas 合并
  // 方案 B：将蒙版作为单独的 mask 字段发送（需要后端支持）
  // 暂时使用方案 A，将蒙版绘制到原图上后返回合并结果
}
```

## 3. UI 设计

### 3.1 缩略图编辑按钮

位置：与删除按钮并列，右上角

```html
<div class="group relative overflow-hidden rounded-xl ...">
  <img :src="img.url" class="..." />
  
  <!-- 序号标签 -->
  <div class="absolute left-2 top-2 ...">{{ idx + 1 }}</div>
  
  <!-- 编辑按钮 - 新增 -->
  <button class="absolute right-10 top-2 ..." @click="editRefImage(idx)">
    <PencilIcon class="w-3.5 h-3.5" />
  </button>
  
  <!-- 删除按钮 -->
  <button class="absolute right-2 top-2 ..." @click="removeRefImage(idx)">
    <XMarkIcon class="w-3.5 h-3.5" />
  </button>
</div>
```

### 3.2 编辑弹窗

复用现有 ImageEditModal，保持画笔工具完整功能。

## 4. 实现步骤

### Step 1: 扩展 ImageEditModal
- [ ] 新增 `mode` prop
- [ ] 调整 `submit` 事件的返回值逻辑

### Step 2: 扩展 GenerateView 数据结构
- [ ] refImages 类型增加 mask 字段
- [ ] 新增编辑弹窗状态管理

### Step 3: 添加编辑按钮
- [ ] 缩略图悬停显示编辑按钮
- [ ] 点击触发编辑弹窗

### Step 4: 处理编辑回调
- [ ] 接收蒙版数据并保存到 refImages
- [ ] 处理蒙版合并逻辑

### Step 5: 调整提交逻辑
- [ ] 生成时正确处理带蒙版的参考图

## 5. 测试场景

1. 上传单张参考图 → 悬停显示编辑按钮 → 点击打开编辑弹窗
2. 编辑后蒙版保存 → 提交生成时蒙版正确发送
3. 删除有蒙版的参考图 → 蒙版数据一并清理
4. 多张参考图，部分有蒙版 → 正确处理混合情况