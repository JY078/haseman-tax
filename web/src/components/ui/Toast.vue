<template>
  <Teleport to="body">
    <div class="fixed top-5 left-1/2 -translate-x-1/2 z-[9999] flex flex-col items-center gap-2 pointer-events-none">
      <TransitionGroup name="toast">
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="pointer-events-auto flex items-start gap-3 min-w-[280px] max-w-sm w-max px-4 py-3 rounded-xl shadow-lg text-sm font-medium cursor-pointer select-none"
          :class="typeClasses[toast.type]"
          @click="remove(toast.id)"
        >
          <component :is="typeIcons[toast.type]" class="w-4 h-4 mt-0.5 flex-shrink-0" />
          <span class="leading-relaxed">{{ toast.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
} from '@heroicons/vue/24/solid'
import { useToast } from '@/composables/useToast'

const { toasts, remove } = useToast()

const typeClasses = {
  success: 'bg-green-50  text-green-800  border border-green-200',
  error:   'bg-red-50    text-red-800    border border-red-200',
  warning: 'bg-orange-50 text-orange-800 border border-orange-200',
  info:    'bg-blue-50   text-blue-800   border border-blue-200',
}

const typeIcons = {
  success: CheckCircleIcon,
  error:   ExclamationCircleIcon,
  warning: ExclamationTriangleIcon,
  info:    InformationCircleIcon,
}
</script>

<style scoped>
.toast-enter-active {
  transition: opacity 0.25s ease, transform 0.25s cubic-bezier(0.22, 1, 0.36, 1);
}
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateY(-12px) scale(0.96);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.96);
}
</style>
