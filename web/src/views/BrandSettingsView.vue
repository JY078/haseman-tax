<template>
  <MainLayout>
    <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-10 space-y-6">
      <div>
        <h1 class="text-3xl font-bold text-neutral-900">品牌 Logo 设置</h1>
        <p class="mt-2 text-sm text-neutral-600">上传全局品牌 Logo。A+ 图生成时会自动读取这里的品牌资产用于品牌化排版。</p>
      </div>

      <div class="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <section class="rounded-2xl border border-neutral-200 bg-white p-5 space-y-5">
          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="block text-sm font-medium text-neutral-700">品牌 Logo</label>
              <span class="text-xs" :class="logoConfigured ? 'text-emerald-600' : 'text-neutral-400'">
                {{ logoConfigured ? '已配置' : '未配置' }}
              </span>
            </div>

            <div v-if="logoPreview" class="rounded-2xl border border-neutral-200 bg-neutral-50 p-5">
              <div class="flex items-center justify-center min-h-[180px] rounded-xl bg-white border border-dashed border-neutral-200">
                <img :src="logoPreview" alt="品牌 Logo" class="max-h-[120px] max-w-full object-contain" />
              </div>
              <div class="mt-4 flex flex-wrap gap-3">
                <label class="inline-flex cursor-pointer items-center justify-center rounded-xl border border-neutral-300 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50">
                  更换 Logo
                  <input ref="fileInputRef" type="file" accept="image/*" class="hidden" @change="handleFileChange" />
                </label>
                <button type="button" class="rounded-xl border border-red-200 px-4 py-2 text-sm text-red-600 hover:bg-red-50" @click="removeLogo">删除 Logo</button>
              </div>
            </div>

            <label
              v-else
              class="flex min-h-[220px] cursor-pointer flex-col items-center justify-center rounded-2xl border-2 border-dashed transition-colors"
              :class="isDragging ? 'border-neutral-900 bg-neutral-50' : 'border-neutral-300 hover:border-neutral-500'"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <div class="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-100 text-neutral-500">
                <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
              </div>
              <p class="mt-4 text-sm font-medium text-neutral-700">{{ isDragging ? '松开鼠标上传 Logo' : '点击或拖拽上传品牌 Logo' }}</p>
              <p class="mt-1 text-xs text-neutral-400">建议使用透明底 PNG，保持 Logo 清晰干净</p>
              <input ref="emptyFileInputRef" type="file" accept="image/*" class="hidden" @change="handleFileChange" />
            </label>
          </div>

          <div v-if="uploadError" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-xs text-red-600">
            {{ uploadError }}
          </div>

          <div class="flex items-center gap-3">
            <button
              type="button"
              class="rounded-xl bg-neutral-900 px-5 py-3 text-sm font-medium text-white transition hover:bg-neutral-800 disabled:cursor-not-allowed disabled:bg-neutral-300"
              :disabled="saving"
              @click="saveSettings"
            >
              {{ saving ? '保存中…' : '保存品牌设置' }}
            </button>
            <span class="text-xs text-neutral-400">保存后，A+ 图生成会自动读取最新品牌资产。</span>
          </div>
        </section>

        <section class="rounded-2xl border border-neutral-200 bg-white p-5 space-y-4">
          <h2 class="text-lg font-semibold text-neutral-900">使用逻辑</h2>
          <ul class="space-y-3 text-sm text-neutral-600 leading-6">
            <li>• 这里的 Logo 是全局品牌资产，不在单个产品里重复上传。</li>
            <li>• A+ 图生成时，系统会将 Logo 作为品牌参考输入给模型。</li>
            <li>• 模型会尽量把 Logo 以小而克制的方式排入版式，不抢产品主体。</li>
            <li>• 若某次任务不想带 Logo，可在产品图生成页手动关闭本次启用。</li>
          </ul>
        </section>
      </div>
    </div>
  </MainLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import request, { resolveImageUrl } from '@/utils/request'

const brandLogoUrl = ref('')
const localPreview = ref('')
const uploadedLogoUrl = ref('')
const isDragging = ref(false)
const uploadError = ref('')
const saving = ref(false)
const fileInputRef = ref(null)
const emptyFileInputRef = ref(null)

const logoPreview = computed(() => resolveImageUrl(localPreview.value || uploadedLogoUrl.value || brandLogoUrl.value))
const logoConfigured = computed(() => !!String(logoPreview.value || '').trim())

onMounted(() => {
  fetchBrandSettings()
})

async function fetchBrandSettings() {
  const res = await request.get('/brand-settings')
  brandLogoUrl.value = res.data?.brand_logo_url || ''
  uploadedLogoUrl.value = ''
  localPreview.value = ''
}

function loadFile(file) {
  if (!file) return
  uploadError.value = ''
  const reader = new FileReader()
  reader.onload = (ev) => {
    localPreview.value = ev.target?.result || ''
  }
  reader.readAsDataURL(file)
  uploadLogo(file)
}

function handleDrop(e) {
  isDragging.value = false
  loadFile(e.dataTransfer?.files?.[0])
}

function handleFileChange(e) {
  loadFile(e.target.files?.[0])
  if (e.target) e.target.value = ''
}

async function uploadLogo(file) {
  try {
    const dataUrl = await fileToDataUrl(file)
    const [meta, data] = String(dataUrl).split(',')
    const mimeType = String(meta || '').split(':')[1]?.split(';')[0] || file.type || 'image/png'
    const res = await request.post('/upload/image', {
      data,
      mime_type: mimeType,
    })
    uploadedLogoUrl.value = res.data?.url || ''
  } catch (err) {
    uploadError.value = err?.response?.data?.error || 'Logo 上传失败'
  }
}

function removeLogo() {
  localPreview.value = ''
  uploadedLogoUrl.value = ''
  brandLogoUrl.value = ''
  uploadError.value = ''
}

async function saveSettings() {
  saving.value = true
  try {
    await request.post('/settings', {
      brand_logo_url: uploadedLogoUrl.value || brandLogoUrl.value || '',
    })
    await fetchBrandSettings()
  } finally {
    saving.value = false
  }
}

function fileToDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}
</script>
