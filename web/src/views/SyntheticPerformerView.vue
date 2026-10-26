<template>
  <MainLayout>
    <main class="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-neutral-900">AI 人物元数据</h1>
          <p class="mt-1 text-sm text-neutral-500">dc:subject · contains-synthetic-performer</p>
        </div>
        <button
          v-if="files.length"
          type="button"
          class="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-neutral-300 bg-white px-4 text-sm font-medium text-neutral-700 hover:bg-neutral-50 disabled:opacity-50"
          :disabled="processing"
          @click="clearFiles"
        >
          <TrashIcon class="h-4 w-4" />
          清空
        </button>
      </header>

      <input ref="fileInput" class="hidden" type="file" accept=".png,.jpg,.jpeg,.webp,image/png,image/jpeg,image/webp" multiple @change="handleFileInput" />
      <button
        type="button"
        class="flex min-h-52 w-full flex-col items-center justify-center border-2 border-dashed px-6 py-10 text-center transition-colors"
        :class="dragging ? 'border-neutral-900 bg-neutral-50' : 'border-neutral-300 bg-white hover:border-neutral-500'"
        @click="fileInput?.click()"
        @dragenter.prevent="dragging = true"
        @dragover.prevent="dragging = true"
        @dragleave.prevent="dragging = false"
        @drop.prevent="handleDrop"
      >
        <ArrowUpTrayIcon class="mb-3 h-8 w-8 text-neutral-500" />
        <span class="text-sm font-semibold text-neutral-900">选择图片</span>
        <span class="mt-1 text-xs text-neutral-500">PNG、JPG、WebP · 最多 100 张 · 超过 100 MB 将自动分批下载</span>
      </button>

      <div v-if="files.length" class="mt-7">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div class="text-sm text-neutral-600">
            已选择 <strong class="text-neutral-900">{{ files.length }}</strong> 张
            <span class="mx-2 text-neutral-300">|</span>
            {{ formatSize(totalSize) }}
          </div>
          <button
            type="button"
            class="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-neutral-900 px-5 text-sm font-semibold text-white hover:bg-neutral-800 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="processing"
            @click="downloadArchive"
          >
            <ArrowDownTrayIcon class="h-4 w-4" />
            {{ processing ? '处理中...' : '下载 ZIP' }}
          </button>
        </div>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          <article v-for="item in files" :key="item.id" class="overflow-hidden rounded-lg border border-neutral-200 bg-white">
            <div class="relative aspect-square bg-neutral-100">
              <img :src="item.preview" :alt="item.file.name" class="h-full w-full object-cover" />
              <button
                type="button"
                class="absolute right-2 top-2 flex h-8 w-8 items-center justify-center rounded-full bg-white text-neutral-700 shadow hover:bg-neutral-100 disabled:opacity-50"
                :disabled="processing"
                title="移除"
                @click="removeFile(item.id)"
              >
                <XMarkIcon class="h-4 w-4" />
              </button>
            </div>
            <div class="px-3 py-2">
              <p class="truncate text-xs font-medium text-neutral-800" :title="item.file.name">{{ item.file.name }}</p>
              <p class="mt-1 text-[11px] text-neutral-400">{{ formatSize(item.file.size) }}</p>
            </div>
          </article>
        </div>
      </div>
    </main>
  </MainLayout>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import { ArrowDownTrayIcon, ArrowUpTrayIcon, TrashIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import request from '@/utils/request'
import { useToast } from '@/composables/useToast'

const allowedExtensions = /\.(png|jpe?g|webp)$/i
const maxFiles = 100
const maxBatchBytes = 95 * 1024 * 1024
const maxFileBytes = 100 * 1024 * 1024
const fileInput = ref(null)
const files = ref([])
const dragging = ref(false)
const processing = ref(false)
const toast = useToast()

const totalSize = computed(() => files.value.reduce((sum, item) => sum + item.file.size, 0))

function addFiles(selected) {
  const incoming = Array.from(selected || []).filter(file => allowedExtensions.test(file.name))
  const rejected = Array.from(selected || []).length - incoming.length
  if (rejected) toast.error('仅支持 PNG、JPG 和 WebP 图片')
  const oversized = incoming.filter(file => file.size > maxFileBytes)
  if (oversized.length) toast.error(`${oversized.map(file => file.name).join('、')} 超过 100 MB，无法处理`)

  const known = new Set(files.value.map(item => `${item.file.name}:${item.file.size}:${item.file.lastModified}`))
  const additions = incoming
    .filter(file => file.size <= maxFileBytes)
    .filter(file => !known.has(`${file.name}:${file.size}:${file.lastModified}`))
    .map(file => ({ id: crypto.randomUUID(), file, preview: URL.createObjectURL(file) }))

  if (files.value.length + additions.length > maxFiles) {
    additions.forEach(item => URL.revokeObjectURL(item.preview))
    toast.error(`每批最多选择 ${maxFiles} 张图片`)
    return
  }
  files.value.push(...additions)
  if (files.value.reduce((sum, item) => sum + item.file.size, 0) > maxFileBytes) {
    toast.info('图片总大小超过 100 MB，下载时将自动分批完成')
  }
}

function handleFileInput(event) {
  addFiles(event.target.files)
  event.target.value = ''
}

function handleDrop(event) {
  dragging.value = false
  addFiles(event.dataTransfer?.files)
}

function removeFile(id) {
  const item = files.value.find(candidate => candidate.id === id)
  if (item) URL.revokeObjectURL(item.preview)
  files.value = files.value.filter(candidate => candidate.id !== id)
}

function clearFiles() {
  files.value.forEach(item => URL.revokeObjectURL(item.preview))
  files.value = []
}

async function downloadArchive() {
  if (!files.value.length || processing.value) return
  processing.value = true
  try {
    const batches = []
    let batch = []
    let batchSize = 0
    for (const item of files.value) {
      if (batch.length && batchSize + item.file.size > maxBatchBytes) {
        batches.push(batch)
        batch = []
        batchSize = 0
      }
      batch.push(item)
      batchSize += item.file.size
    }
    if (batch.length) batches.push(batch)

    for (let index = 0; index < batches.length; index += 1) {
      const form = new FormData()
      batches[index].forEach(item => form.append('files', item.file, item.file.name))
      const response = await request.post('/image-metadata/synthetic-performer/batch', form, { responseType: 'blob' })
      const url = URL.createObjectURL(response.data)
      const link = document.createElement('a')
      link.href = url
      link.download = batches.length > 1
        ? archiveFilename(response.headers?.['content-disposition']).replace(/\.zip$/i, `-part-${index + 1}.zip`)
        : archiveFilename(response.headers?.['content-disposition'])
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
      if (batches.length > 1) await new Promise(resolve => setTimeout(resolve, 350))
    }
    toast.success(batches.length > 1 ? `已分批下载 ${batches.length} 个 ZIP` : 'ZIP 已生成')
  } catch (error) {
    let message = '处理失败，请重试'
    const data = error?.response?.data
    if (data instanceof Blob) {
      try {
        const payload = JSON.parse(await data.text())
        message = payload.error || message
      } catch {}
    }
    toast.error(message)
  } finally {
    processing.value = false
  }
}

function archiveFilename(disposition = '') {
  const utf8 = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8?.[1]) return decodeURIComponent(utf8[1])
  return `synthetic-performer-images-${Date.now()}.zip`
}

function formatSize(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

onBeforeUnmount(clearFiles)
</script>
