<template>
  <MainLayout>
    <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-neutral-900 sm:text-3xl">多角度图片生成</h1>
          <p class="mt-2 text-sm text-neutral-500">上传一张参考图，选择多个电商视角后批量生成。</p>
        </div>
        <div class="rounded-lg border border-neutral-200 bg-white px-3 py-2 text-xs text-neutral-500">
          已选 <span class="font-semibold text-neutral-900">{{ selectedViews.length }}</span> 个角度 · {{ quality }} · 比例 {{ aspectRatio }}
        </div>
      </div>

      <div class="grid gap-6 lg:grid-cols-[340px_minmax(0,1fr)]">
        <aside class="space-y-4">
          <section class="rounded-lg border border-neutral-200 bg-white p-4">
            <div class="mb-3 flex items-center justify-between">
              <h2 class="text-sm font-semibold text-neutral-800">参考图</h2>
              <button v-if="originalImage" class="text-xs font-medium text-red-600 hover:text-red-700" @click="clearImage">清除</button>
            </div>

            <label
              v-if="!originalImage"
              class="flex min-h-[220px] cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed text-center transition"
              :class="isDragging ? 'border-neutral-900 bg-neutral-50' : 'border-neutral-300 hover:border-neutral-500'"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <PhotoIcon class="mb-3 h-10 w-10 text-neutral-400" />
              <span class="text-sm font-medium text-neutral-700">上传产品参考图</span>
              <span class="mt-1 text-xs text-neutral-400">JPG、PNG、WebP</span>
              <input type="file" accept="image/*" class="hidden" @change="handleFileChange" />
            </label>

            <div v-else class="flex aspect-square items-center justify-center overflow-hidden rounded-lg border border-neutral-200 bg-neutral-50">
              <img :src="originalImage" alt="参考图" class="h-full w-full object-contain" />
            </div>
          </section>

          <section class="rounded-lg border border-neutral-200 bg-white p-4">
            <h2 class="mb-3 text-sm font-semibold text-neutral-800">生图配置</h2>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="item in qualityOptions"
                :key="item"
                type="button"
                class="h-9 rounded-lg border text-sm font-medium transition"
                :class="quality === item ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-400'"
                @click="quality = item"
              >
                {{ item }}
              </button>
            </div>
            <div class="mt-4 grid grid-cols-4 gap-2">
              <button
                v-for="item in aspectRatioOptions"
                :key="item"
                type="button"
                class="h-9 rounded-lg border text-sm font-medium transition"
                :class="aspectRatio === item ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-200 bg-white text-neutral-700 hover:border-neutral-400'"
                @click="aspectRatio = item"
              >
                {{ item }}
              </button>
            </div>
          </section>

          <button
            class="flex h-11 w-full items-center justify-center gap-2 rounded-lg text-sm font-semibold text-white transition"
            :class="canGenerate ? 'bg-neutral-900 hover:bg-neutral-800' : 'bg-neutral-300'"
            :disabled="!canGenerate"
            @click="startGenerate"
          >
            <ArrowPathIcon v-if="generating" class="h-4 w-4 animate-spin" />
            <SparklesIcon v-else class="h-4 w-4" />
            {{ generating ? '批量生成中...' : `生成 ${selectedViews.length} 张` }}
          </button>
        </aside>

        <main class="space-y-4">
          <section class="rounded-lg border border-neutral-200 bg-white p-4">
            <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 class="text-sm font-semibold text-neutral-800">角度任务</h2>
                <p class="mt-1 text-xs text-neutral-400">选择一个或多个视角，生成后会按角度展示结果。</p>
              </div>
              <div class="flex gap-2">
                <button class="h-8 rounded-md border border-neutral-200 px-3 text-xs font-medium text-neutral-700 hover:bg-neutral-50" @click="selectAllViews">全选</button>
                <button class="h-8 rounded-md border border-neutral-200 px-3 text-xs font-medium text-neutral-700 hover:bg-neutral-50" @click="clearViews">清空</button>
              </div>
            </div>

            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <button
                v-for="item in viewOptions"
                :key="item.value"
                type="button"
                class="group rounded-lg border p-3 text-left transition"
                :class="selectedViews.includes(item.value) ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-200 bg-white hover:border-neutral-400'"
                @click="toggleView(item.value)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="text-sm font-semibold">{{ item.label }}</div>
                    <div class="mt-1 text-xs" :class="selectedViews.includes(item.value) ? 'text-neutral-300' : 'text-neutral-400'">{{ item.desc }}</div>
                  </div>
                  <div
                    class="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full border text-[10px]"
                    :class="selectedViews.includes(item.value) ? 'border-white bg-white text-neutral-900' : 'border-neutral-300 text-transparent'"
                  >
                    ✓
                  </div>
                </div>
              </button>
            </div>
          </section>

          <section class="rounded-lg border border-neutral-200 bg-white p-4">
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-sm font-semibold text-neutral-800">生成结果</h2>
              <button
                v-if="successfulResults.length"
                class="flex h-8 items-center gap-1 rounded-md border border-neutral-200 px-3 text-xs font-medium text-neutral-700 hover:bg-neutral-50"
                @click="downloadFirstImage"
              >
                <ArrowDownTrayIcon class="h-4 w-4" />
                下载首图
              </button>
            </div>

            <div v-if="generating" class="flex min-h-[320px] flex-col items-center justify-center rounded-lg bg-neutral-50">
              <ArrowPathIcon class="h-12 w-12 animate-spin text-neutral-500" />
              <span class="mt-3 text-sm text-neutral-600">{{ generationMessage || '正在按角度批量生成' }}</span>
              <span class="mt-1 text-xs text-neutral-400">{{ selectedViews.length }} 个角度会依次完成</span>
              <button
                type="button"
                class="mt-4 flex h-9 items-center gap-1 rounded-md border border-neutral-200 bg-white px-3 text-xs font-medium text-neutral-700 hover:bg-neutral-50"
                @click="stopWaiting"
              >
                <StopCircleIcon class="h-4 w-4" />
                停止等待
              </button>
            </div>

            <div v-else-if="generationError" class="rounded-lg border border-red-100 bg-red-50 p-4">
              <div class="text-sm font-medium text-red-700">生成没有完成</div>
              <div class="mt-1 text-xs leading-5 text-red-600">{{ generationError }}</div>
              <button
                v-if="canGenerate"
                type="button"
                class="mt-3 flex h-9 items-center gap-1 rounded-md bg-neutral-900 px-3 text-xs font-semibold text-white hover:bg-neutral-800"
                @click="startGenerate"
              >
                <SparklesIcon class="h-4 w-4" />
                重新生成
              </button>
            </div>

            <div v-else-if="!generatedResults.length" class="grid gap-3 rounded-lg bg-neutral-50 p-4 sm:grid-cols-2 xl:grid-cols-3">
              <div
                v-for="view in selectedViews"
                :key="view"
                class="flex aspect-square flex-col items-center justify-center rounded-lg border border-dashed border-neutral-300 bg-white text-neutral-400"
              >
                <PhotoIcon class="h-8 w-8" />
                <span class="mt-2 text-sm">{{ labelOf(view) }}</span>
                <span class="mt-1 text-xs">待生成</span>
              </div>
            </div>

            <div v-else class="grid gap-4 rounded-lg bg-neutral-50 p-4 sm:grid-cols-2 xl:grid-cols-3">
              <div v-for="item in generatedResults" :key="item.view" class="overflow-hidden rounded-lg border border-neutral-200 bg-white">
                <div class="flex items-center justify-between border-b border-neutral-100 px-3 py-2">
                  <span class="text-sm font-medium text-neutral-800">{{ labelOf(item.view) }}</span>
                  <span class="text-xs" :class="item.succeeded ? 'text-emerald-600' : (item.status === 'failed' ? 'text-red-500' : 'text-neutral-400')">{{ item.succeeded ? '完成' : (item.status === 'failed' ? '失败' : '生成中') }}</span>
                </div>
                <button
                  v-if="item.image_url"
                  type="button"
                  class="flex aspect-square w-full items-center justify-center bg-neutral-100"
                  @click="largePreviewUrl = resolveImageUrl(item.image_url)"
                >
                  <img :src="resolveImageUrl(item.image_url)" alt="生成结果" class="h-full w-full object-contain" />
                </button>
                <div v-else class="flex aspect-square items-center justify-center p-4 text-center text-xs" :class="item.status === 'failed' ? 'text-red-500' : 'text-neutral-400'">{{ item.status === 'failed' ? (item.error || '生成失败') : '生成中' }}</div>
                <button
                  v-if="item.image_url"
                  class="flex h-9 w-full items-center justify-center gap-1 border-t border-neutral-100 text-xs font-medium text-neutral-700 hover:bg-neutral-50"
                  @click="downloadResult(item)"
                >
                  <ArrowDownTrayIcon class="h-4 w-4" />
                  下载
                </button>
              </div>
            </div>
          </section>
        </main>
      </div>

      <ImagePreview
        :visible="!!largePreviewUrl"
        :image-url="largePreviewUrl"
        alt="生成结果大图"
        @close="largePreviewUrl = ''"
      />
    </div>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ArrowDownTrayIcon, ArrowPathIcon, PhotoIcon, SparklesIcon, StopCircleIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import request, { downloadImage as downloadFile, resolveImageUrl } from '@/utils/request'

const originalImage = ref('')
const originalImageData = ref('')
const originalMimeType = ref('')
const isDragging = ref(false)
const generating = ref(false)
const generatedResults = ref([])
const largePreviewUrl = ref('')
const selectedViews = ref(['front', 'left45', 'right45'])
const quality = ref('2K')
const aspectRatio = ref('1:1')
const activeTaskId = ref('')
const generationMessage = ref('')
const generationError = ref('')
const pollToken = ref(0)

const qualityOptions = ['1K', '2K', '4K']
const aspectRatioOptions = ['1:1', '4:3', '3:4', '16:9', '9:16', '21:9', '3:2', '2:3', '4:5', '5:4']
const viewOptions = [
  { value: 'front', label: '正面视角', desc: '主图正面，适合封面' },
  { value: 'left45', label: '左侧45度视角', desc: '左前方，展示立体感' },
  { value: 'left', label: '左侧视角', desc: '展示左侧结构' },
  { value: 'right45', label: '右侧45度视角', desc: '右前方，电商常用' },
  { value: 'right', label: '右侧视角', desc: '展示右侧细节' },
  { value: 'back', label: '背面视角', desc: '展示背部信息' },
  { value: 'top45', label: '俯拍45度视角', desc: '展示顶部和正面' },
  { value: 'top', label: '顶视图', desc: '正上方俯拍' },
  { value: 'low', label: '低角度视角', desc: '强化体量和质感' },
  { value: 'closeup', label: '细节特写', desc: '材质、标签、边缘' },
]

const canGenerate = computed(() => !!originalImageData.value && selectedViews.value.length > 0 && !generating.value)
const successfulResults = computed(() => generatedResults.value.filter(item => item.image_url))
const pollingStoppedMessage = '已停止等待旧任务'

onMounted(() => {
  restoreLatestMultiAngleTask()
})

function labelOf(view) {
  return viewOptions.find(item => item.value === view)?.label || view
}

function toggleView(view) {
  if (selectedViews.value.includes(view)) {
    if (selectedViews.value.length === 1) return
    selectedViews.value = selectedViews.value.filter(item => item !== view)
    generatedResults.value = []
    generationError.value = ''
    return
  }
  selectedViews.value = [...selectedViews.value, view]
  generatedResults.value = []
  generationError.value = ''
}

function selectAllViews() {
  selectedViews.value = viewOptions.map(item => item.value)
  generatedResults.value = []
  generationError.value = ''
}

function clearViews() {
  selectedViews.value = [viewOptions[0].value]
  generatedResults.value = []
  generationError.value = ''
}

function loadFile(file) {
  if (!file) return
  const reader = new FileReader()
  reader.onload = event => {
    const dataUrl = event.target?.result
    if (typeof dataUrl !== 'string') return
    originalImage.value = dataUrl
    originalImageData.value = dataUrl.split(',')[1] || ''
    originalMimeType.value = dataUrl.split(';')[0]?.split(':')[1] || file.type || 'image/png'
    generatedResults.value = []
    generationError.value = ''
  }
  reader.readAsDataURL(file)
}

function handleDrop(event) {
  isDragging.value = false
  loadFile(event.dataTransfer?.files?.[0])
}

function handleFileChange(event) {
  loadFile(event.target.files?.[0])
  event.target.value = ''
}

function clearImage() {
  originalImage.value = ''
  originalImageData.value = ''
  originalMimeType.value = ''
  generatedResults.value = []
  activeTaskId.value = ''
  generationError.value = ''
  generationMessage.value = ''
  pollToken.value += 1
  generating.value = false
}

async function restoreLatestMultiAngleTask() {
  try {
    const res = await request.get('/multi-angle/tasks')
    const latest = Array.isArray(res.data?.items) ? res.data.items[0] : null
    if (!latest) return

    activeTaskId.value = latest.task_id || ''
    quality.value = latest.quality || quality.value
    if (aspectRatioOptions.includes(latest.aspect_ratio)) {
      aspectRatio.value = latest.aspect_ratio
    }
    generatedResults.value = normalizeTaskResults(latest)
    generationError.value = latest.status === 'failed' ? (latest.error_message || '任务生成失败，请重新生成') : ''

    if (activeTaskId.value && ['queued', 'running'].includes(latest.status)) {
      const token = nextPollToken()
      generating.value = true
      generationMessage.value = '正在恢复上次未完成的任务'
      try {
        await pollMultiAngleTask(activeTaskId.value, token)
      } catch (error) {
        if (error.message !== pollingStoppedMessage) {
          console.error('恢复多角度任务失败', error)
          generationError.value = getErrorMessage(error, '上次任务没有完成，请重新生成')
        }
      } finally {
        if (pollToken.value === token) {
          generating.value = false
          generationMessage.value = ''
        }
      }
    }
  } catch (error) {
    console.error('恢复多角度任务失败', error)
    generating.value = false
    generationMessage.value = ''
  }
}

async function startGenerate() {
  if (!canGenerate.value) return
  const token = nextPollToken()
  generating.value = true
  generatedResults.value = []
  activeTaskId.value = ''
  generationError.value = ''
  generationMessage.value = '正在按角度批量生成'

  try {
    const res = await request.post('/multi-angle/tasks', {
      image_data: originalImageData.value,
      mime_type: originalMimeType.value,
      views: selectedViews.value,
      quality: quality.value,
      aspect_ratio: aspectRatio.value,
    })
    activeTaskId.value = res.data?.task_id || ''
    if (!activeTaskId.value) throw new Error('未获取到任务ID')
    await pollMultiAngleTask(activeTaskId.value, token)
  } catch (error) {
    if (error.message !== pollingStoppedMessage) {
      console.error('多角度生成失败', error)
      generationError.value = getErrorMessage(error, '未知错误')
      alert(`生成失败: ${generationError.value}`)
    }
  } finally {
    if (pollToken.value === token) {
      generating.value = false
      generationMessage.value = ''
    }
  }
}

function normalizeTaskResults(task) {
  const results = Array.isArray(task?.results) ? task.results : []
  return results.map(item => ({
    view: item.view,
    label: item.label,
    status: item.status,
    image_url: item.image_url,
    mime_type: item.mime_type || 'image/png',
    error: item.error,
    succeeded: !!item.succeeded || !!item.image_url,
  }))
}

async function pollMultiAngleTask(taskId, token) {
  const deadline = Date.now() + 30 * 60 * 1000
  while (Date.now() < deadline) {
    if (pollToken.value !== token) throw new Error(pollingStoppedMessage)
    const res = await request.get(`/multi-angle/tasks/${taskId}`)
    const task = res.data || {}
    generatedResults.value = normalizeTaskResults(task)

    if (['done', 'partial'].includes(task.status)) {
      if (!generatedResults.value.some(item => item.image_url)) {
        throw new Error(task.error_message || '未获取到图片链接')
      }
      return
    }

    if (task.status === 'failed') {
      throw new Error(task.error_message || '任务生成失败')
    }

    await new Promise(resolve => setTimeout(resolve, 2000))
  }
  throw new Error('任务轮询超时')
}

function nextPollToken() {
  pollToken.value += 1
  return pollToken.value
}

function stopWaiting() {
  pollToken.value += 1
  generating.value = false
  generationMessage.value = ''
  activeTaskId.value = ''
  generationError.value = '已停止等待旧任务，可以重新发起生成。'
}

function getErrorMessage(error, fallback) {
  return error?.response?.data?.error || error?.response?.data?.details || error?.message || fallback
}

function downloadResult(item) {
  if (!item?.image_url) return
  downloadFile(resolveImageUrl(item.image_url), `multi-angle-${item.view}-${quality.value}.png`)
}

function downloadFirstImage() {
  downloadResult(successfulResults.value[0])
}
</script>
