---
name: ref-image-editor-design
description: 参考图上传时支持画笔编辑功能的设计规范
metadata:
  type: spec
  created: 2026-06-08
---

# 参考图画笔编辑功能设计

## 1. 背景与目标

当前 `GenerateView.vue` 支持上传参考图片，但无法在上传后进行编辑。用户希望在上传参考图片后，能够使用画笔在图片上标记修改区域，编辑后的蒙版数据与原图配对提交给后端进行生成。

### 核心需求

- 参考图片上传后，可通过悬停显示的"编辑"按钮触发画笔编辑
- 编辑完成后，画笔轨迹作为蒙版数据保存
- 原图保留，蒙版与原图配对提交（复用现有 ImageEditModal 逻辑）

## 2. 数据结构

### refImages 扩展

```typescript
// 当前结构
refImages = Array<{ data: string; mimeType: string; url: string }>

// 扩展后结构
refImages = Array<{
  data: string      // base64 数据
  mimeType: string  // MIME 类型
  url: string       // data URL
  mask?: string     // 蒙版 base64 数据（可选）
}>
```

## 3. 组件改动

### 3.1 ImageEditModal.vue

**新增 Props：**

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `mode` | `'ref' \| 'result'` | `'result'` | 编辑模式，`'ref'` 用于参考图编辑 |

**Emits 改动：**

- `mode='result'`（默认）：保持现有逻辑，`submit` 返回 `{ prompt, imageData, mimeType, quality }`
- `mode='ref'`：返回 `{ imageUrl, maskData, originalPrompt }`

### 3.2 GenerateView.vue

**缩略图区域改动：**

1. 缩略图容器添加悬停效果
2. 悬停时显示"编辑"按钮（右上角）
3. 点击编辑按钮弹出 ImageEditModal（mode='ref'）
4. 编辑完成后，将蒙版数据保存到对应 refImages 项的 `mask` 字段

**生成提交逻辑改动：**

当 `refImages` 有蒙版时，发送格式调整：

```typescript
// 无蒙版
ref_images: [{ data: base64, mime_type: mimeType }]

// 有蒙版（合并原图+蒙版）
ref_images: [{
  data: base64,           // 原图 base64
  mime_type: mimeType,
  mask: maskBase64        // 蒙版 base64
}]
```

## 4. UI 设计

### 缩略图编辑按钮

```
┌─────────────────────────┐
│ [1]              [编辑] │  ← 悬停时显示
│                         │
│    ┌─────────────┐      │
│    │             │      │
│    │   缩略图    │      │
│    │             │      │
│    └─────────────┘      │
│                         │
└─────────────────────────┘
```

- 按钮位置：右上角，悬停时显示
- 按钮样式：白色圆形背景，阴影，hover 时 scale 放大

### 编辑弹窗（mode='ref'）

复用现有 ImageEditModal，mode='ref' 时：
- 标题改为"编辑参考图"
- 右侧提示词输入区域可留空（参考图不需要提示词）
- 提交按钮文字改为"确认"

## 5. 实现步骤

1. **ImageEditModal 改动**
   - 新增 `mode` prop
   - `mode='ref'` 时调整 submit 回调返回值
   - 调整标题和按钮文案

2. **GenerateView 改动**
   - 缩略图添加悬停编辑按钮
   - 点击按钮打开 ImageEditModal（mode='ref'）
   - 编辑完成后保存蒙版数据到 refImages
   - generate() 函数中处理有蒙版的 ref_images

3. **测试验证**
   - 上传参考图 → 悬停显示编辑按钮 → 点击打开弹窗 → 画笔编辑 → 确认保存
   - 提交生成时验证蒙版数据正确发送

## 6. 风险与约束

- ImageEditModal 目前用于生成后的图片编辑，mode 参数需要向后兼容
- 蒙版数据 base64 会增加请求体积，需注意不超过后端限制
- 参考图最多 6 张，每张都可独立编辑蒙版