<template>
  <MainLayout>
    <section class="flex min-h-[calc(100vh-4rem)] items-center justify-center px-4 py-16">
      <div class="w-full max-w-sm text-center">
        <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-neutral-900 text-sm font-bold text-white">HSM</div>
        <h1 class="mt-5 text-2xl font-semibold text-neutral-900">登录 HSM</h1>
        <p class="mt-2 text-sm text-neutral-500">登录后进入内部系统</p>
        <Button class="mt-7 w-full" @click="authOpen = true">登录</Button>
      </div>
    </section>
    <AuthModal v-model:open="authOpen" />
  </MainLayout>
</template>

<script setup>
import { ref } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import AuthModal from '@/components/ui/AuthModal.vue'
import Button from '@/components/ui/Button.vue'

const authOpen = ref(false)
</script>
