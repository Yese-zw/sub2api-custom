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
    <MonitorIQRing :recent="recent" />
    <span class="iq-entry__cta">{{ detailLabel }}<span aria-hidden="true"> ›</span></span>
  </div>
</template>

<script setup lang="ts">
import MonitorIQRing from './MonitorIQRing.vue'

defineProps<{
  label: string
  detailLabel: string
  /** 最近若干条检测是否通过（旧→新，最多 10 条） */
  recent: boolean[]
}>()
const emit = defineEmits<{ (e: 'open'): void }>()
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
