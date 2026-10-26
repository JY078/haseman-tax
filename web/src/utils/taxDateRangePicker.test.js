import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { parse, compileScript } from '@vue/compiler-sfc'
import { ref, computed, nextTick } from 'vue'

const { descriptor } = parse(readFileSync(new URL('../components/TaxDateRangePicker.vue', import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'tax-period-test' }).content
const create = new Function('computed', 'nextTick', 'onBeforeUnmount', 'onMounted', 'ref', 'watch', script.replace(/^import .*$/gm, '').replace('export default', 'return'))
function setup(from = '', to = '') {
  const events = []
  const component = create(computed, nextTick, () => {}, () => {}, ref, () => {})
  const state = component.setup({ from, to, label: '范围' }, { expose() {}, emit: (...args) => events.push(args) })
  state.syncDraft()
  return { state, events }
}

test('period selection remains a draft until apply, including a single quarter', () => {
  const { state, events } = setup()
  state.selectPeriod('2026-Q2')
  assert.deepEqual(events, [])
  assert.equal(state.triggerText.value, '全部时间')
  assert.deepEqual(state.draftRange.value, { from: '2026-04-01', to: '2026-06-30' })
  state.apply()
  assert.deepEqual(events, [['update:from', '2026-04-01'], ['update:to', '2026-06-30'], ['change']])
})

test('reverse cross-year selections sort their boundaries', () => {
  const { state } = setup()
  state.selectPeriod('2026-Q2')
  state.selectPeriod('2025-Q4')
  assert.deepEqual(state.draftRange.value, { from: '2025-10-01', to: '2026-06-30' })
  assert.equal(state.cellClass('2026-Q1').selected, true)
  assert.equal(state.cellClass('2025-Q3').selected, false)
})

test('month selections include the leap-year month end', () => {
  const { state, events } = setup()
  state.changeMode('month')
  state.selectPeriod('2024-02')
  assert.deepEqual(state.draftRange.value, { from: '2024-02-01', to: '2024-02-29' })
  assert.deepEqual(events, [])
})

test('cancel and reopen discard pending changes and mode changes preserve applied dates', async () => {
  const { state, events } = setup('2026-02-05', '2026-03-12')
  state.changeMode('month')
  assert.deepEqual(state.draftRange.value, { from: '2026-02-05', to: '2026-03-12' })
  state.selectPeriod('2026-07')
  state.close()
  await state.toggle()
  assert.deepEqual(state.draftRange.value, { from: '2026-02-05', to: '2026-03-12' })
  state.apply()
  assert.deepEqual(events, [])
})

test('quick presets and reset only emit after confirmation', () => {
  const { state, events } = setup('2026-01-01', '2026-03-31')
  state.quickSelect('last')
  assert.equal(state.quickActive('last'), true)
  state.clear()
  assert.deepEqual(events, [])
  assert.equal(state.quickActive('all'), true)
  state.apply()
  assert.deepEqual(events, [['update:from', ''], ['update:to', ''], ['change']])
})

test('escape cancels the draft and closes the panel', () => {
  const { state, events } = setup()
  state.open.value = true
  state.selectPeriod('2026-Q1')
  let prevented = false
  state.onDocKeydown({ key: 'Escape', preventDefault() { prevented = true } })
  assert.equal(prevented, true)
  assert.equal(state.open.value, false)
  assert.deepEqual(events, [])
})
