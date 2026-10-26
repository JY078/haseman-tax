/**
 * Fake Progress Composable
 *
 * 假进度条管理，支持页面刷新后进度不重置
 * 支持多个任务同时进行进度动画
 */
import { ref, onUnmounted } from 'vue'

interface PersistedProgress {
  progress: number
  startedAt: number
}

interface TaskProgress {
  taskId: string | number
  intervalId: ReturnType<typeof setInterval> | null
  callback: ((progress: number) => void) | null
  currentProgress: number
}

interface FakeProgressOptions {
  /** 进度存储的 key 前缀 */
  storageKeyPrefix?: string
  /** 初始进度（默认从存储恢复，或从 0 开始） */
  initialProgress?: number
}

export function useFakeProgress(options: FakeProgressOptions = {}) {
  const { storageKeyPrefix = 'fake_progress', initialProgress = 0 } = options

  // 所有任务的进度跟踪（Map 以支持多任务同时进行）
  const taskProgressMap = ref<Record<string, TaskProgress>>({})

  // 是否已完成（达到100%）
  const isComplete = ref(false)

  /**
   * 生成存储 key
   */
  function getStorageKey(taskId: string | number): string {
    return `${storageKeyPrefix}_${taskId}`
  }

  /**
   * 获取存储的进度数据
   */
  function getPersistedProgress(taskId: string | number): PersistedProgress | null {
    try {
      const stored = sessionStorage.getItem(getStorageKey(taskId))
      if (stored) {
        return JSON.parse(stored)
      }
    } catch (e) {
      console.warn('[useFakeProgress] Failed to get persisted progress:', e)
    }
    return null
  }

  /**
   * 保存进度到 sessionStorage
   */
  function saveProgress(taskId: string | number, progressValue: number): void {
    try {
      const data: PersistedProgress = {
        progress: progressValue,
        startedAt: Date.now()
      }
      sessionStorage.setItem(getStorageKey(taskId), JSON.stringify(data))
    } catch (e) {
      console.warn('[useFakeProgress] Failed to save progress:', e)
    }
  }

  /**
   * 清除存储的进度数据
   */
  function clearPersistedProgress(taskId: string | number): void {
    try {
      sessionStorage.removeItem(getStorageKey(taskId))
    } catch (e) {
      console.warn('[useFakeProgress] Failed to clear persisted progress:', e)
    }
  }

  /**
   * 启动假进度条（单个任务）
   * @param taskId - 任务 ID，用于持久化
   * @param onProgressUpdate - 进度更新回调
   */
  function startProgress(taskId: string | number, onProgressUpdate: (progress: number) => void): void {
    const taskKey = String(taskId)

    // 如果已有该任务的进度，先停止它
    const existing = taskProgressMap.value[taskKey]
    if (existing?.intervalId) {
      clearInterval(existing.intervalId)
    }

    // 尝试恢复已有进度
    let currentProgress = initialProgress
    const persisted = getPersistedProgress(taskId)
    if (persisted) {
      currentProgress = persisted.progress
    }

    // 进度更新间隔（毫秒）
    const intervalMs = 200

    // 创建任务进度对象
    const taskProgress: TaskProgress = {
      taskId,
      intervalId: null,
      callback: onProgressUpdate,
      currentProgress
    }

    // 每隔一段时间增加进度
    taskProgress.intervalId = setInterval(() => {
      const task = taskProgressMap.value[taskKey]
      if (!task || !task.callback) return

      if (task.currentProgress >= 100) {
        clearInterval(task.intervalId!)
        task.intervalId = null
        return
      }

      // 进度增长算法：
      // - 0-80%: 每次增加 3-5%
      // - 80-95%: 每次增加 1-2%
      // - 95-100%: 等待后端完成信号，这里只模拟到 98%
      let increment: number
      if (task.currentProgress < 80) {
        increment = 3 + Math.random() * 2  // 3-5%
      } else if (task.currentProgress < 95) {
        increment = 1 + Math.random() * 1  // 1-2%
      } else {
        increment = 0.5 + Math.random() * 0.5  // 0.5-1%
      }

      task.currentProgress = Math.min(100, task.currentProgress + increment)

      // 回调通知
      task.callback(task.currentProgress)

      // 每 10% 保存一次到 sessionStorage
      if (Math.floor(task.currentProgress) % 10 === 0) {
        saveProgress(taskId, task.currentProgress)
      }
    }, intervalMs)

    // 保存到 Map
    taskProgressMap.value[taskKey] = taskProgress
  }

  /**
   * 停止单个任务的进度条
   */
  function stopProgress(taskId?: string | number): void {
    if (taskId !== undefined) {
      const taskKey = String(taskId)
      const task = taskProgressMap.value[taskKey]
      if (task?.intervalId) {
        clearInterval(task.intervalId)
        task.intervalId = null
      }
      delete taskProgressMap.value[taskKey]
    } else {
      // 停止所有任务
      Object.values(taskProgressMap.value).forEach(task => {
        if (task.intervalId) {
          clearInterval(task.intervalId)
        }
      })
      taskProgressMap.value = {}
    }
  }

  /**
   * 完成单个任务的进度（设为 100%）
   */
  function completeProgress(taskId: string | number): void {
    const taskKey = String(taskId)
    const task = taskProgressMap.value[taskKey]
    if (task) {
      if (task.intervalId) {
        clearInterval(task.intervalId)
        task.intervalId = null
      }
      task.currentProgress = 100
      if (task.callback) {
        task.callback(100)
      }
      delete taskProgressMap.value[taskKey]
    }
    clearPersistedProgress(taskId)
  }

  /**
   * 重置单个任务的进度
   */
  function resetProgress(taskId: string | number): void {
    stopProgress(taskId)
    clearPersistedProgress(taskId)
  }

  /**
   * 手动设置单个任务的进度值
   */
  function setProgress(value: number, taskId: string | number): void {
    const taskKey = String(taskId)
    const task = taskProgressMap.value[taskKey]
    if (task) {
      task.currentProgress = value
      if (task.callback) {
        task.callback(value)
      }
    }
    if (value >= 100) {
      clearPersistedProgress(taskId)
    } else {
      saveProgress(taskId, value)
    }
  }

  // 组件卸载时清理
  onUnmounted(() => {
    stopProgress()
  })

  return {
    isComplete,
    startProgress,
    stopProgress,
    completeProgress,
    resetProgress,
    setProgress,
    getPersistedProgress,
    clearPersistedProgress,
    saveProgress,
    getStorageKey
  }
}