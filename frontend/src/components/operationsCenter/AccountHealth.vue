<template>
  <div class="space-y-4">
    <!-- 工具栏 -->
    <div class="card flex items-center justify-between gap-3 p-4">
      <p class="text-xs text-gray-500 dark:text-dark-400">
        <template v-if="data">{{ t('admin.operationsCenter.health.generatedAt', { time: formatDateTime(data.generated_at) }) }}</template>
      </p>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
        <Icon name="refresh" size="sm" class="mr-1" :class="{ 'animate-spin': loading }" />
        {{ t('admin.operationsCenter.refresh') }}
      </button>
    </div>

    <div v-if="loading && !data" class="flex justify-center py-16">
      <LoadingSpinner />
    </div>

    <template v-else-if="data">
      <!-- 汇总 -->
      <div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        <div v-for="card in summaryCards" :key="card.label" class="card p-4">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ card.label }}</p>
          <p class="mt-1 text-2xl font-semibold" :class="card.cls">{{ card.value }}</p>
          <p v-if="card.sub" class="mt-0.5 text-[11px] text-gray-400 dark:text-dark-400">{{ card.sub }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <!-- 平台分布 -->
        <div class="card overflow-hidden">
          <h3 class="border-b border-gray-100 px-4 py-3 text-sm font-semibold text-gray-900 dark:border-dark-700 dark:text-white">
            {{ t('admin.operationsCenter.health.platforms') }}
          </h3>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="p in data.platforms" :key="p.platform" class="px-4 py-2.5">
              <div class="flex items-center justify-between text-sm">
                <span class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
                  <PlatformIcon :platform="p.platform as any" size="sm" />
                  {{ p.platform || '-' }}
                </span>
                <span class="text-xs text-gray-500 dark:text-dark-400">
                  {{ p.schedulable }} / {{ p.total }}
                </span>
              </div>
              <div class="mt-1.5 flex h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="bg-emerald-500" :style="{ width: pct(p.schedulable - p.idle, p.total) }" />
                <div class="bg-gray-400" :style="{ width: pct(p.idle, p.total) }" />
                <div class="bg-red-400" :style="{ width: pct(Math.min(p.issues, p.total - p.schedulable), p.total) }" />
              </div>
              <p class="mt-1 text-[11px] text-gray-400 dark:text-dark-400">
                {{ t('admin.operationsCenter.health.platformLine', { schedulable: p.schedulable, issues: p.issues, idle: p.idle }) }}
              </p>
            </div>
          </div>
        </div>

        <!-- 24h 高失败率 -->
        <div class="card overflow-hidden xl:col-span-2">
          <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.operationsCenter.health.highFailure') }}</h3>
            <span class="text-[11px] text-gray-400 dark:text-dark-400">
              {{ t('admin.operationsCenter.health.highFailureHint', { rate: HIGH_FAILURE_RATE * 100, min: MIN_SAMPLE }) }}
            </span>
          </div>
          <div v-if="highFailure.length === 0" class="py-10 text-center text-sm text-gray-400 dark:text-dark-400">
            {{ t('admin.operationsCenter.health.noHighFailure') }}
          </div>
          <div v-else class="max-h-80 overflow-auto">
            <table class="w-full text-sm">
              <thead class="sticky top-0 bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.operationsCenter.ranking.cols.account') }}</th>
                  <th class="px-4 py-2 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.requests') }}</th>
                  <th class="px-4 py-2 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.errors') }}</th>
                  <th class="px-4 py-2 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.failureRate') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="r in highFailure" :key="r.account_id">
                  <td class="px-4 py-2">
                    <div class="flex items-center gap-2">
                      <PlatformIcon :platform="r.platform as any" size="sm" />
                      <span class="truncate font-medium text-gray-900 dark:text-white">{{ r.name || `#${r.account_id}` }}</span>
                      <span class="text-[11px] text-gray-400">#{{ r.account_id }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-2 text-right text-gray-700 dark:text-gray-200">{{ formatInt(r.requests) }}</td>
                  <td class="px-4 py-2 text-right text-red-600 dark:text-red-400">
                    {{ formatInt(r.errors) }}
                    <span v-if="r.rate_limited > 0" class="ml-1 text-[10px] text-amber-500">429×{{ r.rate_limited }}</span>
                  </td>
                  <td class="px-4 py-2 text-right font-semibold text-red-600 dark:text-red-400">
                    {{ formatPercent(r.errors / (r.requests + r.errors)) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- 问题账号 -->
      <div class="card overflow-hidden">
        <div class="flex flex-col gap-2 border-b border-gray-100 px-4 py-3 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.operationsCenter.health.issues') }}
            <span class="ml-1 text-xs font-normal text-gray-400">{{ filteredIssues.length }}</span>
          </h3>
          <div class="flex flex-wrap gap-1">
            <button
              v-for="k in issueKinds"
              :key="k"
              type="button"
              class="rounded-lg px-2.5 py-1 text-xs font-medium transition-colors"
              :class="issueFilter === k ? 'bg-primary-500 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
              @click="issueFilter = k"
            >
              {{ k === 'all' ? t('admin.operationsCenter.filter.all') : t(`admin.operationsCenter.health.kinds.${k}`) }}
              <span class="opacity-70">{{ issueCount(k) }}</span>
            </button>
          </div>
        </div>
        <div v-if="filteredIssues.length === 0" class="py-10 text-center text-sm text-gray-400 dark:text-dark-400">
          {{ t('admin.operationsCenter.health.noIssues') }}
        </div>
        <div v-else class="max-h-[28rem] overflow-auto">
          <table class="w-full min-w-[720px] text-sm">
            <thead class="sticky top-0 bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.operationsCenter.ranking.cols.account') }}</th>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.operationsCenter.health.kind') }}</th>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.operationsCenter.health.detail') }}</th>
                <th class="px-4 py-2 text-right font-medium">{{ t('admin.operationsCenter.health.until') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="(issue, idx) in filteredIssues" :key="`${issue.account.id}-${issue.kind}-${idx}`">
                <td class="px-4 py-2">
                  <div class="flex items-center gap-2">
                    <PlatformIcon :platform="issue.account.platform as any" size="sm" />
                    <div class="min-w-0">
                      <p class="truncate font-medium text-gray-900 dark:text-white">{{ issue.account.name || `#${issue.account.id}` }}</p>
                      <p class="truncate text-[11px] text-gray-400 dark:text-dark-400">#{{ issue.account.id }} · {{ issue.account.type }}</p>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-2">
                  <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium" :class="kindClass(issue.kind)">
                    {{ t(`admin.operationsCenter.health.kinds.${issue.kind}`) }}
                  </span>
                </td>
                <td class="max-w-md px-4 py-2 text-xs text-gray-500 dark:text-dark-400">
                  <p class="line-clamp-2" :title="issue.detail">{{ issue.detail || '-' }}</p>
                </td>
                <td class="whitespace-nowrap px-4 py-2 text-right text-xs text-gray-500 dark:text-dark-400">
                  <template v-if="issue.until">
                    {{ formatDateTime(issue.until) }}
                    <span class="block text-[10px] text-gray-400">{{ formatRelative(issue.until) }}</span>
                  </template>
                  <template v-else>-</template>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 闲置账号 -->
      <div class="card overflow-hidden">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.operationsCenter.health.idle') }}
            <span class="ml-1 text-xs font-normal text-gray-400">{{ data.idle.length }}</span>
          </h3>
          <span class="text-[11px] text-gray-400 dark:text-dark-400">
            {{ t('admin.operationsCenter.health.idleHint', { days: Math.round(data.idle_threshold_hours / 24) }) }}
          </span>
        </div>
        <div v-if="data.idle.length === 0" class="py-10 text-center text-sm text-gray-400 dark:text-dark-400">
          {{ t('admin.operationsCenter.health.noIdle') }}
        </div>
        <div v-else class="grid max-h-[28rem] grid-cols-1 gap-px overflow-auto bg-gray-100 dark:bg-dark-700 sm:grid-cols-2 xl:grid-cols-3">
          <div v-for="a in data.idle" :key="a.id" class="flex items-center gap-2 bg-white px-4 py-2.5 dark:bg-dark-900">
            <PlatformIcon :platform="a.platform as any" size="sm" />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white" :title="a.base_url">{{ a.name || `#${a.id}` }}</p>
              <p class="truncate text-[11px] text-gray-400 dark:text-dark-400">#{{ a.id }} · {{ a.type }}</p>
            </div>
            <span class="flex-shrink-0 text-[11px]" :class="a.last_used_at ? 'text-gray-500 dark:text-dark-400' : 'text-amber-500'">
              {{ a.last_used_at ? formatRelative(a.last_used_at) : t('admin.operationsCenter.health.neverUsed') }}
            </span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import operationsCenterAPI, {
  type AccountHealthResponse,
  type AccountRequestRankingItem,
  type HealthIssueKind,
} from '@/api/admin/operationsCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const HIGH_FAILURE_RATE = 0.2
const MIN_SAMPLE = 10

const { t, locale } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const data = ref<AccountHealthResponse | null>(null)
const ranking24h = ref<AccountRequestRankingItem[]>([])
const issueFilter = ref<'all' | HealthIssueKind>('all')

const ALL_KINDS: HealthIssueKind[] = [
  'error',
  'expired',
  'quota_exceeded',
  'rate_limited',
  'overloaded',
  'temp_unschedulable',
  'expiring_soon',
  'paused',
]

/** 只展示当前有数据的问题类型 */
const issueKinds = computed<('all' | HealthIssueKind)[]>(() => ['all', ...ALL_KINDS.filter((k) => issueCount(k) > 0)])

function issueCount(kind: 'all' | HealthIssueKind): number {
  const issues = data.value?.issues ?? []
  return kind === 'all' ? issues.length : issues.filter((i) => i.kind === kind).length
}

const filteredIssues = computed(() => {
  const issues = data.value?.issues ?? []
  return issueFilter.value === 'all' ? issues : issues.filter((i) => i.kind === issueFilter.value)
})

const highFailure = computed(() =>
  ranking24h.value
    .filter((r) => r.requests + r.errors >= MIN_SAMPLE && r.errors / (r.requests + r.errors) >= HIGH_FAILURE_RATE)
    .sort((a, b) => b.errors / (b.requests + b.errors) - a.errors / (a.requests + a.errors) || b.errors - a.errors),
)

const summaryCards = computed(() => {
  const s = data.value?.summary
  if (!s) return []
  const limited = s.rate_limited + s.overloaded + s.temp_unschedulable + s.quota_exceeded
  return [
    {
      label: t('admin.operationsCenter.health.summary.total'),
      value: formatInt(s.total),
      sub: t('admin.operationsCenter.health.summary.totalSub', { active: s.active, disabled: s.disabled }),
      cls: 'text-gray-900 dark:text-white',
    },
    {
      label: t('admin.operationsCenter.health.summary.schedulable'),
      value: formatInt(s.schedulable),
      sub: formatPercent(s.total > 0 ? s.schedulable / s.total : 0),
      cls: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: t('admin.operationsCenter.health.summary.error'),
      value: formatInt(s.error),
      sub: s.expired > 0 ? `${t('admin.operationsCenter.health.kinds.expired')} ${s.expired}` : '',
      cls: s.error > 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-white',
    },
    {
      label: t('admin.operationsCenter.health.summary.limited'),
      value: formatInt(limited),
      sub: t('admin.operationsCenter.health.summary.limitedSub', { rl: s.rate_limited, ol: s.overloaded, tu: s.temp_unschedulable }),
      cls: limited > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-gray-900 dark:text-white',
    },
    {
      label: t('admin.operationsCenter.health.summary.paused'),
      value: formatInt(s.paused),
      sub: s.expiring_soon > 0 ? `${t('admin.operationsCenter.health.kinds.expiring_soon')} ${s.expiring_soon}` : '',
      cls: 'text-gray-900 dark:text-white',
    },
    {
      label: t('admin.operationsCenter.health.summary.idle'),
      value: formatInt(s.idle),
      sub: formatPercent(s.schedulable > 0 ? s.idle / s.schedulable : 0),
      cls: s.idle > 0 ? 'text-gray-500 dark:text-gray-300' : 'text-gray-900 dark:text-white',
    },
  ]
})

function kindClass(kind: HealthIssueKind): string {
  switch (kind) {
    case 'error':
    case 'expired':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    case 'quota_exceeded':
    case 'rate_limited':
    case 'overloaded':
    case 'temp_unschedulable':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'expiring_soon':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  }
}

function pct(n: number, total: number): string {
  return total > 0 ? `${(Math.max(0, n) / total) * 100}%` : '0%'
}

function formatInt(v: number): string {
  return Math.round(v || 0).toLocaleString(locale.value)
}

function formatPercent(v: number): string {
  return `${(v * 100).toFixed(v >= 0.1 || v === 0 ? 0 : 1)}%`
}

function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function formatRelative(iso: string): string {
  const ts = new Date(iso).getTime()
  if (!Number.isFinite(ts)) return '-'
  const diffSec = Math.round((ts - Date.now()) / 1000)
  const rtf = new Intl.RelativeTimeFormat(locale.value, { numeric: 'auto' })
  const abs = Math.abs(diffSec)
  if (abs < 60) return rtf.format(diffSec, 'second')
  if (abs < 3600) return rtf.format(Math.round(diffSec / 60), 'minute')
  if (abs < 86400) return rtf.format(Math.round(diffSec / 3600), 'hour')
  return rtf.format(Math.round(diffSec / 86400), 'day')
}

async function load() {
  loading.value = true
  try {
    const end = new Date()
    const start = new Date(end.getTime() - 24 * 3600_000)
    const iso = (d: Date) => d.toISOString().replace(/\.\d{3}Z$/, 'Z')
    // 24h 失败率只是附加信息，失败时不影响健康数据
    const [health, ranking] = await Promise.all([
      operationsCenterAPI.getAccountHealth(),
      operationsCenterAPI
        .getAccountRanking({ start_time: iso(start), end_time: iso(end), group_by: 'account', limit: 1000 })
        .catch(() => null),
    ])
    data.value = health
    ranking24h.value = ranking?.items ?? []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.operationsCenter.health.loadFailed')))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
