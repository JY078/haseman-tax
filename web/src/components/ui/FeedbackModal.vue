<template>
  <TransitionRoot as="template" :show="open">
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
            <DialogPanel class="relative transform overflow-hidden rounded-2xl bg-white text-left shadow-xl transition-all sm:my-8 sm:max-w-lg w-full">
              <div class="bg-white px-6 py-6">
                <div class="flex items-center gap-3 mb-5">
                  <div class="w-10 h-10 rounded-xl bg-neutral-100 flex items-center justify-center flex-shrink-0">
                    <ChatBubbleLeftRightIcon class="w-5 h-5 text-neutral-600" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <DialogTitle as="h3" class="text-lg font-semibold text-neutral-900">意见反馈</DialogTitle>
                    <p class="text-xs text-neutral-400 mt-0.5">您的建议是我们进步的动力</p>
                  </div>
                  <button
                    type="button"
                    class="rounded-lg p-1.5 text-neutral-400 hover:text-neutral-500 hover:bg-neutral-100"
                    @click="handleClose"
                  >
                    <XMarkIcon class="h-5 w-5" />
                  </button>
                </div>

                <div v-if="submitted" class="py-8 text-center">
                  <div class="w-14 h-14 bg-green-50 rounded-full flex items-center justify-center mx-auto mb-4">
                    <CheckIcon class="w-7 h-7 text-green-500" />
                  </div>
                  <p class="text-base font-medium text-neutral-900">感谢您的反馈！</p>
                  <p class="text-sm text-neutral-500 mt-1">我们会认真阅读并持续改进产品</p>
                  <button
                    class="mt-5 px-5 py-2 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors"
                    @click="handleClose"
                  >
                    关闭
                  </button>
                </div>

                <div v-else>
                  <div class="space-y-4">
                    <div>
                      <label class="block text-sm font-medium text-neutral-700 mb-1.5">
                        反馈内容 <span class="text-red-500">*</span>
                      </label>
                      <textarea
                        v-model="content"
                        rows="5"
                        placeholder="请描述您遇到的问题或想法建议，我们会认真阅读每一条反馈..."
                        class="w-full px-3.5 py-2.5 text-sm text-neutral-900 placeholder-neutral-400 border border-neutral-200 rounded-xl resize-none focus:outline-none focus:ring-2 focus:ring-neutral-900/20 focus:border-neutral-400 transition-colors"
                        :class="{ 'border-red-300 focus:ring-red-200 focus:border-red-400': contentError }"
                      />
                      <p v-if="contentError" class="mt-1 text-xs text-red-500">{{ contentError }}</p>
                    </div>

                    <!-- <div>
                      <label class="block text-sm font-medium text-neutral-700 mb-1.5">联系方式（选填）</label>
                      <input
                        v-model="contactInfo"
                        type="text"
                        placeholder="邮箱 / 微信 / 手机，方便我们回复您"
                        class="w-full px-3.5 py-2.5 text-sm text-neutral-900 placeholder-neutral-400 border border-neutral-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-neutral-900/20 focus:border-neutral-400 transition-colors"
                      />
                    </div> -->
                  </div>

                  <div class="mt-6 flex justify-end gap-3">
                    <button
                      class="px-4 py-2 text-sm font-medium rounded-lg text-neutral-600 hover:bg-neutral-100 transition-colors"
                      @click="handleClose"
                    >
                      取消
                    </button>
                    <button
                      class="px-5 py-2 text-sm font-medium rounded-lg bg-neutral-900 text-white hover:bg-neutral-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                      :disabled="submitting"
                      @click="handleSubmit"
                    >
                      <span v-if="submitting" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                      {{ submitting ? '提交中...' : '提交反馈' }}
                    </button>
                  </div>
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
import { ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { XMarkIcon, ChatBubbleLeftRightIcon, CheckIcon } from '@heroicons/vue/24/outline'
import request from '@/utils/request'

const props = defineProps({
  open: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:open'])

const content = ref('')
const contactInfo = ref('')
const contentError = ref('')
const submitting = ref(false)
const submitted = ref(false)

watch(() => props.open, (val) => {
  if (val) {
    content.value = ''
    contactInfo.value = ''
    contentError.value = ''
    submitting.value = false
    submitted.value = false
  }
})

function handleClose() {
  emit('update:open', false)
}

async function handleSubmit() {
  contentError.value = ''
  if (!content.value.trim()) {
    contentError.value = '请填写反馈内容'
    return
  }
  submitting.value = true
  try {
    await request.post('/feedbacks', {
      content: content.value.trim(),
      contact_info: contactInfo.value.trim(),
    })
    submitted.value = true
  } catch {
    // error intercepted globally
  } finally {
    submitting.value = false
  }
}
</script>
