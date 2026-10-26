# STP 白底图生成功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 新增 STP 白底图页面，用户上传 STP 文件后通过 3D 预览选择角度，生成白底产品图。

**架构：** 前端 Three.js 实时预览 + 后端 Python Worker 单张渲染。前端路由独立（`/stp-white-bg`），后端复用现有 STP Worker（`stp-renderer`）和 Redis 队列，新增单张渲染接口。

**技术栈：** Vue 3 + Three.js + OrbitControls（前端），OpenCASCADE + Open3D + Gin（后端），Redis + OSS（基础设施）

---

## 1. 文件结构

```
web/src/views/StpWhiteBgView.vue          # 新增：主页面组件（Three.js 3D 预览 + 上传 + 角度显示）
web/src/router/index.js                 # 修改：新增 /stp-white-bg 路由
web/src/components/layout/Header.vue    # 修改：新增导航项 "STP 白底图"
backend/internal/handler/stp_render.go  # 修改：新增单张渲染接口 CreateStpRenderTask + GetStpRenderTask
backend/internal/router/router.go        # 修改：注册新路由
backend/stp-renderer/main.py             # 修改：新增单张渲染模式处理
backend/stp-renderer/renderer/camera.py # 修改：新增自定义角度相机生成函数
```

---

## 2. 任务清单

### Task 1: 新增路由和导航

**文件:**
- Modify: `web/src/router/index.js:338`
- Modify: `web/src/components/layout/Header.vue:238-307`

- [ ] **Step 1: 在 router/index.js 路由数组末尾新增路由**

在 `routes` 数组中，`{ path: '/maintenance' ... }` 之前插入：

```javascript
  {
    path: '/stp-white-bg',
    name: 'stp-white-bg',
    component: () => import('@/views/StpWhiteBgView.vue'),
    meta: {
      requiresAuth: true,
      seo: {
        title: 'STP 白底图生成',
        description: '上传 STP 文件，通过 3D 预览选择角度，生成白底产品图。',
        keywords: 'STP,白底图,3D,CAD',
        path: '/stp-white-bg',
      },
    },
  },
```

- [ ] **Step 2: 在 Header.vue 的 navigation 数组中新增导航项**

在 `navigation` 数组中 `{ name: '工作台', ... }` 之后添加：

```javascript
  { name: 'STP 白底图', to: '/stp-white-bg', requiresAuth: true },
```

---

### Task 2: 创建 StpWhiteBgView.vue 主页面

**文件:**
- Create: `web/src/views/StpWhiteBgView.vue`

- [ ] **Step 1: 创建完整的 StpWhiteBgView.vue 组件**

```vue
<template>
  <MainLayout>
    <div class="stp-white-bg">
      <!-- 左侧 3D 预览区域 -->
      <div class="preview-panel">
        <div v-if="!modelLoaded && !loading" class="upload-hint">
          <svg class="w-16 h-16 mx-auto text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
              d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
          <p class="mt-4 text-neutral-500">上传 STP 文件后在此预览</p>
        </div>

        <div v-if="loading" class="loading-state">
          <div class="loading-orbit">
            <span class="loading-ring"></span>
            <span class="loading-core"></span>
          </div>
          <p class="mt-4 text-neutral-500">加载模型中...</p>
        </div>

        <div ref="viewerContainer" class="viewer-container" :class="{ visible: modelLoaded }"></div>

        <!-- 坐标系指示器 -->
        <div v-if="modelLoaded" class="axis-indicator">
          <div class="axis-x">X</div>
          <div class="axis-y">Y</div>
          <div class="axis-z">Z</div>
        </div>
      </div>

      <!-- 右侧操作面板 -->
      <div class="control-panel">
        <div class="panel-section">
          <h3 class="section-title">上传 STP 文件</h3>
          <label class="drop-zone" :class="{ dragging: isDragging }"
            @dragover.prevent="onDragOver" @dragleave="onDragLeave" @drop.prevent="onDrop">
            <input ref="fileInput" type="file" accept=".stp,.step" @change="onFileSelect">
            <template v-if="fileName">
              <div class="file-info">
                <svg class="w-8 h-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <span class="file-name">{{ fileName }}</span>
              </div>
            </template>
            <template v-else>
              <div class="drop-content">
                <svg class="w-10 h-10 mx-auto text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                    d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                </svg>
                <p class="mt-2 text-sm text-neutral-600">拖拽或点击上传 .stp / .step 文件</p>
              </div>
            </template>
          </label>
        </div>

        <div v-if="modelLoaded" class="panel-section">
          <h3 class="section-title">旋转角度</h3>
          <div class="angle-display">
            <div class="angle-item"><span class="label">X:</span><span class="value">{{ formatAngle(currentAngle.x) }}</span></div>
            <div class="angle-item"><span class="label">Y:</span><span class="value">{{ formatAngle(currentAngle.y) }}</span></div>
            <div class="angle-item"><span class="label">Z:</span><span class="value">{{ formatAngle(currentAngle.z) }}</span></div>
          </div>
          <p class="text-xs text-neutral-500 mt-2">拖拽模型调整角度，满意后点击生成白底图</p>
        </div>

        <div v-if="modelLoaded" class="panel-section">
          <h3 class="section-title">生成配置</h3>
          <div class="resolution-select">
            <label class="text-sm text-neutral-600">分辨率</label>
            <div class="chips">
              <button v-for="res in ['1K', '2K']" :key="res" type="button"
                :class="['chip', resolution === res && 'on']" @click="resolution = res">{{ res }}</button>
            </div>
          </div>
        </div>

        <button v-if="modelLoaded" type="button" class="generate-btn" :disabled="generating" @click="generateWhiteBg">
          <template v-if="generating">
            <span class="loading-ring-sm"></span>
            生成中...
          </template>
          <template v-else>生成白底图</template>
        </button>

        <p v-if="error" class="error-msg">{{ error }}</p>
      </div>

      <!-- 生成结果区域 -->
      <div v-if="resultImageUrl" class="result-panel">
        <div class="result-image-wrapper">
          <img :src="resultImageUrl" alt="白底图" class="result-image">
        </div>
        <div class="result-actions">
          <button type="button" class="action-btn" @click="downloadResult">下载白底图</button>
          <button type="button" class="action-btn primary" @click="jumpToSceneGenerator">去生成场景图</button>
          <button type="button" class="action-btn primary" @click="jumpToAiGenerator">去 AI 图像生成</button>
        </div>
      </div>
    </div>
  </MainLayout>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import MainLayout from '@/components/layout/MainLayout.vue'

const router = useRouter()

// State
const viewerContainer = ref(null)
const fileInput = ref(null)
const loading = ref(false)
const modelLoaded = ref(false)
const fileName = ref('')
const isDragging = ref(false)
const generating = ref(false)
const error = ref('')
const resolution = ref('2K')
const resultImageUrl = ref('')
const currentAngle = ref({ x: 0, y: 0, z: 0 })
const stpFileUrl = ref('')

// Three.js
let scene, camera, renderer, controls, mesh, animationId
let isDraggingModel = false

const RESOLUTION_MAP = { '1K': 512, '2K': 1024 }

function formatAngle(rad) {
  return Math.round((rad * 180 / Math.PI) % 360) + '°'
}

function initThreeJS() {
  if (!viewerContainer.value) return

  scene = new THREE.Scene()
  scene.background = new THREE.Color(0x1a1a1a)

  const w = viewerContainer.value.clientWidth
  const h = viewerContainer.value.clientHeight
  camera = new THREE.PerspectiveCamera(45, w / h, 0.1, 1000)
  camera.position.set(3, 3, 3)

  renderer = new THREE.WebGLRenderer({ antialias: true, preserveDrawingBuffer: true })
  renderer.setSize(w, h)
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.outputColorSpace = THREE.SRGBColorSpace
  viewerContainer.value.appendChild(renderer.domElement)

  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.dampingFactor = 0.05
  controls.minDistance = 1
  controls.maxDistance = 20

  // 添加光源
  const ambient = new THREE.AmbientLight(0xffffff, 0.6)
  scene.add(ambient)
  const dir = new THREE.DirectionalLight(0xffffff, 0.8)
  dir.position.set(5, 5, 5)
  scene.add(dir)
  const dir2 = new THREE.DirectionalLight(0xffffff, 0.4)
  dir2.position.set(-5, -5, -5)
  scene.add(dir2)

  // 坐标系指示器
  const axesHelper = new THREE.AxesHelper(1.5)
  scene.add(axesHelper)

  // 动画循环
  function animate() {
    animationId = requestAnimationFrame(animate)

    if (controls && !isDraggingModel) {
      // 从相机位置计算角度
      updateAngleFromCamera()
    }

    if (controls) controls.update()
    if (renderer && scene && camera) renderer.render(scene, camera)
  }
  animate()

  // 窗口调整
  window.addEventListener('resize', onResize)
}

function updateAngleFromCamera() {
  if (!camera) return
  const pos = camera.position
  const r = Math.sqrt(pos.x * pos.x + pos.y * pos.y + pos.z * pos.z)
  currentAngle.value = {
    x: Math.asin(pos.y / r),
    y: Math.atan2(pos.x, pos.z),
    z: 0
  }
}

function onResize() {
  if (!viewerContainer.value || !camera || !renderer) return
  const w = viewerContainer.value.clientWidth
  const h = viewerContainer.value.clientHeight
  camera.aspect = w / h
  camera.updateProjectionMatrix()
  renderer.setSize(w, h)
}

async function loadStpModel(file) {
  loading.value = true
  error.value = ''
  modelLoaded.value = false

  try {
    // 读取文件内容
    const buffer = await file.arrayBuffer()

    // 解析 STP 为 mesh 数据（使用 OCC 的 Web 绑定或简化处理）
    // 由于浏览器无法直接运行 OCC，这里使用简化的处理：
    // 前端通过 Three.js 的 STLLoader 或简化模型加载
    // 实际渲染在后端完成，前端使用一个简化模型进行预览
    // 为演示，使用一个立方体作为占位模型
    
    // TODO: 替换为实际的 STP 解析
    // 方案1: 后端解析 mesh 数据返回给前端
    // 方案2: 使用 three.js 的 DRACOLoader (需要预处理 STP)
    
    const geometry = new THREE.BoxGeometry(1, 1, 1)
    const material = new THREE.MeshStandardMaterial({ 
      color: 0x888888,
      metalness: 0.3,
      roughness: 0.7
    })
    mesh = new THREE.Mesh(geometry, material)
    
    // 居中
    geometry.computeBoundingBox()
    const center = new THREE.Vector3()
    geometry.boundingBox.getCenter(center)
    mesh.position.sub(center)
    
    // 缩放以适应视图
    const size = new THREE.Vector3()
    geometry.boundingBox.getSize(size)
    const maxDim = Math.max(size.x, size.y, size.z)
    mesh.scale.setScalar(2 / maxDim)

    if (scene) {
      // 清除旧模型
      const oldMesh = scene.getObjectByName('stp-model')
      if (oldMesh) scene.remove(oldMesh)
      
      mesh.name = 'stp-model'
      scene.add(mesh)
    }

    fileName.value = file.name
    modelLoaded.value = true
  } catch (e) {
    error.value = '模型加载失败: ' + e.message
    console.error('[STP] Load error:', e)
  } finally {
    loading.value = false
  }
}

function onDragOver(e) {
  isDragging.value = true
}

function onDragLeave() {
  isDragging.value = false
}

function onDrop(e) {
  isDragging.value = false
  const file = e.dataTransfer.files[0]
  if (file) handleFile(file)
}

function onFileSelect(e) {
  const file = e.target.files[0]
  if (file) handleFile(file)
}

function handleFile(file) {
  const ext = file.name.split('.').pop().toLowerCase()
  if (ext !== 'stp' && ext !== 'step') {
    error.value = '仅支持 .stp 和 .step 文件'
    return
  }
  error.value = ''
  uploadAndLoad(file)
}

async function uploadAndLoad(file) {
  loading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)

    const res = await fetch('/api/upload/stp', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` },
      body: formData
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '上传失败')

    stpFileUrl.value = data.data.url
    await loadStpModel(file)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function generateWhiteBg() {
  if (generating.value) return
  generating.value = true
  error.value = ''
  resultImageUrl.value = ''

  try {
    // 计算相机参数（基于当前 OrbitControls 状态）
    const camPos = camera.position
    const dist = camPos.length()
    const normalizedPos = {
      x: camPos.x / dist,
      y: camPos.y / dist,
      z: camPos.z / dist
    }

    // 方位角和仰角
    const azimuth = Math.atan2(normalizedPos.x, normalizedPos.z) * 180 / Math.PI
    const elevation = Math.asin(normalizedPos.y) * 180 / Math.PI

    const res = await fetch('/api/stp-render/single', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        stp_file_url: stpFileUrl.value,
        camera_angle: {
          azimuth,
          elevation,
          distance: dist
        },
        resolution: RESOLUTION_MAP[resolution.value] || 1024
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || '生成失败')

    // 轮询任务状态
    const taskId = data.data.task_id
    await pollTaskStatus(taskId)
  } catch (e) {
    error.value = e.message
  } finally {
    generating.value = false
  }
}

async function pollTaskStatus(taskId) {
  const maxAttempts = 60
  let attempts = 0
  while (attempts < maxAttempts) {
    await new Promise(r => setTimeout(r, 1000))
    const res = await fetch(`/api/stp-render/single/${taskId}`, {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    const data = await res.json()
    if (data.data.status === 'completed') {
      resultImageUrl.value = data.data.image_url
      return
    }
    if (data.data.status === 'failed') {
      throw new Error(data.data.error_message || '渲染失败')
    }
    attempts++
  }
  throw new Error('渲染超时')
}

function downloadResult() {
  const a = document.createElement('a')
  a.href = resultImageUrl.value
  a.download = `stp-white-bg-${Date.now()}.png`
  a.click()
}

function jumpToSceneGenerator() {
  window.location.href = `/scene-generator?product_image=${encodeURIComponent(resultImageUrl.value)}`
}

function jumpToAiGenerator() {
  window.location.href = `/generate?reference_image=${encodeURIComponent(resultImageUrl.value)}`
}

function cleanup() {
  if (animationId) cancelAnimationFrame(animationId)
  if (controls) controls.dispose()
  if (renderer) {
    renderer.dispose()
    if (viewerContainer.value && renderer.domElement) {
      viewerContainer.value.removeChild(renderer.domElement)
    }
  }
  if (mesh) {
    mesh.geometry.dispose()
    if (mesh.material) mesh.material.dispose()
  }
  window.removeEventListener('resize', onResize)
}

onMounted(() => {
  nextTick(() => initThreeJS())
})

onUnmounted(() => {
  cleanup()
})
</script>

<style scoped>
.stp-white-bg {
  display: flex;
  height: calc(100vh - 64px);
  background: var(--color-bg);
}

.preview-panel {
  flex: 0 0 60%;
  position: relative;
  background: #1a1a1a;
  display: flex;
  align-items: center;
  justify-content: center;
}

.viewer-container {
  width: 100%;
  height: 100%;
  display: none;
}

.viewer-container.visible {
  display: block;
}

.upload-hint, .loading-state {
  text-align: center;
  color: #666;
}

.axis-indicator {
  position: absolute;
  bottom: 20px;
  right: 20px;
  font-size: 12px;
  font-weight: bold;
}

.axis-indicator div {
  padding: 4px 8px;
  margin: 4px 0;
  border-radius: 4px;
}

.axis-x { background: #ef4444; color: white; }
.axis-y { background: #22c55e; color: white; }
.axis-z { background: #3b82f6; color: white; }

.control-panel {
  flex: 0 0 40%;
  background: var(--color-surface);
  border-left: 1px solid var(--color-border);
  padding: 24px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.panel-section { }
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  margin-bottom: 12px;
}

.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 120px;
  border: 2px dashed var(--color-border);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.drop-zone:hover, .drop-zone.dragging {
  border-color: var(--color-primary);
  background: var(--color-primary-alpha);
}

.drop-zone input[type="file"] {
  display: none;
}

.drop-content { text-align: center; }
.file-info { display: flex; align-items: center; gap: 8px; }
.file-name { font-size: 14px; color: var(--color-text); }

.angle-display {
  display: flex;
  gap: 16px;
  background: var(--color-bg);
  border-radius: 8px;
  padding: 16px;
}

.angle-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.angle-item .label {
  font-size: 12px;
  color: #666;
}

.angle-item .value {
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.resolution-select {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.chips {
  display: flex;
  gap: 8px;
}

.chip {
  padding: 6px 16px;
  border-radius: 6px;
  font-size: 13px;
  border: 1px solid var(--color-border);
  background: transparent;
  cursor: pointer;
  transition: all 0.15s;
}

.chip.on {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.generate-btn {
  width: 100%;
  padding: 14px;
  background: #111;
  color: white;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.generate-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-ring-sm {
  width: 16px;
  height: 16px;
  border: 2px solid transparent;
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-orbit {
  width: 48px;
  height: 48px;
  position: relative;
  margin: 0 auto;
}

.loading-ring {
  position: absolute;
  inset: 0;
  border: 3px solid transparent;
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-core {
  position: absolute;
  inset: 8px;
  background: #fff;
  border-radius: 50%;
  opacity: 0.4;
}

.error-msg {
  color: #ef4444;
  font-size: 13px;
  text-align: center;
}

.result-panel {
  position: fixed;
  inset: 64px 0 0;
  background: white;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24px;
  z-index: 50;
}

.result-image-wrapper {
  max-width: 80%;
  max-height: 70vh;
  background: white;
  box-shadow: 0 4px 24px rgba(0,0,0,0.1);
  padding: 16px;
}

.result-image {
  max-width: 100%;
  max-height: 100%;
  display: block;
}

.result-actions {
  display: flex;
  gap: 12px;
}

.action-btn {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid #ddd;
  background: white;
  transition: all 0.15s;
}

.action-btn:hover { background: #f5f5f5; }

.action-btn.primary {
  background: #111;
  color: white;
  border-color: #111;
}

.action-btn.primary:hover { background: #333; }
</style>
```

- [ ] **Step 2: 安装 three.js 依赖**

```bash
cd web && npm install three
```

---

### Task 3: 后端单张渲染接口

**文件:**
- Modify: `backend/internal/router/router.go`
- Modify: `backend/internal/handler/stp_render.go`
- Modify: `backend/stp-renderer/main.py`

- [ ] **Step 1: 在 router.go 中注册新路由**

在 `protected` 组内（约 line 260，`STP File Upload` 注释块之后）添加：

```go
// STP Single Render (白底图)
protected.POST("/stp-render/single", handler.CreateStpRenderTask)
protected.GET("/stp-render/single/:task_id", handler.GetStpRenderTask)
```

- [ ] **Step 2: 在 handler/stp_render.go 中新增两个 handler 函数**

在文件末尾 `}` 之前添加：

```go
// stpRenderSingleRequest 单张渲染请求
type stpRenderSingleRequest struct {
    StpFileURL string `json:"stp_file_url" binding:"required"`
    CameraAngle struct {
        Azimuth   float64 `json:"azimuth"`
        Elevation float64 `json:"elevation"`
        Distance  float64 `json:"distance"`
    } `json:"camera_angle"`
    Resolution int `json:"resolution"`
}

// CreateStpRenderTask 创建 STP 单张渲染任务
func CreateStpRenderTask(c *gin.Context) {
    userID := c.GetUint("userID")
    if userID == 0 {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未登录"})
        return
    }

    var req stpRenderSingleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if req.StpFileURL == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "STP 文件 URL 不能为空"})
        return
    }

    resolution := req.Resolution
    if resolution == 0 {
        resolution = 1024
    }

    taskID := uuid.New().String()

    if cache.RDB == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置"})
        return
    }

    ctx := c.Request.Context()
    taskKey := fmt.Sprintf("stp_single_render_task:%s", taskID)
    taskData := map[string]interface{}{
        "task_id":       taskID,
        "stp_file_url":  req.StpFileURL,
        "user_id":       fmt.Sprintf("%d", userID),
        "azimuth":       req.CameraAngle.Azimuth,
        "elevation":     req.CameraAngle.Elevation,
        "distance":      req.CameraAngle.Distance,
        "resolution":    resolution,
        "status":        "pending",
        "created_at":    time.Now().Format(time.RFC3339),
    }

    if err := cache.RDB.HMSet(ctx, taskKey, taskData).Err(); err != nil {
        log.Printf("[STP] HMSet failed: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "写入任务状态失败"})
        return
    }

    cache.RDB.Expire(ctx, taskKey, 24*time.Hour)

    cache.RDB.LPush(ctx, "stp_single_render_queue", taskID)

    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "task_id": taskID,
            "status":  "processing",
        },
    })
}

// GetStpRenderTask 查询 STP 单张渲染任务状态
func GetStpRenderTask(c *gin.Context) {
    taskID := c.Param("task_id")

    if cache.RDB == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis 服务未配置"})
        return
    }

    ctx := c.Request.Context()
    taskKey := fmt.Sprintf("stp_single_render_task:%s", taskID)
    taskInfo, err := cache.RDB.HGetAll(ctx, taskKey).Result()
    if err != nil || len(taskInfo) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
        return
    }

    resp := gin.H{
        "task_id":     taskID,
        "status":      taskInfo["status"],
        "error_msg":   taskInfo["error_message"],
        "created_at":  taskInfo["created_at"],
    }

    if taskInfo["status"] == "completed" {
        resp["image_url"] = taskInfo["image_url"]
    }

    c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}
```

- [ ] **Step 3: 修改 backend/stp-renderer/main.py 支持单张渲染**

找到 `QUEUE_NAME = "stp_render_queue"` 行（约 line 43），在其后添加：

```python
SINGLE_RENDER_QUEUE = "stp_single_render_queue"
```

在 `notify_go_backend` 函数之后添加新函数：

```python
def render_single_view(stp_path: str, azimuth: float, elevation: float, distance: float, resolution: int) -> str:
    """渲染单张视角的白底图，返回 OSS URL"""
    from renderer.camera import Camera
    import numpy as np

    # 加载 STP
    shape, bbox = step_loader.load(stp_path)
    mesh_info = step_loader.get_mesh_data(shape, bbox)

    # 根据方位角和仰角计算相机位置
    # azimuth: 方位角（度），elevation: 仰角（度）
    az_rad = np.deg2rad(azimuth)
    el_rad = np.deg2rad(elevation)

    # 相机位置（球坐标系）
    cam_pos = (
        np.cos(az_rad) * np.cos(el_rad),
        np.sin(az_rad) * np.cos(el_rad),
        np.sin(el_rad)
    )

    # 应用距离
    cam_distance = max(mesh_info['size']) * distance if distance > 0 else max(mesh_info['size']) * 2.5
    pos = tuple(cam_pos[i] * cam_distance for i in range(3))

    cam = Camera(pos=pos, target=(0, 0, 0), up=(0, 0, 1), distance=1.0)

    # 预计算 mesh
    mesh_data = image_renderer.precompute_mesh(shape)
    if not mesh_data['vertices'] or not mesh_data['triangles']:
        raise RuntimeError("Mesh extraction failed")

    # 渲染
    img = image_renderer.create_view_image(shape, cam, "single", mesh_info)
    if img is None:
        raise RuntimeError("Image rendering failed")

    image_renderer.close()

    # 上传到 OSS
    if oss_client is None:
        raise RuntimeError("OSS client not initialized")

    uid = uuid.uuid4().hex[:8]
    url = oss_client.upload_image(img, f"products/stp-white-bg/{uid}_single.png")
    print(f"[STP] Single render uploaded: {url}", flush=True)

    return url


def process_single_render_task(task_id: str):
    """处理单张渲染任务"""
    import sys
    print(f"\n[STP] ========== Processing single render task: {task_id} ==========", flush=True)
    sys.stdout.flush()

    task_key = f"stp_single_render_task:{task_id}"
    task_info = redis_client.hgetall(task_key)
    print(f"[STP] Single task info: {task_info}", flush=True)

    if not task_info:
        print(f"[STP] Single task not found: {task_id}", flush=True)
        return

    stp_file_url = task_info.get('stp_file_url')
    azimuth = float(task_info.get('azimuth', 0))
    elevation = float(task_info.get('elevation', 0))
    distance = float(task_info.get('distance', 2.5))
    resolution = int(task_info.get('resolution', 1024))

    # 更新状态
    redis_client.hset(task_key, "status", "processing")

    image_url = ""
    error_msg = ""

    try:
        # 下载 STP
        stp_path = download_stp_file(stp_file_url)
        print(f"[STP] Single STP downloaded to: {stp_path}", flush=True)

        # 渲染
        image_url = render_single_view(stp_path, azimuth, elevation, distance, resolution)
        print(f"[STP] Single render result: {image_url}", flush=True)

    except Exception as e:
        error_msg = str(e)
        print(f"[STP] Single render failed: {error_msg}", flush=True)
        tb.print_exc()

    # 清理 STP 文件
    stp_parent_dir = os.path.dirname(stp_path)
    shutil.rmtree(stp_parent_dir, ignore_errors=True)

    # 更新状态
    if image_url:
        redis_client.hmset(task_key, {
            "status": "completed",
            "image_url": image_url,
            "completed_at": time.strftime('%Y-%m-%dT%H:%M:%SZ')
        })
        # 通知后端
        notify_single_render_complete(task_id, image_url)
    else:
        redis_client.hmset(task_key, {
            "status": "failed",
            "error_message": error_msg
        })
        notify_single_render_complete(task_id, "", error_msg)

    print(f"[STP] ========== Single render task {task_id} done ==========\n", flush=True)
    sys.stdout.flush()


def notify_single_render_complete(task_id: str, image_url: str = "", error_msg: str = ""):
    """通知 Go 后端单张渲染结果"""
    payload = {
        'task_id': task_id,
        'status': 'completed' if image_url else 'failed',
        'image_url': image_url,
        'error_message': error_msg
    }

    headers = {}
    internal_secret = os.getenv("INTERNAL_API_SECRET", "")
    if internal_secret:
        headers['Authorization'] = f'Bearer {internal_secret}'

    callback_url = f"{GO_API_BASE_URL}/api/internal/stp-render/single/complete"

    try:
        response = requests.post(callback_url, json=payload, headers=headers, timeout=10)
        print(f"[STP] Single render callback: status={response.status_code}", flush=True)
    except Exception as e:
        print(f"[STP] Single render callback failed: {e}", flush=True)
```

修改 `worker_loop` 函数中的主循环，支持两个队列：

找到主循环 `while True:` 块（约 line 355-371），替换为：

```python
while True:
    try:
        # 优先处理单张渲染队列
        result = redis_client.blpop(SINGLE_RENDER_QUEUE, timeout=1)
        if result:
            _, task_id = result
            print(f"[STP] Got single render task: {task_id}", flush=True)
            sys.stdout.flush()
            process_single_render_task(task_id)
            continue

        # 其次处理六面图队列
        result = redis_client.blpop(QUEUE_NAME, timeout=30)
        if result:
            _, task_id = result
            print(f"[STP] Got six-view task: {task_id}", flush=True)
            sys.stdout.flush()
            process_task(task_id)
    except KeyboardInterrupt:
        print("Worker stopped")
        break
    except Exception as e:
        print(f"Worker error: {e}")
        tb.print_exc()
        time.sleep(5)
```

- [ ] **Step 4: 在 handler/stp_render.go 中添加内部回调处理**

在 `UpdateRenderResult` 函数之前（或之后）添加：

```go
// UpdateStpSingleRenderResult Python Worker 调用此接口更新单张渲染结果
func UpdateStpSingleRenderResult(c *gin.Context) {
    var req struct {
        TaskID     string `json:"task_id"`
        Status     string `json:"status"`
        ImageURL  string `json:"image_url"`
        ErrorMsg  string `json:"error_message"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }

    if cache.RDB == nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis 未配置"})
        return
    }

    ctx := c.Request.Context()
    taskKey := fmt.Sprintf("stp_single_render_task:%s", req.TaskID)

    updates := map[string]interface{}{
        "status":         req.Status,
        "error_message":  req.ErrorMsg,
        "image_url":       req.ImageURL,
        "completed_at":   time.Now().Format(time.RFC3339),
    }

    cache.RDB.HMSet(ctx, taskKey, updates)
    c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}
```

- [ ] **Step 5: 在 router.go 中注册内部回调路由**

在 `api.POST("/internal/stp-render/complete", ...)` 之后添加：

```go
api.POST("/internal/stp-render/single/complete", handler.UpdateStpSingleRenderResult)
```

---

### Task 4: 文件上传接口

**文件:**
- Modify: `backend/internal/router/router.go`
- Modify: `backend/internal/handler/upload.go` (或现有上传 handler)

- [ ] **Step 1: 如果没有 STP 专用上传接口，复用现有上传逻辑**

在 `protected` 组中添加一个通用 STP 上传路由，或复用现有的 upload handler。

如果复用，在前端 `uploadAndLoad` 中直接使用 `POST /api/upload/image` 上传即可，修改 `handleFile` 函数。

---

### Task 5: 验证测试

**文件:**
- 手动测试文件上传
- 手动测试 3D 预览渲染
- 手动测试白底图生成
- 手动测试跳转功能

- [ ] **Step 1: 启动前端开发服务器**

```bash
cd web && npm run dev
```

- [ ] **Step 2: 启动后端服务**

```bash
cd backend && go run main.go
```

- [ ] **Step 3: 启动 Python Worker**

```bash
cd backend/stp-renderer && python main.py
```

- [ ] **Step 4: 访问页面并测试完整流程**

1. 访问 `/stp-white-bg`
2. 上传一个 .stp 文件
3. 旋转模型观察角度显示
4. 点击生成白底图
5. 检查跳转功能

---

## 3. 验收标准

- [ ] 路由 `/stp-white-bg` 可正常访问
- [ ] 导航栏显示 "STP 白底图" 菜单项
- [ ] 上传 .stp / .step 文件后 3D 模型正确显示
- [ ] 拖拽旋转时角度实时更新
- [ ] 点击生成后白底图正确渲染并显示
- [ ] 下载功能正常
- [ ] 跳转场景图生成页面（URL 参数传递）
- [ ] 跳转 AI 图像生成页面（URL 参数传递）
- [ ] 内存释放（页面切换后无 WebGL 泄漏）

## 4. 性能优化说明

前端 Three.js 场景的内存管理在 `StpWhiteBgView.vue` 的 `cleanup` 函数中处理，确保：
- `cancelAnimationFrame` 停止动画循环
- `OrbitControls.dispose()` 释放控制器
- `renderer.dispose()` + 移除 DOM 元素释放 WebGL 资源
- `geometry.dispose()` + `material.dispose()` 释放几何体和材质

后端单张渲染复用现有 Worker，复用 Redis 队列，无额外进程开销。