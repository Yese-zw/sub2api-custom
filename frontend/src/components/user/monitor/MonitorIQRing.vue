<template>
  <!-- 智商分段环：10 段（旧→新，12 点起顺时针），绿=通过，红=未通过，灰=暂无；中间为通过率 -->
  <div class="iq-ring" :style="{ '--iq-ring-size': size }">
    <svg viewBox="0 0 64 64" aria-hidden="true">
      <circle
        v-for="(seg, i) in segments"
        :key="i"
        :class="['iq-ring__seg', `is-${seg}`]"
        cx="32"
        cy="32"
        r="27"
        pathLength="100"
        :stroke-dasharray="`${SEG_LEN} ${100 - SEG_LEN}`"
        :stroke-dashoffset="`${-(i * SEG_STEP + SEG_GAP / 2)}`"
      />
    </svg>
    <span class="iq-ring__value">
      <b>{{ percentText }}</b><small v-if="total > 0">%</small>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const SEGMENTS = 10
const SEG_STEP = 100 / SEGMENTS
const SEG_GAP = 2
const SEG_LEN = SEG_STEP - SEG_GAP

const props = withDefaults(
  defineProps<{
    /** 最近若干条检测是否通过（旧→新，最多 10 条） */
    recent: boolean[]
    size?: string
  }>(),
  { size: '4.6rem' }
)

const recent = computed(() => props.recent.slice(-SEGMENTS))
const total = computed(() => recent.value.length)
const segments = computed<Array<'pass' | 'fail' | 'empty'>>(() =>
  Array.from({ length: SEGMENTS }, (_, i) => {
    if (i >= recent.value.length) return 'empty'
    return recent.value[i] ? 'pass' : 'fail'
  })
)
const percentText = computed(() => {
  if (total.value === 0) return '-'
  const passed = recent.value.filter(Boolean).length
  return String(Math.round((passed / total.value) * 100))
})
</script>

<style scoped>
.iq-ring {
  position: relative;
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: var(--iq-ring-size);
  height: var(--iq-ring-size);
}

.iq-ring svg {
  position: absolute;
  inset: 0;
  transform: rotate(-90deg);
}

.iq-ring__seg {
  fill: none;
  stroke-width: 5;
  transition: stroke 0.3s ease;
}
.iq-ring__seg.is-pass { stroke: hsl(140 60% 42%); }
.iq-ring__seg.is-fail { stroke: #ef4444; }
.iq-ring__seg.is-empty { stroke: var(--rx-line, #e5e7eb); }

.iq-ring__value {
  position: relative;
  display: flex;
  align-items: baseline;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink, inherit);
}
.iq-ring__value b { font-size: 0.95rem; font-weight: 800; letter-spacing: -0.03em; }
.iq-ring__value small { font-size: 0.6rem; font-weight: 700; }
</style>
