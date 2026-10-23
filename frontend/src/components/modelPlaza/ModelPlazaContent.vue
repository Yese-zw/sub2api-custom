<template>
  <div class="space-y-5">
    <!-- 页头(独立形态下展示标题;后台形态 AppHeader 已有页面标题) -->
    <div v-if="!embedded">
      <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-3xl">{{ t('modelPlaza.title') }}</h1>
      <p class="mt-1.5 text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.description') }}</p>
    </div>

    <!-- 全局价格说明(管理员配置,Markdown) -->
    <div
      v-if="descriptionHtml"
      class="plaza-description rx-glass px-5 py-4 text-sm"
      v-html="descriptionHtml"
    ></div>

    <!-- 未登录提示 -->
    <p
      v-if="!isAuthenticated"
      class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-dark-500"
    >
      <Icon name="infoCircle" size="xs" class="h-3.5 w-3.5" />
      {{ t('modelPlaza.anonymousHint') }}
    </p>

    <!-- 加载/错误/空 -->
    <div v-if="loading" class="plaza-skeleton" aria-busy="true">
      <span class="plaza-skeleton__bar"></span>
      <span v-for="n in 3" :key="n" class="plaza-skeleton__card"></span>
    </div>
    <div
      v-else-if="error"
      class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
    >
      {{ t('modelPlaza.loadFailed') }}
    </div>
    <template v-else>
      <!-- 筛选区:平台 → 分组 → 倍率 -->
      <div class="rx-glass plaza-filter-shell rx-rise" style="--rx-i: 3">
      <PlazaFilterBar
        :platforms="platforms"
        :groups="groupOptions"
        :rates="rates"
        :platform="selectedPlatform"
        :group-id="selectedGroupId"
        :rate="selectedRate"
        :search="searchQuery"
        @update:platform="selectedPlatform = $event"
        @update:group-id="selectedGroupId = $event"
        @update:rate="selectedRate = $event"
        @update:search="searchQuery = $event"
      />
      </div>

      <!-- 分组分节的模型清单(默认按生效倍率升序) -->
      <TransitionGroup v-if="filteredGroups.length > 0" tag="div" name="plaza-list" class="relative space-y-5">
        <PlazaGroupSection
          v-for="(g, index) in filteredGroups"
          :key="g.id"
          :group="g"
          class="plaza-section-rise"
          :style="{ '--rx-i': index }"
        />
      </TransitionGroup>
      <div
        v-else
        class="plaza-empty"
      >
        <span class="plaza-empty__grid" aria-hidden="true"><i v-for="n in 9" :key="n"></i></span>
        {{ searchActive ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  /** 后台内嵌形态(AppLayout 内):隐藏页头。 */
  embedded?: boolean
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref('')

const searchActive = computed(() => searchQuery.value.trim() !== '')

const descriptionHtml = computed(() => {
  const md = props.response?.description?.trim()
  if (!md) return ''
  return DOMPurify.sanitize(marked.parse(md) as string)
})

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
function effectiveRate(g: ModelPlazaGroup): number {
  return g.user_rate_multiplier ?? g.rate_multiplier
}

const platforms = computed(() =>
  [...new Set((props.response?.groups ?? []).map((g) => g.platform).filter(Boolean))].sort()
)

const groupOptions = computed(() =>
  (props.response?.groups ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    platform: g.platform,
    rate: effectiveRate(g)
  }))
)

/** 全量生效倍率;当前组合下不可用的项由 FilterBar 置灰而非隐藏。 */
const rates = computed(() =>
  [...new Set((props.response?.groups ?? []).map(effectiveRate))].sort((a, b) => a - b)
)

/** 数据刷新后选中的倍率可能不复存在,重置为全部。 */
watch(rates, (list) => {
  if (selectedRate.value !== 'all' && !list.includes(selectedRate.value)) {
    selectedRate.value = 'all'
  }
})

const filteredGroups = computed(() => {
  let groups = props.response?.groups ?? []
  if (selectedPlatform.value !== 'all') {
    groups = groups.filter((g) => g.platform === selectedPlatform.value)
  }
  if (selectedGroupId.value !== 'all') {
    groups = groups.filter((g) => g.id === selectedGroupId.value)
  }
  if (selectedRate.value !== 'all') {
    groups = groups.filter((g) => effectiveRate(g) === selectedRate.value)
  }
  // 模型名搜索:分组内只留命中的模型,整组无命中则隐藏该分组。
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    groups = groups
      .map((g) => ({ ...g, models: g.models.filter((m) => m.name.toLowerCase().includes(q)) }))
      .filter((g) => g.models.length > 0)
  }
  // 专属倍率会改变生效值,不能只依赖后端按默认倍率的排序。
  return [...groups].sort(
    (a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name)
  )
})
</script>

<style scoped>
.plaza-filter-shell {
  position: relative;
  z-index: 2;
  padding: 1.1rem 1.25rem;
}

.plaza-section-rise {
  animation: plaza-rise 0.6s var(--rx-ease, ease) both;
  animation-delay: calc(min(var(--rx-i, 0), 6) * 70ms + 200ms);
}

@keyframes plaza-rise {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: none; }
}

/* 筛选切换时分组卡片的进出与重排 */
.plaza-list-move {
  transition: transform 0.45s var(--rx-ease, ease);
}

.plaza-list-enter-active,
.plaza-list-leave-active {
  transition: opacity 0.3s ease, transform 0.35s var(--rx-ease, ease);
}

.plaza-list-enter-from,
.plaza-list-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.99);
}

.plaza-list-leave-active {
  position: absolute;
  width: 100%;
}

.plaza-skeleton {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.plaza-skeleton__bar,
.plaza-skeleton__card {
  display: block;
  border-radius: var(--rx-radius-lg, 1rem);
  background: linear-gradient(100deg, var(--rx-surface-2, #f3f4f6) 30%, var(--rx-surface-hover, #fff) 50%, var(--rx-surface-2, #f3f4f6) 70%);
  background-size: 200% 100%;
  animation: plaza-shimmer 1.5s ease-in-out infinite;
}

.plaza-skeleton__bar {
  height: 9rem;
}

.plaza-skeleton__card {
  height: 14rem;
}

@keyframes plaza-shimmer {
  to { background-position: -200% 0; }
}

.plaza-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 3.5rem 1.25rem;
  border-radius: var(--rx-radius-lg, 1rem);
  border: 1px dashed var(--rx-line-strong, #d1d5db);
  font-size: 0.875rem;
  color: var(--rx-muted, #6b7280);
}

/* 空态：九宫格中只有一格被点亮，缓慢游走 */
.plaza-empty__grid {
  display: grid;
  grid-template-columns: repeat(3, 0.7rem);
  gap: 0.3rem;
}

.plaza-empty__grid i {
  width: 0.7rem;
  height: 0.7rem;
  border-radius: 3px;
  background: var(--rx-line, #e5e7eb);
  animation: plaza-cell 4.5s steps(1) infinite;
  animation-delay: calc(var(--n, 0) * 0.5s);
}

.plaza-empty__grid i:nth-child(1) { --n: 0; }
.plaza-empty__grid i:nth-child(2) { --n: 1; }
.plaza-empty__grid i:nth-child(3) { --n: 2; }
.plaza-empty__grid i:nth-child(6) { --n: 3; }
.plaza-empty__grid i:nth-child(9) { --n: 4; }
.plaza-empty__grid i:nth-child(8) { --n: 5; }
.plaza-empty__grid i:nth-child(7) { --n: 6; }
.plaza-empty__grid i:nth-child(4) { --n: 7; }
.plaza-empty__grid i:nth-child(5) { --n: 8; }

@keyframes plaza-cell {
  0%, 11% { background: var(--rx-accent, #111827); }
  11.1%, 100% { background: var(--rx-line, #e5e7eb); }
}

@media (prefers-reduced-motion: reduce) {
  .plaza-section-rise,
  .plaza-empty__grid i {
    animation: none;
  }
}

.plaza-description {
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.plaza-description :deep(h1),
.plaza-description :deep(h2),
.plaza-description :deep(h3) {
  @apply mb-2 mt-3 font-semibold text-gray-900 first:mt-0 dark:text-white;
}

.plaza-description :deep(p) {
  @apply mb-2 text-gray-700 last:mb-0 dark:text-dark-200;
}

.plaza-description :deep(a) {
  @apply text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300;
}

.plaza-description :deep(ul) {
  @apply mb-2 list-disc pl-5;
}

.plaza-description :deep(ol) {
  @apply mb-2 list-decimal pl-5;
}

.plaza-description :deep(li) {
  @apply mb-0.5 text-gray-700 dark:text-dark-200;
}

.plaza-description :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs dark:bg-dark-800;
}

.plaza-description :deep(blockquote) {
  @apply my-2 border-l-4 border-gray-300 pl-3 text-gray-600 dark:border-dark-600 dark:text-dark-300;
}
</style>
