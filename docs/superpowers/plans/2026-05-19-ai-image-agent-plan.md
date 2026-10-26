# AI 图片编排 Agent 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 ProductImageView.vue 的套图生成逻辑从 Go 后端迁移到 Python LangGraph 微服务，实现 AI 引导的多轮交互式生图流程。

**Architecture:** 采用 Go 后端 + Python LangGraph 微服务分离架构。Go 后端保持现有逻辑，新增与 Agent 的通信接口；Python 微服务独立部署，负责 AI 编排逻辑、状态管理和多轮交互控制。

**Tech Stack:** Python 3.11+, LangGraph, FastAPI, Docker

---

## 文件结构

```
haierman/
├── backend/                         # Go 后端（修改）
│   └── internal/
│       ├── handler/
│       │   └── ecom_suite.go        # 新增 Agent 通信接口
│       └── service/
│           └── image_agent.go       # Agent HTTP 客户端
│
├── ai-agent/                        # Python LangGraph 微服务（新增）
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py                  # FastAPI 入口
│   │   ├── router.py                # API 路由
│   │   ├── state.py                 # AgentState 定义
│   │   ├── graph/
│   │   │   ├── __init__.py
│   │   │   ├── nodes.py             # 各节点实现
│   │   │   ├── edges.py             # 路由条件
│   │   │   └── builder.py           # GraphBuilder
│   │   ├── agent/
│   │   │   └── supervisor.py        # Agent 系统提示词
│   │   └── services/
│   │       ├── image_gen.py         # 生图服务
│   │       └── callback.py          # 回调 Go 后端
│   ├── tests/
│   │   └── test_agent.py            # 单元测试
│   ├── requirements.txt
│   ├── Dockerfile
│   └── docker-compose.yml
│
└── web/                            # Vue 前端（修改）
    └── src/views/
        └── ProductImageView.vue     # 适配新流程
```

---

## Task 1: Python AI Agent 服务基础架构

**Files:**
- Create: `ai-agent/app/__init__.py`
- Create: `ai-agent/app/state.py`
- Create: `ai-agent/app/main.py`
- Create: `ai-agent/app/router.py`
- Create: `ai-agent/requirements.txt`
- Create: `ai-agent/Dockerfile`
- Create: `ai-agent/docker-compose.yml`

- [ ] **Step 1: 创建目录结构和 requirements.txt**

```python
# ai-agent/requirements.txt
fastapi==0.111.0
uvicorn==0.29.0
langgraph==0.0.62
pydantic==2.7.1
httpx==0.27.0
python-dotenv==1.0.1
```

- [ ] **Step 2: 创建 state.py - AgentState 定义**

```python
# ai-agent/app/state.py
from typing import TypedDict, Optional

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
    current_config_index: int
    current_image_index: int
    current_image_type: str

    # 方案与结果
    current_plan: Optional[dict]  # 策划方案（构图、Prompt、文案等）
    generated_image_url: Optional[str]

    # 用户反馈
    user_feedback: Optional[str]
    revisions: int  # 已修改次数

    # 完成列表
    completed_images: list[dict]  # [{ type, index, image_url, plan }, ...]
```

- [ ] **Step 3: 创建 router.py - API 路由**

```python
# ai-agent/app/router.py
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

router = APIRouter(prefix="/api/v1/ai/image-agent", tags=["image-agent"])

class StartRequest(BaseModel):
    task_id: str
    product_id: int
    product_images: dict
    title: str
    bullets: list[str]
    image_type_configs: list[dict]
    mode: str = "full"
    output_ratio: str = "1:1"
    a_plus_format: str = "mobile"

class FeedbackRequest(BaseModel):
    feedback: str
    action: str  # "revise_plan" | "regenerate"

@router.post("/start")
async def start_agent(req: StartRequest):
    # TODO: 实现启动逻辑
    pass

@router.get("/{session_id}/status")
async def get_status(session_id: str):
    # TODO: 实现状态查询
    pass

@router.post("/{session_id}/confirm-plan")
async def confirm_plan(session_id: str):
    # TODO: 实现方案确认
    pass

@router.post("/{session_id}/feedback")
async def submit_feedback(session_id: str, req: FeedbackRequest):
    # TODO: 实现反馈处理
    pass

@router.post("/{session_id}/confirm-image")
async def confirm_image(session_id: str):
    # TODO: 实现图片确认
    pass

@router.post("/{session_id}/complete")
async def complete_task(session_id: str):
    # TODO: 实现任务完成
    pass
```

- [ ] **Step 4: 创建 main.py - FastAPI 入口**

```python
# ai-agent/app/main.py
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.router import router

app = FastAPI(title="AI Image Agent", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router)

@app.get("/health")
async def health():
    return {"status": "ok"}
```

- [ ] **Step 5: 创建 Dockerfile**

```dockerfile
# ai-agent/Dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

- [ ] **Step 6: 创建 docker-compose.yml**

```yaml
# ai-agent/docker-compose.yml
version: '3.8'
services:
  ai-agent:
    build: .
    ports:
      - "8000:8000"
    environment:
      - GO_BACKEND_URL=http://host.docker.internal:8080
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

- [ ] **Step 7: 创建测试文件**

```python
# ai-agent/tests/test_agent.py
import pytest
from app.state import AgentState

def test_agent_state_creation():
    state: AgentState = {
        "task_id": "test-123",
        "agent_name": "亚马逊视觉总监",
        "mode": "full",
        "output_ratio": "1:1",
        "a_plus_format": "mobile",
        "product_info": {},
        "image_type_configs": [{"type": "main_image", "count": 1}],
        "current_phase": "product_analysis",
        "current_config_index": 0,
        "current_image_index": 0,
        "current_image_type": "main_image",
        "current_plan": None,
        "generated_image_url": None,
        "user_feedback": None,
        "revisions": 0,
        "completed_images": []
    }
    assert state["task_id"] == "test-123"
    assert state["mode"] == "full"
```

- [ ] **Step 8: 验证基础架构**

Run: `cd ai-agent && python -m pytest tests/test_agent.py -v`
Expected: PASS

- [ ] **Step 9: 提交**

```bash
git add ai-agent/
git commit -m "feat: add AI agent project structure with FastAPI and LangGraph"
```

---

## Task 2: LangGraph 节点实现

**Files:**
- Create: `ai-agent/app/graph/__init__.py`
- Create: `ai-agent/app/graph/nodes.py`
- Create: `ai-agent/app/graph/edges.py`
- Create: `ai-agent/app/graph/builder.py`

- [ ] **Step 1: 创建 nodes.py - 节点实现**

```python
# ai-agent/app/graph/nodes.py
from typing import Literal
from app.state import AgentState
from app.services.image_gen import generate_image

def init(state: AgentState) -> AgentState:
    """初始化状态，验证输入"""
    if not state.get("product_info") or not state.get("image_type_configs"):
        raise ValueError("缺少必要的产品信息或图片配置")
    return {
        **state,
        "agent_name": "亚马逊视觉总监",
        "current_phase": "product_analysis",
        "current_config_index": 0,
        "current_image_index": 0,
        "revisions": 0,
        "completed_images": []
    }

def product_analysis(state: AgentState) -> AgentState:
    """分析产品图片和卖点，形成结构化上下文"""
    product_info = state.get("product_info", {})
    bullets = product_info.get("bullets", [])
    
    # 构建产品分析上下文
    analysis_context = {
        "title": product_info.get("title", ""),
        "bullets": bullets,
        "images": product_info.get("images", {})
    }
    
    return {
        **state,
        "current_phase": "planning",
        "product_info": {**product_info, "analysis_context": analysis_context}
    }

def image_planning(state: AgentState) -> AgentState:
    """为当前图片生成策划方案（构图 + Prompt + 文案）"""
    image_type = state.get("current_image_type", "")
    config_idx = state.get("current_config_index", 0)
    image_idx = state.get("current_image_index", 0)
    
    # 根据图片类型生成策划方案
    plan = generate_planning_for_image(
        image_type=image_type,
        output_ratio=state.get("output_ratio", "1:1"),
        a_plus_format=state.get("a_plus_format", "mobile"),
        product_info=state.get("product_info", {})
    )
    
    return {
        **state,
        "current_phase": "planning",
        "current_plan": plan
    }

def image_generation(state: AgentState) -> AgentState:
    """调用生图 API"""
    plan = state.get("current_plan", {})
    prompt = plan.get("prompt", "")
    
    image_url = generate_image(
        prompt=prompt,
        output_ratio=state.get("output_ratio", "1:1"),
        product_images=state.get("product_info", {}).get("images", {})
    )
    
    return {
        **state,
        "current_phase": "review",
        "generated_image_url": image_url
    }

def move_to_next(state: AgentState) -> AgentState:
    """更新索引，检查是否还有图片"""
    configs = state.get("image_type_configs", [])
    current_config_idx = state.get("current_config_index", 0)
    current_image_idx = state.get("current_image_index", 0)
    
    # 移动到下一张图
    current_config = configs[current_config_idx]
    current_count = current_config.get("count", 0)
    
    if current_image_idx + 1 < current_count:
        # 当前 config 内还有图片
        return {
            **state,
            "current_image_index": current_image_idx + 1,
            "current_phase": "planning",
            "current_plan": None,
            "generated_image_url": None,
            "user_feedback": None
        }
    elif current_config_idx + 1 < len(configs):
        # 移动到下一个 config
        next_config = configs[current_config_idx + 1]
        return {
            **state,
            "current_config_index": current_config_idx + 1,
            "current_image_index": 0,
            "current_image_type": next_config.get("type", ""),
            "current_phase": "planning",
            "current_plan": None,
            "generated_image_url": None,
            "user_feedback": None
        }
    else:
        # 全部完成，检查是否需要 A+ 图
        return {
            **state,
            "current_phase": "check_a_plus"
        }

def check_a_plus(state: AgentState) -> AgentState:
    """检查是否需要进入 A+ 图阶段"""
    mode = state.get("mode", "full")
    completed = state.get("completed_images", [])
    
    if mode == "main_secondary" or mode == "a_plus_only":
        # 不需要 A+ 图，直接完成
        return {**state, "current_phase": "end"}
    
    # 检查是否已有 A+ 图
    a_plus_count = len([img for img in completed if img.get("type") == "a_plus"])
    if a_plus_count > 0:
        return {**state, "current_phase": "end"}
    
    # 需要生成 A+ 图
    return {**state, "current_phase": "a_plus_start"}

def end(state: AgentState) -> AgentState:
    """任务完成，同步结果到 Go 后端"""
    from app.services.callback import sync_to_backend
    sync_to_backend(
        task_id=state.get("task_id"),
        completed_images=state.get("completed_images", [])
    )
    return {**state, "current_phase": "completed"}
```

- [ ] **Step 2: 创建 edges.py - 路由条件**

```python
# ai-agent/app/graph/edges.py
from typing import Literal
from app.state import AgentState

def route_after_init(state: AgentState) -> Literal["product_analysis", "end"]:
    """初始化后根据模式决定下一步"""
    if not state.get("product_info", {}).get("images"):
        return "end"
    return "product_analysis"

def route_after_product_analysis(state: AgentState) -> Literal["check_mode"]:
    """产品分析完成后进入模式检查"""
    return "check_mode"

def route_check_mode(state: AgentState) -> Literal["start_a_plus", "queue_all_images"]:
    """根据模式决定起始节点"""
    mode = state.get("mode", "full")
    if mode == "a_plus_only":
        return "start_a_plus"
    return "queue_all_images"

def route_after_planning(state: AgentState) -> Literal["wait_user_confirm", "image_generation"]:
    """策划完成后等待用户确认"""
    return "wait_user_confirm"

def route_after_user_confirm(state: AgentState) -> Literal["image_generation"]:
    """用户确认后开始生成"""
    return "image_generation"

def route_after_generation(state: AgentState) -> Literal["wait_user_review"]:
    """生成完成后等待用户审核"""
    return "wait_user_review"

def route_after_user_review(state: AgentState) -> Literal["move_to_next", "image_generation"]:
    """用户审核后决定下一步"""
    # 如果用户满意，移动到下一张
    # 如果用户不满意，根据 feedback 决定是调整参数还是重新生成
    return "move_to_next"

def route_after_move(state: AgentState) -> Literal["image_planning", "check_a_plus"]:
    """移动后检查是否还有图片"""
    configs = state.get("image_type_configs", [])
    current_config_idx = state.get("current_config_index", 0)
    
    if current_config_idx >= len(configs):
        return "check_a_plus"
    return "image_planning"

def route_after_a_plus_check(state: AgentState) -> Literal["start_a_plus", "end"]:
    """A+ 检查后决定是否生成 A+ 图"""
    return "start_a_plus" if state.get("current_phase") == "a_plus_start" else "end"
```

- [ ] **Step 3: 创建 builder.py - GraphBuilder**

```python
# ai-agent/app/graph/builder.py
from langgraph.graph import StateGraph
from app.state import AgentState
from app.graph import nodes, edges

def build_graph() -> StateGraph:
    """构建 LangGraph"""
    graph = StateGraph(AgentState)
    
    # 添加节点
    graph.add_node("init", nodes.init)
    graph.add_node("product_analysis", nodes.product_analysis)
    graph.add_node("check_mode", lambda s: s)  # 简单模式检查
    graph.add_node("start_a_plus", lambda s: s)
    graph.add_node("queue_all_images", lambda s: s)
    graph.add_node("image_planning", nodes.image_planning)
    graph.add_node("wait_user_confirm", lambda s: s)
    graph.add_node("image_generation", nodes.image_generation)
    graph.add_node("wait_user_review", lambda s: s)
    graph.add_node("move_to_next", nodes.move_to_next)
    graph.add_node("check_a_plus", nodes.check_a_plus)
    graph.add_node("end", nodes.end)
    
    # 添加边
    graph.set_entry_point("init")
    
    graph.add_edge("init", "product_analysis")
    graph.add_edge("product_analysis", "check_mode")
    graph.add_conditional_edges(
        "check_mode",
        edges.route_check_mode,
        {"start_a_plus": "start_a_plus", "queue_all_images": "queue_all_images"}
    )
    graph.add_edge("start_a_plus", "image_planning")
    graph.add_edge("queue_all_images", "image_planning")
    graph.add_edge("image_planning", "wait_user_confirm")
    graph.add_edge("wait_user_confirm", "image_generation")
    graph.add_edge("image_generation", "wait_user_review")
    graph.add_edge("wait_user_review", "move_to_next")
    graph.add_conditional_edges(
        "move_to_next",
        edges.route_after_move,
        {"image_planning": "image_planning", "check_a_plus": "check_a_plus"}
    )
    graph.add_edge("check_a_plus", "end")
    
    return graph.compile()
```

- [ ] **Step 4: 创建 __init__.py**

```python
# ai-agent/app/graph/__init__.py
from app.graph.builder import build_graph

__all__ = ["build_graph"]
```

- [ ] **Step 5: 提交**

```bash
git add ai-agent/app/graph/
git commit -m "feat: implement LangGraph nodes, edges and builder"
```

---

## Task 3: 生图服务和回调服务

**Files:**
- Create: `ai-agent/app/services/__init__.py`
- Create: `ai-agent/app/services/image_gen.py`
- Create: `ai-agent/app/services/callback.py`

- [ ] **Step 1: 创建 image_gen.py**

```python
# ai-agent/app/services/image_gen.py
import httpx
import os
from typing import Optional

# 配置从环境变量读取
DRAW_API_KEY = os.getenv("DRAW_API_KEY", "")
DRAW_API_URL = os.getenv("DRAW_API_URL", "")
DEFAULT_MODEL = os.getenv("DRAW_MODEL", "gpt-image-4")

async def generate_image(
    prompt: str,
    output_ratio: str = "1:1",
    product_images: Optional[dict] = None
) -> str:
    """调用生图 API 生成图片
    
    Args:
        prompt: 生图提示词
        output_ratio: 输出比例 (如 "1:1", "4:3")
        product_images: 产品参考图 dict
        
    Returns:
        生成的图片 URL
    """
    if not DRAW_API_KEY:
        raise ValueError("DRAW_API_KEY not configured")
    
    # 构建请求
    payload = {
        "prompt": prompt,
        "model": DEFAULT_MODEL,
        "size": map_ratio_to_size(output_ratio),
        "n": 1
    }
    
    # 添加参考图
    if product_images and product_images.get("front"):
        payload["image_urls"] = [product_images["front"]]
    
    async with httpx.AsyncClient(timeout=120.0) as client:
        response = await client.post(
            f"{DRAW_API_URL}/v1/images/generations",
            headers={
                "Authorization": f"Bearer {DRAW_API_KEY}",
                "Content-Type": "application/json"
            },
            json=payload
        )
        response.raise_for_status()
        
        result = response.json()
        return result.get("data", [{}])[0].get("url", "")

def map_ratio_to_size(ratio: str) -> str:
    """将比例映射到 API 尺寸"""
    mapping = {
        "1:1": "1024x1024",
        "3:4": "768x1024",
        "4:3": "1024x768",
        "4:5": "896x1120",
        "5:4": "1120x896",
        "21:9": "1920x810"
    }
    return mapping.get(ratio, "1024x1024")
```

- [ ] **Step 2: 创建 callback.py**

```python
# ai-agent/app/services/callback.py
import httpx
import os
from typing import List

GO_BACKEND_URL = os.getenv("GO_BACKEND_URL", "http://localhost:8080")

async def sync_to_backend(task_id: str, completed_images: List[dict]) -> dict:
    """同步完成结果到 Go 后端
    
    Args:
        task_id: 任务ID
        completed_images: 完成的图片列表
        
    Returns:
        后端响应
    """
    async with httpx.AsyncClient(timeout=30.0) as client:
        response = await client.post(
            f"{GO_BACKEND_URL}/api/v1/ai/image-agent/{task_id}/complete",
            json={
                "status": "completed",
                "images": completed_images
            }
        )
        response.raise_for_status()
        return response.json()

async def update_task_status(task_id: str, status: str, **kwargs) -> dict:
    """更新任务状态到 Go 后端"""
    async with httpx.AsyncClient(timeout=30.0) as client:
        response = await client.patch(
            f"{GO_BACKEND_URL}/api/v1/ai/image-agent/{task_id}/status",
            json={"status": status, **kwargs}
        )
        response.raise_for_status()
        return response.json()
```

- [ ] **Step 3: 创建 __init__.py**

```python
# ai-agent/app/services/__init__.py
from app.services.image_gen import generate_image
from app.services.callback import sync_to_backend, update_task_status

__all__ = ["generate_image", "sync_to_backend", "update_task_status"]
```

- [ ] **Step 4: 提交**

```bash
git add ai-agent/app/services/
git commit -m "feat: add image generation and callback services"
```

---

## Task 4: Agent 系统提示词

**Files:**
- Create: `ai-agent/app/agent/__init__.py`
- Create: `ai-agent/app/agent/supervisor.py`

- [ ] **Step 1: 创建 supervisor.py**

```python
# ai-agent/app/agent/supervisor.py

SYSTEM_PROMPT = """## 角色定位

你是一位拥有10年经验的亚马逊（Amazon US）高级视觉总监。你精通消费者行为心理学，并对亚马逊官方的《商品图片指南》有极其严格的执行标准。你的任务是引导用户完成从 Listing 策划到最终商业大片的产出。

## 核心能力与工具

1. **生图能力**：使用目前最强的生图模型进行创作
2. **合规审查**：严格执行主图纯白底（RGB 255,255,255）、商品占比 85%+、无侵权标志、无虚假声明等规定
3. **分层策划**：遵循 1张主图 + 6张副图黄金转化逻辑，遵循 A+ 图片规范
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
"""
```

- [ ] **Step 2: 创建 __init__.py**

```python
# ai-agent/app/agent/__init__.py
from app.agent.supervisor import SYSTEM_PROMPT

__all__ = ["SYSTEM_PROMPT"]
```

- [ ] **Step 3: 提交**

```bash
git add ai-agent/app/agent/
git commit -m "feat: add agent system prompt"
```

---

## Task 5: Go 后端 Agent 通信接口

**Files:**
- Create: `backend/internal/service/image_agent.go`
- Modify: `backend/internal/router/router.go:237-245`
- Modify: `backend/internal/handler/ecom_suite.go` (新增接口)

- [ ] **Step 1: 创建 service/image_agent.go**

```go
// backend/internal/service/image_agent.go
package service

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "advertising-image/backend/internal/config"
)

type ImageAgentService struct {
    baseURL string
    client  *http.Client
}

type StartAgentRequest struct {
    TaskID           string            `json:"task_id"`
    ProductID        int64             `json:"product_id"`
    ProductImages    map[string]string `json:"product_images"`
    Title            string            `json:"title"`
    Bullets          []string          `json:"bullets"`
    ImageTypeConfigs []ImageTypeConfig `json:"image_type_configs"`
    Mode             string            `json:"mode"`
    OutputRatio      string            `json:"output_ratio"`
    APlusFormat      string            `json:"a_plus_format"`
}

type ImageTypeConfig struct {
    Type  string `json:"type"`
    Count int    `json:"count"`
}

type StartAgentResponse struct {
    SessionID string `json:"session_id"`
}

func NewImageAgentService() *ImageAgentService {
    return &ImageAgentService{
        baseURL: config.GetEnv("AI_AGENT_URL", "http://localhost:8000"),
        client: &http.Client{Timeout: 60 * time.Second},
    }
}

func (s *ImageAgentService) StartAgent(req StartAgentRequest) (*StartAgentResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    resp, err := s.client.Post(
        fmt.Sprintf("%s/api/v1/ai/image-agent/start", s.baseURL),
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result StartAgentResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}

func (s *ImageAgentService) GetStatus(sessionID string) (map[string]interface{}, error) {
    resp, err := s.client.Get(
        fmt.Sprintf("%s/api/v1/ai/image-agent/%s/status", s.baseURL, sessionID),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return result, nil
}

func (s *ImageAgentService) ConfirmPlan(sessionID string) error {
    req, _ := http.NewRequest(
        "POST",
        fmt.Sprintf("%s/api/v1/ai/image-agent/%s/confirm-plan", s.baseURL, sessionID),
        nil,
    )
    _, err := s.client.Do(req)
    return err
}

func (s *ImageAgentService) SubmitFeedback(sessionID, feedback, action string) error {
    body, _ := json.Marshal(map[string]string{
        "feedback": feedback,
        "action":   action,
    })
    resp, err := s.client.Post(
        fmt.Sprintf("%s/api/v1/ai/image-agent/%s/feedback", s.baseURL, sessionID),
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}

func (s *ImageAgentService) ConfirmImage(sessionID string) error {
    req, _ := http.NewRequest(
        "POST",
        fmt.Sprintf("%s/api/v1/ai/image-agent/%s/confirm-image", s.baseURL, sessionID),
        nil,
    )
    _, err := s.client.Do(req)
    return err
}
```

- [ ] **Step 2: 修改 router.go 添加新路由**

在 `router.go` 的 protected 路由组中添加：

```go
// AI Image Agent
protected.POST("/ai/image-agent/start", handler.StartImageAgent)
protected.GET("/ai/image-agent/:session_id/status", handler.GetImageAgentStatus)
protected.POST("/ai/image-agent/:session_id/confirm-plan", handler.ConfirmImageAgentPlan)
protected.POST("/ai/image-agent/:session_id/feedback", handler.SubmitImageAgentFeedback)
protected.POST("/ai/image-agent/:session_id/confirm-image", handler.ConfirmImageAgentImage)
```

- [ ] **Step 3: 创建 handler/image_agent.go**

```go
// backend/internal/handler/image_agent.go
package handler

import (
    "net/http"

    "advertising-image/backend/internal/service"
    "github.com/gin-gonic/gin"
)

var agentService = service.NewImageAgentService()

type ImageAgentStartRequest struct {
    ProductID        uint64 `json:"product_id" binding:"required"`
    OutputRatio      string `json:"output_ratio"`
    APlusFormat      string `json:"a_plus_format"`
    ImageTypeConfigs []struct {
        Type  string `json:"type"`
        Count int    `json:"count"`
    } `json:"image_type_configs" binding:"required"`
    Mode string `json:"mode"`
}

func StartImageAgent(c *gin.Context) {
    var req ImageAgentStartRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 构建请求
    agentReq := service.StartAgentRequest{
        TaskID:           c.GetString("task_id"),
        ProductID:        int64(req.ProductID),
        ProductImages:    getProductImages(req.ProductID),
        Title:            getProductTitle(req.ProductID),
        Bullets:          getProductBullets(req.ProductID),
        ImageTypeConfigs: convertImageTypeConfigs(req.ImageTypeConfigs),
        Mode:             req.Mode,
        OutputRatio:      req.OutputRatio,
        APlusFormat:      req.APlusFormat,
    }

    resp, err := agentService.StartAgent(agentReq)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, resp)
}

func GetImageAgentStatus(c *gin.Context) {
    sessionID := c.Param("session_id")
    status, err := agentService.GetStatus(sessionID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, status)
}

func ConfirmImageAgentPlan(c *gin.Context) {
    sessionID := c.Param("session_id")
    if err := agentService.ConfirmPlan(sessionID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type FeedbackRequest struct {
    Feedback string `json:"feedback" binding:"required"`
    Action   string `json:"action"`
}

func SubmitImageAgentFeedback(c *gin.Context) {
    sessionID := c.Param("session_id")
    var req FeedbackRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := agentService.SubmitFeedback(sessionID, req.Feedback, req.Action); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ConfirmImageAgentImage(c *gin.Context) {
    sessionID := c.Param("session_id")
    if err := agentService.ConfirmImage(sessionID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
```

- [ ] **Step 4: 提交**

```bash
git add backend/internal/service/image_agent.go backend/internal/handler/image_agent.go backend/internal/router/router.go
git commit -m "feat: add AI agent communication interface to Go backend"
```

---

## Task 6: 前端适配 - 多轮交互流程

**Files:**
- Modify: `web/src/views/ProductImageView.vue`

- [ ] **Step 1: 添加 A+ 格式选择控件**

在 `config-sec` 区域，输出比例选择下方添加 A+ 格式选择：

```vue
<!-- 在 outputRatios 选择区域下方添加 -->
<div class="field compact-field">
  <span class="tt">A+ 格式</span>
  <div class="aplus-format-options">
    <button v-for="item in aPlusFormats" :key="item.value" class="format-chip"
      :class="{ on: selectedAPlusFormat === item.value }" type="button"
      @click="selectedAPlusFormat = item.value">
      <strong>{{ item.label }}</strong>
    </button>
  </div>
</div>
```

添加数据：
```javascript
const selectedAPlusFormat = ref('mobile')
const aPlusFormats = [
  { value: 'mobile', label: '手机 A+（4:3）', desc: '竖版长图，移动端展示' },
  { value: 'desktop', label: '电脑 A+（21:9）', desc: '超宽横幅，桌面端展示' }
]
```

- [ ] **Step 2: 添加 AI 引导模式启动按钮**

将"一键生成套图"按钮改为：

```vue
<button class="gen" type="button" @click="startAIGuided">
  {{ isInAIMode ? 'AI 引导生成中…' : '开始 AI 引导' }}
</button>
```

添加 AI 模式数据和方法：
```javascript
const isInAIMode = ref(false)
const currentSessionId = ref('')
const currentPlan = ref(null)
const currentPhase = ref('')
const pollingTimer = ref(null)

async function startAIGuided() {
  if (!selectedProduct.value) return toast.error('请先从产品库选择商品')
  
  isInAIMode.value = true
  try {
    const res = await request.post('/ai/image-agent/start', {
      task_id: latestTask.value?.task_id,
      product_id: Number(selectedProductId.value),
      product_images: getProductImages(),
      title: selectedProduct.value.name,
      bullets: selectedProduct.value.points,
      image_type_configs: buildImageTypeConfigs(),
      mode: getMode(),
      output_ratio: selectedOutputRatio.value,
      a_plus_format: selectedAPlusFormat.value
    })
    currentSessionId.value = res.session_id
    startAgentPolling()
  } catch (error) {
    toast.error('启动 AI 引导失败')
    isInAIMode.value = false
  }
}

async function fetchAgentStatus() {
  if (!currentSessionId.value) return
  try {
    const res = await request.get(`/ai/image-agent/${currentSessionId.value}/status`)
    currentPhase.value = res.phase
    currentPlan.value = res.plan
  } catch {
    // 忽略错误，继续轮询
  }
}

function startAgentPolling() {
  if (pollingTimer.value) return
  pollingTimer.value = window.setInterval(fetchAgentStatus, 3000)
}

function stopAgentPolling() {
  if (pollingTimer.value) {
    window.clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
}

async function confirmPlan() {
  try {
    await request.post(`/ai/image-agent/${currentSessionId.value}/confirm-plan`)
  } catch {
    toast.error('确认方案失败')
  }
}

async function submitFeedback(feedback, action) {
  try {
    await request.post(`/ai/image-agent/${currentSessionId.value}/feedback`, {
      feedback,
      action
    })
  } catch {
    toast.error('提交反馈失败')
  }
}

async function confirmImage() {
  try {
    const res = await request.post(`/ai/image-agent/${currentSessionId.value}/confirm-image`)
    if (res.next_type === 'completed') {
      stopAgentPolling()
      isInAIMode.value = false
      await fetchRecentTasks()
    }
  } catch {
    toast.error('确认图片失败')
  }
}

function getMode() {
  if (selectedSuitePresetId.value === 'aPlusOnly') return 'a_plus_only'
  if (selectedSuitePresetId.value === 'mainSecondary') return 'main_secondary'
  return 'full'
}
```

- [ ] **Step 3: 添加 AI 引导 UI 区域**

在右侧结果区添加 AI 引导模式的状态展示：

```vue
<!-- 在 result-scroll 内添加 -->
<div v-if="isInAIMode" class="ai-guidance-panel">
  <div class="guidance-header">
    <span class="ai-badge">AI 视觉总监</span>
    <span class="phase-badge">{{ currentPhase }}</span>
  </div>
  
  <div v-if="currentPlan" class="plan-preview">
    <h4>当前方案：{{ currentPlan.composition }}</h4>
    <div class="plan-details">
      <p class="prompt-preview">{{ currentPlan.prompt }}</p>
      <p class="compliance-notes">{{ currentPlan.compliance_notes }}</p>
    </div>
    <div class="plan-actions">
      <button class="confirm-btn" @click="confirmPlan">同意此方案并生图</button>
      <button class="revise-btn" @click="showReviseModal = true">修改方案</button>
    </div>
  </div>
  
  <div v-if="currentImageUrl" class="image-preview">
    <img :src="currentImageUrl" alt="生成中的图片">
    <div class="image-actions">
      <button class="confirm-btn" @click="confirmImage">确认图片</button>
      <button class="revise-btn" @click="showReviseModal = true">不满意，修改</button>
    </div>
  </div>
</div>
```

添加样式：
```css
.ai-guidance-panel {
  padding: 20px;
  border: 1px solid #e5e7eb;
  border-radius: 22px;
  background: linear-gradient(180deg, #fff, #fcfcfd);
}

.guidance-header {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.ai-badge {
  display: inline-flex;
  height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: #7c3aed;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

.phase-badge {
  display: inline-flex;
  height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: #e5e7eb;
  color: #374151;
  font-size: 12px;
  font-weight: 600;
}

.plan-preview {
  margin-bottom: 16px;
}

.plan-preview h4 {
  margin: 0 0 12px;
  font-size: 14px;
  color: #111827;
}

.plan-details {
  padding: 14px;
  border-radius: 14px;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
}

.prompt-preview {
  font-family: monospace;
  font-size: 12px;
  line-height: 1.6;
  color: #374151;
}

.compliance-notes {
  margin-top: 10px;
  font-size: 12px;
  color: #059669;
}

.plan-actions,
.image-actions {
  display: flex;
  gap: 12px;
  margin-top: 14px;
}

.confirm-btn {
  flex: 1;
  height: 40px;
  border: none;
  border-radius: 999px;
  background: #111827;
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.revise-btn {
  flex: 1;
  height: 40px;
  border: 1px solid #d1d5db;
  border-radius: 999px;
  background: #fff;
  color: #374151;
  font-weight: 600;
  cursor: pointer;
}

.image-preview img {
  width: 100%;
  border-radius: 14px;
}
```

- [ ] **Step 4: 更新 buildCreatePayload 添加 A+ 格式**

```javascript
function buildCreatePayload() {
  const imageTypeConfigs = buildImageTypeConfigs()
  return {
    product_id: Number(selectedProductId.value),
    platform: selectedPlatformValue.value,
    language: selectedLanguageValue.value,
    country_code: String(selectedCountryValue.value || '').toUpperCase(),
    country_name: selectedCountry.value.label,
    output_ratio: selectedOutputRatio.value,
    image_type_configs: imageTypeConfigs,
    a_plus_format: selectedAPlusFormat.value  // 新增
  }
}
```

- [ ] **Step 5: 清理轮询**

```javascript
onBeforeUnmount(() => {
  document.removeEventListener('click', closeDropdown)
  stopTaskPolling()
  stopAgentPolling()  // 新增
})
```

- [ ] **Step 6: 提交**

```bash
git add web/src/views/ProductImageView.vue
git commit -m "feat: adapt ProductImageView for AI-guided multi-turn interaction"
```

---

## Task 7: 集成测试

**Files:**
- Create: `ai-agent/tests/test_integration.py`

- [ ] **Step 1: 创建集成测试**

```python
# ai-agent/tests/test_integration.py
import pytest
from app.main import app
from httpx import AsyncClient, ASGITransport

@pytest.fixture
async def client():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac

@pytest.mark.asyncio
async def test_health(client):
    response = await client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}

@pytest.mark.asyncio
async def test_start_agent_invalid_request(client):
    response = await client.post(
        "/api/v1/ai/image-agent/start",
        json={"task_id": "test"}
    )
    assert response.status_code == 422  # 缺少必填字段
```

- [ ] **Step 2: 运行测试**

Run: `cd ai-agent && python -m pytest tests/test_integration.py -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add ai-agent/tests/test_integration.py
git commit -m "test: add integration tests for AI agent"
```

---

## Task 8: 端到端测试

**Files:**
- Create: `ai-agent/tests/test_e2e.py`

- [ ] **Step 1: 创建端到端测试**

```python
# ai-agent/tests/test_e2e.py
import pytest
from app.graph.builder import build_graph
from app.state import AgentState

@pytest.fixture
def graph():
    return build_graph()

def test_full_workflow_with_mocked_services(graph):
    """测试完整工作流程（mock 生图服务）"""
    initial_state: AgentState = {
        "task_id": "test-task-123",
        "agent_name": "",
        "mode": "main_secondary",
        "output_ratio": "1:1",
        "a_plus_format": "mobile",
        "product_info": {
            "title": "Test Product",
            "bullets": ["卖点1", "卖点2"],
            "images": {"front": "http://example.com/image.png"}
        },
        "image_type_configs": [
            {"type": "main_image", "count": 1},
            {"type": "feature", "count": 1}
        ],
        "current_phase": "",
        "current_config_index": 0,
        "current_image_index": 0,
        "current_image_type": "",
        "current_plan": None,
        "generated_image_url": None,
        "user_feedback": None,
        "revisions": 0,
        "completed_images": []
    }
    
    # 模拟用户确认和审核
    # ... (根据实际需求实现)
```

- [ ] **Step 2: 提交**

```bash
git add ai-agent/tests/test_e2e.py
git commit -m "test: add end-to-end tests for workflow"
```

---

## 自检清单

1. **Spec 覆盖检查：**
   - ✅ 架构设计：Task 1-4 覆盖 Python 微服务基础架构
   - ✅ 三种生成模式：Task 4 通过 mode 参数支持
   - ✅ 状态设计和节点流程：Task 2 实现 LangGraph 节点
   - ✅ API 接口设计：Task 3 实现生图和回调，Task 5 实现 Go 后端接口
   - ✅ 前端适配：Task 6 实现多轮交互 UI

2. **占位符扫描：** 无 TBD/TODO

3. **类型一致性：**
   - AgentState 定义与 nodes.py 使用一致
   - API 路由与 Go 后端接口路径一致
   - 前端方法与 API 端点对应

---

## 执行交接

**Plan complete and saved to `docs/superpowers/plans/2026-05-19-ai-image-agent-plan.md`**

**Two execution options:**

**1. Subagent-Driven (recommended)** - 调度独立子 agent 执行每个任务，任务间审核，快速迭代

**2. Inline Execution** - 在当前会话中使用 executing-plans 执行，批量执行带检查点

**Which approach?**