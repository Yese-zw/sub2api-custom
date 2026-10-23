<template>
  <section class="rx-hero">
    <div class="rx-hero__copy">
      <span class="rx-eyebrow rx-rise" style="--rx-i: 0">
        <span class="rx-eyebrow__dot" aria-hidden="true"></span>
        {{ t('payment.rechargeHero.eyebrow') }}
      </span>
      <h1 class="rx-title rx-rise" style="--rx-i: 1">
        <span class="rx-title__lead">{{ t('payment.rechargeHero.titleLead') }}</span>
        <span class="rx-title__glow">{{ t('payment.rechargeHero.titleAccent') }}</span>
      </h1>
      <p class="rx-sub rx-rise" style="--rx-i: 2">{{ t('payment.rechargeHero.subtitle') }}</p>

      <ol v-if="showSteps" class="rx-steps rx-rise" style="--rx-i: 3" :aria-label="t('payment.rechargeHero.progress')">
        <li
          v-for="(label, index) in stepLabels"
          :key="label"
          class="rx-step"
          :class="{ 'is-done': step > index, 'is-current': step === index }"
        >
          <span class="rx-step__mark" aria-hidden="true">
            <svg v-if="step > index" viewBox="0 0 16 16" fill="none"><path d="M3.5 8.5l3 3 6-7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" /></svg>
            <template v-else>{{ index + 1 }}</template>
          </span>
          <span class="rx-step__label">{{ label }}</span>
        </li>
      </ol>
    </div>

    <!-- 全息余额卡：指针驱动 3D 倾斜 + 镜面高光 -->
    <div
      ref="stageRef"
      class="rx-card-stage rx-rise"
      style="--rx-i: 1"
      @pointermove="onPointerMove"
      @pointerleave="onPointerLeave"
    >
      <div class="rx-card" :class="{ 'is-charged': projectedVisible }" :style="cardStyle">
        <div class="rx-card__holo" aria-hidden="true"></div>
        <div class="rx-card__sheen" aria-hidden="true"></div>
        <div class="rx-card__body">
          <div class="rx-card__top">
            <span class="rx-card__chip" aria-hidden="true"><i></i><i></i><i></i></span>
            <span class="rx-card__brand">{{ brand }}</span>
            <svg class="rx-card__wave" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M8.5 7.5a6 6 0 010 9M12 5a9.5 9.5 0 010 14M15.5 2.5a13 13 0 010 19" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
            </svg>
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
              <svg viewBox="0 0 20 20" fill="none" aria-hidden="true"><path d="M4 10h11m0 0l-4-4m4 4l-4 4" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" /></svg>
              <span class="rx-card__after">{{ t('payment.rechargeHero.afterTopUp') }} <b>${{ projectedText }}</b></span>
            </div>
          </Transition>

          <div class="rx-card__foot">
            <span class="rx-card__holder">{{ username || '—' }}</span>
            <span class="rx-card__serial">•••• {{ serial }}</span>
          </div>
        </div>
      </div>
      <div class="rx-card-floor" aria-hidden="true"></div>
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
  /** 进度：0 选金额 / 1 选支付方式 / 2 待确认 / 3 就绪 */
  step?: number
  showSteps?: boolean
}>(), {
  username: '',
  userId: 0,
  brand: 'Sub2API',
  balance: 0,
  credit: 0,
  step: 0,
  showSteps: true,
})

const { t } = useI18n()

const stepLabels = computed(() => [
  t('payment.rechargeHero.stepAmount'),
  t('payment.rechargeHero.stepMethod'),
  t('payment.rechargeHero.stepConfirm'),
])

const balanceTween = useTweenedNumber(toRef(props, 'balance'))
const creditTween = useTweenedNumber(toRef(props, 'credit'), 600)

const money = (v: number) => v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const balanceText = computed(() => money(balanceTween.value))
const deltaText = computed(() => money(creditTween.value))
const projectedText = computed(() => money(balanceTween.value + creditTween.value))
const projectedVisible = computed(() => props.credit > 0)
const serial = computed(() => String(props.userId || 0).padStart(4, '0').slice(-4))

// ---------- 指针倾斜 ----------
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
    tilt.ry = (x - 0.5) * 16
    tilt.rx = (0.5 - y) * 12
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
  display: grid;
  gap: 2rem;
  align-items: center;
  padding: 1.5rem 0.25rem 0.5rem;
}

@media (min-width: 1024px) {
  .rx-hero {
    grid-template-columns: minmax(0, 1fr) minmax(0, 22rem);
    gap: 2.5rem;
    padding: 2.25rem 0.5rem 1rem;
  }
}

@media (min-width: 1280px) {
  .rx-hero {
    grid-template-columns: minmax(0, 1fr) minmax(0, 25rem);
    gap: 3.5rem;
  }
}

/* ---------- 文案 ---------- */
.rx-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.8rem 0.35rem 0.65rem;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--rx-muted);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
  -webkit-backdrop-filter: var(--rx-blur);
  backdrop-filter: var(--rx-blur);
}

.rx-eyebrow__dot {
  position: relative;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: var(--rx-ink);
}

.rx-eyebrow__dot::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: var(--rx-ink);
  animation: rx-ping 2.4s cubic-bezier(0, 0, 0.2, 1) infinite;
}

@keyframes rx-ping {
  0% { transform: scale(1); opacity: 0.45; }
  80%, 100% { transform: scale(3); opacity: 0; }
}

.rx-title {
  margin-top: 1.1rem;
  font-size: clamp(2rem, 1.3rem + 3vw, 3.4rem);
  line-height: 1.06;
  font-weight: 800;
  letter-spacing: -0.04em;
  color: var(--rx-ink);
}

.rx-title__lead {
  display: block;
}

@media (min-width: 480px) {
  .rx-title__lead {
    white-space: nowrap;
  }
}

/* 编辑型强调：衬线斜体 + 手写式下划线，随入场描出 */
.rx-title__glow {
  position: relative;
  display: inline-block;
  padding-bottom: 0.12em;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-weight: var(--rx-display-weight);
  letter-spacing: -0.02em;
  color: var(--rx-ink);
}

.rx-title__glow::after {
  content: '';
  position: absolute;
  left: 0.02em;
  right: 0.02em;
  bottom: 0.02em;
  height: 0.2em;
  border-radius: 9999px;
  background: var(--rx-tint);
  z-index: -1;
  transform: scaleX(0);
  transform-origin: left;
  animation: rx-underline 1.1s var(--rx-ease) 0.45s forwards;
}

@keyframes rx-underline {
  to { transform: scaleX(1); }
}

.rx-sub {
  margin-top: 1rem;
  max-width: 34rem;
  font-size: 0.95rem;
  line-height: 1.7;
  color: var(--rx-muted);
}

/* ---------- 进度：当前项与侧边栏激活项同样的墨色胶囊 ---------- */
.rx-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 1.75rem;
}

.rx-step {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.38rem 0.9rem 0.38rem 0.4rem;
  border-radius: 9999px;
  font-size: 0.8rem;
  font-weight: 650;
  color: var(--rx-faint);
  border: 1px solid var(--rx-line);
  background: transparent;
  transition: color 0.4s var(--rx-ease), background 0.4s var(--rx-ease), border-color 0.4s var(--rx-ease), box-shadow 0.4s var(--rx-ease);
}

.rx-step__mark {
  display: grid;
  place-items: center;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 700;
  border: 1px solid var(--rx-line-strong);
  transition: all 0.45s var(--rx-spring);
}

@media (max-width: 479px) {
  .rx-steps {
    flex-wrap: nowrap;
    gap: 0.35rem;
  }

  .rx-step {
    gap: 0.35rem;
    padding: 0.3rem 0.6rem 0.3rem 0.3rem;
    font-size: 0.72rem;
    white-space: nowrap;
  }

  .rx-step__mark {
    width: 1.25rem;
    height: 1.25rem;
  }
}

.rx-step__mark svg {
  width: 0.85rem;
  height: 0.85rem;
}

.rx-step.is-current {
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  border-color: var(--rx-accent);
  box-shadow: var(--rx-shadow);
}

.rx-step.is-current .rx-step__mark {
  color: var(--rx-accent);
  border-color: transparent;
  background: var(--rx-on-accent);
}

.rx-step.is-done {
  color: var(--rx-ink-2);
  background: var(--rx-surface);
}

.rx-step.is-done .rx-step__mark {
  color: var(--rx-tint-ink);
  border-color: transparent;
  background: var(--rx-tint);
}

/* ---------- 余额卡 ---------- */
.rx-card-stage {
  position: relative;
  width: 100%;
  max-width: 25rem;
  margin-inline: auto;
  perspective: 1100px;
}

.rx-card {
  position: relative;
  aspect-ratio: 1.586;
  border-radius: calc(var(--rx-radius-lg) - 2px);
  overflow: hidden;
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.14),
    inset 0 0 0 1px var(--rx-card-line),
    var(--rx-shadow-lg);
  transform: rotateX(var(--rx-rx)) rotateY(var(--rx-ry));
  transform-style: preserve-3d;
  transition: transform 0.6s var(--rx-ease), box-shadow 0.6s var(--rx-ease);
  animation: rx-float 7s ease-in-out infinite;
}

.rx-card-stage:hover .rx-card {
  transition: transform 0.12s linear, box-shadow 0.6s var(--rx-ease);
  animation-play-state: paused;
}

@keyframes rx-float {
  0%, 100% { translate: 0 0; }
  50% { translate: 0 -5px; }
}

/* 金属箔：缓慢旋转的锥形光，叠加 soft-light，只在卡面留下微光 */
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

/* 细纹理：卡面极淡的斜向拉丝 */
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
  display: flex;
  height: 100%;
  flex-direction: column;
  padding: 1.35rem 1.5rem 1.2rem;
  transform: translateZ(30px);
}

.rx-card__top {
  display: flex;
  align-items: center;
  gap: 0.75rem;
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

.rx-card__wave {
  width: 1.3rem;
  height: 1.3rem;
  margin-left: auto;
  opacity: 0.55;
}

.rx-card__balance {
  display: flex;
  flex-direction: column;
  margin-top: auto;
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
  font-size: clamp(1.8rem, 1.2rem + 2.2vw, 2.4rem);
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
  align-items: center;
  gap: 0.45rem;
  margin-top: 0.45rem;
  font-size: 0.78rem;
  font-variant-numeric: tabular-nums;
}

.rx-card__projected svg {
  width: 0.95rem;
  height: 0.95rem;
  opacity: 0.55;
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
  gap: 1rem;
  margin-top: 0.9rem;
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

.rx-card-floor {
  position: absolute;
  left: 14%;
  right: 14%;
  bottom: -1.2rem;
  height: 1.6rem;
  border-radius: 50%;
  background: radial-gradient(closest-side, rgba(17, 17, 15, 0.22), transparent);
  filter: blur(10px);
  animation: rx-floor 7s ease-in-out infinite;
}

html.dark .rx-card-floor {
  background: radial-gradient(closest-side, rgba(0, 0, 0, 0.6), transparent);
}

@keyframes rx-floor {
  0%, 100% { transform: scaleX(1); opacity: 1; }
  50% { transform: scaleX(0.88); opacity: 0.7; }
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
</style>
