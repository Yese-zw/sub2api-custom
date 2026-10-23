<template>
  <!-- 后台内嵌形态:?embedded=1 且已登录,套完整后台布局 -->
  <AppLayout v-if="isEmbedded">
    <RxStage wide>
      <RxPageHero
        :eyebrow="t('modelPlaza.hero.eyebrow')"
        :lead="t('modelPlaza.hero.titleLead')"
        :accent="t('modelPlaza.hero.titleAccent')"
        :subtitle="t('modelPlaza.description')"
      >
        <template #aside><PlazaStats :response="data" :loading="loading" /></template>
      </RxPageHero>
      <ModelPlazaContent :response="data" :loading="loading" :error="loadFailed" embedded />
    </RxStage>
  </AppLayout>

  <!-- 独立形态:自带导航条(logo/站名 + 登录/回后台) -->
  <div v-else class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <PlazaNavBar />
    <main class="px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
      <RxStage>
        <RxPageHero
          :eyebrow="t('modelPlaza.hero.eyebrow')"
          :lead="t('modelPlaza.hero.titleLead')"
          :accent="t('modelPlaza.hero.titleAccent')"
          :subtitle="t('modelPlaza.description')"
        >
          <template #aside><PlazaStats :response="data" :loading="loading" /></template>
        </RxPageHero>
        <ModelPlazaContent :response="data" :loading="loading" :error="loadFailed" embedded />
      </RxStage>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PlazaNavBar from '@/components/modelPlaza/PlazaNavBar.vue'
import ModelPlazaContent from '@/components/modelPlaza/ModelPlazaContent.vue'
import PlazaStats from '@/components/modelPlaza/PlazaStats.vue'
import RxStage from '@/components/common/RxStage.vue'
import RxPageHero from '@/components/common/RxPageHero.vue'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

// embedded=1 但未登录(如转发的链接)自动降级为独立形态。
const isEmbedded = computed(() => route.query.embedded === '1' && authStore.isAuthenticated)

const data = ref<ModelPlazaResponse | null>(null)
const loading = ref(true)
const loadFailed = ref(false)

onMounted(async () => {
  // 独立形态导航条需要站点名/Logo;有 __APP_CONFIG__ 注入时同步命中缓存。
  void appStore.fetchPublicSettings()
  try {
    data.value = await getModelPlaza()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
})
</script>
