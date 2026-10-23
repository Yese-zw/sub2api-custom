<template>
  <!-- 迷你趋势线：入场描线 + 指针擦拭查看单点数值 -->
  <div
    ref="rootRef"
    class="rx-spark"
    :class="{ 'is-empty': points.length < 2, 'is-scrubbing': hoverIndex !== null }"
    :style="{ height: `${height}px` }"
    @pointermove="onMove"
    @pointerleave="hoverIndex = null"
  >
    <svg v-if="points.length >= 2" class="rx-spark__svg" :viewBox="`0 0 ${W} ${height}`" preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient :id="gradId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="currentColor" stop-opacity="0.22" />
          <stop offset="100%" stop-color="currentColor" stop-opacity="0" />
        </linearGradient>
      </defs>
      <path class="rx-spark__area" :d="areaPath" :fill="`url(#${gradId})`" />
      <path class="rx-spark__line" :d="linePath" pathLength="1" />
      <g v-if="hoverIndex !== null">
        <line class="rx-spark__cursor" :x1="points[hoverIndex].x" :x2="points[hoverIndex].x" y1="0" :y2="height" />
        <circle class="rx-spark__dot" :cx="points[hoverIndex].x" :cy="points[hoverIndex].y" r="3.2" />
      </g>
      <circle v-else class="rx-spark__dot rx-spark__dot--end" :cx="points[points.length - 1].x" :cy="points[points.length - 1].y" r="2.6" />
    </svg>
    <span v-else class="rx-spark__flat" aria-hidden="true"></span>

    <Transition name="rx-spark-tip">
      <span
        v-if="hoverIndex !== null && points[hoverIndex]"
        class="rx-spark__tip"
        :style="{ left: `${(points[hoverIndex].x / W) * 100}%` }"
      >
        <b>{{ format(values[hoverIndex]) }}</b>
        <small v-if="labels?.[hoverIndex]">{{ labels[hoverIndex] }}</small>
      </span>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = withDefaults(defineProps<{
  values: number[]
  labels?: string[]
  height?: number
  format?: (v: number) => string
}>(), {
  labels: undefined,
  height: 44,
  format: (v: number) => v.toLocaleString(),
})

// 内部坐标宽度固定，SVG 横向拉伸填满容器
const W = 200
const PAD = 4
const gradId = `rx-spark-${Math.random().toString(36).slice(2, 9)}`
const rootRef = ref<HTMLElement | null>(null)
const hoverIndex = ref<number | null>(null)

const points = computed(() => {
  const vals = props.values
  if (vals.length < 2) return []
  const max = Math.max(...vals)
  const min = Math.min(...vals)
  const span = max - min || 1
  const h = props.height
  return vals.map((v, i) => ({
    x: (i / (vals.length - 1)) * W,
    y: h - PAD - ((v - min) / span) * (h - PAD * 2),
  }))
})

// Catmull-Rom 转三次贝塞尔，得到平滑曲线
const linePath = computed(() => {
  const p = points.value
  if (p.length < 2) return ''
  let d = `M${p[0].x},${p[0].y}`
  for (let i = 0; i < p.length - 1; i++) {
    const p0 = p[i - 1] ?? p[i]
    const p1 = p[i]
    const p2 = p[i + 1]
    const p3 = p[i + 2] ?? p2
    const c1x = p1.x + (p2.x - p0.x) / 6
    const c1y = p1.y + (p2.y - p0.y) / 6
    const c2x = p2.x - (p3.x - p1.x) / 6
    const c2y = p2.y - (p3.y - p1.y) / 6
    d += ` C${c1x},${c1y} ${c2x},${c2y} ${p2.x},${p2.y}`
  }
  return d
})

const areaPath = computed(() => (linePath.value ? `${linePath.value} L${W},${props.height} L0,${props.height} Z` : ''))

function onMove(e: PointerEvent) {
  const el = rootRef.value
  if (!el || points.value.length < 2) return
  const rect = el.getBoundingClientRect()
  const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
  hoverIndex.value = Math.round(ratio * (points.value.length - 1))
}
</script>

<style scoped>
.rx-spark {
  position: relative;
  width: 100%;
  color: currentColor;
  touch-action: none;
}

.rx-spark__svg {
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.rx-spark__line {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  animation: rx-spark-draw 1.2s cubic-bezier(0.22, 1, 0.36, 1) 0.2s forwards;
}

.rx-spark__area {
  opacity: 0;
  animation: rx-spark-fade 0.8s ease 0.7s forwards;
}

@keyframes rx-spark-draw {
  to { stroke-dashoffset: 0; }
}

@keyframes rx-spark-fade {
  to { opacity: 1; }
}

.rx-spark__cursor {
  stroke: currentColor;
  stroke-width: 1;
  stroke-dasharray: 2 3;
  opacity: 0.45;
  vector-effect: non-scaling-stroke;
}

.rx-spark__dot {
  fill: currentColor;
  stroke: var(--rx-surface-hover, #fff);
  stroke-width: 1.5;
  vector-effect: non-scaling-stroke;
}

.rx-spark__dot--end {
  animation: rx-spark-fade 0.4s ease 1.3s both;
}

.rx-spark__flat {
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  border-top: 1.5px dashed currentColor;
  opacity: 0.25;
}

.rx-spark__tip {
  pointer-events: none;
  position: absolute;
  bottom: calc(100% + 6px);
  z-index: 3;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0.25rem 0.5rem;
  border-radius: 8px;
  white-space: nowrap;
  transform: translateX(-50%);
  color: var(--rx-on-accent, #fff);
  background: var(--rx-accent, #111827);
  box-shadow: 0 6px 16px -6px rgba(17, 17, 15, 0.5);
}

.rx-spark__tip b {
  font-size: 0.72rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.rx-spark__tip small {
  font-size: 0.6rem;
  opacity: 0.7;
}

.rx-spark-tip-enter-active,
.rx-spark-tip-leave-active {
  transition: opacity 0.15s ease;
}

.rx-spark-tip-enter-from,
.rx-spark-tip-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .rx-spark__line {
    animation: none;
    stroke-dashoffset: 0;
  }

  .rx-spark__area,
  .rx-spark__dot--end {
    animation: none;
    opacity: 1;
  }
}
</style>
