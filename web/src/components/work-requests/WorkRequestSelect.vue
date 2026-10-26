<template>
  <Listbox :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <div class="request-select">
      <ListboxButton :id="id" class="request-select-button">
        <span :class="{ placeholder: !modelValue }">{{ selectedLabel }}</span>
        <ChevronDownIcon class="request-select-arrow" aria-hidden="true" />
      </ListboxButton>
      <transition
        enter-active-class="select-enter-active"
        enter-from-class="select-enter-from"
        enter-to-class="select-enter-to"
        leave-active-class="select-leave-active"
        leave-from-class="select-leave-from"
        leave-to-class="select-leave-to"
      >
        <ListboxOptions class="request-select-options">
          <ListboxOption v-slot="{ active, selected }" as="template" value="">
            <li class="request-select-option" :class="{ active }">
              <span :class="{ selected, placeholder: !selected }">{{ placeholder }}</span>
              <CheckIcon v-if="selected" class="request-select-check" aria-hidden="true" />
            </li>
          </ListboxOption>
          <ListboxOption v-for="option in options" :key="option.value" v-slot="{ active, selected }" as="template" :value="option.value">
            <li class="request-select-option" :class="{ active }">
              <span :class="{ selected }">{{ option.label }}</span>
              <CheckIcon v-if="selected" class="request-select-check" aria-hidden="true" />
            </li>
          </ListboxOption>
        </ListboxOptions>
      </transition>
    </div>
  </Listbox>
</template>

<script setup>
import { computed } from 'vue'
import { Listbox, ListboxButton, ListboxOption, ListboxOptions } from '@headlessui/vue'
import { CheckIcon, ChevronDownIcon } from '@heroicons/vue/20/solid'

const props = defineProps({
  id: { type: String, default: '' },
  modelValue: { type: String, default: '' },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '请选择' },
})
const emit = defineEmits(['update:modelValue'])
const selectedLabel = computed(() => props.options.find(option => option.value === props.modelValue)?.label || props.placeholder)
</script>

<style scoped>
.request-select { position: relative; width: 100%; }
.request-select-button { display: flex; width: 100%; height: 42px; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid #d9d6d0; border-radius: 6px; padding: 0 12px; color: #45413c; background: #fff; font-size: 13px; text-align: left; outline: none; transition: border-color .16s ease, box-shadow .16s ease, background .16s ease; }
.request-select-button:hover { border-color: #bdb8b0; background: #fff; }
.request-select-button:focus { border-color: #77736d; box-shadow: 0 0 0 3px rgb(71 67 61 / 7%); }
.request-select-button .placeholder, .request-select-option .placeholder { color: #aaa49c; }
.request-select-arrow { width: 16px; height: 16px; flex: 0 0 auto; color: #817b73; }
.request-select-options { position: absolute; z-index: 30; top: calc(100% + 6px); width: 100%; max-height: 240px; overflow-y: auto; border: 1px solid #dfdcd7; border-radius: 7px; padding: 5px; background: #fff; box-shadow: 0 12px 28px rgb(65 58 49 / 13%), 0 2px 8px rgb(65 58 49 / 7%); outline: none; transform-origin: top; }
.request-select-option { display: flex; min-height: 36px; align-items: center; justify-content: space-between; gap: 10px; border-radius: 5px; padding: 0 9px; color: #4d4944; font-size: 13px; cursor: pointer; }
.request-select-option.active { color: #292724; background: #f3f1ed; }
.request-select-option .selected { color: #292724; font-weight: 700; }
.request-select-check { width: 16px; height: 16px; color: #4e7454; }
.select-enter-active, .select-leave-active { transition: opacity .14s ease, transform .14s ease; }
.select-enter-from, .select-leave-to { opacity: 0; transform: translateY(-4px) scale(.98); }
</style>
