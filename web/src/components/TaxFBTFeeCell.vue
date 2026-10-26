<template>
  <span class="fbt-cell"><span v-if="row.fbt_fee == null" class="pending">待换算</span><span v-else>{{ money(row.fbt_fee) }}</span></span>
</template>
<script setup>
import { computed } from 'vue'
const props = defineProps({ row: { type: Object, required: true }, currency: { type: String, required: true } })
const symbol = computed(() => props.currency === 'USD' ? '$' : '¥')
const money = value => value == null ? '待换算' : Number(value) === 0 ? '-' : symbol.value + Number(value).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
</script>
<style scoped>
.fbt-cell { display:inline-flex; justify-content:flex-end; width:100%; font-variant-numeric:tabular-nums; }
.pending { color:#926a28; }
</style>
