<template>
  <div class="dash-stats space-y-6">
    <header class="rx-section-head">
      <span class="rx-section-head__index">01</span>
      <h2 class="rx-section-head__title">{{ t('dashboard.hero.metricsTitle') }}</h2>
    </header>

    <div class="dash-metrics">
      <article v-if="!isSimple" class="rx-glass rx-metric rx-rise dash-metric dash-metric--ink" style="--rx-i: 3">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.balance') }}</p>
        <p class="rx-metric__value">${{ formatBalance(balance) }}</p>
        <p class="rx-metric__hint">{{ t('common.available') }}</p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric" style="--rx-i: 4">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.apiKeys') }}</p>
        <p class="rx-metric__value">{{ stats?.total_api_keys || 0 }}</p>
        <p class="rx-metric__hint dash-metric__good">{{ stats?.active_api_keys || 0 }} {{ t('common.active') }}</p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric" style="--rx-i: 5">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.todayRequests') }}</p>
        <p class="rx-metric__value">{{ stats?.today_requests || 0 }}</p>
        <p class="rx-metric__hint">{{ t('common.total') }}: {{ formatNumber(stats?.total_requests || 0) }}</p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric dash-metric--cost" style="--rx-i: 6">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.todayCost') }}</p>
        <p class="rx-metric__value">
          <span :title="t('dashboard.actual')">${{ formatCost(stats?.today_actual_cost || 0) }}</span>
          <small :title="t('dashboard.standard')"> / ${{ formatCost(stats?.today_cost || 0) }}</small>
        </p>
        <p class="rx-metric__hint">
          <span>{{ t('common.total') }}: </span>
          <span :title="t('dashboard.actual')">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
          <span :title="t('dashboard.standard')"> / ${{ formatCost(stats?.total_cost || 0) }}</span>
        </p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric" style="--rx-i: 7">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.todayTokens') }}</p>
        <p class="rx-metric__value">{{ formatTokens(stats?.today_tokens || 0) }}</p>
        <p class="rx-metric__hint dash-metric__split">
          <span>{{ t('dashboard.input') }} {{ formatTokens(stats?.today_input_tokens || 0) }}</span>
          <span>{{ t('dashboard.output') }} {{ formatTokens(stats?.today_output_tokens || 0) }}</span>
          <span>{{ t('dashboard.cache') }} {{ formatTokens((stats?.today_cache_creation_tokens || 0) + (stats?.today_cache_read_tokens || 0)) }}</span>
        </p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric" style="--rx-i: 8">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.totalTokens') }}</p>
        <p class="rx-metric__value">{{ formatTokens(stats?.total_tokens || 0) }}</p>
        <p class="rx-metric__hint dash-metric__split">
          <span>{{ t('dashboard.input') }} {{ formatTokens(stats?.total_input_tokens || 0) }}</span>
          <span>{{ t('dashboard.output') }} {{ formatTokens(stats?.total_output_tokens || 0) }}</span>
          <span>{{ t('dashboard.cache') }} {{ formatTokens((stats?.total_cache_creation_tokens || 0) + (stats?.total_cache_read_tokens || 0)) }}</span>
        </p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric" style="--rx-i: 9">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.performance') }}</p>
        <p class="rx-metric__value">{{ formatTokens(stats?.rpm || 0) }}<small> RPM</small></p>
        <p class="rx-metric__hint">{{ formatTokens(stats?.tpm || 0) }} TPM</p>
      </article>

      <article class="rx-glass rx-metric rx-rise dash-metric" style="--rx-i: 10">
        <span class="rx-metric__index dash-metric__idx"></span>
        <p class="rx-metric__label">{{ t('dashboard.avgResponse') }}</p>
        <p class="rx-metric__value">{{ formatDuration(stats?.average_duration_ms || 0) }}</p>
        <p class="rx-metric__hint">{{ t('dashboard.averageTime') }}</p>
      </article>
    </div>

    <section v-if="!isSimple && platformCards.length > 0" class="rx-glass dash-platforms rx-rise" style="--rx-i: 11">
      <header class="rx-section-head">
        <span class="rx-section-head__index">02</span>
        <h2 class="rx-section-head__title">{{ t('dashboard.platformBreakdown') }}</h2>
        <span class="rx-section-head__hint">{{ t('dashboard.platformCount', { count: platformCount }) }}</span>
      </header>
      <div class="dash-plat-grid">
        <article
          v-for="item in platformCards"
          :key="item.platform"
          data-testid="platform-card"
          :data-platform="item.platform"
          class="dash-plat"
          :class="{ 'is-other': item.isOther }"
        >
          <div class="dash-plat__top">
            <span class="dash-plat__name">
              {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
            </span>
            <span class="dash-plat__cost" :title="t('dashboard.actual')">
              ${{ formatCost(item.total_actual_cost) }}
            </span>
          </div>
          <dl class="dash-plat__meta">
            <div>
              <dt>{{ t('dashboard.todayCost') }}</dt>
              <dd>${{ formatCost(item.today_actual_cost) }}</dd>
            </div>
            <div>
              <dt>{{ t('dashboard.requests') }}</dt>
              <dd>{{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}</dd>
            </div>
            <div>
              <dt>{{ t('dashboard.tokens') }}</dt>
              <dd>{{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '-' }}</dd>
            </div>
          </dl>

          <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="dash-quota">
            <p class="dash-quota__title">{{ t('dashboard.platformQuota.title') }}</p>
            <template v-for="w in (['daily', 'weekly', 'monthly'] as const)" :key="w">
              <div v-if="quotaVal(item.quota, `${w}_limit_usd`) != null" class="dash-quota__row">
                <template v-if="(quotaVal(item.quota, `${w}_limit_usd`) as number) === 0">
                  <div class="dash-quota__line">
                    <span>{{ t(`dashboard.platformQuota.${w}`) }}</span>
                    <span class="is-off">{{ t('dashboard.platformQuota.disabled') }}</span>
                  </div>
                  <div class="dash-quota__bar"><i class="is-off" style="width: 100%"></i></div>
                </template>
                <template v-else>
                  <div class="dash-quota__line">
                    <span>{{ t(`dashboard.platformQuota.${w}`) }}</span>
                    <span>
                      ${{ formatUsd((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0) }} / ${{ formatUsd(quotaVal(item.quota, `${w}_limit_usd`) as number) }}
                    </span>
                  </div>
                  <div class="dash-quota__bar">
                    <i
                      :class="quotaTone(calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number))"
                      :style="{ width: calcPercent((quotaVal(item.quota, `${w}_usage_usd`) as number) ?? 0, quotaVal(item.quota, `${w}_limit_usd`) as number) + '%' }"
                    />
                  </div>
                  <p v-if="quotaVal(item.quota, `${w}_window_resets_at`)" class="dash-quota__reset">
                    {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaVal(item.quota, `${w}_window_resets_at`) as string) }) }}
                  </p>
                </template>
              </div>
            </template>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlatformDashboardStats, UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()
const { t } = useI18n()

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  typesafe: 'TypeSafe / Jev',
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

const OTHER_THRESHOLD = 0.0001
const platformCards = computed<FusedPlatformCard[]>(() => {
  const byPlat = new Map<string, PlatformDashboardStats>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  const platforms = new Set<string>(byPlat.keys())
  for (const [platform, q] of byQuota) {
    if (hasAnyLimit(q)) platforms.add(platform)
  }

  const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'typesafe']
  const cards: FusedPlatformCard[] = []

  for (const p of platforms) {
    const stat = byPlat.get(p)
    cards.push({
      platform: p,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(p),
    })
  }

  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const sumTotal = cards.reduce((s, c) => s + c.total_actual_cost, 0)
  const sumToday = cards.reduce((s, c) => s + c.today_actual_cost, 0)
  const diffTotal = Math.max(0, total - sumTotal)
  const diffToday = Math.max(0, today - sumToday)

  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }

  return cards
})

const platformCount = computed(() => platformCards.value.filter((c) => !c.isOther).length)

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

function quotaVal(q: PlatformQuotaItem | undefined, key: QuotaField): PlatformQuotaItem[QuotaField] {
  return q?.[key]
}

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

function quotaTone(p: number): string {
  if (p >= 95) return 'is-hot'
  if (p >= 75) return 'is-warn'
  return 'is-ok'
}

const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
function formatUsd(n: number): string {
  if (!Number.isFinite(n)) return '0.00'
  return usdFormatter.format(n)
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(b)

const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (n: number) => {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
</script>

<style scoped>
.dash-metrics {
  counter-reset: dash-m;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

@media (min-width: 1024px) {
  .dash-metrics {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

.dash-metric {
  min-height: 8.2rem;
}

.dash-metric__idx::before {
  counter-increment: dash-m;
  content: counter(dash-m, decimal-leading-zero);
}

.dash-metric :deep(.rx-metric__value small) {
  margin-left: 0.15rem;
  font-size: 0.42em;
  font-weight: 700;
  letter-spacing: 0;
  color: var(--rx-faint);
}

.dash-metric__good {
  color: #15803d;
}

html.dark .dash-metric__good {
  color: #4ade80;
}

.dash-metric__split {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem 0.7rem;
}

.dash-metric--ink {
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  border-color: transparent;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12), inset 0 0 0 1px var(--rx-card-line), var(--rx-shadow);
}

.dash-metric--ink :deep(.rx-metric__label),
.dash-metric--ink :deep(.rx-metric__hint),
.dash-metric--ink :deep(.rx-metric__index) {
  color: inherit;
  opacity: 0.58;
}

.dash-metric--ink :deep(.rx-metric__value) {
  color: inherit;
}

.dash-metric--cost::after {
  content: '';
  position: absolute;
  left: 0.9rem;
  right: 0.9rem;
  bottom: 0.45rem;
  height: 6px;
  background: radial-gradient(circle, var(--rx-line-strong) 1.2px, transparent 1.3px);
  background-size: 8px 6px;
  opacity: 0.55;
}

.dash-platforms {
  padding: 1.25rem 1.25rem 1.35rem;
}

@media (min-width: 640px) {
  .dash-platforms {
    padding: 1.5rem 1.6rem 1.65rem;
  }
}

.dash-platforms .rx-section-head {
  margin-bottom: 1rem;
}

.dash-plat-grid {
  counter-reset: dash-p;
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.85rem;
}

@media (min-width: 640px) {
  .dash-plat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .dash-plat-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

.dash-plat {
  position: relative;
  overflow: hidden;
  padding: 0.95rem 1rem 1rem;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line);
  transition: transform 0.35s var(--rx-ease), border-color 0.25s ease, background 0.25s ease;
}

.dash-plat::after {
  counter-increment: dash-p;
  content: counter(dash-p, decimal-leading-zero);
  position: absolute;
  right: 0.8rem;
  bottom: 0.55rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 0.78rem;
  color: var(--rx-faint);
  opacity: 0.5;
}

.dash-plat:hover {
  border-color: var(--rx-line-strong);
  background: var(--rx-surface-hover);
  transform: translateY(-2px);
}

.dash-plat.is-other {
  border-style: dashed;
  background: transparent;
}

.dash-plat__top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.6rem;
}

.dash-plat__name {
  font-size: 0.92rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--rx-ink);
}

.dash-plat__cost {
  font-variant-numeric: tabular-nums;
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.dash-plat__meta {
  margin-top: 0.7rem;
  display: grid;
  gap: 0.3rem;
}

.dash-plat__meta div {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  font-size: 0.74rem;
}

.dash-plat__meta dt {
  color: var(--rx-faint);
}

.dash-plat__meta dd {
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink-2);
}

.dash-quota {
  margin-top: 0.85rem;
  padding-top: 0.75rem;
  border-top: 1px dashed var(--rx-line-strong);
}

.dash-quota__title {
  margin-bottom: 0.45rem;
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.dash-quota__row + .dash-quota__row {
  margin-top: 0.45rem;
}

.dash-quota__line {
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 0.72rem;
  color: var(--rx-ink-2);
}

.dash-quota__line .is-off {
  color: var(--rx-hot);
}

.dash-quota__bar {
  margin-top: 0.25rem;
  height: 5px;
  overflow: hidden;
  border-radius: 9999px;
  background: var(--rx-line-soft);
}

.dash-quota__bar i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #16a34a;
  transition: width 0.4s var(--rx-ease);
}

.dash-quota__bar i.is-warn { background: var(--rx-warn); }
.dash-quota__bar i.is-hot,
.dash-quota__bar i.is-off { background: var(--rx-hot); }

.dash-quota__reset {
  margin-top: 0.2rem;
  font-size: 0.62rem;
  color: var(--rx-faint);
}
</style>
