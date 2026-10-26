<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
    <div class="mb-10">
      <h1 class="text-3xl sm:text-4xl font-bold text-neutral-900 mb-3">消除文字 / 水印</h1>
      <p class="text-lg text-neutral-600">一键智能消除图片中的文字、水印、Logo 及版权标识，AI 自动修复填充</p>
      <div class="inline-flex items-center gap-1.5 mt-2 px-3 py-1.5 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-700">
        <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        生成的图片仅保存 3 天，请及时下载保存
      </div>
    </div>

    <div class="flex gap-6">
      <!-- 左侧：上传与操作 -->
      <div class="w-96 flex-shrink-0 space-y-4">
        <!-- 上传区域 -->
        <div class="border border-neutral-200 rounded-lg p-4">
          <h3 class="text-sm font-semibold text-neutral-900 mb-3">上传图片</h3>

          <div v-if="!originalImage" class="flex flex-col items-center justify-center min-h-[240px] gap-2">
            <label
              class="flex flex-col items-center justify-center w-full min-h-[200px] border-2 border-dashed rounded-lg cursor-pointer transition-colors"
              :class="isDragging ? 'border-neutral-900 bg-neutral-50' : 'border-neutral-300 hover:border-neutral-500'"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <svg class="w-12 h-12 mb-3 transition-colors" :class="isDragging ? 'text-neutral-700' : 'text-neutral-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              <span class="transition-colors" :class="isDragging ? 'text-neutral-700 font-medium' : 'text-neutral-500'">{{ isDragging ? '松开鼠标上传' : '点击或拖拽上传图片' }}</span>
              <span class="text-xs text-neutral-400 mt-1">支持含水印、文字的各类图片</span>
              <span class="text-xs text-neutral-400">文件大小 ≤ 10MB</span>
              <input ref="fileInputRef" type="file" accept="image/*" class="hidden" @change="handleFileChange" />
            </label>
            <div v-if="uploadError" class="flex items-center gap-1.5 w-full px-3 py-2 bg-red-50 border border-red-200 rounded-lg text-xs text-red-600">
              <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {{ uploadError }}
            </div>
          </div>

          <div v-else class="flex flex-col items-center">
            <img :src="originalImage" class="max-w-full max-h-[240px] rounded border border-neutral-200" alt="原图预览" />
            <button class="mt-3 px-3 py-1 text-sm bg-red-500 text-white rounded hover:bg-red-600" @click="clearImage">
              清除图片
            </button>
          </div>
        </div>

        <!-- 说明信息 -->
        <div class="border border-neutral-200 rounded-lg p-4">
          <h3 class="text-sm font-semibold text-neutral-900 mb-3">处理说明</h3>
          <ul class="space-y-2 text-xs text-neutral-500">
            <li class="flex items-start gap-2">
              <span class="w-1.5 h-1.5 rounded-full bg-neutral-400 mt-1.5 flex-shrink-0"></span>
              自动识别并消除图片中的文字、水印、Logo
            </li>
            <li class="flex items-start gap-2">
              <span class="w-1.5 h-1.5 rounded-full bg-neutral-400 mt-1.5 flex-shrink-0"></span>
              AI 智能填充消除区域，还原真实背景
            </li>
            <li class="flex items-start gap-2">
              <span class="w-1.5 h-1.5 rounded-full bg-neutral-400 mt-1.5 flex-shrink-0"></span>
              保持原图主体内容与色彩不变
            </li>
            <li class="flex items-start gap-2">
              <span class="w-1.5 h-1.5 rounded-full bg-amber-500 mt-1.5 flex-shrink-0"></span>
              效果受水印复杂程度影响，复杂情况可多次尝试
            </li>
          </ul>
        </div>

        <!-- 操作按钮 -->
        <div class="space-y-3">
          <button
            class="w-full px-4 py-3 text-sm font-medium rounded-lg transition-colors"
            :class="originalImage && !processing ? 'bg-neutral-900 text-white hover:bg-neutral-800' : 'bg-neutral-200 text-neutral-400 cursor-not-allowed'"
            :disabled="!originalImage || processing"
            @click="processImage"
          >
            <span v-if="processing" class="inline-flex items-center gap-2">
              <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              AI 处理中，请稍候...
            </span>
            <span v-else>一键消除文字 / 水印</span>
          </button>

          <button
            v-if="resultImageUrl"
            class="block w-full px-4 py-2 text-sm font-medium text-center rounded-lg border border-neutral-300 hover:bg-neutral-50 transition-colors"
            type="button"
            @click="downloadImage(resultImageUrl, `eraser-result-${Date.now()}.png`)"
          >
            下载处理后图片
          </button>
        </div>
      </div>

      <!-- 右侧：结果预览 -->
      <div class="flex-1 min-w-0">
        <div class="border border-neutral-200 rounded-lg overflow-hidden">
          <!-- 空状态 -->
          <div v-if="!resultImageUrl && !processing && !processError" class="flex flex-col items-center justify-center min-h-[500px] text-neutral-400">
            <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
            <span>上传图片并点击处理按钮</span>
            <span class="text-sm mt-1">处理结果将在此显示</span>
          </div>

          <!-- 错误状态 -->
          <div v-else-if="processError && !processing" class="flex flex-col items-center justify-center min-h-[500px] px-6">
            <div class="w-14 h-14 rounded-full bg-red-50 border border-red-100 flex items-center justify-center mb-4">
              <svg class="w-7 h-7 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <p class="text-sm font-semibold text-neutral-800 mb-1">AI 暂未完成处理</p>
            <p class="text-xs text-neutral-500 mb-5">{{ processError }}</p>
            <button
              class="px-5 py-2 text-xs font-semibold bg-neutral-900 text-white rounded-lg hover:bg-neutral-800 active:scale-[0.97] transition-all flex items-center gap-1.5"
              @click="processImage"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              重新尝试
            </button>
          </div>

          <!-- 处理中状态 -->
          <div v-else-if="processing" class="flex flex-col items-center justify-center min-h-[500px] text-neutral-500">
            <svg class="animate-spin w-12 h-12 mb-4 text-neutral-400" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            <span class="font-medium">AI 正在分析并消除水印...</span>
            <span class="text-sm mt-1 text-neutral-400">通常需要 15-30 秒，请耐心等待</span>
          </div>

          <!-- 处理结果 -->
          <div v-else class="flex flex-col">
            <div class="flex items-center justify-between px-4 py-3 bg-neutral-50 border-b border-neutral-200">
              <span class="text-sm font-medium text-neutral-700">处理结果</span>
              <div class="flex items-center gap-2">
                <button
                  v-for="bg in previewBackgrounds"
                  :key="bg.value"
                  class="px-3 py-1 text-xs rounded transition-colors"
                  :class="previewBg === bg.value ? 'bg-neutral-900 text-white' : 'bg-neutral-200 text-neutral-700 hover:bg-neutral-300'"
                  @click="previewBg = bg.value"
                >{{ bg.label }}</button>
              </div>
            </div>

            <div class="flex items-center justify-center min-h-[350px] p-6" :class="previewBgClass">
              <img :src="resultImageUrl" class="max-w-full max-h-[400px] object-contain rounded" alt="处理结果" />
            </div>

            <!-- 前后对比 -->
            <div class="border-t border-neutral-200 p-4">
              <p class="text-xs text-neutral-500 text-center mb-3">处理前后对比</p>
              <div class="grid grid-cols-2 gap-4">
                <div class="flex flex-col items-center gap-2">
                  <span class="text-xs text-neutral-500">原图</span>
                  <img :src="originalImage" class="w-full aspect-square object-contain border border-neutral-200 rounded p-2" alt="原图" />
                </div>
                <div class="flex flex-col items-center gap-2">
                  <span class="text-xs text-neutral-500">消除后</span>
                  <div
                    class="w-full aspect-square border border-neutral-200 rounded p-2 flex items-center justify-center overflow-hidden"
                    :class="previewBgClass"
                  >
                    <img :src="resultImageUrl" class="max-w-full max-h-full object-contain" alt="结果" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 历史记录 -->
    <div v-if="history.length > 0" class="mt-10">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-semibold text-neutral-700">
          历史记录
          <span class="ml-1.5 text-[11px] font-normal text-neutral-400">（30 分钟内缓存，共 {{ history.length }} 条）</span>
        </h2>
        <button class="text-xs text-neutral-400 hover:text-red-500 transition-colors" @click="clearHistory">
          清空历史
        </button>
      </div>

      <div class="border border-neutral-200 rounded-xl overflow-hidden divide-y divide-neutral-100">
        <div v-for="item in history" :key="item.id">

          <!-- 列表行 -->
          <div class="flex items-center gap-4 px-4 py-3 bg-white hover:bg-neutral-50 transition-colors">
            <!-- 缩略图（点击可展开） -->
            <div
              class="flex items-center gap-2 flex-shrink-0 cursor-pointer"
              @click="item.expanded = !item.expanded"
            >
              <div class="w-14 h-14 rounded-lg overflow-hidden border border-neutral-200 bg-neutral-100">
                <img :src="item.thumbnail" alt="原图" class="w-full h-full object-cover" />
              </div>
              <svg class="w-4 h-4 text-neutral-300 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
              <div class="w-14 h-14 rounded-lg overflow-hidden border border-neutral-200 bg-neutral-100">
                <img :src="item.resultImageUrl" alt="结果" class="w-full h-full object-cover" />
              </div>
            </div>

            <!-- 信息 -->
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-neutral-800">消除文字 / 水印</p>
              <p class="text-xs text-neutral-400 mt-0.5">{{ formatTimeAgo(item.timestamp) }}</p>
            </div>

            <!-- 操作区 -->
            <div class="flex items-center gap-1.5 flex-shrink-0">
              <!-- 展开/收起 -->
              <button
                class="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-lg border transition-colors"
                :class="item.expanded
                  ? 'border-neutral-300 bg-neutral-100 text-neutral-700'
                  : 'border-neutral-200 text-neutral-500 hover:bg-neutral-50'"
                @click="item.expanded = !item.expanded"
              >
                <svg
                  class="w-3.5 h-3.5 transition-transform duration-200"
                  :class="item.expanded ? 'rotate-180' : ''"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
                {{ item.expanded ? '收起' : '查看' }}
              </button>

              <!-- 再次消除 -->
              <button
                class="w-8 h-8 flex items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700 transition-colors"
                :class="item.reErasing ? 'cursor-not-allowed opacity-50' : ''"
                :disabled="item.reErasing"
                title="再次消除"
                @click="reEraseFromHistory(item)"
              >
                <svg v-if="item.reErasing" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>

              <!-- 下载 -->
              <button
                class="w-8 h-8 flex items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700 transition-colors"
                title="下载处理结果"
                @click="downloadImage(item.resultImageUrl, `eraser-result-${item.id}.png`)"
              >
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
              </button>
            </div>
          </div>

          <!-- 展开详情面板 -->
          <Transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0"
            enter-to-class="opacity-100"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
          >
            <div v-if="item.expanded" class="px-4 pb-5 pt-3 bg-neutral-50 border-t border-neutral-100">

              <!-- 完整结果图 -->
              <p class="text-[11px] font-semibold text-neutral-500 uppercase tracking-wide mb-2">处理结果</p>
              <div class="group relative rounded-xl overflow-hidden border border-neutral-200 mb-4">
                <div class="flex items-center justify-center bg-white p-4">
                  <img :src="item.resultImageUrl" alt="处理结果" class="max-w-full max-h-[400px] object-contain rounded" />
                </div>
                <div class="absolute inset-0 bg-black/0 group-hover:bg-black/25 transition-all duration-200 flex items-center justify-center gap-3">
                  <button
                    class="opacity-0 group-hover:opacity-100 transition-opacity px-3.5 py-2 bg-white text-neutral-900 text-xs font-medium rounded-xl flex items-center gap-1.5 shadow-lg hover:bg-neutral-50"
                    :class="item.reErasing ? 'cursor-not-allowed opacity-50' : ''"
                    :disabled="item.reErasing"
                    @click="reEraseFromHistory(item)"
                  >
                    <svg v-if="item.reErasing" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    {{ item.reErasing ? '消除中...' : '再次消除' }}
                  </button>
                  <button
                    class="opacity-0 group-hover:opacity-100 transition-opacity px-3.5 py-2 bg-white text-neutral-900 text-xs font-medium rounded-xl flex items-center gap-1.5 shadow-lg hover:bg-neutral-50"
                    @click="downloadImage(item.resultImageUrl, `eraser-result-${item.id}.png`)"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    下载图片
                  </button>
                </div>
              </div>

              <!-- 原图与结果对比 -->
              <p class="text-[11px] font-semibold text-neutral-500 uppercase tracking-wide mb-2">处理前后对比</p>
              <div class="grid grid-cols-2 gap-3">
                <div class="border border-neutral-200 rounded-xl overflow-hidden bg-white">
                  <div class="px-3 py-1.5 bg-neutral-50 border-b border-neutral-100 text-center">
                    <span class="text-[11px] font-semibold text-neutral-500">原图</span>
                  </div>
                  <div class="p-3 flex items-center justify-center min-h-[200px] bg-white">
                    <img :src="item.preview || item.thumbnail" alt="原图" class="max-w-full max-h-[300px] object-contain" />
                  </div>
                </div>
                <div class="border border-neutral-200 rounded-xl overflow-hidden bg-white">
                  <div class="px-3 py-1.5 bg-neutral-50 border-b border-neutral-100 text-center">
                    <span class="text-[11px] font-semibold text-neutral-500">消除后</span>
                  </div>
                  <div class="p-3 flex items-center justify-center min-h-[200px] bg-white">
                    <img :src="item.resultImageUrl" alt="消除后" class="max-w-full max-h-[300px] object-contain" />
                  </div>
                </div>
              </div>
            </div>
          </Transition>

        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import request, { resolveImageUrl, downloadImage } from '@/utils/request'

const HISTORY_KEY = 'eraser_history'
const HISTORY_TTL = 30 * 60 * 1000
const HISTORY_MAX = 8

const originalImage = ref('')
const originalImageData = ref('')
const originalMimeType = ref('')
const processing = ref(false)
const resultImageUrl = ref('')
const uploadError = ref('')
const isDragging = ref(false)
const processError = ref('')
const previewBg = ref('white')
const fileInputRef = ref(null)

const MAX_FILE_SIZE = 10 * 1024 * 1024

const history = ref([])

const previewBackgrounds = [
  { value: 'white', label: '白底' },
  { value: 'checker', label: '棋盘' },
  { value: 'black', label: '黑底' },
]

const previewBgClass = computed(() => ({
  'bg-white': previewBg.value === 'white',
  'bg-black': previewBg.value === 'black',
  'checker-bg': previewBg.value === 'checker',
}))

/** 将图片缩放为缩略图，避免 localStorage 超限 */
function createThumbnail(dataUrl, maxSize = 120) {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => {
      const scale = Math.min(maxSize / img.width, maxSize / img.height, 1)
      const canvas = document.createElement('canvas')
      canvas.width = Math.round(img.width * scale)
      canvas.height = Math.round(img.height * scale)
      canvas.getContext('2d').drawImage(img, 0, 0, canvas.width, canvas.height)
      resolve(canvas.toDataURL('image/jpeg', 0.7))
    }
    img.onerror = () => resolve(dataUrl)
    img.src = dataUrl
  })
}

/** 创建较大尺寸的预览图，用于详情面板展示 */
function createPreview(dataUrl, maxSize = 500) {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => {
      const scale = Math.min(maxSize / img.width, maxSize / img.height, 1)
      const canvas = document.createElement('canvas')
      canvas.width = Math.round(img.width * scale)
      canvas.height = Math.round(img.height * scale)
      canvas.getContext('2d').drawImage(img, 0, 0, canvas.width, canvas.height)
      resolve(canvas.toDataURL('image/jpeg', 0.85))
    }
    img.onerror = () => resolve(dataUrl)
    img.src = dataUrl
  })
}

function loadHistory() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    const items = Array.isArray(parsed) ? parsed : (parsed.items || [])
    if (!Array.isArray(items)) {
      localStorage.removeItem(HISTORY_KEY)
      return []
    }
    /** 按每条记录自身的 timestamp 过滤，避免 savedAt 被 UI 交互重置后导致过期记录仍可见 */
    const valid = items.filter(item => Date.now() - (item.timestamp || 0) <= HISTORY_TTL)
    if (valid.length === 0) {
      localStorage.removeItem(HISTORY_KEY)
      return []
    }
    return valid.map(item => ({ expanded: false, ...item, reErasing: false }))
  } catch { return [] }
}

function saveHistory(items) {
  try {
    /** 剥离运行时 UI 状态，防止 expanded/reErasing 的变化触发 watch 并重置 savedAt */
    const toSave = items.map(({ expanded, reErasing, ...rest }) => rest)
    localStorage.setItem(HISTORY_KEY, JSON.stringify({ items: toSave, savedAt: Date.now() }))
  } catch { /** ignore quota errors */ }
}

watch(history, (val) => saveHistory(val), { deep: true })

async function addToHistory(resultUrl) {
  const thumbnail = await createThumbnail(originalImage.value)
  const preview = await createPreview(originalImage.value)
  const item = {
    id: Date.now(),
    thumbnail,
    preview,
    resultImageUrl: resultUrl,
    timestamp: Date.now(),
    expanded: false,
    reErasing: false,
  }
  history.value.unshift(item)
  if (history.value.length > HISTORY_MAX) history.value.splice(HISTORY_MAX)
}

/** 从服务端 URL 获取图片并转为 base64，用于再次消除 */
async function fetchImageAsBase64(url) {
  const resolvedUrl = resolveImageUrl(url)
  const token = localStorage.getItem('token')
  const headers = token ? { Authorization: `Bearer ${token}` } : {}

  const res = await fetch(resolvedUrl, { headers })
  if (!res.ok) throw new Error(`获取图片失败：HTTP ${res.status}`)
  const blob = await res.blob()

  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (ev) => {
      const dataUrl = ev.target.result
      resolve({
        imageData: dataUrl.split(',')[1],
        mimeType: blob.type || 'image/png',
      })
    }
    reader.onerror = reject
    reader.readAsDataURL(blob)
  })
}

/** 对历史记录中的结果图再次执行消除处理 */
async function reEraseFromHistory(item) {
  if (item.reErasing) return
  item.reErasing = true

  try {
    const { imageData, mimeType } = await fetchImageAsBase64(item.resultImageUrl)
    const res = await request.post('/eraser', {
      image_data: imageData,
      mime_type: mimeType,
    })

    if (res.data.image_url) {
      /** 更新当前历史条目的结果图 */
      item.resultImageUrl = res.data.image_url
      item.timestamp = Date.now()
      item.expanded = true
    }
  } catch (err) {
    alert('本次未能获取到结果，请稍后重试')
  } finally {
    item.reErasing = false
  }
}

function clearHistory() {
  history.value = []
  localStorage.removeItem(HISTORY_KEY)
}

function formatTimeAgo(ts) {
  const diff = Date.now() - ts
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  return `${Math.floor(mins / 60)} 小时前`
}

onMounted(() => {
  history.value = loadHistory()
})

function handleDrop(e) {
  isDragging.value = false
  processFile(e.dataTransfer?.files?.[0])
}

function handleFileChange(e) {
  processFile(e.target.files[0])
}

function processFile(file) {
  if (!file) return

  uploadError.value = ''
  processError.value = ''

  if (file.size > MAX_FILE_SIZE) {
    uploadError.value = `文件过大（${formatBytes(file.size)}），请上传 10MB 以内的图片`
    if (fileInputRef.value) fileInputRef.value.value = ''
    return
  }

  const reader = new FileReader()
  reader.onload = (ev) => {
    const dataUrl = ev.target.result
    originalImage.value = dataUrl
    originalImageData.value = dataUrl.split(',')[1]
    originalMimeType.value = dataUrl.split(';')[0].split(':')[1]
    resultImageUrl.value = ''
  }
  reader.readAsDataURL(file)
}

function clearImage() {
  originalImage.value = ''
  originalImageData.value = ''
  originalMimeType.value = ''
  resultImageUrl.value = ''
  uploadError.value = ''
  processError.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

async function processImage() {
  if (!originalImageData.value) {
    alert('请先上传图片')
    return
  }

  processing.value = true
  resultImageUrl.value = ''
  processError.value = ''

  try {
    const res = await request.post('/eraser', {
      image_data: originalImageData.value,
      mime_type: originalMimeType.value,
    })

    if (res.data.image_url) {
      resultImageUrl.value = res.data.image_url
      await addToHistory(res.data.image_url)
    }
  } catch (err) {
    processError.value = '本次未能获取到结果，请稍后重试'
  } finally {
    processing.value = false
  }
}

function formatBytes(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped>
.checker-bg {
  background-image:
    linear-gradient(45deg, #e5e5e5 25%, transparent 25%),
    linear-gradient(-45deg, #e5e5e5 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #e5e5e5 75%),
    linear-gradient(-45deg, transparent 75%, #e5e5e5 75%);
  background-size: 16px 16px;
  background-position: 0 0, 0 8px, 8px -8px, -8px 0px;
}
</style>
