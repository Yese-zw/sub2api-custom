<template>
  <!-- 整体状态球：同心环呼吸 + 四项汇总；颜色随整体状态切换 -->
  <div class="status-orb rx-glass" :class="`is-${overallStatus}`">
    <span class="status-orb__sweep" aria-hidden="true"></span>
    <div class="status-orb__core" aria-hidden="true">
      <span class="status-orb__ring"></span>
      <span class="status-orb__ring status-orb__ring--2"></span>
      <span class="status-orb__ring status-orb__ring--3"></span>
      <span class="status-orb__dot"></span>
    </div>
    <dl class="status-orb__stats">
      <div>
        <dt>{{ t('channelStatus.hero.healthy') }}</dt>
        <dd>{{ healthyCount }}<small>/{{ items.length }}</small></dd>
      </div>
      <div>
        <dt>{{ t('channelStatus.hero.issues') }}</dt>
        <dd :class="{ 'is-warn': issueCount > 0 }">{{ issueCount }}</dd>
      </div>
      <div>
        <dt>{{ t('channelStatus.hero.avgAvailability') }}</dt>
        <dd>{{ avgAvailability === null ? '—' : `${avgAvailability.toFixed(2)}%` }}</dd>
      </div>
      <div>
        <dt>{{ t('channelStatus.hero.medianLatency') }}</dt>
        <dd>{{ medianLatency === null ? '—' : `${Math.round(medianLatency)}` }}<small v-if="medianLatency !== null">ms</small></dd>
      </div>
    </dl>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView } from '@/api/channelMonitor'
import { STATUS_OPERATIONAL } from '@/constants/channelMonitor'

const props = defineProps<{
  items: UserMonitorView[]
  overallStatus: 'operational' | 'degraded'
}>()

const { t } = useI18n()

const healthyCount = computed(() => props.items.filter((i) => i.primary_status === STATUS_OPERATIONAL).length)
const issueCount = computed(() => props.items.length - healthyCount.value)

// 7 天可用率均值：列表接口只下发 7d 值，与卡片默认窗口一致
const avgAvailability = computed(() => {
  const vals = props.items.map((i) => i.availability_7d).filter((v) => typeof v === 'number' && !Number.isNaN(v))
  if (vals.length === 0) return null
  return vals.reduce((a, b) => a + b, 0) / vals.length
})

const medianLatency = computed(() => {
  const vals = props.items
    .map((i) => i.primary_latency_ms)
    .filter((v): v is number => typeof v === 'number')
    .sort((a, b) => a - b)
  if (vals.length === 0) return null
  const mid = Math.floor(vals.length / 2)
  return vals.length % 2 ? vals[mid] : (vals[mid - 1] + vals[mid]) / 2
})
</script>

<style scoped>
.status-orb {
  --orb: #16a34a;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  gap: 1.5rem;
  padding: 1.1rem 1.4rem 1.1rem 1.1rem;
}

.status-orb__sweep {
  pointer-events: none;
  position: absolute;
  inset: -20%;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, color-mix(in srgb, var(--orb) 22%, transparent) 50deg, transparent 110deg);
  mix-blend-mode: soft-light;
  animation: orb-sweep 10s linear infinite;
}

@keyframes orb-sweep {
  to { --rx-angle: 360deg; }
}

.status-orb.is-degraded {
  --orb: #d97706;
}

@media (min-width: 1024px) {
  .status-orb {
    width: 30rem;
  }
}

.status-orb__core {
  position: relative;
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 5.5rem;
  height: 5.5rem;
}

.status-orb__ring {
  position: absolute;
  inset: 0;
  border-radius: 9999px;
  border: 1.5px solid color-mix(in srgb, var(--orb) 45%, transparent);
  animation: orb-breathe 3.2s ease-out infinite;
}

.status-orb__ring--2 {
  animation-delay: 1.05s;
}

.status-orb__ring--3 {
  animation-delay: 2.1s;
}

@keyframes orb-breathe {
  0% { transform: scale(0.35); opacity: 0.9; }
  100% { transform: scale(1); opacity: 0; }
}

.status-orb__dot {
  width: 2.2rem;
  height: 2.2rem;
  border-radius: 9999px;
  background: radial-gradient(circle at 32% 30%, color-mix(in srgb, var(--orb) 35%, white), var(--orb) 60%, color-mix(in srgb, var(--orb) 70%, black));
  box-shadow: 0 0 0 6px color-mix(in srgb, var(--orb) 14%, transparent), 0 10px 26px -6px color-mix(in srgb, var(--orb) 70%, transparent);
}

.status-orb__stats {
  display: grid;
  flex: 1;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.7rem 1.25rem;
}

.status-orb__stats dt {
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.status-orb__stats dd {
  margin-top: 0.15rem;
  font-size: 1.35rem;
  font-weight: 800;
  line-height: 1.1;
  letter-spacing: -0.035em;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink);
}

.status-orb__stats dd small {
  margin-left: 0.1rem;
  font-size: 0.6em;
  font-weight: 600;
  color: var(--rx-faint);
}

.status-orb__stats dd.is-warn {
  color: #b45309;
}

html.dark .status-orb__stats dd.is-warn {
  color: #fbbf24;
}

@media (prefers-reduced-motion: reduce) {
  .status-orb__ring,
  .status-orb__sweep {
    animation: none;
    opacity: 0.4;
  }
}
</style>
