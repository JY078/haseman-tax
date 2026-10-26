<template>
  <MainLayout>
    <Transition name="tool-fade" mode="out-in">
      <EraserTool v-if="activeTool === 'eraser'" key="eraser" />
      <RemoveBgTool v-else-if="activeTool === 'remove-bg'" key="remove-bg" />
      <EraserTool v-else key="default" />
    </Transition>
  </MainLayout>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import MainLayout from '@/components/layout/MainLayout.vue'
import EraserTool from '@/components/tools/EraserTool.vue'
import RemoveBgTool from '@/components/tools/RemoveBgTool.vue'

const route = useRoute()

const activeTool = computed(() => route.query.t || 'eraser')
</script>

<style scoped>
.tool-fade-enter-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.tool-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.tool-fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.tool-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
