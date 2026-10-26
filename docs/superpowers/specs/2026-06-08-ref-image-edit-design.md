# 参考图画笔编辑功能设计

**日期**: 2026-06-08
**状态**: 已批准

---

## 1. 目标

在上传参考图片后，支持通过画笔在图片上标记修改区域，编辑完成后将画笔轨迹作为蒙版数据保存，与原图配对提交给后端。

---

## 2. 数据结构

### 2.1 refImages 扩展

```typescript
// 原结构
refImages = Array<{ data: string; mimeType: string; url: string }>

// 新增 mask 字段后
refImages = Array<{
  data: string;       // base64 数据（用于提交）
  mimeType: string;   // MIME 类型
  url: string;        // data URL（用于预览）
  mask?: string;      // 蒙版 base64 数据（可选）
}>
```

### 2.2 生成提交 Payload

```typescript
interface RefImagePayload {
  data: string;           // 原图 base64
  mime_type: string;      // MIME 类型
  mask_data?: string;     // 蒙版 base64（如果有）
}
```

---

## 3. 组件改动

### 3.1 ImageEditModal 改动

**新增 Props**:
- `mode: 'ref' | 'result'`（默认 `'result'`）
  - `'ref'`: 参考图编辑模式，提交时返回 `{ imageUrl, maskData, originalPrompt }`
  - `'result'`: 生成结果编辑模式，保持现有逻辑

**提交返回格式**:
- `'ref'` 模式: `{ imageUrl: string, maskData: string, originalPrompt: string }`
- `'result'` 模式: 现有 `{ prompt, imageData, mimeType, quality }`

### 3.2 GenerateView 改动

**缩略图 UI**:
- 悬停时显示"编辑"按钮（与"删除"按钮并排）
- 点击后打开 ImageEditModal（mode='ref'）

**编辑完成回调**:
- 更新 refImages 对应项的 mask 字段
- 显示 toast 提示"蒙版已保存"

**生成提交逻辑**:
- 遍历 refImages，组装 ref_payload
- 有 mask 的图片，payload 包含 `data` + `mask_data`
- 无 mask 的图片，payload 包含 `data`（现有逻辑）

---

## 4. 用户流程

1. 用户上传参考图片 → 显示缩略图
2. 用户悬停缩略图 → 显示"编辑"按钮
3. 用户点击"编辑" → 弹出 ImageEditModal
4. 用户使用画笔标记区域 → 填写提示词（可选）
5. 用户点击"重新生成" → 保存蒙版数据到 refImages
6. 用户填写提示词后点击"开始生成" → 原图+蒙版一起提交

---

## 5. 改动范围

| 文件 | 改动内容 |
|------|----------|
| `ImageEditModal.vue` | 新增 mode prop，适配 ref 模式提交返回格式 |
| `GenerateView.vue` | 缩略图添加编辑按钮，处理编辑回调，组装 payload |

---

## 6. 兼容性

- 无 mask 的参考图保持现有逻辑不变
- 现有生成结果编辑功能不受影响（mode='result' 默认值）