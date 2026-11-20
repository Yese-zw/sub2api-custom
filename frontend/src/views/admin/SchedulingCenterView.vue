<template>
  <AppLayout>
    <div class="flex flex-col gap-4 lg:h-[calc(100vh-64px-4rem)] lg:flex-row">
      <!-- 左侧：分组列表 -->
      <aside class="card flex w-full flex-col overflow-hidden lg:w-56 lg:flex-shrink-0 xl:w-60">
        <div class="space-y-2 border-b border-gray-100 p-3 dark:border-dark-700">
          <div class="flex items-center justify-between">
            <div class="min-w-0">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.schedulingCenter.groups') }}
              </h2>
              <p class="truncate text-[11px] text-gray-400 dark:text-dark-400">
                {{ t('admin.schedulingCenter.groupCount', { n: filteredGroups.length, on: enabledGroupCount }) }}
              </p>
            </div>
            <button
              type="button"
              class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:opacity-50 dark:hover:bg-dark-700 dark:hover:text-gray-200"
              :title="t('common.refresh')"
              :disabled="groupsLoading"
              @click="loadGroups"
            >
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': groupsLoading }" />
            </button>
          </div>
          <div class="relative">
            <Icon
              name="search"
              size="sm"
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model="groupKeyword"
              type="text"
              class="input py-1.5 pl-9 text-sm"
              :placeholder="t('admin.schedulingCenter.searchGroup')"
            />
          </div>
        </div>

        <div class="max-h-56 flex-1 space-y-0.5 overflow-y-auto p-1.5 lg:max-h-none">
          <div v-if="groupsLoading && groups.length === 0" class="flex justify-center py-10">
            <LoadingSpinner />
          </div>
          <div v-else-if="filteredGroups.length === 0" class="py-10 text-center text-sm text-gray-400 dark:text-dark-400">
            {{ t('admin.schedulingCenter.noGroups') }}
          </div>
          <button
            v-for="group in filteredGroups"
            :key="group.id"
            type="button"
            class="group relative flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-all duration-150"
            :class="
              group.id === selectedGroupId
                ? 'bg-primary-50 ring-1 ring-primary-200 dark:bg-primary-900/20 dark:ring-primary-800/60'
                : 'hover:bg-gray-50 dark:hover:bg-dark-700/60'
            "
            @click="selectGroup(group.id)"
          >
            <span
              class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-md transition-colors"
              :class="
                group.id === selectedGroupId
                  ? 'bg-white text-primary-600 shadow-sm dark:bg-dark-800 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-500 group-hover:bg-white group-hover:shadow-sm dark:bg-dark-700 dark:text-gray-400 dark:group-hover:bg-dark-800'
              "
            >
              <PlatformIcon :platform="group.platform" size="xs" />
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1.5">
                <span
                  class="truncate text-sm font-medium"
                  :class="
                    group.id === selectedGroupId
                      ? 'text-primary-700 dark:text-primary-300'
                      : group.status === 'active'
                        ? 'text-gray-800 dark:text-gray-200'
                        : 'text-gray-400 dark:text-dark-400'
                  "
                >
                  {{ group.name }}
                </span>
                <span
                  v-if="settingsByGroup.get(group.id)?.enabled"
                  class="relative flex h-2 w-2 flex-shrink-0"
                  :title="t('admin.schedulingCenter.schedulingOn')"
                >
                  <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-60"></span>
                  <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
                </span>
                <span v-if="group.status !== 'active'" class="badge badge-gray ml-auto flex-shrink-0 px-1.5 py-0 text-[10px]">
                  {{ t('admin.schedulingCenter.inactive') }}
                </span>
              </div>
            </div>
            <span class="flex-shrink-0 text-[11px] tabular-nums text-gray-400 dark:text-dark-400">
              {{ group.active_account_count ?? 0 }}/{{ group.account_count ?? 0 }}
            </span>
          </button>
        </div>
      </aside>

      <!-- 右侧 -->
      <section class="flex min-w-0 flex-1 flex-col gap-4">
        <div v-if="!selectedGroup" class="card flex flex-1 items-center justify-center py-20">
          <EmptyState :title="t('admin.schedulingCenter.selectGroupHint')" />
        </div>

        <template v-else>
          <!-- 分组概览 + 指标 -->
          <div class="card flex-shrink-0">
            <div class="flex flex-col gap-3 p-4 md:flex-row md:items-start md:justify-between">
              <div class="flex min-w-0 items-start gap-3">
                <span
                  class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-primary-50 to-primary-100 text-primary-600 ring-1 ring-primary-200/60 dark:from-primary-900/40 dark:to-primary-900/10 dark:text-primary-400 dark:ring-primary-800/50"
                >
                  <PlatformIcon :platform="selectedGroup.platform" size="md" />
                </span>
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h1 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
                      {{ selectedGroup.name }}
                    </h1>
                    <span class="text-xs text-gray-400 dark:text-dark-400">#{{ selectedGroup.id }}</span>
                    <span v-if="currentSettings?.enabled" class="badge badge-success">
                      <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                      {{ t('admin.schedulingCenter.schedulingEnabled') }}
                    </span>
                    <span v-else class="badge badge-gray">
                      <span class="h-1.5 w-1.5 rounded-full bg-gray-400"></span>
                      {{ t('admin.schedulingCenter.statusLine.off') }}
                    </span>
                  </div>
                  <div class="mt-2 flex flex-wrap items-center gap-1.5">
                    <span v-for="chip in overviewChips" :key="chip.key" class="meta-chip" :title="chip.title">
                      <Icon :name="chip.icon" size="xs" class="text-gray-400 dark:text-dark-400" />
                      <span class="text-gray-500 dark:text-dark-400">{{ chip.label }}</span>
                      <span class="font-medium" :class="chip.muted ? 'text-gray-400 dark:text-dark-400' : 'text-gray-700 dark:text-gray-200'">
                        {{ chip.value }}
                      </span>
                    </span>
                  </div>
                </div>
              </div>
              <div class="flex flex-shrink-0 items-center gap-2">
                <button type="button" class="btn btn-secondary" @click="settingsOpen = true">
                  <Icon name="cog" size="sm" />
                  {{ t('admin.schedulingCenter.settingsButton') }}
                </button>
                <button type="button" class="btn btn-secondary" @click="exportOpen = true">
                  <Icon name="download" size="sm" />
                  {{ t('admin.schedulingCenter.export.button') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-icon"
                  :title="t('common.refresh')"
                  :disabled="accountsLoading"
                  @click="loadGroupData"
                >
                  <Icon name="refresh" size="sm" :class="{ 'animate-spin': accountsLoading }" />
                </button>
              </div>
            </div>

            <div
              class="grid grid-cols-2 divide-gray-100 border-t border-gray-100 dark:divide-dark-700 dark:border-dark-700 xl:grid-cols-4 xl:divide-x"
            >
              <div
                v-for="(stat, i) in stats"
                :key="stat.key"
                class="flex items-center gap-3 px-4 py-3"
                :class="{ 'border-t border-gray-100 dark:border-dark-700 xl:border-t-0': i >= 2, 'border-l border-gray-100 dark:border-dark-700 xl:border-l-0': i % 2 === 1 }"
              >
                <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg" :class="stat.iconClass">
                  <Icon :name="stat.icon" size="sm" />
                </span>
                <div class="min-w-0">
                  <p class="truncate text-xs text-gray-500 dark:text-dark-400">{{ stat.label }}</p>
                  <p class="flex items-baseline gap-1.5">
                    <span class="text-lg font-bold tabular-nums text-gray-900 dark:text-white">{{ stat.value }}</span>
                    <span v-if="stat.sub" class="truncate text-xs text-gray-400 dark:text-dark-400">{{ stat.sub }}</span>
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- 账号 / 日志 -->
          <div class="card flex min-h-[28rem] flex-1 flex-col overflow-hidden">
            <div
              class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between"
            >
              <div class="tabs inline-flex w-full sm:w-auto" role="tablist">
                <button
                  v-for="tab in poolTabs"
                  :key="tab.value"
                  type="button"
                  role="tab"
                  class="tab inline-flex flex-1 items-center justify-center gap-1.5 sm:flex-none"
                  :class="activeTab === tab.value ? 'tab-active' : ''"
                  :aria-selected="activeTab === tab.value"
                  @click="switchPool(tab.value)"
                >
                  <Icon :name="tab.icon" size="sm" />
                  {{ tab.label }}
                  <span
                    v-if="tab.count != null"
                    class="rounded-full px-1.5 text-[11px] tabular-nums"
                    :class="
                      activeTab === tab.value
                        ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300'
                        : 'bg-gray-200/80 text-gray-500 dark:bg-dark-700 dark:text-dark-300'
                    "
                  >
                    {{ tab.count }}
                  </span>
                </button>
              </div>
              <div v-if="activeTab !== 'compare'" class="flex flex-wrap items-center gap-2">
                <div class="relative w-full sm:w-60">
                  <Icon
                    name="search"
                    size="sm"
                    class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
                  />
                  <input
                    v-model="accountKeyword"
                    type="text"
                    class="input py-2 pl-9"
                    :placeholder="t('admin.schedulingCenter.searchAccount')"
                    @input="page = 1"
                  />
                </div>
                <template v-if="activeTab !== 'log'">
                  <span v-if="selectedIds.size > 0" class="badge badge-primary">
                    {{ t('admin.schedulingCenter.selectedCount', { n: selectedIds.size }) }}
                  </span>
                  <button
                    type="button"
                    class="btn"
                    :class="activePool === 'account' ? 'btn-primary' : 'btn-secondary'"
                    :disabled="selectedIds.size === 0 || moving"
                    @click="moveSelected"
                  >
                    <Icon :name="activePool === 'account' ? 'arrowRight' : 'arrowLeft'" size="sm" />
                    {{ activePool === 'account' ? t('admin.schedulingCenter.moveToScheduling') : t('admin.schedulingCenter.moveToAccount') }}
                  </button>
                </template>
              </div>
            </div>

            <SchedulingLogPanel
              v-if="activeTab === 'log'"
              ref="logPanel"
              :group-id="selectedGroup.id"
              :keyword="accountKeyword"
              @open-account="openDetailById"
            />

            <ComparisonPanel v-else-if="activeTab === 'compare'" :group-id="selectedGroup.id" />

            <template v-else>
              <div class="flex-1 overflow-auto">
                <table class="sc-table">
                  <thead>
                    <tr>
                      <th class="w-11 !pr-0">
                        <input
                          type="checkbox"
                          class="sc-checkbox"
                          :checked="allPageSelected"
                          :indeterminate="somePageSelected && !allPageSelected"
                          :disabled="pagedAccounts.length === 0"
                          @change="togglePage"
                        />
                      </th>
                      <th>{{ t('admin.schedulingCenter.columns.name') }}</th>
                      <th :title="t('admin.schedulingCenter.centerStatusHint')">{{ t('admin.schedulingCenter.columns.centerStatus') }}</th>
                      <th class="text-right">{{ t('admin.schedulingCenter.columns.rateMultiplier') }}</th>
                      <th :title="t('admin.schedulingCenter.window24h')">{{ t('admin.schedulingCenter.columns.cacheRate') }}</th>
                      <th>{{ t('admin.schedulingCenter.columns.iq') }}</th>
                      <th class="text-right" :title="t('admin.schedulingCenter.firstTokenHint', { n: ttftSamples })">
                        {{ t('admin.schedulingCenter.columns.firstToken') }}
                      </th>
                      <th class="text-right">{{ t('admin.schedulingCenter.columns.actions') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-if="accountsLoading && allAccounts.length === 0" class="no-hover">
                      <td colspan="8" class="py-16"><div class="flex justify-center"><LoadingSpinner /></div></td>
                    </tr>
                    <tr v-else-if="pagedAccounts.length === 0" class="no-hover">
                      <td colspan="8" class="py-6">
                        <EmptyState
                          :title="activePool === 'scheduling' ? t('admin.schedulingCenter.emptySchedulingPoolTitle') : t('admin.schedulingCenter.noAccounts')"
                          :description="activePool === 'scheduling' ? t('admin.schedulingCenter.emptySchedulingPool') : ''"
                        />
                      </td>
                    </tr>
                    <tr
                      v-for="account in pagedAccounts"
                      :key="account.id"
                      :class="{ 'opacity-60': accountsLoading, 'is-selected': selectedIds.has(account.id) }"
                    >
                      <td class="!pr-0">
                        <input
                          type="checkbox"
                          class="sc-checkbox"
                          :checked="selectedIds.has(account.id)"
                          @change="toggleOne(account.id)"
                        />
                      </td>
                      <td>
                        <button
                          type="button"
                          class="block max-w-[16rem] truncate text-left font-medium text-gray-900 transition-colors hover:text-primary-600 dark:text-gray-100 dark:hover:text-primary-400"
                          :title="account.name"
                          @click="openDetail(account)"
                        >
                          {{ account.name }}
                        </button>
                        <div class="mt-0.5 flex items-center gap-1.5 text-xs text-gray-400 dark:text-dark-400">
                          <span class="tabular-nums">#{{ account.id }}</span>
                          <span
                            v-if="memberSource.get(account.id) === 'rule'"
                            class="inline-flex items-center gap-0.5 rounded-md bg-sky-50 px-1.5 text-[10px] font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-300"
                          >
                            <Icon name="bolt" size="xs" />
                            {{ t('admin.schedulingCenter.sourceRule') }}
                          </span>
                        </div>
                      </td>
                      <td>
                        <span class="badge" :class="toneClass[centerStatus(account.id).tone]" :title="centerStatus(account.id).title">
                          <span class="h-1.5 w-1.5 rounded-full" :class="toneDot[centerStatus(account.id).tone]"></span>
                          {{ t(`admin.schedulingCenter.centerStatus.${centerStatus(account.id).key}`) }}
                          <span v-if="centerStatus(account.id).extra" class="tabular-nums opacity-75">{{ centerStatus(account.id).extra }}</span>
                        </span>
                      </td>
                      <td class="text-right tabular-nums">
                        <span class="rounded-md bg-gray-100 px-1.5 py-0.5 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300">
                          {{ formatMultiplier(account.rate_multiplier) }}
                        </span>
                      </td>
                      <td>
                        <div v-if="metricsOf(account.id)?.cache_rate != null" class="flex items-center gap-2">
                          <div class="h-1.5 w-14 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                            <div
                              class="h-full rounded-full"
                              :class="cacheRateBar(metricsOf(account.id)!.cache_rate!)"
                              :style="{ width: `${Math.min(100, Math.max(0, metricsOf(account.id)!.cache_rate!))}%` }"
                            ></div>
                          </div>
                          <span class="text-xs font-medium tabular-nums" :class="cacheRateClass(metricsOf(account.id)!.cache_rate!)">
                            {{ metricsOf(account.id)!.cache_rate!.toFixed(1) }}%
                          </span>
                        </div>
                        <span v-else class="text-gray-300 dark:text-dark-500">{{ metricsLoading ? '…' : '—' }}</span>
                      </td>
                      <td>
                        <span class="badge" :class="toneClass[iqView(account.id).tone]" :title="iqTitle(account.id)">
                          {{ t(`admin.schedulingCenter.iq.${iqView(account.id).key}`) }}
                        </span>
                      </td>
                      <td class="text-right tabular-nums" :title="firstTokenTitle(account.id)">
                        <template v-if="metricsOf(account.id)?.first_token_ms != null">
                          <span class="text-sm font-medium" :class="ttftClass(metricsOf(account.id)!.first_token_ms!)">
                            {{ formatMs(metricsOf(account.id)!.first_token_ms!) }}
                          </span>
                          <span v-if="metricsOf(account.id)?.first_token_source === 'traffic'" class="ml-0.5 text-[10px] text-gray-400">*</span>
                        </template>
                        <span v-else class="text-gray-300 dark:text-dark-500">{{ metricsLoading ? '…' : '—' }}</span>
                      </td>
                      <td>
                        <div class="flex items-center justify-end gap-1">
                          <button type="button" class="row-action" @click="openDetail(account)">
                            <Icon name="chart" size="sm" />
                            <span>{{ t('admin.schedulingCenter.viewDetail') }}</span>
                          </button>
                          <button
                            type="button"
                            class="row-action"
                            :class="activePool === 'account' ? 'row-action-primary' : 'row-action-warn'"
                            :disabled="moving"
                            @click="moveAccounts([account.id], activePool === 'account' ? 'scheduling' : 'account')"
                          >
                            <Icon :name="activePool === 'account' ? 'arrowRight' : 'arrowLeft'" size="sm" />
                            <span>{{ activePool === 'account' ? t('admin.schedulingCenter.moveIn') : t('admin.schedulingCenter.moveOut') }}</span>
                          </button>
                        </div>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <div class="border-t border-gray-100 px-4 py-2 dark:border-dark-700">
                <Pagination
                  :total="filteredAccounts.length"
                  :page="page"
                  :page-size="pageSize"
                  @update:page="page = $event"
                  @update:page-size="onPageSizeChange"
                />
              </div>
            </template>
          </div>
        </template>
      </section>
    </div>

    <GroupSettingsDialog
      :show="settingsOpen"
      :group-id="selectedGroupId"
      :group-name="selectedGroup?.name ?? ''"
      @close="settingsOpen = false"
      @saved="onSettingsSaved"
    />
    <ExportDialog
      :show="exportOpen"
      :group-id="selectedGroupId"
      :group-name="selectedGroup?.name ?? ''"
      @close="exportOpen = false"
    />
    <AccountRecordsDialog
      :show="detailAccount != null"
      :account-id="detailAccount?.id ?? null"
      :account-name="detailAccount?.name ?? ''"
      @close="detailAccount = null"
    />
  </AppLayout>
</template>


<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupSettingsDialog from '@/components/schedulingCenter/GroupSettingsDialog.vue'
import AccountRecordsDialog from '@/components/schedulingCenter/AccountRecordsDialog.vue'
import SchedulingLogPanel from '@/components/schedulingCenter/SchedulingLogPanel.vue'
import ComparisonPanel from '@/components/schedulingCenter/ComparisonPanel.vue'
import ExportDialog from '@/components/schedulingCenter/ExportDialog.vue'
import { adminAPI } from '@/api/admin'
import schedulingCenterAPI, {
  type SchedulingCenterAccountMetrics,
  type SchedulingCenterAccountState,
  type SchedulingCenterGroupSettings,
  type SchedulingCenterMemberSource,
  type SchedulingCenterPool,
  type SchedulingCenterPoolMember
} from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { AccountListItem, AdminGroup } from '@/types'

type IconName = InstanceType<typeof Icon>['$props']['name']

const { t } = useI18n()
const appStore = useAppStore()
const route = useRoute()
const router = useRouter()

// ---------- 分组 & 设置 ----------
const groups = ref<AdminGroup[]>([])
const groupsLoading = ref(false)
const groupKeyword = ref('')
const selectedGroupId = ref<number | null>(null)
const settingsByGroup = ref<Map<number, SchedulingCenterGroupSettings>>(new Map())
const settingsOpen = ref(false)

const filteredGroups = computed(() => {
  const kw = groupKeyword.value.trim().toLowerCase()
  if (!kw) return groups.value
  return groups.value.filter(
    (g) => g.name.toLowerCase().includes(kw) || String(g.id) === kw || g.platform.toLowerCase().includes(kw)
  )
})

const selectedGroup = computed(() => groups.value.find((g) => g.id === selectedGroupId.value) ?? null)
const currentSettings = computed(() =>
  selectedGroupId.value != null ? settingsByGroup.value.get(selectedGroupId.value) ?? null : null
)

const enabledGroupCount = computed(() => groups.value.filter((g) => settingsByGroup.value.get(g.id)?.enabled).length)

const CRON_PRESET_KEYS: Record<string, string> = {
  '*/5 * * * *': 'every5m',
  '*/10 * * * *': 'every10m',
  '*/15 * * * *': 'every15m',
  '*/30 * * * *': 'every30m',
  '0 * * * *': 'hourly',
  '0 */6 * * *': 'every6h'
}

interface OverviewChip {
  key: string
  icon: IconName
  label: string
  value: string
  title?: string
  muted?: boolean
}

const overviewChips = computed<OverviewChip[]>(() => {
  const s = currentSettings.value
  const enabled = !!s?.enabled
  const off = t('admin.schedulingCenter.overview.off')
  const cron = s?.cron_expression ?? ''
  const preset = CRON_PRESET_KEYS[cron]
  return [
    {
      key: 'cron',
      icon: 'clock',
      label: t('admin.schedulingCenter.settings.frequency'),
      value: enabled ? (preset ? t(`admin.schedulingCenter.settings.presets.${preset}`) : cron) : off,
      title: enabled ? cron : undefined,
      muted: !enabled
    },
    {
      key: 'model',
      icon: 'cpu',
      label: t('admin.schedulingCenter.settings.model'),
      value: enabled ? s?.model_id || t('admin.schedulingCenter.statusLine.defaultModel') : off,
      muted: !enabled
    },
    {
      key: 'entry',
      icon: 'arrowRight',
      label: t('admin.schedulingCenter.settings.ruleEnabled'),
      value: enabled && s?.rule_enabled ? ruleSummary.value : off,
      muted: !(enabled && s?.rule_enabled)
    },
    {
      key: 'exit',
      icon: 'arrowLeft',
      label: t('admin.schedulingCenter.settings.exitEnabled'),
      value: enabled && s?.exit_enabled ? t('admin.schedulingCenter.overview.on') : off,
      muted: !(enabled && s?.exit_enabled)
    }
  ]
})

const ttftSamples = computed(() => currentSettings.value?.first_token_test_samples || 10)

const ruleSummary = computed(() => {
  const s = currentSettings.value
  if (!s) return ''
  const parts: string[] = []
  if (s.min_cache_rate != null) parts.push(`${t('admin.schedulingCenter.columns.cacheRate')} ≥ ${s.min_cache_rate}%`)
  if (s.require_iq_pass) parts.push(t('admin.schedulingCenter.settings.requireIQPass', { n: s.entry_iq_pass_count || 1 }))
  if (s.max_first_token_ms != null) parts.push(`${t('admin.schedulingCenter.columns.firstToken')} ≤ ${s.max_first_token_ms}ms`)
  return parts.length ? parts.join(' · ') : t('admin.schedulingCenter.statusLine.noCondition')
})

async function loadSettings() {
  try {
    const list = await schedulingCenterAPI.listSettings()
    settingsByGroup.value = new Map(list.map((s) => [s.group_id, s]))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.settings.loadFailed')))
  }
}

async function loadGroups() {
  groupsLoading.value = true
  try {
    const [list] = await Promise.all([adminAPI.groups.getAllIncludingInactive(), loadSettings()])
    groups.value = [...list].sort((a, b) => Number(b.status === 'active') - Number(a.status === 'active'))
    if (selectedGroupId.value == null || !groups.value.some((g) => g.id === selectedGroupId.value)) {
      const fromQuery = Number(route.query.group)
      const initial = groups.value.find((g) => g.id === fromQuery) ?? groups.value[0]
      if (initial) selectGroup(initial.id)
    }
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.loadGroupsFailed')))
  } finally {
    groupsLoading.value = false
  }
}

function selectGroup(id: number) {
  if (selectedGroupId.value === id) return
  selectedGroupId.value = id
  allAccounts.value = []
  members.value = []
  accountStates.value = []
  metrics.value = {}
  selectedIds.value = new Set()
  page.value = 1
  if (Number(route.query.group) !== id) {
    router.replace({ query: { ...route.query, group: String(id) } })
  }
  loadGroupData()
}

function onSettingsSaved(saved: SchedulingCenterGroupSettings) {
  const next = new Map(settingsByGroup.value)
  next.set(saved.group_id, saved)
  settingsByGroup.value = next
  settingsOpen.value = false
  // 保存后后端已同步计划并评估规则，刷新池归属
  if (saved.group_id === selectedGroupId.value) loadGroupData()
}

// ---------- 账号 & 池 ----------
const ACCOUNT_PAGE_FETCH = 500
const METRICS_CHUNK = 500

const allAccounts = ref<AccountListItem[]>([])
const members = ref<SchedulingCenterPoolMember[]>([])
const metrics = ref<Record<string, SchedulingCenterAccountMetrics>>({})
const accountsLoading = ref(false)
const metricsLoading = ref(false)
const moving = ref(false)
type CenterTab = 'log' | 'compare' | SchedulingCenterPool
const exportOpen = ref(false)
const activeTab = ref<CenterTab>('account')
const activePool = computed<SchedulingCenterPool>(() => (activeTab.value === 'scheduling' ? 'scheduling' : 'account'))
const accountStates = ref<SchedulingCenterAccountState[]>([])
const stateByAccount = computed(
  () => new Map<number, SchedulingCenterAccountState>(accountStates.value.map((st) => [st.account_id, st]))
)
const memberByAccount = computed(() => new Map(members.value.map((m) => [m.account_id, m])))
const logPanel = ref<InstanceType<typeof SchedulingLogPanel> | null>(null)
const accountKeyword = ref('')
const page = ref(1)
const pageSize = ref(50)
const selectedIds = ref<Set<number>>(new Set())
const detailAccount = ref<AccountListItem | null>(null)
let loadSeq = 0

const memberSource = computed(
  () => new Map<number, SchedulingCenterMemberSource>(members.value.map((m) => [m.account_id, m.source]))
)
const schedulingPoolAccounts = computed(() => allAccounts.value.filter((a) => memberSource.value.has(a.id)))
const accountPoolAccounts = computed(() => allAccounts.value.filter((a) => !memberSource.value.has(a.id)))

const poolTabs = computed<{ value: CenterTab; label: string; icon: IconName; count?: number }[]>(() => [
  { value: 'account', icon: 'users', label: t('admin.schedulingCenter.accountPool'), count: accountPoolAccounts.value.length },
  { value: 'scheduling', icon: 'bolt', label: t('admin.schedulingCenter.schedulingPool'), count: schedulingPoolAccounts.value.length },
  { value: 'log', icon: 'document', label: t('admin.schedulingCenter.schedulingLog') },
  { value: 'compare', icon: 'chartBar', label: t('admin.schedulingCenter.compare.tab') }
])

interface StatItem {
  key: string
  icon: IconName
  iconClass: string
  label: string
  value: string | number
  sub?: string
}

const stats = computed<StatItem[]>(() => {
  const total = allAccounts.value.length
  const inPool = schedulingPoolAccounts.value.length
  const recovering = accountStates.value.filter(
    (st) => (st.state === 'cooling' || st.state === 'probing') && !memberByAccount.value.has(st.account_id)
  ).length
  const degraded = allAccounts.value.filter((a) => metricsOf(a.id)?.iq_status === 'incorrect').length
  const poolRates = schedulingPoolAccounts.value
    .map((a) => metricsOf(a.id)?.cache_rate)
    .filter((v): v is number => v != null)
  const avgCache = poolRates.length ? poolRates.reduce((x, y) => x + y, 0) / poolRates.length : null
  return [
    {
      key: 'total',
      icon: 'users',
      iconClass: 'stat-icon-primary',
      label: t('admin.schedulingCenter.stats.total'),
      value: total,
      sub: t('admin.schedulingCenter.stats.totalSub', { n: accountPoolAccounts.value.length })
    },
    {
      key: 'pool',
      icon: 'bolt',
      iconClass: 'stat-icon-success',
      label: t('admin.schedulingCenter.stats.inPool'),
      value: inPool,
      sub: total ? `${Math.round((inPool / total) * 100)}%` : undefined
    },
    {
      key: 'recovering',
      icon: 'clock',
      iconClass: 'stat-icon-warning',
      label: t('admin.schedulingCenter.stats.recovering'),
      value: recovering
    },
    {
      key: 'quality',
      icon: 'exclamationTriangle',
      iconClass: 'stat-icon-danger',
      label: t('admin.schedulingCenter.stats.degraded'),
      value: metricsLoading.value && Object.keys(metrics.value).length === 0 ? '…' : degraded,
      sub: avgCache != null ? t('admin.schedulingCenter.stats.poolCache', { rate: avgCache.toFixed(1) }) : undefined
    }
  ]
})

const filteredAccounts = computed(() => {
  const source = activePool.value === 'scheduling' ? schedulingPoolAccounts.value : accountPoolAccounts.value
  const kw = accountKeyword.value.trim().toLowerCase()
  if (!kw) return source
  return source.filter((a) => a.name.toLowerCase().includes(kw) || String(a.id) === kw)
})

const pagedAccounts = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredAccounts.value.slice(start, start + pageSize.value)
})

watch(
  () => filteredAccounts.value.length,
  (len) => {
    const maxPage = Math.max(1, Math.ceil(len / pageSize.value))
    if (page.value > maxPage) page.value = maxPage
  }
)

async function fetchAllGroupAccounts(groupId: number): Promise<AccountListItem[]> {
  const out: AccountListItem[] = []
  for (let p = 1; ; p++) {
    const res = await adminAPI.accounts.list(p, ACCOUNT_PAGE_FETCH, {
      group: String(groupId),
      sort_by: 'priority',
      sort_order: 'asc'
    })
    const items = res.items ?? []
    out.push(...items)
    if (items.length < ACCOUNT_PAGE_FETCH || out.length >= (res.total ?? 0)) break
  }
  return out
}

async function loadGroupData() {
  const groupId = selectedGroupId.value
  if (groupId == null) return
  const seq = ++loadSeq
  accountsLoading.value = true
  try {
    const [accounts, pool] = await Promise.all([
      fetchAllGroupAccounts(groupId),
      schedulingCenterAPI.getPool(groupId)
    ])
    if (seq !== loadSeq) return
    allAccounts.value = accounts
    members.value = pool.members
    accountStates.value = pool.states
    const alive = new Set(accounts.map((a) => a.id))
    selectedIds.value = new Set([...selectedIds.value].filter((id) => alive.has(id)))
  } catch (error) {
    if (seq !== loadSeq) return
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.loadAccountsFailed')))
    return
  } finally {
    if (seq === loadSeq) accountsLoading.value = false
  }
  loadMetrics(seq)
}

async function loadMetrics(seq: number) {
  const ids = allAccounts.value.map((a) => a.id)
  if (ids.length === 0) return
  metricsLoading.value = true
  try {
    const merged: Record<string, SchedulingCenterAccountMetrics> = {}
    for (let i = 0; i < ids.length; i += METRICS_CHUNK) {
      Object.assign(merged, await schedulingCenterAPI.getMetrics(ids.slice(i, i + METRICS_CHUNK), selectedGroupId.value ?? undefined))
      if (seq !== loadSeq) return
    }
    metrics.value = merged
  } catch (error) {
    if (seq !== loadSeq) return
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.loadMetricsFailed')))
  } finally {
    if (seq === loadSeq) metricsLoading.value = false
  }
}

function switchPool(tab: CenterTab) {
  if (activeTab.value === tab) return
  activeTab.value = tab
  selectedIds.value = new Set()
  page.value = 1
}

function onPageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
}

function openDetail(account: AccountListItem) {
  detailAccount.value = account
}

function openDetailById(id: number, name: string) {
  detailAccount.value = allAccounts.value.find((a) => a.id === id) ?? ({ id, name } as AccountListItem)
}

// 后台每分钟评估一次规则：定时刷新池归属与状态，并驱动倒计时
const nowTick = ref(Date.now())
let tickTimer: ReturnType<typeof setInterval> | null = null
let poolTimer: ReturnType<typeof setInterval> | null = null

async function refreshPoolView() {
  const groupId = selectedGroupId.value
  if (groupId == null || accountsLoading.value || moving.value) return
  try {
    const view = await schedulingCenterAPI.getPool(groupId)
    if (groupId !== selectedGroupId.value) return
    members.value = view.members
    accountStates.value = view.states
  } catch {
    // 静默失败，等待下一次刷新
  }
}

// ---------- 选择 & 移动 ----------
const allPageSelected = computed(
  () => pagedAccounts.value.length > 0 && pagedAccounts.value.every((a) => selectedIds.value.has(a.id))
)
const somePageSelected = computed(() => pagedAccounts.value.some((a) => selectedIds.value.has(a.id)))

function toggleOne(id: number) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}

function togglePage() {
  const next = new Set(selectedIds.value)
  if (allPageSelected.value) pagedAccounts.value.forEach((a) => next.delete(a.id))
  else pagedAccounts.value.forEach((a) => next.add(a.id))
  selectedIds.value = next
}

function moveSelected() {
  moveAccounts([...selectedIds.value], activePool.value === 'account' ? 'scheduling' : 'account')
}

async function moveAccounts(ids: number[], target: SchedulingCenterPool) {
  const groupId = selectedGroupId.value
  if (groupId == null || ids.length === 0) return
  moving.value = true
  try {
    const res = await schedulingCenterAPI.moveAccounts(groupId, ids, target)
    if (groupId !== selectedGroupId.value) return
    members.value = res.members
    accountStates.value = res.states
    const next = new Set(selectedIds.value)
    ids.forEach((id) => next.delete(id))
    selectedIds.value = next
    appStore.showSuccess(
      t(target === 'scheduling' ? 'admin.schedulingCenter.movedToScheduling' : 'admin.schedulingCenter.movedToAccount', {
        n: ids.length
      })
    )
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.moveFailed')))
  } finally {
    moving.value = false
  }
}

// ---------- 展示辅助 ----------
type Tone = 'ok' | 'warn' | 'danger' | 'muted' | 'info'

const toneClass: Record<Tone, string> = {
  ok: 'badge-success',
  warn: 'badge-warning',
  danger: 'badge-danger',
  muted: 'badge-gray',
  info: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
}

const toneDot: Record<Tone, string> = {
  ok: 'bg-emerald-500',
  warn: 'bg-amber-500',
  danger: 'bg-red-500',
  muted: 'bg-gray-400',
  info: 'bg-sky-500 animate-pulse'
}

interface CenterStatusView {
  key: string
  tone: Tone
  extra?: string
  title?: string
}

// 调度中心自己的状态，与账号管理中的调度状态无关
function centerStatus(id: number): CenterStatusView {
  void nowTick.value // 倒计时依赖
  const member = memberByAccount.value.get(id)
  if (member) {
    return {
      key: 'inPool',
      tone: 'ok',
      title: member.entered_at ? t('admin.schedulingCenter.centerStatusTitle.inPool', { time: formatDateTime(member.entered_at) }) : undefined
    }
  }
  const st = stateByAccount.value.get(id)
  if (!st) return { key: 'idle', tone: 'muted' }
  const reason = st.reason_code ? t(`admin.schedulingCenter.centerReason.${st.reason_code}`) : ''
  switch (st.state) {
    case 'cooling':
      return {
        key: 'cooling',
        tone: 'warn',
        extra: st.cooldown_until ? formatRemaining(st.cooldown_until) : undefined,
        title: [reason, st.cooldown_until ? formatDateTime(st.cooldown_until) : ''].filter(Boolean).join(' · ')
      }
    case 'probing':
      if (st.probe_status === 'passed') {
        const need = currentSettings.value?.recover_success_count || 3
        return {
          key: 'verifying',
          tone: 'info',
          extra: `${st.recover_ok_count}/${need}`,
          title: t('admin.schedulingCenter.centerStatusTitle.verifying', { n: need })
        }
      }
      return { key: 'probing', tone: 'info', title: t('admin.schedulingCenter.centerStatusTitle.probing') }
    case 'ready':
      return { key: 'ready', tone: 'muted', title: t('admin.schedulingCenter.centerStatusTitle.ready') }
    case 'manual_out':
      return { key: 'manualOut', tone: 'muted', title: t('admin.schedulingCenter.centerStatusTitle.manualOut') }
    default:
      return { key: 'idle', tone: 'muted' }
  }
}

function formatRemaining(until: string): string {
  const msLeft = new Date(until).getTime() - nowTick.value
  if (msLeft <= 0) return t('admin.schedulingCenter.centerStatus.soon')
  const min = Math.ceil(msLeft / 60000)
  return min < 60 ? `${min}m` : `${Math.floor(min / 60)}h${min % 60 ? `${min % 60}m` : ''}`
}

function metricsOf(id: number): SchedulingCenterAccountMetrics | undefined {
  return metrics.value[String(id)]
}

function iqView(id: number): { key: string; tone: Tone } {
  switch (metricsOf(id)?.iq_status) {
    case 'passed':
      return { key: 'passed', tone: 'ok' }
    case 'incorrect':
      return { key: 'degraded', tone: 'danger' }
    case 'error':
      return { key: 'error', tone: 'warn' }
    default:
      return { key: 'unchecked', tone: 'muted' }
  }
}

function iqTitle(id: number): string {
  const m = metricsOf(id)
  if (!m?.iq_checked_at) return t('admin.schedulingCenter.iqHint')
  return `${formatDateTime(m.iq_checked_at)}${m.iq_message ? ` · ${m.iq_message}` : ''}`
}

function firstTokenTitle(id: number): string {
  const m = metricsOf(id)
  if (!m) return ''
  const parts: string[] = []
  if (m.test_first_token_ms != null) parts.push(`${t('admin.schedulingCenter.firstTokenTest', { n: ttftSamples.value })}: ${formatMs(m.test_first_token_ms)}`)
  if (m.traffic_first_token_ms != null) parts.push(`${t('admin.schedulingCenter.firstTokenTraffic')}: ${formatMs(m.traffic_first_token_ms)}`)
  return parts.join('\n')
}

function formatMultiplier(value: number | undefined | null): string {
  const v = value ?? 1
  return `${Number(v.toFixed(4))}x`
}

function formatMs(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
}

function cacheRateClass(rate: number): string {
  if (rate >= 60) return 'text-emerald-600 dark:text-emerald-400'
  if (rate >= 30) return 'text-gray-700 dark:text-gray-300'
  return 'text-amber-600 dark:text-amber-400'
}

function cacheRateBar(rate: number): string {
  if (rate >= 60) return 'bg-emerald-500'
  if (rate >= 30) return 'bg-gray-400 dark:bg-dark-400'
  return 'bg-amber-500'
}

function ttftClass(ms: number): string {
  if (ms <= 2000) return 'text-emerald-600 dark:text-emerald-400'
  if (ms <= 5000) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

onMounted(() => {
  loadGroups()
  tickTimer = setInterval(() => (nowTick.value = Date.now()), 30_000)
  poolTimer = setInterval(() => {
    if (document.visibilityState !== 'visible') return
    if (activeTab.value === 'log') logPanel.value?.reload()
    else if (activeTab.value !== 'compare') refreshPoolView()
  }, 60_000)
})

onBeforeUnmount(() => {
  if (tickTimer) clearInterval(tickTimer)
  if (poolTimer) clearInterval(poolTimer)
})
</script>

<style scoped>
.meta-chip {
  @apply inline-flex max-w-full items-center gap-1.5 rounded-lg bg-gray-50 px-2.5 py-1 text-xs ring-1 ring-inset ring-gray-200/70;
  @apply dark:bg-dark-900/40 dark:ring-dark-700;
}

.meta-chip > span:last-child {
  @apply truncate;
}

.sc-table {
  @apply w-full min-w-max text-sm;
}

.sc-table thead th {
  @apply sticky top-0 z-10 whitespace-nowrap border-b border-gray-200 bg-gray-50/90 px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 backdrop-blur-sm;
  @apply dark:border-dark-700 dark:bg-dark-800/90 dark:text-dark-400;
}

.sc-table thead th.text-right {
  text-align: right;
}

.sc-table tbody td {
  @apply whitespace-nowrap border-b border-gray-100 px-4 py-3 text-gray-700 dark:border-dark-700/60 dark:text-gray-300;
}

.sc-table tbody tr {
  @apply transition-colors duration-150;
}

.sc-table tbody tr:not(.no-hover):hover {
  @apply bg-gray-50/80 dark:bg-dark-700/30;
}

.sc-table tbody tr.is-selected {
  @apply bg-primary-50/60 dark:bg-primary-900/10;
}

.sc-checkbox {
  @apply h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800;
}

.row-action {
  @apply inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium text-gray-500 transition-colors;
  @apply hover:bg-gray-100 hover:text-gray-800 disabled:cursor-not-allowed disabled:opacity-50;
  @apply dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-gray-100;
}

.row-action-primary {
  @apply text-primary-600 hover:bg-primary-50 hover:text-primary-700 dark:text-primary-400 dark:hover:bg-primary-900/20 dark:hover:text-primary-300;
}

.row-action-warn {
  @apply text-amber-600 hover:bg-amber-50 hover:text-amber-700 dark:text-amber-400 dark:hover:bg-amber-900/20 dark:hover:text-amber-300;
}
</style>
