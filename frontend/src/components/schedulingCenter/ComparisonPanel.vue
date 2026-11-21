<template>
  <div class="flex min-h-0 flex-1 flex-col overflow-auto">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
      <div class="inline-flex rounded-lg bg-gray-100 p-0.5 dark:bg-dark-900">
        <button
          v-for="opt in rangeOptions"
          :key="opt.hours"
          type="button"
          class="rounded-md px-3 py-1 text-xs font-medium transition-colors"
          :class="
            hours === opt.hours
              ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
              : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'
          "
          @click="setHours(opt.hours)"
        >
          {{ opt.label }}
        </button>
      </div>
      <p class="max-w-2xl text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.compare.note') }}</p>
    </div>

    <div v-if="loading && !data" class="flex flex-1 items-center justify-center py-16"><LoadingSpinner /></div>
    <div v-else-if="data" class="space-y-5 p-4" :class="{ 'opacity-60': loading }">
      <!-- 当前快照 -->
      <div class="flex flex-wrap gap-2 text-xs">
        <span class="rounded-md bg-emerald-50 px-2 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
          {{ t('admin.schedulingCenter.compare.poolNow', { n: data.snapshot.pool_accounts }) }}
        </span>
        <span class="rounded-md bg-gray-100 px-2 py-1 text-gray-600 dark:bg-dark-700 dark:text-gray-300">
          {{ t('admin.schedulingCenter.compare.accountPoolNow', { n: data.snapshot.account_pool_accounts }) }}
        </span>
        <span
          v-for="(n, state) in data.snapshot.states"
          :key="state"
          class="rounded-md bg-sky-50 px-2 py-1 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300"
        >
          {{ t(`admin.schedulingCenter.compare.state.${state}`) }} {{ n }}
        </span>
      </div>

      <p v-if="lowSample" class="rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.schedulingCenter.compare.lowSample', { n: MIN_SAMPLE }) }}
      </p>

      <!-- 指标对照 -->
      <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
        <table class="min-w-full text-sm">
          <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-900 dark:text-gray-400">
            <tr>
              <th class="px-4 py-2.5 font-medium">{{ t('admin.schedulingCenter.compare.metric') }}</th>
              <th class="px-4 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.schedulingPool') }}</th>
              <th class="px-4 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.accountPool') }}</th>
              <th class="px-4 py-2.5 text-right font-medium">{{ t('admin.schedulingCenter.compare.diff') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="row in rows" :key="row.key">
              <td class="px-4 py-2.5 text-gray-700 dark:text-gray-300">
                {{ row.label }}
                <span v-if="row.hint" class="ml-1 text-xs text-gray-400">{{ row.hint }}</span>
              </td>
              <td class="px-4 py-2.5 text-right tabular-nums font-medium text-gray-900 dark:text-white">{{ row.pool }}</td>
              <td class="px-4 py-2.5 text-right tabular-nums text-gray-700 dark:text-gray-300">{{ row.account }}</td>
              <td class="px-4 py-2.5 text-right">
                <span v-if="row.verdict" class="inline-flex rounded px-1.5 py-0.5 text-xs tabular-nums" :class="verdictClass[row.verdict]">
                  {{ row.diff }}
                </span>
                <span v-else class="text-xs text-gray-300 dark:text-gray-600">{{ row.diff || '—' }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 趋势 -->
      <div class="grid gap-4 xl:grid-cols-2">
        <div v-for="chart in charts" :key="chart.key" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-300">{{ chart.title }}</div>
          <div class="h-56">
            <Line :data="chart.data" :options="chart.options" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import schedulingCenterAPI, {
  type SchedulingCenterComparison,
  type SchedulingCenterSideMetrics
} from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const props = defineProps<{ groupId: number }>()

const { t } = useI18n()
const appStore = useAppStore()

const MIN_SAMPLE = 100
const hours = ref(24)
const loading = ref(false)
const data = ref<SchedulingCenterComparison | null>(null)
let seq = 0

const rangeOptions = computed(() => [
  { hours: 24, label: t('admin.schedulingCenter.compare.range24h') },
  { hours: 72, label: t('admin.schedulingCenter.compare.range3d') },
  { hours: 168, label: t('admin.schedulingCenter.compare.range7d') }
])

async function load() {
  const current = ++seq
  loading.value = true
  try {
    const res = await schedulingCenterAPI.getComparison(props.groupId, hours.value)
    if (current === seq) data.value = res
  } catch (error) {
    if (current === seq) appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.compare.loadFailed')))
  } finally {
    if (current === seq) loading.value = false
  }
}

function setHours(h: number) {
  if (hours.value === h) return
  hours.value = h
  load()
}

watch(() => props.groupId, () => {
  data.value = null
  load()
})
onMounted(load)
defineExpose({ reload: load })

// ---------- 指标对照 ----------

type Verdict = 'better' | 'worse' | 'same'
const verdictClass: Record<Verdict, string> = {
  better: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
  worse: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300',
  same: 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'
}

const lowSample = computed(() => {
  const d = data.value
  if (!d) return false
  return d.pool.requests + d.pool.errors < MIN_SAMPLE || d.account_pool.requests + d.account_pool.errors < MIN_SAMPLE
})

function fmtNum(v: number | null | undefined): string {
  return v == null ? '-' : v.toLocaleString()
}
function fmtPct(v: number | null | undefined): string {
  return v == null ? '-' : `${v.toFixed(2)}%`
}
function fmtMs(v: number | null | undefined): string {
  if (v == null) return '-'
  return `${Math.round(v)}ms`
}

interface Row {
  key: string
  label: string
  hint?: string
  pool: string
  account: string
  diff?: string
  verdict?: Verdict
}

// higherIsBetter: true 时调度池更高为好；unit: pct 用百分点差，ms 用毫秒差
function compare(
  pool: number | null,
  account: number | null,
  higherIsBetter: boolean,
  unit: 'pct' | 'ms'
): { diff?: string; verdict?: Verdict } {
  if (pool == null || account == null) return {}
  const delta = pool - account
  const threshold = unit === 'pct' ? 0.5 : 50
  const verdict: Verdict = Math.abs(delta) < threshold ? 'same' : (delta > 0) === higherIsBetter ? 'better' : 'worse'
  const sign = delta > 0 ? '+' : delta < 0 ? '−' : ''
  const abs = Math.abs(delta)
  const diff = unit === 'pct' ? `${sign}${abs.toFixed(2)} ${t('admin.schedulingCenter.compare.points')}` : `${sign}${fmtMs(abs)}`
  return { diff, verdict }
}

const rows = computed<Row[]>(() => {
  const d = data.value
  if (!d) return []
  const p: SchedulingCenterSideMetrics = d.pool
  const a: SchedulingCenterSideMetrics = d.account_pool
  const label = (k: string) => t(`admin.schedulingCenter.compare.metrics.${k}`)
  return [
    { key: 'accounts', label: label('accounts'), pool: fmtNum(p.accounts), account: fmtNum(a.accounts) },
    { key: 'requests', label: label('requests'), pool: fmtNum(p.requests), account: fmtNum(a.requests) },
    { key: 'errors', label: label('errors'), pool: fmtNum(p.errors), account: fmtNum(a.errors) },
    { key: 'error_rate', label: label('errorRate'), pool: fmtPct(p.error_rate), account: fmtPct(a.error_rate), ...compare(p.error_rate, a.error_rate, false, 'pct') },
    { key: 'cache_rate', label: label('cacheRate'), pool: fmtPct(p.cache_rate), account: fmtPct(a.cache_rate), ...compare(p.cache_rate, a.cache_rate, true, 'pct') },
    { key: 'ttft', label: label('avgFirstToken'), pool: fmtMs(p.avg_first_token_ms), account: fmtMs(a.avg_first_token_ms), ...compare(p.avg_first_token_ms, a.avg_first_token_ms, false, 'ms') },
    {
      key: 'slow',
      label: label('slowRate'),
      hint: `> ${fmtMs(d.slow_threshold_ms)}`,
      pool: fmtPct(p.slow_rate),
      account: fmtPct(a.slow_rate),
      ...compare(p.slow_rate, a.slow_rate, false, 'pct')
    },
    { key: 'iq_checks', label: label('iqChecks'), pool: fmtNum(p.iq_checks), account: fmtNum(a.iq_checks) },
    { key: 'iq_pass', label: label('iqPassRate'), pool: fmtPct(p.iq_pass_rate), account: fmtPct(a.iq_pass_rate), ...compare(p.iq_pass_rate, a.iq_pass_rate, true, 'pct') },
    { key: 'iq_ttft', label: label('iqFirstToken'), pool: fmtMs(p.iq_avg_first_token_ms), account: fmtMs(a.iq_avg_first_token_ms), ...compare(p.iq_avg_first_token_ms, a.iq_avg_first_token_ms, false, 'ms') }
  ]
})

// ---------- 趋势 ----------

const POOL_COLOR = 'rgb(16, 185, 129)'
const ACCOUNT_COLOR = 'rgb(148, 163, 184)'

function hourLabel(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return hours.value > 24 ? `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:00` : `${pad(d.getHours())}:00`
}

function lineChart(title: string, pick: (m: SchedulingCenterSideMetrics) => number | null, unit: '%' | 'ms') {
  const series = data.value?.series ?? []
  return {
    key: title,
    title,
    data: {
      labels: series.map((p) => hourLabel(p.time)),
      datasets: [
        {
          label: t('admin.schedulingCenter.schedulingPool'),
          data: series.map((p) => pick(p.pool)),
          borderColor: POOL_COLOR,
          backgroundColor: 'rgba(16, 185, 129, 0.08)',
          fill: true,
          tension: 0.3,
          pointRadius: 0,
          spanGaps: true
        },
        {
          label: t('admin.schedulingCenter.accountPool'),
          data: series.map((p) => pick(p.account_pool)),
          borderColor: ACCOUNT_COLOR,
          backgroundColor: 'rgba(148, 163, 184, 0.08)',
          fill: false,
          tension: 0.3,
          pointRadius: 0,
          spanGaps: true
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index' as const, intersect: false },
      scales: {
        x: { ticks: { maxTicksLimit: 8, autoSkip: true } },
        y: { beginAtZero: true, ticks: { callback: (v: string | number) => `${v}${unit}` } }
      },
      plugins: { legend: { position: 'top' as const, labels: { boxWidth: 12 } } }
    }
  }
}

const charts = computed(() => {
  if (!data.value) return []
  return [
    lineChart(t('admin.schedulingCenter.compare.chartErrorRate'), (m) => m.error_rate, '%'),
    lineChart(t('admin.schedulingCenter.compare.chartFirstToken'), (m) => m.avg_first_token_ms, 'ms'),
    lineChart(t('admin.schedulingCenter.compare.chartCacheRate'), (m) => m.cache_rate, '%'),
    lineChart(t('admin.schedulingCenter.compare.chartSlowRate'), (m) => m.slow_rate, '%')
  ]
})
</script>
