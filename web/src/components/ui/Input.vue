<template>
  <div class="w-full">
    <label v-if="label" :for="id" class="block text-sm font-medium text-neutral-700 mb-1.5">
      {{ label }}
    </label>
    <input
      :id="id"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="$emit('update:modelValue', $event.target.value)"
      :class="[
        'w-full px-4 py-2.5 rounded-lg border transition-all duration-200 focus:outline-none',
        error
          ? 'border-red-300 focus:border-red-500'
          : 'border-neutral-200 focus:border-neutral-400',
        { 'bg-neutral-50 cursor-not-allowed': disabled }
      ]"
    />
    <p v-if="error" class="mt-1.5 text-sm text-red-500">{{ error }}</p>
    <p v-if="hint && !error" class="mt-1.5 text-sm text-neutral-500">{{ hint }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  label: {
    type: String,
    default: '',
  },
  type: {
    type: String,
    default: 'text',
  },
  placeholder: {
    type: String,
    default: '',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: '',
  },
  hint: {
    type: String,
    default: '',
  },
})

defineEmits(['update:modelValue'])

const id = computed(() => `input-${Math.random().toString(36).substr(2, 9)}`)
</script>
