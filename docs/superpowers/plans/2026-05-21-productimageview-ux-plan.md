# ProductImageView UX 优化实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 AI 图片生成工作流的用户体验问题：加载无反馈、交互体验差、布局不美观

**Architecture:** 调整左右布局比例至 60/40，重构右侧为"方案/图片展示 + 反馈输入"模式，添加按钮 loading 状态，对齐 router.py 新接口

**Tech Stack:** Vue 3 Composition API + 现有模板/CSS + 现有后端 router.py

---

## 文件结构

- `web/src/views/ProductImageView.vue` — 唯一需要修改的文件（3590 行），包含模板、脚本、样式

---

## Task 1: 调整布局比例 (60/40)

**Files:**
- Modify: `web/src/views/ProductImageView.vue:1114-1129`

- [ ] **Step 1: 修改 grid 布局比例**

将现有的 `grid-template-columns: minmax(320px, 380px) minmax(0, 1fr)` 改为 `minmax(0, 60%) minmax(0, 40%)`：

```css
.pg {
  /* ... 其他保持不变 ... */
  display: grid;
  grid-template-columns: minmax(0, 60%) minmax(0, 40%);
  gap: 18px;
  /* ... 其他保持不变 ... */
}
```

- [ ] **Step 2: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "refactor(ProductImageView): 调整左右布局比例为 60/40"
```

---

## Task 2: 缩小产品图片尺寸

**Files:**
- Modify: `web/src/views/ProductImageView.vue:10-15` (template)
- Modify: `web/src/views/ProductImageView.vue` (style section)

- [ ] **Step 1: 模板中产品图片 button 添加固定尺寸**

```html
<!-- 找到产品选择区的 picked button，添加 style -->
<button v-if="selectedProduct" ... style="width: 80px; height: 80px;">
  <img ... style="width: 100%; height: 100%; object-fit: cover;">
</button>
```

- [ ] **Step 2: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "refactor(ProductImageView): 产品图尺寸缩小至 80x80px"
```

---

## Task 3: 对齐 API 接口路径

**Files:**
- Modify: `web/src/views/ProductImageView.vue:885-895` (submitProductConfirm)
- Modify: `web/src/views/ProductImageView.vue:968` (confirmPlan)
- Modify: `web/src/views/ProductImageView.vue:988` (submitFeedback)
- Modify: `web/src/views/ProductImageView.vue:1000` (confirmImage)
- Modify: `web/src/views/ProductImageView.vue:1037` (submitImageFeedback)
- Modify: `web/src/views/ProductImageView.vue:1051` (retryGeneration)
- Modify: `web/src/views/ProductImageView.vue:913` (fetchAgentStatus)

- [ ] **Step 1: submitProductConfirm 中的 start 接口路径**

找到 `request.post('/ai/image-agent/start',` 改为：

```typescript
const res = await request.post('/api/v1/ai/image-agent/start', {
  task_id: latestTask.value?.task_id || `temp-${Date.now()}`,
  product_id: Number(selectedProductId.value),
  product_images: productImages,
  title: confirmTitle.value,
  bullets: confirmBullets.value,
  image_type_configs: buildImageTypeConfigs(),
  mode: getMode(),
  output_ratio: selectedOutputRatio.value,
  a_plus_format: selectedAPlusFormat.value
})
```

- [ ] **Step 2: confirmPlan 中的 confirm-plan 接口**

找到 `request.post(\`/ai/image-agent/${currentSessionId.value}/confirm-plan\`)` 改为：

```typescript
await request.post(`/api/v1/ai/image-agent/confirm/${currentSessionId.value}`)
```

- [ ] **Step 3: submitFeedback 中的 feedback 接口**

找到 `request.post(\`/ai/image-agent/${currentSessionId.value}/feedback\`, {` 改为：

```typescript
await request.post(`/api/v1/ai/image-agent/feedback/${currentSessionId.value}?feedback=${encodeURIComponent(feedback)}&action=revise_plan`)
```

- [ ] **Step 4: submitImageFeedback 中的 feedback 接口**

找到 `request.post(\`/ai/image-agent/${currentSessionId.value}/feedback\`, {` 改为：

```typescript
await request.post(`/api/v1/ai/image-agent/feedback/${currentSessionId.value}?feedback=${encodeURIComponent(feedback)}&action=regenerate`)
```

- [ ] **Step 5: confirmImage 中的 confirm-image 接口**

找到 `request.post(\`/ai/image-agent/${currentSessionId.value}/confirm-image\`)` 改为：

```typescript
const res = await request.post(`/api/v1/ai/image-agent/confirm/${currentSessionId.value}`)
```

- [ ] **Step 6: retryGeneration 中的 confirm-plan 接口**

找到 `request.post(\`/ai/image-agent/${currentSessionId.value}/confirm-plan\`)` 改为：

```typescript
await request.post(`/api/v1/ai/image-agent/confirm/${currentSessionId.value}`)
```

- [ ] **Step 7: fetchAgentStatus 中的 status 接口**

找到 `request.get(\`/ai/image-agent/${currentSessionId.value}/status\`)` 改为：

```typescript
const res = await request.get(`/api/v1/ai/image-agent/status/${currentSessionId.value}`)
```

- [ ] **Step 8: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "refactor(ProductImageView): 对齐 router.py 新 API 接口路径"
```

---

## Task 4: 添加按钮 loading 状态

**Files:**
- Modify: `web/src/views/ProductImageView.vue` (script + template)

- [ ] **Step 1: 添加 loading 相关的 refs**

在 `script` 部分添加：

```typescript
// Loading 状态
const isOperationLoading = ref(false)
const loadingText = ref('')

// 操作按钮 loading 封装
async function withLoading<T>(
  operation: () => Promise<T>,
  text: string
): Promise<T | null> {
  isOperationLoading.value = true
  loadingText.value = text
  try {
    const result = await operation()
    return result
  } catch (error) {
    toast.error(text.replace('...', '失败') + '，请稍后重试')
    return null
  } finally {
    isOperationLoading.value = false
    loadingText.value = ''
  }
}
```

- [ ] **Step 2: 修改 confirmPlan 函数使用 withLoading**

```typescript
async function confirmPlan() {
  const res = await withLoading(
    () => request.post(`/api/v1/ai/image-agent/confirm/${currentSessionId.value}`),
    '确认中...'
  )
  if (!res) return
  currentPhase.value = 'generating'
  pendingPlanType.value = ''
  pendingPlanPrompt.value = ''
  pendingPlanComposition.value = ''
  pendingPlanCompliance.value = ''
}
```

- [ ] **Step 3: 修改 submitFeedback 函数使用 withLoading**

```typescript
async function submitFeedback() {
  if (!reviseFeedback.value.trim()) return
  const feedback = reviseFeedback.value
  showReviseModal.value = false
  reviseFeedback.value = ''
  currentPhase.value = 'planning'
  pendingPlanPrompt.value = '正在重新生成方案...'
  const res = await withLoading(
    () => request.post(`/api/v1/ai/image-agent/feedback/${currentSessionId.value}?feedback=${encodeURIComponent(feedback)}&action=revise_plan`),
    '修改方案中...'
  )
  if (!res) {
    currentPhase.value = 'plan_ready'
  }
}
```

- [ ] **Step 4: 修改 submitImageFeedback 函数使用 withLoading**

```typescript
async function submitImageFeedback() {
  showImageFeedbackModal.value = false
  const feedback = imageFeedback.value.trim() || '请重新生成这张图片'
  imageFeedback.value = ''
  currentPhase.value = 'generating'
  pendingPlanPrompt.value = '正在重新生成图片...'
  const res = await withLoading(
    () => request.post(`/api/v1/ai/image-agent/feedback/${currentSessionId.value}?feedback=${encodeURIComponent(feedback)}&action=regenerate`),
    '重新生成中...'
  )
  if (!res) {
    currentPhase.value = 'image_ready'
  }
}
```

- [ ] **Step 5: 修改 confirmImage 函数使用 withLoading**

```typescript
async function confirmImage() {
  const res = await withLoading(
    () => request.post(`/api/v1/ai/image-agent/confirm/${currentSessionId.value}`),
    '确认中...'
  )
  if (!res) return
  const data = res.data || res
  if (data.is_complete || data.phase === 'completed') {
    currentPhase.value = 'completed'
    stopAgentPolling()
    toast.success('所有图片生成完成')
    await fetchRecentTasks()
  } else {
    if (data.plan) {
      currentPlan.value = data.plan
      pendingPlanType.value = data.plan.image_type || data.plan.type || ''
      pendingPlanComposition.value = data.plan.composition || ''
      pendingPlanPrompt.value = data.plan.prompt || ''
      pendingPlanCompliance.value = data.plan.compliance_notes || ''
    }
    currentImageUrl.value = ''
    currentPhase.value = 'plan_ready'
  }
}
```

- [ ] **Step 6: 修改 retryGeneration 函数使用 withLoading**

```typescript
async function retryGeneration() {
  errorMessage.value = ''
  const res = await withLoading(
    () => request.post(`/api/v1/ai/image-agent/confirm/${currentSessionId.value}`),
    '重试中...'
  )
  if (!res) return
  currentPhase.value = 'generating'
}
```

- [ ] **Step 7: 在模板中添加 disabled 状态到操作按钮**

找到 plan_ready 阶段的按钮，添加 `:disabled`：

```html
<!-- 确认方案按钮 -->
<button
  class="confirm-plan-btn"
  type="button"
  :disabled="isOperationLoading"
  @click="confirmPlan">
  {{ isOperationLoading ? '确认中...' : '✓ 确认方案，开始生成' }}
</button>

<!-- 修改方案按钮 -->
<button
  class="revise-plan-btn"
  type="button"
  :disabled="isOperationLoading"
  @click="showReviseModal = true">
  ✎ 修改方案
</button>
```

找到 image_ready 阶段的按钮，添加 `:disabled`：

```html
<button
  class="confirm-image-btn"
  type="button"
  :disabled="isOperationLoading"
  @click="confirmImage">
  {{ isOperationLoading ? '确认中...' : '✓ 确认图片' }}
</button>
<button
  class="reject-image-btn"
  type="button"
  :disabled="isOperationLoading"
  @click="regenerateImage">
  ↻ 重新生成
</button>
```

找到 completed 阶段的 view result 按钮，添加 `:disabled`：

```html
<button
  class="view-result-btn"
  type="button"
  @click="viewFinalResults">
  查看最终结果
</button>
```

找到 error 阶段的按钮，添加 `:disabled`：

```html
<button
  class="retry-btn"
  type="button"
  :disabled="isOperationLoading"
  @click="retryGeneration">
  {{ isOperationLoading ? '重试中...' : '↻ 重试' }}
</button>
<button class="exit-btn" type="button" @click="exitAIMode">退出</button>
```

- [ ] **Step 8: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "feat(ProductImageView): 添加操作按钮 loading 状态与禁用反馈"
```

---

## Task 5: submitProductConfirm 添加 loading 状态

**Files:**
- Modify: `web/src/views/ProductImageView.vue:864-907`

- [ ] **Step 1: 修改 submitProductConfirm 函数**

```typescript
async function submitProductConfirm() {
  showProductConfirmModal.value = false

  isInAIMode.value = true
  isGenerating.value = true
  currentPhase.value = 'waiting'
  currentPlan.value = null
  currentImageUrl.value = ''
  completedImages.value = []
  currentAnalysisDetail.value = ''

  // 构建产品图片信息（用于AI生图参考）
  const productImages = {}
  if (selectedProduct.value?.whiteBgImage) {
    productImages.front = selectedProduct.value.whiteBgImage
  } else if (selectedProduct.value?.image) {
    productImages.front = selectedProduct.value.image
  }

  try {
    const res = await request.post('/api/v1/ai/image-agent/start', {
      task_id: latestTask.value?.task_id || `temp-${Date.now()}`,
      product_id: Number(selectedProductId.value),
      product_images: productImages,
      title: confirmTitle.value,
      bullets: confirmBullets.value,
      image_type_configs: buildImageTypeConfigs(),
      mode: getMode(),
      output_ratio: selectedOutputRatio.value,
      a_plus_format: selectedAPlusFormat.value
    })
    currentSessionId.value = res.data?.session_id || res.session_id
    startAgentPolling()
  } catch (error) {
    console.error('[submitProductConfirm] Error:', error)
    toast.error('启动 AI 引导失败')
    isInAIMode.value = false
  } finally {
    isGenerating.value = false
  }
}
```

- [ ] **Step 2: 修改"一键生成套图"按钮，添加 loading 状态**

找到 `gen-primary` 按钮，添加 `:disabled`：

```html
<button
  v-if="!isInAIMode"
  class="gen gen-primary"
  type="button"
  :disabled="isGenerating"
  @click="startAIGuided">
  {{ isGenerating ? '启动中...' : '一键生成套图' }}
</button>
```

- [ ] **Step 3: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "feat(ProductImageView): 启动按钮添加 loading 状态"
```

---

## Task 6: 修改 workflow-steps 显示逻辑（可选）

**Files:**
- Modify: `web/src/views/ProductImageView.vue:158-167`

- [ ] **Step 1: 当前步骤指示器保持，但不影响核心功能**

暂不修改，保持现状。

- [ ] **Step 2: 提交（跳过）**

无变更。

---

## Task 7: 页面刷新时检测 session 中断

**Files:**
- Modify: `web/src/views/ProductImageView.vue` (onMounted + 初始化逻辑)

- [ ] **Step 1: 在 onMounted 中添加 session 检查**

在 `onMounted` 后面的初始化逻辑中，暂不实现 session 持久化（根据设计规范，选择方案 A：刷新提示重新开始）。可以在 `onMounted` 中添加注释说明：

```typescript
onMounted(() => {
  document.addEventListener('click', closeDropdown)
  initLibraries()
  // 注意：刷新页面后 session 会中断，用户需要重新开始。
  // 如需持久化，可将 session_id 保存到 localStorage。
})
```

- [ ] **Step 2: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "chore(ProductImageView): 添加 session 中断提示注释"
```

---

## 验证清单

完成所有任务后验证：

- [ ] 左侧配置面板宽度约 60%，右侧方案展示约 40%
- [ ] 产品图片尺寸约 80x80px，与"选择产品库"按钮对齐
- [ ] 点击操作按钮后，按钮显示 loading 文案并禁用
- [ ] 所有 API 接口调用 `/api/v1/ai/image-agent/...` 前缀
- [ ] `confirm-plan` → `/confirm/{session_id}`
- [ ] `feedback` → `/feedback/{session_id}?feedback=xxx&action=xxx`
- [ ] `confirm-image` → `/confirm/{session_id}`
- [ ] `status` → `/status/{session_id}`
- [ ] 刷新页面后 session 清空（符合方案 A）
- [ ] 无 console.error 或 API 404 错误