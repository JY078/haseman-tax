<template>
  <div class="min-h-screen bg-neutral-50 flex items-center justify-center px-4">
    <div class="max-w-lg w-full text-center">
      <div class="mb-8">
        <div class="w-20 h-20 mx-auto bg-neutral-100 rounded-2xl flex items-center justify-center mb-6">
          <WrenchScrewdriverIcon class="w-10 h-10 text-neutral-400" />
        </div>
        <h1 class="text-3xl sm:text-4xl font-bold text-neutral-900 mb-4">
          系统维护中
        </h1>
        <p class="text-lg text-neutral-500 leading-relaxed whitespace-pre-line">
          {{ message }}
        </p>
      </div>
      <div class="bg-white rounded-xl border border-neutral-200 p-6 shadow-sm">
        <div class="flex items-center justify-center gap-2 text-neutral-400 text-sm">
          <ArrowPathIcon class="w-4 h-4 animate-spin" />
          <span>工程师正在紧张处理中，请稍后刷新重试</span>
        </div>
      </div>
      <button
        class="mt-6 inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-neutral-900 text-white text-sm font-medium hover:bg-neutral-800 transition-colors"
        @click="refresh"
      >
        <ArrowPathIcon class="w-4 h-4" />
        刷新页面
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { WrenchScrewdriverIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

const message = ref('系统维护中，请稍后再试')

onMounted(async () => {
  try {
    const baseURL = import.meta.env.VITE_API_BASE_URL || '/api'
    const res = await fetch(`${baseURL}/maintenance-status`, { cache: 'no-store' })
    const data = await res.json()
    if (data.message) {
      message.value = data.message
    }
  } catch {
    // 静默失败，使用默认消息
  }
})

const refresh = () => {
  window.location.reload()
}
</script>
