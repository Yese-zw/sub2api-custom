<template>
  <div>
    <div
      data-testid="payment-method-grid"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4"
    >
      <button
        v-for="(method, index) in sortedMethods"
        :key="method.type"
        type="button"
        :title="methodLabel(method)"
        :disabled="!method.available"
        :aria-pressed="selected === method.type"
        :style="{ '--rx-brand': methodBrand(method.type), '--rx-i': index }"
        :class="[
          'rx-method min-w-0',
          !method.available
            ? 'is-disabled'
            : selected === method.type
              ? ['is-selected', methodSelectedClass(method.type)]
              : '',
        ]"
        @click="method.available && emit('select', method.type)"
      >
        <span class="rx-method__glow" aria-hidden="true"></span>
        <span class="rx-method__icon">
          <img :src="methodIcon(method.type)" :alt="methodLabel(method)" class="h-6 w-6 object-contain" />
        </span>
        <span class="flex min-w-0 flex-1 flex-col items-start leading-tight">
          <span data-testid="payment-method-label" class="rx-method__label block w-full truncate">
            {{ methodLabel(method) }}
          </span>
          <span v-if="method.fee_rate > 0" class="rx-method__fee">
            {{ t('payment.fee') }} {{ method.fee_rate }}%
          </span>
        </span>
        <span class="rx-method__radio" aria-hidden="true"></span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER, isBuiltInAlipayMethod, isBuiltInWxpayMethod } from './providerConfig'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import paymentIcon from '@/assets/icons/payment.svg'

export interface PaymentMethodOption {
  type: string
  display_name?: string
  fee_rate: number
  available: boolean
}

const props = defineProps<{
  methods: PaymentMethodOption[]
  selected: string
}>()

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()

const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
  airwallex: airwallexIcon,
  credit_card: paymentIcon,
}

const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function methodIcon(type: string): string {
  if (isBuiltInAlipayMethod(type)) return METHOD_ICONS.alipay
  if (isBuiltInWxpayMethod(type)) return METHOD_ICONS.wxpay
  if (type === 'airwallex') return METHOD_ICONS.airwallex
  return METHOD_ICONS[type] || paymentIcon
}

function methodLabel(method: PaymentMethodOption): string {
  return method.display_name || t(`payment.methods.${method.type}`, method.type)
}

// 选中态保留品牌色边框类（测试与主题依赖），光晕颜色走 --rx-brand
function methodSelectedClass(type: string): string {
  if (isBuiltInAlipayMethod(type)) return 'border-[#02A9F1]'
  if (isBuiltInWxpayMethod(type)) return 'border-[#09BB07]'
  if (type === 'stripe') return 'border-[#676BE5]'
  if (type === 'airwallex') return 'border-[#FF6B3D]'
  return 'border-primary-500'
}

function methodBrand(type: string): string {
  if (isBuiltInAlipayMethod(type)) return '#02A9F1'
  if (isBuiltInWxpayMethod(type)) return '#09BB07'
  if (type === 'stripe') return '#676BE5'
  if (type === 'airwallex') return '#FF6B3D'
  return '#14b8a6'
}
</script>

<style scoped>
.rx-method {
  position: relative;
  isolation: isolate;
  display: flex;
  height: 4rem;
  align-items: center;
  gap: 0.6rem;
  overflow: hidden;
  padding: 0 1.1rem 0 0.6rem;
  border-radius: var(--rx-radius-md, 16px);
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  color: var(--rx-ink, #0f172a);
  background: var(--rx-surface-2, rgba(248, 250, 252, 0.7));
  box-shadow: var(--rx-highlight, none);
  text-align: left;
  transition: transform 0.4s var(--rx-spring, ease), border-color 0.3s ease, box-shadow 0.4s ease, background 0.3s ease;
  animation: rx-method-in 0.55s var(--rx-ease, ease) both;
  animation-delay: calc(var(--rx-i, 0) * 50ms + 200ms);
}

@keyframes rx-method-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: none; }
}

.rx-method:not(.is-disabled):not(.is-selected):hover {
  transform: translateY(-2px);
  background: var(--rx-surface-hover, #fff);
  border-color: var(--rx-line-strong, rgba(15, 23, 42, 0.16));
}

.rx-method:not(.is-disabled):active {
  transform: scale(0.97);
}

.rx-method:focus-visible {
  outline: 2px solid var(--rx-accent, #0d9488);
  outline-offset: 3px;
}

/* 品牌色只作极淡的左侧晕染，不抢主色 */
.rx-method__glow {
  pointer-events: none;
  position: absolute;
  inset: 0;
  z-index: -1;
  background: radial-gradient(90% 140% at 0% 50%, color-mix(in srgb, var(--rx-brand) 12%, transparent), transparent 70%);
  opacity: 0;
  transition: opacity 0.4s ease;
}

.rx-method__icon {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: var(--rx-radius-sm, 11px);
  background: #fff;
  box-shadow: inset 0 0 0 1px var(--rx-line, rgba(15, 23, 42, 0.08));
  transition: transform 0.45s var(--rx-spring, ease);
}

.rx-method__label {
  font-size: 0.92rem;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.rx-method__fee {
  margin-top: 0.15rem;
  font-size: 0.66rem;
  letter-spacing: 0.04em;
  color: var(--rx-faint, #94a3b8);
}

/* 右上角选中指示点：绝对定位，不挤占标签宽度；点本身是品牌色 */
.rx-method__radio {
  position: absolute;
  top: 0.55rem;
  right: 0.55rem;
  width: 0.8rem;
  height: 0.8rem;
  border-radius: 9999px;
  border: 1.5px solid var(--rx-line-strong, rgba(15, 23, 42, 0.2));
  transition: border-color 0.3s ease;
}

.rx-method__radio::after {
  content: '';
  position: absolute;
  inset: 2px;
  border-radius: inherit;
  background: var(--rx-brand);
  transform: scale(0);
  transition: transform 0.4s var(--rx-spring, ease);
}

.rx-method.is-selected {
  border-color: var(--rx-accent, #0d9488);
  background: var(--rx-surface-hover, #fff);
  box-shadow: 0 0 0 1px var(--rx-accent, #0d9488), var(--rx-shadow, none);
}

.rx-method.is-selected .rx-method__glow {
  opacity: 1;
}

.rx-method.is-selected .rx-method__icon {
  transform: rotate(-5deg) scale(1.05);
}

.rx-method.is-selected .rx-method__radio {
  border-color: var(--rx-brand);
}

.rx-method.is-selected .rx-method__radio::after {
  transform: scale(1);
}

.rx-method.is-disabled {
  cursor: not-allowed;
  opacity: 0.42;
  filter: grayscale(0.6);
}

@media (prefers-reduced-motion: reduce) {
  .rx-method {
    animation: none;
  }
}
</style>
