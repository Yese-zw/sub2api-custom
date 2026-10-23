<template>
  <AppLayout>
    <RxStage>
      <!-- Filters -->
      <div class="rx-glass rx-orders-toolbar rx-rise" style="--rx-i: 3">
        <span class="rx-orders-toolbar__label">{{ t('payment.orders.status') }}</span>
        <Select v-model="currentFilter" :options="statusFilters" class="w-36" @change="handlePageChange(1)" />
        <span class="rx-orders-toolbar__count">{{ t('payment.ordersHero.showing', { count: orders.length }) }}</span>
        <button type="button" :disabled="loading" class="rx-icon-btn" :title="t('common.refresh')" @click="fetchOrders">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>

      <!-- Orders -->
      <section class="rx-glass rx-orders rx-rise" style="--rx-i: 4">
        <div v-if="loading && orders.length === 0" class="rx-orders__skeleton" aria-busy="true">
          <span v-for="n in 4" :key="n" class="rx-orders__skeleton-row"></span>
        </div>

        <div v-else-if="orders.length === 0" class="rx-orders__empty">
          <div class="rx-receipt-stack" aria-hidden="true">
            <span></span><span></span><span></span>
          </div>
          <p class="rx-orders__empty-title">{{ t('payment.orders.empty') }}</p>
          <p class="rx-orders__empty-hint">{{ t('payment.ordersHero.emptyHint') }}</p>
          <button type="button" class="rx-orders-cta" @click="router.push('/purchase')">
            {{ t('payment.result.backToRecharge') }}
          </button>
        </div>

        <div v-else :class="{ 'is-refreshing': loading }">
          <template v-for="group in orderGroups" :key="group.key">
            <h3 class="rx-orders__date">
              <span class="rx-orders__date-main">{{ group.label }}</span>
              <span class="rx-orders__date-count">{{ group.items.length }}</span>
            </h3>
            <ul class="rx-orders__list">
              <li
                v-for="(order, index) in group.items"
                :key="order.id"
                class="rx-order"
                :class="`tone-${statusTone(order.status)}`"
                :style="{ '--rx-i': index }"
              >
                <span class="rx-order__icon" aria-hidden="true">
                  <img :src="methodIcon(order.payment_type)" alt="" />
                </span>

                <div class="rx-order__main">
                  <p class="rx-order__title">
                    {{ order.order_type === 'subscription' ? t('payment.admin.subscriptionOrder') : t('payment.admin.balanceOrder') }}
                    <span class="rx-order__id">#{{ order.id }}</span>
                  </p>
                  <p class="rx-order__meta">
                    <span class="rx-order__no" :title="order.out_trade_no">{{ order.out_trade_no }}</span>
                    <span aria-hidden="true">·</span>
                    <span>{{ t('payment.methods.' + order.payment_type, order.payment_type) }}</span>
                    <span aria-hidden="true">·</span>
                    <span>{{ formatTime(order.created_at) }}</span>
                  </p>
                </div>

                <div class="rx-order__amount">
                  <p class="rx-order__pay">{{ currencySymbol(order.currency) }}{{ order.pay_amount.toFixed(2) }}</p>
                  <p v-if="order.amount !== order.pay_amount || (order.bonus_amount ?? 0) > 0" class="rx-order__credit">
                    {{ t('payment.orders.creditedAmount') }} ${{ order.amount.toFixed(2) }}
                    <span v-if="(order.bonus_amount ?? 0) > 0" class="rx-order__bonus">
                      {{ t('payment.orders.bonusIncluded', { amount: '$' + (order.bonus_amount ?? 0).toFixed(2) }) }}
                    </span>
                  </p>
                  <p v-else-if="order.fee_rate > 0" class="rx-order__credit">{{ t('payment.orders.fee') }} {{ order.fee_rate }}%</p>
                </div>

                <div class="rx-order__status">
                  <OrderStatusBadge :status="order.status" />
                </div>

                <div v-if="order.status === 'PENDING' || canRequestRefund(order)" class="rx-order__actions">
                  <button v-if="order.status === 'PENDING'" type="button" class="rx-order__action" @click="handleCancel(order.id)">
                    <Icon name="x" size="sm" />
                    <span>{{ t('payment.orders.cancel') }}</span>
                  </button>
                  <button v-if="canRequestRefund(order)" type="button" class="rx-order__action" @click="openRefundDialog(order)">
                    <Icon name="dollar" size="sm" />
                    <span>{{ t('payment.orders.requestRefund') }}</span>
                  </button>
                </div>
              </li>
            </ul>
          </template>
        </div>
      </section>

      <!-- Pagination -->
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </RxStage>

    <!-- Cancel Confirm Dialog -->
    <BaseDialog :show="!!cancelTargetId" :title="t('payment.orders.cancel')" width="narrow" @close="cancelTargetId = null">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('payment.confirmCancel') }}</p>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="cancelTargetId = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="actionLoading" @click="confirmCancel">{{ actionLoading ? t('common.processing') : t('payment.orders.cancel') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Refund Dialog -->
    <BaseDialog :show="!!refundTarget" :title="t('payment.orders.requestRefund')" @close="refundTarget = null">
      <div v-if="refundTarget" class="space-y-4">
        <div class="rounded-2xl border border-gray-100 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800/60">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">#{{ refundTarget.id }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('payment.orders.amount') }}</span>
            <span class="text-gray-900 dark:text-white">${{ refundTarget.amount.toFixed(2) }}</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('payment.refundReason') }}</label>
          <textarea v-model="refundReason" rows="3" class="input mt-1 w-full" :placeholder="t('payment.refundReasonPlaceholder')" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="refundTarget = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="actionLoading || !refundReason.trim()" @click="confirmRefund">{{ actionLoading ? t('common.processing') : t('payment.orders.requestRefund') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import RxStage from '@/components/common/RxStage.vue'
import { currencySymbol } from '@/components/payment/currency'
import { isBuiltInAlipayMethod, isBuiltInWxpayMethod } from '@/components/payment/providerConfig'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import paymentIcon from '@/assets/icons/payment.svg'
import type { OrderStatus } from '@/types/payment'

const { t, locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

function methodIcon(type: string): string {
  if (isBuiltInAlipayMethod(type)) return alipayIcon
  if (isBuiltInWxpayMethod(type)) return wxpayIcon
  if (type === 'stripe') return stripeIcon
  if (type === 'airwallex') return airwallexIcon
  return paymentIcon
}

// 状态色调：仅用于左侧细条与图标底色，标签颜色仍由 OrderStatusBadge 决定
function statusTone(status: OrderStatus): 'good' | 'warn' | 'info' | 'bad' | 'mute' {
  if (status === 'COMPLETED') return 'good'
  if (status === 'PENDING' || status === 'PAID' || status === 'RECHARGING') return 'warn'
  if (status === 'FAILED' || status === 'REFUND_FAILED') return 'bad'
  if (status === 'EXPIRED' || status === 'CANCELLED') return 'mute'
  return 'info'
}

function localeTag(): string | undefined {
  const raw = locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) return String((raw as { value?: string }).value || '') || undefined
  return undefined
}

function dayKey(date: Date): string {
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
}

// 按本地日期分组（列表本身已按创建时间倒序）
const orderGroups = computed(() => {
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  const groups: { key: string; label: string; items: PaymentOrder[] }[] = []
  for (const order of orders.value) {
    const date = new Date(order.created_at)
    const key = Number.isNaN(date.getTime()) ? 'unknown' : dayKey(date)
    let group = groups[groups.length - 1]
    if (!group || group.key !== key) {
      let label = t('common.unknown')
      if (key === dayKey(today)) label = t('payment.ordersHero.today')
      else if (key === dayKey(yesterday)) label = t('payment.ordersHero.yesterday')
      else if (key !== 'unknown') {
        label = date.toLocaleDateString(localeTag(), {
          month: 'long',
          day: 'numeric',
          ...(date.getFullYear() !== today.getFullYear() ? { year: 'numeric' } : {}),
        })
      }
      group = { key, label, items: [] }
      groups.push(group)
    }
    group.items.push(order)
  }
  return groups
})

function formatTime(dateStr: string): string {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleTimeString(localeTag(), { hour: '2-digit', minute: '2-digit' })
}

async function fetchOrders() {
  loading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentFilter.value || undefined,
    })
    orders.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) { pagination.page = page; fetchOrders() }
function handlePageSizeChange(size: number) { pagination.page_size = size; pagination.page = 1; fetchOrders() }

function handleCancel(orderId: number) { cancelTargetId.value = orderId }

async function confirmCancel() {
  if (!cancelTargetId.value) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess(t('common.success'))
    cancelTargetId.value = null
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) { refundTarget.value = order; refundReason.value = '' }

async function confirmRefund() {
  if (!refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess(t('common.success'))
    refundTarget.value = null
    refundReason.value = ''
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder): boolean {
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => { fetchOrders(); loadRefundEligibility() })
</script>

<style scoped>
.rx-orders-cta {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.7rem 1.1rem;
  border-radius: var(--rx-radius-sm);
  font-size: 0.85rem;
  font-weight: 700;
  white-space: nowrap;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  box-shadow: 0 1px 2px rgba(17, 17, 15, 0.14), 0 8px 20px -10px rgba(17, 17, 15, 0.4);
  transition: transform 0.4s var(--rx-spring), box-shadow 0.3s ease;
}

.rx-orders-cta svg {
  width: 1rem;
  height: 1rem;
  transition: transform 0.4s var(--rx-spring);
}

.rx-orders-cta:hover {
  transform: translateY(-1px);
}

.rx-orders-cta:hover svg {
  transform: translateX(3px);
}

.rx-orders-cta:focus-visible {
  outline: 2px solid var(--rx-accent);
  outline-offset: 3px;
}

/* ---------- 工具栏 ---------- */
.rx-orders-toolbar {
  position: relative;
  z-index: 5;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  padding: 0.7rem 0.75rem 0.7rem 1.25rem;
}

.rx-orders-toolbar__label {
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.rx-orders-toolbar__count {
  margin-left: auto;
  font-size: 0.78rem;
  color: var(--rx-faint);
  font-variant-numeric: tabular-nums;
}

.rx-icon-btn {
  display: grid;
  place-items: center;
  width: 2.4rem;
  height: 2.4rem;
  border-radius: var(--rx-radius-sm);
  color: var(--rx-muted);
  border: 1px solid var(--rx-line);
  background: var(--rx-surface-2);
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.rx-icon-btn:hover:not(:disabled) {
  color: var(--rx-ink);
  background: var(--rx-surface-hover);
  border-color: var(--rx-line-strong);
}

.rx-icon-btn:disabled {
  opacity: 0.5;
}

/* ---------- 列表 ---------- */
.rx-orders {
  padding: 0.5rem 1rem 1rem;
}

@media (min-width: 640px) {
  .rx-orders {
    padding: 0.75rem 1.5rem 1.5rem;
  }
}

.is-refreshing {
  opacity: 0.55;
  transition: opacity 0.3s ease;
}

.rx-orders__date {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
  margin: 1.25rem 0 0.6rem 0.25rem;
}

.rx-orders__date-main {
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 1.05rem;
  font-weight: 500;
  color: var(--rx-ink);
}

.rx-orders__date-count {
  padding: 0 0.45rem;
  border-radius: 9999px;
  font-size: 0.68rem;
  font-weight: 700;
  line-height: 1.4rem;
  color: var(--rx-muted);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
}

.rx-orders__list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.rx-order {
  --tone: var(--rx-faint);
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  grid-template-areas:
    'icon main amount'
    'icon status actions';
  align-items: center;
  gap: 0.35rem 0.9rem;
  overflow: hidden;
  padding: 0.85rem 1rem 0.85rem 1.1rem;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
  transition: transform 0.35s var(--rx-spring), background 0.25s ease, border-color 0.25s ease, box-shadow 0.3s ease;
  animation: rx-order-in 0.5s var(--rx-ease) both;
  animation-delay: calc(var(--rx-i, 0) * 45ms + 120ms);
}

@media (min-width: 768px) {
  .rx-order {
    grid-template-columns: auto minmax(0, 1fr) 12rem 7.5rem minmax(0, auto);
    grid-template-areas: 'icon main amount status actions';
    gap: 1rem;
  }
}

@keyframes rx-order-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: none; }
}

/* 左侧状态细条 */
.rx-order::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.9rem;
  bottom: 0.9rem;
  width: 3px;
  border-radius: 0 3px 3px 0;
  background: var(--tone);
  opacity: 0.85;
  transition: top 0.3s var(--rx-ease), bottom 0.3s var(--rx-ease);
}

.rx-order:hover {
  transform: translateY(-1px);
  background: var(--rx-surface-hover);
  border-color: var(--rx-line);
  box-shadow: var(--rx-shadow);
}

.rx-order:hover::before {
  top: 0.4rem;
  bottom: 0.4rem;
}

.tone-good { --tone: #16a34a; }
.tone-warn { --tone: #d97706; }
.tone-info { --tone: #6366f1; }
.tone-bad { --tone: #dc2626; }
.tone-mute { --tone: var(--rx-line-strong); }

.rx-order__icon {
  grid-area: icon;
  display: grid;
  place-items: center;
  width: 2.6rem;
  height: 2.6rem;
  border-radius: var(--rx-radius-sm);
  background: #fff;
  box-shadow: inset 0 0 0 1px var(--rx-line);
}

.rx-order__icon img {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
}

.rx-order__main {
  grid-area: main;
  min-width: 0;
}

.rx-order__title {
  display: flex;
  align-items: baseline;
  gap: 0.45rem;
  font-size: 0.92rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.rx-order__id {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.72rem;
  font-weight: 500;
  color: var(--rx-faint);
}

.rx-order__meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.4rem;
  margin-top: 0.2rem;
  font-size: 0.74rem;
  color: var(--rx-faint);
  white-space: nowrap;
}

.rx-order__no {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

/* 窄屏：订单号独占一行，避免被金额列挤成一个字符 */
@media (max-width: 767px) {
  .rx-order__meta {
    flex-wrap: wrap;
    row-gap: 0.1rem;
  }

  .rx-order {
    column-gap: 0.7rem;
    padding-right: 0.8rem;
  }

  .rx-order__icon {
    width: 2.2rem;
    height: 2.2rem;
  }

  .rx-order__no {
    flex-basis: 100%;
    font-size: 0.68rem;
    letter-spacing: -0.02em;
  }

  .rx-order__no + span {
    display: none;
  }

  .rx-order__amount {
    max-width: 7.5rem;
  }

  .rx-order__credit {
    white-space: normal;
  }
}

.rx-order__amount {
  grid-area: amount;
  text-align: right;
}

.rx-order__pay {
  font-size: 1.1rem;
  font-weight: 800;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink);
}

.rx-order__credit {
  margin-top: 0.1rem;
  white-space: nowrap;
  font-size: 0.7rem;
  color: var(--rx-faint);
  font-variant-numeric: tabular-nums;
}

.rx-order__bonus {
  display: table;
  margin: 0.2rem 0 0 auto;
  padding: 0 0.4rem;
  border-radius: 9999px;
  font-size: 0.66rem;
  line-height: 1.15rem;
  font-weight: 700;
  color: var(--rx-hot);
  background: color-mix(in srgb, var(--rx-hot) 9%, transparent);
}

.rx-order__status {
  grid-area: status;
}

@media (min-width: 768px) {
  .rx-order__status {
    justify-self: center;
  }
}

.rx-order__actions {
  grid-area: actions;
  display: flex;
  justify-content: flex-end;
  gap: 0.35rem;
}

.rx-order__action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.35rem 0.65rem;
  border-radius: 9999px;
  font-size: 0.72rem;
  font-weight: 650;
  color: var(--rx-ink-2);
  border: 1px solid var(--rx-line);
  background: var(--rx-surface-hover);
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.rx-order__action:hover {
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  border-color: var(--rx-accent);
}

/* ---------- 骨架与空态 ---------- */
.rx-orders__skeleton {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-top: 1rem;
}

.rx-orders__skeleton-row {
  position: relative;
  display: block;
  height: 4.25rem;
  overflow: hidden;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
}

.rx-orders__skeleton-row::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(100deg, transparent 20%, var(--rx-surface-hover) 50%, transparent 80%);
  opacity: 0.7;
  transform: translateX(-100%);
  animation: rx-orders-shimmer 1.6s ease-in-out infinite;
}

@keyframes rx-orders-shimmer {
  to { transform: translateX(100%); }
}

.rx-orders__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 3.5rem 1rem 3rem;
  text-align: center;
}

.rx-orders__empty-title {
  margin-top: 1.5rem;
  font-size: 1rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.rx-orders__empty-hint {
  margin: 0.35rem 0 1.25rem;
  font-size: 0.85rem;
  color: var(--rx-muted);
}

/* 空态插画：三张错落的小票 */
.rx-receipt-stack {
  position: relative;
  width: 5.5rem;
  height: 6rem;
}

.rx-receipt-stack span {
  position: absolute;
  inset: 0;
  border-radius: 10px;
  background:
    linear-gradient(var(--rx-line) 0 0) 0.8rem 1.2rem / 55% 2px no-repeat,
    linear-gradient(var(--rx-line-soft) 0 0) 0.8rem 2rem / 70% 2px no-repeat,
    linear-gradient(var(--rx-line-soft) 0 0) 0.8rem 2.6rem / 40% 2px no-repeat,
    linear-gradient(var(--rx-line-strong) 0 0) 0.8rem calc(100% - 1.1rem) / 35% 3px no-repeat,
    var(--rx-surface-hover);
  border: 1px solid var(--rx-line);
  box-shadow: var(--rx-shadow);
  -webkit-mask: radial-gradient(circle 5px at 50% 100%, transparent 98%, #000) bottom / 12px 100% repeat-x;
  mask: radial-gradient(circle 5px at 50% 100%, transparent 98%, #000) bottom / 12px 100% repeat-x;
}

.rx-receipt-stack span:nth-child(1) { transform: rotate(-12deg) translateX(-14px); opacity: 0.5; }
.rx-receipt-stack span:nth-child(2) { transform: rotate(8deg) translateX(12px); opacity: 0.75; }
.rx-receipt-stack span:nth-child(3) { animation: rx-receipt-bob 4s ease-in-out infinite; }

@keyframes rx-receipt-bob {
  0%, 100% { transform: translateY(0) rotate(-2deg); }
  50% { transform: translateY(-5px) rotate(1deg); }
}
</style>
