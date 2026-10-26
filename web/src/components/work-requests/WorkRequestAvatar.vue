<template><span class="request-avatar" :style="{ background: color }">{{ initial }}</span></template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ user: { type: Object, default: null } })
const initial = computed(() => (props.user?.name || props.user?.username || '?').trim().charAt(0).toUpperCase())
const palette = ['#56758a', '#57766c', '#75647a', '#8a6c4d', '#8a5f62', '#666b82']
const color = computed(() => palette[Number(props.user?.id || 0) % palette.length])
</script>
<style scoped>.request-avatar { display: inline-grid; width: 24px; height: 24px; flex: 0 0 auto; place-items: center; border-radius: 6px; color: #fff; font-size: 10px; font-weight: 800; }</style>
