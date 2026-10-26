# ProductImageView UX 优化设计

**Goal:** 解决 AI 图片生成工作流的用户体验问题——加载无反馈、交互体验差、布局不美观

**Architecture:** 保持现有左右布局，调整比例至 60/40，右侧改为"方案/图片展示 + 反馈输入"模式，不需要完整对话气泡界面

**Tech Stack:** Vue 3 Composition API + Arco Design Vue + 现有后端 router.py

---

## 1. 布局与结构

- 左右布局：左侧配置面板 60%，右侧方案展示 40%
- 产品图片尺寸缩小至 `80x80px`，与"选择产品库"按钮对齐，紧凑排列
- 右侧方案展示区高度占满，可滚动

```
┌─────────────────────────────────────────────────────┐
│  Header / 面包屑 + 标题                              │
├──────────────────────────┬──────────────────────────┤
│  左侧配置面板 (60%)       │  右侧方案展示 (40%)       │
│  ┌──────────────────┐     │  ┌────────────────────┐  │
│  │ 产品选择区       │     │  │ 当前方案/图片展示   │  │
│  │ 产品图: 80x80   │     │  │                    │  │
│  │ 紧凑排列        │     │  │                    │  │
│  └──────────────────┘     │  └────────────────────┘  │
│  ┌──────────────────┐     │  ┌────────────────────┐  │
│  │ 图片类型配置     │     │  │ 状态指示           │  │
│  └──────────────────┘     │  └────────────────────┘  │
│  ┌──────────────────┐     │  ┌────────────────────┐  │
│  │ 开始按钮         │     │  │ 反馈输入框        │  │
│  └──────────────────┘     │  │ 操作按钮          │  │
│                           │  └────────────────────┘  │
└──────────────────────────┴──────────────────────────┘
```

---

## 2. 方案/图片展示 + 反馈输入模式

**不需要对话气泡**。用户不需要看到自己发送了什么，只要看到：
- 当前方案（构图、Prompt、合规说明）
- 当前生成的图片预览
- AI 根据反馈修改后的新方案/新图片

### 展示内容

- **方案信息**：构图描述、英文 Prompt、合规说明
- **图片预览**：生成的图片，附带"点击查看大图"功能
- **状态文本**：当前工作流阶段（"等待反馈"、"生成方案中"、"生成图片中..."）
- **进度信息**：已完成 N/M 张图片

### 反馈输入

- 单行/多行文本输入框，placeholder = "输入您的反馈..."
- 三个操作按钮：
  - **确认方案** → 触发 AI confirm_image 逻辑
  - **重新生成** → 触发重新生成图片
  - **修改方案** → 触发 AI 根据反馈修改方案
- 用户输入反馈后，AI 处理，结果更新在方案/图片展示区，用户只看到结果

---

## 3. 按钮状态与加载反馈

### 按钮状态规则

任意操作按钮点击后：
- 所有操作按钮置 `disabled`
- 点击的按钮显示转圈图标 + 对应文案
- 接口完成后恢复按钮

### 不同操作的加载文案

| 操作 | 加载文案 |
|------|---------|
| 开始生成 | "启动中..." |
| 确认方案 | "确认中..." |
| 生成方案中 | "分析中..." |
| 生成图片中 | "生成图片中..." |
| 重新生成 | "重新生成中..." |
| 修改方案 | "修改方案中..." |

### 错误处理

- 接口报错 → 按钮恢复 + Arco Design `Message` toast 提示错误信息
- 不使用弹窗打断用户，toast 提示即可

---

## 4. API 接口对接

**前端调用的 router.py 新接口：**

| 用途 | 接口 | 方法 |
|------|------|------|
| 启动 Agent | `/api/v1/ai/image-agent/start` | POST |
| 发送消息/反馈 | `/api/v1/ai/image-agent/chat?session_id=xxx` | POST |
| 确认当前图片 | `/api/v1/ai/image-agent/confirm/{session_id}` | POST |
| 提交反馈 | `/api/v1/ai/image-agent/feedback/{session_id}?feedback=xxx&action=revise` | POST |
| 获取状态 | `/api/v1/ai/image-agent/status/{session_id}` | GET |
| 完成任务 | `/api/v1/ai/image-agent/complete/{session_id}` | POST |

### ChatRequest body

```typescript
interface ChatRequest {
  message: string;       // 用户输入的反馈内容
  context?: string;       // 可选上下文
}
```

### Feedback 接口参数

```
POST /api/v1/ai/image-agent/feedback/{session_id}?feedback={feedback}&action={action}
- action: "revise_plan" | "regenerate" | "keep"
```

### Session 管理

- `/start` 返回 `session_id`，前端存储在组件 `ref` 中
- 所有后续请求通过 `session_id` 参数关联会话
- 页面刷新 → 提示"对话已中断，请重新开始"，不持久化 session
- `/complete/{session_id}` 调用后清理本地 session

### 请求封装示例

```typescript
// 统一的加载 + 错误处理封装
async function withLoading<T>(
  fn: () => Promise<T>,
  buttons: Ref<ButtonInstance[]>,
  loadingText: string
): Promise<T | null> {
  // 1. 所有按钮 disabled + 显示 loading 文案
  // 2. 执行 fn()
  // 3. 成功 → 恢复按钮
  // 4. 失败 → 恢复按钮 + toast 错误
}
```

---

## 5. 工作流状态与展示对应

| current_phase | 右侧展示内容 |
|---------------|-------------|
| init | 等待开始 |
| analyzing | "分析产品中..." + 加载动画 |
| planning | "生成方案中..." + 加载动画 |
| plan_ready | 方案卡片（构图、Prompt、合规说明）+ 操作按钮 |
| generating | "生成图片中..." + 加载动画 |
| image_ready | 图片预览 + 方案信息 + 确认/重新生成按钮 |
| completed | 最终汇总 + 所有完成图片列表 |

---

## 6. 实现范围

### 需要修改

- `web/src/views/ProductImageView.vue` — 重构右侧方案展示区、添加加载状态、对齐新 API

### 不改动

- 左侧配置面板的表单逻辑和校验
- 后端 router.py 及 LangGraph 工作流
- 其他页面
