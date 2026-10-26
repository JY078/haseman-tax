<template>
  <span v-if="messages?.length" ref="anchor" class="validation-dot" tabindex="0" role="button"
    :aria-label="`查看该季度汇总校验差异，共 ${messages.length} 项`"
    @mouseenter="show" @mouseleave="hide" @focus="show" @blur="hide">!</span>
  <Teleport to="body">
    <section v-if="open" ref="panel" class="validation-popover" :style="position" aria-label="汇总校验差异明细">
      <header class="popover-head"><strong>汇总校验差异</strong><span>{{ messages.length }} 项</span></header>
      <div class="popover-scroll">
        <article v-for="(item, index) in parsed" :key="index" class="issue" :class="{ unresolved: !item.issue }">
          <template v-if="item.issue">
            <div class="issue-head"><b>{{ item.issue.metric }}</b><span v-if="item.issue.range">{{ item.issue.range }}</span></div>
            <dl>
              <div><dt>明细</dt><dd>{{ item.issue.detail }}</dd></div>
              <div><dt>PDF</dt><dd>{{ item.issue.pdf }}</dd></div>
              <div class="diff"><dt>差额</dt><dd>{{ item.issue.difference }}{{ item.issue.currency ? ` ${item.issue.currency}` : '' }}</dd></div>
            </dl>
          </template>
          <p v-else class="issue-raw">{{ item.message }}</p>
        </article>
      </div>
    </section>
  </Teleport>
</template>
<script setup>
import { computed, nextTick, ref } from 'vue'
import { parseAmazonValidationMessages } from '@/utils/amazonValidation'
const props = defineProps({ messages: { type: Array, default: () => [] } })
const anchor = ref(null), panel = ref(null), open = ref(false), position = ref({})
const parsed = computed(() => parseAmazonValidationMessages(props.messages))
let hideTimer = null
async function show() {
  clearTimeout(hideTimer)
  if (!open.value) open.value = true
  await nextTick()
  if (!anchor.value || !panel.value) return
  const rect = anchor.value.getBoundingClientRect()
  const width = Math.min(340, window.innerWidth - 24)
  position.value = {
    width: `${width}px`,
    left: `${Math.max(12, Math.min(rect.right - width, window.innerWidth - width - 12))}px`,
    top: `${Math.min(rect.bottom + 8, window.innerHeight - 60)}px`,
  }
  await nextTick()
  const height = panel.value.getBoundingClientRect().height
  if (rect.bottom + height + 20 > window.innerHeight) position.value = { ...position.value, top: `${Math.max(12, rect.top - height - 8)}px` }
}
function hide() { hideTimer = setTimeout(() => { open.value = false }, 80) }
</script>
<style scoped>
.validation-dot { display:inline-grid; place-items:center; width:15px; height:15px; margin-left:6px; border-radius:50%; background:#c96745; color:#fff; font-size:10px; font-weight:700; vertical-align:1px; cursor:help; }
.validation-dot:focus-visible { outline:2px solid #285d83; outline-offset:2px; }
.validation-popover { position:fixed; z-index:1100; box-sizing:border-box; padding:10px 11px; border:1px solid #e7c2b1; border-radius:7px; background:#fffaf7; color:#6d493b; box-shadow:0 10px 24px rgb(20 35 50 / 14%); font-size:11px; text-align:left; }
.popover-head { display:flex; align-items:center; justify-content:space-between; gap:12px; padding-bottom:7px; border-bottom:1px solid #efcdbc; }
.popover-head strong { color:#7b3f2b; font-size:12px; }
.popover-head span { color:#9a7568; font-size:10px; }
.popover-scroll { display:grid; gap:7px; max-height:min(300px, 50vh); overflow:auto; margin-top:7px; scrollbar-width:thin; }
.issue { min-width:0; padding:7px 9px; border:1px solid #f0d9cd; border-radius:6px; background:#fff; }
.issue-head { display:flex; align-items:baseline; justify-content:space-between; gap:10px; }
.issue-head b { color:#6d493b; font-size:12px; }
.issue-head span { color:#9a7568; font-size:10px; font-variant-numeric:tabular-nums; }
.issue dl { display:flex; flex-wrap:wrap; gap:4px 14px; margin:6px 0 0; }
.issue dl>div { display:flex; align-items:baseline; gap:5px; }
.issue dt { color:#97786d; }
.issue dd { margin:0; color:#5e4036; font-weight:600; font-variant-numeric:tabular-nums; }
.issue .diff dd { color:#b65032; }
.issue-raw { margin:0; color:#97786d; overflow-wrap:anywhere; }
</style>
