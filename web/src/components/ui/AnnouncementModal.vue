<template>
  <TransitionRoot as="template" :show="visible">
    <Dialog as="div" class="relative z-50" @close="handleClose">
      <TransitionChild
        as="template"
        enter="ease-out duration-300"
        enter-from="opacity-0"
        enter-to="opacity-100"
        leave="ease-in duration-200"
        leave-from="opacity-100"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-neutral-900/50 backdrop-blur-sm transition-opacity" />
      </TransitionChild>

      <div class="fixed inset-0 z-10 overflow-y-auto">
        <div class="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <TransitionChild
            as="template"
            enter="ease-out duration-300"
            enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            enter-to="opacity-100 translate-y-0 sm:scale-100"
            leave="ease-in duration-200"
            leave-from="opacity-100 translate-y-0 sm:scale-100"
            leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          >
            <DialogPanel
              class="relative transform overflow-hidden rounded-2xl bg-white text-left shadow-xl transition-all sm:my-8 sm:max-w-lg w-full"
            >
              <div class="bg-white px-6 py-6">
                <!-- 顶部标识 -->
                <div class="flex items-center gap-3 mb-4">
                  <div class="w-10 h-10 rounded-xl bg-blue-50 flex items-center justify-center flex-shrink-0">
                    <MegaphoneIcon class="w-5 h-5 text-blue-600" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <DialogTitle as="h3" class="text-lg font-semibold text-neutral-900 truncate">
                      {{ currentAnnouncement?.title }}
                    </DialogTitle>
                    <p class="text-xs text-neutral-400 mt-0.5">
                      {{ formatTime(currentAnnouncement?.created_at) }}
                      <span v-if="list.length > 1" class="ml-2">
                        ({{ currentIndex + 1 }} / {{ list.length }})
                      </span>
                    </p>
                  </div>
                  <button
                    type="button"
                    class="rounded-lg p-1.5 text-neutral-400 hover:text-neutral-500 hover:bg-neutral-100"
                    @click="handleClose"
                  >
                    <XMarkIcon class="h-5 w-5" />
                  </button>
                </div>

                <!-- 公告内容 -->
                <div class="text-sm text-neutral-600 leading-relaxed whitespace-pre-line max-h-[50vh] overflow-y-auto">
                  {{ currentAnnouncement?.content }}
                </div>
              </div>

              <!-- 底部操作 -->
              <div class="bg-neutral-50 px-6 py-4 flex items-center justify-between">
                <span v-if="list.length > 1" class="text-xs text-neutral-400">
                  还有 {{ list.length - currentIndex - 1 }} 条未读公告
                </span>
                <span v-else />
                <div class="flex gap-2">
                  <button
                    v-if="hasNext"
                    class="px-4 py-2 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors"
                    @click="markReadAndNext"
                  >
                    下一条
                  </button>
                  <button
                    v-else
                    class="px-4 py-2 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors"
                    @click="markReadAndClose"
                  >
                    知道了
                  </button>
                </div>
              </div>
            </DialogPanel>
          </TransitionChild>
        </div>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { XMarkIcon, MegaphoneIcon } from '@heroicons/vue/24/outline'
import request from '@/utils/request'

const visible = ref(false)
const list = ref([])
const currentIndex = ref(0)

const currentAnnouncement = computed(() => list.value[currentIndex.value] || null)
const hasNext = computed(() => currentIndex.value < list.value.length - 1)

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString()
}

async function markCurrentAsRead() {
  const item = currentAnnouncement.value
  if (!item) return
  try {
    await request.post(`/announcements/${item.id}/read`)
  } catch {
    // 静默
  }
}

async function markReadAndNext() {
  await markCurrentAsRead()
  currentIndex.value++
}

async function markReadAndClose() {
  await markCurrentAsRead()
  visible.value = false
}

function handleClose() {
  markCurrentAsRead()
  visible.value = false
}

async function checkUnread() {
  try {
    const res = await request.get('/me/unread-announcements')
    const data = res.data || []
    if (data.length > 0) {
      list.value = data
      currentIndex.value = 0
      visible.value = true
    }
  } catch {
    // 静默
  }
}

defineExpose({ checkUnread })
</script>
