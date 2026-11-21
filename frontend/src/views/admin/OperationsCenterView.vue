<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- 标签页 -->
      <div class="flex gap-1 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tb in tabs"
          :key="tb"
          type="button"
          class="-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors"
          :class="
            tab === tb
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-gray-200'
          "
          @click="switchTab(tb)"
        >
          {{ t(`admin.operationsCenter.tabs.${tb}`) }}
        </button>
      </div>

      <AccountRequestRanking v-if="opened.ranking" v-show="tab === 'ranking'" />
      <SiteTrend v-if="opened.trend" v-show="tab === 'trend'" />
      <AccountHealth v-if="opened.health" v-show="tab === 'health'" />

      <template v-if="tab === 'sites'">
      <!-- 工具栏 -->
      <div class="card flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between">
        <div class="flex flex-1 flex-col gap-3 sm:flex-row sm:items-center">
          <div class="relative w-full sm:max-w-xs">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input v-model="keyword" type="text" class="input py-1.5 pl-9 text-sm" :placeholder="t('admin.operationsCenter.search')" />
          </div>
          <div class="flex flex-wrap gap-1">
            <button
              v-for="f in filters"
              :key="f"
              type="button"
              class="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors"
              :class="
                activeFilter === f
                  ? 'bg-primary-500 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'
              "
              @click="activeFilter = f"
            >
              {{ t(`admin.operationsCenter.filter.${f}`) }}
            </button>
          </div>
        </div>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadSites">
            <Icon name="refresh" size="sm" class="mr-1" :class="{ 'animate-spin': loading }" />
            {{ t('admin.operationsCenter.refresh') }}
          </button>
          <button type="button" class="btn btn-primary btn-sm" :disabled="refreshing" @click="refreshBalances">
            <Icon name="dollar" size="sm" class="mr-1" :class="{ 'animate-pulse': refreshing }" />
            {{ refreshing ? t('admin.operationsCenter.refreshing') : t('admin.operationsCenter.refreshBalances') }}
          </button>
        </div>
      </div>

      <!-- 汇总 -->
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.operationsCenter.summary.sites') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ sites.length }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.operationsCenter.summary.accounts') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ totalAccounts }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.operationsCenter.summary.remaining') }}</p>
          <p class="mt-1 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
            <template v-if="summaryByUnit.length === 0">-</template>
            <span v-for="(s, i) in summaryByUnit" :key="s.unit" class="block text-xl leading-tight first:text-2xl" :class="{ 'mt-0.5': i > 0 }">
              {{ formatAmount(s.remaining, s.unit) }}
            </span>
          </p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.operationsCenter.summary.total') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
            <template v-if="summaryByUnit.length === 0">-</template>
            <span v-for="(s, i) in summaryByUnit" :key="s.unit" class="block text-xl leading-tight first:text-2xl" :class="{ 'mt-0.5': i > 0 }">
              {{ formatAmount(s.total, s.unit) }}
            </span>
          </p>
          <p v-if="noDataCount > 0" class="mt-1 text-[11px] text-gray-400 dark:text-dark-400">
            {{ t('admin.operationsCenter.summary.noData') }}: {{ noDataCount }}
          </p>
        </div>
        <div class="card p-4">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.operationsCenter.summary.cost24h') }}</p>
          <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatAmount(totalCost24h, 'USD') }}</p>
          <p class="mt-1 text-[11px] text-gray-400 dark:text-dark-400">
            {{ t('admin.operationsCenter.summary.dailyAvg7d') }}: {{ formatAmount(totalDailyAvg, 'USD') }}
          </p>
        </div>
      </div>

      <!-- 卡片 -->
      <div v-if="loading && sites.length === 0" class="flex justify-center py-16">
        <LoadingSpinner />
      </div>
      <div v-else-if="filteredSites.length === 0" class="card py-16">
        <EmptyState :title="t('admin.operationsCenter.empty')" />
      </div>
      <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <a
          v-for="site in filteredSites"
          :key="site.key"
          :href="site.site_url"
          target="_blank"
          rel="noopener noreferrer"
          class="card group flex flex-col gap-3 p-4 transition-all duration-150 hover:-translate-y-0.5 hover:shadow-lg hover:ring-1 hover:ring-primary-200 dark:hover:ring-primary-800/60"
          :title="t('admin.operationsCenter.card.open')"
        >
          <!-- 头部 -->
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <div class="flex items-center gap-1.5">
                <img
                  :src="faviconUrl(site)"
                  class="h-4 w-4 flex-shrink-0 rounded-sm"
                  alt=""
                  loading="lazy"
                  @error="onFaviconError"
                />
                <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ site.host || site.base_url }}</h3>
              </div>
              <p class="mt-0.5 truncate text-[11px] text-gray-400 dark:text-dark-400">{{ site.base_url }}</p>
            </div>
            <Icon name="externalLink" size="sm" class="flex-shrink-0 text-gray-300 transition-colors group-hover:text-primary-500 dark:text-dark-500" />
          </div>

          <!-- 额度 -->
          <template v-if="site.has_balance">
            <div>
              <p class="text-[11px] text-gray-500 dark:text-dark-400">{{ t('admin.operationsCenter.card.remaining') }}</p>
              <p class="text-2xl font-semibold" :class="remainingClass(site)">
                <template v-if="site.unlimited">{{ t('admin.operationsCenter.card.unlimited') }}</template>
                <template v-else>{{ formatAmount(site.remaining_quota, site.unit) }}</template>
              </p>
            </div>
            <div v-if="usageRatio(site) !== null" class="h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <div class="h-full rounded-full transition-all" :class="barClass(site)" :style="{ width: `${(usageRatio(site) ?? 0) * 100}%` }" />
            </div>
            <div class="grid grid-cols-2 gap-2 text-xs">
              <div>
                <p class="text-gray-400 dark:text-dark-400">{{ t('admin.operationsCenter.card.total') }}</p>
                <p class="font-medium text-gray-700 dark:text-gray-200">{{ formatAmount(site.total_quota, site.unit) }}</p>
              </div>
              <div>
                <p class="text-gray-400 dark:text-dark-400">{{ t('admin.operationsCenter.card.used') }}</p>
                <p class="font-medium text-gray-700 dark:text-gray-200">{{ formatAmount(site.used_quota, site.unit) }}</p>
              </div>
            </div>
          </template>
          <div v-else class="flex-1 rounded-lg bg-gray-50 p-3 text-xs text-gray-400 dark:bg-dark-800 dark:text-dark-400">
            {{ t('admin.operationsCenter.card.noData') }}
          </div>

          <!-- 消耗与可用天数（账号口径成本，按近 7 天日均估算） -->
          <div v-if="siteBurn(site).cost7d > 0 || siteBurn(site).cost24h > 0" class="grid grid-cols-3 gap-2 rounded-lg bg-gray-50 px-3 py-2 text-xs dark:bg-dark-800">
            <div>
              <p class="text-gray-400 dark:text-dark-400">{{ t('admin.operationsCenter.card.cost24h') }}</p>
              <p class="font-medium text-gray-700 dark:text-gray-200">{{ formatAmount(siteBurn(site).cost24h, 'USD') }}</p>
            </div>
            <div>
              <p class="text-gray-400 dark:text-dark-400">{{ t('admin.operationsCenter.card.dailyAvg') }}</p>
              <p class="font-medium text-gray-700 dark:text-gray-200">{{ formatAmount(siteBurn(site).dailyAvg, 'USD') }}</p>
            </div>
            <div :title="t('admin.operationsCenter.card.daysLeftHint')">
              <p class="text-gray-400 dark:text-dark-400">{{ t('admin.operationsCenter.card.daysLeft') }}</p>
              <p class="font-semibold" :class="daysLeftClass(siteBurn(site).daysLeft)">{{ formatDays(siteBurn(site).daysLeft) }}</p>
            </div>
          </div>

          <!-- 底部 -->
          <div class="mt-auto flex items-center justify-between gap-2 border-t border-gray-100 pt-2 text-[11px] text-gray-400 dark:border-dark-700 dark:text-dark-400">
            <span class="truncate" :title="site.accounts.map((a) => a.name).join('\n')">
              {{ t('admin.operationsCenter.card.accounts', { n: site.account_count }) }}
              <template v-if="site.plan_name"> · {{ site.plan_name }}</template>
            </span>
            <span
              v-if="site.updated_at"
              class="flex-shrink-0"
              :title="site.source_account ? t('admin.operationsCenter.card.source', { name: site.source_account.name }) : ''"
            >
              {{ t('admin.operationsCenter.card.updatedAt', { time: formatRelative(site.updated_at) }) }}
            </span>
          </div>
        </a>
      </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import AccountRequestRanking from '@/components/operationsCenter/AccountRequestRanking.vue'
import SiteTrend from '@/components/operationsCenter/SiteTrend.vue'
import AccountHealth from '@/components/operationsCenter/AccountHealth.vue'
import { adminAPI } from '@/api/admin'
import operationsCenterAPI, { type AccountConsumption, type OperationsSite } from '@/api/admin/operationsCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

type FilterKey = 'all' | 'withBalance' | 'low' | 'soon' | 'noData'
type TabKey = 'sites' | 'ranking' | 'trend' | 'health'

const { t, locale } = useI18n()
const appStore = useAppStore()

const sites = ref<OperationsSite[]>([])
const loading = ref(false)
const refreshing = ref(false)
const keyword = ref('')
const filters: FilterKey[] = ['all', 'withBalance', 'low', 'soon', 'noData']
const activeFilter = ref<FilterKey>('all')
const tabs: TabKey[] = ['sites', 'ranking', 'trend', 'health']
const tab = ref<TabKey>('sites')
// 子标签页首次打开时才挂载（避免进入页面就并发请求），之后用 v-show 保留状态
const opened = reactive<Record<TabKey, boolean>>({ sites: true, ranking: false, trend: false, health: false })

function switchTab(next: TabKey) {
  tab.value = next
  opened[next] = true
}

const LOW_RATIO = 0.2
const LOW_ABSOLUTE = 5
const SOON_DAYS = 7

const consumption = ref<Map<number, AccountConsumption>>(new Map())

interface SiteBurn {
  cost24h: number
  cost7d: number
  dailyAvg: number
  /** 预计可用天数；无法估算（无限额度 / 无余额 / 无消耗 / 非 USD）时为 null */
  daysLeft: number | null
}

/** 站点消耗：汇总站点下所有账号；日均按实际有数据的天数（最多 7 天，最少 1 天）计算，避免新账号被低估 */
function siteBurn(site: OperationsSite): SiteBurn {
  let cost24h = 0
  let cost7d = 0
  let earliest = Infinity
  for (const a of site.accounts) {
    const c = consumption.value.get(a.id)
    if (!c) continue
    cost24h += c.cost_24h
    cost7d += c.cost_7d
    const ts = new Date(c.first_used_at).getTime()
    if (Number.isFinite(ts)) earliest = Math.min(earliest, ts)
  }
  const spanDays = Number.isFinite(earliest) ? Math.min(7, Math.max(1, (Date.now() - earliest) / 86400_000)) : 7
  const dailyAvg = cost7d / spanDays
  const unit = (site.unit || 'USD').toUpperCase()
  let daysLeft: number | null = null
  if (site.has_balance && !site.unlimited && site.remaining_quota != null && dailyAvg > 0 && unit === 'USD') {
    daysLeft = Math.max(0, site.remaining_quota) / dailyAvg
  }
  return { cost24h, cost7d, dailyAvg, daysLeft }
}

const totalCost24h = computed(() => sites.value.reduce((sum, s) => sum + siteBurn(s).cost24h, 0))
const totalDailyAvg = computed(() => sites.value.reduce((sum, s) => sum + siteBurn(s).dailyAvg, 0))

function formatDays(days: number | null): string {
  if (days == null) return '-'
  if (days >= 365) return '365+'
  if (days < 1) return t('admin.operationsCenter.card.lessThanDay')
  return t('admin.operationsCenter.card.days', { n: days >= 10 ? Math.round(days) : days.toFixed(1) })
}

function daysLeftClass(days: number | null): string {
  if (days == null) return 'text-gray-500 dark:text-gray-400'
  if (days < 3) return 'text-red-600 dark:text-red-400'
  if (days < SOON_DAYS) return 'text-amber-600 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

function isLow(site: OperationsSite): boolean {
  if (!site.has_balance || site.unlimited || site.remaining_quota == null) return false
  if (site.total_quota && site.total_quota > 0) return site.remaining_quota / site.total_quota < LOW_RATIO
  return site.remaining_quota < LOW_ABSOLUTE
}

const filteredSites = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return sites.value.filter((site) => {
    if (activeFilter.value === 'withBalance' && !site.has_balance) return false
    if (activeFilter.value === 'noData' && site.has_balance) return false
    if (activeFilter.value === 'low' && !isLow(site)) return false
    if (activeFilter.value === 'soon' && !((siteBurn(site).daysLeft ?? Infinity) < SOON_DAYS)) return false
    if (!kw) return true
    return (
      site.base_url.toLowerCase().includes(kw) ||
      site.host.toLowerCase().includes(kw) ||
      site.accounts.some((a) => a.name.toLowerCase().includes(kw))
    )
  })
})

const totalAccounts = computed(() => sites.value.reduce((sum, s) => sum + s.account_count, 0))
const noDataCount = computed(() => sites.value.filter((s) => !s.has_balance).length)

const summaryByUnit = computed(() => {
  const map = new Map<string, { unit: string; remaining: number; total: number }>()
  for (const site of sites.value) {
    if (!site.has_balance || site.unlimited || site.remaining_quota == null) continue
    const unit = site.unit || 'USD'
    const entry = map.get(unit) ?? { unit, remaining: 0, total: 0 }
    entry.remaining += site.remaining_quota
    entry.total += site.total_quota ?? site.remaining_quota
    map.set(unit, entry)
  }
  return [...map.values()]
})

function usageRatio(site: OperationsSite): number | null {
  if (site.unlimited || !site.total_quota || site.total_quota <= 0 || site.remaining_quota == null) return null
  const used = 1 - site.remaining_quota / site.total_quota
  return Math.min(1, Math.max(0, used))
}

function remainingClass(site: OperationsSite): string {
  if (site.unlimited) return 'text-primary-600 dark:text-primary-400'
  if ((site.remaining_quota ?? 0) <= 0) return 'text-red-600 dark:text-red-400'
  if (isLow(site)) return 'text-amber-600 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

function barClass(site: OperationsSite): string {
  if ((site.remaining_quota ?? 0) <= 0) return 'bg-red-500'
  if (isLow(site)) return 'bg-amber-500'
  return 'bg-emerald-500'
}

function formatAmount(value: number | undefined | null, unit?: string): string {
  if (value == null || !Number.isFinite(value)) return '-'
  const abs = Math.abs(value)
  const digits = abs >= 1000 ? 0 : abs >= 1 ? 2 : 4
  const num = value.toLocaleString(locale.value, { minimumFractionDigits: 0, maximumFractionDigits: digits })
  const u = (unit || 'USD').toUpperCase()
  if (u === 'USD') return `$${num}`
  if (u === 'CNY' || u === 'RMB') return `¥${num}`
  return `${num} ${unit}`
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

function faviconUrl(site: OperationsSite): string {
  try {
    return `${new URL(site.site_url).origin}/favicon.ico`
  } catch {
    return ''
  }
}

function onFaviconError(e: Event) {
  const img = e.target as HTMLImageElement
  img.style.display = 'none'
}

async function loadSites() {
  loading.value = true
  try {
    // 消耗数据只用于可用天数估算，失败时不影响站点卡片展示
    const [data, items] = await Promise.all([
      operationsCenterAPI.listSites(),
      operationsCenterAPI.getConsumption().catch(() => [] as AccountConsumption[]),
    ])
    sites.value = data.items ?? []
    consumption.value = new Map(items.map((c) => [c.account_id, c]))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.operationsCenter.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function refreshBalances() {
  refreshing.value = true
  try {
    const result = await adminAPI.accounts.refreshUpstreamBalances()
    appStore.showSuccess(
      t('admin.operationsCenter.refreshResult', {
        success: result.success,
        skipped: result.skipped ?? 0,
        failed: result.failed,
      }),
    )
    await loadSites()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.operationsCenter.refreshFailed')))
  } finally {
    refreshing.value = false
  }
}

onMounted(loadSites)
</script>
