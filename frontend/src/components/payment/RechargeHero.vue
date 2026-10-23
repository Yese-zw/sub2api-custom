<template>
  <section class="rx-hero">
    <div
      ref="stageRef"
      class="rx-card-stage"
      @pointermove="onPointerMove"
      @pointerleave="onPointerLeave"
    >
      <div class="rx-card" :class="{ 'is-charged': projectedVisible }" :style="cardStyle">
        <div class="rx-card__holo" aria-hidden="true"></div>
        <div class="rx-card__sheen" aria-hidden="true"></div>
        <div class="rx-card__body">
          <div class="rx-card__brand-row">
            <span class="rx-card__chip" aria-hidden="true"><i></i><i></i><i></i></span>
            <span class="rx-card__brand">{{ brand }}</span>
          </div>

          <div class="rx-card__balance">
            <span class="rx-card__label">{{ t('payment.currentBalance') }}</span>
            <span class="rx-card__amount">
              <span class="rx-card__currency">$</span>{{ balanceText }}
            </span>
          </div>

          <Transition name="rx-projected">
            <div v-if="projectedVisible" class="rx-card__projected" data-testid="recharge-projected-balance">
              <span class="rx-card__delta">+${{ deltaText }}</span>
              <span class="rx-card__after">{{ t('payment.rechargeHero.afterTopUp') }} <b>${{ projectedText }}</b></span>
            </div>
          </Transition>

          <div class="rx-card__foot">
            <span class="rx-card__holder">{{ username || '—' }}</span>
            <span class="rx-card__serial">•••• {{ serial }}</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTweenedNumber } from '@/composables/useTweenedNumber'

const props = withDefaults(defineProps<{
  username?: string
  userId?: number
  brand?: string
  balance?: number
  /** 本次将到账的 USD（含赠金）；0 表示尚未选择金额 */
  credit?: number
}>(), {
  username: '',
  userId: 0,
  brand: 'Sub2API',
  balance: 0,
  credit: 0,
})

const { t } = useI18n()

const balanceTween = useTweenedNumber(toRef(props, 'balance'))
const creditTween = useTweenedNumber(toRef(props, 'credit'), 600)

const money = (v: number) => v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const balanceText = computed(() => money(balanceTween.value))
const deltaText = computed(() => money(creditTween.value))
const projectedText = computed(() => money(balanceTween.value + creditTween.value))
const projectedVisible = computed(() => props.credit > 0)
const serial = computed(() => String(props.userId || 0).padStart(4, '0').slice(-4))

const stageRef = ref<HTMLElement | null>(null)
const tilt = reactive({ rx: 0, ry: 0, px: 50, py: 30, active: false })
let raf = 0

function onPointerMove(e: PointerEvent) {
  if (e.pointerType === 'touch') return
  const el = stageRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const x = (e.clientX - rect.left) / rect.width
  const y = (e.clientY - rect.top) / rect.height
  if (raf) cancelAnimationFrame(raf)
  raf = requestAnimationFrame(() => {
    tilt.ry = (x - 0.5) * 10
    tilt.rx = (0.5 - y) * 6
    tilt.px = x * 100
    tilt.py = y * 100
    tilt.active = true
  })
}

function onPointerLeave() {
  if (raf) cancelAnimationFrame(raf)
  tilt.rx = 0
  tilt.ry = 0
  tilt.px = 50
  tilt.py = 30
  tilt.active = false
}

const cardStyle = computed(() => ({
  '--rx-rx': `${tilt.rx}deg`,
  '--rx-ry': `${tilt.ry}deg`,
  '--rx-px': `${tilt.px}%`,
  '--rx-py': `${tilt.py}%`,
  '--rx-sheen': tilt.active ? '1' : '0.55',
}))
</script>

<style scoped>
.rx-hero {
  padding: 0.25rem 0 0;
}

.rx-card-stage {
  position: relative;
  width: 100%;
  perspective: 1100px;
}

.rx-card {
  position: relative;
  overflow: hidden;
  border-radius: var(--rx-radius-lg);
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.14),
    inset 0 0 0 1px var(--rx-card-line),
    var(--rx-shadow-lg);
  transform: rotateX(var(--rx-rx)) rotateY(var(--rx-ry));
  transform-style: preserve-3d;
  transition: transform 0.6s var(--rx-ease), box-shadow 0.6s var(--rx-ease);
}

.rx-card-stage:hover .rx-card {
  transition: transform 0.12s linear, box-shadow 0.6s var(--rx-ease);
}

.rx-card__holo {
  position: absolute;
  inset: -40%;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, var(--rx-foil) 60deg, transparent 120deg, transparent 200deg, var(--rx-foil) 260deg, transparent 320deg);
  mix-blend-mode: soft-light;
  animation: rx-spin 18s linear infinite;
}

@keyframes rx-spin {
  to { --rx-angle: 360deg; }
}

.rx-card__holo::after {
  content: '';
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(115deg, rgba(255, 255, 255, 0.025) 0 1px, transparent 1px 4px);
}

.rx-card__sheen {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at var(--rx-px) var(--rx-py), rgba(255, 255, 255, 0.28), rgba(255, 255, 255, 0.04) 30%, transparent 55%);
  mix-blend-mode: soft-light;
  opacity: var(--rx-sheen);
  transition: opacity 0.4s var(--rx-ease);
}

.rx-card__body {
  position: relative;
  display: grid;
  gap: 0.85rem 1.5rem;
  padding: 1.15rem 1.35rem 1.1rem;
}

@media (min-width: 768px) {
  .rx-card__body {
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: end;
    padding: 1.25rem 1.6rem 1.2rem;
  }

  .rx-card__projected {
    grid-column: 2;
  }

  .rx-card__foot {
    grid-column: 3;
    grid-row: 1 / span 2;
    align-self: center;
    margin-top: 0;
    padding-top: 0;
    border-top: 0;
    flex-direction: column;
    align-items: flex-end;
    text-align: right;
  }
}

.rx-card__brand-row {
  display: flex;
  align-items: center;
  gap: 0.7rem;
}

.rx-card__chip {
  position: relative;
  display: grid;
  grid-template-rows: repeat(3, 1fr);
  gap: 2px;
  width: 2.3rem;
  height: 1.7rem;
  padding: 4px 5px;
  border-radius: 6px;
  background: linear-gradient(135deg, #efe6cf, #b9a578 55%, #e6dcc0);
  box-shadow: inset 0 0 0 1px rgba(60, 48, 20, 0.3);
}

.rx-card__chip i {
  border-top: 1px solid rgba(60, 48, 20, 0.35);
}

.rx-card__brand {
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  opacity: 0.8;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rx-card__balance {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.rx-card__label {
  font-size: 0.66rem;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  opacity: 0.55;
}

.rx-card__amount {
  margin-top: 0.2rem;
  font-size: clamp(1.8rem, 1.2rem + 1.6vw, 2.4rem);
  font-weight: 750;
  letter-spacing: -0.035em;
  line-height: 1.05;
  font-variant-numeric: tabular-nums;
}

.rx-card__currency {
  margin-right: 0.15rem;
  font-size: 0.6em;
  font-weight: 600;
  vertical-align: 0.42em;
  opacity: 0.6;
}

.rx-card__projected {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.78rem;
  font-variant-numeric: tabular-nums;
}

.rx-card__delta {
  padding: 0.1rem 0.5rem;
  border-radius: 9999px;
  font-weight: 700;
  color: #11110f;
  background: var(--rx-card-ink);
}

.rx-card__after {
  opacity: 0.7;
}

.rx-card__after b {
  font-weight: 700;
}

.rx-card__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem 1rem;
  margin-top: 0.15rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--rx-card-line);
  font-size: 0.75rem;
  letter-spacing: 0.1em;
  opacity: 0.7;
}

.rx-card__holder {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-transform: uppercase;
}

.rx-card__serial {
  flex-shrink: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.rx-projected-enter-active,
.rx-projected-leave-active {
  transition: opacity 0.45s var(--rx-ease), transform 0.45s var(--rx-ease), filter 0.45s var(--rx-ease);
}

.rx-projected-enter-from,
.rx-projected-leave-to {
  opacity: 0;
  transform: translateY(6px);
  filter: blur(4px);
}

@media (prefers-reduced-motion: reduce) {
  .rx-card__holo {
    animation: none;
  }
}
</style>
