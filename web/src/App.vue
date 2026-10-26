<template>
  <RouterView v-slot="{ Component, route }">
    <Transition :name="transitionName" mode="out-in">
      <component :is="Component" :key="route.path" />
    </Transition>
  </RouterView>
  <Toast />
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Toast from '@/components/ui/Toast.vue'

/**
 * 路由在主导航中的位置顺序（与 Header 顶栏从左到右一致）。
 * 数字越大表示越靠右；同组工具页共用同一序号，切换时为纵向浮入。
 * 未列入的路由（如个人中心）：to/from 任一侧无序号则走 nav-up。
 */
const ROUTE_ORDER = {
  '/': 0,
  '/stp-white-bg': 1,
  '/scene-generator': 1,
  '/generate': 1,
  '/video-generate': 2,
  '/multi-angle': 1,
  '/depth': 1,
  '/light-shadow': 1,
  '/image-translate': 1,
  '/image-to-prompt': 1,
  '/white-bg': 1,
  '/assets': 1,
  '/product-library': 2,
  '/product-prices': 3,
  '/order-to-mcf': 3,
  '/work-requests': 3,
  '/purchase-plans': 3,
}

const transitionName = ref('nav-up')
const router = useRouter()

router.beforeEach((to, from) => {
  const toOrder   = ROUTE_ORDER[to.path]
  const fromOrder = ROUTE_ORDER[from.path]

  /** 不在主导航的页面使用纵向浮入。 */
  if (toOrder === undefined || fromOrder === undefined) {
    transitionName.value = 'nav-up'
    return
  }

  if (toOrder > fromOrder) {
    transitionName.value = 'nav-forward'   // 向右走 → 内容从右侧滑入
  } else if (toOrder < fromOrder) {
    transitionName.value = 'nav-backward'  // 向左走 → 内容从左侧滑入
  } else {
    transitionName.value = 'nav-up'
  }
})
</script>

<style>
/* ── 共用参数 ─────────────────────────────────────────────
   进场：cubic-bezier(0.22, 1, 0.36, 1) = 弹性落地感
   离场：cubic-bezier(0.4, 0, 1, 1)    = 快速让位
──────────────────────────────────────────────────────── */

/* 向前（导航向右）：旧页面向左消失，新页面从右侧滑入 */
.nav-forward-leave-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 1, 1),
              transform 0.2s cubic-bezier(0.4, 0, 1, 1);
}
.nav-forward-enter-active {
  transition: opacity 0.32s cubic-bezier(0.22, 1, 0.36, 1),
              transform 0.32s cubic-bezier(0.22, 1, 0.36, 1);
}
.nav-forward-leave-to  { opacity: 0; transform: translateX(-28px); }
.nav-forward-enter-from { opacity: 0; transform: translateX(36px); }

/* 向后（导航向左）：旧页面向右消失，新页面从左侧滑入 */
.nav-backward-leave-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 1, 1),
              transform 0.2s cubic-bezier(0.4, 0, 1, 1);
}
.nav-backward-enter-active {
  transition: opacity 0.32s cubic-bezier(0.22, 1, 0.36, 1),
              transform 0.32s cubic-bezier(0.22, 1, 0.36, 1);
}
.nav-backward-leave-to  { opacity: 0; transform: translateX(28px); }
.nav-backward-enter-from { opacity: 0; transform: translateX(-36px); }

/* 纵向浮入：个人中心等侧边页面 */
.nav-up-leave-active {
  transition: opacity 0.18s cubic-bezier(0.4, 0, 1, 1),
              transform 0.18s cubic-bezier(0.4, 0, 1, 1);
}
.nav-up-enter-active {
  transition: opacity 0.3s cubic-bezier(0.22, 1, 0.36, 1),
              transform 0.3s cubic-bezier(0.22, 1, 0.36, 1);
}
.nav-up-leave-to   { opacity: 0; transform: translateY(-10px) scale(0.99); }
.nav-up-enter-from { opacity: 0; transform: translateY(14px); }
</style>
