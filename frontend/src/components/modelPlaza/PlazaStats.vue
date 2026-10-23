<template>
  <!-- 模型广场概览：平台 / 分组 / 模型数 + 最低生效倍率 -->
  <div class="plaza-stats" :class="{ 'is-loading': loading }">
    <div class="plaza-stats__cell plaza-stats__cell--ink">
      <span class="plaza-stats__label">{{ t('modelPlaza.hero.statModels') }}</span>
      <strong class="plaza-stats__value">{{ loading ? '—' : modelCount }}</strong>
    </div>
    <div class="plaza-stats__cell">
      <span class="plaza-stats__label">{{ t('modelPlaza.hero.statGroups') }}</span>
      <strong class="plaza-stats__value">{{ loading ? '—' : groupCount }}</strong>
    </div>
    <div class="plaza-stats__cell">
      <span class="plaza-stats__label">{{ t('modelPlaza.hero.statPlatforms') }}</span>
      <strong class="plaza-stats__value">{{ loading ? '—' : platformCount }}</strong>
    </div>
    <div class="plaza-stats__cell">
      <span class="plaza-stats__label">{{ t('modelPlaza.hero.statLowestRate') }}</span>
      <strong class="plaza-stats__value">{{ loading || lowestRate === null ? '—' : `${lowestRate}x` }}</strong>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelPlazaResponse } from '@/api/modelPlaza'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
}>()

const { t } = useI18n()

const groups = computed(() => props.response?.groups ?? [])
const groupCount = computed(() => groups.value.length)
// 同名模型可能出现在多个分组，按名称去重
const modelCount = computed(() => new Set(groups.value.flatMap((g) => g.models.map((m) => m.name))).size)
const platformCount = computed(() => new Set(groups.value.map((g) => g.platform).filter(Boolean)).size)
// 生效倍率 = 用户专属倍率 ?? 分组默认倍率（与 ModelPlazaContent 一致）
const lowestRate = computed(() => {
  if (groups.value.length === 0) return null
  return Math.min(...groups.value.map((g) => g.user_rate_multiplier ?? g.rate_multiplier))
})
</script>

<style scoped>
.plaza-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.6rem;
}

@media (min-width: 1024px) {
  .plaza-stats {
    width: 24rem;
  }
}

.plaza-stats__cell {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.9rem 1rem;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
  box-shadow: var(--rx-highlight), var(--rx-shadow);
  transition: transform 0.35s var(--rx-spring);
}

.plaza-stats__cell:hover {
  transform: translateY(-2px);
}

.plaza-stats__cell--ink {
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  border-color: var(--rx-accent);
}

.plaza-stats__label {
  font-size: 0.64rem;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.plaza-stats__cell--ink .plaza-stats__label {
  color: inherit;
  opacity: 0.65;
}

.plaza-stats__value {
  font-size: 1.55rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
}

.plaza-stats.is-loading .plaza-stats__value {
  opacity: 0.4;
}
</style>
