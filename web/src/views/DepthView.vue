<template>
  <MainLayout>
    <section class="mx-2 grid h-[calc(100vh-92px)] grid-cols-[minmax(320px,380px)_minmax(0,1fr)] gap-4 py-4 max-lg:h-auto max-lg:grid-cols-1">
      <aside class="sticky top-[18px] flex max-h-[calc(100vh-110px)] flex-col overflow-hidden rounded-[24px] border border-neutral-200 bg-white shadow-[0_18px_50px_rgba(15,23,42,0.05)] max-lg:static max-lg:max-h-none">
        <div class="border-b border-neutral-100 px-5 py-4">
          <p class="text-xs font-medium uppercase tracking-[0.16em] text-neutral-400">Depth</p>
          <h1 class="mt-1 text-lg font-semibold text-neutral-900">景深图生成</h1>
        </div>

        <div class="flex-1 space-y-4 overflow-y-auto p-5">
          <div class="rounded-xl border border-neutral-200 bg-white p-4">
            <div class="mb-3 flex items-center justify-between">
              <label class="text-sm font-semibold text-neutral-900">上传图片</label>
              <span class="rounded-full bg-neutral-100 px-2.5 py-1 text-xs font-medium text-neutral-600">{{ images.length }}/6</span>
            </div>
            <label
              v-if="images.length < 6"
              class="flex min-h-[132px] cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed border-neutral-300 bg-neutral-50 px-4 text-center transition hover:border-neutral-500 hover:bg-white"
              @dragover.prevent
              @drop.prevent="handleDrop"
            >
              <PhotoIcon class="h-8 w-8 text-neutral-400" />
              <span class="mt-3 text-sm font-medium text-neutral-800">点击或拖入图片</span>
              <span class="mt-1 text-xs text-neutral-400">最多 6 张</span>
              <input type="file" accept="image/*" multiple class="hidden" @change="handleFileChange" />
            </label>

            <div v-if="images.length" class="mt-4 grid grid-cols-3 gap-2">
              <div v-for="(img, index) in images" :key="img.id" class="group relative aspect-square overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50">
                <img :src="img.url" alt="参考图" class="h-full w-full object-cover" />
                <button
                  class="absolute right-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-full bg-white/95 text-neutral-700 opacity-0 shadow transition group-hover:opacity-100"
                  type="button"
                  @click="removeImage(index)"
                >
                  <XMarkIcon class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </div>

          <div class="rounded-xl border border-neutral-200 bg-white p-4">
            <label class="mb-3 block text-sm font-semibold text-neutral-900">生图比例</label>
            <div class="grid grid-cols-3 gap-2">
              <button
                v-for="ratio in aspectRatios"
                :key="ratio"
                class="rounded-xl border px-3 py-2 text-sm font-medium transition"
                :class="aspectRatio === ratio ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-200 text-neutral-600 hover:border-neutral-400'"
                type="button"
                @click="aspectRatio = ratio"
              >
                {{ ratio }}
              </button>
            </div>
          </div>

          <div class="rounded-xl border border-neutral-200 bg-white p-4">
            <label class="mb-3 block text-sm font-semibold text-neutral-900">画质</label>
            <div class="grid grid-cols-2 gap-2">
              <button
                v-for="q in qualities"
                :key="q"
                class="rounded-xl border px-3 py-2 text-sm font-medium transition"
                :class="quality === q ? 'border-neutral-900 bg-neutral-900 text-white' : 'border-neutral-200 text-neutral-600 hover:border-neutral-400'"
                type="button"
                @click="quality = q"
              >
                {{ q }}
              </button>
            </div>
          </div>
        </div>

        <div class="border-t border-neutral-100 bg-white p-5">
          <button
            class="flex w-full items-center justify-center gap-2 rounded-xl py-3 text-sm font-semibold text-white transition"
            :class="canGenerate ? 'bg-neutral-900 hover:bg-neutral-800' : 'cursor-not-allowed bg-neutral-300'"
            :disabled="!canGenerate"
            type="button"
            @click="createTask"
          >
            <SparklesIcon class="h-4 w-4" />
            {{ isGenerating ? '生成中...' : `生成 ${images.length || 0} 张` }}
          </button>
          <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>
        </div>
      </aside>

      <main class="flex min-w-0 flex-col overflow-hidden rounded-[24px] border border-neutral-200 bg-white shadow-[0_18px_50px_rgba(15,23,42,0.05)]">
        <div class="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <div>
            <h2 class="text-lg font-semibold text-neutral-900">图片展览区</h2>
            <p class="mt-1 text-xs text-neutral-400">生成完成后会持久化保存，并写入使用记录。</p>
          </div>
          <button class="rounded-xl border border-neutral-200 px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-50" type="button" @click="loadHistory">刷新</button>
        </div>

        <div class="flex-1 overflow-auto p-5">
          <div v-if="displayItems.length" class="grid grid-cols-3 gap-4 max-xl:grid-cols-2 max-md:grid-cols-1">
            <div v-for="item in displayItems" :key="item.key" class="group overflow-hidden rounded-2xl border border-neutral-200 bg-neutral-50">
              <div class="relative aspect-[4/3] overflow-hidden bg-neutral-100">
                <img v-if="item.imageUrl" :src="item.imageUrl" alt="景深图" class="h-full w-full object-cover" />
                <div v-else class="flex h-full w-full flex-col items-center justify-center gap-3 text-neutral-400">
                  <div class="h-9 w-9 animate-spin rounded-full border-2 border-neutral-200 border-t-neutral-900" />
                  <span class="text-sm">{{ item.status === 'error' ? item.error || '生成失败' : '生成中' }}</span>
                </div>
                <div v-if="item.imageUrl" class="pointer-events-none absolute inset-0 flex items-center justify-center gap-2 bg-black/0 opacity-0 transition group-hover:pointer-events-auto group-hover:bg-black/40 group-hover:opacity-100">
                  <button class="rounded-full bg-white px-4 py-2 text-sm font-semibold text-neutral-900 shadow" type="button" @click="openViewer(item)">查看</button>
                  <button class="rounded-full bg-white px-4 py-2 text-sm font-semibold text-neutral-900 shadow" type="button" @click="downloadImage(item.imageUrl, `depth-${Date.now()}.png`)">下载</button>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="flex min-h-[420px] items-center justify-center rounded-2xl border border-dashed border-neutral-300 bg-neutral-50 text-neutral-500">
            上传图片后开始生成，结果会展示在这里。
          </div>
        </div>
      </main>
    </section>

    <ImagePreview
      :visible="!!viewerItem"
      :imageUrl="viewerItem?.imageUrl || ''"
      alt="景深图"
      @close="viewerItem = null"
    />
  </MainLayout>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { PhotoIcon, SparklesIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import request, { resolveImageUrl, downloadImage } from '@/utils/request'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const images = ref([])
const aspectRatio = ref('1:1')
const quality = ref('2K')
const error = ref('')
const isGenerating = ref(false)
const currentItems = ref([])
const historyItems = ref([])
const activeTaskId = ref('')
const viewerItem = ref(null)
let pollTimer = null

const aspectRatios = ['1:1', '4:3', '3:4', '16:9', '9:16', '21:9']
const qualities = ['2K', '4K']
const canGenerate = computed(() => images.value.length > 0 && !isGenerating.value)
const displayItems = computed(() => currentItems.value.length ? currentItems.value : historyItems.value)

function loadFiles(fileList) {
  const files = Array.from(fileList || []).filter(file => file.type.startsWith('image/'))
  const slots = 6 - images.value.length
  files.slice(0, slots).forEach(file => {
    const reader = new FileReader()
    reader.onload = event => {
      const dataUrl = String(event.target?.result || '')
      images.value.push({
        id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
        data: dataUrl.split(',')[1] || '',
        mimeType: file.type || 'image/png',
        url: dataUrl,
      })
    }
    reader.readAsDataURL(file)
  })
}

function handleFileChange(event) {
  loadFiles(event.target.files)
  event.target.value = ''
}

function handleDrop(event) {
  loadFiles(event.dataTransfer?.files)
}

function removeImage(index) {
  images.value.splice(index, 1)
}

async function createTask() {
  if (!canGenerate.value) return
  error.value = ''
  isGenerating.value = true
  currentItems.value = images.value.map((_, index) => ({
    key: `current-${Date.now()}-${index}`,
    imageUrl: '',
    status: 'generating',
    error: '',
  }))

  try {
    const res = await request.post('/depth/tasks', {
      images: images.value.map(item => ({ data: item.data, mime_type: item.mimeType })),
      aspect_ratio: aspectRatio.value,
      quality: quality.value,
    })
    activeTaskId.value = res.data?.task_id || ''
    if (!activeTaskId.value) throw new Error('未获取到任务ID')
    startPolling()
  } catch (err) {
    error.value = err?.response?.data?.error || '提交任务失败'
    isGenerating.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(fetchTask, 2000)
  fetchTask()
}

function stopPolling() {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
}

async function fetchTask() {
  if (!activeTaskId.value) return
  try {
    const res = await request.get(`/depth/tasks/${activeTaskId.value}`)
    const data = res.data || {}
    currentItems.value = (data.images || []).map((item, index) => ({
      key: `current-${activeTaskId.value}-${index}`,
      imageUrl: resolveImageUrl(item.image_url || ''),
      status: item.status || 'pending',
      error: item.error || '',
    }))
    const done = currentItems.value.every(item => item.status === 'done' || item.status === 'error')
    if (done || data.status === 'done' || data.status === 'failed') {
      stopPolling()
      isGenerating.value = false
      toast.success('景深图生成完成')
      loadHistory()
    }
  } catch {
    // keep polling
  }
}

async function loadHistory() {
  try {
    const res = await request.get('/depth/tasks', { params: { page: 1, page_size: 8 } })
    const tasks = res.data?.tasks || []
    historyItems.value = tasks.flatMap(task => {
      const taskImages = Array.isArray(task.images) ? task.images : []
      return taskImages
        .filter(item => item.image_url)
        .map((item, index) => ({
          key: `history-${task.task_id}-${item.sort ?? index}`,
          imageUrl: resolveImageUrl(item.image_url),
          status: 'done',
          error: '',
        }))
    })
  } catch {
    // ignore
  }
}

function openViewer(item) {
  viewerItem.value = item
}

onMounted(loadHistory)
onBeforeUnmount(stopPolling)
</script>
