<template>
  <div class="space-y-4">
    <!-- 时间范围 -->
    <div class="card flex flex-col gap-3 p-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.operationsCenter.ranking.timeRange') }}:</span>
        <button
          v-for="p in presets"
          :key="p"
          type="button"
          class="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors"
          :class="
            preset === p
              ? 'bg-primary-500 text-white'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
          "
          @click="selectPreset(p)"
        >
          {{ t(`admin.operationsCenter.ranking.presets.${p}`) }}
        </button>
        <DateRangePicker
          v-if="preset === 'custom'"
          v-model:start-date="startDate"
          v-model:end-date="endDate"
          @change="load"
        />
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <div class="flex overflow-hidden rounded-lg border border-gray-200 text-xs dark:border-dark-600">
          <button
            v-for="g in groupModes"
            :key="g"
            type="button"
            class="px-3 py-1.5 font-medium transition-colors"
            :class="groupBy === g ? 'bg-primary-500 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
            @click="selectGroupBy(g)"
          >
            {{ t(`admin.operationsCenter.ranking.groupBy.${g}`) }}
          </button>
        </div>
        <div class="relative">
          <Icon
            name="search"
            size="sm"
            class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
          />
          <input v-model="keyword" type="text" class="input w-48 py-1.5 pl-9 text-sm" :placeholder="t('admin.operationsCenter.ranking.search')" />
        </div>
        <select v-model.number="limit" class="input w-auto py-1.5 text-sm" @change="load">
          <option v-for="n in [50, 100, 200, 500]" :key="n" :value="n">Top {{ n }}</option>
        </select>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
          <Icon name="refresh" size="sm" class="mr-1" :class="{ 'animate-spin': loading }" />
          {{ t('admin.operationsCenter.refresh') }}
        </button>
      </div>
    </div>

    <!-- 汇总 -->
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-6">
      <div v-for="card in summaryCards" :key="card.label" class="card p-4">
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ card.label }}</p>
        <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ card.value }}</p>
      </div>
    </div>

    <!-- 排行表 -->
    <div class="card overflow-hidden">
      <div v-if="loading && !data" class="flex justify-center py-16">
        <LoadingSpinner />
      </div>
      <div v-else-if="sortedItems.length === 0" class="py-16">
        <EmptyState :title="t('admin.operationsCenter.ranking.empty')" />
      </div>
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[1120px] text-sm">
          <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="w-12 px-3 py-2.5 text-left font-medium">#</th>
              <th class="px-3 py-2.5 text-left font-medium">
                {{ groupBy === 'site' ? t('admin.operationsCenter.ranking.cols.site') : t('admin.operationsCenter.ranking.cols.account') }}
              </th>
              <th
                v-for="col in metricCols"
                :key="col.key"
                class="cursor-pointer select-none whitespace-nowrap px-3 py-2.5 text-right font-medium hover:text-gray-700 dark:hover:text-gray-200"
                :class="{ 'text-primary-600 dark:text-primary-400': sortKey === col.key }"
                @click="sortKey = col.key"
              >
                {{ col.label }}
                <span v-if="sortKey === col.key">↓</span>
              </th>
              <th class="whitespace-nowrap px-3 py-2.5 text-right font-medium">{{ t('admin.operationsCenter.ranking.cols.lastUsed') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="(item, idx) in sortedItems" :key="item.key" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
              <td class="px-3 py-2.5">
                <span
                  class="inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                  :class="rankClass(idx)"
                >
                  {{ idx + 1 }}
                </span>
              </td>
              <td class="max-w-[300px] px-3 py-2.5">
                <div class="flex items-center gap-2">
                  <PlatformIcon :platform="item.platform as any" size="sm" />
                  <div class="min-w-0">
                    <a
                      v-if="item.siteUrl"
                      :href="item.siteUrl"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="flex items-center gap-1 truncate font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
                      :title="item.baseUrl"
                    >
                      <span class="truncate">{{ item.title }}</span>
                      <Icon name="externalLink" size="xs" class="flex-shrink-0 opacity-60" />
                    </a>
                    <p v-else class="truncate font-medium text-gray-900 dark:text-white" :title="item.title">{{ item.title }}</p>
                    <p class="truncate text-[11px] text-gray-400 dark:text-dark-400" :title="item.subtitleTitle">
                      {{ item.subtitle }}
                      <span v-if="item.deleted" class="text-red-500"> · {{ t('admin.operationsCenter.ranking.deleted') }}</span>
                      <span v-else-if="item.status && item.status !== 'active'" class="text-amber-500"> · {{ item.status }}</span>
                    </p>
                  </div>
                </div>
              </td>
              <td class="px-3 py-2.5 text-right">
                <div class="font-semibold text-gray-900 dark:text-white">{{ formatInt(item.requests) }}</div>
                <div class="ml-auto mt-1 flex w-28 items-center gap-1.5">
                  <div class="h-1 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                    <div class="h-full rounded-full bg-primary-500" :style="{ width: `${share(item.requests) * 100}%` }" />
                  </div>
                  <span class="w-10 text-right text-[10px] text-gray-400">{{ formatPercent(share(item.requests)) }}</span>
                </div>
              </td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200" :title="tokenTitle(item)">{{ formatCompact(item.tokens) }}</td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatMoney(item.account_cost) }}</td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatMoney(item.user_cost) }}</td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatMs(item.avg_duration_ms) }}</td>
              <td class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatMs(item.avg_first_token_ms) }}</td>
              <td class="px-3 py-2.5 text-right">
                <div :class="item.errors > 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-400'">{{ formatInt(item.errors) }}</div>
                <div v-if="item.rate_limited > 0" class="text-[10px] text-amber-500">429 × {{ formatInt(item.rate_limited) }}</div>
              </td>
              <td class="px-3 py-2.5 text-right font-medium" :class="failureClass(item.failure_rate)">
                {{ item.requests + item.errors > 0 ? formatPercent(item.failure_rate) : '-' }}
              </td>
              <td v-if="groupBy === 'account'" class="px-3 py-2.5 text-right text-gray-700 dark:text-gray-200">{{ formatInt(item.users) }}</td>
              <td class="whitespace-nowrap px-3 py-2.5 text-right text-xs text-gray-400 dark:text-dark-400">
                {{ item.last_used_at ? formatDateTime(item.last_used_at) : '-' }}
              </td>
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
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import operationsCenterAPI, {
  type AccountRequestRankingItem,
  type AccountRequestRankingParams,
  type AccountRequestRankingResponse,
  type SiteRequestRankingItem,
} from '@/api/admin/operationsCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

type Preset = '1h' | '6h' | '24h' | '7d' | '30d' | 'custom'
type SortKey = 'requests' | 'tokens' | 'account_cost' | 'user_cost' | 'avg_duration_ms' | 'avg_first_token_ms' | 'errors' | 'failure_rate' | 'users'
type GroupBy = 'site' | 'account'

/** 站点行与账号行统一后的展示行 */
interface RankingRow {
  key: string
  title: string
  subtitle: string
  subtitleTitle: string
  platform: string
  status: string
  deleted: boolean
  siteUrl: string
  baseUrl: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_tokens: number
  tokens: number
  account_cost: number
  user_cost: number
  avg_duration_ms: number
  avg_first_token_ms: number
  users: number
  errors: number
  rate_limited: number
  /** 失败率 = 失败 / (成功 + 失败) */
  failure_rate: number
  last_used_at?: string
}

const { t, locale } = useI18n()
const appStore = useAppStore()

const presets: Preset[] = ['1h', '6h', '24h', '7d', '30d', 'custom']
const PRESET_HOURS: Record<Exclude<Preset, 'custom'>, number> = { '1h': 1, '6h': 6, '24h': 24, '7d': 24 * 7, '30d': 24 * 30 }
const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'

const preset = ref<Preset>('24h')
const startDate = ref(formatLocalDate(new Date(Date.now() - 6 * 86400_000)))
const endDate = ref(formatLocalDate(new Date()))
const limit = ref(100)
const keyword = ref('')
const sortKey = ref<SortKey>('requests')
const groupModes: GroupBy[] = ['site', 'account']
const groupBy = ref<GroupBy>('site')
const loading = ref(false)
const data = ref<AccountRequestRankingResponse | null>(null)

const metricCols = computed<{ key: SortKey; label: string }[]>(() => {
  const cols: { key: SortKey; label: string }[] = [
  { key: 'requests', label: t('admin.operationsCenter.ranking.cols.requests') },
  { key: 'tokens', label: t('admin.operationsCenter.ranking.cols.tokens') },
  { key: 'account_cost', label: t('admin.operationsCenter.ranking.cols.accountCost') },
  { key: 'user_cost', label: t('admin.operationsCenter.ranking.cols.userCost') },
  { key: 'avg_duration_ms', label: t('admin.operationsCenter.ranking.cols.avgDuration') },
  { key: 'avg_first_token_ms', label: t('admin.operationsCenter.ranking.cols.avgFirstToken') },
  { key: 'errors', label: t('admin.operationsCenter.ranking.cols.errors') },
  { key: 'failure_rate', label: t('admin.operationsCenter.ranking.cols.failureRate') },
  ]
  if (groupBy.value === 'account') cols.push({ key: 'users', label: t('admin.operationsCenter.ranking.cols.users') })
  return cols
})

function accountRow(i: AccountRequestRankingItem): RankingRow {
  return {
    key: `account:${i.account_id}`,
    title: i.name || `#${i.account_id}`,
    subtitle: `#${i.account_id} · ${i.type || '-'}`,
    subtitleTitle: i.base_url || '',
    platform: i.platform,
    status: i.status,
    deleted: i.deleted,
    siteUrl: '',
    baseUrl: i.base_url || '',
    requests: i.requests,
    input_tokens: i.input_tokens,
    output_tokens: i.output_tokens,
    cache_tokens: i.cache_tokens,
    tokens: i.tokens,
    account_cost: i.account_cost,
    user_cost: i.user_cost,
    avg_duration_ms: i.avg_duration_ms,
    avg_first_token_ms: i.avg_first_token_ms,
    users: i.users,
    errors: i.errors,
    rate_limited: i.rate_limited,
    failure_rate: failureRate(i.requests, i.errors),
    last_used_at: i.last_used_at,
  }
}

function siteRow(s: SiteRequestRankingItem): RankingRow {
  const first = s.accounts[0]
  const names = s.accounts.map((a) => `${a.name || '#' + a.id} (#${a.id})`)
  const single = s.account_count === 1 && !s.host
  const platforms = new Set(s.accounts.map((a) => a.platform))
  return {
    key: s.key,
    title: s.host || first?.name || s.key,
    subtitle: single
      ? `#${first?.id} · ${first?.type || '-'}`
      : `${t('admin.operationsCenter.card.accounts', { n: s.account_count })} · ${names.slice(0, 3).join(', ')}${names.length > 3 ? '…' : ''}`,
    subtitleTitle: names.join('\n'),
    platform: platforms.size === 1 ? first?.platform || '' : '',
    status: single ? first?.status || '' : '',
    deleted: false,
    siteUrl: s.site_url,
    baseUrl: s.base_url,
    requests: s.requests,
    input_tokens: s.input_tokens,
    output_tokens: s.output_tokens,
    cache_tokens: s.cache_tokens,
    tokens: s.tokens,
    account_cost: s.account_cost,
    user_cost: s.user_cost,
    avg_duration_ms: s.avg_duration_ms,
    avg_first_token_ms: s.avg_first_token_ms,
    users: 0,
    errors: s.errors,
    rate_limited: s.rate_limited,
    failure_rate: failureRate(s.requests, s.errors),
    last_used_at: s.last_used_at,
  }
}

const rows = computed<RankingRow[]>(() => {
  const d = data.value
  if (!d) return []
  return d.group_by === 'site' ? (d.sites ?? []).map(siteRow) : (d.items ?? []).map(accountRow)
})

const sortedItems = computed<RankingRow[]>(() => {
  const kw = keyword.value.trim().toLowerCase()
  const items = rows.value.filter(
    (i) =>
      !kw ||
      i.title.toLowerCase().includes(kw) ||
      i.baseUrl.toLowerCase().includes(kw) ||
      i.subtitleTitle.toLowerCase().includes(kw) ||
      i.subtitle.toLowerCase().includes(kw) ||
      i.platform.toLowerCase().includes(kw),
  )
  const key = sortKey.value
  return [...items].sort((a, b) => (b[key] as number) - (a[key] as number) || b.requests - a.requests)
})

const summaryCards = computed(() => {
  const d = data.value
  return [
    d?.group_by === 'site'
      ? { label: t('admin.operationsCenter.ranking.summary.sites'), value: formatInt(d.total_sites ?? 0) }
      : { label: t('admin.operationsCenter.ranking.summary.accounts'), value: d ? formatInt(d.total_accounts) : '-' },
    { label: t('admin.operationsCenter.ranking.summary.requests'), value: d ? formatInt(d.total_requests) : '-' },
    { label: t('admin.operationsCenter.ranking.summary.tokens'), value: d ? formatCompact(d.total_tokens) : '-' },
    { label: t('admin.operationsCenter.ranking.summary.accountCost'), value: d ? formatMoney(d.total_account_cost) : '-' },
    { label: t('admin.operationsCenter.ranking.summary.userCost'), value: d ? formatMoney(d.total_user_cost) : '-' },
    {
      label: t('admin.operationsCenter.ranking.summary.errors'),
      value: d ? `${formatInt(d.total_errors)} (${formatPercent(failureRate(d.total_requests, d.total_errors))})` : '-',
    },
  ]
})

function failureRate(requests: number, errors: number): number {
  const total = requests + errors
  return total > 0 ? errors / total : 0
}

function failureClass(rate: number): string {
  if (rate >= 0.2) return 'text-red-600 dark:text-red-400'
  if (rate >= 0.05) return 'text-amber-600 dark:text-amber-400'
  return 'text-gray-700 dark:text-gray-200'
}

function share(requests: number): number {
  const total = data.value?.total_requests ?? 0
  return total > 0 ? requests / total : 0
}

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (idx === 1) return 'bg-gray-200 text-gray-700 dark:bg-dark-600 dark:text-gray-200'
  if (idx === 2) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
  return 'text-gray-400 dark:text-dark-400'
}

function tokenTitle(item: RankingRow): string {
  return t('admin.operationsCenter.ranking.tokenDetail', {
    input: formatInt(item.input_tokens),
    output: formatInt(item.output_tokens),
    cache: formatInt(item.cache_tokens),
  })
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

function formatMs(v: number): string {
  if (!v) return '-'
  return v >= 1000 ? `${(v / 1000).toFixed(2)}s` : `${Math.round(v)}ms`
}

function formatPercent(v: number): string {
  return `${(v * 100).toFixed(v >= 0.1 ? 0 : 1)}%`
}

function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function buildParams(): AccountRequestRankingParams {
  if (preset.value === 'custom') {
    return { start_date: startDate.value, end_date: endDate.value, timezone, limit: limit.value, group_by: groupBy.value }
  }
  const end = new Date()
  const start = new Date(end.getTime() - PRESET_HOURS[preset.value] * 3600_000)
  // 秒级 RFC3339，去掉毫秒以匹配后端解析
  const iso = (d: Date) => d.toISOString().replace(/\.\d{3}Z$/, 'Z')
  return { start_time: iso(start), end_time: iso(end), limit: limit.value, group_by: groupBy.value }
}

function selectGroupBy(g: GroupBy) {
  if (groupBy.value === g) return
  groupBy.value = g
  if (g === 'site' && sortKey.value === 'users') sortKey.value = 'requests'
  load()
}

function selectPreset(p: Preset) {
  preset.value = p
  load()
}

async function load() {
  loading.value = true
  try {
    data.value = await operationsCenterAPI.getAccountRanking(buildParams())
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.operationsCenter.ranking.loadFailed')))
  } finally {
    loading.value = false
  }
}

onMounted(load)

defineExpose({ load })
</script>
