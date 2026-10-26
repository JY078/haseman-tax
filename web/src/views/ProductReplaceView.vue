<template>
  <MainLayout>
    <section
      class="mx-2 grid h-[calc(100vh-92px)] grid-cols-[minmax(320px,368px)_minmax(0,1fr)] gap-[18px] py-4 max-lg:h-auto max-lg:grid-cols-1 max-lg:overflow-visible">
      <div
        class="sticky top-[18px] flex max-h-[calc(100vh-110px)] flex-col rounded-[30px] border border-neutral-200 bg-white shadow-[0_18px_50px_rgba(15,23,42,0.05)] max-lg:static max-lg:max-h-none max-lg:overflow-visible max-lg:flex-row max-lg:gap-[18px]">

        <!-- Scrollable content area -->
        <div class="flex-1 overflow-y-auto p-5 space-y-4 max-lg:overflow-visible">
          <!-- Mode Selector -->
          <div class="bg-white border border-neutral-200 rounded-xl p-3">
            <div class="flex p-1 bg-neutral-100 rounded-xl">
              <button v-for="m in modes" :key="m.value" type="button"
                class="flex-1 py-2.5 text-sm font-medium rounded-lg transition-all"
                :class="mode === m.value
                  ? 'bg-white text-neutral-900 shadow-sm'
                  : 'text-neutral-500 hover:text-neutral-700'"
                @click="mode = m.value">
                {{ m.label }}
              </button>
            </div>
          </div>

          <!-- ===== WATER FLOW MODE ===== -->
          <template v-if="mode === 'water'">
            <!-- Water Library Selection -->
            <div class="bg-white border border-neutral-200 rounded-xl p-5">
              <div class="flex items-center justify-between mb-3">
                <label class="text-sm font-semibold text-neutral-900">
                  水流效果
                  <span class="text-red-400 ml-0.5">*</span>
                </label>
                <div class="px-2.5 py-1 rounded-full bg-neutral-100 text-xs font-medium text-neutral-600">
                  {{ selectedWater ? '1/1' : '0/1' }}
                </div>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <button v-for="water in waterLibrary" :key="water.id" type="button"
                  class="group relative overflow-hidden rounded-xl border-2 transition-all aspect-[4/3]"
                  :class="selectedWater?.id === water.id
                    ? 'border-neutral-900 ring-2 ring-neutral-900/20'
                    : 'border-neutral-200 hover:border-neutral-400'"
                  @click="selectedWater = selectedWater?.id === water.id ? null : water">
                  <img :src="water.thumbnail" :alt="water.name" class="w-full h-full object-cover" />
                  <!-- Hover overlay with name -->
                  <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex items-end justify-center pb-3">
                    <span class="text-white text-sm font-medium drop-shadow">{{ water.name }}</span>
                  </div>
                  <!-- Selected checkmark -->
                  <div class="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity"
                    :class="selectedWater?.id === water.id ? 'opacity-100' : ''">
                    <div class="w-8 h-8 rounded-full bg-neutral-900 flex items-center justify-center">
                      <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                      </svg>
                    </div>
                  </div>
                </button>
              </div>
            </div>

            <!-- Original Image Upload -->
            <div class="bg-white border rounded-xl p-5 transition-all duration-200" :class="[
              refDragOver ? 'border-neutral-900 bg-neutral-50 ring-2 ring-neutral-900/10' : 'border-neutral-200',
              originalImage ? 'border-emerald-400 bg-emerald-50/50' : ''
            ]" @dragover.prevent="onRefDragOver" @dragleave="onRefDragLeave" @drop.prevent="onRefDrop">
              <div class="flex items-center justify-between mb-3">
                <div>
                  <label class="text-sm font-semibold text-neutral-900">
                    原图
                    <span class="text-red-400 ml-0.5">*</span>
                  </label>
                </div>
              </div>
              <div v-if="!originalImage"
                class="relative flex min-h-[148px] w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-4 py-6 text-center transition-all"
                :class="refDragOver
                  ? 'border-neutral-900 bg-neutral-100 ring-4 ring-neutral-900/10 scale-[1.02]'
                  : 'border-neutral-300 bg-neutral-50 hover:border-neutral-500 hover:bg-white'"
                @click="onUploadAreaClick('original')"
                @dragover.stop.prevent="onRefDragOver"
                @dragleave.stop="onRefDragLeave"
                @drop.stop.prevent="onRefDrop">
                <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-black/5">
                  <PhotoIcon class="w-6 h-6 text-neutral-500" />
                </div>
                <div class="mt-4">
                  <p class="text-sm font-medium text-neutral-800">
                    <template v-if="refDragOver">松开即可添加</template>
                    <template v-else>上传原图</template>
                  </p>
                </div>
                <input ref="originalUploadInputWater" type="file" accept="image/*" class="hidden" @change="handleOriginalImage" />
              </div>
              <div v-else class="relative overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 aspect-[4/3]">
                <img :src="originalImage.url" class="w-full h-full object-contain bg-white p-1.5" />
                <button class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md transition-all hover:bg-white"
                  @click="originalImage = null; originalUploadInputWater.value = ''">
                  <XMarkIcon class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </template>

          <!-- ===== PRODUCT REPLACE MODE ===== -->
          <template v-if="mode === 'product'">
            <!-- Original Image Upload -->
            <div class="bg-white border rounded-xl p-5 transition-all duration-200" :class="[
              originalDragOver ? 'border-neutral-900 bg-neutral-50 ring-2 ring-neutral-900/10' : 'border-neutral-200',
              originalImage ? 'border-emerald-400 bg-emerald-50/50' : ''
            ]" @dragover.prevent="onRefDragOver" @dragleave="onRefDragLeave" @drop.prevent="onRefDrop">
              <div class="flex items-center justify-between mb-3">
                <label class="text-sm font-semibold text-neutral-900">
                  原图
                  <span class="text-red-400 ml-0.5">*</span>
                </label>
              </div>
              <div v-if="!originalImage"
                class="relative flex min-h-[148px] w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-4 py-6 text-center transition-all"
                :class="originalDragOver
                  ? 'border-neutral-900 bg-neutral-100 ring-4 ring-neutral-900/10 scale-[1.02]'
                  : 'border-neutral-300 bg-neutral-50 hover:border-neutral-500 hover:bg-white'"
                @click="onUploadAreaClick"
                @dragover.stop.prevent="onRefDragOver"
                @dragleave.stop="onRefDragLeave"
                @drop.stop.prevent="onRefDrop">
                <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-black/5">
                  <PhotoIcon class="w-6 h-6 text-neutral-500" />
                </div>
                <div class="mt-4">
                  <p class="text-sm font-medium text-neutral-800">
                    <template v-if="originalDragOver">松开即可添加</template>
                    <template v-else>上传原图</template>
                  </p>
                </div>
                <input ref="originalUploadInput" type="file" accept="image/*" class="hidden" @change="handleOriginalImage" />
              </div>
              <div v-else class="relative overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 aspect-[4/3]">
                <img :src="originalImage.url" class="w-full h-full object-contain bg-white p-1.5" />
                <button class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md transition-all hover:bg-white"
                  @click="originalImage = null">
                  <XMarkIcon class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            <!-- Product Image Upload -->
            <div class="bg-white border rounded-xl p-5 transition-all duration-200" :class="[
              productDragOver ? 'border-neutral-900 bg-neutral-50 ring-2 ring-neutral-900/10' : 'border-neutral-200',
              productImage ? 'border-emerald-400 bg-emerald-50/50' : ''
            ]" @dragover.prevent="onProductDragOver" @dragleave="onProductDragLeave" @drop.prevent="onProductDrop">
              <div class="flex items-center justify-between mb-3">
                <label class="text-sm font-semibold text-neutral-900">
                  替换产品图
                  <span class="text-red-400 ml-0.5">*</span>
                </label>
              </div>
              <div v-if="!productImage"
                class="relative flex min-h-[148px] w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-4 py-6 text-center transition-all"
                :class="productDragOver
                  ? 'border-neutral-900 bg-neutral-100 ring-4 ring-neutral-900/10 scale-[1.02]'
                  : 'border-neutral-300 bg-neutral-50 hover:border-neutral-500 hover:bg-white'"
                @click="productUploadInput.click()"
                @dragover.stop.prevent="onProductDragOver"
                @dragleave.stop="onProductDragLeave"
                @drop.stop.prevent="onProductDrop">
                <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-black/5">
                  <PhotoIcon class="w-6 h-6 text-neutral-500" />
                </div>
                <div class="mt-4">
                  <p class="text-sm font-medium text-neutral-800">
                    <template v-if="productDragOver">松开即可添加</template>
                    <template v-else>上传需要替换的产品图</template>
                  </p>
                </div>
                <input ref="productUploadInput" type="file" accept="image/*" class="hidden" @change="handleProductImage" />
              </div>
              <div v-else class="relative overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 aspect-[4/3]">
                <img :src="productImage.url" class="w-full h-full object-contain bg-white p-1.5" />
                <button class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md transition-all hover:bg-white"
                  @click="productImage = null">
                  <XMarkIcon class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </template>

          <!-- Aspect Ratio -->
          <div class="bg-white border border-neutral-200 rounded-xl p-5 overflow-hidden">
            <label class="block text-sm font-semibold text-neutral-900 mb-3">图片比例</label>
            <div class="ratio-visual-grid">
              <button v-for="item in ratioOptionsForCard" :key="item.value" type="button"
                :class="['ratio-card', aspectRatio === item.value && 'on']" @click="aspectRatio = item.value">
                <span class="ratio-card-frame">
                  <span :class="['ratio-card-shape', item.shapeClass]"></span>
                </span>
                <span class="ratio-card-label">{{ item.value }}</span>
              </button>
            </div>
          </div>

          <!-- Count & Resolution -->
          <div class="grid grid-cols-2 gap-3">
            <div class="bg-white border border-neutral-200 rounded-xl p-4">
              <label class="block text-sm font-semibold text-neutral-900 mb-3">生成数量</label>
              <div class="grid grid-cols-4 gap-2">
                <button v-for="n in 4" :key="n" class="h-9 text-sm font-medium rounded-lg border-2 transition-all"
                  :class="count === n
                    ? 'border-neutral-900 bg-neutral-900 text-white'
                    : 'border-neutral-200 text-neutral-600 hover:border-neutral-400'" @click="count = n">{{ n }}</button>
              </div>
            </div>
            <div class="bg-white border border-neutral-200 rounded-xl p-4">
              <label class="block text-sm font-semibold text-neutral-900 mb-3">分辨率</label>
              <div class="grid grid-cols-2 gap-2">
                <button v-for="q in qualityOptions" :key="q" class="h-9 text-sm font-medium rounded-lg border-2 transition-all"
                  :class="quality === q
                    ? 'border-neutral-900 bg-neutral-900 text-white'
                    : 'border-neutral-200 text-neutral-600 hover:border-neutral-400'" @click="quality = q">{{ q }}</button>
              </div>
            </div>
          </div>
        </div>

        <!-- Fixed bottom: Generate button -->
        <div
          class="shrink-0 z-[9999] flex flex-col items-stretch gap-3 border-t border-neutral-200 bg-white px-6 pb-5 pt-[18px] shadow-[0_-12px_32px_rgba(15,23,42,0.06)]">
          <button
            class="w-full py-3.5 text-sm font-semibold rounded-xl text-white transition-all flex items-center justify-center gap-2 shadow-sm"
            :class="canGenerate ? 'bg-neutral-900 hover:bg-neutral-800 active:scale-[0.98]' : 'bg-neutral-300 cursor-not-allowed'"
            :disabled="!canGenerate" @click="generate">
            <template v-if="isGenerating">
              <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              {{ batchProgress.current }}/{{ batchProgress.total }} 生成中（可继续查看右侧结果）
            </template>
            <template v-else>
              <SparklesIcon class="w-4 h-4" />
              开始生成
            </template>
          </button>
          <p v-if="globalError" class="m-0 text-sm text-red-600">{{ globalError }}</p>
        </div>
      </div>

      <!-- ===== Right Panel ===== -->
      <div
        class="sticky top-[18px] flex min-w-0 max-h-[calc(100vh-110px)] flex-col overflow-hidden rounded-[30px] border border-neutral-200 bg-white shadow-[0_18px_50px_rgba(15,23,42,0.05)] max-lg:static max-lg:max-h-none">
        <div
          class="z-10 m-0 flex items-center justify-between gap-3 border-b border-[#eef2f7] bg-white px-5 pb-2.5 pt-3 shadow-[0_10px_18px_-18px_rgba(15,23,42,0.18)]">
          <div>
            <h2 class="m-0 text-[18px] font-bold tracking-[0.01em] text-neutral-900">最近任务</h2>
            <span class="mt-1 block text-[11px] leading-[1.5] text-neutral-400">生成中的任务与已完成任务都会展示在这里</span>
          </div>
          <button v-if="hasAnyResult && !isGenerating"
            class="h-9 shrink-0 rounded-full bg-neutral-100 px-3.5 text-sm font-medium text-neutral-900 transition-all hover:-translate-y-px hover:shadow-[0_10px_22px_rgba(15,23,42,0.08)]"
            type="button" @click="generate">重新生成</button>
        </div>

        <div class="flex-1 overflow-auto overflow-x-hidden pb-2 max-lg:overflow-visible">
          <template v-if="displayTasks.length">
            <div class="flex min-h-0 flex-1 flex-col gap-2 overflow-auto px-[14px] pb-3 pt-2.5">
              <section v-for="(task, taskIndex) in displayTasks" :key="task.key"
                class="overflow-hidden rounded-2xl border border-neutral-200 bg-[linear-gradient(180deg,#fff,#fcfcfd)] p-2.5 transition-[box-shadow,border-color] hover:border-neutral-300 hover:shadow-[0_10px_24px_rgba(15,23,42,0.05)]">
                <div class="flex items-start justify-between gap-2 border-b border-[#eef2f7] pb-1.5">
                  <div class="flex min-w-0 items-center gap-2">
                    <img :src="task.cover" :alt="task.title"
                      class="h-[54px] w-[54px] rounded-[14px] bg-neutral-100 object-cover" />
                    <div>
                      <strong class="block line-clamp-1 text-[13px] leading-[1.25] text-neutral-900">{{ task.title }}</strong>
                      <span class="mt-0.5 block line-clamp-1 text-[11px] leading-[1.3] text-neutral-500">{{ task.subtitle }}</span>
                    </div>
                  </div>
                  <div class="flex min-w-[84px] shrink-0 flex-col items-end gap-[3px] text-right">
                    <div class="flex items-center justify-end gap-1.5">
                      <span v-if="task.items.some(item => item.status === 'generating' || item.status === 'pending')"
                        class="inline-flex min-h-6 items-center rounded-full bg-indigo-50 px-[9px] text-[11px] font-semibold text-indigo-700">生成中</span>
                      <span v-else
                        class="inline-flex min-h-6 items-center rounded-full px-[9px] text-[11px] font-semibold"
                        :class="task.failedCount ? 'bg-rose-50 text-rose-700' : 'bg-emerald-50 text-emerald-700'">
                        {{ task.failedCount ? `完成 ${task.doneCount}/${task.total}` : '已完成' }}
                      </span>
                    </div>
                    <span class="whitespace-nowrap text-[11px] font-semibold text-neutral-400">{{ task.timeText }}</span>
                    <!-- <div v-if="task.items.some(item => item.status === 'done')" class="mt-0.5 flex justify-end">
                      <button class="border-none bg-transparent p-0 text-xs font-semibold text-neutral-900 transition hover:underline"
                        type="button" @click="saveTaskGroupToDesign(task)">保存本批可用结果</button>
                    </div> -->
                  </div>
                </div>

                <div class="mt-2 grid items-start gap-1.5"
                  :class="task.items.length === 1 ? 'grid-cols-[minmax(0,340px)] max-lg:grid-cols-1' : 'grid-cols-4 max-xl:grid-cols-3 max-lg:grid-cols-2 max-sm:grid-cols-1'">
                  <div v-for="(item, index) in task.items" :key="item.key" class="relative min-w-0">
                    <div class="group relative flex aspect-[16/9] items-center justify-center overflow-hidden rounded-[14px] bg-neutral-100 shadow-[inset_0_0_0_1px_rgba(229,231,235,.8)]">
                      <template v-if="item.status === 'done' && item.imageUrl">
                        <img :src="item.imageUrl" :alt="item.prompt || `结果${index + 1}`" class="h-full w-full object-cover" />
                      </template>
                      <div v-else-if="item.status === 'error'"
                        class="absolute inset-0 z-[2] flex flex-col items-center justify-center gap-2.5 bg-[rgba(127,29,29,.78)] p-3.5 text-center text-white">
                        <ExclamationCircleIcon class="w-7 h-7" />
                        <p class="m-0 text-[13px] font-semibold leading-[1.5]">生成失败</p>
                        <span class="text-[11px] leading-[1.4] text-white/80">{{ item.error || '请稍后重试' }}</span>
                      </div>
                      <div v-else class="flex flex-col items-center justify-center gap-2 p-3 text-center text-[11px] text-neutral-500">
                        <!-- 进度条 -->
                        <div class="w-full px-2">
                          <div class="relative h-2 w-full overflow-hidden rounded-full bg-neutral-200">
                            <div class="absolute left-0 top-0 h-full rounded-full bg-neutral-900 transition-all duration-200"
                              :style="{ width: `${Math.min(progressTrackers[item.key]?.current ?? 0, getProgressLimit(item.status))}%` }">
                            </div>
                          </div>
                          <div class="mt-1.5 flex items-center justify-between text-[10px] text-neutral-400">
                            <span>{{ item.status === 'generating' || item.status === 'processing' ? '生成中' : '等待中' }}</span>
                            <span>{{ Math.round(Math.min(progressTrackers[item.key]?.current ?? 0, getProgressLimit(item.status))) }}%</span>
                          </div>
                        </div>
                        <span class="text-[11px] leading-[1.4] text-neutral-500">正在努力生成画面…</span>
                      </div>
                      <div v-if="item.status === 'done' && item.imageUrl"
                        class="pointer-events-none absolute inset-0 flex flex-wrap items-center justify-center gap-1 bg-[rgba(17,24,39,0)] p-2 opacity-0 transition group-hover:pointer-events-auto group-hover:bg-[rgba(17,24,39,.38)] group-hover:opacity-100">
                        <button class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button" @click.prevent="openImageViewer(item, index)">查看</button>
                        <button class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button"
                          @click.prevent="downloadImage(item.imageUrl, `replace-${taskIndex + 1}-${index + 1}-${Date.now()}.png`)">下载</button>
                        <button class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button" :disabled="!item.id || item.savedToDesign || isSaving(item)"
                          @click.prevent="saveToDesign(item)">{{ item.savedToDesign ? '已保存' : (isSaving(item) ? '保存中' : '保存') }}</button>
                      </div>
                    </div>
                    <p class="mx-0.5 mb-0 mt-2 line-clamp-2 text-[11px] leading-[1.5] text-neutral-500">{{ task.title || '未填写提示词' }}</p>
                  </div>
                </div>
              </section>
            </div>
          </template>
          <div v-else
            class="m-[10px_20px_16px] flex flex-1 items-center justify-center overflow-auto rounded-[26px] border border-dashed border-neutral-300 bg-[linear-gradient(180deg,#fcfcfd,#f9fafb)] p-6 text-neutral-500">
            <div class="flex max-w-[420px] flex-col items-center gap-2.5 text-center">
              <strong class="text-[15px] text-neutral-900">暂无任务</strong>
              <span class="text-[13px] leading-[1.6]">在左侧配置参数后开始生成，结果会在这里以任务卡片形式展示。</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Image Viewer Lightbox -->
    <ImagePreview
      :visible="!!viewerItem"
      :imageUrl="viewerItem?.imageUrl ?? ''"
      :alt="viewerItem?.prompt ?? ''"
      :maxWidth="1200"
      @close="closeImageViewer"
    >
      <template #footer>
        <div class="preview-footer-bar">
          <p class="preview-footer-prompt">{{ viewerItem?.prompt }}</p>
          <div class="preview-footer-actions">
            <button class="preview-download-btn" @click="downloadImage(viewerItem.imageUrl, `replace-${viewerIndex + 1}-${Date.now()}.png`)">
              <ArrowDownTrayIcon class="w-3.5 h-3.5" />
              下载
            </button>
          </div>
        </div>
      </template>
    </ImagePreview>
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  SparklesIcon, PhotoIcon, XMarkIcon,
  ExclamationCircleIcon, ArrowDownTrayIcon,
} from '@heroicons/vue/24/outline'
import MainLayout from '@/components/layout/MainLayout.vue'
import request, { resolveImageUrl, downloadImage } from '@/utils/request'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import { useToast } from '@/composables/useToast'
import { useFakeProgress } from '@/composables/useFakeProgress'

const toast = useToast()
const fakeProgress = useFakeProgress({ storageKeyPrefix: 'prod_progress' })

// 进度跟踪器类型
interface ProgressTracker {
  current: number
}
const progressTrackers = ref<Record<string, ProgressTracker>>({})

// 获取进度条上限值
function getProgressLimit(status: string): number {
  if (status === 'generating' || status === 'processing') return 99
  return 50
}

// ---------- Mode ----------
const mode = ref('water') // 'water' | 'product'
const modes = [
  { value: 'water', label: '水流替换' },
  { value: 'product', label: '产品替换' },
]

// ---------- Water Library ----------
// TODO: 替换为实际水流效果图 URL
const waterLibrary = ref([
  { id: 1, name: '柱状水', thumbnail: 'https://haseman-upload-prod.oss-cn-shenzhen.aliyuncs.com/uploads/images/20260513-181402.jpg' },
  { id: 2, name: '花洒水', thumbnail: 'https://haseman-upload-prod.oss-cn-shenzhen.aliyuncs.com/uploads/images/20260513-181414.jpg' },
  { id: 3, name: '刀锋水', thumbnail: 'https://haseman-upload-prod.oss-cn-shenzhen.aliyuncs.com/uploads/images/20260513-181424.jpg' },
  { id: 4, name: '瀑布水', thumbnail: 'https://haseman-upload-prod.oss-cn-shenzhen.aliyuncs.com/uploads/images/20260513-181432.jpg' },
])
const selectedWater = ref(null)

// ---------- Images ----------
const originalImage = ref(null)
const productImage = ref(null)
const originalUploadInput = ref(null)
const originalUploadInputWater = ref(null)
const productUploadInput = ref(null)
const refDragOver = ref(false)
const originalDragOver = ref(false)
const productDragOver = ref(false)

// ---------- Generate state ----------
const count = ref(1)
const aspectRatio = ref('1:1')
const quality = ref('2K')
const qualityOptions = ['2K', '4K']
const isGenerating = ref(false)
const globalError = ref('')
const batchResults = ref([])
const batchProgress = ref({ current: 0, total: 0 })
const savingIds = ref([])

// ---------- History ----------
const HISTORY_CACHE_KEY = 'product_replace_history'
const HISTORY_TTL_MS = 30 * 60 * 1000

function loadHistoryCache() {
  try {
    const raw = localStorage.getItem(HISTORY_CACHE_KEY)
    if (!raw) return []
    const { items, savedAt } = JSON.parse(raw)
    if (!Array.isArray(items) || Date.now() - savedAt > HISTORY_TTL_MS) {
      localStorage.removeItem(HISTORY_CACHE_KEY)
      return []
    }
    return items.map((item, index) => normalizeHistoryTask(item, index)).filter(Boolean)
  } catch { return [] }
}

function normalizeHistoryTask(item, index = 0) {
  if (!item || typeof item !== 'object') return null
  if (Array.isArray(item.items)) {
    const items = item.items.filter(entry => entry?.imageUrl).map((entry, entryIndex) => ({
      id: entry.id || null,
      prompt: entry.prompt || '',
      imageUrl: entry.imageUrl,
      status: 'done',
      error: '',
      savedToDesign: !!entry.savedToDesign,
      key: `history-item-${index}-${entryIndex}`,
    }))
    if (!items.length) return null
    return {
      key: item.key || `history-${index}`,
      type: 'history',
      title: item.title || items[0].prompt || `历史记录 ${index + 1}`,
      subtitle: item.subtitle || '历史生成结果',
      cover: item.cover || items[0].imageUrl,
      timeText: item.timeText || (item.createdAt ? new Date(item.createdAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : ''),
      total: typeof item.total === 'number' ? item.total : items.length,
      doneCount: typeof item.doneCount === 'number' ? item.doneCount : items.length,
      failedCount: typeof item.failedCount === 'number' ? item.failedCount : 0,
      createdAt: item.createdAt || '',
      items,
    }
  }
  if (!item.imageUrl) return null
  return {
    key: `history-${index}`,
    type: 'history',
    title: item.prompt || `历史记录 ${index + 1}`,
    subtitle: '历史生成结果',
    cover: item.imageUrl,
    timeText: item.createdAt ? new Date(item.createdAt).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : '',
    total: 1, doneCount: 1, failedCount: 0,
    createdAt: item.createdAt || '',
    items: [{ id: item.id || null, prompt: item.prompt || '', imageUrl: item.imageUrl, status: 'done', error: '', savedToDesign: !!item.savedToDesign, key: `history-item-${index}-0` }],
  }
}

function saveHistoryCache(items) {
  try { localStorage.setItem(HISTORY_CACHE_KEY, JSON.stringify({ items, savedAt: Date.now() })) } catch { /**/ }
}

const history = ref(loadHistoryCache())
watch(history, (val) => { saveHistoryCache(val) }, { deep: true })

// ---------- Image viewer ----------
const viewerItem = ref(null)
const viewerIndex = ref(0)

function openImageViewer(item, index) { viewerItem.value = item; viewerIndex.value = index }
function closeImageViewer() { viewerItem.value = null }

// ---------- Static data ----------
const ratioOptionsForCard = [
  { value: '1:1', shapeClass: 'ratio-shape-square' },
  { value: '4:3', shapeClass: 'ratio-shape-landscape' },
  { value: '3:4', shapeClass: 'ratio-shape-portrait' },
  { value: '16:9', shapeClass: 'ratio-shape-wide' },
  { value: '9:16', shapeClass: 'ratio-shape-tall' },
  { value: '21:9', shapeClass: 'ratio-shape-cinema' },
]

// ---------- Computed ----------
const canGenerate = computed(() => {
  if (isGenerating.value) return false
  if (mode.value === 'water') {
    return !!selectedWater.value && !!originalImage.value
  }
  return !!originalImage.value && !!productImage.value
})

const hasAnyResult = computed(() => batchResults.value.length > 0 || history.value.length > 0)

const displayTasks = computed(() => {
  const tasks = []
  if (batchResults.value.length > 0 && isGenerating.value) {
    const firstDone = batchResults.value.find(item => item.imageUrl)
    const modeName = mode.value === 'water' ? '水流替换' : '产品替换'
    tasks.push({
      key: 'current-task', type: 'current',
      title: isGenerating.value ? '当前生成任务' : '最近生成任务',
      subtitle: `${modeName} · ${count.value} 张 · 比例 ${aspectRatio.value}`,
      cover: firstDone?.imageUrl || 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="160" height="160"><rect width="100%" height="100%" fill="%23f3f4f6"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="%239ca3af" font-size="14">AI</text></svg>',
      timeText: '刚刚', total: batchResults.value.length,
      doneCount: batchResults.value.filter(item => item.status === 'done').length,
      failedCount: batchResults.value.filter(item => item.status === 'error').length,
      items: batchResults.value.map((item, index) => ({ ...item, key: `current-${index}` })),
    })
  }
  history.value.slice(0, 6).forEach((task, index) => {
    tasks.push({
      ...task,
      key: task.key || `history-${index}`,
      items: task.items.map((item, itemIndex) => ({ ...item, status: 'done', key: item.key || `history-item-${index}-${itemIndex}` })),
    })
  })
  return tasks
})

// ---------- Image handlers ----------
function handleOriginalImage(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const mimeType = file.type || 'image/png'
  const reader = new FileReader()
  reader.onload = (ev) => {
    const dataUrl = String(ev.target?.result || '')
    originalImage.value = { data: dataUrl.includes(',') ? dataUrl.split(',')[1] : dataUrl, mimeType, url: dataUrl }
  }
  reader.readAsDataURL(file)
  e.target.value = ''
}

function handleProductImage(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const mimeType = file.type || 'image/png'
  const reader = new FileReader()
  reader.onload = (ev) => {
    const dataUrl = String(ev.target?.result || '')
    productImage.value = { data: dataUrl.includes(',') ? dataUrl.split(',')[1] : dataUrl, mimeType, url: dataUrl }
  }
  reader.readAsDataURL(file)
  e.target.value = ''
}

function onUploadAreaClick(mode) {
  if (mode === 'original') {
    originalUploadInputWater.value?.click()
  } else {
    originalUploadInput.value?.click()
  }
}

function onRefDragOver(e) { if (originalImage.value) return; const types = e.dataTransfer?.types || []; if (types.includes('Files')) refDragOver.value = true }
function onRefDragLeave(e) { if (!e.currentTarget.contains(e.relatedTarget)) refDragOver.value = false }
function onRefDrop(e) { refDragOver.value = false; if (originalImage.value) return; const file = e.dataTransfer?.files?.[0]; if (file && file.type.startsWith('image/')) { const fake = { target: { files: [file] } }; handleOriginalImage(fake) } }

function onProductDragOver(e) { if (productImage.value) return; const types = e.dataTransfer?.types || []; if (types.includes('Files')) productDragOver.value = true }
function onProductDragLeave(e) { if (!e.currentTarget.contains(e.relatedTarget)) productDragOver.value = false }
function onProductDrop(e) { productDragOver.value = false; if (productImage.value) return; const file = e.dataTransfer?.files?.[0]; if (file && file.type.startsWith('image/')) { const fake = { target: { files: [file] } }; handleProductImage(fake) } }

// ---------- Generate ----------
const pollingTaskId = ref(null)
const pollingTimer = ref(null)
const POLLING_INTERVAL_MS = 2000
const MAX_POLLING_TIME_MS = 10 * 60 * 1000 // 10 minutes max

function isSaving(item) { return !!item?.id && savingIds.value.includes(item.id) }

async function saveToDesign(item) {
  if (!item?.id || !item.imageUrl || item.savedToDesign || isSaving(item)) return
  savingIds.value.push(item.id)
  try {
    await request.post(`/usage-logs/${item.id}/save`)
    item.savedToDesign = true
    toast.success('已保存到我的设计')
  } catch (error) {
    toast.error(error?.response?.data?.error || '保存失败，请稍后重试')
  } finally {
    savingIds.value = savingIds.value.filter(id => id !== item.id)
  }
}

async function saveTaskGroupToDesign(task) {
  const availableItems = task.items.filter(item => item.status === 'done' && item.id && !item.savedToDesign)
  if (!availableItems.length) return
  await Promise.allSettled(availableItems.map(item => saveToDesign(item)))
}

async function generate() {
  if (!canGenerate.value) return
  if (isGenerating.value) return
  const currentCount = count.value

  batchResults.value = []
  for (let i = 0; i < currentCount; i++) {
    batchResults.value.push({ id: null, prompt: '', imageUrl: '', status: 'generating', error: '', savedToDesign: false, key: `generate-${Date.now()}-${i}` })
  }
  isGenerating.value = true
  batchProgress.value = { current: 0, total: batchResults.value.length }
  globalError.value = ''

  // 启动每个任务的假进度
  batchResults.value.forEach((item, i) => {
    const taskId = item.key
    progressTrackers.value[taskId] = { current: 0 }
    fakeProgress.startProgress(taskId, (p) => {
      const tracker = progressTrackers.value[taskId]
      if (tracker) tracker.current = p
    })
  })

  try {
    const payload = {
      mode: mode.value,
      aspect_ratio: aspectRatio.value,
      count: count.value,
      quality: quality.value,
      prompt: '',
    }
    if (mode.value === 'water') {
      payload.water_effect_id = selectedWater.value.id
      payload.reference_image_1 = { data: originalImage.value.data, mime_type: originalImage.value.mimeType }
      payload.reference_image_2_url = selectedWater.value.thumbnail
    } else {
      payload.original_image = { data: originalImage.value.data, mime_type: originalImage.value.mimeType }
      payload.product_image = { data: productImage.value.data, mime_type: productImage.value.mimeType }
    }

    const res = await request.post('/product-replace/tasks', payload)
    const taskId = res.data?.taskId
    if (!taskId) {
      throw new Error('未获取到任务ID')
    }

    pollingTaskId.value = taskId
    startPolling()
  } catch (e) {
    // 清理所有进度追踪
    Object.keys(progressTrackers.value).forEach(key => {
      fakeProgress.completeProgress(key)
      delete progressTrackers.value[key]
    })
    batchResults.value.forEach((item, i) => {
      if (item.status === 'generating') {
        batchResults.value[i].status = 'error'
        batchResults.value[i].error = '提交任务失败，请稍后重试'
      }
    })
    isGenerating.value = false
  }
}

function startPolling() {
  if (pollingTimer.value) clearInterval(pollingTimer.value)

  let elapsed = 0
  pollingTimer.value = setInterval(async () => {
    elapsed += POLLING_INTERVAL_MS
    if (elapsed > MAX_POLLING_TIME_MS) {
      stopPolling()
      batchResults.value.forEach((item, i) => {
        if (item.status === 'generating') {
          batchResults.value[i].status = 'error'
          batchResults.value[i].error = '生成超时，请稍后重试'
        }
      })
      isGenerating.value = false
      return
    }

    try {
      const res = await request.get(`/product-replace/tasks/${pollingTaskId.value}`)
      const data = res.data

      if (!data || !data.images) return

      // 更新每个图片的结果
      data.images.forEach((img, i) => {
        if (i < batchResults.value.length) {
          const item = batchResults.value[i]
          if (img.image_url) {
            batchResults.value[i].imageUrl = resolveImageUrl(img.image_url)
            batchResults.value[i].status = 'done'
            // 停止进度追踪
            if (item.key && progressTrackers.value[item.key]) {
              fakeProgress.completeProgress(item.key)
              delete progressTrackers.value[item.key]
            }
          } else if (img.error) {
            batchResults.value[i].status = 'error'
            batchResults.value[i].error = img.error
            // 停止进度追踪
            if (item.key && progressTrackers.value[item.key]) {
              fakeProgress.completeProgress(item.key)
              delete progressTrackers.value[item.key]
            }
          }
        }
      })

      // 更新进度
      const doneCount = batchResults.value.filter(item => item.status === 'done').length
      batchProgress.value.current = doneCount

      // 检查是否全部完成
      const allDone = batchResults.value.every(item => item.status === 'done' || item.status === 'error')
      if (allDone || data.status === 'done' || data.status === 'failed') {
        stopPolling()
        saveToHistory()
      }
    } catch (e) {
      // 继续轮询，忽略单个请求错误
    }
  }, POLLING_INTERVAL_MS)
}

function stopPolling() {
  if (pollingTimer.value) {
    clearInterval(pollingTimer.value)
    pollingTimer.value = null
  }
  pollingTaskId.value = null
  isGenerating.value = false
}

function saveToHistory() {
  const doneItems = batchResults.value.filter(item => item.status === 'done' && item.imageUrl)
  if (doneItems.length > 0) {
    const modeName = mode.value === 'water' ? '水流替换' : '产品替换'
    history.value.unshift(normalizeHistoryTask({
      type: 'history', title: modeName, subtitle: `${batchResults.value.length} 张 · 比例 ${aspectRatio.value}`,
      cover: doneItems[0].imageUrl, total: batchResults.value.length,
      doneCount: doneItems.length, failedCount: batchResults.value.filter(item => item.status === 'error').length,
      createdAt: new Date().toISOString(),
      items: doneItems.map((item, index) => ({
        id: item.id || null, prompt: item.prompt || modeName, imageUrl: item.imageUrl,
        status: 'done', error: '', savedToDesign: !!item.savedToDesign, key: `history-${Date.now()}-${index}`,
      })),
    }, 0))
    if (history.value.length > 12) history.value = history.value.slice(0, 12)
  }
}

</script>

<style scoped>
.preview-footer-bar {
  background: rgba(17, 24, 39, .95);
  padding: 12px 16px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  border-radius: 18px;
  margin-top: 12px;
}

.preview-footer-prompt {
  font-size: 12px;
  color: #d1d5db;
  line-height: 1.6;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  flex: 1;
}

.preview-footer-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.preview-download-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: background .15s ease;
}

.preview-download-btn:hover { background: rgba(255, 255, 255, 0.2); }

.ratio-visual-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.ratio-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 124px;
  padding: 12px 10px;
  border: 1px solid #dddfe4;
  border-radius: 18px;
  background: #fff;
  color: #374151;
  cursor: pointer;
  transition: all .2s ease;
}

.ratio-card:hover { transform: translateY(-1px); box-shadow: 0 10px 22px rgba(15, 23, 42, .08); }
.ratio-card.on { background: #171717; border-color: #171717; color: #fff; box-shadow: none; }
.ratio-card-frame { display: flex; align-items: center; justify-content: center; width: 100%; min-height: 54px; }
.ratio-card-shape { display: block; border: 3px solid currentColor; border-radius: 7px; opacity: .76; }
.ratio-card.on .ratio-card-shape { opacity: 1; }
.ratio-card-label { font-size: 14px; font-weight: 500; letter-spacing: .01em; }
.ratio-shape-square { width: 34px; height: 34px; }
.ratio-shape-landscape { width: 46px; height: 34px; }
.ratio-shape-wide { width: 50px; height: 28px; }
.ratio-shape-cinema { width: 54px; height: 24px; }
.ratio-shape-portrait { width: 28px; height: 42px; }
.ratio-shape-tall { width: 24px; height: 46px; }
</style>
