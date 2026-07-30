<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="card p-5">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.profit.totalProfit') }}</p>
          <p class="mt-2 text-2xl font-semibold" :class="profitClass(summary.total_profit)">
            ${{ formatCost(summary.total_profit) }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.profit.allTimeHint') }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.profit.monthProfit') }}</p>
          <p class="mt-2 text-2xl font-semibold" :class="profitClass(summary.month_profit)">
            ${{ formatCost(summary.month_profit) }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.profit.monthProfitHint') }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.profit.todayProfit') }}</p>
          <p class="mt-2 text-2xl font-semibold" :class="profitClass(summary.today_profit)">
            ${{ formatCost(summary.today_profit) }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.profit.todayProfitHint') }}</p>
        </div>
        <div class="card p-5">
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.profit.currentTotalBalance') }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            ${{ formatCost(summary.current_total_balance) }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.profit.currentTotalBalanceHint') }}</p>
        </div>
      </div>

      <div class="card p-4">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.profit.trendTitle') }}</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.profit.trendDescription') }}</p>
          </div>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadData">
            {{ t('common.refresh') }}
          </button>
        </div>
        <div v-if="loading" class="flex h-72 items-center justify-center">
          <LoadingSpinner />
        </div>
        <div v-else-if="trend.length > 0 && chartData" class="h-72">
          <Line :data="chartData" :options="lineOptions" />
        </div>
        <div v-else class="flex h-72 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.dashboard.noDataAvailable') }}
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.profit.detailTitle') }}</h3>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr class="text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
                <th class="px-4 py-3">{{ t('usage.time') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.profit.balanceRevenue') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.profit.accountCost') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.profit.balanceProfit') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.profit.subscriptionCost') }}</th>
                <th class="px-4 py-3 text-right">{{ t('admin.profit.profit') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="item in reversedTrend" :key="item.date" class="text-gray-700 dark:text-gray-300">
                <td class="whitespace-nowrap px-4 py-3 font-medium text-gray-900 dark:text-white">{{ item.date }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-green-600 dark:text-green-400">${{ formatCost(item.revenue) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-orange-500 dark:text-orange-400">${{ formatCost(item.account_cost) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right" :class="profitClass(item.balance_profit)">${{ formatCost(item.balance_profit) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right">${{ formatCost(item.subscription_cost) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right font-semibold" :class="profitClass(item.profit)">${{ formatCost(item.profit) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { ProfitTrendPoint, ProfitTrendResponse } from '@/api/admin/dashboard'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const trend = ref<ProfitTrendPoint[]>([])
const summary = ref<ProfitTrendResponse>({
  trend: [],
  start_date: '',
  end_date: '',
  total_revenue: 0,
  total_account_cost: 0,
  total_balance_profit: 0,
  total_subscription_cost: 0,
  total_profit: 0,
  month_profit: 0,
  today_revenue: 0,
  today_profit: 0,
  current_total_balance: 0
})

const reversedTrend = computed(() => [...trend.value].reverse())

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb',
  profit: '#2563eb',
  revenue: '#16a34a',
  cost: '#f97316'
}))

const chartData = computed(() => {
  if (!trend.value.length) return null
  return {
    labels: trend.value.map((item) => item.date),
    datasets: [
      {
        label: t('admin.profit.profit'),
        data: trend.value.map((item) => item.profit),
        borderColor: chartColors.value.profit,
        backgroundColor: `${chartColors.value.profit}1f`,
        fill: true,
        tension: 0.3
      },
      {
        label: t('admin.profit.balanceRevenue'),
        data: trend.value.map((item) => item.revenue),
        borderColor: chartColors.value.revenue,
        backgroundColor: `${chartColors.value.revenue}14`,
        fill: false,
        tension: 0.3
      },
      {
        label: t('admin.profit.accountCost'),
        data: trend.value.map((item) => item.account_cost),
        borderColor: chartColors.value.cost,
        backgroundColor: `${chartColors.value.cost}14`,
        fill: false,
        tension: 0.3
      }
    ]
  }
})

const lineOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: { size: 11 }
      }
    },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.dataset.label}: $${formatCost(context.raw)}`
      }
    }
  },
  scales: {
    x: {
      grid: { color: chartColors.value.grid },
      ticks: { color: chartColors.value.text, font: { size: 10 } }
    },
    y: {
      grid: { color: chartColors.value.grid },
      ticks: {
        color: chartColors.value.text,
        font: { size: 10 },
        callback: (value: string | number) => `$${formatCost(Number(value))}`
      }
    }
  }
}))

const loadData = async () => {
  loading.value = true
  try {
    const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
    const response = await adminAPI.dashboard.getProfitTrend({ timezone })
    summary.value = response
    trend.value = response.trend || []
  } catch (error) {
    console.error('Failed to load profit trend:', error)
    appStore.showError(t('admin.profit.failedToLoad'))
  } finally {
    loading.value = false
  }
}

const profitClass = (value: number) => value >= 0
  ? 'text-blue-600 dark:text-blue-400'
  : 'text-red-600 dark:text-red-400'

const formatCost = (value: number): string => {
  const abs = Math.abs(value || 0)
  const sign = value < 0 ? '-' : ''
  if (abs >= 1000) return `${sign}${(abs / 1000).toFixed(2)}K`
  if (abs >= 1) return `${sign}${abs.toFixed(2)}`
  if (abs >= 0.01) return `${sign}${abs.toFixed(3)}`
  return `${sign}${abs.toFixed(4)}`
}

onMounted(() => {
  loadData()
})
</script>
