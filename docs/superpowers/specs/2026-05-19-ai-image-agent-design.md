# AI 图片编排 Agent 设计

## 1. 背景与目标

用 Python LangGraph 微服务接管 ProductImageView.vue 的 AI 图片编排逻辑，不影响现有的 Go 后端和其他功能。

目标：
- 创建一个"亚马逊视觉总监"AI Agent
- 按用户选择的模式（主副图 / A+ 图 / 全套）逐步引导生成
- 每张图都经过：方案规划 → 用户确认 → 生图 → 用户确认 → 下一张
- 用户不满意可以修改指令或重新生成
- 全程用户把控节奏，AI 只负责策划和建议

## 2. 架构设计

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────────┐
│   前端 Vue      │────▶│  Go 后端 (现有)  │────▶│  Python LangGraph   │
│ ProductImage    │     │  任务管理/存储   │     │  AI Agent 微服务     │
│   View          │◀────│                 │◀────│                     │
└─────────────────┘     └─────────────────┘     └─────────────────────┘
```

- **Go 后端**：保持现有逻辑不变，新增与 LangGraph 的通信接口
- **Python LangGraph 微服务**：独立部署，负责 AI 编排逻辑、状态管理、循环控制
- **前端**：适配新的多轮交互流程（方案确认、图片确认）

## 3. 生成配置

### 3.1 三种生成模式

| 模式 | image_type_configs | LangGraph 工作流 |
|------|-------------------|-----------------|
| 仅 A+ 图 | `{ a_plus: 4 }` | 跳过主副图，直接做 A+ 图 |
| 仅主副图 | `{ main_image: 1, feature: 2, ... }` | 做主副图，跳过 A+ |
| 全套（标准） | `{ main_image: 1, ..., a_plus: 4 }` | 主副图 → A+ 图 |

前端已有套图方案选择器，LangGraph 根据传入的 `image_type_configs` 动态调整工作流。

### 3.2 输出比例

前端 ProductImageView.vue 已有比例选择器，用户在开始前选择：

| 比例 | 用途 |
|------|------|
| 1:1 | 方形通用（主图、附图） |
| 3:4 | 竖版长图（手机 A+） |
| 4:3 | 横版展示 |
| 4:5 | 标准横版 |
| 5:4 | 标准竖版 |
| 21:9 | 超宽横幅（电脑 A+） |

这些比例会传入 LangGraph，生图时按比例生成。

### 3.3 A+ 图类型选择

生成 A+ 图时，用户可选择终端适配：

| 类型 | 比例 | 说明 |
|------|------|------|
| **手机 A+** | 4:3 | 竖版长图，适合移动端展示，文字排版更紧凑 |
| **电脑 A+** | 21:9 | 超宽横幅，适合桌面端展示，文字排版更宽松 |

选择结果作为 `a_plus_format` 参数传入 LangGraph。

## 4. LangGraph 状态设计

```python
class AgentState(TypedDict):
    # 任务上下文
    task_id: str
    agent_name: str  # "亚马逊视觉总监"
    mode: str  # "full" | "main_secondary" | "a_plus_only"

    # 生成配置
    output_ratio: str  # "1:1" | "3:4" | "4:3" | "4:5" | "5:4" | "21:9"
    a_plus_format: str  # "mobile" | "desktop"

    # 产品信息
    product_info: dict  # { images: {...}, title, bullets }
    image_type_configs: list[dict]  # [{ type, count }, ...]

    # 当前进度
    current_phase: str  # "product_analysis" | "planning" | "generating" | "review"
    current_config_index: int  # 当前处理到第几个 config
    current_image_index: int  # 当前 config 内的第几张图
    current_image_type: str

    # 方案与结果
    current_plan: dict  # 策划方案（构图、Prompt、文案等）
    generated_image_url: str | None

    # 用户反馈
    user_feedback: str | None
    revisions: int  # 已修改次数

    # 完成列表
    completed_images: list[dict]  # [{ type, index, image_url, plan }, ...]
```

## 5. LangGraph 节点流程

```
init
  │
  ▼
product_analysis ────────────────────────────────────▶ end
  │                                                  (无产品图)
  ▼
check_mode ─────────────┐
  │                     │
只有 A+?               有主副图?
  │                     │
  ▼                     ▼
start_a_plus      queue_all_images
  │                     │
  └──────┐             │
         ▼              │
  image_planning ◀──────┘
    │              (读取反馈/重试)
    ▼
wait_user_confirm
    │
    ├── 确认 ──▶ image_generation
    │                 │
    │    ◀────────────┘
    ▼                 │
wait_user_review      │
    │                 │
    ├── 确认 ──▶ move_to_next
    │                 │
    └── 不满意 ──▶ image_generation (调整参数重试)
                    │
                    ▼
                move_to_next
                    │
          ┌─────────┴─────────┐
          │                   │
      还有图片?            全部完成?
          │                   │
          ▼                   ▼
    image_planning        check_a_plus
                              │
                        还有 A+?
                          │     │
                          ▼     ▼
                     start_a_plus  all_done
                          │         │
                          └────┬────┘
                               ▼
                             end
```

### 节点职责

| 节点 | 职责 |
|------|------|
| `init` | 初始化状态，验证输入 |
| `product_analysis` | 分析产品图片和卖点，形成结构化上下文 |
| `check_mode` | 根据 mode 决定起始节点 |
| `image_planning` | 为当前图片生成策划方案（构图 + Prompt + 文案） |
| `wait_user_confirm` | 等待用户确认方案，路由到 confirm / revise |
| `image_generation` | 调用生图 API |
| `wait_user_review` | 等待用户确认生成的图片，路由到 confirm / revise |
| `move_to_next` | 更新索引，检查是否还有图片 |
| `check_a_plus` | 检查是否需要进入 A+ 图阶段 |
| `end` | 任务完成，同步结果到 Go 后端 |

## 6. Agent 系统提示词

### 角色名
**亚马逊视觉总监**

### 描述
专注于 Amazon US 图片合规审查、策划与商业级生图。

### 系统指令

```
## 角色定位

你是一位拥有10年经验的亚马逊（Amazon US）高级视觉总监。你精通消费者行为心理学，并对亚马逊官方的《商品图片指南》有极其严格的执行标准。你的任务是引导用户完成从 Listing 策划到最终商业大片的产出。

## 核心能力与工具

1. **生图能力**：使用目前最强的生图模型进行创作
2. **合规审查**：严格执行主图纯白底（RGB 255,255,255）、商品占比 85%+、无侵权标志、无虚假声明等规定
3. **分层策划**：遵循 1张主图 + 6张附图黄金转化逻辑，遵循 A+ 图片规范
4. **比例适配**：根据用户选择的输出比例（1:1, 3:4, 4:3, 4:5, 5:4, 21:9）和 A+ 格式（手机 4:3 / 电脑 21:9）生成符合规范的图片

## 互动工作流（必须按此流程执行）

### 第一阶段：初始化

- 礼貌地要求用户提供产品的实拍图（若有）
- 分析已提供的标题和5点描述，提炼核心卖点
- 如缺少必要信息，主动告知用户

### 第二阶段：逐张引导制作

不要一次性生成所有图片。按以下顺序，每张图确认一次：

**Step 1：视觉策划与合规声明**
- 针对当前图片位（如 Main、PT01 等），结合用户卖点提出构图方案
- 明确指出该方案如何符合《亚马逊图片规范》

**Step 2：提供生图提示词（Prompt）**
- 编写一段专业的英文生图提示词
- 主图必须包含：`on a seamless pure white background RGB 255, 255, 255, professional studio lighting, product takes up 85% of the frame, high resolution, photorealistic`

**Step 3：等待确认并生图**
- 询问用户："是否同意此方案并开始生图？"
- 得到确认后，立即调用生图工具生成图片

**Step 4：循环直至完成**
- 一张图完成后，自动提议下一张图的策划方向
- 直到完成所有图片

## 语气与风格

专业、利落、富有商业审美，像是一个随时为你把控风险的资深顾问。
```

## 7. API 接口设计

### 7.1 启动 Agent 任务

```
POST /api/v1/ai/image-agent/start

Request:
{
  "task_id": "string",  // Go 后端的任务ID
  "product_id": 123,
  "product_images": {
    "front": "url",      // 正面白底图
    "side": "url",       // 侧面白底图
    "back": "url",       // 背面白底图
    "detail": "url",     // 细节特写图
    "parts": "url",      // 拆开的配件
    "scene": "url"       // 使用场景图
  },
  "title": "string",
  "bullets": ["string", ...],  // 5点描述
  "image_type_configs": [
    { "type": "main_image", "count": 1 },
    { "type": "feature", "count": 2 },
    { "type": "a_plus", "count": 4 }
  ],
  "mode": "full",  // "full" | "main_secondary" | "a_plus_only"
  "output_ratio": "1:1",  // "1:1" | "3:4" | "4:3" | "4:5" | "5:4" | "21:9"
  "a_plus_format": "mobile"  // "mobile" (4:3) | "desktop" (21:9)
}

Response:
{ "session_id": "string" }
```

### 7.2 获取当前状态

```
GET /api/v1/ai/image-agent/{session_id}/status

Response:
{
  "phase": "planning" | "generating" | "review",
  "current_image_type": "main_image",
  "current_image_index": 0,
  "output_ratio": "1:1",
  "a_plus_format": "mobile",
  "plan": {
    "composition": "string",
    "prompt": "string",
    "compliance_notes": "string"
  },
  "generated_image_url": "string" | null,
  "completed_images": [
    { "type": "main_image", "index": 0, "image_url": "...", "plan": {...} }
  ],
  "total_count": 11,
  "completed_count": 0
}
```

### 7.3 用户确认方案

```
POST /api/v1/ai/image-agent/{session_id}/confirm-plan

Response:
{ "status": "ok", "message": "开始生成图片..." }
```

### 7.4 用户反馈（修改/重试）

```
POST /api/v1/ai/image-agent/{session_id}/feedback

Request:
{
  "feedback": "string",  // 用户修改意见
  "action": "revise_plan" | "regenerate"
}

Response:
{ "status": "ok", "new_plan": {...} }
```

### 7.5 确认生成结果

```
POST /api/v1/ai/image-agent/{session_id}/confirm-image

Response:
{ "status": "ok", "next_type": "feature", "next_index": 1 }
```

### 7.6 任务完成

```
POST /api/v1/ai/image-agent/{session_id}/complete

Response:
{ "status": "ok", "task_id": "...", "images": [...] }

后端同步：
- 更新任务状态为 completed
- 保存所有图片URL
```

## 8. 前端适配

### 8.1 流程变化

| 之前 | 之后 |
|------|------|
| 点击"一键生成套图"，等待所有图片完成 | 点击"开始 AI 引导"，进入逐步确认流程 |
| 轮询任务状态 | 轮询 Agent 状态，展示当前方案/图片 |
| 图片完成后统一展示 | 实时展示当前图片 + 已完成图片 |

### 8.2 UI 调整

- **开始按钮**：改为"开始 AI 引导"（而非一键生成）
- **方案预览区**：展示 AI 生成的当前图片策划方案（构图说明、Prompt预览、合规提示）
- **确认按钮**：方案区下方添加"同意此方案并生图"
- **图片确认区**：生成的图片展示后，添加"确认"或"不满意，请修改"按钮
- **进度指示**：显示当前进度（图片 3/11）

### 8.3 新增 A+ 格式选择

前端已有输出比例选择器（6种比例），新增 A+ 格式选择控件：

- 在套图方案选择区下方、A+ 图数量配置旁，添加"终端适配"选择：
  - **手机 A+（4:3）** — 竖版长图，移动端展示（默认）
  - **电脑 A+（21:9）** — 超宽横幅，桌面端展示

选择后传给 LangGraph，A+ 图阶段按对应比例生图。

## 9. 项目结构

```
haierman/
├── backend/                    # Go 后端（现有）
│   └── internal/
│       ├── handler/
│       │   └── ecom_suite.go   # 新增与 Agent 的通信接口
│       └── service/
│           └── image_agent.go # Agent 通信服务
│
├── ai-agent/                   # Python LangGraph 微服务（新增）
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py             # FastAPI 入口
│   │   ├── router.py           # API 路由
│   │   ├── state.py            # AgentState 定义
│   │   ├── graph/
│   │   │   ├── __init__.py
│   │   │   ├── nodes.py        # 各节点实现
│   │   │   ├── edges.py        # 路由条件
│   │   │   └── builder.py      # GraphBuilder
│   │   ├── agent/
│   │   │   └── supervisor.py   # Agent 系统提示词
│   │   └── services/
│   │       ├── image_gen.py    # 生图服务
│   │       └── callback.py    # 回调 Go 后端
│   ├── requirements.txt
│   └── Dockerfile
│
└── web/                        # Vue 前端（现有）
    └── src/views/
        └── ProductImageView.vue  # 适配新流程
```

## 10. 依赖与技术栈

- **Python**: 3.11+
- **LangGraph**: 最新稳定版
- **FastAPI**: HTTP 服务
- **生图 SDK**: 待定（SD API / FLUX API / Midjourney API）
- **部署**: Docker Compose（Go + Python Agent 联动）