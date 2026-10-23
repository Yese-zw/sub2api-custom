<template>
  <span
    class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium"
    :class="statusClass"
  >
    <span class="order-status-dot" :class="{ 'is-live': isLive }" aria-hidden="true"></span>
    {{ statusLabel }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OrderStatus } from '@/types/payment'

const props = defineProps<{
  status: OrderStatus
}>()

const { t } = useI18n()

const statusMap: Record<OrderStatus, { key: string; class: string }> = {
  PENDING: { key: 'payment.status.pending', class: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400' },
  PAID: { key: 'payment.status.paid', class: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400' },
  RECHARGING: { key: 'payment.status.recharging', class: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400' },
  COMPLETED: { key: 'payment.status.completed', class: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' },
  EXPIRED: { key: 'payment.status.expired', class: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400' },
  CANCELLED: { key: 'payment.status.cancelled', class: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400' },
  FAILED: { key: 'payment.status.failed', class: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400' },
  REFUND_REQUESTED: { key: 'payment.status.refund_requested', class: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400' },
  REFUNDING: { key: 'payment.status.refunding', class: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400' },
  REFUND_PENDING: { key: 'payment.status.refund_pending', class: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400' },
  REFUNDED: { key: 'payment.status.refunded', class: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400' },
  PARTIALLY_REFUNDED: { key: 'payment.status.partially_refunded', class: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400' },
  REFUND_FAILED: { key: 'payment.status.refund_failed', class: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400' },
}

const statusLabel = computed(() => {
  const entry = statusMap[props.status]
  return entry ? t(entry.key) : props.status
})

// 进行中的状态：圆点呼吸
const isLive = computed(() => ['PENDING', 'PAID', 'RECHARGING', 'REFUNDING', 'REFUND_PENDING'].includes(props.status))

const statusClass = computed(() => {
  const entry = statusMap[props.status]
  return entry?.class ?? 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400'
})
</script>

<style scoped>
.order-status-dot {
  position: relative;
  width: 6px;
  height: 6px;
  flex-shrink: 0;
  border-radius: 9999px;
  background: currentColor;
}

.order-status-dot.is-live::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: currentColor;
  animation: order-status-ping 1.8s cubic-bezier(0, 0, 0.2, 1) infinite;
}

@keyframes order-status-ping {
  0% { transform: scale(1); opacity: 0.6; }
  80%, 100% { transform: scale(2.6); opacity: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .order-status-dot.is-live::after {
    animation: none;
  }
}
</style>
