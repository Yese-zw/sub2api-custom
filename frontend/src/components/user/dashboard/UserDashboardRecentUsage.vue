<template>
  <section class="rx-glass dash-ledger">
    <header class="rx-section-head">
      <span class="rx-section-head__index">04</span>
      <h2 class="rx-section-head__title">{{ t('dashboard.recentUsage') }}</h2>
      <span class="rx-section-head__hint">{{ t('dashboard.last7Days') }}</span>
    </header>

    <div v-if="loading" class="dash-ledger__busy">
      <LoadingSpinner size="lg" />
    </div>
    <div v-else-if="data.length === 0" class="py-6">
      <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    </div>
    <ol v-else class="dash-ledger__list">
      <li v-for="log in data" :key="log.id" class="dash-ledger__row">
        <span class="dash-ledger__mark" aria-hidden="true"></span>
        <div class="min-w-0">
          <p class="dash-ledger__model">{{ log.model }}</p>
          <p class="dash-ledger__time">{{ formatDateTime(log.created_at) }}</p>
        </div>
        <div class="dash-ledger__sum">
          <p>
            <span :title="t('dashboard.actual')">${{ formatCost(log.actual_cost) }}</span>
            <small :title="t('dashboard.standard')"> / ${{ formatCost(log.total_cost) }}</small>
          </p>
          <p>{{ (log.input_tokens + log.output_tokens).toLocaleString() }} tokens</p>
        </div>
      </li>
    </ol>

    <router-link v-if="!loading && data.length > 0" to="/usage" class="dash-ledger__more">
      {{ t('dashboard.viewAllUsage') }}
      <Icon name="arrowRight" size="sm" />
    </router-link>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const formatCost = (c: number) => c.toFixed(4)
</script>

<style scoped>
.dash-ledger {
  padding: 1.25rem 1.25rem 1.1rem;
}

@media (min-width: 640px) {
  .dash-ledger {
    padding: 1.5rem 1.6rem 1.25rem;
  }
}

.dash-ledger .rx-section-head {
  margin-bottom: 0.9rem;
}

.dash-ledger__busy {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3rem 0;
}

.dash-ledger__list {
  counter-reset: dash-l;
  display: grid;
  gap: 0.55rem;
}

.dash-ledger__row {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.85rem 1rem 0.85rem 1.15rem;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
  transition: background 0.2s ease, border-color 0.2s ease;
}

.dash-ledger__row:hover {
  background: var(--rx-surface-hover);
  border-color: var(--rx-line);
}

.dash-ledger__mark {
  position: absolute;
  left: 0;
  top: 0.85rem;
  bottom: 0.85rem;
  width: 3px;
  border-radius: 9999px;
  background: var(--rx-accent);
  opacity: 0.35;
}

.dash-ledger__model {
  font-size: 0.88rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--rx-ink);
}

.dash-ledger__time {
  margin-top: 0.15rem;
  font-size: 0.72rem;
  color: var(--rx-faint);
}

.dash-ledger__sum {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.dash-ledger__sum p:first-child {
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.dash-ledger__sum small,
.dash-ledger__sum p:last-child {
  font-size: 0.72rem;
  font-weight: 500;
  color: var(--rx-faint);
}

.dash-ledger__more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  margin-top: 0.85rem;
  padding: 0.7rem 0 0.15rem;
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.dash-ledger__more:hover {
  color: var(--rx-ink-2);
}
</style>
