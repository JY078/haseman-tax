<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
    <div class="mb-10">
      <h1 class="text-3xl sm:text-4xl font-bold text-neutral-900 mb-3">图片文字翻译</h1>
      <p class="text-lg text-neutral-600">上传包含文字的图片，将图片中的语言文字翻译为目标语言，保持原图风格</p>
      <!-- <div class="inline-flex items-center gap-1.5 mt-2 px-3 py-1.5 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-700">
        <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        生成的图片仅保存 3 天，请及时下载保存
      </div> -->
    </div>

    <div class="flex gap-6">
      <!-- 左侧：上传与设置 -->
      <div class="w-96 flex-shrink-0 space-y-4">
        <!-- 上传区域 -->
        <div class="border border-neutral-200 rounded-lg p-4">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-semibold text-neutral-900">上传图片</h3>
            <span class="text-xs text-neutral-400">{{ batchImages.length }}/8 张</span>
          </div>

          <div v-if="batchImages.length === 0" class="flex flex-col items-center justify-center min-h-[200px] gap-2">
            <label
              class="flex flex-col items-center justify-center w-full min-h-[180px] border-2 border-dashed rounded-lg cursor-pointer transition-colors"
              :class="isDragging ? 'border-neutral-900 bg-neutral-50' : 'border-neutral-300 hover:border-neutral-500'"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <svg class="w-12 h-12 mb-3 transition-colors" :class="isDragging ? 'text-neutral-700' : 'text-neutral-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span class="transition-colors" :class="isDragging ? 'text-neutral-700 font-medium' : 'text-neutral-500'">{{ isDragging ? '松开鼠标上传' : '点击或拖拽上传图片' }}</span>
              <span class="text-xs text-neutral-400 mt-1">支持包含文字的产品图、海报、截图等</span>
              <span class="text-xs text-neutral-400">文件大小 ≤ 50MB，每次最多 8 张</span>
              <input ref="fileInputRef" type="file" accept="image/*" multiple class="hidden" @change="handleFileChange" />
            </label>
            <div v-if="uploadError" class="flex items-center gap-1.5 w-full px-3 py-2 bg-red-50 border border-red-200 rounded-lg text-xs text-red-600">
              <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {{ uploadError }}
            </div>
          </div>

          <div v-else class="space-y-3">
            <!-- 批量图片网格 -->
            <div class="grid grid-cols-4 gap-2">
              <div
                v-for="(img, index) in batchImages"
                :key="index"
                class="relative aspect-square rounded-lg overflow-hidden border border-neutral-200 group"
              >
                <img :src="img.preview" class="w-full h-full object-cover" :alt="`图片 ${index + 1}`" />
                <button
                  class="absolute top-1 right-1 w-5 h-5 bg-red-500 text-white rounded-full flex items-center justify-center hover:bg-red-600 transition-colors shadow-sm opacity-0 group-hover:opacity-100"
                  @click="removeBatchImage(index)"
                >
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <!-- 添加更多按钮 -->
              <label
                v-if="batchImages.length < 8"
                class="aspect-square rounded-lg border-2 border-dashed border-neutral-300 hover:border-neutral-500 flex flex-col items-center justify-center cursor-pointer transition-colors"
              >
                <svg class="w-8 h-8 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
                <input ref="fileInputRef" type="file" accept="image/*" multiple class="hidden" @change="handleFileChange" />
              </label>
            </div>

            <div class="flex items-center justify-between">
              <button class="text-xs text-red-500 hover:text-red-600 transition-colors" @click="clearBatchImages">
                清空全部
              </button>
              <p class="text-xs text-neutral-400">{{ batchImages.length }} 张图片待翻译</p>
            </div>
          </div>
        </div>

        <!-- 目标语言 -->
        <div class="border border-neutral-200 rounded-lg p-4">
          <h3 class="text-sm font-semibold text-neutral-900 mb-3">目标语言</h3>

          <div class="relative">
            <button
              class="w-full flex items-center justify-between px-3 py-2.5 border border-neutral-200 rounded-lg hover:border-neutral-400 transition-colors"
              @click="showLanguageDropdown = !showLanguageDropdown"
            >
              <div class="flex items-center gap-2">
                <span class="text-lg">{{ currentLanguage?.flag }}</span>
                <span class="text-sm font-medium text-neutral-900">{{ currentLanguage?.name }}</span>
              </div>
              <svg class="w-4 h-4 text-neutral-400 transition-transform" :class="showLanguageDropdown ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            <Transition
              enter-active-class="transition-all duration-150 ease-out"
              enter-from-class="opacity-0 translate-y-1"
              enter-to-class="opacity-100 translate-y-0"
              leave-active-class="transition-all duration-100 ease-in"
              leave-from-class="opacity-100 translate-y-0"
              leave-to-class="opacity-0 translate-y-1"
            >
              <div v-if="showLanguageDropdown" class="absolute z-20 w-full mt-1 bg-white border border-neutral-200 rounded-lg shadow-lg overflow-hidden">
                <div class="max-h-60 overflow-y-auto">
                  <button
                    v-for="lang in languages"
                    :key="lang.code"
                    class="w-full flex items-center gap-2 px-3 py-2.5 hover:bg-neutral-50 transition-colors"
                    :class="params.targetLanguage === lang.code ? 'bg-neutral-100' : ''"
                    @click="selectLanguage(lang)"
                  >
                    <span class="text-lg">{{ lang.flag }}</span>
                    <span class="text-sm font-medium text-neutral-900">{{ lang.name }}</span>
                    <span class="text-xs text-neutral-400 ml-auto">{{ lang.nativeName }}</span>
                  </button>
                </div>
              </div>
            </Transition>
          </div>
        </div>

        <!-- 生图比例 -->
        <div class="border border-neutral-200 rounded-lg p-4">
          <h3 class="text-sm font-semibold text-neutral-900 mb-3">生图比例</h3>

          <div class="grid grid-cols-4 gap-2">
            <button
              v-for="ratio in aspectRatios"
              :key="ratio.value"
              class="group flex flex-col items-center gap-1.5 p-2.5 rounded-lg border-2 transition-all duration-200"
              :class="params.aspectRatio === ratio.value
                ? 'border-neutral-900 bg-neutral-900/5 shadow-sm'
                : 'border-neutral-200 hover:border-neutral-400 hover:bg-neutral-50'"
              @click="params.aspectRatio = ratio.value"
            >
              <!-- 比例图标 -->
              <div
                class="relative rounded flex items-center justify-center transition-all duration-200"
                :class="params.aspectRatio === ratio.value ? 'bg-neutral-900' : 'bg-neutral-300 group-hover:bg-neutral-400'"
                :style="getRatioStyle(ratio.value)"
              >
                <!-- 选中指示器 -->
                <div
                  v-if="params.aspectRatio === ratio.value"
                  class="absolute -top-1 -right-1 w-3 h-3 bg-neutral-900 rounded-full flex items-center justify-center"
                >
                  <svg class="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                </div>
              </div>
              <!-- 比例标签 -->
              <span
                class="text-xs font-semibold transition-colors duration-200"
                :class="params.aspectRatio === ratio.value ? 'text-neutral-900' : 'text-neutral-600 group-hover:text-neutral-800'"
              >
                {{ ratio.label }}
              </span>
            </button>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="space-y-3">
          <button
            class="w-full px-4 py-3 text-sm font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
            :class="canProcess ? 'bg-neutral-900 text-white hover:bg-neutral-800' : 'bg-neutral-200 text-neutral-400 cursor-not-allowed'"
            :disabled="!canProcess"
            @click="processImage"
          >
            <span v-if="processing" class="inline-flex items-center gap-2">
              <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              AI 翻译中，请稍候...
            </span>
            <span v-else class="flex items-center gap-2">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" />
              </svg>
              开始翻译
            </span>
          </button>

          <a
            v-if="batchResults.length > 0 && batchResults.length === batchImages.filter(i => i.resultUrl).length"
            href="#"
            download
            class="block w-full px-4 py-2.5 text-sm font-medium text-center rounded-lg border border-neutral-300 hover:bg-neutral-50 transition-colors flex items-center justify-center gap-2"
            @click.prevent="downloadAllResults"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            下载全部翻译结果
          </a>
        </div>
      </div>

      <!-- 右侧：结果预览 -->
      <div class="flex-1 min-w-0">
        <div class="border border-neutral-200 rounded-lg overflow-hidden">
          <!-- 空状态 -->
          <div v-if="batchImages.length === 0 && !processing" class="flex flex-col items-center justify-center min-h-[500px] text-neutral-400">
            <svg class="w-20 h-20 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" />
            </svg>
            <span class="font-medium text-neutral-500">上传图片并选择目标语言</span>
            <span class="text-sm mt-1">翻译结果将在此显示</span>
          </div>

          <!-- 有图片时显示结果网格 -->
          <div v-else class="flex flex-col">
            <div class="flex items-center justify-between px-4 py-3 bg-neutral-50 border-b border-neutral-200">
              <div class="flex items-center gap-3">
                <span class="text-sm font-medium text-neutral-700">翻译结果</span>
                <span v-if="processing" class="flex items-center gap-1.5 text-xs text-blue-600">
                  <svg class="w-3.5 h-3.5 animate-spin" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  处理中
                </span>
                <span v-else-if="batchResults.length > 0" class="text-xs text-green-600">
                  {{ batchResults.length }}/{{ batchImages.length }} 张完成
                </span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs text-neutral-500">
                  <span class="px-2 py-1 bg-neutral-200 rounded">{{ getLanguageName(params.targetLanguage) }}</span>
                  <span class="ml-1">{{ getRatioLabel(params.aspectRatio) }}</span>
                  <span class="ml-1">2K</span>
                </span>
              </div>
            </div>

            <!-- 批量结果网格 -->
            <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 p-4 bg-neutral-100">
              <div
                v-for="(img, index) in batchImages"
                :key="index"
                class="relative rounded-lg overflow-hidden border border-neutral-200 bg-white group"
              >
                <div class="aspect-square relative">
                  <!-- 有结果时显示翻译结果 -->
                  <img
                    v-if="img.resultUrl"
                    :src="img.resultUrl"
                    class="w-full h-full object-cover cursor-pointer"
                    alt="翻译结果"
                    @click="openPreview(img.resultUrl)"
                  />
                  <!-- 失败状态 -->
                  <div v-else-if="img.error" class="w-full h-full flex flex-col items-center justify-center bg-red-50">
                    <svg class="w-8 h-8 text-red-400 mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span class="text-xs text-red-500">翻译失败</span>
                  </div>
                  <!-- 处理中状态 -->
                  <div v-else-if="processing" class="w-full h-full flex flex-col items-center justify-center bg-neutral-50">
                    <img :src="img.preview" class="absolute inset-0 w-full h-full object-cover opacity-30" alt="原文" />
                    <div class="relative z-10 flex flex-col items-center">
                      <svg class="w-8 h-8 text-blue-500 animate-spin mb-1" viewBox="0 0 24 24">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                      </svg>
                      <span class="text-xs text-neutral-500">翻译中...</span>
                    </div>
                  </div>
                  <!-- 等待处理 -->
                  <div v-else class="w-full h-full flex items-center justify-center bg-neutral-50">
                    <img :src="img.preview" class="w-full h-full object-cover opacity-40" alt="原文" />
                    <span class="absolute text-xs text-neutral-400 bg-white/60 px-2 py-1 rounded">待处理</span>
                  </div>
                </div>
                <!-- 操作按钮 -->
                <div v-if="img.resultUrl" class="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    class="p-1.5 bg-white/90 text-neutral-700 rounded-lg hover:bg-white shadow-sm"
                    title="下载"
                    @click.stop="downloadImage(img.resultUrl, `translated-${index + 1}.png`)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                  </button>
                  <button
                    class="p-1.5 bg-white/90 text-neutral-700 rounded-lg hover:bg-white shadow-sm"
                    title="全屏预览"
                    @click.stop="openPreview(img.resultUrl)"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            <!-- 一键下载全部按钮 -->
            <div v-if="batchResults.length > 0 && !processing" class="border-t border-neutral-200 p-4 bg-neutral-50">
              <button
                class="w-full px-4 py-3 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors flex items-center justify-center gap-2"
                @click="downloadAllResults"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                下载全部翻译结果
              </button>
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
            <!-- 缩略图 -->
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
              <p class="text-sm font-medium text-neutral-800">图片翻译</p>
              <div class="flex items-center gap-2 mt-0.5">
                <p class="text-xs text-neutral-400">{{ formatTimeAgo(item.timestamp) }}</p>
                <span class="text-xs px-1.5 py-0.5 bg-neutral-100 text-neutral-500 rounded">
                  {{ getLanguageName(item.targetLanguage) }}
                </span>
                <span class="text-xs px-1.5 py-0.5 bg-neutral-100 text-neutral-500 rounded">
                  {{ getRatioLabel(item.aspectRatio) }}
                </span>
              </div>
            </div>

            <!-- 操作区 -->
            <div class="flex items-center gap-1.5 flex-shrink-0">
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

              <button
                class="w-8 h-8 flex items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700 transition-colors"
                :class="item.reProcessing ? 'cursor-not-allowed opacity-50' : ''"
                :disabled="item.reProcessing"
                title="重新翻译"
                @click="reProcessFromHistory(item)"
              >
                <svg v-if="item.reProcessing" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>

              <button
                class="w-8 h-8 flex items-center justify-center rounded-lg border border-neutral-200 text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700 transition-colors"
                title="下载"
                @click="downloadImage(item.resultImageUrl, `image-translate-${item.id}.png`)"
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
              <p class="text-[11px] font-semibold text-neutral-500 uppercase tracking-wide mb-2">翻译结果</p>
              <div class="group relative rounded-xl overflow-hidden border border-neutral-200 mb-4">
                <div class="flex items-center justify-center p-4 bg-neutral-100 min-h-[300px]">
                  <img :src="item.resultImageUrl" alt="翻译结果" class="max-w-full max-h-[350px] object-contain rounded shadow-sm cursor-pointer hover:opacity-90 transition-opacity" @click="openPreview(item.resultImageUrl)" />
                </div>
                <div class="absolute inset-0 bg-black/0 group-hover:bg-black/25 transition-all duration-200 flex items-center justify-center gap-3">
                  <button
                    class="opacity-0 group-hover:opacity-100 transition-opacity px-3.5 py-2 bg-white text-neutral-900 text-xs font-medium rounded-xl flex items-center gap-1.5 shadow-lg hover:bg-neutral-50"
                    :class="item.reProcessing ? 'cursor-not-allowed opacity-50' : ''"
                    :disabled="item.reProcessing"
                    @click="reProcessFromHistory(item)"
                  >
                    <svg v-if="item.reProcessing" class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    {{ item.reProcessing ? '处理中...' : '重新翻译' }}
                  </button>
                  <button
                    class="opacity-0 group-hover:opacity-100 transition-opacity px-3.5 py-2 bg-white text-neutral-900 text-xs font-medium rounded-xl flex items-center gap-1.5 shadow-lg hover:bg-neutral-50"
                    @click="openPreview(item.resultImageUrl)"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                    </svg>
                    全屏预览
                  </button>
                  <button
                    class="opacity-0 group-hover:opacity-100 transition-opacity px-3.5 py-2 bg-white text-neutral-900 text-xs font-medium rounded-xl flex items-center gap-1.5 shadow-lg hover:bg-neutral-50"
                    @click="downloadImage(item.resultImageUrl, `image-translate-${item.id}.png`)"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    下载图片
                  </button>
                </div>
              </div>

              <!-- 原图与结果对比 -->
              <p class="text-[11px] font-semibold text-neutral-500 uppercase tracking-wide mb-2">翻译前后对比</p>
              <div class="grid grid-cols-2 gap-3">
                <div class="border border-neutral-200 rounded-xl overflow-hidden bg-white">
                  <div class="px-3 py-1.5 bg-neutral-50 border-b border-neutral-100 text-center">
                    <span class="text-[11px] font-semibold text-neutral-500">原图</span>
                  </div>
                  <div class="p-3 flex items-center justify-center min-h-[200px] bg-neutral-50">
                    <img :src="item.preview || item.thumbnail" alt="原图" class="max-w-full max-h-[250px] object-contain" />
                  </div>
                </div>
                <div class="border border-neutral-200 rounded-xl overflow-hidden bg-white">
                  <div class="px-3 py-1.5 bg-neutral-50 border-b border-neutral-100 text-center">
                    <span class="text-[11px] font-semibold text-neutral-500">翻译后 - {{ getLanguageName(item.targetLanguage) }}</span>
                  </div>
                  <div class="p-3 flex items-center justify-center min-h-[200px] bg-neutral-50">
                    <img :src="item.resultImageUrl" alt="翻译后" class="max-w-full max-h-[250px] object-contain" />
                  </div>
                </div>
              </div>
            </div>
          </Transition>

        </div>
      </div>
    </div>

    <!-- 全屏预览模态框 -->
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="previewModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/90 p-4"
        @click="closePreview"
      >
        <div class="relative max-w-[90vw] max-h-[90vh]">
          <button
            class="absolute -top-10 right-0 p-2 text-white/70 hover:text-white transition-colors"
            @click="closePreview"
          >
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
          <img
            :src="previewImage"
            class="max-w-full max-h-[90vh] object-contain rounded-lg"
            alt="全屏预览"
            @click.stop
          />
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import request, { downloadImage, resolveImageUrl } from '@/utils/request'

const HISTORY_KEY = 'image_translate_history'
const HISTORY_TTL = 30 * 60 * 1000
const HISTORY_MAX = 8
const MAX_BATCH_SIZE = 8
const MAX_FILE_SIZE = 50 * 1024 * 1024

const processing = ref(false)
const uploadError = ref('')
const isDragging = ref(false)
const showLanguageDropdown = ref(false)
const fileInputRef = ref(null)
const previewModal = ref(false)
const previewImage = ref('')

const batchImages = ref([])
const batchResults = ref([])

const history = ref([])

const params = reactive({
  targetLanguage: 'zh',
  aspectRatio: '1:1',
})

function buildTranslatePrompt(language, localAudience) {
  return `Localize only the text already visible in this ecommerce image into natural ${language} ecommerce advertising copy for ${localAudience}. Do not translate word by word: rewrite each existing text block as fluent, concise, benefit-focused product-listing or promotional image copy while preserving the original selling point and intent. Keep every text block short enough to fit its original area. Do not add, invent, duplicate, summarize, or remove any text blocks. Do not add captions, slogans, labels, badges, watermarks, icons, or decorative text outside the original text areas. If a text block is unclear, keep that region visually unchanged. Preserve the original layout, text positions, font sizes, colors, styles, hierarchy, spacing, products, background, and all non-text image content.`
}

const languages = [
  { code: 'zh', name: '简体中文', nativeName: 'Chinese', flag: '🇨🇳', prompt: buildTranslatePrompt('Simplified Chinese', 'native Chinese speakers') },
  { code: 'en', name: 'English', nativeName: '英语', flag: '🇺🇸', prompt: buildTranslatePrompt('English', 'native English speakers') },
  { code: 'de', name: 'Deutsch', nativeName: '德语', flag: '🇩🇪', prompt: buildTranslatePrompt('German', 'native German speakers') },
  { code: 'fr', name: 'Français', nativeName: '法语', flag: '🇫🇷', prompt: buildTranslatePrompt('French', 'native French speakers') },
  { code: 'it', name: 'Italiano', nativeName: '意大利语', flag: '🇮🇹', prompt: buildTranslatePrompt('Italian', 'native Italian speakers') },
  { code: 'es', name: 'Español', nativeName: '西班牙语', flag: '🇪🇸', prompt: buildTranslatePrompt('Spanish', 'native Spanish speakers') },
  { code: 'nl', name: 'Nederlands', nativeName: '荷兰语', flag: '🇳🇱', prompt: buildTranslatePrompt('Dutch', 'native Dutch speakers') },
  { code: 'pl', name: 'Polski', nativeName: '波兰语', flag: '🇵🇱', prompt: buildTranslatePrompt('Polish', 'native Polish speakers') },
  { code: 'sv', name: 'Svenska', nativeName: '瑞典语', flag: '🇸🇪', prompt: buildTranslatePrompt('Swedish', 'native Swedish speakers') },
]

const aspectRatios = [
  { value: '1:1', label: '1:1' },
  { value: '4:3', label: '4:3' },
  { value: '16:9', label: '16:9' },
  { value: '9:16', label: '9:16' },
  { value: '3:4', label: '3:4' },
  { value: '4:5', label: '4:5' },
  { value: '5:4', label: '5:4' },
]

const currentLanguage = computed(() => languages.find(l => l.code === params.targetLanguage))

const canProcess = computed(() => batchImages.value.length > 0 && !processing.value)

function getLanguageName(code) {
  return languages.find(l => l.code === code)?.name || code
}

function getRatioLabel(value) {
  return aspectRatios.find(r => r.value === value)?.label || value
}

function getRatioStyle(value) {
  const [w, h] = value.split(':').map(Number)
  const maxW = 28
  const maxH = 22
  const aspect = w / h

  let width, height
  if (aspect >= 1) {
    width = maxW
    height = Math.round(maxW / aspect)
  } else {
    height = maxH
    width = Math.round(maxH * aspect)
  }

  return { width: `${width}px`, height: `${height}px`, minWidth: `${width}px`, minHeight: `${height}px` }
}

function selectLanguage(lang) {
  params.targetLanguage = lang.code
  showLanguageDropdown.value = false
}

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
    const valid = items.filter(item => Date.now() - (item.timestamp || 0) <= HISTORY_TTL)
    if (valid.length === 0) {
      localStorage.removeItem(HISTORY_KEY)
      return []
    }
    return valid.map(item => ({ expanded: false, ...item, reProcessing: false }))
  } catch { return [] }
}

function saveHistory(items) {
  try {
    const toSave = items.map(({ expanded, reProcessing, ...rest }) => rest)
    localStorage.setItem(HISTORY_KEY, JSON.stringify({ items: toSave, savedAt: Date.now() }))
  } catch { /** ignore quota errors */ }
}

watch(history, (val) => saveHistory(val), { deep: true })

async function addToHistory(resultUrl, originalImage) {
  const thumbnail = await createThumbnail(originalImage || batchImages.value[0]?.preview)
  const preview = await createPreview(originalImage || batchImages.value[0]?.preview)
  const item = {
    id: Date.now() + Math.random(),
    thumbnail,
    preview,
    resultImageUrl: resultUrl,
    targetLanguage: params.targetLanguage,
    aspectRatio: params.aspectRatio,
    prompt: currentLanguage.value?.prompt || '',
    timestamp: Date.now(),
    expanded: false,
    reProcessing: false,
  }
  history.value.unshift(item)
  if (history.value.length > HISTORY_MAX) history.value.splice(HISTORY_MAX)
}

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

async function reProcessFromHistory(item) {
  if (item.reProcessing) return
  item.reProcessing = true

  try {
    const { imageData, mimeType } = await fetchImageAsBase64(item.resultImageUrl)
    const lang = languages.find(l => l.code === item.targetLanguage)
    const prompt = lang?.prompt || ''
    const res = await request.post('/image-translate', {
      image_data: imageData,
      mime_type: mimeType,
      target_language: item.targetLanguage,
      aspect_ratio: item.aspectRatio,
      prompt: prompt,
      resolution: '2K',
    })

    if (res.data.image_url) {
      item.resultImageUrl = res.data.image_url
      item.timestamp = Date.now()
      item.expanded = true
    }
  } catch (err) {
    alert('本次未能获取到结果，请稍后重试')
  } finally {
    item.reProcessing = false
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

async function downloadAllResults() {
  for (let i = 0; i < batchImages.value.length; i++) {
    const img = batchImages.value[i]
    if (img.resultUrl) {
      await new Promise(resolve => setTimeout(resolve, 300))
      downloadImage(img.resultUrl, `translated-${i + 1}.png`)
    }
  }
}

function openPreview(url) {
  previewImage.value = url
  previewModal.value = true
  document.body.style.overflow = 'hidden'
}

function closePreview() {
  previewModal.value = false
  previewImage.value = ''
  document.body.style.overflow = ''
}

onMounted(() => {
  history.value = loadHistory()
  document.addEventListener('click', handleOutsideClick)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleOutsideClick)
  document.removeEventListener('keydown', handleKeydown)
})

function handleOutsideClick(e) {
  if (showLanguageDropdown.value && !e.target.closest('.relative')) {
    showLanguageDropdown.value = false
  }
}

function handleKeydown(e) {
  if (e.key === 'Escape' && previewModal.value) {
    closePreview()
  }
}

function handleDrop(e) {
  isDragging.value = false
  processFiles(e.dataTransfer?.files)
}

function handleFileChange(e) {
  processFiles(e.target.files)
}

function processFiles(files) {
  if (!files || files.length === 0) return

  uploadError.value = ''

  const filesToAdd = Array.from(files).filter(file => {
    if (file.size > MAX_FILE_SIZE) {
      uploadError.value = `文件过大（${formatBytes(file.size)}），请上传 50MB 以内的图片`
      return false
    }
    if (!file.type.startsWith('image/')) {
      uploadError.value = '请上传图片文件'
      return false
    }
    return true
  })

  const remaining = MAX_BATCH_SIZE - batchImages.value.length
  if (filesToAdd.length > remaining) {
    uploadError.value = `最多只能添加 ${MAX_BATCH_SIZE} 张图片，已选择前 ${remaining} 张`
    filesToAdd.splice(remaining)
  }

  filesToAdd.forEach(file => {
    const reader = new FileReader()
    reader.onload = async (ev) => {
      const dataUrl = ev.target.result
      const preview = await createPreview(dataUrl)
      batchImages.value.push({
        file,
        dataUrl,
        data: dataUrl.split(',')[1],
        mimeType: file.type || 'image/png',
        preview,
        processing: false,
        resultUrl: '',
        resultMimeType: '',
        error: '',
      })
    }
    reader.readAsDataURL(file)
  })

  if (fileInputRef.value) fileInputRef.value.value = ''
}

function removeBatchImage(index) {
  batchImages.value.splice(index, 1)
}

function clearBatchImages() {
  batchImages.value = []
  batchResults.value = []
  uploadError.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

function clearImage() {
  clearBatchImages()
}

async function processImage() {
  if (batchImages.value.length === 0) {
    alert('请先上传图片')
    return
  }

  processing.value = true

  try {
    const prompt = currentLanguage.value?.prompt || ''

    // 并行发送所有请求，实时更新结果
    const promises = batchImages.value.map((img, i) => {
      return request.post('/image-translate', {
        image_data: img.data,
        mime_type: img.mimeType,
        target_language: params.targetLanguage,
        aspect_ratio: params.aspectRatio,
        prompt: prompt,
        resolution: '2K',
      }).then(res => {
        // 立即更新当前图片结果
        img.resultUrl = res.data.image_url
        img.resultMimeType = res.data.mime_type || 'image/png'
        img.error = ''
        batchResults.value.push({ originalIndex: i, resultUrl: res.data.image_url })
        return { index: i, success: true, data: res.data, error: null }
      }).catch(err => {
        img.error = err.message || '翻译失败'
        return { index: i, success: false, data: null, error: err.message || '翻译失败' }
      })
    })

    // 等待所有请求完成
    await Promise.all(promises)

    // 添加到历史记录
    for (const result of batchResults.value) {
      const img = batchImages.value[result.originalIndex]
      if (img && img.preview) {
        await addToHistory(result.resultUrl, img.preview)
      }
    }
  } catch (err) {
    alert('本次未能获取到结果，请稍后重试')
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
</style>
