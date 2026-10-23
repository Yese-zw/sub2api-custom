<template>
  <AppLayout>
    <RxStage>
      <div v-if="loading" class="dash-loading" aria-busy="true">
        <span v-for="n in 5" :key="n"></span>
      </div>

      <template v-else-if="stats">
        <RxPageHero
          :eyebrow="t('dashboard.hero.eyebrow')"
          :lead="t('dashboard.hero.titleLead')"
          :accent="t('dashboard.hero.titleAccent')"
          :subtitle="t('dashboard.hero.subtitle')"
        >
          <template #aside>
            <article v-if="!authStore.isSimpleMode" class="dash-wallet">
              <span class="dash-wallet__sheen" aria-hidden="true"></span>
              <div class="dash-wallet__top">
                <span class="dash-wallet__chip" aria-hidden="true"><i></i><i></i><i></i></span>
                <span class="dash-wallet__brand">{{ t('dashboard.hero.walletLabel') }}</span>
              </div>
              <p class="dash-wallet__label">{{ t('dashboard.balance') }}</p>
              <p class="dash-wallet__amount">${{ formatBalance(user?.balance || 0) }}</p>
              <p class="dash-wallet__hint">{{ t('common.available') }}</p>
              <div class="dash-wallet__foot">
                <span>{{ t('dashboard.hero.holder') }} · <b>{{ passHolder }}</b></span>
                <span>•••• {{ passSerial }}</span>
              </div>
            </article>
            <article v-else class="dash-pulse rx-glass">
              <p class="dash-pulse__label">{{ t('dashboard.hero.pulseLabel') }}</p>
              <p class="dash-pulse__value">{{ stats.today_requests || 0 }}</p>
              <p class="dash-pulse__hint">{{ t('dashboard.todayRequests') }}</p>
              <div class="dash-pulse__foot">
                <span>{{ t('dashboard.todayTokens') }}</span>
                <b>{{ formatTokens(stats.today_tokens || 0) }}</b>
              </div>
            </article>
          </template>
        </RxPageHero>

        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          :platform-quotas="platformQuotas"
        />
        <UserDashboardCharts
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          v-model:granularity="granularity"
          :loading="loadingCharts"
          :trend="trendData"
          :models="modelStats"
          @dateRangeChange="loadCharts"
          @granularityChange="loadCharts"
          @refresh="refreshAll"
        />
        <div class="grid grid-cols-1 items-start gap-6 lg:grid-cols-3">
          <div class="lg:col-span-2">
            <UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" />
          </div>
          <div class="lg:col-span-1">
            <UserDashboardQuickActions />
          </div>
        </div>
      </template>
    </RxStage>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import RxStage from '@/components/common/RxStage.vue'
import RxPageHero from '@/components/common/RxPageHero.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { formatDateLocalInput } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref('day')

const passHolder = computed(() => user.value?.username || user.value?.email || t('dashboard.title'))
const passSerial = computed(() => String(user.value?.id || 0).padStart(4, '0'))

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(b)

const formatTokens = (n: number) => {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}

const loadStats = async () => {
  loading.value = true
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}
const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const res = await Promise.all([
      usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as any }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value }),
    ])
    trendData.value = res[0].trend || []
    modelStats.value = res[1].models || []
  } catch (error) {
    console.error('Failed to load charts:', error)
  } finally {
    loadingCharts.value = false
  }
}
const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const res = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = res.items.slice(0, 5)
  } catch (error) {
    console.error('Failed to load recent usage:', error)
  } finally {
    loadingUsage.value = false
  }
}
const loadPlatformQuotas = async () => {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (error) {
    console.warn('Failed to load platform quotas:', error)
    platformQuotas.value = []
  }
}
const refreshAll = () => {
  loadStats()
  loadCharts()
  loadRecent()
  loadPlatformQuotas()
}

onMounted(() => {
  refreshAll()
})
</script>

<style scoped>
.dash-loading {
  display: grid;
  gap: 1rem;
  padding-top: 1.5rem;
}

.dash-loading span {
  height: 6.5rem;
  border-radius: var(--rx-radius-lg);
  background: linear-gradient(100deg, var(--rx-surface-2) 30%, var(--rx-surface-hover) 50%, var(--rx-surface-2) 70%);
  background-size: 200% 100%;
  animation: dash-shimmer 1.5s ease-in-out infinite;
}

.dash-loading span:first-child {
  height: 10rem;
}

@keyframes dash-shimmer {
  to { background-position: -200% 0; }
}

.dash-wallet {
  position: relative;
  overflow: hidden;
  width: 100%;
  padding: 1.35rem 1.5rem 1.2rem;
  border-radius: var(--rx-radius-lg);
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12), inset 0 0 0 1px var(--rx-card-line), var(--rx-shadow-lg);
}

@media (min-width: 1024px) {
  .dash-wallet {
    width: 22rem;
  }
}

.dash-wallet__sheen {
  pointer-events: none;
  position: absolute;
  inset: -40%;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, var(--rx-foil) 60deg, transparent 120deg, transparent 220deg, var(--rx-foil) 280deg, transparent 340deg);
  mix-blend-mode: soft-light;
  animation: dash-foil 18s linear infinite;
}

@keyframes dash-foil {
  to { --rx-angle: 360deg; }
}

.dash-wallet__top,
.dash-wallet__label,
.dash-wallet__amount,
.dash-wallet__hint,
.dash-wallet__foot {
  position: relative;
}

.dash-wallet__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.dash-wallet__chip {
  display: flex;
  gap: 3px;
}

.dash-wallet__chip i {
  width: 10px;
  height: 8px;
  border-radius: 2px;
  background: linear-gradient(135deg, #f4e7b5, #c9a227 55%, #f8f1cf);
}

.dash-wallet__brand,
.dash-wallet__label,
.dash-wallet__hint {
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  opacity: 0.6;
}

.dash-wallet__amount {
  margin-top: 0.35rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 2.55rem;
  font-weight: 500;
  line-height: 1;
  letter-spacing: -0.04em;
}

.dash-wallet__hint {
  margin-top: 0.4rem;
}

.dash-wallet__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
  margin-top: 1.1rem;
  padding-top: 0.9rem;
  border-top: 1px solid var(--rx-card-line);
  font-size: 0.78rem;
  opacity: 0.85;
}

.dash-wallet__foot b {
  font-weight: 800;
}

.dash-pulse {
  width: 100%;
  padding: 1.35rem 1.5rem 1.2rem;
}

@media (min-width: 1024px) {
  .dash-pulse {
    width: 22rem;
  }
}

.dash-pulse__label {
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.dash-pulse__value {
  margin-top: 0.4rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 2.6rem;
  font-weight: 500;
  line-height: 1;
  letter-spacing: -0.04em;
  color: var(--rx-ink);
}

.dash-pulse__hint {
  margin-top: 0.35rem;
  font-size: 0.8rem;
  color: var(--rx-muted);
}

.dash-pulse__foot {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 1.1rem;
  padding-top: 0.85rem;
  border-top: 1px dashed var(--rx-line-strong);
  font-size: 0.78rem;
  color: var(--rx-faint);
}

.dash-pulse__foot b {
  font-size: 1.05rem;
  font-weight: 800;
  color: var(--rx-ink);
}

@media (prefers-reduced-motion: reduce) {
  .dash-wallet__sheen,
  .dash-loading span {
    animation: none;
  }
}
</style>
