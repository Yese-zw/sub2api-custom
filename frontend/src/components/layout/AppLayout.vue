<template>
  <div class="app-shell min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Background Decoration -->
    <div class="app-grid pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="p-4 md:p-6 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style>
html.pixel-ui .app-shell { background: var(--pixel-canvas); }
html.pixel-ui .app-grid { background-image: linear-gradient(var(--pixel-grid) 1px, transparent 1px), linear-gradient(90deg, var(--pixel-grid) 1px, transparent 1px); background-size: 32px 32px; }

/* 编辑型卡片风：暖白画布 + 顶部柔光，取代像素网格底纹 */
html.editorial-ui .app-shell { background: var(--ed-canvas); }
html.editorial-ui .app-grid {
  background-image: radial-gradient(circle at 50% 0%, rgba(255, 255, 255, 0.85), transparent 42%);
  opacity: 1;
}
html.editorial-ui.dark .app-grid {
  background-image: radial-gradient(circle at 50% 0%, rgba(255, 255, 255, 0.06), transparent 42%);
}
</style>
