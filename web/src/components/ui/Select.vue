<template>
  <div class="w-full" ref="wrapperRef">
    <label v-if="label" class="block text-sm font-medium text-neutral-700 mb-1.5">
      {{ label }}
    </label>

    <!-- Loading state -->
    <div v-if="loading" class="w-full flex items-center gap-2 px-4 py-2.5 border border-neutral-200 rounded-xl bg-neutral-50 text-sm text-neutral-400">
      <svg class="w-4 h-4 animate-spin flex-shrink-0" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
      </svg>
      加载中...
    </div>

    <!-- Custom dropdown -->
    <div v-else class="relative">
      <!-- Trigger button -->
      <button
        type="button"
        :disabled="disabled"
        class="w-full flex items-center justify-between gap-2 px-4 py-2.5 text-sm rounded-xl border transition-all duration-150 focus:outline-none focus:ring-2 focus:ring-neutral-900/20 text-left"
        :class="[
          isOpen
            ? 'border-neutral-900 bg-white ring-2 ring-neutral-900/10'
            : error
              ? 'border-red-300 bg-white'
              : 'border-neutral-300 bg-white hover:border-neutral-400',
          disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
        ]"
        @click="toggleOpen"
      >
        <span :class="selectedLabel ? 'text-neutral-800 font-medium' : 'text-neutral-400'">
          {{ selectedLabel || placeholder || '请选择...' }}
        </span>
        <svg
          class="w-4 h-4 text-neutral-400 flex-shrink-0 transition-transform duration-200"
          :class="isOpen ? 'rotate-180' : ''"
          fill="none" stroke="currentColor" viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      <!-- Dropdown panel -->
      <Transition
        enter-active-class="transition ease-out duration-150"
        enter-from-class="opacity-0 translate-y-1"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-100"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-1"
      >
        <div
          v-show="isOpen"
          class="absolute z-50 w-full mt-1.5 bg-white border border-neutral-200 rounded-xl shadow-lg overflow-hidden"
        >
          <div class="max-h-60 overflow-y-auto py-1">
            <div
              v-for="opt in options"
              :key="opt.value"
              class="flex items-center justify-between px-4 py-2.5 text-sm cursor-pointer select-none transition-colors duration-100"
              :class="isSelected(opt.value)
                ? 'bg-neutral-900 text-white'
                : 'text-neutral-700 hover:bg-neutral-50'"
              @click="selectOption(opt)"
            >
              <span class="truncate">{{ opt.label }}</span>
              <svg
                v-if="isSelected(opt.value)"
                class="w-4 h-4 flex-shrink-0 ml-2"
                fill="none" stroke="currentColor" viewBox="0 0 24 24"
              >
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <div v-if="options.length === 0" class="px-4 py-3 text-sm text-neutral-400 text-center">
              暂无选项
            </div>
          </div>
        </div>
      </Transition>
    </div>

    <p v-if="error" class="mt-1.5 text-xs text-red-500">{{ error }}</p>
    <p v-if="hint && !error" class="mt-1.5 text-xs text-neutral-500">{{ hint }}</p>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  options: {
    type: Array,
    default: () => [],
  },
  label: {
    type: String,
    default: '',
  },
  placeholder: {
    type: String,
    default: '',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: '',
  },
  hint: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['update:modelValue'])

const isOpen = ref(false)
const wrapperRef = ref(null)

const selectedLabel = computed(() => {
  const found = props.options.find(o => String(o.value) === String(props.modelValue))
  return found ? found.label : ''
})

function isSelected(value) {
  return String(props.modelValue) === String(value)
}

function toggleOpen() {
  if (!props.disabled) isOpen.value = !isOpen.value
}

function selectOption(opt) {
  emit('update:modelValue', opt.value)
  isOpen.value = false
}

function handleClickOutside(e) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target)) {
    isOpen.value = false
  }
}

onMounted(() => document.addEventListener('mousedown', handleClickOutside))
onUnmounted(() => document.removeEventListener('mousedown', handleClickOutside))
</script>
