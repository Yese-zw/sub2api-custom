<template>
  <div>
    <div
      v-if="loading && items.length === 0"
      class="grid gap-5 grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
    >
      <div
        v-for="i in 6"
        :key="i"
        class="p-5 rounded-2xl min-h-[280px] bg-white/70 dark:bg-dark-800/60 border border-gray-200/80 dark:border-dark-700/70 animate-pulse"
      >
        <div class="flex items-start gap-3">
          <div class="w-9 h-9 rounded-xl bg-gray-200 dark:bg-dark-700"></div>
          <div class="flex-1 space-y-2">
            <div class="h-4 w-2/3 rounded bg-gray-200 dark:bg-dark-700"></div>
            <div class="h-3 w-1/2 rounded bg-gray-200 dark:bg-dark-700"></div>
          </div>
          <div class="h-6 w-16 rounded-full bg-gray-200 dark:bg-dark-700"></div>
        </div>
        <div class="mt-5 grid grid-cols-2 gap-2">
          <div class="h-16 rounded-xl bg-gray-100 dark:bg-dark-900/40"></div>
          <div class="h-16 rounded-xl bg-gray-100 dark:bg-dark-900/40"></div>
        </div>
        <div class="mt-6 h-5 w-full rounded bg-gray-100 dark:bg-dark-900/40"></div>
      </div>
    </div>

    <EmptyState
      v-else-if="items.length === 0"
      :title="t('channelStatus.empty.title')"
      :description="t('channelStatus.empty.description')"
    />

    <!-- 按平台（provider）分组：每组一个标题 + 该平台的卡片网格 -->
    <div v-else class="space-y-8">
      <section v-for="group in groups" :key="group.provider" :aria-label="group.label">
        <header class="mb-3 flex items-center gap-2">
          <ProviderIcon :provider="group.provider" :size="18" />
          <h2 class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ group.label }}</h2>
          <span
            class="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium tabular-nums text-gray-600 dark:bg-dark-700 dark:text-gray-300"
          >
            {{ group.items.length }}
          </span>
          <span
            v-if="group.issueCount > 0"
            class="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
          >
            {{ t('channelStatus.groupIssues', { n: group.issueCount }) }}
          </span>
        </header>
        <div class="rx-monitor-grid grid gap-5 grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          <MonitorCard
            v-for="item in group.items"
            :key="item.id"
            :item="item"
            :window="window"
            :availability-value="resolveAvailability(item)"
            :countdown-seconds="countdownSeconds"
            @click="emit('cardClick', item)"
          />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView, UserMonitorDetail } from '@/api/channelMonitor'
import EmptyState from '@/components/common/EmptyState.vue'
import MonitorCard from './MonitorCard.vue'
import ProviderIcon from './ProviderIcon.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import { STATUS_OPERATIONAL } from '@/constants/channelMonitor'

const props = defineProps<{
  items: UserMonitorView[]
  window: '7d' | '15d' | '30d'
  countdownSeconds: number
  loading: boolean
  detailCache: Record<number, UserMonitorDetail>
}>()

const emit = defineEmits<{
  (e: 'cardClick', item: UserMonitorView): void
}>()

const { t } = useI18n()
const { providerLabel } = useChannelMonitorFormat()

// 按 provider 分组，组顺序取各平台首次出现的顺序，组内保持原顺序。
const groups = computed(() => {
  const map = new Map<string, UserMonitorView[]>()
  for (const item of props.items) {
    const list = map.get(item.provider)
    if (list) list.push(item)
    else map.set(item.provider, [item])
  }
  return Array.from(map, ([provider, items]) => ({
    provider,
    label: providerLabel(provider),
    items,
    issueCount: items.filter(i => i.primary_status !== STATUS_OPERATIONAL).length,
  }))
})

function resolveAvailability(item: UserMonitorView): number | null {
  if (props.window === '7d') {
    return item.availability_7d ?? null
  }
  const detail = props.detailCache[item.id]
  if (!detail) return null
  const primary = detail.models.find(m => m.model === item.primary_model)
  if (!primary) return null
  return props.window === '15d' ? primary.availability_15d ?? null : primary.availability_30d ?? null
}
</script>

<style scoped>
.rx-monitor-grid {
  counter-reset: monitor-idx;
}

.rx-monitor-grid :deep(.rx-monitor-card) {
  counter-increment: monitor-idx;
}

.rx-monitor-grid :deep(.rx-monitor-card)::after {
  content: counter(monitor-idx, decimal-leading-zero);
  position: absolute;
  right: 0.95rem;
  bottom: 0.7rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 0.82rem;
  color: var(--rx-faint);
  opacity: 0.5;
  pointer-events: none;
}
</style>
