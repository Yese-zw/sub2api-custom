<template>
  <div class="mt-5 grid grid-cols-2 gap-2">
    <div
      class="metric-chip rounded-xl p-3"
    >
      <div
        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-gray-400"
      >
        <Icon :name="primaryIcon" size="xs" />
        <span>{{ primaryLabel }}</span>
      </div>
      <div class="metric-chip__value mt-1.5 text-lg font-bold tabular-nums">
        {{ primaryValue }}<span class="text-xs font-normal text-gray-400 ml-0.5">{{ primaryUnit }}</span>
      </div>
    </div>
    <div
      class="metric-chip rounded-xl p-3"
    >
      <div
        class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-gray-400"
      >
        <Icon :name="secondaryIcon" size="xs" />
        <span>{{ secondaryLabel }}</span>
      </div>
      <div class="metric-chip__value mt-1.5 text-lg font-bold tabular-nums">
        {{ secondaryValue }}<span class="text-xs font-normal text-gray-400 ml-0.5">{{ secondaryUnit }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  primaryLabel: string
  primaryValue: string
  primaryUnit: string
  primaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
  secondaryLabel: string
  secondaryValue: string
  secondaryUnit: string
  secondaryIcon: 'bolt' | 'globe' | 'clock' | 'link'
}>()
</script>

<style scoped>
.metric-chip {
  border-radius: var(--rx-radius-sm, 0.75rem);
  background: var(--rx-surface-2, rgba(249, 250, 251, 0.8));
  border: 1px solid var(--rx-line-soft, #f3f4f6);
  transition: border-color 0.2s ease, background 0.2s ease;
}

.metric-chip:hover {
  background: var(--rx-surface-hover, #fff);
  border-color: var(--rx-line, #e5e7eb);
}

.metric-chip__value {
  letter-spacing: -0.03em;
  color: var(--rx-ink, #111827);
}
</style>
