<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <div class="flex-1 overflow-auto">
      <table class="log-table">
        <thead>
          <tr>
            <th>{{ t('admin.schedulingCenter.log.time') }}</th>
            <th>{{ t('admin.schedulingCenter.columns.name') }}</th>
            <th>{{ t('admin.schedulingCenter.log.action') }}</th>
            <th>{{ t('admin.schedulingCenter.log.source') }}</th>
            <th class="w-full">{{ t('admin.schedulingCenter.log.reason') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading && events.length === 0" class="no-hover">
            <td colspan="5" class="py-16"><div class="flex justify-center"><LoadingSpinner /></div></td>
          </tr>
          <tr v-else-if="events.length === 0" class="no-hover">
            <td colspan="5" class="py-6"><EmptyState :title="t('admin.schedulingCenter.log.empty')" /></td>
          </tr>
          <tr v-for="ev in events" :key="ev.id" class="align-top" :class="{ 'opacity-60': loading }">
            <td class="whitespace-nowrap text-xs tabular-nums text-gray-500 dark:text-dark-400" :title="formatDateTime(ev.created_at)">
              {{ formatDateTime(ev.created_at) }}
            </td>
            <td class="whitespace-nowrap">
              <button
                type="button"
                class="block max-w-[14rem] truncate text-left font-medium text-gray-900 transition-colors hover:text-primary-600 dark:text-gray-100 dark:hover:text-primary-400"
                :title="ev.account_name"
                @click="emit('open-account', ev.account_id, ev.account_name)"
              >
                {{ ev.account_name }}
              </button>
              <div class="mt-0.5 text-xs tabular-nums text-gray-400 dark:text-dark-400">#{{ ev.account_id }}</div>
            </td>
            <td class="whitespace-nowrap">
              <span class="badge" :class="actionClass(ev)">
                <span class="h-1.5 w-1.5 rounded-full bg-current opacity-70"></span>
                {{ t(`admin.schedulingCenter.log.actions.${ev.action}`) }}
              </span>
            </td>
            <td class="whitespace-nowrap">
              <span v-if="ev.source" class="rounded-md bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">
                {{ t(`admin.schedulingCenter.log.sources.${ev.source}`) }}
              </span>
              <span v-else class="text-gray-300 dark:text-dark-500">—</span>
            </td>
            <td class="min-w-[18rem] text-gray-700 dark:text-gray-300">
              <div class="leading-relaxed">
                {{ describe(ev) }}
                <span v-if="ev.detail.shared" class="text-xs text-gray-400">{{ t('admin.schedulingCenter.log.sharedProbe') }}</span>
              </div>
              <div
                v-if="ev.is_bad && ev.detail.cooldown_minutes != null"
                class="mt-1 inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-400"
              >
                <Icon name="clock" size="xs" />
                {{ t('admin.schedulingCenter.log.cooldown', { n: ev.detail.bad_count, minutes: ev.detail.cooldown_minutes }) }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div class="border-t border-gray-100 px-4 py-2 dark:border-dark-700">
      <Pagination
        :total="total"
        :page="page"
        :page-size="pageSize"
        @update:page="onPage"
        @update:page-size="onPageSize"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import schedulingCenterAPI, { type SchedulingCenterEvent } from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  groupId: number
  keyword: string
}>()

const emit = defineEmits<{ (e: 'open-account', id: number, name: string): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const events = ref<SchedulingCenterEvent[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(50)
const total = ref(0)
let seq = 0

async function load() {
  const current = ++seq
  loading.value = true
  try {
    const res = await schedulingCenterAPI.listEvents(props.groupId, {
      page: page.value,
      page_size: pageSize.value,
      keyword: props.keyword.trim() || undefined
    })
    if (current !== seq) return
    events.value = res.items ?? []
    total.value = res.total ?? 0
  } catch (error) {
    if (current !== seq) return
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.log.loadFailed')))
  } finally {
    if (current === seq) loading.value = false
  }
}

function onPage(p: number) {
  page.value = p
  load()
}

function onPageSize(size: number) {
  pageSize.value = size
  page.value = 1
  load()
}

const reloadFromFirstPage = () => {
  page.value = 1
  load()
}
const debouncedReload = useDebounceFn(reloadFromFirstPage, 300)

watch(() => props.groupId, reloadFromFirstPage)
watch(() => props.keyword, debouncedReload)
onMounted(load)

defineExpose({ reload: load })

// ---------- 展示 ----------

function actionClass(ev: SchedulingCenterEvent): string {
  if (ev.is_bad || ev.action === 'probe_fail' || ev.action === 'recover_fail') {
    return 'badge-danger'
  }
  switch (ev.action) {
    case 'enter':
    case 'recover_pass':
    case 'probe_pass':
      return 'badge-success'
    case 'exit':
      return 'badge-gray'
    default:
      return 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
  }
}

function num(v: unknown): number | null {
  return typeof v === 'number' && Number.isFinite(v) ? v : null
}

function ms(v: unknown): string {
  const n = num(v)
  if (n == null) return '-'
  return `${Math.round(n)}ms`
}

function pct(v: unknown): string {
  const n = num(v)
  return n == null ? '-' : `${Number(n.toFixed(2))}%`
}

function iqText(status: unknown): string {
  switch (status) {
    case 'passed':
      return t('admin.schedulingCenter.iq.passed')
    case 'incorrect':
      return t('admin.schedulingCenter.iq.degraded')
    case 'error':
      return t('admin.schedulingCenter.iq.error')
    default:
      return t('admin.schedulingCenter.iq.unchecked')
  }
}

function describe(ev: SchedulingCenterEvent): string {
  const d = ev.detail || {}
  const r = (key: string, params: Record<string, unknown> = {}) => t(`admin.schedulingCenter.log.reasons.${key}`, params)
  switch (ev.reason) {
    case 'rule_match': {
      const parts: string[] = []
      if (d.cache_rate != null) parts.push(`${t('admin.schedulingCenter.columns.cacheRate')} ${pct(d.cache_rate)}`)
      if (d.first_token_ms != null) parts.push(`${t('admin.schedulingCenter.columns.firstToken')} ${ms(d.first_token_ms)}`)
      if (d.iq_status) parts.push(`${t('admin.schedulingCenter.columns.iq')} ${iqText(d.iq_status)}`)
      return parts.length ? `${r('rule_match')}（${parts.join('，')}）` : r('rule_match')
    }
    case 'exit_cache':
      return r('exit_cache', {
        rate: pct(d.cache_rate),
        threshold: pct(d.threshold),
        entry: pct(d.entry),
        drop: num(d.drop) ?? '-',
        requests: num(d.requests) ?? '-'
      })
    case 'exit_iq':
      return r('exit_iq', { checks: num(d.checks) ?? '-', degraded: num(d.degraded) ?? '-', max: num(d.max) ?? '-' })
    case 'exit_latency': {
      const values = Array.isArray(d.values) ? d.values.map((v) => ms(v)).join(' / ') : '-'
      // 旧日志没有 window，按「连续 N 次」展示
      if (d.window == null) return r('exit_latency', { count: num(d.count) ?? '-', limit: ms(d.limit), values })
      return r('exit_latency_window', { window: num(d.window) ?? '-', slow: num(d.slow) ?? '-', limit: ms(d.limit), values })
    }
    case 'exit_error':
      if (d.window == null) return r('exit_error', { count: num(d.count) ?? '-' })
      return r('exit_error_window', { window: num(d.window) ?? '-', errors: num(d.errors) ?? '-' })
    case 'probe_passed':
      // 当前探测：测速（常规请求 hi）+ 智商检测；兼容旧日志的 skipped / connectivity / iq
      if (d.kind === 'probe') {
        return d.iq_skipped
          ? r('probe_passed_latency_only', { ft: ms(d.first_token_ms), total: ms(d.total_ms) })
          : r('probe_passed_both', { ft: ms(d.first_token_ms), total: ms(d.total_ms) })
      }
      if (d.kind === 'skipped') return r('probe_skipped', { error: (d.error as string) || '-' })
      return d.kind === 'connectivity'
        ? r('probe_passed_connectivity')
        : r('probe_passed_iq', { ft: ms(d.first_token_ms) })
    case 'probe_failed': {
      if (d.latency_failed) return r('probe_failed_latency', { error: (d.latency_error as string) || '-' })
      if (d.slow) return r('probe_failed_slow', { ft: ms(d.first_token_ms), limit: ms(d.limit) })
      if (d.verdict === 'incorrect') return r('probe_failed_degraded')
      return r('probe_failed_error', { error: (d.error as string) || '-' })
    }
    case 'recover_ok':
      return r('recover_ok', { n: num(d.ok_count) ?? '-' })
    case 'recover_error':
      return r('recover_error', { n: num(d.ok_count) ?? 0 })
    case 'recover_latency':
      return r('recover_latency', { ft: ms(d.first_token_ms), limit: ms(d.limit) })
    case 'recovered':
      return r(d.source === 'manual' ? 'recovered_manual' : 'recovered')
    case 'fallback_fill': {
      const parts: string[] = []
      if (d.rate_multiplier != null) parts.push(`${t('admin.schedulingCenter.columns.rateMultiplier')} ${num(d.rate_multiplier)}x`)
      if (d.first_token_ms != null) parts.push(`${t('admin.schedulingCenter.columns.firstToken')} ${ms(d.first_token_ms)}`)
      if (d.cache_rate != null) parts.push(`${t('admin.schedulingCenter.columns.cacheRate')} ${pct(d.cache_rate)}`)
      if (d.iq_status) parts.push(`${t('admin.schedulingCenter.columns.iq')} ${iqText(d.iq_status)}`)
      const base = r('fallback_fill', { min: num(d.min_accounts) ?? 1, rank: num(d.rank) ?? 1 })
      return parts.length ? `${base}（${parts.join('，')}）` : base
    }
    case 'fallback_release':
      return r('fallback_release')
    case 'manual_out_expired':
      return r('manual_out_expired', { hours: num(d.hours) ?? '-' })
    case 'manual_in':
    case 'manual_out':
    case 'cooldown_end':
      return r(ev.reason)
    default:
      return ev.reason || '-'
  }
}
</script>

<style scoped>
.log-table {
  @apply w-full min-w-max text-sm;
}

.log-table thead th {
  @apply sticky top-0 z-10 whitespace-nowrap border-b border-gray-200 bg-gray-50/90 px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 backdrop-blur-sm;
  @apply dark:border-dark-700 dark:bg-dark-800/90 dark:text-dark-400;
}

.log-table tbody td {
  @apply border-b border-gray-100 px-4 py-3 dark:border-dark-700/60;
}

.log-table tbody tr {
  @apply transition-colors duration-150;
}

.log-table tbody tr:not(.no-hover):hover {
  @apply bg-gray-50/80 dark:bg-dark-700/30;
}
</style>
