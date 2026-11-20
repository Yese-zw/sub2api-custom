<template>
  <BaseDialog
    :show="show"
    :title="t('admin.schedulingCenter.detail.title', { name: accountName })"
    width="extra-wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="flex justify-center py-16"><LoadingSpinner /></div>
    <div v-else-if="records.length === 0" class="py-16 text-center text-sm text-gray-400 dark:text-dark-400">
      {{ t('admin.schedulingCenter.detail.empty') }}
    </div>
    <div v-else class="space-y-4">
      <!-- 汇总 -->
      <div class="grid grid-cols-2 overflow-hidden rounded-xl border border-gray-200 sm:grid-cols-5 dark:border-dark-700">
        <div
          v-for="(item, i) in summary"
          :key="item.key"
          class="min-w-0 px-4 py-3"
          :class="[
            'border-gray-200 dark:border-dark-700',
            i > 0 ? 'sm:border-l' : '',
            i % 2 === 1 ? 'max-sm:border-l' : '',
            i >= 2 ? 'max-sm:border-t' : ''
          ]"
        >
          <div class="truncate text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</div>
          <div class="mt-1 truncate text-lg font-semibold tabular-nums" :class="item.class || 'text-gray-900 dark:text-white'">
            {{ item.value }}
          </div>
          <div v-if="item.sub" class="truncate text-[11px] text-gray-400 dark:text-dark-400">{{ item.sub }}</div>
        </div>
      </div>

      <!-- 趋势：首 Token / 总耗时，点的颜色表示智商结果 -->
      <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
        <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.schedulingCenter.detail.trend') }}</span>
          <div class="flex flex-wrap items-center gap-3 text-[11px] text-gray-500 dark:text-dark-400">
            <span v-for="lg in iqLegend" :key="lg.key" class="inline-flex items-center gap-1">
              <span class="h-2 w-2 rounded-full" :style="{ background: lg.color }"></span>{{ lg.label }}
            </span>
          </div>
        </div>
        <div class="h-52">
          <Line :data="chartData" :options="chartOptions" />
        </div>
        <p v-if="noStreamCount > 0" class="mt-2 text-[11px] text-amber-600 dark:text-amber-400">
          {{ t('admin.schedulingCenter.detail.noStreamHint', { n: noStreamCount }) }}
        </p>
      </div>

      <!-- 筛选 -->
      <div class="flex flex-wrap items-center gap-1.5">
        <button
          v-for="f in filters"
          :key="f.key"
          type="button"
          class="rounded-full px-3 py-1 text-xs transition-colors"
          :class="
            filter === f.key
              ? 'bg-primary-600 text-white'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
          "
          @click="filter = f.key"
        >
          {{ f.label }} <span class="tabular-nums opacity-70">{{ f.count }}</span>
        </button>
      </div>

      <!-- 明细 -->
      <div class="max-h-[22rem] overflow-y-auto rounded-xl border border-gray-200 dark:border-dark-700">
        <table class="w-full table-fixed text-sm">
          <colgroup>
            <col class="w-40" />
            <col class="w-24" />
            <col class="w-24" />
            <col class="w-24" />
            <col />
          </colgroup>
          <thead class="sticky top-0 z-10 bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2.5 font-medium">{{ t('admin.schedulingCenter.detail.time') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.columns.firstToken') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.detail.latency') }}</th>
              <th class="px-3 py-2.5 font-medium">{{ t('admin.schedulingCenter.columns.iq') }}</th>
              <th class="px-3 py-2.5 font-medium">{{ t('admin.schedulingCenter.detail.message') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-if="filtered.length === 0">
              <td colspan="5" class="px-3 py-8 text-center text-xs text-gray-400">{{ t('admin.schedulingCenter.detail.filterEmpty') }}</td>
            </tr>
            <tr v-for="rec in filtered" :key="rec.id" class="align-top hover:bg-gray-50/80 dark:hover:bg-dark-700/30">
              <td class="whitespace-nowrap px-3 py-2 tabular-nums text-gray-600 dark:text-gray-300">
                {{ formatDateTime(rec.started_at) }}
                <span v-if="isProbe(rec)" class="ml-1 rounded bg-sky-50 px-1 text-[10px] text-sky-700 dark:bg-sky-900/30 dark:text-sky-300">
                  {{ t('admin.schedulingCenter.detail.probeTag') }}
                </span>
              </td>
              <td class="px-3 py-2 text-right tabular-nums" :title="rec.first_token_ms == null ? t('admin.schedulingCenter.detail.noFirstToken') : undefined">
                <span v-if="rec.first_token_ms != null" class="text-gray-900 dark:text-white">{{ formatMs(rec.first_token_ms) }}</span>
                <span v-else class="text-gray-300 dark:text-dark-500">—</span>
              </td>
              <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatMs(rec.latency_ms) }}</td>
              <td class="px-3 py-2">
                <span class="badge" :class="iqBadgeClass(rec.iq_status)">
                  {{ t(`admin.schedulingCenter.iq.${iqKey(rec.iq_status)}`) }}
                </span>
              </td>
              <td class="break-words px-3 py-2 text-xs leading-relaxed text-gray-500 dark:text-dark-400">
                {{ messageOf(rec) }}
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
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  type TooltipItem
} from 'chart.js'
import { Line } from 'vue-chartjs'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import schedulingCenterAPI, { type SchedulingCenterTestRecord } from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)

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
type FilterKey = 'all' | 'passed' | 'incorrect' | 'error'
const filter = ref<FilterKey>('all')

watch(
  () => [props.show, props.accountId] as const,
  async ([show, accountId]) => {
    if (!show || accountId == null) return
    loading.value = true
    records.value = []
    filter.value = 'all'
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

// 从旧到新
const chronological = computed(() => [...records.value].reverse())

const IQ_COLORS: Record<string, string> = {
  passed: 'rgb(16, 185, 129)',
  incorrect: 'rgb(239, 68, 68)',
  error: 'rgb(245, 158, 11)',
  '': 'rgb(156, 163, 175)'
}
const iqColor = (status: string) => IQ_COLORS[status] ?? IQ_COLORS['']

const iqLegend = computed(() => [
  { key: 'passed', color: IQ_COLORS.passed, label: t('admin.schedulingCenter.iq.passed') },
  { key: 'incorrect', color: IQ_COLORS.incorrect, label: t('admin.schedulingCenter.iq.degraded') },
  { key: 'error', color: IQ_COLORS.error, label: t('admin.schedulingCenter.iq.error') }
])

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

function formatMs(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
}

const PROBE_PREFIX = '恢复探测'
function isProbe(rec: SchedulingCenterTestRecord): boolean {
  return rec.iq_message?.startsWith(PROBE_PREFIX) ?? false
}

function messageOf(rec: SchedulingCenterTestRecord): string {
  return rec.error_message || rec.iq_message || '-'
}

const noStreamCount = computed(
  () => records.value.filter((r) => r.status === 'success' && r.first_token_ms == null).length
)

// ---------- 汇总 ----------

function avg(values: number[]): number | null {
  return values.length ? values.reduce((a, b) => a + b, 0) / values.length : null
}

const summary = computed(() => {
  const recs = records.value
  const judged = recs.filter((r) => r.iq_status === 'passed' || r.iq_status === 'incorrect')
  const passed = judged.filter((r) => r.iq_status === 'passed').length
  const passRate = judged.length ? (passed / judged.length) * 100 : null
  const ttft = recs.filter((r) => r.first_token_ms != null).map((r) => r.first_token_ms as number)
  const avgTTFT = avg(ttft)
  const avgLatency = avg(recs.filter((r) => r.status === 'success').map((r) => r.latency_ms))
  const latest = recs[0]
  return [
    {
      key: 'count',
      label: t('admin.schedulingCenter.detail.count'),
      value: String(recs.length),
      sub: t('admin.schedulingCenter.detail.countSub', { n: recs.filter((r) => r.iq_status === 'error').length })
    },
    {
      key: 'pass',
      label: t('admin.schedulingCenter.detail.passRate'),
      value: passRate != null ? `${passRate.toFixed(0)}%` : '-',
      sub: judged.length ? `${passed}/${judged.length}` : undefined,
      class:
        passRate == null
          ? undefined
          : passRate >= 90
            ? 'text-emerald-600 dark:text-emerald-400'
            : passRate >= 70
              ? 'text-amber-600 dark:text-amber-400'
              : 'text-red-600 dark:text-red-400'
    },
    {
      key: 'ttft',
      label: t('admin.schedulingCenter.detail.avgFirstToken'),
      value: avgTTFT != null ? formatMs(avgTTFT) : '-',
      sub: ttft.length ? t('admin.schedulingCenter.detail.samples', { n: ttft.length }) : undefined
    },
    {
      key: 'latency',
      label: t('admin.schedulingCenter.detail.avgLatency'),
      value: avgLatency != null ? formatMs(avgLatency) : '-'
    },
    {
      key: 'latest',
      label: t('admin.schedulingCenter.detail.latest'),
      value: latest ? t(`admin.schedulingCenter.iq.${iqKey(latest.iq_status)}`) : '-',
      sub: latest ? formatDateTime(latest.started_at) : undefined
    }
  ]
})

// ---------- 筛选 ----------

const filters = computed(() => {
  const count = (k: FilterKey) => (k === 'all' ? records.value.length : records.value.filter((r) => r.iq_status === k).length)
  return [
    { key: 'all' as const, label: t('admin.schedulingCenter.detail.filterAll'), count: count('all') },
    { key: 'passed' as const, label: t('admin.schedulingCenter.iq.passed'), count: count('passed') },
    { key: 'incorrect' as const, label: t('admin.schedulingCenter.iq.degraded'), count: count('incorrect') },
    { key: 'error' as const, label: t('admin.schedulingCenter.iq.error'), count: count('error') }
  ]
})

const filtered = computed(() =>
  filter.value === 'all' ? records.value : records.value.filter((r) => r.iq_status === filter.value)
)

// ---------- 图表 ----------

function shortTime(iso: string, multiDay: boolean): string {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  return multiDay ? `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${hm}` : hm
}

const chartData = computed(() => {
  const recs = chronological.value
  const first = recs[0]
  const last = recs[recs.length - 1]
  const multiDay = !!first && !!last && new Date(last.started_at).getTime() - new Date(first.started_at).getTime() > 20 * 3600 * 1000
  return {
    labels: recs.map((r) => shortTime(r.started_at, multiDay)),
    datasets: [
      {
        label: t('admin.schedulingCenter.columns.firstToken'),
        data: recs.map((r) => r.first_token_ms),
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        pointBackgroundColor: recs.map((r) => iqColor(r.iq_status)),
        pointBorderColor: recs.map((r) => iqColor(r.iq_status)),
        pointRadius: 3,
        pointHoverRadius: 5,
        borderWidth: 2,
        tension: 0.25,
        spanGaps: true
      },
      {
        label: t('admin.schedulingCenter.detail.latency'),
        data: recs.map((r) => r.latency_ms),
        borderColor: 'rgba(148, 163, 184, 0.9)',
        backgroundColor: 'rgba(148, 163, 184, 0.1)',
        borderDash: [4, 4],
        pointRadius: 0,
        pointHoverRadius: 3,
        borderWidth: 1.5,
        tension: 0.25,
        spanGaps: true
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: false as const,
  interaction: { mode: 'index' as const, intersect: false },
  scales: {
    x: { ticks: { maxTicksLimit: 8, autoSkip: true, maxRotation: 0 }, grid: { display: false } },
    y: {
      beginAtZero: true,
      ticks: { callback: (v: string | number) => formatMs(Number(v)) }
    }
  },
  plugins: {
    legend: { position: 'top' as const, align: 'start' as const, labels: { boxWidth: 12, boxHeight: 2 } },
    tooltip: {
      callbacks: {
        title: (items: TooltipItem<'line'>[]) => {
          const rec = chronological.value[items[0]?.dataIndex ?? 0]
          return rec ? formatDateTime(rec.started_at) : ''
        },
        label: (item: TooltipItem<'line'>) =>
          `${item.dataset.label}: ${item.parsed.y == null ? '—' : formatMs(item.parsed.y)}`,
        afterBody: (items: TooltipItem<'line'>[]) => {
          const rec = chronological.value[items[0]?.dataIndex ?? 0]
          return rec ? `${t('admin.schedulingCenter.columns.iq')}: ${t(`admin.schedulingCenter.iq.${iqKey(rec.iq_status)}`)}` : ''
        }
      }
    }
  }
}))
</script>
