<template>
  <MainLayout>
    <div class="w-full px-2 py-8 sm:px-3 lg:px-4 xl:px-5">
      <!-- <div
        class="mb-6 rounded-[28px] border border-neutral-200 bg-[linear-gradient(135deg,#fafaf9_0%,#f5f5f4_60%,#eef2ff_100%)] px-6 py-6 sm:px-8">
        <div class="max-w-4xl">
          <div
            class="inline-flex items-center gap-2 rounded-full border border-neutral-200 bg-white/80 px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] text-neutral-500">
            Internal Workspace</div>
          <h1 class="mt-4 text-3xl font-semibold tracking-tight text-neutral-900 sm:text-[40px]">图片转提示词工作台</h1>
          <p class="mt-3 max-w-3xl text-sm leading-7 text-neutral-600 sm:text-base">
            先把图片解析成可直接使用的提示词；如果用户还想继续验证效果，再展开生图参数。默认只服务“我要提示词”这个核心任务。</p>
        </div>
      </div> -->

      <div class="flex flex-col gap-5 xl:flex-row xl:items-start">
        <div class="w-full xl:w-[24rem] xl:flex-shrink-0 space-y-4">
          <section class="rounded-[24px] border border-neutral-200 bg-white shadow-[0_10px_30px_rgba(0,0,0,0.04)]">
            <div class="border-b border-neutral-100 px-5 py-4">
              <p class="text-xs font-medium uppercase tracking-[0.16em] text-neutral-400">Reference</p>
              <h2 class="mt-1 text-base font-semibold text-neutral-900">参考图输入</h2>
            </div>
            <div class="p-5">
              <div v-if="!previewUrl">
                <label
                  class="group flex min-h-[280px] w-full cursor-pointer flex-col items-center justify-center rounded-[24px] border-2 border-dashed px-6 text-center transition-all duration-200"
                  :class="isDragging ? 'border-neutral-900 bg-neutral-50 shadow-inner' : 'border-neutral-300 bg-[linear-gradient(180deg,#fafaf9_0%,#fff_100%)] hover:border-neutral-500 hover:bg-neutral-50'"
                  @dragover.prevent="isDragging = true" @dragleave.prevent="isDragging = false"
                  @drop.prevent="handleDrop">
                  <svg class="mb-3 h-10 w-10" :class="isDragging ? 'text-neutral-800' : 'text-neutral-400'" fill="none"
                    stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                  </svg>
                  <span class="text-sm font-medium" :class="isDragging ? 'text-neutral-800' : 'text-neutral-700'">{{
                    isDragging ? '释放文件，开始解析' : '拖入参考图或点击上传' }}</span>
                  <span class="mt-1 text-xs text-neutral-400">返回结果仅保留提示词内容，方便直接复制使用</span>
                  <input type="file" accept="image/*" class="hidden" @change="handleFileChange" />
                </label>
              </div>
              <div v-else class="space-y-4">
                <div
                  class="overflow-hidden rounded-[24px] border border-neutral-200 bg-[linear-gradient(180deg,#fafaf9_0%,#fff_100%)]">
                  <img :src="previewUrl" alt="参考图" class="h-[240px] w-full object-contain bg-white" /></div>
                <div class="grid grid-cols-2 gap-3">
                  <button
                    class="rounded-2xl border border-neutral-200 px-3 py-3 text-sm font-medium text-neutral-700 transition-colors hover:bg-neutral-50"
                    @click="clearImage">重新上传</button>
                  <button
                    class="rounded-2xl border border-red-200 px-3 py-3 text-sm font-medium text-red-500 transition-colors hover:bg-red-50"
                    @click="clearImage">移除</button>
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-[24px] border border-neutral-200 bg-white p-5 shadow-[0_10px_30px_rgba(0,0,0,0.04)]">
            <p class="text-xs font-medium uppercase tracking-[0.16em] text-neutral-400">Parse</p>
            <h2 class="mt-1 text-base font-semibold text-neutral-900">提示词解析</h2>
            <div class="mt-4 space-y-4">
              <button type="button"
                class="flex w-full items-center justify-center gap-2 rounded-2xl py-3.5 text-sm font-semibold text-white transition-all"
                :class="canSubmit ? 'bg-neutral-900 hover:bg-neutral-800 shadow-sm' : 'bg-neutral-300 cursor-not-allowed'"
                :disabled="!canSubmit" @click="generatePrompt">{{ loading ? '解析中…' : '生成提示词' }}</button>
              <p
                class="rounded-2xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-xs leading-6 text-neutral-500">
                生成后会直接写入右侧工作区。默认只需要复制提示词即可；需要生图时再展开下一步。</p>
            </div>
          </section>
          <section v-if="history.length > 0"
            class="rounded-[24px] border border-neutral-200 bg-white p-5 shadow-[0_10px_30px_rgba(0,0,0,0.04)]">
            <div class="mb-4 flex items-center justify-between">
              <div>
                <p class="text-xs font-medium uppercase tracking-[0.16em] text-neutral-400">Recent</p>
                <h2 class="mt-1 text-base font-semibold text-neutral-900">最近解析</h2>
              </div>
              <button class="text-xs text-neutral-400 transition-colors hover:text-red-500"
                @click="clearHistory">清空</button>
            </div>
            <div class="grid grid-cols-1 gap-3">
              <button v-for="item in history.slice(0, 5)" :key="item.id" type="button"
                class="group flex w-full items-start gap-3 rounded-2xl border border-neutral-100 bg-neutral-50/70 px-3 py-3 text-left transition-all hover:border-neutral-300 hover:bg-white hover:shadow-sm"
                @click="reuseHistoryPrompt(item)">
                <img :src="item.imageUrl" alt="缩略图"
                  class="h-14 w-14 rounded-xl border border-neutral-200 object-cover" />
                <div class="min-w-0 flex-1">
                  <p class="line-clamp-2 text-sm leading-6 text-neutral-800">{{ item.prompt }}</p>
                  <p class="mt-1 text-xs text-neutral-400">{{ formatTimeAgo(item.timestamp) }}</p>
                </div>
                <span
                  class="rounded-xl border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-600 transition-colors group-hover:border-neutral-300 group-hover:text-neutral-900">使用</span>
              </button>
            </div>
          </section>
        </div>

        <div class="min-w-0 flex-1 space-y-5">
          <div
            class="overflow-hidden rounded-[24px] border border-neutral-200 bg-white shadow-[0_12px_32px_rgba(0,0,0,0.05)]">
            <div class="flex flex-col gap-4 border-b border-neutral-100 px-6 py-5 lg:flex-row lg:items-start lg:justify-between">
              <div>
                <p class="text-xs font-medium uppercase tracking-[0.16em] text-neutral-400">Workspace</p>
                <h2 class="mt-1 text-lg font-semibold text-neutral-900">提示词工作区</h2>
                <p class="mt-1 text-sm text-neutral-500">先拿到提示词，再决定是否继续生图验证。</p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button type="button"
                  class="rounded-xl border border-neutral-300 px-3 py-2 text-sm text-neutral-800 transition-colors hover:bg-neutral-50 disabled:opacity-40"
                  :disabled="!promptText.trim()" @click="copyPrompt">复制提示词</button>
                <button type="button"
                  class="rounded-xl px-3 py-2 text-sm font-medium transition-colors disabled:opacity-40"
                  :class="showGenerator ? 'bg-neutral-900 text-white hover:bg-neutral-800' : 'border border-neutral-300 text-neutral-800 hover:bg-neutral-50'"
                  :disabled="!promptText.trim()" @click="toggleGenerator">{{ showGenerator ? '收起生图配置' : '用该提示词去生图' }}</button>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-0 xl:grid-cols-[minmax(0,1fr)_320px]" :class="showGenerator ? '' : 'xl:grid-cols-1'">
              <div class="p-6" :class="showGenerator ? 'xl:border-r xl:border-neutral-100' : ''">
                <textarea v-model="promptText" placeholder="模型返回的提示词会写入这里。可继续追加品牌、材质、光源、构图或镜头约束。"
                  class="w-full resize-none rounded-[22px] border border-neutral-200 bg-[linear-gradient(180deg,#fff_0%,#fafaf9_100%)] px-5 py-5 text-sm leading-7 text-neutral-800 outline-none transition focus:border-neutral-300 focus:ring-4 focus:ring-neutral-900/5"
                  :class="showGenerator ? 'min-h-[520px]' : 'min-h-[560px]'" />
              </div>

              <Transition name="fade-slide">
                <div v-if="showGenerator"
                  class="border-t border-neutral-100 bg-neutral-50/70 p-6 xl:border-t-0">
                  <div class="rounded-[22px] border border-neutral-200 bg-white p-5">
                    <p class="text-xs font-medium uppercase tracking-[0.14em] text-neutral-400">Generate</p>
                    <h3 class="mt-2 text-base font-semibold text-neutral-900">生图配置</h3>
                    <p class="mt-2 text-sm leading-6 text-neutral-500">在当前页面直接验证提示词效果。</p>

                    <div class="mt-5 space-y-5">
                      <div>
                        <label class="mb-2 block text-xs font-medium uppercase tracking-[0.12em] text-neutral-400">比例</label>
                        <div class="grid grid-cols-3 gap-2">
                          <button v-for="ratio in aspectRatios" :key="ratio" type="button"
                            class="rounded-2xl border px-3 py-2.5 text-xs font-medium transition-all"
                            :class="aspectRatio === ratio ? 'border-neutral-900 bg-neutral-900 text-white shadow-sm' : 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-400'"
                            @click="aspectRatio = ratio">{{ ratio }}</button>
                        </div>
                      </div>
                      <div>
                        <label class="mb-2 block text-xs font-medium uppercase tracking-[0.12em] text-neutral-400">批次</label>
                        <div class="grid grid-cols-4 gap-2">
                          <button v-for="n in 4" :key="n" type="button"
                            class="h-10 rounded-2xl border text-sm font-medium transition-all"
                            :class="count === n ? 'border-neutral-900 bg-neutral-900 text-white shadow-sm' : 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-400'"
                            @click="count = n">{{ n }}</button>
                        </div>
                      </div>
                      <div v-if="supportedResolutions.length > 0">
                        <label class="mb-2 block text-xs font-medium uppercase tracking-[0.12em] text-neutral-400">分辨率</label>
                        <div class="flex flex-wrap gap-2">
                          <button v-for="r in supportedResolutions" :key="r" type="button"
                            class="rounded-2xl border px-3 py-2 text-sm font-medium transition-all"
                            :class="resolution === r ? 'border-neutral-900 bg-neutral-900 text-white shadow-sm' : 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-400'"
                            @click="resolution = r">{{ r }}</button>
                        </div>
                      </div>
                      <button type="button"
                        class="flex w-full items-center justify-center gap-2 rounded-[20px] py-3.5 text-sm font-semibold text-white transition-all"
                        :class="canGenerateImage ? 'bg-neutral-900 hover:bg-neutral-800 shadow-sm' : 'bg-neutral-300 cursor-not-allowed'"
                        :disabled="!canGenerateImage" @click="generateImages">{{ isGeneratingImages ? `${batchProgress.current}/${batchProgress.total} 执行中` : `提交绘图任务（${count} 张）` }}</button>
                      <div v-if="imageError"
                        class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600">{{ imageError }}</div>
                    </div>
                  </div>
                </div>
              </Transition>
            </div>
          </div>

          <section v-if="showGenerator || results.length > 0 || isGeneratingImages"
            class="rounded-[24px] border border-neutral-200 bg-white shadow-[0_12px_32px_rgba(0,0,0,0.05)]">
            <div class="flex items-center justify-between border-b border-neutral-100 px-6 py-5">
              <div>
                <p class="text-xs font-medium uppercase tracking-[0.16em] text-neutral-400">Results</p>
                <h2 class="mt-1 text-lg font-semibold text-neutral-900">生成结果</h2>
                <p class="mt-1 text-sm text-neutral-500">悬停图片后可直接查看大图或下载。</p>
              </div>
              <div class="text-xs text-neutral-400">{{ results.length }} 项</div>
            </div>

            <div class="p-6">
              <div v-if="results.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                <div v-for="(item, index) in results" :key="index"
                  class="group overflow-hidden rounded-[22px] border border-neutral-200 bg-white shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-[0_14px_32px_rgba(0,0,0,0.08)]">
                  <div class="relative aspect-square bg-neutral-100">
                    <div v-if="item.status !== 'done'"
                      class="flex h-full items-center justify-center p-6 text-center text-sm"
                      :class="item.status === 'error' ? 'text-red-500' : 'text-neutral-400'">{{ item.status === 'generating' ? '绘图任务执行中…' : item.status === 'error' ? item.error : '等待调度' }}</div>
                    <template v-else>
                      <img :src="item.imageUrl" :alt="item.prompt" class="h-full w-full object-cover" />
                      <div class="absolute inset-0 bg-black/0 transition-colors group-hover:bg-black/35" />
                      <div class="absolute left-3 top-3 rounded-full bg-black/55 px-2 py-1 text-[11px] font-semibold text-white">#{{ index + 1 }}</div>
                      <div class="absolute inset-x-0 bottom-0 flex translate-y-4 items-center justify-center gap-2 px-4 pb-4 opacity-0 transition-all group-hover:translate-y-0 group-hover:opacity-100">
                        <button type="button"
                          class="rounded-xl bg-white px-3 py-2 text-xs font-medium text-neutral-800 shadow-sm transition hover:bg-neutral-100"
                          @click="viewerItem = item">查看</button>
                        <button type="button"
                          class="rounded-xl bg-white px-3 py-2 text-xs font-medium text-neutral-800 shadow-sm transition hover:bg-neutral-100"
                          @click="downloadImage(item.imageUrl, `image-to-prompt-${index + 1}-${Date.now()}.png`)">下载</button>
                      </div>
                    </template>
                  </div>
                </div>
              </div>
              <div v-else
                class="flex min-h-[280px] flex-col items-center justify-center rounded-[24px] border-2 border-dashed border-neutral-200 bg-[linear-gradient(180deg,#fafaf9_0%,#fff_100%)] px-6 text-center">
                <div class="flex h-16 w-16 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-black/5">
                  <svg class="h-8 w-8 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                </div>
                <h3 class="mt-5 text-base font-semibold text-neutral-700">等待绘图任务</h3>
                <p class="mt-2 max-w-md text-sm leading-6 text-neutral-400">点击右上角“用该提示词去生图”后，在右侧配置并提交生成任务。</p>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="viewerItem" class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
        @click.self="viewerItem = null">
        <img :src="viewerItem.imageUrl" :alt="viewerItem.prompt"
          class="max-h-[88vh] max-w-[92vw] rounded-2xl object-contain" />
      </div>
    </Teleport>
  </MainLayout>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import request, { resolveImageUrl, downloadImage } from '@/utils/request'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const HISTORY_KEY = 'image_to_prompt_history'
const HISTORY_TTL = 30 * 60 * 1000
const HISTORY_MAX = 10
const DEFAULT_TEXT_LANGUAGE = 'zh'

const previewUrl = ref('')
const isDragging = ref(false)
const imageBase64 = ref('')
const mimeType = ref('')
const loading = ref(false)
const promptText = ref('')
const lastInputTokens = ref(0)
const lastOutputTokens = ref(0)
const aspectRatio = ref('1:1')
const count = ref(1)
const resolution = ref('')
const supportedResolutions = ref([])
const isGeneratingImages = ref(false)
const imageError = ref('')
const results = ref([])
const batchProgress = ref({ current: 0, total: 0 })
const viewerItem = ref(null)
const activeResultIndex = ref(0)
const showGenerator = ref(false)

const aspectRatios = ['1:1', '4:3', '16:9', '3:4', '9:16', '21:9']
const canSubmit = computed(() => !!imageBase64.value && !loading.value)
const canGenerateImage = computed(() => !!promptText.value.trim() && !isGeneratingImages.value)
const history = ref(loadHistory())
const activeResult = computed(() => results.value[activeResultIndex.value] || null)

watch(history, val => saveHistory(val), { deep: true })
watch(results, val => {
  if (!val.length) {
    activeResultIndex.value = 0
    return
  }
  if (activeResultIndex.value >= val.length) {
    activeResultIndex.value = 0
  }
  const firstDoneIndex = val.findIndex(item => item.status === 'done')
  if (firstDoneIndex >= 0 && val[activeResultIndex.value]?.status !== 'done') {
    activeResultIndex.value = firstDoneIndex
  }
}, { deep: true })

function loadHistory() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    const items = Array.isArray(parsed) ? parsed : parsed.items || []
    return items.filter(item => Date.now() - (item.timestamp || 0) <= HISTORY_TTL)
  } catch {
    return []
  }
}

function saveHistory(items) {
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify({ items, savedAt: Date.now() }))
  } catch {
    // ignore
  }
}

function addToHistory(item) {
  history.value.unshift(item)
  if (history.value.length > HISTORY_MAX) history.value.splice(HISTORY_MAX)
}

function clearHistory() {
  history.value = []
  localStorage.removeItem(HISTORY_KEY)
}

function formatTimeAgo(ts) {
  const mins = Math.floor((Date.now() - ts) / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  return `${Math.floor(mins / 60)} 小时前`
}

function resetImageResults() {
  results.value = []
  activeResultIndex.value = 0
  batchProgress.value = { current: 0, total: 0 }
  imageError.value = ''
}

function loadFile(file) {
  if (!file) return
  const reader = new FileReader()
  reader.onload = ev => {
    const dataUrl = String(ev.target?.result || '')
    imageBase64.value = dataUrl.split(',')[1] || ''
    mimeType.value = dataUrl.split(';')[0]?.split(':')[1] || 'image/png'
    previewUrl.value = dataUrl
    promptText.value = ''
    lastInputTokens.value = 0
    lastOutputTokens.value = 0
    resetImageResults()
  }
  reader.readAsDataURL(file)
}

function handleDrop(e) {
  isDragging.value = false
  loadFile(e.dataTransfer?.files?.[0])
}

function handleFileChange(e) {
  loadFile(e.target.files?.[0])
  e.target.value = ''
}

function clearImage() {
  previewUrl.value = ''
  imageBase64.value = ''
  mimeType.value = ''
  promptText.value = ''
  lastInputTokens.value = 0
  lastOutputTokens.value = 0
  showGenerator.value = false
  resetImageResults()
}

function toggleGenerator() {
  if (!promptText.value.trim()) return
  showGenerator.value = !showGenerator.value
}

async function generatePrompt() {
  if (!imageBase64.value) return
  loading.value = true
  try {
    const res = await request.post('/image-to-prompt/generate', {
      image_data: imageBase64.value,
      mime_type: mimeType.value,
    })
    promptText.value = (res.data?.prompt || '').trim()
    lastInputTokens.value = res.data?.input_tokens || 0
    lastOutputTokens.value = res.data?.output_tokens || 0
    resetImageResults()

    if (!promptText.value) {
      toast.warning('未返回有效内容，请重试')
    } else {
      showGenerator.value = false
      toast.success('提示词已写入工作区')
      addToHistory({
        id: Date.now(),
        imageUrl: previewUrl.value,
        prompt: promptText.value,
        inputTokens: lastInputTokens.value,
        outputTokens: lastOutputTokens.value,
        timestamp: Date.now(),
      })
    }
  } catch {
    toast.error('本次未能获取到结果，请稍后重试')
  } finally {
    loading.value = false
  }
}

async function generateImages() {
  if (!canGenerateImage.value) return
  const prompts = Array(count.value).fill(promptText.value.trim())
  results.value = prompts.map(prompt => ({ prompt, imageUrl: '', status: 'pending', error: '' }))
  activeResultIndex.value = 0
  batchProgress.value = { current: 0, total: prompts.length }
  imageError.value = ''
  isGeneratingImages.value = true

  for (let i = 0; i < prompts.length; i++) {
    results.value[i].status = 'generating'
    batchProgress.value.current = i + 1
    try {
      const payload = {
        prompt: prompts[i],
        aspect_ratio: aspectRatio.value,
        text_language: DEFAULT_TEXT_LANGUAGE,
      }
      if (resolution.value) payload.resolution = resolution.value

      const res = await request.post('/image/generate', payload)
      const data = res.data || {}
      results.value[i] = {
        prompt: prompts[i],
        imageUrl: resolveImageUrl(data.image_url) || (data.image_data ? `data:${data.mime_type};base64,${data.image_data}` : ''),
        status: 'done',
        error: '',
      }
    } catch {
      results.value[i].status = 'error'
      results.value[i].error = '本次未能获取到结果，请稍后重试'
      imageError.value = '部分图片生成失败，可重新提交当前批次。'
    }
  }

  isGeneratingImages.value = false
  if (!imageError.value && results.value.some(item => item.status === 'done')) {
    toast.success('绘图任务完成')
  }
}

async function copyPrompt() {
  const text = promptText.value.trim()
  if (!text) return
  try {
    await copyText(text)
    toast.success('已复制到剪贴板')
  } catch {
    toast.error('复制失败，请手动选择复制')
  }
}

async function copyText(text) {
  if (navigator.clipboard?.writeText && window.isSecureContext) {
    await navigator.clipboard.writeText(text)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '0'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()

  try {
    const copied = document.execCommand('copy')
    if (!copied) throw new Error('copy command failed')
  } finally {
    document.body.removeChild(textarea)
  }
}

function reuseHistoryPrompt(item) {
  promptText.value = item.prompt || ''
  showGenerator.value = false
  toast.success('已载入历史提示词')
}

async function loadDefaultDrawingModel() {
  try {
    const res = await request.get('/models', { params: { type: 'drawing' } })
    const models = res.data || []
    const model = models.find(m => m.is_default && m.is_active) || models.find(m => m.is_active)
    if (model?.supported_resolutions?.length) {
      supportedResolutions.value = model.supported_resolutions
      resolution.value = model.supported_resolutions.includes('1K') ? '1K' : model.supported_resolutions[0]
    }
  } catch {
    // ignore
  }
}

onMounted(loadDefaultDrawingModel)
</script>
