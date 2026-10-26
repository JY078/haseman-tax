<template>
  <div ref="pickerRef" class="calendar-picker">
    <button
      type="button"
      ref="triggerRef"
      class="date-picker-button"
      aria-haspopup="dialog"
      :aria-expanded="calendarOpen"
      :aria-label="ariaLabel"
      @click="toggleCalendar"
    >
      <span :class="modelValue ? 'date-value' : 'date-placeholder'">
        {{ displayValue }}
      </span>
      <CalendarDaysIcon class="h-4 w-4" aria-hidden="true" />
    </button>

    <div v-if="calendarOpen" class="calendar-popover" role="dialog" :aria-label="ariaLabel">
      <div class="picker-modes" role="group" :aria-label="en ? 'Date filter mode' : '日期筛选方式'">
        <button v-for="option in ['day', 'month', 'year']" :key="option" type="button" :class="{ active: mode === option }" :aria-pressed="mode === option" @click="changeMode(option)">{{ en ? {day:'Day',month:'Month',year:'Year'}[option] : {day:'按天',month:'按月',year:'按年'}[option] }}</button>
      </div>
      <div class="calendar-toolbar">
        <button type="button" class="calendar-nav" :aria-label="en ? 'Previous period' : '上一时段'" @click="shiftMonth(-1)">
          <ChevronLeftIcon class="h-4 w-4" />
        </button>
        <strong>{{ monthLabel }}</strong>
        <button type="button" class="calendar-nav" :aria-label="en ? 'Next period' : '下一时段'" @click="shiftMonth(1)">
          <ChevronRightIcon class="h-4 w-4" />
        </button>
      </div>
      <div v-if="mode === 'day'" class="calendar-weekdays" aria-hidden="true">
        <span v-for="label in weekLabels" :key="label">{{ label }}</span>
      </div>
      <div v-if="mode === 'day'" class="calendar-grid">
        <button
          v-for="day in calendarDays"
          :key="day.key"
          type="button"
          class="calendar-day"
          :aria-label="day.key"
          :class="{ 'outside-month': !day.isCurrentMonth, today: day.isToday, selected: day.isSelected }"
          @click="chooseDate(day.key)"
        >
          {{ day.day }}
        </button>
      </div>
      <div v-else-if="mode === 'month'" class="month-grid">
        <button v-for="month in calendarMonths" :key="month.key" type="button" class="month-option" :class="{ selected: month.key === modelValue }" @click="chooseMonth(month.key)">{{ month.label }}</button>
      </div>
      <div v-else class="month-grid"><button v-for="year in calendarYears" :key="year" type="button" class="month-option" :class="{ selected: String(year) === modelValue }" @click="chooseYear(year)">{{ year }}{{ en ? '' : '年' }}</button></div>
      <div class="calendar-footer">
        <button type="button" class="today-button" @click="mode === 'year' ? chooseYear(new Date().getFullYear()) : mode === 'month' ? selectCurrentMonth() : selectToday()">{{ en ? {day:'Today',month:'This month',year:'This year'}[mode] : {day:'今天',month:'本月',year:'今年'}[mode] }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { CalendarDaysIcon, ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '请选择日期' },
  ariaLabel: { type: String, default: '选择日期' },
  mode: { type: String, default: 'day' },
  locale: { type: String, default: 'zh' },
})
const emit = defineEmits(['update:modelValue', 'change', 'update:mode'])

const pickerRef = ref(null)
const triggerRef = ref(null)
const calendarOpen = ref(false)
const calendarMonth = ref(startOfMonth(new Date()))
const en = computed(() => props.locale === 'en')
const weekLabels = computed(() => en.value ? ['Mo','Tu','We','Th','Fr','Sa','Su'] : ['一', '二', '三', '四', '五', '六', '日'])
const monthLabel = computed(() => props.mode === 'year' ? `${calendarYears.value[0]} – ${calendarYears.value[11]}` : props.mode === 'month' ? String(calendarMonth.value.getFullYear()) : calendarMonth.value.toLocaleDateString(en.value ? 'en-US' : 'zh-CN', { year: 'numeric', month: 'long' }))
const displayValue = computed(() => {
  if (!props.modelValue) return props.placeholder
  if (en.value) return props.modelValue
  if (props.mode === 'year') return props.modelValue ? `${props.modelValue}年` : props.placeholder
  if (props.mode === 'month') {
    const [year, month] = props.modelValue.split('-').map(Number)
    return year && month ? `${year}年${month}月` : props.modelValue
  }
  return props.modelValue
})
const calendarMonths = computed(() => Array.from({ length: 12 }, (_, index) => ({
  key: `${calendarMonth.value.getFullYear()}-${padDatePart(index + 1)}`,
  label: en.value ? new Date(2024,index,1).toLocaleDateString('en-US',{month:'short'}) : `${index + 1}月`,
})))
const calendarYears = computed(() => Array.from({ length: 12 }, (_, index) => calendarMonth.value.getFullYear() - 5 + index))
const calendarDays = computed(() => buildCalendarDays(calendarMonth.value, props.modelValue))

function padDatePart(value) {
  return String(value).padStart(2, '0')
}

function dateKey(date) {
  return `${date.getFullYear()}-${padDatePart(date.getMonth() + 1)}-${padDatePart(date.getDate())}`
}

function parseDateKey(value) {
  if (!value) return null
  const [year, month, day] = String(value).split('-').map(Number)
  if (!year || !month || !day) return null
  const date = new Date(year, month - 1, day)
  return Number.isNaN(date.getTime()) ? null : date
}

function startOfMonth(date) {
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

function buildCalendarDays(monthDate, selectedValue) {
  const firstDay = startOfMonth(monthDate)
  const mondayOffset = (firstDay.getDay() + 6) % 7
  const gridStart = new Date(firstDay.getFullYear(), firstDay.getMonth(), 1 - mondayOffset)
  const todayKey = dateKey(new Date())
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(gridStart)
    date.setDate(gridStart.getDate() + index)
    const key = dateKey(date)
    return {
      key,
      day: date.getDate(),
      isCurrentMonth: date.getMonth() === firstDay.getMonth(),
      isToday: key === todayKey,
      isSelected: key === selectedValue,
    }
  })
}

function toggleCalendar() {
  calendarOpen.value = !calendarOpen.value
  if (calendarOpen.value) {
    const value = props.mode === 'month' ? parseMonthKey(props.modelValue) : props.mode === 'year' ? parseYearKey(props.modelValue) : parseDateKey(props.modelValue)
    calendarMonth.value = startOfMonth(value || new Date())
  }
}

function shiftMonth(offset) {
  calendarMonth.value = new Date(calendarMonth.value.getFullYear(), calendarMonth.value.getMonth() + offset * (props.mode === 'year' ? 144 : props.mode === 'month' ? 12 : 1), 1)
}

function chooseDate(value) {
  emit('update:modelValue', value)
  emit('change', value)
  calendarOpen.value = false
  triggerRef.value?.focus()
}

function parseMonthKey(value) {
  if (!/^\d{4}-\d{2}$/.test(String(value || ''))) return null
  const [year, month] = value.split('-').map(Number)
  return new Date(year, month - 1, 1)
}

function parseYearKey(value) {
  if (!/^\d{4}$/.test(String(value || ''))) return null
  return new Date(Number(value), 0, 1)
}

function changeMode(value) {
  if (value === props.mode) return
  emit('update:mode', value)
  emit('update:modelValue', '')
  emit('change', '')
  calendarMonth.value = startOfMonth(new Date())
}

function chooseMonth(value) {
  emit('update:modelValue', value)
  emit('change', value)
  calendarOpen.value = false
  triggerRef.value?.focus()
}
function chooseYear(value) { chooseDate(String(value)) }

function selectCurrentMonth() {
  const now = new Date()
  chooseMonth(`${now.getFullYear()}-${padDatePart(now.getMonth() + 1)}`)
}

function selectToday() {
  const today = new Date()
  calendarMonth.value = startOfMonth(today)
  chooseDate(dateKey(today))
}

function handleDocumentPointerDown(event) {
  if (calendarOpen.value && !pickerRef.value?.contains(event.target)) calendarOpen.value = false
}

function handleDocumentKeydown(event) {
  if (event.key === 'Escape' && calendarOpen.value) { calendarOpen.value = false; triggerRef.value?.focus() }
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown)
  document.addEventListener('keydown', handleDocumentKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown)
  document.removeEventListener('keydown', handleDocumentKeydown)
})
</script>

<style scoped>
.calendar-picker { position: relative; }
.date-picker-button { width: 100%; height: 40px; padding: 0 12px; display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid #d5dae1; border-radius: 7px; background: #fff; color: #667085; text-align: left; outline: none; box-shadow: 0 1px 2px rgba(16,24,40,.03); transition: border-color .15s, box-shadow .15s, background-color .15s; }
.date-picker-button:hover { border-color: #aeb6c2; background: #fcfcfd; }
.date-picker-button:focus-visible { border-color: #4b5563; box-shadow: 0 0 0 3px rgba(75,85,99,.1); }
.date-picker-button > svg { flex: 0 0 auto; color: #667085; }
.date-placeholder { color: #a1a8b3; font-size: 13px; }
.date-value { color: #111827; font-size: 13px; font-variant-numeric: tabular-nums; }
.calendar-popover { position: absolute; z-index: 30; top: calc(100% + 6px); left: 0; width: min(320px, calc(100vw - 32px)); padding: 12px; border: 1px solid #dfe3e8; border-radius: 8px; background: #fff; box-shadow: 0 14px 34px rgba(16,24,40,.16), 0 2px 8px rgba(16,24,40,.06); }
.picker-modes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 3px; margin-bottom: 12px; padding: 3px; border-radius: 6px; background: #f2f4f7; }
.picker-modes button { height: 28px; border-radius: 4px; color: #667085; font-size: 11px; font-weight: 650; transition: background-color .12s, color .12s, box-shadow .12s; }
.picker-modes button:hover { color: #111827; }
.picker-modes button.active { background: #fff; color: #111827; box-shadow: 0 1px 2px rgba(16,24,40,.12); }
.calendar-toolbar { min-height: 34px; padding-bottom: 10px; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #edf0f2; }
.calendar-toolbar strong { color: #20252b; font-size: 13px; line-height: 20px; font-weight: 700; }
.calendar-nav { width: 30px; height: 30px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid transparent; border-radius: 6px; color: #667085; }
.calendar-nav:hover { border-color: #dfe3e8; background: #f4f5f7; color: #111827; }
.calendar-nav:focus-visible, .calendar-day:focus-visible, .today-button:focus-visible { outline: 2px solid #667085; outline-offset: 1px; }
.calendar-weekdays, .calendar-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 3px; }
.calendar-weekdays { margin-top: 10px; color: #98a2b3; font-size: 10px; line-height: 18px; text-align: center; }
.calendar-grid { margin-top: 3px; }
.month-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 6px; margin-top: 14px; }
.month-option { height: 42px; border-radius: 6px; color: #344054; font-size: 12px; font-weight: 600; transition: background-color .12s, color .12s; }
.month-option:hover { background: #f1f3f5; color: #111827; }
.month-option.selected { background: #20252b; color: #fff; }
.calendar-day { aspect-ratio: 1; min-width: 0; display: inline-flex; align-items: center; justify-content: center; border-radius: 5px; color: #344054; font-size: 12px; line-height: 1; font-variant-numeric: tabular-nums; transition: background-color .12s, color .12s; }
.calendar-day:hover { background: #f1f3f5; color: #111827; }
.calendar-day.outside-month { color: #b5bbc3; }
.calendar-day.today { box-shadow: inset 0 0 0 1px #98a2b3; }
.calendar-day.selected { background: #20252b; color: #fff; box-shadow: none; font-weight: 700; }
.calendar-day.selected:hover { background: #111827; color: #fff; }
.calendar-footer { margin-top: 10px; padding-top: 10px; display: flex; justify-content: flex-end; border-top: 1px solid #edf0f2; }
.today-button { height: 28px; padding: 0 9px; border-radius: 5px; color: #475467; font-size: 11px; line-height: 16px; font-weight: 650; }
.today-button:hover { background: #f1f3f5; color: #111827; }

@media (max-width: 768px) {
  .calendar-popover { position: fixed; top: auto; right: 16px; bottom: 16px; left: 16px; width: auto; }
}
</style>
