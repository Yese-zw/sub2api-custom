<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="card flex flex-col gap-3 p-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.operationsCenter.ranking.timeRange') }}:</span>
        <button
          v-for="p in presets"
          :key="p"
          type="button"
          class="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors"
          :class="preset === p ? activeBtn : idleBtn"
          @click="selectPreset(p)"
        >
          {{ t(`admin.operationsCenter.ranking.presets.${p}`) }}
        </button>
        <DateRangePicker v-if="preset === 'custom'" v-model:start-date="startDate" v-model:end-date="endDate" @change="load" />
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex overflow-hidden rounded-lg border border-gray-200 text-xs dark:border-dark-600">
          <button
            v-for="m in metrics"
            :key="m"
            type="button"
            class="px-3 py-1.5 font-medium transition-colors"
            :class="metric === m ? 'bg-primary-500 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
            @click="metric = m"
          >
            {{ t(`admin.operationsCenter.trend.metrics.${m}`) }}
          </button>
        </div>
        <select v-model.number="top" class="input w-auto py-1.5 text-sm" @change="load">
          <option v-for="n in [5, 8, 12, 20]" :key="n" :value="n">Top {{ n }}</option>
        </select>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
          <Icon name="refresh" size="sm" class="mr-1" :class="{ 'animate-spin': loading }" />
          {{ t('admin.operationsCenter.refresh') }}
        </button>
      </div>
    </div>

    <!-- 汇总 -->
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div v-for="card in summaryCards" :key="card.label" class="card p-4">
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ card.label }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ card.value }}</p>
        <p v-if="card.sub" class="mt-0.5 text-[11px] text-gray-400 dark:text-dark-400">{{ card.sub }}</p>
      </div>
    </div>

    <!-- 图表 -->
    <div class="card p-4">
      <div class="mb-3 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.operationsCenter.trend.chartTitle', { metric: t(`admin.operationsCenter.trend.metrics.${metric}`) }) }}
        </h3>
        <span v-if="data" class="text-[11px] text-gray-400 dark:text-dark-400">
          {{ t(`admin.operationsCenter.trend.granularity.${data.granularity}`) }}
        </span>
      </div>
      <div v-if="loading && !data" class="flex h-80 items-center justify-center">
        <LoadingSpinner />
      </div>
      <div v-else-if="!data || data.series.length === 0" class="flex h-80 items-center justify-center">
        <EmptyState :title="t('admin.operationsCenter.ranking.empty')" />
      </div>
      <div v-else class="h-80">
        <Line :data="chartData" :options="chartOptions" />
      </div>
    </div>

    <!-- 星期 × 小时热力图 -->
    <div v-if="heatmap.length > 0" class="card p-4">
      <div class="mb-3 flex items-center justify-between">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.operationsCenter.trend.heatmapTitle', { metric: t(`admin.operationsCenter.trend.metrics.${heatMetric}`) }) }}
        </h3>
        <span class="text-[11px] text-gray-400 dark:text-dark-400">{{ t('admin.operationsCenter.trend.heatmapHint') }}</span>
      </div>
      <div class="overflow-x-auto">
        <div class="min-w-[640px]">
          <div class="ml-10 grid gap-0.5 text-[10px] text-gray-400 dark:text-dark-400" style="grid-template-columns: repeat(24, minmax(0, 1fr))">
            <span v-for="h in 24" :key="h" class="text-center">{{ (h - 1) % 3 === 0 ? h - 1 : '' }}</span>
          </div>
          <div v-for="d in 7" :key="d" class="mt-0.5 flex items-center">
            <span class="w-10 flex-shrink-0 text-[11px] text-gray-500 dark:text-dark-400">{{ weekdayLabel(d) }}</span>
            <div class="grid flex-1 gap-0.5" style="grid-template-columns: repeat(24, minmax(0, 1fr))">
              <div
                v-for="h in 24"
                :key="h"
                class="h-6 rounded-sm"
                :style="{ backgroundColor: heatColor(heatValue(d, h - 1)) }"
                :title="heatTitle(d, h - 1)"
              />
            </div>
          </div>
          <div class="mt-2 flex items-center justify-end gap-1 text-[10px] text-gray-400 dark:text-dark-400">
            <span>{{ t('admin.operationsCenter.trend.less') }}</span>
            <span v-for="i in 5" :key="i" class="h-3 w-3 rounded-sm" :style="{ backgroundColor: heatColor((heatMax * (i - 1)) / 4) }" />
            <span>{{ t('admin.operationsCenter.trend.more') }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 站点占比表 -->
    <div v-if="data && data.series.length > 0" class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[720px] text-sm">
          <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-3 py-2.5 text-left font-medium">{{ t('admin.operationsCenter.ranking.cols.site') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.requests') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.trend.share') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.accountCost') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.errors') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.failureRate') }}</th>
              <th class="px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.trend.peak') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="(sr, idx) in data.series" :key="sr.key" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
              <td class="px-3 py-2.5">
                <div class="flex items-center gap-2">
                  <span class="h-2.5 w-2.5 flex-shrink-0 rounded-full" :style="{ backgroundColor: colorAt(idx, sr.key) }" />
                  <a
                    v-if="sr.site_url"
                    :href="sr.site_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="truncate font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
                  >
                    {{ seriesLabel(sr) }}
                  </a>
                  <span v-else class="truncate font-medium text-gray-900 dark:text-white">{{ seriesLabel(sr) }}</span>
                  <span class="flex-shrink-0 text-[11px] text-gray-400">
                    {{ t('admin.operationsCenter.card.accounts', { n: sr.account_count }) }}
                  </span>
                </div>
              </td>
              <td class="px-3 py-2.5 text-right font-semibold text-gray-900 dark:text-white">{{ formatInt(sr.total_requests) }}</td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatPercent(share(sr.total_requests)) }}</td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatMoney(sr.total_account_cost) }}</td>
              <td class="px-3 py-2.5 text-right" :class="sr.total_errors > 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-400'">
                {{ formatInt(sr.total_errors) }}
              </td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">
                {{ formatPercent(failureRate(sr.total_requests, sr.total_errors)) }}
              </td>
              <td class="whitespace-nowrap px-3 py-2.5 text-right text-xs text-gray-500 dark:text-dark-400">{{ peakLabel(sr) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import operationsCenterAPI, {
  type HeatmapCell,
  type SiteTrendResponse,
  type SiteTrendSeries,
  type SiteTrendParams,
} from '@/api/admin/operationsCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

type Preset = '24h' | '7d' | '30d' | 'custom'
type Metric = 'requests' | 'account_cost' | 'tokens' | 'errors'

const { t, locale } = useI18n()
const appStore = useAppStore()

const activeBtn = 'bg-primary-500 text-white'
const idleBtn = 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
const presets: Preset[] = ['24h', '7d', '30d', 'custom']
const PRESET_HOURS: Record<Exclude<Preset, 'custom'>, number> = { '24h': 24, '7d': 24 * 7, '30d': 24 * 30 }
const metrics: Metric[] = ['requests', 'account_cost', 'tokens', 'errors']
const PALETTE = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#06b6d4', '#8b5cf6', '#ec4899', '#84cc16', '#f97316', '#14b8a6',
  '#3b82f6', '#a855f7', '#eab308', '#22c55e', '#e11d48', '#0ea5e9', '#d946ef', '#65a30d', '#fb7185', '#2dd4bf']
const OTHER_KEY = '__other__'
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'

const preset = ref<Preset>('7d')
const startDate = ref(formatLocalDate(new Date(Date.now() - 6 * 86400_000)))
const endDate = ref(formatLocalDate(new Date()))
const metric = ref<Metric>('requests')
const top = ref(8)
const loading = ref(false)
const data = ref<SiteTrendResponse | null>(null)
const heatmap = ref<HeatmapCell[]>([])

type HeatMetric = 'requests' | 'account_cost' | 'errors'
/** 热力图没有 Token 维度，选 Token 时回退到请求数 */
const heatMetric = computed<HeatMetric>(() => (metric.value === 'tokens' ? 'requests' : metric.value))
const heatIndex = computed(() => {
  const m = new Map<string, HeatmapCell>()
  for (const c of heatmap.value) m.set(`${c.weekday}-${c.hour}`, c)
  return m
})
const heatMax = computed(() => Math.max(0, ...heatmap.value.map((c) => c[heatMetric.value])))

function heatValue(weekday: number, hour: number): number {
  return heatIndex.value.get(`${weekday}-${hour}`)?.[heatMetric.value] ?? 0
}

function heatColor(v: number): string {
  if (v <= 0 || heatMax.value <= 0) return isDark.value ? 'rgba(55,65,81,0.6)' : '#f3f4f6'
  const ratio = Math.min(1, v / heatMax.value)
  const base = heatMetric.value === 'errors' ? '239,68,68' : '99,102,241'
  return `rgba(${base},${(0.15 + ratio * 0.85).toFixed(3)})`
}

function weekdayLabel(weekday: number): string {
  // 2024-01-01 是周一
  return new Date(2024, 0, weekday).toLocaleDateString(locale.value, { weekday: 'short' })
}

function heatTitle(weekday: number, hour: number): string {
  const c = heatIndex.value.get(`${weekday}-${hour}`)
  const hh = String(hour).padStart(2, '0')
  return `${weekdayLabel(weekday)} ${hh}:00-${hh}:59\n${t('admin.operationsCenter.trend.metrics.requests')}: ${formatInt(c?.requests ?? 0)}\n${t('admin.operationsCenter.trend.metrics.account_cost')}: ${formatMoney(c?.account_cost ?? 0)}\n${t('admin.operationsCenter.trend.metrics.errors')}: ${formatInt(c?.errors ?? 0)}`
}

const isDark = computed(() => document.documentElement.classList.contains('dark'))

function colorAt(idx: number, key: string): string {
  return key === OTHER_KEY ? '#9ca3af' : PALETTE[idx % PALETTE.length]
}

function seriesLabel(sr: SiteTrendSeries): string {
  return sr.key === OTHER_KEY ? t('admin.operationsCenter.trend.other') : sr.label || sr.key
}

function bucketLabel(bucket: string): string {
  // bucket 为后端按所选时区截断后的本地时间 "YYYY-MM-DDTHH:mm:ss"
  const [date, time] = bucket.split('T')
  if (data.value?.granularity === 'day') return date.slice(5)
  return `${date.slice(5)} ${time.slice(0, 5)}`
}

function seriesValues(sr: SiteTrendSeries): number[] {
  return sr[metric.value] as number[]
}

const chartData = computed(() => {
  const d = data.value
  if (!d) return { labels: [], datasets: [] }
  return {
    labels: d.buckets.map(bucketLabel),
    datasets: d.series.map((sr, idx) => {
      const color = colorAt(idx, sr.key)
      return {
        label: seriesLabel(sr),
        data: seriesValues(sr),
        borderColor: color,
        backgroundColor: `${color}22`,
        borderWidth: 2,
        pointRadius: d.buckets.length > 48 ? 0 : 2,
        pointHoverRadius: 4,
        tension: 0.3,
        fill: false,
      }
    }),
  }
})

const chartOptions = computed<ChartOptions<'line'>>(() => {
  const text = isDark.value ? '#e5e7eb' : '#374151'
  const grid = isDark.value ? '#374151' : '#e5e7eb'
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index', intersect: false },
    plugins: {
      legend: { position: 'bottom', labels: { color: text, boxWidth: 10, boxHeight: 10, usePointStyle: true } },
      tooltip: {
        itemSort: (a, b) => (b.parsed.y ?? 0) - (a.parsed.y ?? 0),
        callbacks: {
          label: (ctx) => `${ctx.dataset.label}: ${formatMetric(ctx.parsed.y ?? 0)}`,
        },
      },
    },
    scales: {
      x: { ticks: { color: text, maxRotation: 0, autoSkip: true, maxTicksLimit: 12 }, grid: { color: grid } },
      y: { beginAtZero: true, ticks: { color: text, callback: (v) => formatMetric(Number(v)) }, grid: { color: grid } },
    },
  }
})

const summaryCards = computed(() => {
  const d = data.value
  if (!d) return []
  const total = d.total
  const peakIdx = argMax(seriesValues(total))
  const buckets = Math.max(1, d.buckets.length)
  return [
    {
      label: t('admin.operationsCenter.ranking.summary.requests'),
      value: formatInt(total.total_requests),
      sub: t(`admin.operationsCenter.trend.avgPer.${d.granularity}`, { n: formatInt(total.total_requests / buckets) }),
    },
    {
      label: t('admin.operationsCenter.ranking.summary.accountCost'),
      value: formatMoney(total.total_account_cost),
      sub: t(`admin.operationsCenter.trend.avgPer.${d.granularity}`, { n: formatMoney(total.total_account_cost / buckets) }),
    },
    {
      label: t('admin.operationsCenter.ranking.summary.errors'),
      value: formatInt(total.total_errors),
      sub: formatPercent(failureRate(total.total_requests, total.total_errors)),
    },
    {
      label: t('admin.operationsCenter.trend.peakOf', { metric: t(`admin.operationsCenter.trend.metrics.${metric.value}`) }),
      value: peakIdx >= 0 ? formatMetric(seriesValues(total)[peakIdx]) : '-',
      sub: peakIdx >= 0 ? bucketLabel(d.buckets[peakIdx]) : '',
    },
  ]
})

function argMax(values: number[]): number {
  let idx = -1
  let max = 0
  values.forEach((v, i) => {
    if (v > max) {
      max = v
      idx = i
    }
  })
  return idx
}

function peakLabel(sr: SiteTrendSeries): string {
  const vals = seriesValues(sr)
  const idx = argMax(vals)
  if (idx < 0 || !data.value) return '-'
  return `${bucketLabel(data.value.buckets[idx])} · ${formatMetric(vals[idx])}`
}

function share(requests: number): number {
  const total = data.value?.total.total_requests ?? 0
  return total > 0 ? requests / total : 0
}

function failureRate(requests: number, errors: number): number {
  const total = requests + errors
  return total > 0 ? errors / total : 0
}

function formatMetric(v: number): string {
  if (metric.value === 'account_cost') return formatMoney(v)
  if (metric.value === 'tokens') return formatCompact(v)
  return formatInt(v)
}

function formatLocalDate(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function formatInt(v: number): string {
  return Math.round(v || 0).toLocaleString(locale.value)
}

function formatCompact(v: number): string {
  return new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 2 }).format(v || 0)
}

function formatMoney(v: number): string {
  const abs = Math.abs(v || 0)
  const digits = abs >= 100 ? 2 : abs >= 1 ? 3 : 4
  return `$${(v || 0).toLocaleString(locale.value, { maximumFractionDigits: digits })}`
}

function formatPercent(v: number): string {
  return `${(v * 100).toFixed(v >= 0.1 ? 0 : 1)}%`
}

function buildParams(): SiteTrendParams {
  if (preset.value === 'custom') {
    return { start_date: startDate.value, end_date: endDate.value, timezone, top: top.value }
  }
  const end = new Date()
  const start = new Date(end.getTime() - PRESET_HOURS[preset.value] * 3600_000)
  const iso = (d: Date) => d.toISOString().replace(/\.\d{3}Z$/, 'Z')
  return { start_time: iso(start), end_time: iso(end), timezone, top: top.value }
}

function selectPreset(p: Preset) {
  preset.value = p
  if (p !== 'custom') load()
}

async function load() {
  loading.value = true
  try {
    const params = buildParams()
    const { top: _top, ...heatParams } = params
    // 热力图为附加信息，失败时不影响趋势图
    const [trend, cells] = await Promise.all([
      operationsCenterAPI.getSiteTrend(params),
      operationsCenterAPI.getHeatmap(heatParams).catch(() => [] as HeatmapCell[]),
    ])
    data.value = trend
    heatmap.value = cells
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.operationsCenter.trend.loadFailed')))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
