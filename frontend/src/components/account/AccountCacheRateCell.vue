<template>
  <div class="text-xs" data-testid="account-cache-rate-cell">
    <div
      v-if="loading && !data"
      class="h-3 w-12 animate-pulse rounded bg-gray-200 dark:bg-gray-700"
      data-testid="account-cache-rate-loading"
    />
    <div v-else-if="error && !data" class="text-red-500">{{ error }}</div>
    <span v-else-if="rate == null" class="text-gray-400" data-testid="account-cache-rate-empty">-</span>
    <HelpTooltip v-else class="!ml-0" width-class="w-56">
      <template #trigger>
        <span
          class="cursor-help font-medium tabular-nums"
          :class="toneClass"
          data-testid="account-cache-rate-value"
        >{{ rate.toFixed(2) }}%</span>
      </template>
      <div class="space-y-0.5" data-testid="account-cache-rate-tooltip">
        <div class="mb-1 text-gray-300">{{ t('admin.accounts.cacheRate24h.formula') }}</div>
        <div class="flex justify-between gap-3">
          <span class="text-gray-400">{{ t('admin.accounts.cacheRate24h.requests') }}</span>
          <span class="tabular-nums">{{ formatNumber(data?.requests ?? 0) }}</span>
        </div>
        <div class="flex justify-between gap-3">
          <span class="text-gray-400">{{ t('admin.accounts.cacheRate24h.inputTokens') }}</span>
          <span class="tabular-nums">{{ formatNumber(data?.input_tokens ?? 0) }}</span>
        </div>
        <div class="flex justify-between gap-3">
          <span class="text-gray-400">{{ t('admin.accounts.cacheRate24h.cacheCreationTokens') }}</span>
          <span class="tabular-nums">{{ formatNumber(data?.cache_creation_tokens ?? 0) }}</span>
        </div>
        <div class="flex justify-between gap-3">
          <span class="text-gray-400">{{ t('admin.accounts.cacheRate24h.cacheReadTokens') }}</span>
          <span class="tabular-nums">{{ formatNumber(data?.cache_read_tokens ?? 0) }}</span>
        </div>
      </div>
    </HelpTooltip>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import type { AccountCacheRate24h } from '@/api/admin/accounts'
import { formatNumber } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    data?: AccountCacheRate24h | null
    loading?: boolean
    error?: string | null
  }>(),
  {
    data: null,
    loading: false,
    error: null
  }
)

const { t } = useI18n()

const rate = computed<number | null>(() => {
  const value = props.data?.rate
  return typeof value === 'number' && Number.isFinite(value) ? value : null
})

// ≥70% 绿色，30%–70% 黄色，<30% 红色
const toneClass = computed(() => {
  const value = rate.value ?? 0
  if (value >= 70) return 'text-emerald-600 dark:text-emerald-400'
  if (value >= 30) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
})
</script>
