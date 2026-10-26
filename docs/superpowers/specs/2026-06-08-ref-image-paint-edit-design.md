# 参考图片画笔编辑功能设计

**日期**: 2026-06-08

## 目标

在 GenerateView 的参考图片上传后，支持用户使用画笔标记修改区域，编辑完成后将画笔轨迹作为蒙版数据保存，与原图配对提交生成。

## 背景

- 现有 GenerateView 支持上传最多 6 张参考图片
- 现有 ImageEditModal 已实现完整的画笔编辑功能（用于生成后编辑图片）
- 用户希望在上传参考图时就能标记局部修改区域

## 方案概述

复用现有 ImageEditModal 组件，通过 mode prop 区分场景，实现最小改动。

## 数据结构

### refImages 扩展

```typescript
// 现有结构
interface RefImage {
  data: string;      // base64 数据
  mimeType: string;  // MIME 类型
  url: string;       // 数据 URL
}

// 新增字段
interface RefImage {
  data: string;
  mimeType: string;
  url: string;
  mask?: string;    // 画笔蒙版 base64 数据（可选）
}
```

### ImageEditModal props 扩展

```typescript
interface Props {
  visible: boolean;
  imageUrl: string;
  originalPrompt?: string;
  mode?: 'ref' | 'result';  // 新增：'ref' = 参考图编辑，'result' = 生成后编辑
}
```

## 组件改动

### ImageEditModal.vue

**1. props 新增**
- `mode`: 默认 `'result'`（保持向后兼容）

**2. emit 事件扩展**
```typescript
// 现有
emit('submit', { prompt, imageData, mimeType, quality })

// 'ref' 模式下
emit('submit', { prompt, imageData, maskData, mimeType })
// - imageData: 合并后的图片（原图+蒙版）
// - maskData: 蒙版数据（用于后续可能的单独处理）
```

**3. 逻辑调整**
- `'ref'` 模式：隐藏"输出质量"选择（参考图模式不需要）
- 提交时额外返回蒙版数据 `maskData`

### GenerateView.vue

**1. 缩略图 UI 调整**
- 悬停时显示"编辑"按钮
- 已有 mask 的图片显示蒙版标记（可选）

**2. 编辑弹窗触发**
- 点击编辑按钮时，传入 refImages[idx].url 和 mode='ref'

**3. 编辑完成处理**
- 接收 maskData，保存到 refImages[idx].mask
- 原图 data 保持不变

**4. 提交生成逻辑调整**
- 遍历 refImages，有 mask 的图片需要：
  1. 将原图和蒙版合并为一张图片
  2. 或者分别传递原图和蒙版（取决于后端 API）
- 建议：后端支持同时传递原图+蒙版，前端按现有格式扩展

## 后端 API 预期

### 现有 API（不变）
```json
{
  "prompt": "...",
  "ref_images": [
    { "data": "base64", "mime_type": "image/png" }
  ]
}
```

### 扩展后（推荐）
```json
{
  "prompt": "...",
  "ref_images": [
    { "data": "base64", "mime_type": "image/png" },
    // 或者支持蒙版
    { "data": "base64", "mime_type": "image/png", "mask": "base64" }
  ]
}
```

**备选方案**：前端直接合并原图+蒙版为一张图片发送（后端无需改动）

## UI 交互流程

```
1. 用户上传参考图片 → 缩略图网格显示
2. 鼠标悬停缩略图 → 显示"编辑"按钮
3. 点击"编辑"按钮 → 弹出 ImageEditModal（mode='ref'）
4. 用户使用画笔标记区域 → 输入描述词
5. 点击"重新生成" → 
   - 蒙版数据保存到 refImages[idx].mask
   - 原图 data 保持不变
6. 提交生成时 → 有 mask 的图片自动合并发送
```

## 改动预估

| 文件 | 改动量 | 说明 |
|------|--------|------|
| ImageEditModal.vue | ~30行 | 新增 mode prop 和 maskData 返回 |
| GenerateView.vue | ~50行 | 编辑按钮、mask 处理、提交逻辑 |
| 后端 API | 可选 | 如需支持蒙版字段则扩展 |

## 测试要点

1. 上传参考图后，点击编辑按钮能正常弹出画笔界面
2. 画笔标记后提交，蒙版数据正确保存
3. 同一张图片可多次编辑，蒙版会覆盖
4. 清除画笔后，蒙版字段应清除（或设为空）
5. 提交生成时蒙版数据正确发送

## 风险与限制

- 后端 API 需要支持蒙版数据（如果选择分别发送）
- 参考图编辑模式下，隐藏输出质量选项
- 蒙版图片与原图尺寸一致，合并时需注意 Canvas 尺寸