<template>
  <MainLayout>
    <section
      class="mx-2 grid h-[calc(100vh-92px)] gap-[18px] py-4 max-lg:h-auto max-lg:grid-cols-1 max-lg:overflow-visible"
      :style="{ gridTemplateColumns: 'min(29%, 360px) 1fr' }">
      <div
        class="sticky top-[18px] flex max-h-[calc(100vh-110px)] flex-col overflow-hidden rounded-[30px] border border-neutral-200 bg-white shadow-[0_18px_50px_rgba(15,23,42,0.05)] max-lg:static max-lg:max-h-none">
        <div class="flex-1 min-h-0 overflow-y-auto p-5 space-y-4 max-lg:overflow-visible">

          <!-- Reference Images -->
          <div class="bg-white border rounded-xl p-5 transition-all duration-200" :class="refDragOver
            ? 'border-neutral-900 bg-neutral-50 ring-2 ring-neutral-900/10'
            : 'border-neutral-200'" @dragover.prevent="onRefDragOver" @dragleave="onRefDragLeave"
            @drop.prevent="onRefDrop">
            <div class="flex items-center justify-between mb-3">
              <div>
                <label class="text-sm font-semibold text-neutral-900">参考图片</label>
              </div>
              <div class="px-2.5 py-1 rounded-full bg-neutral-100 text-xs font-medium text-neutral-600">
                {{ refImages.length }}/6
              </div>
            </div>

            <div v-if="refImages.length < 6"
              class="relative flex min-h-[148px] w-full cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-4 py-6 text-center transition-all"
              :class="refDragOver
                ? 'border-neutral-900 bg-neutral-100 ring-4 ring-neutral-900/10 scale-[1.02]'
                : 'border-neutral-300 bg-neutral-50 hover:border-neutral-500 hover:bg-white'"
              @click="onUploadAreaClick" @dragover.stop.prevent="onRefDragOver" @dragleave.stop="onRefDragLeave"
              @drop.stop.prevent="onRefDrop">
              <div
                class="flex h-12 w-12 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-black/5">
                <PhotoIcon class="w-6 h-6 transition-colors"
                  :class="refDragOver ? 'text-neutral-800' : 'text-neutral-500'" />
              </div>
              <div class="mt-4">
                <p class="text-sm font-medium text-neutral-800">
                  <template v-if="refDragOver">松开即可添加</template>
                  <template v-else>
                    从
                    <button type="button" data-design-library-trigger="true"
                      class="font-medium text-emerald-600 transition-colors hover:text-emerald-700 hover:underline"
                      @click.stop.prevent="openDesignLibrary">资产库选择参考图</button>
                    或上传图片
                  </template>
                </p>
              </div>
              <!-- <div class="mt-4 inline-flex items-center gap-2 rounded-full border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-600 shadow-sm">
                <span>JPG / PNG / WEBP</span>
                <span class="text-neutral-300">•</span>
                <span>可多选</span>
              </div> -->
              <input ref="refUploadInput" type="file" accept="image/*" multiple class="hidden"
                @change="handleRefImage" />
            </div>

            <div v-if="refImages.length > 0" class="mt-4">
              <div class="grid grid-cols-3 gap-3">
                <div v-for="(img, idx) in refImages" :key="idx"
                  class="group relative overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 aspect-[4/3] transition-all hover:border-neutral-300 hover:shadow-sm">
                  <img :src="img.url" class="w-full h-full object-contain bg-white p-1.5" @contextmenu.prevent.stop />
                  <div
                    class="absolute left-2 top-2 inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-black/60 px-1.5 text-[10px] font-bold text-white">
                    {{ idx + 1 }}
                  </div>
                  <!-- 编辑按钮 -->
                  <button
                    class="absolute right-10 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md opacity-0 transition-all group-hover:opacity-100 hover:bg-white"
                    title="编辑蒙版"
                    @click="openRefImageEdit(idx)">
                    <PencilIcon class="w-3.5 h-3.5" />
                  </button>
                  <!-- 删除按钮 -->
                  <button
                    class="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-white/92 text-neutral-700 shadow-md opacity-0 transition-all group-hover:opacity-100 hover:bg-white"
                    @click="removeRefImage(idx)">
                    <XMarkIcon class="w-3.5 h-3.5" />
                  </button>
                  <!-- 蒙版标记 -->
                  <div v-if="img.mask"
                    class="absolute left-2 bottom-2 px-1.5 py-0.5 rounded bg-emerald-500/90 text-[9px] font-bold text-white">
                    已编辑
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Prompt -->
          <div class="prompt-panel bg-white border border-neutral-200 rounded-xl p-5">
            <div class="flex items-center justify-between mb-3">
              <label class="text-sm font-semibold text-neutral-900">
                提示词
                <span class="text-red-400 ml-0.5">*</span>
                <span v-if="count > 1" class="text-xs font-normal text-neutral-400 ml-1">（所有图片共用）</span>
              </label>
              <button
                class="p-1 rounded-md hover:bg-neutral-100 transition-colors text-neutral-400 hover:text-neutral-600"
                title="全屏编辑" @click="promptFullscreen = true">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M4 8V4m0 0h4M4 4l5 5m11-5h-4m4 0v4m0-4l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                </svg>
              </button>
            </div>
            <textarea v-model="prompt" placeholder="描述你想生成的图片" rows="8"
              class="prompt-textarea w-full text-sm border border-neutral-200 rounded-lg px-3 py-3 resize-y bg-neutral-50 focus:outline-none focus:ring-2 focus:ring-neutral-900 focus:border-transparent transition-all placeholder:text-neutral-400 leading-relaxed" />
            <div class="flex items-center justify-between mt-2">
              <span class="text-xs text-neutral-400">{{ prompt.length }} 字</span>
              <button v-if="prompt" class="text-xs text-neutral-400 hover:text-neutral-600 transition-colors"
                @click="prompt = ''">清空</button>
            </div>
          </div>
          <div class="generate-params-panel">
            <div class="generate-params-heading"><div><span>生成参数</span><small>控制输出规格</small></div></div>
            <div v-if="drawingModels.length > 0" class="generate-model-picker">
              <button type="button" class="generate-model-trigger" :aria-expanded="modelMenuOpen" @click.stop="modelMenuOpen = !modelMenuOpen">
                <span class="generate-model-trigger-text">{{ selectedModel || '选择模型' }}</span>
                <svg class="generate-model-chevron" :class="{ open: modelMenuOpen }" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.51a.75.75 0 01-1.08 0l-4.25-4.51a.75.75 0 01.02-1.06z" clip-rule="evenodd" /></svg>
              </button>
              <div v-if="modelMenuOpen" class="generate-model-menu">
                <button v-for="model in drawingModels" :key="model.id" type="button" class="generate-model-option" :class="{ active: selectedModel === model.name }" @click="selectedModel = model.name; modelMenuOpen = false">{{ model.name }}</button>
              </div>
            </div>
            <div class="generate-params-row">
              <div class="generate-param-group"><span class="generate-param-label">{{ autoMatchResolution ? '生成分辨率' : (sizeMode === 'custom' && selectedModel.toLowerCase().includes('gpt-image') ? '画质档位' : '清晰度') }}</span><div v-if="autoMatchResolution" class="generate-auto-resolution">自动匹配 <strong>{{ customGenerationSize?.resolution || '—' }}</strong></div><div v-else class="generate-chip-row"><button v-for="r in supportedResolutions" :key="r" type="button" class="generate-chip" :class="{ active: resolution === r }" @click="resolution = String(r).toUpperCase()">{{ String(r).toUpperCase() }}</button></div></div>
              <div class="generate-param-group"><span class="generate-param-label">生成数量</span><div class="generate-chip-row generate-count-grid"><button v-for="n in 4" :key="n" type="button" class="generate-chip generate-chip--count" :class="{ active: count === n }" @click="count = n">{{ n }}张</button></div></div>
            </div>
            <div class="generate-size-section">
              <div class="generate-size-heading"><span class="generate-param-label">输出尺寸</span><span class="generate-size-unit">PX</span></div>
              <div class="generate-size-tabs" role="group" aria-label="输出尺寸模式">
                <button type="button" :aria-pressed="sizeMode === 'preset'" :class="{ active: sizeMode === 'preset' }" @click="sizeMode = 'preset'">预设比例</button>
                <button type="button" :aria-pressed="sizeMode === 'custom'" :class="{ active: sizeMode === 'custom' }" @click="sizeMode = 'custom'">自定义尺寸</button>
              </div>
              <div v-if="sizeMode === 'custom'" class="generate-custom-size">
                <div class="generate-size-inputs">
                  <label for="output-width">宽度<div class="generate-size-field"><input id="output-width" v-model="customWidth" type="number" inputmode="numeric" min="64" max="8192" step="1" :aria-invalid="!!sizeError" :aria-describedby="sizeError ? 'custom-size-help' : undefined" /><span>px</span></div></label>
                  <span class="generate-size-separator" aria-hidden="true">×</span>
                  <label for="output-height">高度<div class="generate-size-field"><input id="output-height" v-model="customHeight" type="number" inputmode="numeric" min="64" max="8192" step="1" :aria-invalid="!!sizeError" :aria-describedby="sizeError ? 'custom-size-help' : undefined" /><span>px</span></div></label>
                </div>
                <p v-if="sizeError" id="custom-size-help" class="generate-size-error" role="alert">{{ sizeError }}</p>
              </div>
            </div>
            <div v-if="sizeMode === 'preset'" class="generate-param-group"><span class="generate-param-label">图片比例</span><div class="ratio-visual-grid ratio-visual-grid--compact"><button v-for="item in ratioOptionsForCard" :key="item.value" type="button" :disabled="item.value === 'original' && !refImages.length" :class="['ratio-card', aspectRatio === item.value && 'on', item.value === 'original' && !refImages.length && 'opacity-40 cursor-not-allowed']" @click="aspectRatio = item.value"><span class="ratio-card-frame"><span class="ratio-card-shape" :style="ratioShapeStyle(item.value === 'original' ? '4:3' : item.value)"></span></span><span class="ratio-card-label">{{ item.label }}</span></button></div><p v-if="aspectRatio === 'original'" class="mt-2 text-xs text-neutral-500">跟随第一张参考图；非标准比例将匹配模型支持的最接近比例，实际比例显示在任务中。</p></div>
          </div>
        </div>

        <div
          class="shrink-0 flex flex-col items-stretch gap-3 border-t border-neutral-200 bg-white px-6 pb-5 pt-[18px] shadow-[0_-10px_18px_-18px_rgba(15,23,42,0.12)]">
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
          class="z-10 m-0 flex flex-wrap items-center justify-between gap-3 border-b border-[#eef2f7] bg-white px-5 pb-2.5 pt-3 shadow-[0_10px_18px_-18px_rgba(15,23,42,0.18)]">
          <div>
            <h2 class="m-0 text-[18px] font-bold tracking-[0.01em] text-neutral-900">最近任务</h2>
            <span class="mt-1 block text-[11px] leading-[1.5] text-neutral-400">生成中的任务与已完成任务都会展示在这里</span>
          </div>
          <div class="ml-auto flex shrink-0 items-center justify-end gap-3">
          <button type="button" class="inline-flex h-9 shrink-0 items-center gap-2 rounded-full border border-neutral-200 bg-neutral-50 px-3.5 text-sm font-medium text-neutral-700 transition duration-150 hover:border-neutral-400 hover:bg-white hover:text-neutral-900 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900 focus-visible:ring-offset-2" aria-haspopup="dialog" :aria-expanded="violationVisible" @click="violationVisible = true">
            <TableCellsIcon class="h-4 w-4" aria-hidden="true" />
            <span>违规图片示例</span>
          </button>
          <button
            class="h-9 shrink-0 rounded-full border border-neutral-200 bg-white px-3.5 text-sm font-semibold text-neutral-800 transition-all hover:-translate-y-px hover:border-neutral-300 hover:shadow-[0_10px_22px_rgba(15,23,42,0.08)] disabled:cursor-not-allowed disabled:opacity-40"
            type="button" :disabled="!batchDownloadItems.length" @click="toggleBatchDownloadMode">
            {{ batchDownloadMode ? '取消选择' : '批量下载' }}
          </button>
          </div>
          <!-- <button v-if="hasAnyResult && !isGenerating"
            class="h-9 shrink-0 rounded-full bg-neutral-100 px-3.5 text-sm font-medium text-neutral-900 transition-all hover:-translate-y-px hover:shadow-[0_10px_22px_rgba(15,23,42,0.08)]"
            type="button" @click="generate">重新生成</button> -->
        </div>

        <div class="flex-1 overflow-auto overflow-x-hidden pb-2 max-lg:overflow-visible">
          <template v-if="displayTasks.length">
            <div class="flex min-h-0 flex-1 flex-col gap-2 px-[14px] pb-3 pt-2.5">
              <section v-for="(task, taskIndex) in displayTasks" :key="task.key"
                class="task-card overflow-hidden rounded-2xl border border-neutral-200 bg-[linear-gradient(180deg,#fff,#fcfcfd)] p-2.5 transition-[box-shadow,border-color] hover:border-neutral-300 hover:shadow-[0_10px_24px_rgba(15,23,42,0.05)]">
                <div class="flex items-start justify-between gap-2 border-b border-[#eef2f7] pb-1.5">
                  <div class="flex min-w-0 items-center gap-2">
                    <div v-if="task.referenceImages.length"
                      class="grid max-w-[180px] shrink-0 gap-1"
                      :class="{
                        'grid-cols-1': task.referenceImages.length === 1,
                        'grid-cols-2': task.referenceImages.length === 2,
                        'grid-cols-3': task.referenceImages.length >= 3,
                      }">
                      <img v-for="(referenceImage, referenceIndex) in task.referenceImages"
                        :key="`${task.key}-reference-${referenceIndex}-${referenceImage}`"
                        :src="getImageThumbnailUrl(referenceImage, 160)"
                        :alt="`${task.title}参考图${referenceIndex + 1}`"
                        @contextmenu.prevent.stop
                        class="h-[54px] w-[54px] rounded-[14px] bg-neutral-100 object-cover" loading="lazy"
                        decoding="async" fetchpriority="low" />
                    </div>
                    <img v-else :key="`${task.key}-cover-${task.cover}`"
                      :src="getImageThumbnailUrl(task.cover, 160)" :alt="task.title"
                      @contextmenu.prevent.stop
                      class="h-[54px] w-[54px] rounded-[14px] bg-neutral-100 object-cover" loading="lazy"
                      decoding="async" fetchpriority="low" />
                    <div class="min-w-0">
                      <strong class="block truncate text-[13px] leading-[1.25] text-neutral-900">{{ task.title
                      }}</strong>
                      <span class="mt-0.5 block truncate text-[11px] leading-[1.3] text-neutral-500">{{
                        task.subtitle }}</span>
                    </div>
                  </div>
                  <div class="flex min-w-[84px] shrink-0 flex-col items-end gap-[3px] text-right">
                    <div class="flex items-center justify-end gap-1.5">
                      <span v-if="task.hasInProgress"
                        class="inline-flex min-h-6 items-center rounded-full bg-indigo-50 px-[9px] text-[11px] font-semibold text-indigo-700">{{ task.submissionStatus || '生成中' }}</span>
                      <span v-else
                        class="inline-flex min-h-6 items-center rounded-full px-[9px] text-[11px] font-semibold"
                        :class="task.failedCount ? 'bg-rose-50 text-rose-700' : 'bg-emerald-50 text-emerald-700'">
                        {{ task.failedCount ? `完成 ${task.doneCount}/${task.total}` : '已完成' }}
                      </span>
                    </div>
                    <span class="whitespace-nowrap text-[11px] font-semibold text-neutral-400">{{ task.timeText
                    }}</span>
                    <!-- <div v-if="task.items.some(item => item.status === 'done')" class="mt-0.5 flex justify-end">
                      <button
                        class="border-none bg-transparent p-0 text-xs font-semibold text-neutral-900 transition hover:underline"
                        type="button" @click="saveTaskGroupToDesign(task)">保存本批可用结果</button>
                    </div> -->
                  </div>
                </div>

                <div class="mt-2 grid items-start"
                  :class="[
                    task.items.length > 1 ? 'gap-0.5' : 'gap-1.5',
                    task.items.length === 1 ? 'grid-cols-[minmax(0,340px)] max-lg:grid-cols-1' : 'grid-cols-4 max-xl:grid-cols-3 max-lg:grid-cols-2 max-sm:grid-cols-1'
                  ]">
                  <div v-for="(item, index) in task.items" :key="`${task.key}-${item.key || item.id || index}`" class="relative min-w-0">
                    <div
                      class="result-image-tile group relative flex items-center justify-center overflow-hidden rounded-[14px] bg-neutral-100 shadow-[inset_0_0_0_1px_rgba(229,231,235,.8)]"
                      :class="{ 'batch-selected': batchDownloadSelectedIds.includes(`${task.key}-${item.key || item.id || index}`), 'result-image-tile--square': task.items.length === 1 && normalizeBatchAspectRatio(task.aspectRatio) === '1:1', 'batch-pickable': batchDownloadMode && item.status === 'done' && item.imageUrl }"
                      @click="batchDownloadMode && item.status === 'done' && item.imageUrl ? toggleBatchDownloadItem(`${task.key}-${item.key || item.id || index}`) : undefined"
                      :style="{ aspectRatio: ratioToCss(task.aspectRatio) }">
                      <template v-if="item.status === 'done' && item.imageUrl">
                        <img :key="`${task.key}-${item.key || item.id || index}-${item.imageUrl}`"
                          :src="getImageThumbnailUrl(item.imageUrl, 640)" :alt="getItemAlt(item, index)"
                          @contextmenu.prevent.stop
                          class="h-full w-full object-contain bg-white" loading="lazy" decoding="async"
                          fetchpriority="low" />
                        <!-- 编辑效果生成中遮罩 -->
                        <div v-if="item.candidateStatus === 'generating'"
                          class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2.5 bg-black/50 p-3.5 text-center">
                          <span class="relative inline-flex h-[42px] w-[42px] items-center justify-center">
                            <span
                              class="h-[42px] w-[42px] animate-spin rounded-full border-[3px] border-white/30 border-t-white"></span>
                            <span class="absolute h-3.5 w-3.5 animate-pulse rounded-full bg-white/40"></span>
                          </span>
                          <p class="m-0 text-[13px] font-semibold text-white">编辑效果生成中</p>
                          <span class="text-[11px] leading-[1.4] text-white/70">可继续处理其他任务</span>
                        </div>
                        <!-- 编辑结果待确认遮罩 -->
                        <div v-else-if="item.candidateStatus === 'ready'"
                          class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-3 bg-black/50 p-3.5 text-center pointer-events-auto">
                          <p class="m-0 text-[13px] font-semibold text-white">有新的编辑结果待确认</p>
                          <button
                            class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold text-neutral-900 transition hover:bg-white"
                            type="button" @click="openCandidateModal(item)">查看对比</button>
                        </div>
                        <!-- 编辑失败遮罩 -->
                        <div v-else-if="item.candidateStatus === 'failed'"
                          class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2.5 bg-[rgba(127,29,29,.78)] p-3.5 text-center text-white">
                          <p class="m-0 text-[13px] font-semibold">编辑预览生成失败</p>
                          <span class="text-[11px] leading-[1.4] text-white/80">{{ item.candidateErrorMessage ||
                            '请重新发起编辑' }}</span>
                        </div>
                      </template>
                      <div v-else-if="item.status === 'error'"
                        class="absolute inset-0 z-[2] flex flex-col items-center justify-center gap-2.5 bg-[rgba(127,29,29,.78)] p-3.5 text-center text-white">
                        <ExclamationCircleIcon class="w-7 h-7" />
                        <p class="m-0 text-[13px] font-semibold leading-[1.5]">生成失败</p>
                        <span class="text-[11px] leading-[1.4] text-white/80">{{ item.error || '请稍后重试' }}</span>
                      </div>
                      <div v-else
                        class="flex flex-col items-center justify-center gap-2 p-3 text-center text-[11px] text-neutral-500">
                        <!-- 进度条 -->
                        <div class="w-full px-2">
                          <div class="relative h-2 w-full overflow-hidden rounded-full bg-neutral-200">
                            <div class="absolute left-0 top-0 h-full rounded-full bg-neutral-900 transition-all duration-200"
                              :style="{ width: `${Math.min(progressTrackers[item.id?.toString() || item.key]?.current ?? 0, getProgressLimit(item.status))}%` }">
                            </div>
                          </div>
                          <div class="mt-1.5 flex items-center justify-between text-[10px] text-neutral-400">
                            <span>{{ task.submissionStatus || (item.status === 'generating' || item.status === 'processing' ? '生成中' : '等待中') }}</span>
                            <span>{{ Math.round(Math.min(progressTrackers[item.id?.toString() || item.key]?.current ?? 0, getProgressLimit(item.status))) }}%</span>
                          </div>
                        </div>
                        <span class="text-[11px] leading-[1.4] text-neutral-500">{{ task.submissionStatus ? `${task.submissionStatus}，请稍候…` : '正在努力生成画面…' }}</span>
                      </div>
                      <div v-if="item.status === 'done' && item.imageUrl && !batchDownloadMode"
                        class="pointer-events-none absolute inset-0 flex flex-wrap items-center justify-center gap-1 bg-[rgba(17,24,39,0)] p-2 opacity-0 transition group-hover:pointer-events-auto group-hover:bg-[rgba(17,24,39,.38)] group-hover:opacity-100">
                        <button
                          class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button" @click.prevent="openImageViewer(item, index)">查看</button>
                        <button
                          class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button"
                          @click.prevent="downloadImage(item.imageUrl, `ai-${taskIndex + 1}-${index + 1}-${Date.now()}.png`)">下载</button>
                        <button
                          class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button" @click.prevent="openEdit(item)">编辑</button>
                        <button
                          class="pointer-events-auto rounded-full bg-white/95 px-[9px] py-1.5 text-[11px] font-semibold leading-none text-neutral-900 transition hover:-translate-y-px hover:bg-white hover:shadow-[0_12px_24px_rgba(15,23,42,.18)]"
                          type="button" @click.prevent="openExpand(item, task.aspectRatio)">AI 扩图</button>
                      </div>
                    </div>
                  </div>
                </div>
              </section>
            </div>
            <div v-if="batchDownloadMode" class="batch-inline-bar">
              <span>已选择 {{ batchDownloadSelectedItems.length }} 张图片</span>
              <div><button type="button" class="ghost" @click="toggleBatchDownloadAll(!batchDownloadAllSelected)">{{ batchDownloadAllSelected ? '取消全选' : '全选' }}</button><button type="button" class="batch-inline-confirm" :disabled="!batchDownloadSelectedItems.length || batchDownloading" @click="confirmBatchDownload">{{ batchDownloading ? `下载中 ${batchDownloadProgress}/${batchDownloadSelectedItems.length}` : '下载已选图片' }}</button></div>
            </div>
          </template>
          <div v-else
            class="m-[10px_20px_16px] flex flex-1 items-center justify-center overflow-auto rounded-[26px] border border-dashed border-neutral-300 bg-[linear-gradient(180deg,#fcfcfd,#f9fafb)] p-6 text-neutral-500">
            <div class="flex max-w-[420px] flex-col items-center gap-2.5 text-center">
              <strong class="text-[15px] text-neutral-900">暂无任务</strong>
              <span class="text-[13px] leading-[1.6]">在左侧配置提示词与参数后开始生成，结果会在这里以任务卡片形式展示。</span>
              <!-- <div class="mt-2 flex flex-wrap justify-center gap-2">
                  <button v-for="tip in quickPrompts" :key="tip" class="h-9 rounded-full bg-neutral-100 px-3.5 text-sm font-medium text-neutral-900 transition-all hover:-translate-y-px hover:shadow-[0_10px_22px_rgba(15,23,42,0.08)]" type="button" @click="prompt = tip">{{ tip }}</button>
                </div> -->
            </div>
          </div>
        </div>
      </div>
    </section>
<!-- Image Edit Modal -->
    <ImageEditModal :visible="editModalVisible" :imageUrl="editingItem?.imageUrl ?? ''"
      :originalPrompt="editingItem?.prompt ?? ''" @close="editModalVisible = false" @submit="handleEditSubmit" />

    <ImageExpandModal :visible="expandModalVisible" :imageUrl="expandImageUrl"
      :originalPrompt="editingItem?.prompt ?? ''" :originalAspectRatio="expandOriginalAspectRatio"
      :resolutions="supportedResolutions" :defaultResolution="resolution"
      @close="closeExpandModal" @submit="handleExpandSubmit" />

    <!-- Reference Image Edit Modal -->
    <ImageEditModal :visible="refEditModalVisible" :imageUrl="refEditImageUrl" :originalPrompt="''" mode="ref"
      @close="refEditModalVisible = false" @submit="handleRefEditSubmit" />

    <!-- Candidate Confirmation Modal -->
    <Teleport to="body">
      <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0"
        enter-to-class="opacity-100" leave-active-class="transition-all duration-150" leave-from-class="opacity-100"
        leave-to-class="opacity-0">
        <div v-if="candidateModalVisible"
          class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
          @click.self="closeCandidateModal">
          <div class="bg-white rounded-[24px] shadow-2xl w-full max-w-4xl overflow-hidden border border-white/60">
            <div class="px-6 py-4 border-b border-neutral-100 flex items-center justify-between">
              <div>
                <h3 class="text-lg font-semibold text-neutral-900">编辑预览</h3>
                <p class="text-sm text-neutral-500 mt-0.5">先看编辑效果，再决定是否替换原图</p>
              </div>
              <button
                class="p-2 rounded-xl hover:bg-neutral-100 transition-colors text-neutral-400 hover:text-neutral-700"
                @click="closeCandidateModal">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div class="candidate-grid p-6">
              <div class="candidate-pane">
                <span class="candidate-label">原图</span>
                <div class="candidate-frame">
                  <img v-if="candidateOriginalItem" :src="resolveImageUrl(candidateOriginalItem.imageUrl)" alt="原图" @contextmenu.prevent.stop />
                  <div v-else class="w-full h-full bg-neutral-100" />
                </div>
              </div>
              <div class="candidate-pane">
                <span class="candidate-label">编辑效果</span>
                <div class="candidate-frame">
                  <img :src="resolveImageUrl(candidateImageUrl)" alt="编辑效果图" @contextmenu.prevent.stop />
                </div>
              </div>
            </div>
            <div class="px-6 pb-5 flex justify-end gap-3">
              <button
                class="pointer-events-auto px-5 py-2.5 text-sm font-medium rounded-xl border-2 border-neutral-200 text-neutral-600 hover:border-neutral-300 hover:bg-neutral-50 transition-all"
                type="button" :disabled="candidateLoading" @click="rejectCandidate">
                保留原图
              </button>
              <button
                class="pointer-events-auto px-5 py-2.5 text-sm font-semibold rounded-xl bg-neutral-900 text-white hover:bg-neutral-800 transition-all disabled:opacity-50"
                type="button" :disabled="candidateLoading" @click="acceptCandidate">
                {{ candidateLoading ? '处理中…' : '采用编辑结果' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Image Viewer Lightbox -->
    <ImagePreview :visible="!!viewerItem" :imageUrl="viewerItem?.imageUrl ?? ''" :alt="viewerItem?.prompt ?? ''"
      :maxWidth="1200" @close="closeImageViewer">
      <template #footer>
        <div class="preview-footer-bar">
          <p class="preview-footer-prompt">{{ viewerItem?.prompt }}</p>
          <div class="preview-footer-actions">
            <button class="preview-download-btn"
              @click="downloadImage(viewerItem.imageUrl, `ai-${viewerIndex + 1}-${Date.now()}.png`)">
              <ArrowDownTrayIcon class="w-3.5 h-3.5" />
              下载
            </button>
          </div>
        </div>
      </template>
    </ImagePreview>

    <!-- 违规图片示例：表格优先展示 -->
    <TransitionRoot as="template" :show="violationVisible">
      <Dialog class="relative z-[60]" @close="closeViolationDialog">
        <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0" enter-to="opacity-100" leave="duration-150 ease-in" leave-from="opacity-100" leave-to="opacity-0">
          <div class="fixed inset-0 bg-neutral-900/45 backdrop-blur-sm" />
        </TransitionChild>
        <div class="fixed inset-0 flex items-center justify-center p-3 sm:p-6">
          <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0 translate-y-3 scale-[0.98]" enter-to="opacity-100 translate-y-0 scale-100" leave="duration-150 ease-in" leave-from="opacity-100 scale-100" leave-to="opacity-0 scale-[0.98]">
            <DialogPanel class="flex h-[min(760px,88dvh)] w-full max-w-6xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl">
              <div class="flex shrink-0 items-start justify-between gap-4 border-b border-neutral-200 px-5 py-5 sm:px-6">
                <div class="min-w-0">
                  <DialogTitle class="flex items-center gap-2.5 text-lg font-semibold text-neutral-900"><TableCellsIcon class="h-5 w-5 text-neutral-500" />违规图片示例<span class="rounded-md bg-neutral-100 px-2 py-0.5 text-xs font-medium text-neutral-500">{{ violationItems.length }} 条</span></DialogTitle>
                  <p class="mt-1.5 text-xs leading-5 text-neutral-500">查看各平台图片问题，点击附件查看原图。</p>
                </div>
                <div class="flex shrink-0 items-center gap-2 sm:gap-4">
                  <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition hover:border-neutral-300 hover:bg-neutral-50 hover:text-neutral-900 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900" @click="openViolationForm"><PlusIcon class="h-3.5 w-3.5" />新增示例</button>
                  <button type="button" class="rounded-lg p-2 text-neutral-400 transition hover:bg-neutral-100 hover:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900" aria-label="关闭违规图片示例" @click="violationVisible = false"><XMarkIcon class="h-5 w-5" /></button>
                </div>
              </div>
              <div class="min-h-0 flex-1 overflow-auto">
                <table class="w-full min-w-[900px] table-fixed text-left text-sm">
                  <colgroup><col class="w-[12%]" /><col class="w-[15%]" /><col class="w-[29%]" /><col class="w-[20%]" /><col class="w-[12%]" /><col class="w-[12%]" /></colgroup>
                  <thead class="sticky top-0 z-10 bg-neutral-50 text-xs font-semibold text-neutral-500"><tr><th v-for="heading in ['平台', '问题类型', '问题描述', '附件图片', '提交人', '提交日期']" :key="heading" scope="col" class="border-b border-neutral-200 px-5 py-3.5">{{ heading }}</th></tr></thead>
                  <tbody class="divide-y divide-neutral-100">
                    <tr v-for="item in violationItems" :key="item.id" class="transition-colors hover:bg-neutral-50/80">
                      <td class="px-5 py-4 align-top"><span class="inline-flex rounded-md border border-neutral-200 bg-white px-2 py-1 text-xs font-medium text-neutral-700">{{ item.platform }}</span></td>
                      <td class="break-words px-5 py-4 align-top font-medium text-neutral-800">{{ item.problem_type }}</td>
                      <td class="whitespace-pre-wrap break-words px-5 py-4 align-top leading-6 text-neutral-600">{{ item.description }}</td>
                      <td class="px-5 py-4 align-top"><button v-if="item.image_url" type="button" class="group w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900 focus-visible:ring-offset-2" :aria-label="`预览${item.platform}的${item.problem_type}示例图片`" @click="previewViolation(item.image_url)"><div class="relative h-24 w-full max-w-[160px] overflow-hidden rounded-lg bg-neutral-100"><img :src="getDisplayImageUrl(resolveImageUrl(item.image_url))" class="h-full w-full object-cover transition duration-200 group-hover:scale-105" alt="违规示例" /><span class="absolute inset-0 flex items-center justify-center bg-neutral-900/40 text-white opacity-0 transition group-hover:opacity-100 group-focus-visible:opacity-100"><ArrowsPointingOutIcon class="h-5 w-5" /></span></div></button><span v-else class="text-neutral-400">—</span></td>
                      <td class="break-words px-5 py-4 align-top text-neutral-600">{{ item.submitter }}</td>
                      <td class="whitespace-nowrap px-5 py-4 align-top text-xs leading-6 tabular-nums text-neutral-500">{{ violationDate(item.created_at) }}</td>
                    </tr>
                    <tr v-if="violationLoading"><td colspan="6" class="px-5 py-20 text-center text-sm text-neutral-500" role="status">正在加载示例…</td></tr>
                    <tr v-else-if="violationLoadError"><td colspan="6" class="px-5 py-20 text-center"><p class="text-sm text-red-600" role="alert">{{ violationLoadError }}</p><button type="button" class="mt-3 rounded-lg border border-neutral-200 px-4 py-2 text-xs hover:bg-neutral-50" @click="loadViolationExamples">重新加载</button></td></tr>
                    <tr v-else-if="!violationItems.length"><td colspan="6" class="px-5 py-24 text-center"><TableCellsIcon class="mx-auto h-9 w-9 text-neutral-300" /><p class="mt-4 text-sm font-medium text-neutral-600">暂无违规图片示例</p><p class="mt-2 text-xs text-neutral-400">已提交的示例将在这里按平台、问题和附件展示。</p></td></tr>
                  </tbody>
                </table>
              </div>
              <div class="flex shrink-0 items-center justify-between gap-3 border-t border-neutral-200 bg-white px-5 py-3 text-xs text-neutral-400 sm:px-6"><span>共 {{ violationItems.length }} 条示例</span><span>附件支持大图预览</span></div>
            </DialogPanel>
          </TransitionChild>
        </div>
              <Dialog :open="violationFormVisible" class="relative z-[70]" @close="closeViolationForm">
                <div class="fixed inset-0 bg-black/50" />
                <div class="fixed inset-0 flex items-center justify-center p-4">
                  <DialogPanel class="max-h-[90dvh] w-full max-w-lg overflow-y-auto rounded-3xl bg-white p-6 shadow-2xl">
                    <div class="flex items-center justify-between">
                      <DialogTitle class="text-lg font-bold">新增违规图片示例</DialogTitle>
                      <button type="button" class="rounded-lg p-2 hover:bg-neutral-100 disabled:opacity-40" :disabled="violationSaving" aria-label="关闭新增表单" @click="closeViolationForm"><XMarkIcon class="h-5 w-5" /></button>
                    </div>
                    <form class="mt-5 space-y-4" @submit.prevent="submitViolation">
                      <fieldset class="space-y-4 disabled:opacity-60" :disabled="violationSaving">
                        <Select v-model="violationDraft.platform" label="平台" :options="violationPlatformOptions" :disabled="violationSaving" />
                        <label class="block text-sm font-medium">问题类型<input v-model="violationDraft.type" maxlength="100" required class="mt-1.5 w-full rounded-xl border border-neutral-200 px-3 py-2.5" placeholder="如：文字侵权" /></label>
                        <label class="block text-sm font-medium">问题描述<textarea v-model="violationDraft.description" maxlength="5000" required rows="3" class="mt-1.5 w-full resize-none rounded-xl border border-neutral-200 px-3 py-2.5" placeholder="描述触发违规的原因" /></label>
                        <div>
                          <span class="text-sm font-medium">附件图片</span>
                          <button type="button" class="mt-1.5 flex min-h-32 w-full flex-col items-center justify-center overflow-hidden rounded-2xl border-2 border-dashed text-center transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900 disabled:cursor-wait" :class="violationDragOver ? 'border-amber-500 bg-amber-50' : 'border-neutral-200 bg-neutral-50 hover:border-neutral-400'" :disabled="violationSaving" @dragover.prevent="violationDragOver = !violationSaving" @dragleave.prevent="violationDragOver = false" @drop.prevent="handleViolationDrop" @click="violationUpload?.click()">
                            <img v-if="violationDraft.image" :src="violationDraft.image" class="h-40 w-full object-cover" alt="待上传" />
                            <template v-else><PhotoIcon class="h-7 w-7 text-neutral-400" /><p class="mt-2 text-xs text-neutral-500">拖拽图片到此处，或点击上传</p></template>
                          </button>
                          <input ref="violationUpload" type="file" accept="image/*" class="hidden" :disabled="violationSaving" @change="handleViolationFile" />
                          <p class="mt-1.5 text-xs text-neutral-400">{{ violationDraft.image ? '点击图片或再次拖拽可替换 · ' : '' }}单张图片不超过 30 MB</p>
                        </div>
                      </fieldset>
                      <p v-if="violationSubmitError" class="text-sm text-red-600" role="alert">{{ violationSubmitError }}</p>
                      <div class="flex justify-end gap-3 pt-2">
                        <button type="button" class="rounded-xl border border-neutral-200 px-4 py-2.5 text-sm disabled:opacity-40" :disabled="violationSaving" @click="closeViolationForm">取消</button>
                        <button type="submit" class="rounded-xl bg-neutral-900 px-5 py-2.5 text-sm font-semibold text-white disabled:opacity-40" :disabled="violationSaving || !violationDraft.type.trim() || !violationDraft.description.trim() || !violationFile">{{ violationSaving ? '保存中…' : '提交' }}</button>
                      </div>
                    </form>
                  </DialogPanel>
                </div>
              </Dialog>

    <Dialog :open="!!violationPreview" class="relative z-[80]" @close="violationPreview = ''">
      <div class="fixed inset-0 bg-neutral-900/80 backdrop-blur-sm" />
      <div class="fixed inset-0 flex items-center justify-center p-6">
        <DialogPanel class="relative flex max-h-[90dvh] max-w-6xl flex-col items-center gap-3">
          <DialogTitle class="sr-only">违规图片示例预览</DialogTitle>
          <button type="button" class="absolute right-2 top-2 rounded-full bg-neutral-900/70 p-2 text-white transition hover:bg-neutral-900 focus-visible:ring-2 focus-visible:ring-white" aria-label="关闭图片预览" @click="violationPreview = ''"><XMarkIcon class="h-5 w-5" /></button>
          <img :src="violationPreview" class="max-h-[85dvh] max-w-full rounded-lg object-contain" alt="违规图片示例预览" />
        </DialogPanel>
      </div>
    </Dialog>

      </Dialog>
    </TransitionRoot>
    <!-- 设计库弹窗 -->
    <Teleport to="body">
      <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0"
        enter-to-class="opacity-100" leave-active-class="transition-all duration-150" leave-from-class="opacity-100"
        leave-to-class="opacity-0">
        <div v-if="designLibraryVisible"
          class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
          @click.self="designLibraryVisible = false">
          <div
            class="bg-white rounded-[28px] shadow-[0_24px_80px_rgba(0,0,0,0.18)] w-full max-w-6xl h-[86vh] overflow-hidden flex flex-col border border-white/60">
            <div class="relative px-6 py-5 border-b border-neutral-100 flex-shrink-0 overflow-hidden">
              <div
                class="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.10),_transparent_32%),radial-gradient(circle_at_top_right,_rgba(99,102,241,0.10),_transparent_28%)]" />
              <div class="relative flex items-center justify-between gap-4">
                <div>
                  <!-- <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-neutral-900 text-white text-[11px] font-medium shadow-sm">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                    我的设计
                  </div> -->
                  <h3 class="mt-3 text-xl font-semibold tracking-tight text-neutral-900">“设计库”</h3>
                  <p class="text-sm text-neutral-500 mt-1.5">选择一张已保存图片作为当前参考图</p>
                </div>
                <button
                  class="p-2.5 rounded-xl bg-white/80 border border-neutral-200 hover:bg-white hover:border-neutral-300 transition-colors text-neutral-400 hover:text-neutral-700 shadow-sm"
                  @click="designLibraryVisible = false">
                  <XMarkIcon class="w-5 h-5" />
                </button>
              </div>
            </div>

            <div
              class="px-6 py-4 border-b border-neutral-100 flex flex-col gap-3 flex-shrink-0 bg-neutral-50/80 backdrop-blur-sm">
              <div class="flex items-center justify-between gap-4 flex-wrap">
                <div
                  class="inline-flex max-w-full items-center gap-1.5 rounded-2xl bg-white border border-neutral-200 p-1 shadow-sm overflow-x-auto">
                  <button v-for="tab in designLibraryTabs" :key="tab.value" type="button"
                    class="px-4 py-2 text-sm font-medium rounded-xl transition-all" :class="designTabClass(tab)"
                    @click="designLibrarySource = tab.value; designLibraryGroup = null; fetchDesignLibrary()">
                    {{ tab.label }}
                  </button>
                </div>
                <div
                  class="text-xs text-neutral-400 rounded-full bg-white border border-neutral-200 px-3 py-1.5 shadow-sm">
                  已选参考图 {{ refImages.length }}/6
                </div>
              </div>
              <!-- <div class="relative">
                <span class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-neutral-300">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m21 21-4.35-4.35m1.85-5.15a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </span>
                <input
                  v-model.trim="designLibraryKeyword"
                  type="text"
                  placeholder="搜索标题、提示词或日期"
                  class="w-full rounded-2xl border border-neutral-200 bg-white pl-11 pr-4 py-3 text-sm text-neutral-700 placeholder:text-neutral-400 focus:outline-none focus:ring-2 focus:ring-neutral-900/10 focus:border-neutral-300 shadow-sm"
                />
              </div> -->
            </div>

            <div class="flex-1 min-h-0 p-6 bg-[linear-gradient(180deg,#fafaf9_0%,#f5f5f4_100%)] overflow-hidden">
              <div class="relative h-full min-h-[520px] overflow-auto pr-1">
                <div v-if="designLibraryGroup" class="space-y-4">
                  <div
                    class="flex items-center justify-between gap-4 sticky top-0 z-10 bg-[linear-gradient(180deg,#fafaf9_0%,rgba(250,250,249,0.98)_74%,rgba(250,250,249,0)_100%)] pb-4">
                    <button type="button"
                      class="inline-flex items-center gap-1.5 px-1 py-1 text-sm font-medium text-neutral-500 hover:text-neutral-900 transition-colors"
                      @click="closeDesignLibraryGroup">
                      <span class="text-base leading-none">←</span>
                      <span>返回任务列表</span>
                    </button>
                    <span
                      class="shrink-0 px-2 py-0.5 rounded-full bg-black/5 text-neutral-500 text-[11px] font-medium">{{
                        designLibraryGroup.images?.length || 0 }} 张</span>
                  </div>

                  <div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                    <button v-for="(image, index) in designLibraryGroup.images"
                      :key="`${designLibraryGroup.id}-${index}`" type="button"
                      class="group relative overflow-hidden rounded-[22px] border border-neutral-200 bg-white text-left hover:shadow-[0_18px_40px_rgba(0,0,0,0.10)] hover:border-neutral-300 transition-all duration-200 hover:-translate-y-0.5"
                      @click="selectDesignLibraryImage(image, designLibraryGroup.prompt)">
                      <div class="relative aspect-square bg-neutral-100 overflow-hidden">
                        <img :src="getImageThumbnailUrl(image, 520)" :alt="`${designLibraryGroup.title}-${index + 1}`"
                          @contextmenu.prevent.stop
                          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
                        <div class="absolute inset-0 bg-gradient-to-t from-black/72 via-black/20 to-transparent" />
                        <div
                          class="absolute top-3 right-3 px-2.5 py-1 rounded-full bg-black/45 text-white text-[11px] font-medium backdrop-blur-sm">
                          {{ index + 1 }} / {{ designLibraryGroup.images.length }}
                        </div>
                        <div class="absolute inset-x-3 bottom-3 flex items-end justify-between gap-3">
                          <div>
                            <div class="text-sm font-semibold text-white">选择这张图</div>
                            <div class="mt-1 text-[11px] text-white/80">加入到当前参考图</div>
                          </div>
                          <div
                            class="px-2.5 py-1 rounded-full bg-white/12 text-white text-[11px] font-medium backdrop-blur-sm opacity-0 group-hover:opacity-100 transition-opacity">
                            点击选择
                          </div>
                        </div>
                      </div>
                    </button>
                  </div>
                </div>

                <template v-else>
                  <div class="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4"
                    :class="designLibraryLoading ? 'opacity-60' : ''">
                    <button v-for="item in filteredDesignLibraryItems" :key="item.id" type="button"
                      class="group bg-white/92 border border-neutral-200 rounded-[22px] overflow-hidden text-left hover:shadow-[0_18px_40px_rgba(0,0,0,0.10)] hover:border-neutral-300 transition-all duration-200 hover:-translate-y-0.5"
                      @click="pickDesignTemplate(item)">
                      <div class="relative aspect-square bg-neutral-100 overflow-hidden">
                        <img :src="getImageThumbnailUrl(item.imageUrl, 520)" :alt="item.title"
                          @contextmenu.prevent.stop
                          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
                        <div class="absolute top-3 left-3">
                          <span
                            class="px-2.5 py-1 rounded-full bg-black/45 text-white text-[11px] font-medium backdrop-blur-sm">
                            {{ designSourceLabel(item) }}
                          </span>
                        </div>
                        <div v-if="item.type === 'scene_task' && item.images?.length"
                          class="absolute top-3 right-3 px-2.5 py-1 rounded-full bg-black/45 text-white text-[11px] font-medium backdrop-blur-sm">
                          {{ item.images.length }} 张
                        </div>
                        <div class="absolute inset-0 bg-gradient-to-t from-black/72 via-black/18 to-transparent" />
                        <div class="absolute inset-x-3 bottom-3">
                          <div class="flex items-end justify-between gap-3">
                            <div class="min-w-0">
                              <div class="text-sm font-semibold text-white line-clamp-2 leading-relaxed">{{ item.title
                              }}</div>
                              <div class="mt-1 flex items-center gap-2 text-[11px] text-white/78">
                                <span v-if="item.subtitle" class="line-clamp-1">{{ item.subtitle }}</span>
                                <span v-if="item.type === 'scene_task'" class="shrink-0">点击进入选图</span>
                                <span v-else class="shrink-0">点击添加参考图</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </button>
                  </div>

                  <div v-if="!designLibraryLoading && !filteredDesignLibraryItems.length"
                    class="absolute inset-0 flex items-center justify-center">
                    <div
                      class="w-full max-w-2xl rounded-[24px] border border-neutral-200 bg-white shadow-sm py-16 px-8 text-center">
                      <div class="w-16 h-16 mx-auto rounded-2xl bg-neutral-100 flex items-center justify-center mb-5">
                        <PhotoIcon class="w-8 h-8 text-neutral-300" />
                      </div>
                      <h4 class="text-base font-semibold text-neutral-800">当前分类暂无内容</h4>
                      <p class="mt-2 text-sm text-neutral-400">请切换其他分类，或先去对应页面保存作品后再回来选择</p>
                    </div>
                  </div>

                  <div v-if="designLibraryLoading"
                    class="absolute inset-0 bg-neutral-50/72 backdrop-blur-[1px] rounded-[24px]">
                    <div class="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
                      <div v-for="i in 8" :key="i"
                        class="rounded-2xl overflow-hidden bg-white border border-neutral-200 animate-pulse">
                        <div class="aspect-square bg-neutral-100" />
                        <div class="p-3 space-y-2">
                          <div class="h-3 bg-neutral-100 rounded w-3/4" />
                          <div class="h-3 bg-neutral-100 rounded w-1/2" />
                        </div>
                      </div>
                    </div>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 提示词全屏编辑弹窗 -->
    <Teleport to="body">
      <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0"
        enter-to-class="opacity-100" leave-active-class="transition-all duration-150" leave-from-class="opacity-100"
        leave-to-class="opacity-0">
        <div v-if="promptFullscreen"
          class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
          @click.self="promptFullscreen = false">
          <div class="bg-white rounded-2xl shadow-2xl w-full max-w-2xl flex flex-col overflow-hidden"
            style="height: min(80vh, 600px)">
            <!-- Header -->
            <div class="flex items-center justify-between px-5 py-3.5 border-b border-neutral-100 flex-shrink-0">
              <div class="flex items-center gap-2">
                <h3 class="text-sm font-semibold text-neutral-900">提示词</h3>
                <span class="text-[10px] text-neutral-400 bg-neutral-100 px-2 py-0.5 rounded-full">描述你想生成的画面</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs text-neutral-400">{{ prompt.length }} 字</span>
                <button
                  class="p-1.5 rounded-lg hover:bg-neutral-100 transition-colors text-neutral-400 hover:text-neutral-600"
                  @click="promptFullscreen = false">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
            <!-- Textarea -->
            <textarea v-model="prompt" placeholder="描述你想生成的图片，例如：一只可爱的橙色猫咪坐在窗台上，背景是模糊的城市夜景..."
              class="flex-1 w-full text-sm px-5 py-4 resize-none bg-white focus:outline-none leading-relaxed placeholder:text-neutral-400 text-neutral-800"
              autofocus />
            <!-- Footer -->
            <div
              class="flex items-center justify-between px-5 py-3 border-t border-neutral-100 flex-shrink-0 bg-neutral-50">
              <button class="text-xs text-neutral-400 hover:text-red-500 transition-colors"
                @click="prompt = ''">清空内容</button>
              <button
                class="px-4 py-2 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors"
                @click="promptFullscreen = false">完成</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <AssetPickerModal
      :visible="assetPickerVisible"
      title="从资产库选择参考图"
      multiple
      :max="Math.max(1, 6 - refImages.length)"
      :type-filter="assetPickerImageTypes"
      @close="assetPickerVisible = false"
      @select="selectAssetsAsReferenceImages"
    />
  </MainLayout>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick, reactive } from 'vue'
import { useRoute } from 'vue-router'
import {
  SparklesIcon, PhotoIcon, XMarkIcon,
  ExclamationCircleIcon, ArrowDownTrayIcon,
  PencilIcon, TableCellsIcon, PlusIcon, ArrowsPointingOutIcon,
} from '@heroicons/vue/24/outline'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import request, { downloadImage, fetchImageAsDataUrl, getDisplayImageUrl, getDisplayThumbnailUrl, resolveImageUrl } from '@/utils/request'
import AssetPickerModal from '@/components/assets/AssetPickerModal.vue'
import ImageEditModal from '@/components/ui/ImageEditModal.vue'
import ImageExpandModal from '@/components/ui/ImageExpandModal.vue'
import ImagePreview from '@/components/ui/ImagePreview.vue'
import Select from '@/components/ui/Select.vue'

import { useToast } from '@/composables/useToast'
import { useFakeProgress } from '@/composables/useFakeProgress'
import { resolveReferenceAspectRatio } from '@/utils/referenceAspectRatio'
import { customSizeError, closestImageRatio, matchGPTImageSize, matchPresetImageSize } from '@/utils/customImageSize'

const route = useRoute()
const toast = useToast()

const violationVisible = ref(false)
const violationFormVisible = ref(false)
const violationDragOver = ref(false)
const violationUpload = ref<HTMLInputElement | null>(null)
const violationPlatforms = ['Amazon', 'TikTok', 'Homedepot', 'TEMU', 'Walmart', 'Lowes', 'Wayfair']
const violationPlatformOptions = violationPlatforms.map(platform => ({ label: platform, value: platform }))
type ViolationExample = { id: number; platform: string; problem_type: string; description: string; image_url: string; submitter: string; created_at: string }
const violationItems = ref<ViolationExample[]>([])
const violationDraft = reactive({ platform: violationPlatforms[0], type: '', description: '', image: '' })
const violationFile = ref<File | null>(null)
const violationUploadedUrl = ref('')
const violationPreview = ref('')
const violationLoading = ref(false)
const violationLoadError = ref('')
const violationSaving = ref(false)
const violationSubmitError = ref('')

watch(violationVisible, visible => { if (visible) loadViolationExamples() })

async function loadViolationExamples() {
  if (violationLoading.value) return
  violationLoading.value = true
  violationLoadError.value = ''
  try {
    const response = await request.get('/image-violation-examples')
    violationItems.value = response.data.data
  } catch (error: any) {
    violationLoadError.value = error.response?.data?.error || '获取示例失败，请重试'
  } finally {
    violationLoading.value = false
  }
}

function closeViolationDialog() {
  if (violationFormVisible.value || violationPreview.value) return
  violationVisible.value = false
}

function clearViolationFile() {
  if (violationDraft.image) URL.revokeObjectURL(violationDraft.image)
  violationDraft.image = ''
  violationFile.value = null
  violationUploadedUrl.value = ''
}

function openViolationForm() {
  clearViolationFile()
  violationDraft.platform = violationPlatforms[0]
  violationDraft.type = ''
  violationDraft.description = ''
  violationSubmitError.value = ''
  violationDragOver.value = false
  violationFormVisible.value = true
}

function closeViolationForm() {
  if (violationSaving.value) return
  violationFormVisible.value = false
  clearViolationFile()
}

function readViolationFile(file?: File) {
  violationDragOver.value = false
  if (violationSaving.value || !file) return
  if (!file.type.startsWith('image/')) { toast.error('请选择图片文件'); return }
  if (file.size > 30 * 1024 * 1024) { toast.error('单张图片不能超过 30 MB'); return }
  clearViolationFile()
  violationFile.value = file
  violationDraft.image = URL.createObjectURL(file)
  violationSubmitError.value = ''
}

function handleViolationFile(event: Event) {
  const input = event.target as HTMLInputElement
  readViolationFile(input.files?.[0])
  input.value = ''
}

function handleViolationDrop(event: DragEvent) { readViolationFile(event.dataTransfer?.files?.[0]) }

async function submitViolation() {
  if (violationSaving.value || !violationFile.value || !violationDraft.type.trim() || !violationDraft.description.trim()) return
  violationSaving.value = true
  violationSubmitError.value = ''
  try {
    if (!violationUploadedUrl.value) {
      const form = new FormData()
      form.append('image', violationFile.value)
      const upload = await request.post('/upload/image', form)
      violationUploadedUrl.value = upload.data.url
    }
    const response = await request.post('/image-violation-examples', {
      platform: violationDraft.platform,
      problem_type: violationDraft.type.trim(),
      description: violationDraft.description.trim(),
      image_url: violationUploadedUrl.value,
    })
    violationItems.value.unshift(response.data)
    violationFormVisible.value = false
    clearViolationFile()
    toast.success('示例已保存')
  } catch (error: any) {
    violationSubmitError.value = error.response?.data?.error || '保存失败，请重试'
  } finally {
    violationSaving.value = false
  }
}

function violationDate(value: string) {
  return new Date(value).toLocaleDateString('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' })
}

function previewViolation(image: string) { violationPreview.value = getDisplayImageUrl(resolveImageUrl(image)) }

// ---------- Form state ----------
const count = ref(1)
const prompt = ref('')
const promptFullscreen = ref(false)
const aspectRatio = ref('1:1')
const sizeMode = ref('preset')
const customWidth = ref<number | string>(2928)
const customHeight = ref<number | string>(1200)
const sizeError = computed(() => customSizeError(customWidth.value, customHeight.value))
const autoMatchResolution = computed(() => sizeMode.value === 'custom' && /gemini-3(?:\.1)?-(?:flash|pro)-image/i.test(selectedModel.value))
const customGenerationSize = computed(() => {
  if (sizeError.value) return null
  const gpt = matchGPTImageSize(selectedModel.value, customWidth.value, customHeight.value)
  if (gpt) return { ...gpt, ratio: closestImageRatio(gpt.width, gpt.height, dynamicAspectRatioOptions.value), resolution: resolution.value }
  const resolutions = autoMatchResolution.value ? supportedResolutions.value : [resolution.value]
  const candidates = resolutions.flatMap(quality => dynamicAspectRatioOptions.value.flatMap(ratio => {
    const pixels = ratioPixelSize(ratio, quality)
    if (!pixels) return []
    const [width, height] = pixels.split(' × ').map(Number)
    return [{ width, height, ratio, resolution: quality }]
  }))
  return matchPresetImageSize(customWidth.value, customHeight.value, candidates)
})
const customRatio = computed(() => customGenerationSize.value?.ratio || closestImageRatio(customWidth.value, customHeight.value, dynamicAspectRatioOptions.value))
const resolution = ref('')           // 当前选中的分辨率
const supportedResolutions = ref([]) // 默认绘画模型支持的分辨率列表

// 模型选择
const drawingModels = ref<any[]>([])
const selectedModel = ref<string>('')
const modelMenuOpen = ref(false)
const refUploadInput = ref(null)
const refImages = ref<Array<{ data?: string; mimeType: string; url: string; thumbnailUrl?: string; mask?: string; remoteUrl?: string }>>([])
const refDragOver = ref(false)
const designLibraryVisible = ref(false)
const designLibraryLoading = ref(false)
const designLibraryKeyword = ref('')
const designLibrarySource = ref('generate')
const designLibraryItems = ref([])
const designLibraryGroup = ref(null)
const assetPickerVisible = ref(false)
const assetPickerImageTypes = ['图片', '产品图', '场景图', '背景', '风格参考图', '色块', '参考方案图']

// ---------- Generate state ----------
const isGenerating = ref(false)
const globalError = ref('')
// taskBatches: 每次提交生成一个新 batch，已完成/历史的 batch 保留在此数组中
const taskBatches = ref<any[]>([])  // 从 API 加载，不再使用 localStorage

const batchProgress = ref({ current: 0, total: 0 })
const savingIds = ref([])

// 当前正在使用的 batch items 引用
const batchResults = ref<any[]>([])
const batchDownloadMode = ref(false)
const batchDownloadSelectedIds = ref<string[]>([])
const batchDownloading = ref(false)
const batchDownloadProgress = ref(0)

// 全局轮询机制
let pollTimer: number | null = null
let modelPollTimer: number | null = null

// 假进度条跟踪器（以 taskId 为 key）
// 使用普通对象 + Vue 响应式触发更新
const progressTrackers = ref<Record<string, { current: number }>>({})
const fakeProgress = useFakeProgress({ storageKeyPrefix: 'gen_progress' })

// 获取进度条上限值
function getProgressLimit(status: string): number {
  if (status === 'generating' || status === 'processing') return 99
  return 50
}

// 懒加载图片
const lazyImageObserver = new IntersectionObserver(
  (entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        const img = entry.target as HTMLImageElement
        const src = img.dataset.src
        if (src) {
          img.src = src
          img.removeAttribute('data-src')
          img.decode?.()
            .catch(() => undefined)
            .finally(() => {
              img.classList.remove('lazy-img')
              img.classList.add('lazy-img-loaded')
            })
        }
        lazyImageObserver.unobserve(img)
      }
    })
  },
  { rootMargin: '24px', threshold: 0.05 }
)

function observeLazyImages() {
  const images = document.querySelectorAll('.lazy-img')
  images.forEach((img) => lazyImageObserver.observe(img))
}

function mountLazyImages(container: HTMLElement) {
  const images = container.querySelectorAll('.lazy-img')
  images.forEach((img) => lazyImageObserver.observe(img))
}

// ========== 假进度条管理 ==========

function hasPendingTasks() {
  return taskBatches.value.some(batch =>
    batch.items.some(item => item.status === 'pending' || item.status === 'generating'))
}

function startPolling() {
  if (!hasPendingTasks()) return stopPolling()
  stopPolling()
  pollTimer = window.setInterval(() => {
    loadTaskBatchesFromAPI().then(() => {
      // loadTaskBatchesFromAPI 内部会根据当前批次状态停止轮询
    })
  }, 1500)
}

function stopPolling() {
  if (pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
}

function getCurrentBatch() {
  return taskBatches.value.length > 0 ? taskBatches.value[taskBatches.value.length - 1] : null
}

// 找到包含指定 item 的 batch（使用 ID 匹配，兼容引用失效的情况）
function findBatchByItem(item: any) {
  console.log('[findBatchByItem] searching for item:', item?.id, item?.key)
  for (const batch of taskBatches.value) {
    // 使用 ID 匹配（优先）或引用匹配（兼容）
    const idx = item?.id
      ? batch.items.findIndex((i: any) => i.id === item.id)
      : batch.items.indexOf(item)
    console.log('[findBatchByItem] batch.key=', batch.key, 'idx=', idx)
    if (idx !== -1) return batch
  }
  return null
}

function updateBatchItem(idx, updates, batch = getCurrentBatch()) {
  if (batch) {
    const itemId = batch.items[idx]?.id
    const itemKey = batch.items[idx]?.key
    Object.assign(batch.items[idx], updates)
    // 同步更新到 batchResults
    if (batchResults.value === batch.items && batchResults.value[idx]) {
      Object.assign(batchResults.value[idx], updates)
    }
    // 如果有数据库 ID，通过 API 更新
    if (itemId) {
      updateBatchItemAPI(batch.key, itemId, updates)
    }
  }
}

async function updateBatchItemAPI(batchKey, itemId, updates) {
  try {
    await request.put(`/image/batches/${batchKey}/items/${itemId}`, updates)
  } catch (e) {
    console.error('更新批次项失败:', e)
  }
}

// 从 API 加载批次列表（合并更新而非完全替换，保留本地 pending items）
async function loadTaskBatchesFromAPI() {
  try {
    const res = await request.get('/image/batches')
    const serverBatches = res.data || []

    // 合并更新：原地更新已存在的 batch，保留本地状态和响应式引用
    serverBatches.forEach((serverBatch: any) => {
      const existingBatch = taskBatches.value.find((b: any) => b.key === serverBatch.key)
      if (existingBatch) {
        // 更新基础字段
        existingBatch.status = serverBatch.status
        existingBatch.total = serverBatch.total
        existingBatch.doneCount = serverBatch.done_count
        existingBatch.failedCount = serverBatch.failed_count
        existingBatch.timeText = serverBatch.time_text
        existingBatch.referenceImages = (serverBatch.reference_images || []).map(resolveImageUrl).filter(Boolean)
        existingBatch.aspectRatio = normalizeBatchAspectRatio(serverBatch.aspect_ratio || serverBatch.subtitle)
        // 原地更新 items（不清空数组，保留响应式引用）
        const serverItems = serverBatch.items || []
        // 合并：更新已存在的 items，追加新 items
        existingBatch.items.forEach((item: any) => {
          // 跳过本地 pendingItem（它们没有 server id）
          if (item.isEditedResult || (!item.id && item.status === 'generating')) return
          // 找到对应的 server item
          const serverItem = serverItems.find((s: any) => s.id === item.id)
          if (serverItem) {
            item.imageUrl = serverItem.image_url ? resolveImageUrl(serverItem.image_url) : ''
            item.status = serverItem.status
            item.error = serverItem.error || ''
            item.savedToDesign = !!serverItem.saved_to_design
            item.imageId = serverItem.image_id
          }
        })

        // 如果批次已完成但仍有 item 处于 pending/generating，标记为失败
        // 这种情况说明后端批次状态与 item 状态不同步，需要修正
        if (serverBatch.status === 'done' || serverBatch.status === 'completed') {
          existingBatch.items.forEach((item: any) => {
            if (item.status === 'pending' || item.status === 'generating') {
              item.status = 'error'
              item.error = item.error || '生成未完成，请稍后重试'
            }
          })
        }

      } else {
        // 新批次，加入数组
        const newBatch: any = {
          id: serverBatch.id,
          key: serverBatch.key,
          title: serverBatch.title,
          subtitle: serverBatch.subtitle,
          cover: serverBatch.cover,
          referenceImages: (serverBatch.reference_images || []).map(resolveImageUrl).filter(Boolean),
          aspectRatio: normalizeBatchAspectRatio(serverBatch.aspect_ratio || serverBatch.subtitle),
          timeText: serverBatch.time_text,
          status: serverBatch.status,
          createdAt: serverBatch.created_at,
          total: serverBatch.total,
          doneCount: serverBatch.done_count,
          failedCount: serverBatch.failed_count,
          items: (serverBatch.items || []).map((item: any, idx: number) => ({
            id: item.id,
            key: `${serverBatch.key}-${idx}`,
            prompt: item.prompt,
            imageUrl: item.image_url ? resolveImageUrl(item.image_url) : '',
            status: item.status,
            error: item.error || '',
            mimeType: item.mime_type || 'image/png',
            savedToDesign: !!item.saved_to_design,
            imageId: item.image_id,
          })),
        }
        taskBatches.value.push(newBatch)
      }
    })

    Object.values(editTaskTrackers.value).forEach(tracker => {
      const batch = findBatchByTracker(tracker)
      if (batch) tracker.editingBatch = batch
    })

    // 更新 batchResults（用于显示当前批次）
    // 优先使用正在编辑的 batch（如果有 pending 生成中的 item）
    let targetBatch = null
    // 如果没有正在编辑的，使用最新的 batch
    if (!targetBatch && taskBatches.value.length > 0) {
      const latestBatch = [...taskBatches.value].sort((a: any, b: any) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      )[0]
      targetBatch = latestBatch
    }
    if (targetBatch) {
      batchResults.value = targetBatch.items
    }
    // 更新轮询状态
    // 检查所有 batch 中是否有任何 pending items，而不是只检查当前显示的 batch
    const anyPendingItems = taskBatches.value.some((batch: any) =>
      batch.items.some((item: any) => item.status === 'pending' || item.status === 'generating')
    )
    if (anyPendingItems) {
      isGenerating.value = true
      // 只有在没有运行轮询时才启动（避免重复启动）
      if (!pollTimer) {
        pollTimer = window.setInterval(() => {
          loadTaskBatchesFromAPI().catch((e: any) => {
            console.error('轮询失败:', e)
          })
        }, 1500)
      }
    } else if (isGenerating.value) {
      stopPolling()
      isGenerating.value = false
    }

    // 进度追踪：检查所有任务的状态
    taskBatches.value.forEach((batch: any) => {
      batch.items.forEach((item: any) => {
        const taskId = item.id?.toString() || item.key
        if (!taskId) return

        if (item.status === 'generating' || item.status === 'processing') {
          // 确保进度追踪器存在，并启动进度（如果还没启动）
          if (!progressTrackers.value[taskId]) {
            progressTrackers.value[taskId] = { current: 0 }
            // 尝试从 sessionStorage 恢复进度
            const persisted = fakeProgress.getPersistedProgress(taskId)
            if (persisted) {
              progressTrackers.value[taskId].current = persisted.progress
            }
            // 启动假进度
            fakeProgress.startProgress(taskId, (p) => {
              const tracker = progressTrackers.value[taskId]
              if (tracker) {
                tracker.current = p
              }
            })
          }
        } else if (item.status === 'pending') {
          // pending 状态也需要初始化进度追踪器（但不启动动画）
          if (!progressTrackers.value[taskId]) {
            progressTrackers.value[taskId] = { current: 0 }
            const persisted = fakeProgress.getPersistedProgress(taskId)
            if (persisted) {
              progressTrackers.value[taskId].current = persisted.progress
            }
          }
        } else if ((item.status === 'done' || item.status === 'error') && progressTrackers.value[taskId]) {
          // 任务完成或失败，停止进度
          fakeProgress.completeProgress(taskId)
          progressTrackers.value[taskId].current = 100
          delete progressTrackers.value[taskId]
        }
      })
    })

    // 懒加载图片
    nextTick(() => observeLazyImages())
  } catch (e) {
    console.error('加载批次列表失败:', e)
    stopPolling()
  }
}

// 创建批次（API）
async function createBatchAPI(title: string, subtitle: string, cover: string, referenceImages: string[], items: Array<{ prompt: string; status: string }>) {
  const res = await request.post('/image/batches', {
    title,
    subtitle,
    cover,
    reference_images: referenceImages,
    items: items.map(item => ({
      prompt: item.prompt,
      status: item.status,
    })),
  })
  return res.data
}

// ---------- History（localStorage 缓存 30 分钟）----------
const HISTORY_CACHE_KEY = 'ai_generate_history'
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
  } catch {
    return []
  }
}

function normalizeHistoryTask(item, index = 0) {
  if (!item || typeof item !== 'object') return null

  if (Array.isArray(item.items)) {
    const items = item.items
      .filter(entry => entry?.imageUrl)
      .map((entry, entryIndex) => ({
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
      aspectRatio: normalizeBatchAspectRatio(item.aspectRatio || item.subtitle),
      timeText: item.timeText || (item.createdAt ? new Date(item.createdAt).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : ''),
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
    aspectRatio: normalizeBatchAspectRatio(item.aspectRatio || item.subtitle),
    timeText: item.createdAt ? new Date(item.createdAt).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) : '',
    total: 1,
    doneCount: 1,
    failedCount: 0,
    createdAt: item.createdAt || '',
    items: [{
      id: item.id || null,
      prompt: item.prompt || '',
      imageUrl: item.imageUrl,
      status: 'done',
      error: '',
      savedToDesign: !!item.savedToDesign,
      key: `history-item-${index}-0`,
    }],
  }
}

function saveHistoryCache(items) {
  try {
    localStorage.setItem(HISTORY_CACHE_KEY, JSON.stringify({ items, savedAt: Date.now() }))
  } catch {
    /** ignore quota / private-mode errors */
  }
}

const history = ref(loadHistoryCache())

watch(history, (val) => {
  saveHistoryCache(val)
}, { deep: true })

// 模型切换时更新支持的分辨率
watch(selectedModel, () => {
  updateSupportedResolutions()
  if (aspectRatio.value !== 'original' && !dynamicAspectRatioOptions.value.includes(aspectRatio.value)) {
    aspectRatio.value = dynamicAspectRatioOptions.value[0] || '1:1'
  }
})

// ---------- Edit modal state ----------
const editModalVisible = ref(false)
const editingItem = ref(null)

// ---------- AI expand modal state ----------
const expandModalVisible = ref(false)
const expandImageUrl = ref('')
const expandOriginalAspectRatio = ref('1:1')

// ---------- Reference image edit modal state ----------
const refEditModalVisible = ref(false)
const refEditIndex = ref(-1)  // 当前编辑的 refImages 索引
const refEditImageUrl = ref('')

// ---------- Candidate confirmation modal state ----------
const candidateModalVisible = ref(false)
const candidateOriginalItem = ref<any>(null)
const candidateImageUrl = ref('')
const candidateTaskId = ref('')
const candidateLoading = ref(false)

// ---------- Image viewer (lightbox) state ----------
const viewerItem = ref<any>(null)
const viewerIndex = ref(0)

function openImageViewer(item, index) {
  viewerItem.value = item
  viewerIndex.value = index
}

function closeImageViewer() {
  viewerItem.value = null
}

// ---------- Static data ----------
const dynamicAspectRatioOptions = computed(() => {
  const model = drawingModels.value.find(m => m.name === selectedModel.value)
  const name = String(model?.name || '').toLowerCase()
  if (name.includes('gemini-3.1-flash-image')) return ['1:1', '2:3', '3:2', '3:4', '4:3', '4:5', '5:4', '9:16', '16:9', '1:4', '1:8', '4:1', '8:1', '21:9']
  if (name.includes('gemini-3-pro-image')) return ['1:1', '2:3', '3:2', '3:4', '4:3', '4:5', '5:4', '9:16', '16:9', '21:9']
  if (name.includes('gpt-image-2')) return ['1:1', '1:2', '2:1', '9:16', '16:9', '3:4', '4:3', '3:2', '2:3', '5:4', '4:5', '21:9', '9:21']
	if (isDoubaoSeedream50(name)) return ['1:1', '4:3', '3:4', '16:9', '9:16', '3:2', '2:3', '21:9']
  return ['1:1', '1:2', '2:1', '9:16', '16:9', '3:4', '4:3', '3:2', '2:3', '5:4', '4:5', '21:9', '9:21', '1:4', '1:8', '4:1', '8:1']
})
const ratioOptionsForCard = computed(() => [
  { value: 'original', label: '原图比例' },
  ...dynamicAspectRatioOptions.value.map(value => ({ value, label: value })),
])

watch(() => refImages.value.length, (length, previousLength) => {
  if (length > previousLength) aspectRatio.value = 'original'
  if (!length && aspectRatio.value === 'original') aspectRatio.value = '1:1'
})

const designLibraryTabs = [
  { value: 'generate', label: 'AI 生成' },
  { value: 'multi_angle', label: '多视角' },
  { value: 'eraser', label: '消除水印' },
  { value: 'remove-bg', label: '去除背景' },
]

const filteredDesignLibraryItems = computed(() => designLibraryItems.value)

function normalizeBatchAspectRatio(value: unknown) {
  const match = String(value || '').match(/(?:比例\s*)?(\d+(?:\.\d+)?):(\d+(?:\.\d+)?)/)
  if (!match) return '1:1'
  const width = Number(match[1])
  const height = Number(match[2])
  return width > 0 && height > 0 ? `${match[1]}:${match[2]}` : '1:1'
}

function ratioToCss(value: unknown) {
  const ratio = normalizeBatchAspectRatio(value)
  const [width, height] = ratio.split(':')
  return `${width} / ${height}`
}

// ---------- Computed ----------
const canGenerate = computed(() => {
  return prompt.value.trim().length > 0 && !!selectedModel.value && (sizeMode.value !== 'custom' || !sizeError.value)
})

const hasAnyResult = computed(() => batchResults.value.length > 0 || history.value.length > 0)

const displayTasks = computed(() => {
  const tasks = []

  // 按创建时间倒序（最新的在前）
  const sortedBatches = [...taskBatches.value].sort((a, b) =>
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  )

  // 显示所有批次（包括当前生成中的和历史批次）
  // 注意：pendingItem 已经在 loadTaskBatchesFromAPI 中合并到 batch.items 中，不需要额外处理
  sortedBatches.slice(0, 10).forEach((batch: any) => {
    // 计算真实的状态，确保 badge 与 items 状态一致
    const hasInProgress = batch.items.some((item: any) => item.status === 'pending' || item.status === 'generating')
    const doneCount = batch.items.filter((item: any) => item.status === 'done').length
    const failedCount = batch.items.filter((item: any) => item.status === 'error').length
    const allDone = !hasInProgress

    tasks.push({
      key: batch.key,
      type: hasInProgress ? 'current' : 'history',
      title: batch.title,
      subtitle: batch.subtitle,
      cover: batch.cover,
      referenceImages: Array.isArray(batch.referenceImages) ? batch.referenceImages : [],
      aspectRatio: normalizeBatchAspectRatio(batch.aspectRatio || batch.subtitle),
      timeText: batch.timeText,
      createdAt: batch.createdAt,
      total: batch.items.length,
      doneCount,
      failedCount,
      hasInProgress,
      submissionStatus: batch.submissionStatus || '',
      allDone,
      items: batch.items,
    })
  })

  return tasks
})

const batchDownloadItems = computed(() => displayTasks.value.flatMap((task: any, taskIndex: number) =>
  task.items.filter((item: any) => item.status === 'done' && item.imageUrl).map((item: any, index: number) => ({
    id: `${task.key}-${item.key || item.id || index}`,
    url: resolveImageUrl(item.imageUrl),
    thumbnail: getImageThumbnailUrl(item.imageUrl, 160),
    label: `${task.title || 'AI 图片'} · ${index + 1}`,
    filename: `ai-batch-${taskIndex + 1}-${index + 1}.png`,
  }))))
const batchDownloadSelectedItems = computed(() => batchDownloadItems.value.filter(item => batchDownloadSelectedIds.value.includes(item.id)))
const batchDownloadAllSelected = computed(() => batchDownloadItems.value.length > 0 && batchDownloadSelectedItems.value.length === batchDownloadItems.value.length)

// ---------- Methods ----------
function addRefImageFile(file) {
  if (!file || refImages.value.length >= 6) return
  if (!file.type.startsWith('image/')) return
  // 单张参考图限制 50MB，超过则提示（base64 约 67MB，留余量给 prompt + 批次图片）
  const maxRefSize = 50 * 1024 * 1024
  if (file.size > maxRefSize) {
    toast.error(`参考图 "${file.name}" 超过 50MB，请选择更小的图片`)
    return
  }
  const mimeType = file.type
  const reader = new FileReader()
  reader.onload = (ev) => {
    const dataUrl = ev.target.result as string | ArrayBuffer | null
    if (dataUrl && typeof dataUrl === 'string') {
      refImages.value.push({ data: dataUrl.split(',')[1], mimeType, url: dataUrl, thumbnailUrl: dataUrl })
    }
  }
  reader.readAsDataURL(file)
}

function handleRefImage(e) {
  const files = Array.from(e.target.files || []).filter(file => file.type.startsWith('image/'))
  const slots = 6 - refImages.value.length
  files.slice(0, slots).forEach(addRefImageFile)
  e.target.value = ''
}

function onRefDragOver(e) {
  if (refImages.value.length >= 6) return
  const types = e.dataTransfer?.types || []
  if (types.includes('Files')) {
    refDragOver.value = true
  }
}

function onRefDragLeave(e) {
  // 仅在离开整个容器时（relatedTarget 不在容器内）才取消高亮
  if (!e.currentTarget.contains(e.relatedTarget)) {
    refDragOver.value = false
  }
}

function onRefDrop(e) {
  refDragOver.value = false
  if (refImages.value.length >= 6) return
  const files = Array.from(e.dataTransfer?.files || []).filter(f => f.type.startsWith('image/'))
  const slots = 6 - refImages.value.length
  files.slice(0, slots).forEach(addRefImageFile)
}

function removeRefImage(idx) { refImages.value.splice(idx, 1) }

// 打开参考图编辑弹窗
function openRefImageEdit(idx: number) {
  refEditIndex.value = idx
  refEditImageUrl.value = refImages.value[idx].url
  refEditModalVisible.value = true
}

// 处理参考图编辑提交
function handleRefEditSubmit({
  imageUrl,
  maskData,
  imageData,
  mimeType,
  dataUrl,
}: {
  imageUrl: string
  maskData: string
  imageData?: string
  mimeType?: string
  dataUrl?: string
}) {
  if (refEditIndex.value === -1) return

  const current = refImages.value[refEditIndex.value]
  refImages.value[refEditIndex.value] = {
    ...current,
    data: imageData || current.data,
    mimeType: mimeType || current.mimeType,
    url: dataUrl || imageUrl || current.url,
    thumbnailUrl: dataUrl || imageUrl || current.thumbnailUrl,
    mask: maskData,
  }

  // 关闭弹窗
  refEditModalVisible.value = false
  refEditIndex.value = -1
  refEditImageUrl.value = ''

  toast.success('蒙版已保存')
}

async function appendReferenceImageFromUrl(url) {
  if (!url || refImages.value.length >= 6) return
  const { dataUrl, data, mimeType } = await fetchImageAsDataUrl(url)
  refImages.value.push({
    data,
    mimeType,
    url: dataUrl,
    thumbnailUrl: dataUrl,
  })
}

function appendReferenceImageFromAsset(asset) {
  if (!asset || refImages.value.length >= 6) return false
  const remoteUrl = asset.fileUrl || ''
  if (!remoteUrl) return false
  refImages.value.push({
    data: '',
    mimeType: mimeTypeFromUrl(remoteUrl),
    url: getDisplayImageUrl(remoteUrl),
    thumbnailUrl: getDisplayThumbnailUrl(remoteUrl, { width: 360, height: null, fit: 'lfit' }),
    remoteUrl,
  })
  return true
}

function mimeTypeFromUrl(url = '') {
  const path = String(url).split('?')[0].split('#')[0].toLowerCase()
  if (path.endsWith('.jpg') || path.endsWith('.jpeg')) return 'image/jpeg'
  if (path.endsWith('.webp')) return 'image/webp'
  if (path.endsWith('.gif')) return 'image/gif'
  if (path.endsWith('.bmp')) return 'image/bmp'
  if (path.endsWith('.avif')) return 'image/avif'
  if (path.endsWith('.svg')) return 'image/svg+xml'
  return 'image/png'
}

async function fetchDesignLibrary() {
  designLibraryLoading.value = true
  try {
    if (designLibrarySource.value === 'scene_generator') {
      const res = await request.get('/scene-generation/tasks', { params: { saved_only: 1 } })
      const tasks = (res.data.items || []).filter(item => item.isSavedToDesign)
      designLibraryItems.value = tasks.map(task => ({
        id: `scene-${task.taskId}`,
        type: 'scene_task',
        imageUrl: resolveImageUrl(task.imageUrls?.[0] || task.productImageUrl),
        title: [task.productCategory, task.styleName].filter(Boolean).join(' · ') || '场景图任务',
        subtitle: [task.productColorFamily, task.aspectRatio].filter(Boolean).join(' · '),
        prompt: task.referenceImageTitle || '',
        images: (task.imageUrls || []).map(resolveImageUrl).filter(Boolean),
      })).filter(item => item.imageUrl)
      return
    }

    const params = { image_only: 1, saved_only: 1, page: 1, page_size: 24 }
    if (designLibrarySource.value) params.source = designLibrarySource.value
    const res = await request.get('/usage-logs', { params })
    designLibraryItems.value = (res.data.data || []).map(item => ({
      id: `image-${item.id}`,
      type: 'image',
      imageUrl: resolveImageUrl(item.image_url),
      referenceImageUrl: resolveImageUrl(item.reference_image_url || ''),
      title: item.prompt || 'AI 生成图片',
      subtitle: item.created_at ? new Date(item.created_at).toLocaleDateString('zh-CN') : '',
      prompt: item.prompt || '',
      source: item.source || designLibrarySource.value,
    })).filter(item => item.imageUrl)
  } catch {
    designLibraryItems.value = []
    toast.error('设计库加载失败，请稍后重试')
  } finally {
    designLibraryLoading.value = false
  }
}

function openDesignLibrary() {
  assetPickerVisible.value = true
}

function openDesignLibraryGroup(item) {
  if (!item?.images?.length) return
  designLibraryGroup.value = item
}

function closeDesignLibraryGroup() {
  designLibraryGroup.value = null
}

function designTabClass(tab) {
  if (designLibrarySource.value === tab.value) return 'bg-neutral-900 text-white shadow-sm'
  return 'text-neutral-500 hover:text-neutral-900 hover:bg-neutral-50'
}

function designSourceLabel(item) {
  if (item.type === 'scene_task') return '场景图'
  const map = {
    generate: 'AI 生成',
    multi_angle: '多视角',
    eraser: '消除水印',
    'remove-bg': '去除背景',
  }
  return map[item.source] || '图片'
}

async function selectDesignLibraryImage(imageUrl, promptText = '') {
  if (!imageUrl) return
  try {
    await appendReferenceImageFromUrl(imageUrl)
    if (promptText && !prompt.value.trim()) prompt.value = promptText
    designLibraryVisible.value = false
    designLibraryGroup.value = null
    toast.success('已添加到参考图片')
  } catch {
    toast.error('添加参考图失败，请重试')
  }
}

async function selectAssetsAsReferenceImages(items) {
  const selected = Array.isArray(items) ? items : [items]
  let added = 0
  let failed = 0
  try {
    for (const asset of selected) {
      if (refImages.value.length >= 6) break
      if (appendReferenceImageFromAsset(asset)) {
        added += 1
      } else {
        failed += 1
      }
    }
    if (!added) throw new Error('没有可用图片')
    assetPickerVisible.value = false
    toast.success(failed ? `已添加 ${added} 张参考图，${failed} 张添加失败` : '已添加到参考图片')
  } catch {
    toast.error('添加资产参考图失败，请重试')
  }
}

async function pickDesignTemplate(item) {
  if (!item) return
  if (item.type === 'scene_task') {
    openDesignLibraryGroup(item)
    return
  }
  await selectDesignLibraryImage(item.imageUrl, item.prompt)
}

function onUploadAreaClick(event) {
  if (event?.target?.closest?.('[data-design-library-trigger="true"]')) return
  refUploadInput.value?.click()
}

function isSaving(item) {
  return !!item?.id && savingIds.value.includes(item.id)
}

function getItemAlt(item: any, index: any) {
  const idx = Number(index) + 1
  const prompt = String(item.prompt || '')
  return prompt ? `${prompt} ${idx}` : `结果${idx}`
}

function getImageThumbnailUrl(url: string, width = 640) {
  const resolved = resolveImageUrl(url)
  if (!resolved || resolved.startsWith('data:')) return resolved
  return getDisplayThumbnailUrl(resolved, { width, height: null, fit: 'lfit', quality: 82 })
}

function toggleBatchDownloadMode() {
  batchDownloadMode.value = !batchDownloadMode.value
  batchDownloadSelectedIds.value = []
  batchDownloadProgress.value = 0
  
}

function toggleBatchDownloadAll(checked: boolean) {
  batchDownloadSelectedIds.value = checked ? batchDownloadItems.value.map(item => item.id) : []
}

function toggleBatchDownloadItem(id: string) {
  batchDownloadSelectedIds.value = batchDownloadSelectedIds.value.includes(id)
    ? batchDownloadSelectedIds.value.filter(itemId => itemId !== id)
    : [...batchDownloadSelectedIds.value, id]
}

async function confirmBatchDownload() {
  if (batchDownloading.value || !batchDownloadSelectedItems.value.length) return
  batchDownloading.value = true
  batchDownloadProgress.value = 0
  let failed = 0
  let downloadMeta = {}
  for (const [index, item] of batchDownloadSelectedItems.value.entries()) {
    try {
      const result = await downloadImage(item.url, item.filename, index === 0 ? {} : { brand: downloadMeta.brand, sku: downloadMeta.sku, skipSkuPicker: true })
      if (index === 0 && result === null) break
      if (index === 0 && result) downloadMeta = result
    } catch {
      failed += 1
    }
    batchDownloadProgress.value = index + 1
    await new Promise(resolve => setTimeout(resolve, 180))
  }
  batchDownloading.value = false
  batchDownloadMode.value = false
  toast[failed ? 'error' : 'success'](failed ? `${failed} 张图片下载失败，请重试` : `已逐张下载 ${batchDownloadProgress.value} 张图片`)
}

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

// ========== 蒙版处理辅助函数 ==========

function loadImageFromBase64(base64: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = base64.startsWith('data:') ? base64 : `data:image/png;base64,${base64}`
  })
}

async function mergeImageAndMask(imageData: string, maskData: string): Promise<string> {
  const [image, mask] = await Promise.all([
    loadImageFromBase64(imageData),
    loadImageFromBase64(maskData)
  ])

  // 使用蒙版尺寸作为 canvas 尺寸（用户实际绘制的尺寸）
  const canvas = document.createElement('canvas')
  canvas.width = mask.width
  canvas.height = mask.height
  const ctx = canvas.getContext('2d')!

  // 1. 先画原图（缩放到蒙版尺寸）
  ctx.drawImage(image, 0, 0, mask.width, mask.height)
  // 2. 叠加蒙版笔触（黑色笔触区域 = 蒙版区域，AI 会修改这些区域）
  ctx.drawImage(mask, 0, 0)

  return canvas.toDataURL('image/png').split(',')[1]
}

async function uploadGenerationRefImage(data: string, mimeType = 'image/png') {
  const form = new FormData()
  form.append('image', base64ToBlob(data, mimeType || 'image/png'), `ref.${extensionFromMimeType(mimeType || 'image/png')}`)
  form.append('mime_type', mimeType || 'image/png')
  const res = await request.post('/upload/image', form)
  const url = res.data?.url
  if (!url) throw new Error('参考图上传失败')
  return { url, mime_type: mimeType || 'image/png' }
}

function base64ToBlob(data: string, mimeType = 'image/png') {
  const clean = String(data || '').includes(',') ? String(data).split(',').pop() || '' : String(data || '')
  const binary = atob(clean)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return new Blob([bytes], { type: mimeType })
}

function extensionFromMimeType(mimeType = 'image/png') {
  if (mimeType.includes('jpeg') || mimeType.includes('jpg')) return 'jpg'
  if (mimeType.includes('webp')) return 'webp'
  if (mimeType.includes('gif')) return 'gif'
  return 'png'
}

async function buildGenerationRefPayload(img) {
  if (img.mask) {
    if (!img.data) {
      const loaded = await fetchImageAsDataUrl(img.remoteUrl || img.url)
      img.data = loaded.data
      img.mimeType = loaded.mimeType
    }
    const mergedData = await mergeImageAndMask(img.data, img.mask)
    return uploadGenerationRefImage(mergedData, img.mimeType)
  }
  if (img.remoteUrl) {
    return {
      url: img.remoteUrl,
      mime_type: img.mimeType || mimeTypeFromUrl(img.remoteUrl),
    }
  }
  return uploadGenerationRefImage(img.data, img.mimeType)
}

async function generate() {
  if (!canGenerate.value) return

  const text = prompt.value.trim()
  const newPrompts = count.value > 1 ? Array(count.value).fill(text) : [text]

  // 固定本次提交的数据，上传期间修改表单不影响已显示的任务。
  const refs = refImages.value.map(img => ({ ...img }))
  const outputSize = sizeMode.value === 'custom' ? { width: Number(customWidth.value), height: Number(customHeight.value) } : null
  const useOriginalRatio = !outputSize && aspectRatio.value === 'original'
  const supportedRatios = [...dynamicAspectRatioOptions.value]
  let aspectRatioVal = outputSize ? customRatio.value : (useOriginalRatio ? '1:1' : aspectRatio.value)
  const resolutionVal = outputSize ? (customGenerationSize.value?.resolution || resolution.value) : resolution.value
  const modelName = selectedModel.value
  const localKey = `batch-${crypto.randomUUID()}`
  const batchTitle = text || '未填写提示词'
  let batchSubtitle = `${newPrompts.length} 张 · 比例 ${useOriginalRatio ? '原图比例' : aspectRatioVal}${resolutionVal ? ` · ${resolutionVal}` : ''}`
  if (outputSize) batchSubtitle = `${newPrompts.length} 张 · ${outputSize.width} × ${outputSize.height} px · 比例 ${outputSize.width}:${outputSize.height}${resolutionVal ? ` · ${resolutionVal}` : ''}`
  const batchCover = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="160" height="160"><rect width="100%" height="100%" fill="%23f3f4f6"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="%239ca3af" font-size="14">AI</text></svg>'

  const initialItems = newPrompts.map(promptText => ({
    prompt: promptText,
    status: 'pending',
  }))

  // 在任何上传或创建请求之前显示本地任务，成功后原地补齐服务端标识。
  const batch: any = reactive({
    id: null,
    key: localKey,
    submissionStatus: refs.length ? '上传参考图' : '提交任务',
    title: batchTitle,
    subtitle: batchSubtitle,
    aspectRatio: outputSize ? `${outputSize.width}:${outputSize.height}` : aspectRatioVal,
    outputSize,
    cover: batchCover,
    referenceImages: refs.map(img => img.url || img.remoteUrl).filter(Boolean),
    timeText: new Date().toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }),
    status: 'running',
    createdAt: new Date().toISOString(),
    items: newPrompts.map((p, i) => ({
      id: null,
      key: `${localKey}-${i}`,
      prompt: p,
      imageUrl: '',
      status: 'pending',
      error: '',
      savedToDesign: false,
    })),
  })
  taskBatches.value.push(batch)
  // 确保 batchResults 指向当前正在生成的 batch
  // 避免被 editTaskTrackers 中残留的 tracker 或之前的 batch 覆盖
  batchResults.value = batch.items
  isGenerating.value = true

  batchProgress.value = { current: 0, total: batch.items.length }
  globalError.value = ''

  // 让任务卡片先完成渲染，再执行可能较耗时的图片转换与上传。
  await nextTick()
  await new Promise<void>(resolve => requestAnimationFrame(() => { setTimeout(resolve, 0) }))

  try {
    if (useOriginalRatio) {
      aspectRatioVal = await resolveReferenceAspectRatio(refs, supportedRatios)
      batchSubtitle = `${newPrompts.length} 张 · 比例 ${aspectRatioVal}${resolutionVal ? ` · ${resolutionVal}` : ''}`
      batch.aspectRatio = aspectRatioVal
      batch.subtitle = batchSubtitle
    }
    const refPayload = refs.length > 0
      ? await Promise.all(refs.map(buildGenerationRefPayload))
      : null
    const referenceImageUrls = (refPayload || [])
      .map((item: any) => resolveImageUrl(item?.url || ''))
      .filter(Boolean)
    batch.referenceImages = referenceImageUrls
    batch.submissionStatus = '提交任务'
    const serverBatch = await createBatchAPI(batchTitle, batchSubtitle, batchCover, referenceImageUrls, initialItems)
    // 并行轮询可能先读到刚创建的服务端批次，保留已经显示的本地卡片。
    taskBatches.value = taskBatches.value.filter(existing => existing === batch || existing.key !== serverBatch.key)
    batch.id = serverBatch.id
    batch.key = serverBatch.key
    batch.items.forEach((item, i) => { item.id = serverBatch.items[i].id })
    batch.submissionStatus = ''

    // 每个请求绑定自己的批次，连续提交时不会更新到另一个任务。
    const generationRequests = newPrompts.map((p, i) =>
      doGenerateSingle(i, p, aspectRatioVal, resolutionVal, refPayload, batch, modelName))
    startPolling()
    const results = await Promise.allSettled(generationRequests)
    results.forEach((result, i) => {
      if (result.status === 'rejected') {
        const message = result.reason?.message || '任务提交失败'
        updateBatchItem(i, { status: 'error', error: message }, batch)
      }
    })
  } catch (e) {
    const errMsg = (e as any)?.response?.data?.error || (e as Error)?.message || '任务提交失败'
    batch.submissionStatus = ''
    batch.status = 'error'
    batch.items.forEach((item, i) => updateBatchItem(i, { status: 'error', error: errMsg }, batch))
    toast.error(errMsg)
  } finally {
    isGenerating.value = taskBatches.value.some(batch =>
      batch.items.some(item => item.status === 'pending' || item.status === 'generating'))
    if (!isGenerating.value) stopPolling()
  }
}

// 提交异步任务，由全局轮询统一更新状态
async function doGenerateSingle(idx, promptText, aspectRatioVal, resolutionVal, refPayload, batch, modelName) {
  updateBatchItem(idx, { status: 'generating' }, batch)
  const batchKey = batch?.key
  const itemId = batch?.items[idx]?.id

  const payload: any = { prompt: promptText, aspect_ratio: aspectRatioVal, model: modelName }
  if (batch.outputSize) {
    payload.output_width = batch.outputSize.width
    payload.output_height = batch.outputSize.height
  }
  if (resolutionVal) payload.resolution = resolutionVal
  if (refPayload) payload.ref_images = refPayload
  if (batchKey && itemId) {
    payload.batch_key = batchKey
    payload.batch_item_id = itemId
  }

  let asyncErrorMsg = ''
  try {
    await request.post('/image/generate-async', payload)
    // 全局轮询会负责更新任务状态，无需等待
  } catch (e) {
    // 如果异步 API 不可用，回退到同步模式
    const errData = (e as any)?.response?.data
    const errMsg = typeof errData === 'string' ? errData : (errData?.error || (e as Error)?.message || '')
    asyncErrorMsg = errMsg
    // 旧同步接口没有自定义尺寸能力，不能回退后交付错误尺寸的图片。
    if (batch.outputSize) {
      updateBatchItem(idx, { status: 'error', error: errMsg || '自定义尺寸任务提交失败，请重试' }, batch)
      return
    }
    // 401 直接重抛，不要静默 fallback，避免掩盖真实原因
    if ((e as any)?.response?.status === 401) {
      updateBatchItem(idx, { status: 'error', error: '登录已失效，请刷新页面重新登录' }, batch)
      return
    }
    try {
      const res = await request.post('/image/generate', payload)
      const data = res.data
      const imageUrl = resolveImageUrl(data.image_url) || (data.image_data ? `data:${data.mime_type};base64,${data.image_data}` : '')
      updateBatchItem(idx, { imageUrl, status: 'done', savedToDesign: false }, batch)
    } catch (syncError) {
      const syncErrData = (syncError as any)?.response?.data
      const syncErrMsg = typeof syncErrData === 'string' ? syncErrData : (syncErrData?.error || (syncError as Error)?.message || '')
      if ((syncError as any)?.response?.status === 401) {
        updateBatchItem(idx, { status: 'error', error: '登录已失效，请刷新页面重新登录' }, batch)
        return
      }
      // 优先显示后端真实错误，两个接口都失败时拼接提示
      const finalMsg = syncErrMsg || asyncErrorMsg || '本次未能获取到结果，请稍后重试'
      updateBatchItem(idx, { status: 'error', error: finalMsg }, batch)
    }
  }
}

function viewHistory(task) {
  if (!task?.items?.length) return
  batchResults.value = task.items.map(item => ({
    id: item.id || null,
    prompt: item.prompt,
    imageUrl: item.imageUrl,
    status: 'done',
    error: '',
    savedToDesign: !!item.savedToDesign,
  }))
  prompt.value = task.items[0]?.prompt || ''
}

function openEdit(item) {
  console.log('[openEdit] item:', item, 'id:', item?.id, 'imageUrl:', item?.imageUrl)
  editingItem.value = item
  editModalVisible.value = true
}

async function openExpand(item, originalAspectRatio) {
  try {
    const source = await fetchImageAsDataUrl(item.imageUrl)
    editingItem.value = item
    expandImageUrl.value = source.dataUrl
    expandOriginalAspectRatio.value = normalizeBatchAspectRatio(originalAspectRatio)
    expandModalVisible.value = true
  } catch (error) {
    toast.error(error?.message || '扩图图片加载失败，请重试')
  }
}

function closeExpandModal() {
  expandModalVisible.value = false
  expandImageUrl.value = ''
}

function handleExpandSubmit(payload) {
  handleEditSubmit({ ...payload, source: 'expand' })
}

function openCandidateModal(item: any) {
  const trackerEntry = findTrackerByItem(item)
  const tracker = trackerEntry?.[1]
  if (!tracker) return

  const targetItem = findItemByTracker(tracker) || item
  candidateTaskId.value = trackerEntry?.[0] || ''
  candidateOriginalItem.value = targetItem
  candidateImageUrl.value = targetItem.candidateImageUrl || item.imageUrl
  candidateModalVisible.value = true
}

// 编辑任务的跟踪映射
interface EditTaskTracker {
  originalItem: any
  pendingItem: any
  serverTaskId?: string
  editingBatch: any  // 保存 batch 引用，用于 pollEditTask 中更新 batchResults
  editingBatchIdx: number
  batchKey?: string
  itemKey?: string
  itemId?: number | string
}
const editTaskTrackers = ref<Record<string, EditTaskTracker>>({})  // taskId -> { originalItem, pendingItem, editingItemIndex, editingBatchIdx }

function findBatchByTracker(tracker?: EditTaskTracker | null) {
  if (!tracker) return null
  if (tracker.batchKey) {
    const batch = taskBatches.value.find((b: any) => b.key === tracker.batchKey)
    if (batch) return batch
  }
  if (tracker.editingBatch && taskBatches.value.includes(tracker.editingBatch)) {
    return tracker.editingBatch
  }
  return null
}

function findItemByTracker(tracker?: EditTaskTracker | null) {
  const batch = findBatchByTracker(tracker)
  if (!batch || !tracker) return null
  if (tracker.itemId) {
    const item = batch.items.find((i: any) => i.id === tracker.itemId)
    if (item) return item
  }
  if (tracker.itemKey) {
    const item = batch.items.find((i: any) => i.key === tracker.itemKey)
    if (item) return item
  }
  return batch.items.find((i: any) => i === tracker.originalItem || i === tracker.pendingItem) || null
}

function findTrackerByItem(item: any) {
  return Object.entries(editTaskTrackers.value).find(([, tracker]) => findItemByTracker(tracker) === item)
}

function closeCandidateModal() {
  console.log('[closeCandidateModal] called, candidateLoading:', candidateLoading.value, 'candidateModalVisible:', candidateModalVisible.value)
  if (candidateLoading.value) {
    console.log('[closeCandidateModal] returning early due to candidateLoading')
    return
  }
  candidateModalVisible.value = false
  candidateOriginalItem.value = null
  candidateImageUrl.value = ''
  candidateTaskId.value = ''
  console.log('[closeCandidateModal] done, candidateModalVisible:', candidateModalVisible.value)
}

async function acceptCandidate() {
  if (candidateLoading.value) return
  candidateLoading.value = true
  const taskId = candidateTaskId.value
  let shouldCloseModal = false
  try {
    const tracker = editTaskTrackers.value[taskId]
    const foundBatch = findBatchByTracker(tracker)
    const foundItem = findItemByTracker(tracker)

    if (!foundBatch || !foundItem || !foundItem.candidateImageUrl) {
      toast.error('未找到当前编辑结果，请刷新后重试')
      return
    }

    foundItem.imageUrl = foundItem.candidateImageUrl
    foundItem.savedToDesign = false
    foundItem.candidateStatus = ''
    foundItem.candidateImageUrl = ''
    foundItem.candidateErrorMessage = ''
    foundItem.candidatePrompt = ''

    if (foundItem.id) {
      updateBatchItemAPI(foundBatch.key, foundItem.id, {
        image_url: foundItem.imageUrl,
        saved_to_design: false,
      })
    }

    taskBatches.value = [...taskBatches.value]
    toast.success('已采用编辑结果')
    shouldCloseModal = true
  } catch (e) {
    toast.error(e?.response?.data?.error || '采用失败，请稍后重试')
  } finally {
    candidateLoading.value = false
    if (shouldCloseModal) closeCandidateModal()
    if (taskId) delete editTaskTrackers.value[taskId]
  }
}

function rejectCandidate() {
  if (candidateLoading.value) return
  const taskId = candidateTaskId.value
  const tracker = editTaskTrackers.value[taskId]
  const foundItem = findItemByTracker(tracker) || candidateOriginalItem.value

  if (foundItem) {
    foundItem.candidateStatus = ''
    foundItem.candidateImageUrl = ''
    foundItem.candidateErrorMessage = ''
    foundItem.candidatePrompt = ''
  }

  if (taskId) delete editTaskTrackers.value[taskId]
  taskBatches.value = [...taskBatches.value]
  toast.info('已保留原图')
  closeCandidateModal()
}

// ---------- Edit task polling state ----------
let editPollTimer: number | null = null

// 轮询编辑任务结果（独立于批次轮询）
async function pollEditTask(taskId: string) {
  const tracker = editTaskTrackers.value[taskId]
  if (!tracker) return

  try {
    const res = await request.get(`/image/generate-async/${tracker.serverTaskId}`)
    const taskStatus = res.data?.status
    const taskResult = res.data

    console.log('[pollEditTask] API response:', JSON.stringify(res.data))
    console.log('[pollEditTask] taskId:', taskId, 'serverTaskId:', tracker.serverTaskId)
    console.log('[pollEditTask] full response:', res)
    console.log('[pollEditTask] res.data:', res.data)
    console.log('[pollEditTask] taskStatus:', taskStatus, 'typeof:', typeof taskStatus, 'length:', taskStatus?.length)
    console.log('[pollEditTask] char codes:', [...(taskStatus || '')].map(c => c.charCodeAt(0)))
    console.log('[pollEditTask] taskStatus === "done":', taskStatus === 'done', 'taskStatus === "done\\n":', taskStatus === 'done\n')
    console.log('[pollEditTask] pendingItem.candidateStatus (before):', tracker.pendingItem.candidateStatus)
    console.log('[pollEditTask] editingBatch:', tracker.editingBatch)

    // 可能的 status 值: queued, running, done, failed
    if (taskStatus === 'done') {
      const targetItem = findItemByTracker(tracker)
      if (targetItem) {
        targetItem.candidateImageUrl = resolveImageUrl(taskResult?.image_url) || ''
        targetItem.candidateStatus = 'ready'
      }
      taskBatches.value = [...taskBatches.value]
      stopEditPolling()
    } else if (taskStatus === 'failed') {
      const targetItem = findItemByTracker(tracker)
      if (targetItem) {
        targetItem.candidateStatus = 'failed'
        targetItem.candidateErrorMessage = taskResult?.error || '生成失败'
      }
      taskBatches.value = [...taskBatches.value]
      toast.error('编辑预览生成失败')
      delete editTaskTrackers.value[taskId]
      stopEditPolling()
    } else {
      const targetItem = findItemByTracker(tracker)
      if (targetItem) targetItem.candidateStatus = 'generating'
      taskBatches.value = [...taskBatches.value]
      if (editPollTimer) window.clearTimeout(editPollTimer)
      editPollTimer = window.setTimeout(() => pollEditTask(taskId), 1500)
    }
  } catch (e) {
    console.error('轮询编辑任务失败:', e)
    // 继续尝试
    if (editPollTimer) window.clearTimeout(editPollTimer)
    editPollTimer = window.setTimeout(() => pollEditTask(taskId), 1500)
  }
}

function stopEditPolling() {
  if (editPollTimer) {
    window.clearTimeout(editPollTimer)
    editPollTimer = null
  }
}

async function handleEditSubmit({ prompt: editPromptText, imageData, mimeType, aspectRatio: requestedAspectRatio, resolution: requestedResolution, source = 'edit' }) {
  // 保存原始 item 引用
  const originalItem: any = editingItem.value
  console.log('[handleEditSubmit] originalItem:', originalItem, 'id:', originalItem?.id)
  if (!originalItem) {
    editModalVisible.value = false
    return
  }

  // 关闭编辑弹窗
  editModalVisible.value = false
  if (source === 'expand') closeExpandModal()

  const targetAspectRatio = requestedAspectRatio || (aspectRatio.value === 'original'
    ? normalizeBatchAspectRatio(taskBatches.value.find(batch => batch.items.includes(originalItem))?.aspectRatio)
    : aspectRatio.value)
  const targetResolution = requestedResolution || resolution.value

  // 在原 item 上设置候选状态，而不是创建新 item
  originalItem.candidateStatus = 'generating'
  originalItem.candidateImageUrl = ''
  originalItem.candidateErrorMessage = ''
  originalItem.candidatePrompt = editPromptText
  batchResults.value = [...batchResults.value]
  console.log('[handleEditSubmit] editing started for item:', originalItem.id)

  // 生成唯一任务 ID 用于跟踪
  const editTaskId = `edit-${Date.now()}`

  // 直接找到包含 originalItem 的 batch，避免依赖 findBatchByItem（item.id 可能为 null）
  const editingBatch = taskBatches.value.find(b =>
    b.items.includes(originalItem) ||
    (originalItem.id && b.items.some((i: any) => i.id === originalItem.id))
  ) || null

  // 保存跟踪信息
  editTaskTrackers.value[editTaskId] = {
    originalItem,
    pendingItem: originalItem,  // pendingItem 现在指向同一个对象
    editingBatch,
    editingBatchIdx: 0,
    batchKey: editingBatch?.key,
    itemKey: originalItem.key,
    itemId: originalItem.id,
  }

  // 尝试异步模式
  try {
    const asyncRefImage = await uploadGenerationRefImage(imageData, mimeType)
    const payload: any = {
      prompt: editPromptText,
      aspect_ratio: targetAspectRatio,
      model: selectedModel.value,
      ref_images: [asyncRefImage],
    }
    if (targetResolution) payload.resolution = targetResolution

    const res = await request.post('/image/generate-async', payload)
    const serverTaskId = res.data?.task_id

    console.log('[handleEditSubmit] API response:', res.data)
    console.log('[handleEditSubmit] serverTaskId:', serverTaskId)
    console.log('[handleEditSubmit] editingBatch:', editingBatch)

    // 保存跟踪信息并启动独立轮询
    if (serverTaskId) {
      editTaskTrackers.value[editTaskId].serverTaskId = serverTaskId
      // 启动独立轮询监控编辑结果
      pollEditTask(editTaskId)
    } else {
      // 如果没有返回 task_id，直接更新
      originalItem.candidateImageUrl = resolveImageUrl(res.data?.image_url) || ''
      // 不要用 task result id 覆盖原 item id，否则 acceptCandidate 无法正确持久化
      originalItem.candidateStatus = 'ready'
      batchResults.value = [...batchResults.value]
      // 显示候选弹窗
      candidateTaskId.value = editTaskId
      candidateOriginalItem.value = originalItem
      candidateImageUrl.value = originalItem.candidateImageUrl
      candidateModalVisible.value = true
    }
  } catch (e) {
    // 回退到同步模式
    try {
      const payload: any = {
        prompt: editPromptText,
        aspect_ratio: targetAspectRatio,
        model: selectedModel.value,
        ref_images: [{ data: imageData, mime_type: mimeType }],
      }
      if (targetResolution) payload.resolution = targetResolution
      const res = await request.post('/image/generate', payload)
      const data = res.data
      originalItem.candidateImageUrl = resolveImageUrl(data.image_url) || (data.image_data ? `data:${data.mime_type};base64,${data.image_data}` : '')
      // 不要用 task result id 覆盖原 item id，否则 acceptCandidate 无法正确持久化
      originalItem.candidateStatus = 'ready'
      batchResults.value = [...batchResults.value]
      // 显示候选弹窗
      candidateTaskId.value = editTaskId
      candidateOriginalItem.value = originalItem
      candidateImageUrl.value = originalItem.candidateImageUrl
      candidateModalVisible.value = true
    } catch (syncError) {
      originalItem.candidateStatus = 'failed'
      originalItem.candidateErrorMessage = '本次未能获取到结果，请稍后重试'
      batchResults.value = [...batchResults.value]
      toast.error('编辑预览生成失败，请稍后重试')
    }
  }
}

function updateSupportedResolutions() {
  // 如果当前选中的模型有 supported_resolutions，使用它
  const currentModel = drawingModels.value.find(m => m.name === selectedModel.value)
  if (isDoubaoSeedream50(currentModel?.name)) {
    supportedResolutions.value = ['1K', '2K']
  } else if (currentModel?.supported_resolutions?.length) {
    supportedResolutions.value = currentModel.supported_resolutions.map(r => String(r).toUpperCase())
  } else {
    // 否则使用默认分辨率列表
    supportedResolutions.value = ['1K', '2K']
  }
  // 如果当前选中的分辨率不在支持列表中，重置为第一个
  resolution.value = String(resolution.value || '').toUpperCase()
  if (!supportedResolutions.value.includes(resolution.value)) {
    resolution.value = supportedResolutions.value[0] || '1K'
  }
}

async function loadDrawingModels() {
  try {
    const res = await request.get('/models', { params: { type: 'drawing', _: Date.now() } })
    // 过滤仅活跃的绘图模型
    drawingModels.value = (res.data || []).filter((m: any) =>
      m.is_active && m.types?.includes('drawing')
    )
    // 保留当前选择；当前模型不可用时再回退到默认模型
    if (!drawingModels.value.some((m: any) => m.name === selectedModel.value)) {
      const defaultModel = drawingModels.value.find((m: any) => m.is_default)
      selectedModel.value = defaultModel?.name || drawingModels.value[0]?.name || ''
    }
    // 更新支持的分辨率
    updateSupportedResolutions()
  } catch {
    drawingModels.value = []
  }
}

function refreshDrawingModelsWhenVisible() {
  if (document.visibilityState === 'visible') loadDrawingModels()
}

onMounted(() => {
  if (route.query.prompt) {
    prompt.value = String(route.query.prompt)
  }
  loadDrawingModels()
  modelPollTimer = window.setInterval(loadDrawingModels, 10_000)
  document.addEventListener('visibilitychange', refreshDrawingModelsWhenVisible)
  loadTaskBatchesFromAPI()
})

const DOUBAO_SIZE_MAP = {
  '1K': { '1:1': '1024 × 1024', '4:3': '1152 × 864', '3:4': '864 × 1152', '16:9': '1424 × 800', '9:16': '800 × 1424', '3:2': '1248 × 832', '2:3': '832 × 1248', '21:9': '1568 × 672' },
  '2K': { '1:1': '2048 × 2048', '4:3': '2368 × 1776', '3:4': '1776 × 2368', '16:9': '2816 × 1584', '9:16': '1584 × 2816', '3:2': '2496 × 1664', '2:3': '1664 × 2496', '21:9': '3136 × 1344' }
}
const isDoubaoSeedream50 = (name) => String(name || '').toLowerCase().split('/').pop() === 'doubao-seedream-5-0-pro-260628'
const GPT_SIZE_MAP = {
  '1K': { '1:1': '1024 × 1024', '4:3': '1536 × 1152', '3:4': '1152 × 1536', '16:9': '1536 × 864', '9:16': '864 × 1536', '4:5': '832 × 1040', '5:4': '1040 × 832', '21:9': '1504 × 624', '2:1': '1408 × 704', '1:2': '704 × 1408', '3:2': '1344 × 896', '2:3': '896 × 1344', '9:21': '624 × 1504' },
  '2K': { '1:1': '2048 × 2048', '4:3': '2048 × 1536', '3:4': '1536 × 2048', '16:9': '2048 × 1152', '9:16': '1152 × 2048', '4:5': '1664 × 2080', '5:4': '2080 × 1664', '21:9': '2928 × 1200', '2:1': '2304 × 1152', '1:2': '1152 × 2304', '3:2': '2112 × 1408', '2:3': '1408 × 2112', '9:21': '864 × 2016' },
  '4K': { '1:1': '4096 × 4096', '4:3': '2880 × 2160', '3:4': '2160 × 2880', '16:9': '3200 × 1800', '9:16': '1800 × 3200', '4:5': '2560 × 3200', '5:4': '3200 × 2560', '21:9': '3360 × 1440', '2:1': '3200 × 1600', '1:2': '1600 × 3200', '3:2': '2880 × 1920', '2:3': '1920 × 2880', '9:21': '1440 × 3360' }
}
// Gemini 的输出尺寸按官方清晰度分别配置，不能由 1K 像素值推算。
const GEMINI_SIZE_MAP = {
  '1K': { '1:1': '1024 × 1024', '2:3': '848 × 1264', '3:2': '1264 × 848', '3:4': '896 × 1200', '4:3': '1200 × 896', '4:5': '928 × 1152', '5:4': '1152 × 928', '9:16': '768 × 1376', '16:9': '1376 × 768', '21:9': '1584 × 672', '1:4': '512 × 2048', '4:1': '2048 × 512', '1:8': '384 × 3072', '8:1': '3072 × 384' },
  '2K': { '1:1': '2048 × 2048', '2:3': '1696 × 2528', '3:2': '2528 × 1696', '3:4': '1792 × 2400', '4:3': '2400 × 1792', '4:5': '1856 × 2304', '5:4': '2304 × 1856', '9:16': '1536 × 2752', '16:9': '2752 × 1536', '21:9': '3168 × 1344', '1:4': '1024 × 4096', '4:1': '4096 × 1024', '1:8': '768 × 6144', '8:1': '6144 × 768' },
  '4K': { '1:1': '4096 × 4096', '2:3': '3392 × 5056', '3:2': '5056 × 3392', '3:4': '3584 × 4800', '4:3': '4800 × 3584', '4:5': '3712 × 4608', '5:4': '4608 × 3712', '9:16': '3072 × 5504', '16:9': '5504 × 3072', '21:9': '6336 × 2688', '1:4': '2048 × 8192', '4:1': '8192 × 2048', '1:8': '1536 × 12288', '8:1': '12288 × 1536' }
}
const geminiPixelSize = (ratio, quality) => GEMINI_SIZE_MAP[String(quality || '1K').toUpperCase()]?.[ratio] || ''
const ratioPixelSize = (ratio, requestedResolution = resolution.value) => {
  const name = selectedModel.value.toLowerCase()
  const quality = String(requestedResolution || '1K').toUpperCase()
  if (isDoubaoSeedream50(name)) return DOUBAO_SIZE_MAP[quality]?.[ratio] || ''
  if (name.includes('gpt-image')) return GPT_SIZE_MAP[quality]?.[ratio] || ''
  if (name.includes('gemini-3.1-flash-image') || name.includes('gemini-3-pro-image')) return geminiPixelSize(ratio, quality)
  return ''
}
const ratioShapeStyle = (ratio) => {
  const [w, h] = ratio.split(':').map(Number)
  const scale = Math.min(44 / w, 38 / h)
  const minEdge = Math.max(w, h) >= 8 ? 7 : 9
  return { width: `${Math.max(minEdge, Math.round(w * scale))}px`, height: `${Math.max(minEdge, Math.round(h * scale))}px` }
}
const selectedRatioSize = computed(() => ratioPixelSize(aspectRatio.value))

onBeforeUnmount(() => {
  clearViolationFile()
  stopPolling()
  if (modelPollTimer) window.clearInterval(modelPollTimer)
  document.removeEventListener('visibilitychange', refreshDrawingModelsWhenVisible)
})
</script>

<style scoped>
/* Lightbox card: scale in when backdrop fades in, scale out when fading out */
.viewer-card {
  animation: viewer-in 0.25s cubic-bezier(0.34, 1.4, 0.64, 1) both;
}

/* When the parent Transition leave is active the backdrop class changes — we
   use a sibling selector trick via the backdrop's opacity transition timing:
   add a second animation targeting the leave phase. */
.v-leave-active .viewer-card {
  animation: viewer-out 0.2s ease-in both;
}

@keyframes viewer-in {
  from {
    opacity: 0;
    transform: scale(0.88);
  }

  to {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes viewer-out {
  from {
    opacity: 1;
    transform: scale(1);
  }

  to {
    opacity: 0;
    transform: scale(0.92);
  }
}

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

.preview-download-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

/* Candidate Confirmation Modal */
.candidate-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.candidate-pane {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.candidate-label {
  font-size: 12px;
  font-weight: 700;
  color: #6b7280;
}

.candidate-frame {
  position: relative;
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  overflow: hidden;
  background: #f8fafc;
  aspect-ratio: 1/1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.candidate-frame img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #fff;
}

.task-card {
  content-visibility: auto;
  contain-intrinsic-size: 360px;
}

.result-image-tile {
  contain: layout paint style;
}

.result-image-tile--square {
  max-width: 280px;
  margin-inline: auto;
}

@media (max-width: 640px) {
  .result-image-tile--square {
    max-width: 100%;
  }

  .candidate-grid {
    grid-template-columns: 1fr;
  }

  .candidate-frame {
    aspect-ratio: 4/3;
  }
}

.ratio-visual-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.prompt-panel { box-shadow: 0 6px 20px rgba(15,23,42,.035); }
.prompt-textarea { min-height: 184px; max-height: 360px; }
.generate-params-panel { border: 1px solid #e2e5e9; border-radius: 14px; padding: 0; overflow: hidden; background: #fff; }
.generate-params-heading { display: flex; align-items: center; justify-content: space-between; padding: 14px 15px 12px; border-bottom: 1px solid #eef0f2; font-size: 13px; font-weight: 700; color: #171717; }
.generate-params-heading div { display: flex; flex-direction: column; gap: 2px; }
.generate-params-heading small { font-size: 10px; font-weight: 500; color: #a1a1aa; }
.generate-params-model { max-width: 52%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 10px; font-weight: 600; color: #71717a; }
.generate-model-picker { position: relative; margin: 13px 15px 0; }
.generate-model-trigger { display: flex; align-items: center; justify-content: space-between; width: 100%; min-height: 42px; padding: 0 12px; border: 1px solid #e4e4e7; border-radius: 9px; background: #fafafa; color: #27272a; font-size: 12px; text-align: left; transition: border-color .16s ease, box-shadow .16s ease; }
.generate-model-trigger:hover, .generate-model-trigger[aria-expanded="true"] { border-color: #a1a1aa; box-shadow: 0 0 0 3px rgba(24,24,27,.05); }
.generate-model-trigger-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.generate-model-chevron { width: 16px; height: 16px; flex: 0 0 auto; color: #71717a; transition: transform .16s ease; }
.generate-model-chevron.open { transform: rotate(180deg); }
.generate-model-menu { position: absolute; z-index: 20; top: calc(100% + 6px); left: 0; right: 0; max-height: 220px; overflow-y: auto; padding: 5px; border: 1px solid #e4e4e7; border-radius: 10px; background: #fff; box-shadow: 0 14px 30px rgba(15,23,42,.12); }
.generate-model-option { display: block; width: 100%; padding: 9px 10px; border-radius: 7px; color: #52525b; font-size: 12px; text-align: left; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.generate-model-option:hover { background: #f4f4f5; color: #18181b; }
.generate-model-option.active { background: #27272a; color: #fff; }
.generate-params-row { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin: 16px 15px 0; padding-bottom: 16px; border-bottom: 1px solid #eef0f2; }
.generate-param-group { min-width: 0; }
.generate-size-section { padding: 15px; }
.generate-size-heading { display: flex; align-items: center; justify-content: space-between; margin-bottom: 9px; }
.generate-size-heading .generate-param-label { margin: 0; }
.generate-size-unit { font-size: 10px; letter-spacing: .08em; color: #a3a3a3; }
.generate-size-tabs { display: grid; grid-template-columns: 1fr 1fr; padding: 3px; gap: 3px; border-radius: 9px; background: #f4f4f5; }
.generate-size-tabs button { min-height: 36px; border-radius: 7px; font-size: 12px; color: #737373; }
.generate-size-tabs button.active { background: #fff; color: #171717; font-weight: 650; box-shadow: 0 1px 3px #00000012; }
.generate-custom-size { margin-top: 16px; }
.generate-size-inputs { display: grid; grid-template-columns: minmax(0, 1fr) 16px minmax(0, 1fr); align-items: end; gap: 6px; }
.generate-size-separator { align-self: end; line-height: 42px; text-align: center; color: #a3a3a3; }
.generate-size-inputs label { min-width: 0; font-size: 11px; color: #737373; }
.generate-size-field { display: flex; align-items: center; gap: 4px; margin-top: 7px; padding: 0 9px; height: 42px; border: 1px solid #d4d4d4; border-radius: 8px; background: #fff; }
.generate-size-field:focus-within { border-color: #171717; box-shadow: 0 0 0 3px #1717170d; }
.generate-size-field:has(input[aria-invalid="true"]) { border-color: #dc2626; }
.generate-size-field input { width: 100%; min-width: 0; outline: none; font-size: 14px; font-weight: 600; font-variant-numeric: tabular-nums; color: #262626; appearance: textfield; -moz-appearance: textfield; }
.generate-size-field input::-webkit-inner-spin-button, .generate-size-field input::-webkit-outer-spin-button { -webkit-appearance: none; margin: 0; }
.generate-size-field > span { font-size: 10px; color: #a3a3a3; }
.generate-size-tabs button:focus-visible { outline: 2px solid #171717; outline-offset: 2px; }
.generate-size-error { font-size: 11px; line-height: 1.7; color: #737373; }
.generate-auto-resolution { display: flex; align-items: center; justify-content: space-between; gap: 6px; min-height: 34px; padding: 0 9px; border: 1px solid #e4e4e7; border-radius: 8px; background: #fafafa; color: #737373; font-size: 11px; }
.generate-auto-resolution strong { color: #262626; font-size: 12px; }
.generate-size-error { margin-top: 10px; color: #dc2626; }
.generate-params-panel > .generate-param-group { padding: 14px 15px 16px; }
.generate-param-label { display: block; margin-bottom: 8px; font-size: 11px; font-weight: 650; color: #71717a; }
.generate-chip-row { display: flex; flex-wrap: wrap; gap: 8px; }
.generate-chip { min-width: 54px; height: 34px; padding: 0 12px; border: 1px solid #e4e4e7; border-radius: 8px; background: #fafafa; color: #52525b; font-size: 12px; font-weight: 650; transition: all .16s ease; }
.generate-count-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.generate-chip--count { width: 100%; min-width: 0; padding: 0 8px; }
.generate-chip:hover { border-color: #a3a3a3; color: #171717; }
.generate-chip.active { border-color: #27272a; background: #27272a; color: #fff; }
.ratio-visual-grid--compact { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 7px; margin-top: 3px; }
.ratio-visual-grid--compact .ratio-card { min-height: 88px; gap: 4px; padding: 8px 4px 7px; border-radius: 9px; background: #fafafa; }
.ratio-visual-grid--compact .ratio-card-frame { min-height: 40px; }
.ratio-visual-grid--compact .ratio-card-shape { border-width: 2px; border-radius: 3px; }
.ratio-visual-grid--compact .ratio-card-label { font-size: 11px; font-weight: 700; }
.ratio-visual-grid--compact .ratio-card.on { border-color: #27272a; background: #fff; color: #18181b; box-shadow: inset 0 0 0 1px #27272a; }
.ratio-visual-grid--compact .ratio-card.on small { color: #71717a; }

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

.ratio-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 10px 22px rgba(15, 23, 42, .08);
}

.ratio-card.on {
  background: #171717;
  border-color: #171717;
  color: #fff;
  box-shadow: none;
}

.ratio-card-frame {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 54px;
}

.ratio-card-shape {
  display: block;
  border: 3px solid currentColor;
  border-radius: 7px;
  opacity: .76;
}

.ratio-card.on .ratio-card-shape {
  opacity: 1;
}

.ratio-card-label {
  font-size: 14px;
  font-weight: 500;
  letter-spacing: .01em;
}

.ratio-shape-square {
  width: 34px;
  height: 34px;
}

.ratio-shape-landscape {
  width: 46px;
  height: 34px;
}

.ratio-shape-wide {
  width: 50px;
  height: 28px;
}

.ratio-shape-cinema {
  width: 54px;
  height: 24px;
}

.ratio-shape-portrait {
  width: 28px;
  height: 42px;
}

.ratio-shape-tall {
  width: 24px;
  height: 46px;
}

/* Lazy loading image styles */
.lazy-img {
  opacity: 0;
  transition: opacity 0.3s ease;
  transform: translateZ(0);
  backface-visibility: hidden;
}

.lazy-img-loaded {
  opacity: 1;
}

.batch-selected { box-shadow: 0 0 0 2px #111827, 0 10px 24px rgba(15,23,42,.12); }
.batch-pickable { cursor: pointer; }
.batch-select-dot { position: absolute; z-index: 12; top: 8px; left: 8px; width: 24px; height: 24px; border: 2px solid #fff; border-radius: 50%; background: #fff; color: #fff; box-shadow: 0 2px 8px rgba(15,23,42,.2); cursor: pointer; }
.batch-selected .batch-select-dot { background: #111827; }
.batch-inline-bar { position: sticky; bottom: 10px; z-index: 20; display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 12px 14px 4px; padding: 10px 12px; border: 1px solid #e5e7eb; border-radius: 14px; background: rgba(255,255,255,.96); box-shadow: 0 10px 28px rgba(15,23,42,.12); font-size: 12px; font-weight: 700; color: #475569; }
.batch-inline-bar > div { display: flex; gap: 8px; align-items: center; }
.batch-inline-confirm { min-height: 34px; border: 0; border-radius: 999px; padding: 0 14px; background: #111827; color: #fff; font-size: 12px; font-weight: 700; }
.batch-inline-confirm:disabled { cursor: not-allowed; opacity: .45; }
</style>
