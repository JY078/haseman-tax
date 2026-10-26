import { reactive } from 'vue'

/** 全局 toast 列表，跨组件共享同一份响应式数据 */
const toasts = reactive([])
let nextId = 0

function addToast(message, type = 'info', duration = 4000) {
  const id = ++nextId
  toasts.push({ id, message, type })
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration)
  }
  return id
}

function removeToast(id) {
  const index = toasts.findIndex(t => t.id === id)
  if (index !== -1) toasts.splice(index, 1)
}

export function useToast() {
  return {
    toasts,
    error:   (msg, duration) => addToast(msg, 'error',   duration),
    success: (msg, duration) => addToast(msg, 'success', duration),
    warning: (msg, duration) => addToast(msg, 'warning', duration),
    info:    (msg, duration) => addToast(msg, 'info',    duration),
    remove:  removeToast,
  }
}
