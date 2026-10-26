<template>
  <div class="min-h-screen flex flex-col bg-neutral-50">
    <Header :locale="locale" />
    <main class="flex-1">
      <slot />
    </main>
    <AnnouncementModal v-if="!callsOnly" ref="announcementRef" />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, watch } from 'vue'
import Header from './Header.vue'
import AnnouncementModal from '@/components/ui/AnnouncementModal.vue'
import { useAuthStore } from '@/stores/auth'
import { isCustomerServiceUser } from '@/utils/userLandingRoute'

defineProps({ locale: { type: String, default: '' } })

const authStore = useAuthStore()
const callsOnly = computed(() => isCustomerServiceUser(authStore.user))
const announcementRef = ref(null)

function tryCheckAnnouncements() {
  if (authStore.isLoggedIn && announcementRef.value) {
    announcementRef.value.checkUnread()
  }
}

onMounted(() => {
  tryCheckAnnouncements()
})

watch(() => authStore.isLoggedIn, (loggedIn) => {
  if (loggedIn) {
    setTimeout(tryCheckAnnouncements, 500)
  }
})
</script>
