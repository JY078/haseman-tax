<template>
  <Menu as="div" class="relative inline-block text-left">
    <div>
      <MenuButton
        class="inline-flex w-full justify-center gap-x-1.5 rounded-lg bg-white px-4 py-2.5 text-sm font-medium text-neutral-700 shadow-sm ring-1 ring-inset ring-neutral-200 hover:bg-neutral-50"
      >
        {{ modelValue || placeholder }}
        <ChevronDownIcon class="-mr-1 h-5 w-5 text-neutral-400" aria-hidden="true" />
      </MenuButton>
    </div>

    <transition
      enter-active-class="transition ease-out duration-100"
      enter-from-class="transform opacity-0 scale-95"
      enter-to-class="transform opacity-100 scale-100"
      leave-active-class="transition ease-in duration-75"
      leave-from-class="transform opacity-100 scale-100"
      leave-to-class="transform opacity-0 scale-95"
    >
      <MenuItems
        class="absolute right-0 z-10 mt-2 w-56 origin-top-right rounded-lg bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none"
      >
        <div class="py-1">
          <MenuItem v-for="option in options" :key="option.value" v-slot="{ active }">
            <button
              :class="[
                active ? 'bg-neutral-50 text-neutral-900' : 'text-neutral-700',
                'block w-full px-4 py-2.5 text-left text-sm'
              ]"
              @click="$emit('update:modelValue', option.value)"
            >
              {{ option.label }}
            </button>
          </MenuItem>
        </div>
      </MenuItems>
    </transition>
  </Menu>
</template>

<script setup>
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { ChevronDownIcon } from '@heroicons/vue/20/solid'

defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  options: {
    type: Array,
    required: true,
  },
  placeholder: {
    type: String,
    default: '请选择',
  },
})

defineEmits(['update:modelValue'])
</script>
