<template>
  <AppLayout>
    <RxStage>
      <RxPageHero
        :eyebrow="t('redeem.hero.eyebrow')"
        :lead="t('redeem.hero.titleLead')"
        :accent="t('redeem.hero.titleAccent')"
        :subtitle="t('redeem.description')"
      >
        <template #aside>
          <div class="rx-redeem-stats">
            <div class="rx-glass rx-redeem-stat">
              <span class="rx-redeem-stat__icon" aria-hidden="true"><Icon name="creditCard" size="md" /></span>
              <p class="rx-redeem-stat__label">{{ t('redeem.currentBalance') }}</p>
              <p class="rx-redeem-stat__value">${{ user?.balance?.toFixed(2) || '0.00' }}</p>
            </div>
            <div class="rx-glass rx-redeem-stat">
              <span class="rx-redeem-stat__icon" aria-hidden="true"><Icon name="bolt" size="md" /></span>
              <p class="rx-redeem-stat__label">{{ t('redeem.concurrency') }}</p>
              <p class="rx-redeem-stat__value">{{ user?.concurrency || 0 }} <small>{{ t('redeem.requests') }}</small></p>
            </div>
          </div>
        </template>
      </RxPageHero>

      <div class="grid items-start gap-6 lg:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
        <section class="rx-glass rx-redeem-panel rx-rise" style="--rx-i: 3">
          <header class="rx-section-head">
            <span class="rx-section-head__index">01</span>
            <h2 class="rx-section-head__title">{{ t('redeem.redeemCodeLabel') }}</h2>
            <span class="rx-section-head__hint">{{ t('redeem.redeemCodeHint') }}</span>
          </header>
          <form class="space-y-4" @submit.prevent="handleRedeem">
            <!-- 票券：左侧票根 + 齿孔分隔 + 代码区；成功时盖章，失败时轻晃 -->
            <div
              :key="shakeKey"
              class="rx-ticket"
              :class="{ 'is-filled': redeemCode.length > 0, 'is-error': !!errorMessage, 'is-busy': submitting }"
            >
              <div class="rx-ticket__stub" aria-hidden="true">
                <Icon name="gift" size="md" />
                <span>{{ t('redeem.hero.ticketStub') }}</span>
              </div>
              <div class="rx-ticket__body">
                <label for="code" class="sr-only">{{ t('redeem.redeemCodeLabel') }}</label>
                <input
                  id="code"
                  v-model="redeemCode"
                  type="text"
                  required
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="t('redeem.redeemCodePlaceholder')"
                  :disabled="submitting"
                  class="rx-ticket__input"
                />
                <span class="rx-ticket__scan" aria-hidden="true"></span>
              </div>
              <Transition name="rx-stamp">
                <span v-if="stampVisible" class="rx-stamp" aria-hidden="true">
                  <span class="rx-stamp__inner">{{ t('redeem.hero.stamp') }}</span>
                </span>
              </Transition>
            </div>

            <button type="submit" :disabled="!redeemCode || submitting" class="rx-redeem-cta">
              <span v-if="submitting" class="rx-redeem-cta__spinner" aria-hidden="true"></span>
              <Icon v-else name="checkCircle" size="md" />
              {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
            </button>
          </form>

          <!-- Success Message -->
          <Transition name="rx-fade">
            <div v-if="redeemResult" class="rx-result is-success">
              <div class="rx-result__head">
                <span class="rx-result__icon" aria-hidden="true"><Icon name="checkCircle" size="md" /></span>
                <div class="min-w-0 flex-1">
                  <h3 class="rx-result__title">{{ t('redeem.redeemSuccess') }}</h3>
                  <p class="rx-result__message">{{ redeemResult.message }}</p>
                </div>
                <p v-if="redeemResult.type === 'balance'" class="rx-result__value">+${{ redeemResult.value.toFixed(2) }}</p>
                <p v-else-if="redeemResult.type === 'concurrency'" class="rx-result__value">+{{ redeemResult.value }}</p>
              </div>
              <dl class="rx-result__rows">
                <div v-if="redeemResult.type === 'concurrency'">
                  <dt>{{ t('redeem.added') }}</dt>
                  <dd>{{ redeemResult.value }} {{ t('redeem.concurrentRequests') }}</dd>
                </div>
                <div v-else-if="redeemResult.type === 'subscription'">
                  <dt>{{ t('redeem.subscriptionAssigned') }}</dt>
                  <dd>
                    <span v-if="redeemResult.group_name">{{ redeemResult.group_name }}</span>
                    <span v-if="redeemResult.validity_days"> · {{ t('redeem.subscriptionDays', { days: redeemResult.validity_days }) }}</span>
                  </dd>
                </div>
                <div v-else-if="redeemResult.type === 'balance'">
                  <dt>{{ t('redeem.added') }}</dt>
                  <dd>${{ redeemResult.value.toFixed(2) }}</dd>
                </div>
                <div v-if="redeemResult.new_balance !== undefined">
                  <dt>{{ t('redeem.newBalance') }}</dt>
                  <dd>${{ redeemResult.new_balance.toFixed(2) }}</dd>
                </div>
                <div v-if="redeemResult.new_concurrency !== undefined">
                  <dt>{{ t('redeem.newConcurrency') }}</dt>
                  <dd>{{ redeemResult.new_concurrency }} {{ t('redeem.requests') }}</dd>
                </div>
              </dl>
            </div>
          </Transition>

          <!-- Error Message -->
          <Transition name="rx-fade">
            <div v-if="errorMessage" class="rx-result is-error">
              <div class="rx-result__head">
                <span class="rx-result__icon" aria-hidden="true"><Icon name="exclamationCircle" size="md" /></span>
                <div class="min-w-0 flex-1">
                  <h3 class="rx-result__title">{{ t('redeem.redeemFailed') }}</h3>
                  <p class="rx-result__message">{{ errorMessage }}</p>
                </div>
              </div>
            </div>
          </Transition>
        </section>

        <!-- Information Card -->
        <aside class="rx-glass rx-redeem-panel rx-rise" style="--rx-i: 4">
          <header class="rx-section-head">
            <span class="rx-section-head__index">02</span>
            <h2 class="rx-section-head__title">{{ t('redeem.aboutCodes') }}</h2>
          </header>
          <ol class="rx-rules">
            <li>{{ t('redeem.codeRule1') }}</li>
            <li>{{ t('redeem.codeRule2') }}</li>
            <li>
              {{ t('redeem.codeRule3') }}
              <span v-if="contactInfo" class="rx-rules__contact">{{ contactInfo }}</span>
            </li>
            <li>{{ t('redeem.codeRule4') }}</li>
          </ol>
        </aside>
      </div>

      <!-- Recent Activity -->
      <section class="rx-glass rx-redeem-panel rx-rise" style="--rx-i: 5">
        <header class="rx-section-head">
          <span class="rx-section-head__index">03</span>
          <h2 class="rx-section-head__title">{{ t('redeem.recentActivity') }}</h2>
        </header>

        <!-- Loading State -->
        <div v-if="loadingHistory" class="rx-timeline__skeleton" aria-busy="true">
          <span v-for="n in 3" :key="n"></span>
        </div>

        <!-- History List -->
        <ol v-else-if="history.length > 0" class="rx-timeline">
          <li
            v-for="(item, index) in history"
            :key="item.id"
            class="rx-timeline__item"
            :class="`kind-${historyKind(item)}`"
            :style="{ '--rx-i': index }"
          >
            <span class="rx-timeline__node" aria-hidden="true">
              <Icon v-if="isBalanceType(item.type)" name="dollar" size="sm" />
              <Icon v-else-if="isSubscriptionType(item.type)" name="badge" size="sm" />
              <Icon v-else name="bolt" size="sm" />
            </span>
            <div class="rx-timeline__main">
              <p class="rx-timeline__title">{{ getHistoryItemTitle(item) }}</p>
              <p class="rx-timeline__time">{{ formatDateTime(item.used_at) }}</p>
            </div>
            <div class="rx-timeline__side">
              <p class="rx-timeline__value" :class="{ 'is-negative': item.value < 0 && !isSubscriptionType(item.type) }">
                {{ formatHistoryValue(item) }}
              </p>
              <p v-if="!isAdminAdjustment(item.type)" class="rx-timeline__code">{{ item.code.slice(0, 8) }}...</p>
              <p v-else class="rx-timeline__code">{{ t('redeem.adminAdjustment') }}</p>
              <p v-if="item.notes" class="rx-timeline__note" :title="item.notes">{{ item.notes }}</p>
            </div>
          </li>
        </ol>

        <!-- Empty State -->
        <div v-else class="rx-timeline__empty">
          <span class="rx-timeline__empty-icon" aria-hidden="true"><Icon name="clock" size="xl" /></span>
          <p>{{ t('redeem.historyWillAppear') }}</p>
        </div>

        <footer class="rx-pager">
          <span class="rx-pager__total">{{ t('common.total') }}: {{ historyTotal }} {{ t('pagination.results') }}</span>
          <div class="rx-pager__controls">
            <label class="rx-pager__size">
              {{ t('pagination.perPage') }}
              <select
                v-model="historyPageSize"
                class="rx-pager__select"
                :disabled="loadingHistory || submitting"
                @change="fetchHistory(1)"
              >
                <option v-for="size in [20, 50, 100]" :key="size" :value="size">{{ size }}</option>
              </select>
            </label>
            <button
              type="button"
              class="rx-pager__btn"
              :disabled="loadingHistory || submitting || historyPage <= 1"
              @click="fetchHistory(historyPage - 1)"
            >{{ t('pagination.previous') }}</button>
            <button
              type="button"
              class="rx-pager__btn"
              :disabled="loadingHistory || submitting || historyPage * historyPageSize >= historyTotal"
              @click="fetchHistory(historyPage + 1)"
            >{{ t('pagination.next') }}</button>
          </div>
        </footer>
      </section>
    </RxStage>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, authAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import RxStage from '@/components/common/RxStage.vue'
import RxPageHero from '@/components/common/RxPageHero.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)

const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
  group_name?: string
  validity_days?: number
} | null>(null)
const errorMessage = ref('')
// 成功盖章与失败轻晃的动效状态
const stampVisible = ref(false)
const shakeKey = ref(0)
let stampTimer: ReturnType<typeof setTimeout> | undefined

function showStamp() {
  stampVisible.value = true
  if (stampTimer) clearTimeout(stampTimer)
  stampTimer = setTimeout(() => { stampVisible.value = false }, 2200)
}

onBeforeUnmount(() => {
  if (stampTimer) clearTimeout(stampTimer)
})

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
let historyRequest = 0
let loadedHistoryPageSize = 20
const contactInfo = ref('')

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const historyKind = (item: RedeemHistoryItem) => {
  if (isSubscriptionType(item.type)) return 'subscription'
  if (item.value < 0) return 'negative'
  return isBalanceType(item.type) ? 'balance' : 'concurrency'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : '-'
    return `${sign}$${Math.abs(item.value).toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async (page = 1) => {
  const request = ++historyRequest
  const pageSize = historyPageSize.value
  loadingHistory.value = true
  try {
    const result = await redeemAPI.getHistory(page, pageSize)
    if (request !== historyRequest) return
    history.value = result.items
    historyTotal.value = result.total
    historyPage.value = page
    historyPageSize.value = pageSize
    loadedHistoryPageSize = pageSize
  } catch (error) {
    if (request !== historyRequest) return
    historyPageSize.value = loadedHistoryPageSize
    appStore.showError(t('redeem.historyLoadFailed'))
    console.error('Failed to fetch history:', error)
  } finally {
    if (request === historyRequest) loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (!redeemCode.value.trim()) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null

  try {
    const result = await redeemAPI.redeem(redeemCode.value.trim())

    redeemResult.value = result
    showStamp()

    // Refresh user data to get updated balance/concurrency
    try {
      await authStore.refreshUser()
    } catch (error) {
      console.error('Failed to refresh user after redeem:', error)
      appStore.showWarning(t('redeem.userRefreshFailed'))
    }

    // If subscription type, immediately refresh subscription status
    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true) // force refresh
      } catch (error) {
        console.error('Failed to refresh subscriptions after redeem:', error)
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    // Clear the input
    redeemCode.value = ''

    // Refresh history
    await fetchHistory()

    // Show success toast
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('redeem.failedToRedeem')
    shakeKey.value++

    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
  } catch (error) {
    console.error('Failed to load contact info:', error)
  }
})
</script>


<style scoped>
/* ---------- 首屏统计 ---------- */
.rx-redeem-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

@media (min-width: 1024px) {
  .rx-redeem-stats {
    width: 24rem;
  }
}

.rx-redeem-stat {
  padding: 1rem 1.1rem;
}

.rx-redeem-stat__icon {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  margin-bottom: 0.75rem;
  border-radius: var(--rx-radius-sm);
  color: var(--rx-on-accent);
  background: var(--rx-accent);
}

.rx-redeem-stat__label {
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.rx-redeem-stat__value {
  margin-top: 0.2rem;
  font-size: 1.55rem;
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -0.035em;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink);
}

.rx-redeem-stat__value small {
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0;
  color: var(--rx-faint);
}

.rx-redeem-panel {
  padding: 1.4rem;
}

@media (min-width: 640px) {
  .rx-redeem-panel {
    padding: 1.75rem;
  }
}

/* ---------- 票券输入 ---------- */
.rx-ticket {
  --notch: 11px;
  position: relative;
  display: flex;
  min-height: 5.5rem;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-hover);
  border: 1px solid var(--rx-line);
  box-shadow: var(--rx-highlight), var(--rx-shadow);
  transition: border-color 0.3s ease, box-shadow 0.3s ease, transform 0.4s var(--rx-spring);
  /* 票根与正文交界处上下两个半圆缺口 */
  -webkit-mask:
    radial-gradient(circle var(--notch) at 5.25rem 0, transparent 98%, #000) top / 100% 51% no-repeat,
    radial-gradient(circle var(--notch) at 5.25rem 100%, transparent 98%, #000) bottom / 100% 51% no-repeat;
  mask:
    radial-gradient(circle var(--notch) at 5.25rem 0, transparent 98%, #000) top / 100% 51% no-repeat,
    radial-gradient(circle var(--notch) at 5.25rem 100%, transparent 98%, #000) bottom / 100% 51% no-repeat;
}

.rx-ticket:focus-within {
  border-color: var(--rx-line-strong);
  transform: translateY(-1px);
}

.rx-ticket__stub {
  position: relative;
  display: flex;
  width: 5.25rem;
  flex-shrink: 0;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  font-size: 0.6rem;
  font-weight: 800;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  transition: background 0.3s ease;
}

/* 齿孔虚线 */
.rx-ticket__stub::after {
  content: '';
  position: absolute;
  top: calc(var(--notch) + 4px);
  bottom: calc(var(--notch) + 4px);
  right: -1px;
  width: 2px;
  background-image: linear-gradient(var(--rx-surface-hover) 50%, transparent 50%);
  background-size: 2px 8px;
}

.rx-ticket__body {
  position: relative;
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  padding: 0 1.25rem;
  overflow: hidden;
}

.rx-ticket__input {
  width: 100%;
  min-width: 0;
  border: 0;
  padding: 0;
  background: transparent;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 1.25rem;
  font-weight: 600;
  letter-spacing: 0.14em;
  color: var(--rx-ink);
  outline: none;
  box-shadow: none;
}

.rx-ticket__input:focus {
  outline: none;
  box-shadow: none;
}

.rx-ticket__input::placeholder {
  font-family: inherit;
  font-size: 0.95rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--rx-faint);
}

.rx-ticket__input:disabled {
  opacity: 0.6;
}

/* 聚焦/提交时的一道扫描光 */
.rx-ticket__scan {
  pointer-events: none;
  position: absolute;
  inset: 0;
  background: linear-gradient(100deg, transparent 30%, var(--rx-accent-soft) 50%, transparent 70%);
  transform: translateX(-100%);
}

.rx-ticket:focus-within .rx-ticket__scan {
  animation: rx-scan 1.6s var(--rx-ease) 1;
}

.rx-ticket.is-busy .rx-ticket__scan {
  animation: rx-scan 1.1s linear infinite;
}

@keyframes rx-scan {
  to { transform: translateX(100%); }
}

.rx-ticket.is-error {
  border-color: color-mix(in srgb, var(--rx-hot) 45%, transparent);
  animation: rx-shake 0.45s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
}

@keyframes rx-shake {
  15%, 85% { transform: translateX(-2px); }
  30%, 70% { transform: translateX(4px); }
  45%, 55% { transform: translateX(-5px); }
}

/* 成功盖章 */
.rx-stamp {
  pointer-events: none;
  position: absolute;
  right: 1.25rem;
  top: 50%;
  display: grid;
  place-items: center;
  width: 4.6rem;
  height: 4.6rem;
  margin-top: -2.3rem;
  border-radius: 9999px;
  border: 2px solid #16a34a;
  color: #16a34a;
  transform: rotate(-14deg);
}

.rx-stamp__inner {
  display: grid;
  place-items: center;
  width: 3.8rem;
  height: 3.8rem;
  border-radius: 9999px;
  border: 1px dashed currentColor;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-align: center;
}

.rx-stamp-enter-active {
  animation: rx-stamp-in 0.5s cubic-bezier(0.2, 1.6, 0.4, 1) both;
}

.rx-stamp-leave-active {
  transition: opacity 0.5s ease;
}

.rx-stamp-leave-to {
  opacity: 0;
}

@keyframes rx-stamp-in {
  0% { opacity: 0; transform: rotate(-14deg) scale(1.8); }
  60% { opacity: 1; transform: rotate(-14deg) scale(0.92); }
  100% { opacity: 1; transform: rotate(-14deg) scale(1); }
}

.rx-redeem-cta {
  display: flex;
  width: 100%;
  height: 3.1rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: var(--rx-radius-md);
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  box-shadow: 0 1px 2px rgba(17, 17, 15, 0.14), 0 10px 24px -10px rgba(17, 17, 15, 0.35);
  transition: transform 0.4s var(--rx-spring), opacity 0.3s ease, box-shadow 0.3s ease;
}

.rx-redeem-cta:not(:disabled):hover {
  transform: translateY(-2px);
}

.rx-redeem-cta:not(:disabled):active {
  transform: scale(0.98);
}

.rx-redeem-cta:disabled {
  cursor: not-allowed;
  opacity: 0.35;
  box-shadow: none;
}

.rx-redeem-cta__spinner {
  width: 1rem;
  height: 1rem;
  border-radius: 9999px;
  border: 2px solid color-mix(in srgb, var(--rx-on-accent) 35%, transparent);
  border-top-color: var(--rx-on-accent);
  animation: rx-redeem-spin 0.8s linear infinite;
}

@keyframes rx-redeem-spin {
  to { transform: rotate(360deg); }
}

/* ---------- 结果 ---------- */
.rx-result {
  margin-top: 1.25rem;
  padding: 1rem 1.1rem;
  border-radius: var(--rx-radius-md);
  border: 1px solid var(--rx-line-soft);
}

.rx-result.is-success {
  --tone: #16a34a;
  background: color-mix(in srgb, #16a34a 7%, var(--rx-surface-hover));
}

.rx-result.is-error {
  --tone: var(--rx-hot);
  background: color-mix(in srgb, var(--rx-hot) 7%, var(--rx-surface-hover));
}

.rx-result__head {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.rx-result__icon {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 2rem;
  height: 2rem;
  border-radius: var(--rx-radius-sm);
  color: #fff;
  background: var(--tone);
}

.rx-result__title {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.rx-result__message {
  margin-top: 0.15rem;
  font-size: 0.82rem;
  color: var(--rx-muted);
}

.rx-result__value {
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.035em;
  font-variant-numeric: tabular-nums;
  color: var(--tone);
}

.rx-result__rows {
  display: grid;
  gap: 0.4rem;
  margin-top: 0.85rem;
  padding-top: 0.75rem;
  border-top: 1px dashed var(--rx-line-strong);
  font-size: 0.82rem;
}

.rx-result__rows div {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.rx-result__rows dt {
  color: var(--rx-muted);
}

.rx-result__rows dd {
  font-weight: 650;
  color: var(--rx-ink);
  font-variant-numeric: tabular-nums;
}

/* ---------- 规则 ---------- */
.rx-rules {
  counter-reset: rule;
  display: flex;
  flex-direction: column;
}

.rx-rules li {
  position: relative;
  counter-increment: rule;
  padding: 0.8rem 0 0.8rem 2.4rem;
  font-size: 0.87rem;
  line-height: 1.6;
  color: var(--rx-ink-2);
  border-top: 1px solid var(--rx-line-soft);
}

.rx-rules li:first-child {
  border-top: 0;
  padding-top: 0.2rem;
}

.rx-rules li::before {
  content: counter(rule, decimal-leading-zero);
  position: absolute;
  left: 0;
  top: 0.78rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 1rem;
  color: var(--rx-faint);
}

.rx-rules li:first-child::before {
  top: 0.18rem;
}

.rx-rules__contact {
  display: inline-flex;
  margin-left: 0.35rem;
  padding: 0.05rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 650;
  color: var(--rx-tint-ink);
  background: var(--rx-tint);
}

/* ---------- 时间线 ---------- */
.rx-timeline {
  position: relative;
  display: flex;
  flex-direction: column;
}

.rx-timeline::before {
  content: '';
  position: absolute;
  left: 1.05rem;
  top: 1.2rem;
  bottom: 1.2rem;
  width: 1px;
  background: linear-gradient(var(--rx-line-strong), var(--rx-line-soft));
}

.rx-timeline__item {
  --tone: var(--rx-ink);
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.9rem;
  padding: 0.7rem 0.6rem 0.7rem 0;
  border-radius: var(--rx-radius-sm);
  transition: background 0.25s ease;
  animation: rx-timeline-in 0.5s var(--rx-ease) both;
  animation-delay: calc(var(--rx-i, 0) * 40ms);
}

.rx-timeline__item:hover {
  background: var(--rx-surface-2);
}

@keyframes rx-timeline-in {
  from { opacity: 0; transform: translateX(-6px); }
  to { opacity: 1; transform: none; }
}

.kind-balance { --tone: #16a34a; }
.kind-concurrency { --tone: #2563eb; }
.kind-subscription { --tone: #7c3aed; }
.kind-negative { --tone: var(--rx-hot); }

.rx-timeline__node {
  position: relative;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 2.1rem;
  height: 2.1rem;
  border-radius: 9999px;
  color: var(--tone);
  background: color-mix(in srgb, var(--tone) 10%, var(--rx-surface-hover));
  border: 1px solid color-mix(in srgb, var(--tone) 22%, transparent);
  box-shadow: 0 0 0 4px var(--rx-surface-hover);
  transition: transform 0.4s var(--rx-spring);
}

.rx-timeline__item:hover .rx-timeline__node {
  transform: scale(1.08);
}

.rx-timeline__title {
  font-size: 0.88rem;
  font-weight: 650;
  color: var(--rx-ink);
}

.rx-timeline__time {
  margin-top: 0.1rem;
  font-size: 0.72rem;
  color: var(--rx-faint);
  font-variant-numeric: tabular-nums;
}

.rx-timeline__side {
  text-align: right;
}

.rx-timeline__value {
  font-size: 0.92rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink);
}

.rx-timeline__value.is-negative {
  color: var(--rx-hot);
}

.rx-timeline__code {
  margin-top: 0.1rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.7rem;
  color: var(--rx-faint);
}

.rx-timeline__note {
  max-width: 200px;
  margin-top: 0.2rem;
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.72rem;
  font-style: italic;
  color: var(--rx-muted);
}

.rx-timeline__skeleton {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.rx-timeline__skeleton span {
  height: 3.2rem;
  border-radius: var(--rx-radius-sm);
  background: linear-gradient(100deg, var(--rx-surface-2) 30%, var(--rx-surface-hover) 50%, var(--rx-surface-2) 70%);
  background-size: 200% 100%;
  animation: rx-timeline-shimmer 1.5s ease-in-out infinite;
}

@keyframes rx-timeline-shimmer {
  to { background-position: -200% 0; }
}

.rx-timeline__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 2.5rem 1rem;
  font-size: 0.85rem;
  color: var(--rx-muted);
}

.rx-timeline__empty-icon {
  display: grid;
  place-items: center;
  width: 3.5rem;
  height: 3.5rem;
  border-radius: 9999px;
  color: var(--rx-faint);
  background: var(--rx-surface-2);
  border: 1px dashed var(--rx-line-strong);
}

/* ---------- 分页 ---------- */
.rx-pager {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--rx-line-soft);
  font-size: 0.82rem;
}

@media (min-width: 640px) {
  .rx-pager {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

.rx-pager__total {
  color: var(--rx-faint);
  font-variant-numeric: tabular-nums;
}

.rx-pager__controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.rx-pager__size {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--rx-muted);
}

.rx-pager__select {
  height: 2.2rem;
  padding: 0 1.75rem 0 0.7rem;
  border-radius: var(--rx-radius-sm);
  font-size: 0.82rem;
  color: var(--rx-ink);
  background-color: var(--rx-surface-2);
  border: 1px solid var(--rx-line);
}

.rx-pager__btn {
  height: 2.2rem;
  padding: 0 0.9rem;
  border-radius: var(--rx-radius-sm);
  font-weight: 650;
  color: var(--rx-ink-2);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line);
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.rx-pager__btn:hover:not(:disabled) {
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  border-color: var(--rx-accent);
}

.rx-pager__btn:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.rx-fade-enter-active,
.rx-fade-leave-active {
  transition: opacity 0.35s ease, transform 0.35s var(--rx-ease);
}

.rx-fade-enter-from,
.rx-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
