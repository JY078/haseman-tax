<template>
  <div
    :class="[
      'bg-white rounded-xl overflow-hidden transition-all duration-200',
      shadow ? 'shadow-sm hover:shadow-md' : '',
      hover ? 'hover:-translate-y-1 cursor-pointer' : '',
      bordered ? 'border border-neutral-200' : ''
    ]"
  >
    <slot />
  </div>
</template>

<script setup>
defineProps({
  shadow: {
    type: Boolean,
    default: true,
  },
  hover: {
    type: Boolean,
    default: false,
  },
  bordered: {
    type: Boolean,
    default: false,
  },
})
</script>
