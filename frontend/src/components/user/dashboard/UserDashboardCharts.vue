<template>
  <div class="dash-charts space-y-4">
    <header class="rx-section-head">
      <span class="rx-section-head__index">03</span>
      <h2 class="rx-section-head__title">{{ t('dashboard.hero.chartsTitle') }}</h2>
    </header>

    <section class="rx-toolbar">
      <div class="rx-toolbar__row">
        <div class="rx-toolbar__fields">
          <label class="dash-filter">
            <span class="dash-filter__label">{{ t('dashboard.timeRange') }}</span>
            <DateRangePicker
              :start-date="startDate"
              :end-date="endDate"
              @update:startDate="$emit('update:startDate', $event)"
              @update:endDate="$emit('update:endDate', $event)"
              @change="$emit('dateRangeChange', $event)"
            />
          </label>
          <label class="dash-filter dash-filter--sm">
            <span class="dash-filter__label">{{ t('dashboard.granularity') }}</span>
            <Select
              :model-value="granularity"
              :options="[{value:'day', label:t('dashboard.day')}, {value:'hour', label:t('dashboard.hour')}]"
              @update:model-value="$emit('update:granularity', $event)"
              @change="$emit('granularityChange')"
            />
          </label>
        </div>
        <div class="rx-toolbar__actions">
          <button type="button" class="rx-icon-btn is-wide" :disabled="loading" @click="$emit('refresh')">
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>
    </section>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <div class="rx-glass dash-chart relative overflow-hidden">
        <div v-if="loading" class="dash-chart__veil">
          <LoadingSpinner size="md" />
        </div>
        <header class="dash-chart__head">
          <span class="dash-chart__idx">01</span>
          <h3>{{ t('dashboard.modelDistribution') }}</h3>
        </header>
        <div class="flex flex-col items-center gap-4 sm:flex-row sm:gap-6">
          <div class="h-48 w-48 shrink-0">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div v-else class="dash-chart__empty">{{ t('dashboard.noDataAvailable') }}</div>
          </div>
          <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
            <table class="w-full text-xs">
              <thead>
                <tr>
                  <th class="pb-2 text-left">{{ t('dashboard.model') }}</th>
                  <th class="pb-2 text-right">{{ t('dashboard.requests') }}</th>
                  <th class="pb-2 text-right">{{ t('dashboard.tokens') }}</th>
                  <th class="pb-2 text-right">{{ t('dashboard.actual') }}</th>
                  <th class="pb-2 text-right">{{ t('dashboard.standard') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.model">
                  <td class="max-w-[100px] truncate py-1.5 font-medium" :title="model.model">{{ model.model }}</td>
                  <td class="py-1.5 text-right">{{ formatNumber(model.requests) }}</td>
                  <td class="py-1.5 text-right">{{ formatTokens(model.total_tokens) }}</td>
                  <td class="py-1.5 text-right" :title="t('dashboard.actual')">${{ formatCost(model.actual_cost) }}</td>
                  <td class="py-1.5 text-right dash-chart__dim" :title="t('dashboard.standard')">${{ formatCost(model.cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="dash-trend">
        <TokenUsageTrend :trend-data="trend" :loading="loading" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import { Doughnut } from 'vue-chartjs'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: ['#1d1d1a', '#c2410c', '#0f766e', '#a16207', '#44403c', '#9a3412', '#57534e', '#78716c'],
    borderWidth: 0,
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}
</script>

<style scoped>
.dash-filter {
  display: grid;
  gap: 0.3rem;
}

.dash-filter--sm {
  min-width: 7.5rem;
}

.dash-filter__label {
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.dash-chart {
  padding: 1.15rem 1.2rem 1.25rem;
}

.dash-chart__head {
  display: flex;
  align-items: baseline;
  gap: 0.65rem;
  margin-bottom: 1rem;
}

.dash-chart__idx {
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 0.9rem;
  color: var(--rx-faint);
}

.dash-chart__head h3 {
  font-size: 0.92rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.dash-chart__veil {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--rx-surface) 72%, transparent);
  backdrop-filter: blur(6px);
}

.dash-chart__empty {
  display: flex;
  height: 100%;
  align-items: center;
  justify-content: center;
  font-size: 0.82rem;
  color: var(--rx-faint);
}

.dash-chart__dim {
  color: var(--rx-faint);
}

.dash-trend :deep(.card) {
  position: relative;
  height: 100%;
  padding: 1.15rem 1.2rem 1.25rem;
}

.dash-trend :deep(h3) {
  display: flex;
  align-items: baseline;
  gap: 0.65rem;
  margin-bottom: 1rem;
  font-size: 0.92rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.dash-trend :deep(h3)::before {
  content: '02';
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--rx-faint);
}
</style>
