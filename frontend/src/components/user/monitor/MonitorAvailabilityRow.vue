<template>
  <!-- 可用率环：描边按百分比展开，颜色沿用原 hsl 分档 -->
  <div class="avail-row">
    <div class="min-w-0">
      <div class="avail-row__label">{{ windowLabel }}</div>
      <div v-if="samplesLabel" class="avail-row__samples">{{ samplesLabel }}</div>
    </div>
    <div class="avail-ring" :style="colorStyle">
      <svg viewBox="0 0 64 64" aria-hidden="true">
        <circle class="avail-ring__track" cx="32" cy="32" r="27" />
        <circle
          class="avail-ring__bar"
          cx="32"
          cy="32"
          r="27"
          pathLength="100"
          :style="{ strokeDashoffset: `${100 - ringPct}` }"
        />
      </svg>
      <span class="avail-ring__value">
        <b>{{ displayValue }}</b><small>%</small>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { hslForPct } from '@/composables/useChannelMonitorFormat'

const props = defineProps<{
  windowLabel: string
  value: number | null
  samplesLabel?: string
}>()

const { t } = useI18n()

const displayValue = computed(() => {
  if (props.value === null || Number.isNaN(props.value)) return t('monitorCommon.latencyEmpty')
  return props.value.toFixed(2)
})

// 环形描边比例：无数据时为 0
const ringPct = computed(() => {
  if (props.value === null || Number.isNaN(props.value)) return 0
  return Math.min(100, Math.max(0, props.value))
})

const colorStyle = computed(() => {
  const colour = hslForPct(props.value)
  return colour ? { color: colour } : { color: 'rgb(156 163 175)' }
})
</script>

<style scoped>
.avail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.75rem;
}

.avail-row__label {
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint, #9ca3af);
}

.avail-row__samples {
  margin-top: 0.2rem;
  font-size: 0.7rem;
  color: var(--rx-faint, #9ca3af);
}

.avail-ring {
  position: relative;
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 4.6rem;
  height: 4.6rem;
}

.avail-ring svg {
  position: absolute;
  inset: 0;
  transform: rotate(-90deg);
}

.avail-ring__track {
  fill: none;
  stroke: var(--rx-line, #e5e7eb);
  stroke-width: 5;
}

.avail-ring__bar {
  fill: none;
  stroke: currentColor;
  stroke-width: 5;
  stroke-linecap: round;
  stroke-dasharray: 100;
  transition: stroke-dashoffset 1.1s var(--rx-ease, ease);
  animation: avail-draw 1.3s var(--rx-ease, ease) both 0.2s;
}

@keyframes avail-draw {
  from { stroke-dashoffset: 100; }
}

.avail-ring__value {
  position: relative;
  display: flex;
  align-items: baseline;
  font-variant-numeric: tabular-nums;
}

.avail-ring__value b {
  font-size: 0.95rem;
  font-weight: 800;
  letter-spacing: -0.03em;
}

.avail-ring__value small {
  font-size: 0.6rem;
  font-weight: 700;
}

@media (prefers-reduced-motion: reduce) {
  .avail-ring__bar {
    animation: none;
  }
}
</style>
