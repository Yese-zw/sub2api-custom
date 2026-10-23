<template>
  <!-- 用量概览便当格：消费（墨色大格）/ 请求 / 耗时 / Token 构成与走势 -->
  <section class="usage-bento" :class="{ 'is-loading': loading }">
    <article class="ub-tile ub-tile--cost">
      <span class="ub-tile__sheen" aria-hidden="true"></span>
      <header class="ub-head">
        <span class="ub-label">{{ t('usage.totalCost') }}</span>
        <span class="ub-chip">{{ t('usage.inSelectedRange') }}</span>
      </header>
      <p class="ub-value ub-value--xl">
        <small>$</small>{{ costText }}
      </p>
      <p class="ub-sub">
        {{ t('usage.standardCost') }}
        <s>${{ (stats?.total_cost || 0).toFixed(4) }}</s>
        <span v-if="savingPercent > 0" class="ub-save">−{{ savingPercent }}%</span>
      </p>
      <div class="ub-spark ub-spark--tall">
        <RxSparkline :values="costSeries" :labels="dateLabels" :height="96" :format="(v) => `$${v.toFixed(4)}`" />
      </div>
    </article>

    <article class="ub-tile">
      <header class="ub-head">
        <span class="ub-label">{{ t('usage.totalRequests') }}</span>
        <span class="ub-icon" aria-hidden="true"><Icon name="bolt" size="sm" /></span>
      </header>
      <p class="ub-value">{{ requestsText }}</p>
      <div class="ub-spark">
        <RxSparkline :values="requestSeries" :labels="dateLabels" :height="40" />
      </div>
    </article>

    <article class="ub-tile">
      <header class="ub-head">
        <span class="ub-label">{{ t('usage.avgDuration') }}</span>
        <span class="ub-icon" aria-hidden="true"><Icon name="clock" size="sm" /></span>
      </header>
      <p class="ub-value">{{ durationText }}</p>
      <!-- 装饰性心电线：表达「响应节拍」，不代表具体数据 -->
      <svg class="ub-pulse" viewBox="0 0 200 40" preserveAspectRatio="none" aria-hidden="true">
        <path d="M0 20 H62 L70 8 L78 32 L86 4 L94 26 L100 20 H200" />
      </svg>
    </article>

    <article class="ub-tile ub-tile--wide">
      <header class="ub-head">
        <span class="ub-label">{{ t('usage.totalTokens') }}</span>
        <span class="ub-value ub-value--inline">{{ formatTokens(stats?.total_tokens || 0) }}</span>
      </header>
      <!-- 构成条：输入 / 输出 / 缓存 按比例展开 -->
      <div class="ub-split" role="img" :aria-label="t('usage.totalTokens')">
        <span v-for="seg in segments" :key="seg.key" class="ub-split__seg" :class="`is-${seg.key}`" :style="{ '--w': `${seg.pct}%` }"></span>
      </div>
      <ul class="ub-legend">
        <li v-for="seg in segments" :key="seg.key" :class="`is-${seg.key}`">
          <i aria-hidden="true"></i>{{ seg.label }}
          <b>{{ formatTokens(seg.value) }}</b>
          <small>{{ seg.pct.toFixed(0) }}%</small>
        </li>
      </ul>
      <!-- 走势柱：每个时间桶一根，悬浮显示数值 -->
      <div v-if="tokenBars.length > 1" class="ub-bars">
        <span
          v-for="(bar, index) in tokenBars"
          :key="index"
          class="ub-bars__bar"
          :style="{ '--h': `${bar.pct}%`, '--i': index }"
          :title="`${bar.label} · ${formatTokens(bar.value)}`"
        ></span>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import RxSparkline from '@/components/common/RxSparkline.vue'
import { useTweenedNumber } from '@/composables/useTweenedNumber'
import type { TrendDataPoint, UsageStatsResponse } from '@/types'

const props = defineProps<{
  stats: UsageStatsResponse | null
  trend: TrendDataPoint[]
  loading?: boolean
}>()

const { t } = useI18n()

const costTween = useTweenedNumber(computed(() => props.stats?.total_actual_cost || 0))
const requestTween = useTweenedNumber(computed(() => props.stats?.total_requests || 0))

const costText = computed(() => costTween.value.toFixed(4))
const requestsText = computed(() => Math.round(requestTween.value).toLocaleString())

const savingPercent = computed(() => {
  const std = props.stats?.total_cost || 0
  const actual = props.stats?.total_actual_cost || 0
  if (std <= 0 || actual >= std) return 0
  return Math.round((1 - actual / std) * 100)
})

const durationText = computed(() => {
  const ms = props.stats?.average_duration_ms || 0
  return ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`
})

function formatTokens(value: number): string {
  if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`
  if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`
  if (value >= 1e3) return `${(value / 1e3).toFixed(1)}K`
  return value.toLocaleString()
}

const dateLabels = computed(() => props.trend.map((p) => p.date))
const costSeries = computed(() => props.trend.map((p) => p.actual_cost || 0))
const requestSeries = computed(() => props.trend.map((p) => p.requests || 0))

const segments = computed(() => {
  const s = props.stats
  const input = s?.total_input_tokens || 0
  const output = s?.total_output_tokens || 0
  const cache = s?.total_cache_tokens || 0
  const sum = input + output + cache || 1
  return [
    { key: 'in', label: t('usage.in'), value: input, pct: (input / sum) * 100 },
    { key: 'out', label: t('usage.out'), value: output, pct: (output / sum) * 100 },
    { key: 'cache', label: t('usage.cacheTotal'), value: cache, pct: (cache / sum) * 100 },
  ]
})

// 柱数过多时按时间顺序抽样，保证每根柱有足够宽度
const tokenBars = computed(() => {
  const list = props.trend
  const max = Math.max(1, ...list.map((p) => p.total_tokens || 0))
  const step = Math.max(1, Math.ceil(list.length / 48))
  return list
    .filter((_, i) => i % step === 0)
    .map((p) => ({ label: p.date, value: p.total_tokens || 0, pct: Math.max(4, ((p.total_tokens || 0) / max) * 100) }))
})
</script>

<style scoped>
.usage-bento {
  display: grid;
  gap: 1rem;
  grid-template-columns: minmax(0, 1fr);
}

@media (min-width: 768px) {
  .usage-bento {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ub-tile--cost {
    grid-column: span 2;
  }

  .ub-tile--wide {
    grid-column: span 2;
  }
}

@media (min-width: 1280px) {
  .usage-bento {
    grid-template-columns: minmax(0, 1.25fr) minmax(0, 1fr) minmax(0, 1fr);
    grid-template-rows: auto auto;
  }

  .ub-tile--cost {
    grid-column: 1;
    grid-row: 1 / span 2;
  }

  .ub-tile--wide {
    grid-column: 2 / span 2;
  }
}

.ub-tile {
  position: relative;
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.6rem;
  overflow: hidden;
  padding: 1.25rem 1.35rem;
  border-radius: var(--rx-radius-lg);
  color: var(--rx-ink);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
  box-shadow: var(--rx-highlight), var(--rx-shadow);
  transition: transform 0.4s var(--rx-spring), border-color 0.3s ease, box-shadow 0.3s ease;
  animation: rx-rise 0.7s var(--rx-ease) both;
}

.ub-tile:nth-child(2) { animation-delay: 80ms; }
.ub-tile:nth-child(3) { animation-delay: 160ms; }
.ub-tile:nth-child(4) { animation-delay: 240ms; }

.ub-tile:hover {
  transform: translateY(-3px);
  border-color: var(--rx-line-strong);
  box-shadow: var(--rx-highlight), var(--rx-shadow-lg);
}

/* 消费大格：墨色卡 + 金属箔微光 */
.ub-tile--cost {
  min-height: 15rem;
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  border-color: transparent;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12), inset 0 0 0 1px var(--rx-card-line), var(--rx-shadow-lg);
}

.ub-tile--cost::after {
  content: '';
  position: absolute;
  left: 1.1rem;
  right: 1.1rem;
  bottom: 0.7rem;
  height: 6px;
  background-image: radial-gradient(circle, var(--rx-card-line) 1.2px, transparent 1.4px);
  background-size: 10px 6px;
  background-repeat: repeat-x;
  opacity: 0.7;
}

.ub-tile__sheen {
  pointer-events: none;
  position: absolute;
  inset: -40%;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, var(--rx-foil) 60deg, transparent 130deg, transparent 220deg, var(--rx-foil) 290deg, transparent 350deg);
  mix-blend-mode: soft-light;
  animation: ub-spin 20s linear infinite;
}

@keyframes ub-spin {
  to { --rx-angle: 360deg; }
}

.ub-head {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.ub-label {
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.ub-tile--cost .ub-label {
  color: inherit;
  opacity: 0.6;
}

.ub-chip {
  padding: 0.1rem 0.55rem;
  border-radius: 9999px;
  font-size: 0.64rem;
  font-weight: 700;
  border: 1px solid var(--rx-card-line);
  opacity: 0.75;
}

.ub-icon {
  display: grid;
  place-items: center;
  width: 1.9rem;
  height: 1.9rem;
  border-radius: 9999px;
  color: var(--rx-muted);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
}

.ub-value {
  position: relative;
  font-size: 1.9rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.045em;
  font-variant-numeric: tabular-nums;
}

.ub-value--xl {
  margin-top: 0.4rem;
  font-size: clamp(2.4rem, 1.6rem + 2.4vw, 3.4rem);
}

.ub-value--xl small {
  margin-right: 0.2rem;
  font-size: 0.45em;
  font-weight: 600;
  vertical-align: 0.65em;
  opacity: 0.6;
}

.ub-value--inline {
  font-size: 1.4rem;
}

.ub-sub {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.78rem;
  opacity: 0.65;
  font-variant-numeric: tabular-nums;
}

.ub-save {
  padding: 0 0.45rem;
  border-radius: 9999px;
  font-size: 0.68rem;
  font-weight: 800;
  line-height: 1.2rem;
  color: #11110f;
  background: var(--rx-card-ink);
  opacity: 1;
}

.ub-spark {
  position: relative;
  margin-top: auto;
  color: var(--rx-ink);
}

.ub-spark--tall {
  color: var(--rx-card-ink);
}

.ub-pulse {
  width: 100%;
  height: 40px;
  margin-top: auto;
  overflow: visible;
}

.ub-pulse path {
  fill: none;
  stroke: var(--rx-ink);
  stroke-width: 1.6;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
  stroke-dasharray: 60 260;
  animation: ub-beat 2.6s linear infinite;
  opacity: 0.8;
}

@keyframes ub-beat {
  from { stroke-dashoffset: 60; }
  to { stroke-dashoffset: -260; }
}

/* ---------- Token 构成 ---------- */
.ub-split {
  display: flex;
  height: 0.7rem;
  gap: 3px;
  overflow: hidden;
  border-radius: 9999px;
  background: var(--rx-surface-2);
}

.ub-split__seg {
  width: var(--w);
  min-width: 0;
  border-radius: 9999px;
  animation: ub-grow 1s var(--rx-ease) both 0.3s;
  transform-origin: left;
}

@keyframes ub-grow {
  from { transform: scaleX(0); }
  to { transform: scaleX(1); }
}

.is-in { --seg: var(--rx-ink); }
.is-out { --seg: color-mix(in srgb, var(--rx-ink) 45%, transparent); }
.is-cache { --seg: var(--rx-tint-ink); }

.ub-split__seg {
  background: var(--seg);
}

.ub-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 1.25rem;
  font-size: 0.75rem;
  color: var(--rx-muted);
}

.ub-legend li {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.ub-legend i {
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 3px;
  background: var(--seg);
}

.ub-legend b {
  font-weight: 750;
  color: var(--rx-ink);
  font-variant-numeric: tabular-nums;
}

.ub-legend small {
  color: var(--rx-faint);
}

.ub-bars {
  display: flex;
  height: 3.2rem;
  align-items: flex-end;
  gap: 3px;
  margin-top: auto;
}

.ub-bars__bar {
  flex: 1;
  min-width: 2px;
  height: var(--h);
  border-radius: 3px 3px 1px 1px;
  background: color-mix(in srgb, var(--rx-ink) 22%, transparent);
  transform-origin: bottom;
  animation: ub-bar 0.7s var(--rx-ease) both;
  animation-delay: calc(var(--i, 0) * 14ms + 300ms);
  transition: background 0.2s ease;
}

.ub-bars__bar:hover {
  background: var(--rx-ink);
}

@keyframes ub-bar {
  from { transform: scaleY(0); }
  to { transform: scaleY(1); }
}

.usage-bento.is-loading .ub-value {
  opacity: 0.45;
}

@media (prefers-reduced-motion: reduce) {
  .ub-tile,
  .ub-tile__sheen,
  .ub-pulse path,
  .ub-split__seg,
  .ub-bars__bar {
    animation: none;
  }
}
</style>
