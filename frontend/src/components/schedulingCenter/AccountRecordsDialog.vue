<template>
  <BaseDialog
    :show="show"
    :title="t('admin.schedulingCenter.detail.title', { name: accountName })"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="flex justify-center py-12"><LoadingSpinner /></div>
    <div v-else-if="records.length === 0" class="py-12 text-center text-sm text-gray-400">
      {{ t('admin.schedulingCenter.detail.empty') }}
    </div>
    <div v-else class="space-y-5">
      <!-- 汇总 -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div v-for="item in summary" :key="item.label" class="rounded-xl border border-gray-100 bg-gray-50/70 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ item.label }}</div>
          <div class="mt-1 text-xl font-bold tabular-nums text-gray-900 dark:text-white">{{ item.value }}</div>
        </div>
      </div>

      <!-- 首 Token 趋势（从旧到新） -->
      <div v-if="ttftSeries.values.length > 1">
        <div class="mb-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.detail.ttftTrend') }}</div>
        <RxSparkline :values="ttftSeries.values" :labels="ttftSeries.labels" :height="56" :format="formatMs" />
      </div>

      <!-- 智商状态条（从旧到新） -->
      <div>
        <div class="mb-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.detail.iqTrend') }}</div>
        <div class="flex h-5 gap-px overflow-hidden rounded-lg">
          <div
            v-for="rec in chronological"
            :key="rec.id"
            class="flex-1"
            :class="iqBarClass(rec.iq_status)"
            :title="`${formatDateTime(rec.started_at)} · ${t(`admin.schedulingCenter.iq.${iqKey(rec.iq_status)}`)}`"
          ></div>
        </div>
      </div>

      <!-- 明细 -->
      <div class="max-h-[22rem] overflow-auto rounded-xl border border-gray-200 dark:border-dark-700">
        <table class="min-w-full whitespace-nowrap text-sm">
          <thead class="sticky top-0 bg-gray-50/95 text-left text-xs uppercase tracking-wider text-gray-500 backdrop-blur-sm dark:bg-dark-800/95 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2.5 font-medium">{{ t('admin.schedulingCenter.detail.time') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.columns.firstToken') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.detail.latency') }}</th>
              <th class="px-3 py-2.5 font-medium">{{ t('admin.schedulingCenter.columns.iq') }}</th>
              <th class="px-3 py-2.5 font-medium">{{ t('admin.schedulingCenter.detail.message') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="rec in records" :key="rec.id" class="transition-colors hover:bg-gray-50/80 dark:hover:bg-dark-700/30">
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ formatDateTime(rec.started_at) }}</td>
              <td class="px-3 py-2 text-right tabular-nums text-gray-900 dark:text-white">
                {{ rec.first_token_ms != null ? formatMs(rec.first_token_ms) : '-' }}
              </td>
              <td class="px-3 py-2 text-right tabular-nums text-gray-500">{{ formatMs(rec.latency_ms) }}</td>
              <td class="px-3 py-2">
                <span class="badge" :class="iqBadgeClass(rec.iq_status)">
                  {{ t(`admin.schedulingCenter.iq.${iqKey(rec.iq_status)}`) }}
                </span>
              </td>
              <td class="max-w-[20rem] truncate px-3 py-2 text-xs text-gray-500" :title="rec.error_message || rec.iq_message">
                {{ rec.error_message || rec.iq_message || '-' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import RxSparkline from '@/components/common/RxSparkline.vue'
import schedulingCenterAPI, { type SchedulingCenterTestRecord } from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  show: boolean
  accountId: number | null
  accountName: string
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const records = ref<SchedulingCenterTestRecord[]>([])

watch(
  () => [props.show, props.accountId] as const,
  async ([show, accountId]) => {
    if (!show || accountId == null) return
    loading.value = true
    records.value = []
    try {
      const list = await schedulingCenterAPI.getAccountRecords(accountId, 100)
      if (accountId === props.accountId) records.value = list
    } catch (error) {
      appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.detail.loadFailed')))
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

const chronological = computed(() => [...records.value].reverse())

const ttftSeries = computed(() => {
  const points = chronological.value.filter((r) => r.first_token_ms != null)
  return {
    values: points.map((r) => r.first_token_ms as number),
    labels: points.map((r) => formatDateTime(r.started_at))
  }
})

const summary = computed(() => {
  const total = records.value.length
  const judged = records.value.filter((r) => r.iq_status === 'passed' || r.iq_status === 'incorrect')
  const passed = judged.filter((r) => r.iq_status === 'passed').length
  const ttft = ttftSeries.value.values
  const avg = ttft.length ? ttft.reduce((a, b) => a + b, 0) / ttft.length : null
  const latest = records.value[0]
  return [
    { label: t('admin.schedulingCenter.detail.count'), value: String(total) },
    {
      label: t('admin.schedulingCenter.detail.passRate'),
      value: judged.length ? `${((passed / judged.length) * 100).toFixed(0)}%` : '-'
    },
    { label: t('admin.schedulingCenter.detail.avgFirstToken'), value: avg != null ? formatMs(avg) : '-' },
    {
      label: t('admin.schedulingCenter.detail.latest'),
      value: latest ? t(`admin.schedulingCenter.iq.${iqKey(latest.iq_status)}`) : '-'
    }
  ]
})

function iqKey(status: string): string {
  switch (status) {
    case 'passed':
      return 'passed'
    case 'incorrect':
      return 'degraded'
    case 'error':
      return 'error'
    default:
      return 'unchecked'
  }
}

function iqBadgeClass(status: string): string {
  switch (status) {
    case 'passed':
      return 'badge-success'
    case 'incorrect':
      return 'badge-danger'
    case 'error':
      return 'badge-warning'
    default:
      return 'badge-gray'
  }
}

function iqBarClass(status: string): string {
  switch (status) {
    case 'passed':
      return 'bg-emerald-500'
    case 'incorrect':
      return 'bg-red-500'
    case 'error':
      return 'bg-amber-400'
    default:
      return 'bg-gray-300 dark:bg-dark-600'
  }
}

function formatMs(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
}
</script>
