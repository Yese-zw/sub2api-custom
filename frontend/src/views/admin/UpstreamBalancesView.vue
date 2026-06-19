<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-center">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-72">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.upstreamBalances.searchPlaceholder')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>
            <Select
              :model-value="params.status"
              :options="statusOptions"
              :placeholder="t('admin.accounts.allStatus')"
              class="w-full sm:w-40"
              clearable
              @update:model-value="updateStatusFilter"
              @change="reload"
            />
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading || refreshingAll"
              :title="t('common.refresh')"
              @click="handleRefreshAllClick"
            >
              <Icon name="refresh" size="md" :class="loading || refreshingAll ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="accounts"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="schedulable"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-name="{ row }">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
              <div class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span>#{{ row.id }}</span>
                <span class="text-gray-300 dark:text-dark-700">·</span>
                <span>{{ getBaseURL(row) || '-' }}</span>
              </div>
            </div>
          </template>

          <template #cell-mode="{ row }">
            <span :class="modeBadgeClass(getConfigMode(row))">
              {{ modeLabel(getConfigMode(row)) }}
            </span>
          </template>

          <template #cell-schedulable="{ row }">
            <span :class="schedulableBadgeClass(row)">
              {{ schedulableLabel(row) }}
            </span>
          </template>

          <template #cell-balance_ratio="{ row }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-200">
              {{ formatRatio(getBalanceRatio(row)) }}
            </span>
          </template>

          <template #cell-upstream_balance="{ row }">
            <div class="min-w-[8rem]">
              <div
                v-if="getSnapshot(row)?.error"
                class="truncate text-sm font-medium text-rose-600 dark:text-rose-300"
                :title="String(getSnapshot(row)?.error || '')"
              >
                {{ t('admin.upstreamBalances.queryFailed') }}
              </div>
              <div v-else class="font-medium text-gray-900 dark:text-white">
                {{ formatSnapshotBalance(getSnapshot(row)) }}
              </div>
              <div v-if="getSnapshot(row)?.updated_at" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ formatRelativeTime(getSnapshot(row)?.updated_at) }}
              </div>
            </div>
          </template>

          <template #cell-adjusted_balance="{ row }">
            <span class="font-medium text-gray-900 dark:text-white">
              {{ formatAdjustedBalance(row) }}
            </span>
          </template>

          <template #cell-status="{ row }">
            <div class="flex flex-col gap-1">
              <span :class="statusBadgeClass(row.status)">
                {{ t(`admin.accounts.status.${row.status}`) }}
              </span>
              <span v-if="getRefreshError(row)" class="max-w-[16rem] truncate text-xs text-rose-500" :title="getRefreshError(row)">
                {{ getRefreshError(row) }}
              </span>
              <span v-else-if="getSnapshot(row)?.latency_ms" class="text-xs text-gray-500 dark:text-dark-400">
                {{ getSnapshot(row)?.latency_ms }}ms
              </span>
            </div>
          </template>

          <template #cell-updated_at="{ row }">
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ getSnapshot(row)?.updated_at ? formatDateTime(getSnapshot(row)?.updated_at) : '-' }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <a
                v-if="getWebsiteURL(row)"
                :href="getWebsiteURL(row)"
                target="_blank"
                rel="noopener noreferrer"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('admin.upstreamBalances.openWebsite')"
              >
                <Icon name="externalLink" size="sm" />
              </a>
              <span
                v-else
                class="cursor-not-allowed rounded-lg p-1.5 text-gray-300 opacity-50 dark:text-dark-500"
                :title="t('admin.upstreamBalances.openWebsiteUnavailable')"
              >
                <Icon name="externalLink" size="sm" />
              </span>
              <button
                type="button"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('admin.upstreamBalances.configure')"
                @click="openConfig(row)"
              >
                <Icon name="cog" size="sm" />
              </button>
              <button
                type="button"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-primary-50 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
                :disabled="!canRefresh(row) || isRefreshing(row.id)"
                :title="refreshTitle(row)"
                @click="refreshBalance(row)"
              >
                <Icon name="refresh" size="sm" :class="isRefreshing(row.id) ? 'animate-spin' : ''" />
              </button>
            </div>
          </template>

          <template #empty>
            <div class="flex flex-col items-center">
              <Icon name="database" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">
                {{ t('admin.upstreamBalances.emptyTitle') }}
              </p>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="configDialogOpen"
      :title="t('admin.upstreamBalances.configureTitle')"
      width="normal"
      @close="closeConfig"
    >
      <form id="upstream-balance-config-form" class="space-y-4" @submit.prevent="saveConfig">
        <div>
          <label class="input-label">{{ t('admin.upstreamBalances.account') }}</label>
          <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100">
            {{ configAccount?.name || '-' }}
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstreamBalances.mode') }}</label>
          <Select
            :model-value="configForm.mode"
            :options="modeOptions"
            :placeholder="t('admin.upstreamBalances.autoDetect')"
            clearable
            @update:model-value="updateConfigMode"
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstreamBalances.balanceRatio') }}</label>
          <input
            v-model.number="configForm.balance_ratio"
            type="number"
            class="input"
            min="0"
            step="0.0001"
            required
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstreamBalances.newAPIUserID') }}</label>
          <input
            v-model="configForm.new_api_user_id"
            type="text"
            class="input"
            :placeholder="t('admin.upstreamBalances.newAPIUserIDPlaceholder')"
            autocomplete="off"
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.upstreamBalances.accountAccessKey') }}</label>
          <input
            v-model="configForm.account_access_key"
            type="password"
            class="input"
            :placeholder="accessKeyPlaceholder"
            autocomplete="off"
          />
          <label class="mt-2 inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <input v-model="configForm.clear_access_key" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.upstreamBalances.clearAccessKey') }}</span>
          </label>
        </div>

        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <label class="inline-flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200">
            <input v-model="configForm.low_balance_notify_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.upstreamBalances.lowBalanceNotifyEnabled') }}</span>
          </label>
          <div class="mt-3 grid gap-3 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.upstreamBalances.lowBalanceNotifyThreshold') }}</label>
              <input
                v-model.number="configForm.low_balance_notify_threshold"
                type="number"
                class="input"
                min="0"
                step="0.01"
                :disabled="!configForm.low_balance_notify_enabled"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.upstreamBalances.lowBalanceNotifyEmails') }}</label>
              <textarea
                v-model="configForm.low_balance_notify_emails"
                class="input min-h-[5.5rem]"
                :placeholder="t('admin.upstreamBalances.lowBalanceNotifyEmailsPlaceholder')"
                :disabled="!configForm.low_balance_notify_enabled"
              />
            </div>
          </div>
        </div>
      </form>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeConfig">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="upstream-balance-config-form"
          class="btn btn-primary"
          :disabled="savingConfig"
        >
          <Icon name="check" size="sm" class="mr-2" />
          {{ savingConfig ? t('admin.upstreamBalances.saving') : t('common.save') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import { getPersistedPageSize, setPersistedPageSize } from '@/composables/usePersistedPageSize'
import type { Account, PaginatedResponse, UpstreamBalanceConfig, UpstreamBalanceSnapshot } from '@/types'

type SortOrder = 'asc' | 'desc'
type BalanceMode = NonNullable<UpstreamBalanceConfig['mode']>
type SelectValue = string | number | boolean | null

const { t } = useI18n()
const appStore = useAppStore()

const accounts = ref<Account[]>([])
const loading = ref(false)
const searchQuery = ref('')
const refreshingIds = ref<Set<number>>(new Set())
const refreshingAll = ref(false)
const configDialogOpen = ref(false)
const configAccount = ref<Account | null>(null)
const savingConfig = ref(false)

const params = reactive<{
  status?: string
  search?: string
  sort_by?: string
  sort_order?: SortOrder
}>({
  sort_by: 'schedulable',
  sort_order: 'desc'
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const configForm = reactive<{
  mode: BalanceMode
  balance_ratio: number
  new_api_user_id: string
  low_balance_notify_enabled: boolean
  low_balance_notify_threshold: number
  low_balance_notify_emails: string
  account_access_key: string
  clear_access_key: boolean
}>({
  mode: '',
  balance_ratio: 1,
  new_api_user_id: '',
  low_balance_notify_enabled: false,
  low_balance_notify_threshold: 0,
  low_balance_notify_emails: '',
  account_access_key: '',
  clear_access_key: false
})

const columns = computed(() => [
  { key: 'name', label: t('admin.upstreamBalances.columns.account'), sortable: true },
  { key: 'mode', label: t('admin.upstreamBalances.columns.mode'), sortable: false },
  { key: 'schedulable', label: t('admin.upstreamBalances.columns.schedulable'), sortable: true },
  { key: 'balance_ratio', label: t('admin.upstreamBalances.columns.balanceRatio'), sortable: false },
  { key: 'upstream_balance', label: t('admin.upstreamBalances.columns.upstreamBalance'), sortable: false },
  { key: 'adjusted_balance', label: t('admin.upstreamBalances.columns.adjustedBalance'), sortable: false },
  { key: 'status', label: t('admin.accounts.columns.status'), sortable: true },
  { key: 'updated_at', label: t('admin.upstreamBalances.columns.updatedAt'), sortable: false },
  { key: 'actions', label: t('admin.accounts.columns.actions'), sortable: false }
])

const statusOptions = computed(() => [
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') }
])

const modeOptions = computed(() => [
  { value: 'sub2api', label: 'Sub2API' },
  { value: 'new_api', label: 'New API' }
])

const accessKeyPlaceholder = computed(() => {
  if (configAccount.value?.credentials_status?.has_upstream_balance_access_key) {
    return t('admin.upstreamBalances.accessKeyConfiguredPlaceholder')
  }
  return t('admin.upstreamBalances.accessKeyPlaceholder')
})

const load = async () => {
  loading.value = true
  try {
    const result: PaginatedResponse<Account> = await adminAPI.accounts.list(
      pagination.page,
      pagination.page_size,
      {
        ...params,
        type: 'apikey,upstream',
        sort_by: params.sort_by,
        sort_order: params.sort_order
      }
    )
    accounts.value = result.items || []
    pagination.total = result.total || 0
    pagination.pages = result.pages || 0
  } catch (error) {
    console.error('Failed to load upstream balances:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.upstreamBalances.loadFailed')))
  } finally {
    loading.value = false
  }
}

const reload = () => {
  pagination.page = 1
  return load()
}

const debouncedReload = useDebounceFn(reload, 300)

const handleSearch = () => {
  params.search = searchQuery.value.trim()
  debouncedReload()
}

const selectStringValue = (value: SelectValue): string | undefined => {
  return typeof value === 'string' && value.trim() !== '' ? value : undefined
}

const updateStatusFilter = (value: SelectValue) => {
  params.status = selectStringValue(value)
}

const handlePageChange = (page: number) => {
  pagination.page = Math.max(1, Math.min(page, pagination.pages || 1))
  load()
}

const handlePageSizeChange = (size: number) => {
  pagination.page_size = size
  pagination.page = 1
  setPersistedPageSize(size)
  load()
}

const handleSort = (key: string, order: SortOrder) => {
  params.sort_by = key
  params.sort_order = order
  reload()
}

const patchAccount = (updated: Account) => {
  const index = accounts.value.findIndex(account => account.id === updated.id)
  if (index === -1) return
  const next = [...accounts.value]
  next[index] = updated
  accounts.value = next
  if (configAccount.value?.id === updated.id) {
    configAccount.value = updated
  }
}

const getSnapshot = (account: Account): UpstreamBalanceSnapshot | undefined => {
  return account.extra?.upstream_balance
}

const getConfig = (account: Account): UpstreamBalanceConfig | undefined => {
  return account.extra?.upstream_balance_config
}

const getConfigMode = (account: Account): BalanceMode => {
  return getConfig(account)?.mode || ''
}

const getBalanceRatio = (account: Account): number => {
  const ratio = getConfig(account)?.balance_ratio
  return typeof ratio === 'number' && Number.isFinite(ratio) && ratio >= 0 ? ratio : 1
}

const getBaseURL = (account: Account): string => {
  const value = account.credentials?.base_url
  return typeof value === 'string' ? value : ''
}

const getWebsiteURL = (account: Account): string => {
  const raw = getBaseURL(account).trim()
  if (!raw) return ''
  const withProtocol = /^[a-z][a-z\d+\-.]*:\/\//i.test(raw) ? raw : `https://${raw}`
  try {
    return new URL(withProtocol).origin
  } catch {
    return ''
  }
}

const hasAccessKey = (account: Account): boolean => {
  return account.credentials_status?.has_upstream_balance_access_key === true
}

const hasNewAPIUserID = (account: Account): boolean => {
  const value = getConfig(account)?.new_api_user_id
  return typeof value === 'string' && value.trim() !== ''
}

const hasAPIKey = (account: Account): boolean => {
  return account.credentials_status?.has_api_key === true || typeof account.credentials?.api_key === 'string'
}

const firstFiniteNumber = (...values: Array<number | undefined>): number | null => {
  for (const value of values) {
    if (typeof value === 'number' && Number.isFinite(value)) return value
  }
  return null
}

const snapshotValue = (snapshot: UpstreamBalanceSnapshot | undefined): number | null => {
  return firstFiniteNumber(snapshot?.quota_remaining, snapshot?.remaining, snapshot?.balance)
}

const formatBalanceNumber = (value: number, unit?: string): string => {
  const normalizedUnit = (unit || 'USD').trim().toUpperCase()
  if (normalizedUnit === 'USD') return formatUSD(value)
  return `${value.toFixed(2)} ${normalizedUnit}`
}

const formatUSD = (value: number): string => {
  const sign = value < 0 ? '-' : ''
  return `${sign}${Math.abs(value).toFixed(2)}$`
}

const formatSnapshotBalance = (snapshot: UpstreamBalanceSnapshot | undefined): string => {
  if (!snapshot) return t('admin.upstreamBalances.neverRefreshed')
  const value = snapshotValue(snapshot)
  if (value === null) {
    return snapshot.quota_unlimited ? t('admin.upstreamBalances.unlimited') : '-'
  }
  return formatBalanceNumber(value, snapshot.unit)
}

const formatAdjustedBalance = (account: Account): string => {
  const snapshot = getSnapshot(account)
  const value = snapshotValue(snapshot)
  if (value === null) {
    return snapshot?.quota_unlimited ? t('admin.upstreamBalances.unlimited') : '-'
  }
  return formatBalanceNumber(value * getBalanceRatio(account), snapshot?.unit)
}

const formatRatio = (value: number): string => {
  return Number.isInteger(value) ? `${value}` : value.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')
}

const parseNotifyEmails = (value: string): string[] => {
  return value
    .split(/[\s,;]+/)
    .map(email => email.trim())
    .filter(Boolean)
}

const modeLabel = (mode: BalanceMode): string => {
  if (mode === 'sub2api') return 'Sub2API'
  if (mode === 'new_api') return 'New API'
  return t('admin.upstreamBalances.autoDetect')
}

const modeBadgeClass = (mode: BalanceMode): string => {
  if (mode === 'new_api') return 'badge badge-primary'
  if (mode === 'sub2api') return 'badge badge-success'
  return 'badge badge-gray'
}

const statusBadgeClass = (status: Account['status']): string => {
  if (status === 'active') return 'badge badge-success'
  if (status === 'error') return 'badge badge-danger'
  return 'badge badge-gray'
}

const isSchedulable = (account: Account): boolean => {
  return account.schedulable === true
}

const schedulableLabel = (account: Account): string => {
  return isSchedulable(account) ? t('admin.upstreamBalances.schedulableActive') : t('admin.upstreamBalances.schedulableInactive')
}

const schedulableBadgeClass = (account: Account): string => {
  return isSchedulable(account) ? 'badge badge-success' : 'badge badge-gray'
}

const getRefreshError = (account: Account): string => {
  return String(getSnapshot(account)?.error || '')
}

const canRefresh = (account: Account): boolean => {
  if (account.type !== 'apikey' && account.type !== 'upstream') return false
  if (!getBaseURL(account)) return false
  const mode = getConfigMode(account)
  if (mode === 'new_api') return hasAccessKey(account) && hasNewAPIUserID(account)
  if (mode === 'sub2api') return hasAPIKey(account)
  return hasAPIKey(account) || (hasAccessKey(account) && hasNewAPIUserID(account))
}

const isRefreshing = (accountID: number): boolean => {
  return refreshingIds.value.has(accountID)
}

const setRefreshing = (accountID: number, refreshing: boolean) => {
  const next = new Set(refreshingIds.value)
  if (refreshing) {
    next.add(accountID)
  } else {
    next.delete(accountID)
  }
  refreshingIds.value = next
}

const refreshTitle = (account: Account): string => {
  if (isRefreshing(account.id)) return t('admin.upstreamBalances.refreshing')
  if (!canRefresh(account)) return t('admin.upstreamBalances.refreshUnsupported')
  return t('admin.upstreamBalances.refresh')
}

const refreshBalance = async (account: Account) => {
  if (!canRefresh(account) || isRefreshing(account.id)) return
  setRefreshing(account.id, true)
  try {
    const updated = await adminAPI.accounts.refreshUpstreamBalance(account.id)
    patchAccount(updated)
    appStore.showSuccess(t('admin.upstreamBalances.refreshSuccess'))
  } catch (error) {
    console.error('Failed to refresh upstream balance:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.upstreamBalances.refreshFailed')))
  } finally {
    setRefreshing(account.id, false)
  }
}

const refreshAllBalances = async (showToast = true) => {
  if (refreshingAll.value) return
  refreshingAll.value = true
  loading.value = true
  try {
    const result = await adminAPI.accounts.refreshUpstreamBalances({
      status: params.status,
      search: params.search
    })
    for (const updated of result.items || []) {
      patchAccount(updated)
    }
    await load()
    if (showToast) {
      const skipped = result.skipped || 0
      if (result.failed > 0) {
        appStore.showError(t('admin.upstreamBalances.refreshPartial', { success: result.success, skipped, failed: result.failed }))
      } else {
        appStore.showSuccess(t('admin.upstreamBalances.refreshAllSuccess', { count: result.success, skipped }))
      }
    }
  } catch (error) {
    console.error('Failed to refresh all upstream balances:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.upstreamBalances.refreshFailed')))
  } finally {
    refreshingAll.value = false
    loading.value = false
  }
}

const handleRefreshAllClick = () => {
  refreshAllBalances(true)
}

const openConfig = (account: Account) => {
  configAccount.value = account
  const config = getConfig(account)
  configForm.mode = config?.mode || ''
  configForm.balance_ratio = getBalanceRatio(account)
  configForm.new_api_user_id = config?.new_api_user_id || ''
  configForm.low_balance_notify_enabled = config?.low_balance_notify_enabled === true
  configForm.low_balance_notify_threshold = config?.low_balance_notify_threshold || 0
  configForm.low_balance_notify_emails = (config?.low_balance_notify_emails || []).join('\n')
  configForm.account_access_key = ''
  configForm.clear_access_key = false
  configDialogOpen.value = true
}

const updateConfigMode = (value: SelectValue) => {
  const mode = typeof value === 'string' ? value : ''
  configForm.mode = mode === 'sub2api' || mode === 'new_api' ? mode : ''
}

const closeConfig = () => {
  configDialogOpen.value = false
  configAccount.value = null
}

const saveConfig = async () => {
  if (!configAccount.value) return
  if (!Number.isFinite(configForm.balance_ratio) || configForm.balance_ratio < 0) {
    appStore.showError(t('admin.upstreamBalances.invalidRatio'))
    return
  }
  const notifyEmails = parseNotifyEmails(configForm.low_balance_notify_emails)
  if (configForm.low_balance_notify_enabled) {
    if (!Number.isFinite(configForm.low_balance_notify_threshold) || configForm.low_balance_notify_threshold <= 0) {
      appStore.showError(t('admin.upstreamBalances.invalidLowBalanceThreshold'))
      return
    }
    if (notifyEmails.length === 0) {
      appStore.showError(t('admin.upstreamBalances.invalidLowBalanceEmails'))
      return
    }
  }
  savingConfig.value = true
  try {
    const updated = await adminAPI.accounts.updateUpstreamBalanceConfig(configAccount.value.id, {
      mode: configForm.mode || '',
      balance_ratio: configForm.balance_ratio,
      new_api_user_id: configForm.new_api_user_id.trim(),
      low_balance_notify_enabled: configForm.low_balance_notify_enabled,
      low_balance_notify_threshold: configForm.low_balance_notify_threshold || 0,
      low_balance_notify_emails: notifyEmails,
      account_access_key: configForm.account_access_key.trim(),
      clear_access_key: configForm.clear_access_key
    })
    patchAccount(updated)
    appStore.showSuccess(t('admin.upstreamBalances.configSaved'))
    closeConfig()
  } catch (error) {
    console.error('Failed to save upstream balance config:', error)
    appStore.showError(extractApiErrorMessage(error, t('admin.upstreamBalances.configSaveFailed')))
  } finally {
    savingConfig.value = false
  }
}

onMounted(() => {
  refreshAllBalances(false)
})
</script>
