<template>
  <div class="lazy-image" :class="{ loaded: isLoaded, loading: isLoading }">
    <img
      v-if="shouldLoad"
      ref="imgRef"
      :src="resolvedSrc"
      :alt="alt"
      :class="imgClass"
      @load="onLoad"
      @error="onError"
    />
    <div v-else :class="placeholderClass">
      <slot name="placeholder">
        <div class="lazy-placeholder animate-pulse bg-neutral-200 rounded" :style="placeholderStyle"></div>
      </slot>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useLazyLoad } from '@/composables/useLazyLoad'

const props = defineProps({
  src: {
    type: String,
    required: true
  },
  alt: {
    type: String,
    default: ''
  },
  imgClass: {
    type: [String, Array, Object],
    default: ''
  },
  placeholderClass: {
    type: String,
    default: 'w-full h-full flex items-center justify-center'
  },
  placeholderStyle: {
    type: String,
    default: ''
  },
  rootMargin: {
    type: String,
    default: '100px'
  },
  threshold: {
    type: Number,
    default: 0.1
  }
})

const emit = defineEmits(['load', 'error', 'visible'])

const imgRef = ref(null)
const isLoaded = ref(false)
const isLoading = ref(false)
const hasError = ref(false)
const shouldLoad = ref(false)

const { observe, unobserve } = useLazyLoad({ rootMargin: props.rootMargin, threshold: props.threshold })

const resolvedSrc = computed(() => props.src)

const onLoad = () => {
  isLoaded.value = true
  isLoading.value = false
  emit('load')
}

const onError = () => {
  hasError.value = true
  isLoading.value = false
  emit('error')
}

let unwatch = null

onMounted(() => {
  if (imgRef.value) {
    observe(imgRef.value, (el) => {
      shouldLoad.value = true
      isLoading.value = true
      emit('visible')
    })
  }
})

onBeforeUnmount(() => {
  if (unwatch) unwatch()
  if (imgRef.value) {
    unobserve(imgRef.value)
  }
})
</script>

<style scoped>
.lazy-image {
  position: relative;
  overflow: hidden;
}

.lazy-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: opacity 0.3s ease;
}

.lazy-image:not(.loaded) img {
  opacity: 0;
}

.lazy-image.loaded img {
  opacity: 1;
}

.lazy-placeholder {
  width: 100%;
  height: 100%;
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>