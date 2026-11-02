<template>
  <!-- 智商状态入口：标题在上、分段环在下、底部「查看详情」提示可点击。
       卡片本身是 button，这里用 role=button 的 div 避免嵌套 button。 -->
  <div
    role="button"
    tabindex="0"
    class="iq-entry"
    :title="detailLabel"
    @click.stop="emit('open')"
    @keydown.enter.stop.prevent="emit('open')"
    @keydown.space.stop.prevent="emit('open')"
  >
    <div class="iq-entry__label">{{ label }}</div>
    <div class="iq-ring">
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
    <span class="iq-entry__cta">{{ detailLabel }}<span aria-hidden="true"> ›</span></span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const SEGMENTS = 10
const SEG_STEP = 100 / SEGMENTS
const SEG_GAP = 2
const SEG_LEN = SEG_STEP - SEG_GAP

const props = defineProps<{
  label: string
  detailLabel: string
  /** 最近若干条检测是否通过（旧→新，最多 10 条） */
  recent: boolean[]
}>()
const emit = defineEmits<{ (e: 'open'): void }>()

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
.iq-entry {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.4rem;
  cursor: pointer;
  border-radius: 0.75rem;
}

.iq-entry:focus-visible {
  outline: 2px solid var(--rx-accent, #14b8a6);
  outline-offset: 3px;
}

.iq-entry__label {
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint, #9ca3af);
  white-space: nowrap;
}

.iq-ring {
  position: relative;
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 4.6rem;
  height: 4.6rem;
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
}
.iq-ring__value b { font-size: 0.95rem; font-weight: 800; letter-spacing: -0.03em; }
.iq-ring__value small { font-size: 0.6rem; font-weight: 700; }

/* 「查看详情」胶囊：提示此处可点击 */
.iq-entry__cta {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  border: 1px solid var(--rx-accent, #14b8a6);
  padding: 0.1rem 0.6rem;
  font-size: 0.66rem;
  font-weight: 700;
  color: var(--rx-accent, #14b8a6);
  white-space: nowrap;
  transition: background 0.2s ease, color 0.2s ease;
}
.iq-entry:hover .iq-entry__cta,
.iq-entry:focus-visible .iq-entry__cta {
  background: var(--rx-accent, #14b8a6);
  color: #fff;
}
</style>
