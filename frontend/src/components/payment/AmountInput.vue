<template>
  <div class="rx-amount">
    <!-- Quick Amount Buttons -->
    <div>
      <p class="rx-amount__label">{{ t('payment.quickAmounts') }}</p>
      <div class="rx-amount__grid">
        <button
          v-for="(amt, index) in filteredAmounts"
          :key="amt"
          type="button"
          :class="[
            'rx-tile',
            modelValue === amt ? 'is-selected' : '',
            quoteFor(amt).percent > 0 ? 'is-promo' : '',
          ]"
          :style="{ '--rx-i': index }"
          :aria-pressed="modelValue === amt"
          :data-testid="`quick-amount-${amt}`"
          @pointermove="trackPointer"
          @click="selectAmount(amt)"
        >
          <span class="rx-tile__ring" aria-hidden="true"></span>
          <span class="rx-tile__spot" aria-hidden="true"></span>
          <!-- 促销价签：仅命中档位的金额显示 -->
          <span
            v-if="quoteFor(amt).percent > 0"
            class="rx-tile__badge"
            data-testid="quick-amount-bonus-badge"
          >{{ badgeText(amt) }}</span>
          <span class="rx-tile__value">
            <span class="rx-tile__symbol">{{ symbol }}</span>{{ amt }}
          </span>
          <!-- 配置了优惠阶梯时，所有按钮都显示第二行，保持高度一致：赠金显示到账 USD，折扣显示折后实付 -->
          <span
            v-if="showSecondLine"
            class="rx-tile__meta"
            data-testid="quick-amount-credited"
          >{{ secondLine(amt) }}</span>
          <span class="rx-tile__check" aria-hidden="true">
            <svg viewBox="0 0 16 16" fill="none"><path d="M3.5 8.5l3 3 6-7" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" /></svg>
          </span>
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
    <label class="rx-custom" :class="{ 'is-filled': customText !== '' && !isPreset }">
      <span class="rx-amount__label">{{ t('payment.customAmount') }}</span>
      <span class="rx-custom__field">
        <span class="rx-custom__symbol">{{ symbol }}</span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="rx-custom__input"
          @input="handleInput"
        />
        <span class="rx-custom__line" aria-hidden="true"></span>
      </span>
    </label>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RechargeBonusTier } from '@/types/payment'
import { formatRechargeBonusNumber, quoteRechargeBonus, type RechargeBonusMode } from '@/utils/rechargeBonus'
import { currencySymbol, formatPaymentAmount } from './currency'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  /** 充值优惠阶梯（按 min_amount 升序）；为空时不显示价签与第二行 */
  bonusTiers?: RechargeBonusTier[]
  /** 阶梯模式：bonus 赠金 / discount 折扣 */
  bonusMode?: RechargeBonusMode
  /** 充值倍率（1 支付币种 = multiplier USD），用于计算到账金额 */
  multiplier?: number
  /** 支付币种（折扣模式第二行实付金额的币种与精度） */
  currency?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  bonusTiers: () => [],
  bonusMode: 'bonus',
  multiplier: 1,
  currency: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const showSecondLine = computed(() => props.bonusTiers.length > 0)
const symbol = computed(() => currencySymbol(props.currency))
const isPreset = computed(() => props.modelValue !== null && filteredAmounts.value.includes(props.modelValue))

// 光斑跟随指针：写入按钮自身的 CSS 变量，避免整组重渲染
function trackPointer(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mx', `${e.clientX - rect.left}px`)
  el.style.setProperty('--my', `${e.clientY - rect.top}px`)
}

function currencyDigits(): number {
  if (!props.currency) return 2
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: props.currency }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function quoteFor(amt: number) {
  return quoteRechargeBonus(props.bonusTiers, amt, {
    multiplier: props.multiplier,
    mode: props.bonusMode,
    currencyDigits: currencyDigits(),
  })
}

// 价签文案：赠金「+20%」，折扣「20% OFF」
function badgeText(amt: number): string {
  const percent = formatRechargeBonusNumber(quoteFor(amt).percent)
  return props.bonusMode === 'discount' ? `${percent}% OFF` : `+${percent}%`
}

function secondLine(amt: number): string {
  const quote = quoteFor(amt)
  if (props.bonusMode === 'discount') {
    return t('payment.rechargeBonus.payShort', { amount: formatPaymentAmount(quote.payBase, props.currency) })
  }
  return t('payment.rechargeBonus.creditedShort', { amount: '$' + quote.credited.toFixed(2) })
}

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val)) {
    input.value = customText.value
    return
  }
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>

<style scoped>
.rx-amount {
  display: flex;
  flex-direction: column;
  gap: 1.75rem;
}

.rx-amount__label {
  display: block;
  margin-bottom: 0.75rem;
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--rx-faint, #94a3b8);
}

.rx-amount__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  padding-top: 0.5rem;
}

@media (min-width: 640px) {
  .rx-amount__grid {
    gap: 0.85rem;
  }
}

/* ---------- 金额磁贴 ---------- */
.rx-tile {
  --mx: 50%;
  --my: 50%;
  position: relative;
  isolation: isolate;
  display: flex;
  min-height: 4.6rem;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  padding: 0.85rem 1rem;
  border-radius: var(--rx-radius-md, 16px);
  text-align: left;
  color: var(--rx-ink, #0f172a);
  background: var(--rx-surface-2, rgba(248, 250, 252, 0.7));
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  box-shadow: var(--rx-highlight, none);
  transition:
    transform 0.45s var(--rx-spring, ease),
    background 0.3s ease,
    color 0.3s ease,
    border-color 0.3s ease,
    box-shadow 0.4s ease;
  animation: rx-tile-in 0.6s var(--rx-ease, ease) both;
  animation-delay: calc(var(--rx-i, 0) * 40ms + 120ms);
}

@keyframes rx-tile-in {
  from { opacity: 0; transform: translateY(10px) scale(0.97); }
  to { opacity: 1; transform: none; }
}

.rx-tile:hover:not(.is-selected) {
  transform: translateY(-2px);
  background: var(--rx-surface-hover, #fff);
  border-color: var(--rx-line-strong, rgba(15, 23, 42, 0.16));
  box-shadow: var(--rx-highlight, none), var(--rx-shadow, none);
}

.rx-tile:active {
  transform: translateY(0) scale(0.97);
  transition-duration: 0.12s;
}

.rx-tile:focus-visible {
  outline: 2px solid var(--rx-accent, #0d9488);
  outline-offset: 3px;
}

/* 指针光斑：用墨色极淡晕染，和纸面同调 */
.rx-tile__spot {
  pointer-events: none;
  position: absolute;
  inset: 0;
  z-index: -1;
  border-radius: inherit;
  background: radial-gradient(130px circle at var(--mx) var(--my), var(--rx-accent-soft, rgba(13, 148, 136, 0.14)), transparent 70%);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.rx-tile:hover .rx-tile__spot {
  opacity: 1;
}

.rx-tile__value {
  font-size: 1.35rem;
  font-weight: 750;
  line-height: 1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.rx-tile__symbol {
  margin-right: 0.12rem;
  font-size: 0.62em;
  font-weight: 600;
  vertical-align: 0.38em;
  color: var(--rx-faint, #94a3b8);
  transition: color 0.3s ease;
}

.rx-tile__meta {
  margin-top: 0.4rem;
  font-size: 0.7rem;
  line-height: 1.2;
  color: var(--rx-faint, #94a3b8);
  font-variant-numeric: tabular-nums;
  transition: color 0.3s ease;
}

.rx-tile.is-promo .rx-tile__meta {
  color: var(--rx-hot, #e11d48);
  font-weight: 650;
}

/* 选中：整块翻为墨色，与侧边栏激活项一致；边缘一道缓慢流动的高光 */
.rx-tile__ring {
  pointer-events: none;
  position: absolute;
  inset: -1px;
  border-radius: inherit;
  padding: 1px;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, rgba(255, 255, 255, 0.7) 50deg, transparent 110deg, transparent 360deg);
  -webkit-mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
  -webkit-mask-composite: xor;
  mask: linear-gradient(#000 0 0) content-box exclude, linear-gradient(#000 0 0);
  opacity: 0;
  transition: opacity 0.35s ease;
  animation: rx-ring-spin 3.6s linear infinite;
}

@keyframes rx-ring-spin {
  to { --rx-angle: 360deg; }
}

.rx-tile.is-selected {
  color: var(--rx-on-accent, #fff);
  background: var(--rx-accent, #0d9488);
  border-color: var(--rx-accent, #0d9488);
  box-shadow: var(--rx-shadow-lg, none);
  transform: translateY(-2px);
}

.rx-tile.is-selected .rx-tile__ring {
  opacity: 0.6;
}

.rx-tile.is-selected .rx-tile__symbol,
.rx-tile.is-selected .rx-tile__meta {
  color: inherit;
  opacity: 0.65;
}

.rx-tile.is-selected.is-promo .rx-tile__meta {
  opacity: 0.85;
}

.rx-tile__check {
  position: absolute;
  right: 0.7rem;
  bottom: 0.7rem;
  display: grid;
  place-items: center;
  width: 1.15rem;
  height: 1.15rem;
  border-radius: 9999px;
  color: var(--rx-accent, #0d9488);
  background: var(--rx-on-accent, #fff);
  transform: scale(0);
  transition: transform 0.4s var(--rx-spring, ease);
}

.rx-tile__check svg {
  width: 0.7rem;
  height: 0.7rem;
}

.rx-tile.is-selected .rx-tile__check {
  transform: scale(1);
}

/* 窄屏：磁贴变窄，收紧留白；选中由整块墨色表达，隐藏勾选以免压住到账金额 */
@media (max-width: 639px) {
  .rx-tile {
    padding: 0.75rem 0.8rem;
  }

  .rx-tile__value {
    font-size: 1.2rem;
  }

  .rx-tile__check {
    display: none;
  }
}

/* 促销价签：实色小票，轻微倾斜 */
.rx-tile__badge {
  pointer-events: none;
  position: absolute;
  top: -0.6rem;
  right: -0.35rem;
  z-index: 2;
  padding: 0.16rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.64rem;
  font-weight: 800;
  letter-spacing: 0.02em;
  white-space: nowrap;
  color: #fff;
  background: var(--rx-hot, #e11d48);
  box-shadow: 0 0 0 2px var(--rx-surface-hover, #fff), 0 6px 14px -6px var(--rx-hot, #e11d48);
  transform: rotate(5deg);
  transition: transform 0.4s var(--rx-spring, ease);
}

.rx-tile:hover .rx-tile__badge,
.rx-tile.is-selected .rx-tile__badge {
  transform: rotate(0deg) scale(1.06);
}

/* ---------- 自定义金额 ---------- */
.rx-custom {
  display: block;
}

.rx-custom__field {
  position: relative;
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  padding: 0.9rem 1.1rem 0.8rem;
  border-radius: var(--rx-radius-md, 16px);
  background: var(--rx-surface-2, rgba(248, 250, 252, 0.7));
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  transition: border-color 0.3s ease, box-shadow 0.3s ease, background 0.3s ease;
}

.rx-custom__field:focus-within {
  background: var(--rx-surface-hover, #fff);
  border-color: var(--rx-line-strong, rgba(15, 23, 42, 0.16));
  box-shadow: 0 0 0 4px var(--rx-accent-soft, rgba(13, 148, 136, 0.14));
}

.rx-custom__symbol {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--rx-faint, #94a3b8);
  transition: color 0.3s ease;
}

.rx-custom__field:focus-within .rx-custom__symbol,
.rx-custom.is-filled .rx-custom__symbol {
  color: var(--rx-ink, #0f172a);
}

.rx-custom__input {
  width: 100%;
  min-width: 0;
  border: 0;
  padding: 0;
  background: transparent;
  font-size: 1.6rem;
  font-weight: 750;
  letter-spacing: -0.025em;
  color: var(--rx-ink, #0f172a);
  font-variant-numeric: tabular-nums;
  outline: none;
  box-shadow: none;
}

.rx-custom__input:focus {
  outline: none;
  box-shadow: none;
}

.rx-custom__input::placeholder {
  font-size: 1rem;
  font-weight: 500;
  letter-spacing: 0;
  color: var(--rx-faint, #94a3b8);
  opacity: 0.8;
}

/* 聚焦时底部墨线描出 */
.rx-custom__line {
  pointer-events: none;
  position: absolute;
  left: 1.1rem;
  right: 1.1rem;
  bottom: 0.45rem;
  height: 1.5px;
  border-radius: 9999px;
  background: var(--rx-accent, #0d9488);
  transform: scaleX(0);
  transform-origin: left;
  transition: transform 0.6s var(--rx-ease, ease);
}

.rx-custom__field:focus-within .rx-custom__line {
  transform: scaleX(1);
}

@media (prefers-reduced-motion: reduce) {
  .rx-tile,
  .rx-tile__ring {
    animation: none;
  }
}
</style>
