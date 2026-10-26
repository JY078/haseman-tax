import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 图片懒加载 composable
 * 使用 Intersection Observer 监听图片进入视口时再加载
 */
export function useLazyLoad(options = {}) {
  const { rootMargin = '100px', threshold = 0.1 } = options
  let observer = null

  const observe = (el, callback) => {
    if (!el) return

    if (!observer) {
      observer = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              callback(entry.target)
              observer.unobserve(entry.target)
            }
          })
        },
        { rootMargin, threshold }
      )
    }

    observer.observe(el)
  }

  const unobserve = (el) => {
    if (observer && el) {
      observer.unobserve(el)
    }
  }

  const disconnect = () => {
    if (observer) {
      observer.disconnect()
      observer = null
    }
  }

  onUnmounted(() => {
    disconnect()
  })

  return { observe, unobserve, disconnect }
}

/**
 * 创建懒加载图片 src
 * 图片初始不加载，当进入视口时设置 src
 */
export function useLazyImage() {
  const { observe, disconnect } = useLazyLoad()

  const imageStates = ref(new Map())

  const registerImage = (el, src) => {
    if (!el || !src) return

    imageStates.value.set(el, {
      src,
      loaded: false,
    })

    // 先用一个极小的透明像素占位，避免闪烁
    el.src = 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7'

    observe(el, (target) => {
      const state = imageStates.value.get(target)
      if (state && !state.loaded) {
        target.src = state.src
        state.loaded = true
      }
    })
  }

  const unregisterImage = (el) => {
    if (imageStates.value.has(el)) {
      imageStates.value.delete(el)
    }
  }

  onUnmounted(() => {
    disconnect()
    imageStates.value.clear()
  })

  return { registerImage, unregisterImage }
}