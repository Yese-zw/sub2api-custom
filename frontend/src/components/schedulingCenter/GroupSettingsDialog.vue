<template>
  <BaseDialog :show="show" :title="t('admin.schedulingCenter.settings.title', { name: groupName })" width="wide" @close="emit('close')">
    <div v-if="loading" class="flex justify-center py-10"><LoadingSpinner /></div>
    <div v-else class="space-y-4">
      <!-- 开启调度 -->
      <section class="settings-section">
        <div class="settings-section-head">
          <span class="settings-section-icon bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"><Icon name="cog" size="sm" /></span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.schedulingCenter.settings.enabled') }}</div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.enabledHint') }}</p>
          </div>
          <Toggle v-model="form.enabled" />
        </div>

        <div class="settings-section-body" :class="{ 'pointer-events-none opacity-50': !form.enabled }">
          <label class="input-label">
            {{ t('admin.schedulingCenter.settings.frequency') }}
          </label>
          <div class="flex flex-col gap-2 sm:flex-row">
            <Select v-model="cronPreset" :options="cronPresetOptions" class="sm:w-48" />
            <input
              v-if="cronPreset === CUSTOM"
              v-model="customCron"
              type="text"
              class="input flex-1 font-mono text-sm"
              placeholder="*/30 * * * *"
            />
          </div>
          <p class="input-hint">{{ t('admin.schedulingCenter.settings.cronHint') }}</p>

          <label class="input-label mt-4">
            {{ t('admin.schedulingCenter.settings.model') }}
          </label>
          <input
            v-model="modelId"
            type="text"
            list="scheduling-center-model-candidates"
            class="input font-mono text-sm"
            :placeholder="t('admin.schedulingCenter.settings.modelPlaceholder')"
          />
          <datalist id="scheduling-center-model-candidates">
            <option v-for="m in modelCandidates" :key="m" :value="m" />
          </datalist>
          <p class="input-hint">{{ t('admin.schedulingCenter.settings.modelHint') }}</p>

          <label class="mt-4 flex flex-wrap items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <span>{{ t('admin.schedulingCenter.settings.ttftSamples1') }}</span>
            <input v-model="params.firstTokenSamples" type="number" min="1" max="1000" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.ttftSamples2') }}</span>
          </label>
          <label class="mt-3 flex flex-wrap items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <span>{{ t('admin.schedulingCenter.settings.retention1') }}</span>
            <input v-model="params.recordRetention" type="number" min="10" max="1000" class="input input-inline w-24" />
            <span>{{ t('admin.schedulingCenter.settings.retention2') }}</span>
          </label>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingCenter.settings.retentionHint') }}</p>
          <label class="mt-3 flex flex-wrap items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <span>{{ t('admin.schedulingCenter.settings.eventRetention1') }}</span>
            <input v-model="params.eventRetentionDays" type="number" min="8" max="365" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.eventRetention2') }}</span>
          </label>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.schedulingCenter.settings.eventRetentionHint') }}</p>
          <p
            v-if="costEstimate"
            class="mt-3 rounded-lg bg-sky-50 px-3 py-2 text-xs text-sky-700 dark:bg-sky-900/20 dark:text-sky-300"
          >
            {{ t('admin.schedulingCenter.settings.costEstimate', costEstimate) }}
          </p>
          <p
            v-if="planOwners && planOwners.length > 0"
            class="mt-3 rounded-lg bg-violet-50 px-3 py-2 text-xs text-violet-700 dark:bg-violet-900/20 dark:text-violet-300"
          >
            {{ t('admin.schedulingCenter.settings.sharedPlansHint', { n: planOwners.length, groups: planOwnerSummary }) }}
          </p>
        </div>
      </section>

      <!-- 调度池生效（灰度） -->
      <section class="settings-section" :class="{ 'border-red-300 dark:border-red-800': form.enabled && form.enforce_enabled }">
        <div class="settings-section-head">
          <span class="settings-section-icon bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-400"><Icon name="bolt" size="sm" /></span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.schedulingCenter.settings.enforce') }}</div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.enforceHint') }}</p>
          </div>
          <Toggle v-model="form.enforce_enabled" />
        </div>
        <div v-if="form.enforce_enabled" class="settings-section-body space-y-2 text-xs">
          <p class="rounded-lg bg-red-50 px-3 py-2 text-red-700 dark:bg-red-900/20 dark:text-red-300">
            {{ t('admin.schedulingCenter.settings.enforceWarning') }}
          </p>
          <ul class="list-disc space-y-1 pl-5 text-gray-600 dark:text-gray-300">
            <li>{{ t('admin.schedulingCenter.settings.enforceFallback1') }}</li>
            <li>{{ t('admin.schedulingCenter.settings.enforceFallback2') }}</li>
            <li>{{ t('admin.schedulingCenter.settings.enforceFallback3') }}</li>
          </ul>
          <p v-if="!form.enabled" class="text-amber-600 dark:text-amber-400">{{ t('admin.schedulingCenter.settings.enforceNeedsEnabled') }}</p>
          <p v-else-if="!fallbackForm.enabled" class="text-amber-600 dark:text-amber-400">{{ t('admin.schedulingCenter.settings.enforceNoFallback') }}</p>
        </div>
      </section>

      <!-- 入池规则 -->
      <section class="settings-section">
        <div class="settings-section-head">
          <span class="settings-section-icon bg-emerald-50 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400"><Icon name="arrowRight" size="sm" /></span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.schedulingCenter.settings.ruleEnabled') }}</div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.ruleHint') }}</p>
          </div>
          <Toggle v-model="form.rule_enabled" />
        </div>

        <div class="settings-section-body space-y-3" :class="{ 'pointer-events-none opacity-50': !form.rule_enabled }">
          <div class="flex items-center gap-3">
            <span class="w-24 flex-shrink-0 text-sm text-gray-700 dark:text-gray-300">{{ t('admin.schedulingCenter.columns.cacheRate') }} ≥</span>
            <input v-model="minCacheRate" type="number" min="0" max="100" step="1" class="input input-inline w-28" :placeholder="t('admin.schedulingCenter.settings.noLimit')" />
            <span class="text-sm text-gray-500">%</span>
            <span class="text-sm text-gray-500">{{ t('admin.schedulingCenter.settings.entryMinReq1') }}</span>
            <input v-model="params.entryCacheMinRequests" type="number" min="1" class="input input-inline w-20" />
            <span class="text-sm text-gray-500">{{ t('admin.schedulingCenter.settings.entryMinReq2') }}</span>
          </div>
          <div class="flex items-center gap-3">
            <span class="w-24 flex-shrink-0 text-sm text-gray-700 dark:text-gray-300">{{ t('admin.schedulingCenter.columns.firstToken') }} ≤</span>
            <input v-model="maxFirstTokenMs" type="number" min="1" step="100" class="input input-inline w-28" :placeholder="t('admin.schedulingCenter.settings.noLimit')" />
            <span class="text-sm text-gray-500">ms</span>
          </div>
          <label class="flex items-center gap-3">
            <span class="w-24 flex-shrink-0 text-sm text-gray-700 dark:text-gray-300">{{ t('admin.schedulingCenter.columns.iq') }}</span>
            <input v-model="form.require_iq_pass" type="checkbox" class="sc-checkbox" />
            <span class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.entryIQ1') }}</span>
            <input v-model="params.entryIQPassCount" type="number" min="1" max="1000" class="input input-inline w-20" />
            <span class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.entryIQ2') }}</span>
          </label>
          <p v-if="form.rule_enabled && !hasCondition" class="text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.schedulingCenter.settings.needCondition') }}
          </p>
          <p v-if="form.rule_enabled && !form.enabled" class="text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.schedulingCenter.settings.ruleNeedsEnabled') }}
          </p>
        </div>

        <!-- 手动移出有效期：同时影响入池规则与兜底，不随入池规则开关禁用 -->
        <div class="settings-section-body space-y-1 border-t border-gray-100 pt-3 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-300">
          <label class="flex flex-wrap items-center gap-2">
            <span>{{ t('admin.schedulingCenter.settings.manualOut1') }}</span>
            <input v-model="params.manualOutHours" type="number" min="0" max="720" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.manualOut2') }}</span>
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.manualOutHint') }}</p>
        </div>
      </section>

      <!-- 出池规则 -->
      <section class="settings-section">
        <div class="settings-section-head">
          <span class="settings-section-icon bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400"><Icon name="arrowLeft" size="sm" /></span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.schedulingCenter.settings.exitEnabled') }}</div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.exitHint') }}</p>
          </div>
          <Toggle v-model="exitForm.enabled" />
        </div>

        <div class="settings-section-body space-y-3 text-sm text-gray-700 dark:text-gray-300" :class="{ 'pointer-events-none opacity-50': !exitForm.enabled }">
          <div class="sub-heading">{{ t('admin.schedulingCenter.settings.exitGroupTest') }}</div>
          <label class="flex flex-wrap items-center gap-2">
            <input v-model="exitForm.cacheOn" type="checkbox" class="sc-checkbox" />
            <span>{{ t('admin.schedulingCenter.settings.exitCache1') }}</span>
            <input v-model="exitForm.cacheDrop" type="number" min="0" max="100" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitCache2') }}</span>
            <input v-model="exitForm.cacheMinRequests" type="number" min="1" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitCache3') }}</span>
          </label>
          <p v-if="exitForm.cacheOn && parseOptionalNumber(minCacheRate) == null" class="pl-6 text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.schedulingCenter.settings.exitCacheNeedsEntry') }}
          </p>
          <label class="flex flex-wrap items-center gap-2">
            <input v-model="exitForm.iqOn" type="checkbox" class="sc-checkbox" />
            <span>{{ t('admin.schedulingCenter.settings.exitIQ1') }}</span>
            <input v-model="exitForm.iqWindow" type="number" min="1" max="1000" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitIQ2') }}</span>
            <input v-model="exitForm.iqMax" type="number" min="0" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitIQ3') }}</span>
          </label>

          <div class="sub-heading pt-2">{{ t('admin.schedulingCenter.settings.exitGroupTraffic') }}</div>
          <label class="flex flex-wrap items-center gap-2">
            <span>{{ t('admin.schedulingCenter.settings.trafficWindow1') }}</span>
            <input v-model="params.exitTrafficWindowHours" type="number" min="1" max="168" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.trafficWindow2') }}</span>
          </label>
          <label class="flex flex-wrap items-center gap-2">
            <input v-model="exitForm.latencyOn" type="checkbox" class="sc-checkbox" />
            <span>{{ t('admin.schedulingCenter.settings.exitLatency1') }}</span>
            <input v-model="exitForm.latencyWindow" type="number" min="1" max="100" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitLatency2') }}</span>
            <input v-model="exitForm.latencyCount" type="number" min="1" max="100" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitLatency3') }}</span>
            <input v-model="exitForm.latencyMs" type="number" min="1" step="500" class="input input-inline w-24" />
            <span>ms</span>
          </label>
          <label class="flex flex-wrap items-center gap-2">
            <input v-model="exitForm.errorOn" type="checkbox" class="sc-checkbox" />
            <span>{{ t('admin.schedulingCenter.settings.exitError1') }}</span>
            <input v-model="exitForm.errorWindow" type="number" min="1" max="100" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitError2') }}</span>
            <input v-model="exitForm.errorCount" type="number" min="1" max="100" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.exitError3') }}</span>
          </label>
          <p class="pl-6 text-xs text-gray-400">{{ t('admin.schedulingCenter.settings.exitErrorHint') }}</p>
          <p v-if="opsMonitoringEnabled === false" class="pl-6 text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.schedulingCenter.settings.exitErrorOpsDisabled') }}
          </p>

          <div class="sub-heading pt-2">{{ t('admin.schedulingCenter.settings.recoverGroup') }}</div>
          <label class="flex flex-wrap items-center gap-2">
            <span>{{ t('admin.schedulingCenter.settings.badWindow1') }}</span>
            <input v-model="params.badWindowMinutes" type="number" min="1" max="10080" class="input input-inline w-24" />
            <span>{{ t('admin.schedulingCenter.settings.badWindow2') }}</span>
            <input v-model="params.cooldownSteps" type="text" class="input input-inline w-40 font-mono" placeholder="5, 15, 30, 60" />
            <span>{{ t('admin.schedulingCenter.settings.badWindow3') }}</span>
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.backoffHint', { preview: cooldownPreview }) }}</p>
          <label class="flex flex-wrap items-center gap-2">
            <span>{{ t('admin.schedulingCenter.settings.recoverLatency1') }}</span>
            <input v-model="params.recoverLatencyMs" type="number" min="1" step="500" class="input input-inline w-24" :placeholder="t('admin.schedulingCenter.settings.noLimit')" />
            <span>{{ t('admin.schedulingCenter.settings.recoverLatency2') }}</span>
          </label>
          <label class="flex flex-wrap items-center gap-2">
            <span>{{ t('admin.schedulingCenter.settings.recover1') }}</span>
            <input v-model="exitForm.recoverCount" type="number" min="1" max="50" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.recover2') }}</span>
          </label>
          <p v-if="exitForm.enabled && !hasExitCondition" class="text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.schedulingCenter.settings.exitNeedCondition') }}
          </p>
          <p v-if="hysteresisWarning" class="text-xs text-amber-600 dark:text-amber-400">{{ hysteresisWarning }}</p>
        </div>
      </section>

      <!-- 兜底规则 -->
      <section class="settings-section">
        <div class="settings-section-head">
          <span class="settings-section-icon bg-sky-50 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400"><Icon name="shield" size="sm" /></span>
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.schedulingCenter.settings.fallbackEnabled') }}</div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.fallbackHint') }}</p>
          </div>
          <Toggle v-model="fallbackForm.enabled" />
        </div>

        <div class="settings-section-body space-y-3 text-sm text-gray-700 dark:text-gray-300" :class="{ 'pointer-events-none opacity-50': !fallbackForm.enabled }">
          <label class="flex flex-wrap items-center gap-2">
            <span>{{ t('admin.schedulingCenter.settings.fallbackMin1') }}</span>
            <input v-model="fallbackForm.minAccounts" type="number" min="1" max="50" class="input input-inline w-20" />
            <span>{{ t('admin.schedulingCenter.settings.fallbackMin2') }}</span>
          </label>

          <div class="sub-heading pt-1">{{ t('admin.schedulingCenter.settings.fallbackOrder') }}</div>
          <ol class="space-y-1.5">
            <li
              v-for="(key, i) in fallbackForm.order"
              :key="key"
              class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-1.5 dark:border-dark-600"
            >
              <span class="w-5 text-xs tabular-nums text-gray-400">{{ i + 1 }}</span>
              <span class="flex-1">{{ t(`admin.schedulingCenter.settings.fallbackKeys.${key}`) }}</span>
              <button
                type="button"
                class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-30 dark:hover:bg-dark-700"
                :disabled="i === 0"
                :title="t('admin.schedulingCenter.settings.moveUp')"
                @click="moveFallbackKey(i, -1)"
              >
                <Icon name="chevronUp" size="sm" />
              </button>
              <button
                type="button"
                class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700 disabled:opacity-30 dark:hover:bg-dark-700"
                :disabled="i === fallbackForm.order.length - 1"
                :title="t('admin.schedulingCenter.settings.moveDown')"
                @click="moveFallbackKey(i, 1)"
              >
                <Icon name="chevronDown" size="sm" />
              </button>
            </li>
          </ol>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.fallbackOrderHint') }}</p>

          <label class="flex items-center gap-2 pt-1">
            <input v-model="fallbackForm.requireIQPass" type="checkbox" class="sc-checkbox" />
            <span>{{ t('admin.schedulingCenter.settings.fallbackRequireIQ') }}</span>
          </label>
          <p class="pl-6 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.settings.fallbackRequireIQHint') }}</p>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loading || saving" @click="save">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import schedulingCenterAPI, {
  type SchedulingCenterGroupSettings,
  type SchedulingCenterPlanOwner
} from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  groupId: number | null
  groupName: string
  /** 本分组中检测计划归属于其他分组的账号 */
  planOwners?: SchedulingCenterPlanOwner[]
  /** 分组账号数，用于估算每天的检测请求量 */
  accountCount?: number
  /** 运维监控是否开启（未开启时错误出池规则无效） */
  opsMonitoringEnabled?: boolean
}>()

// 按归属分组汇总，提示这些账号的检测频率 / 模型 / 保留条数不按本分组设置执行
const planOwnerSummary = computed(() => {
  const byGroup = new Map<string, number>()
  for (const o of props.planOwners ?? []) {
    const name = o.group_name || `#${o.group_id}`
    byGroup.set(name, (byGroup.get(name) ?? 0) + 1)
  }
  return [...byGroup.entries()].map(([name, n]) => `「${name}」${n}`).join('、')
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', settings: SchedulingCenterGroupSettings): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const CUSTOM = 'custom'
const PRESET_KEYS: Record<string, string> = {
  '*/5 * * * *': 'every5m',
  '*/10 * * * *': 'every10m',
  '*/15 * * * *': 'every15m',
  '*/30 * * * *': 'every30m',
  '0 * * * *': 'hourly',
  '0 */6 * * *': 'every6h'
}
const PRESETS = Object.keys(PRESET_KEYS)

const loading = ref(false)
const saving = ref(false)
const form = reactive({ enabled: false, enforce_enabled: false, rule_enabled: false, require_iq_pass: false })
const cronPreset = ref<string>('*/30 * * * *')
const customCron = ref('')
const modelId = ref('')
const modelCandidates = ref<string[]>([])
// number input 的 v-model 可能给 number 或 ''，统一按字符串处理
const minCacheRate = ref<string | number>('')
const maxFirstTokenMs = ref<string | number>('')

type NumInput = string | number
const DEFAULT_FALLBACK_ORDER = ['rate_multiplier', 'first_token', 'cache_rate']
const fallbackForm = reactive({
  enabled: true,
  minAccounts: 1 as string | number,
  order: [...DEFAULT_FALLBACK_ORDER],
  requireIQPass: true
})

function moveFallbackKey(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= fallbackForm.order.length) return
  const next = [...fallbackForm.order]
  ;[next[index], next[target]] = [next[target], next[index]]
  fallbackForm.order = next
}

const exitForm = reactive({
  enabled: false,
  cacheOn: true,
  cacheDrop: 20 as NumInput,
  cacheMinRequests: 20 as NumInput,
  iqOn: true,
  iqWindow: 10 as NumInput,
  iqMax: 3 as NumInput,
  latencyOn: true,
  latencyCount: 3 as NumInput,
  latencyWindow: 3 as NumInput,
  errorWindow: 3 as NumInput,
  latencyMs: 10000 as NumInput,
  errorOn: true,
  errorCount: 3 as NumInput,
  recoverCount: 3 as NumInput
})

// 其余规则参数
const params = reactive({
  firstTokenSamples: 10 as NumInput,
  recordRetention: 100 as NumInput,
  entryCacheMinRequests: 1 as NumInput,
  entryIQPassCount: 1 as NumInput,
  exitTrafficWindowHours: 24 as NumInput,
  badWindowMinutes: 180 as NumInput,
  cooldownSteps: '5, 15, 30, 60',
  recoverLatencyMs: 10000 as NumInput,
  manualOutHours: 24 as NumInput,
  eventRetentionDays: 30 as NumInput
})

// 估算 5 段 cron 每天执行的次数；无法解析时返回 null
function cronFieldCount(field: string, lo: number, hi: number): number | null {
  const values = new Set<number>()
  for (const part of field.split(',')) {
    const [base, stepText] = part.split('/')
    const step = stepText == null ? 1 : Number(stepText)
    if (!Number.isInteger(step) || step <= 0) return null
    let from: number
    let to: number
    if (base === '*') {
      from = lo
      to = hi
    } else if (base.includes('-')) {
      const [a, b] = base.split('-').map(Number)
      from = a
      to = b
    } else {
      from = Number(base)
      to = stepText == null ? from : hi
    }
    if (!Number.isInteger(from) || !Number.isInteger(to) || from < lo || to > hi || from > to) return null
    for (let v = from; v <= to; v += step) values.add(v)
  }
  return values.size
}

function estimateCronRunsPerDay(expr: string): number | null {
  const fields = expr.trim().split(/\s+/)
  if (fields.length !== 5) return null
  const minutes = cronFieldCount(fields[0], 0, 59)
  const hours = cronFieldCount(fields[1], 0, 23)
  if (minutes == null || hours == null) return null
  let runs = minutes * hours
  // 限定日期 / 星期时按比例折算成日均
  if (fields[2] !== '*') {
    const dom = cronFieldCount(fields[2], 1, 31)
    if (dom == null) return null
    runs *= dom / 30
  }
  if (fields[4] !== '*') {
    const dow = cronFieldCount(fields[4].replace(/7/g, '0'), 0, 6)
    if (dow == null) return null
    runs *= dow / 7
  }
  return runs
}

// 每次定时检测 = 1 次智商检测 + 1 次测速（常规请求 hi），都会消耗真实上游额度
const costEstimate = computed(() => {
  if (!form.enabled) return null
  const cron = cronPreset.value === CUSTOM ? customCron.value.trim() : cronPreset.value
  const runs = estimateCronRunsPerDay(cron)
  if (runs == null) return null
  const accounts = Math.max(0, (props.accountCount ?? 0) - (props.planOwners?.length ?? 0))
  if (accounts === 0) return null
  const checks = Math.round(accounts * runs)
  return {
    accounts,
    runs: Math.round(runs * 10) / 10,
    checks: checks.toLocaleString(),
    requests: (checks * 2).toLocaleString()
  }
})

function parseCooldownSteps(text: string): number[] | null {
  const parts = text
    .split(/[,，\s]+/)
    .map((p) => p.trim())
    .filter(Boolean)
  if (parts.length === 0 || parts.length > 10) return null
  const out: number[] = []
  for (const p of parts) {
    const n = Number(p)
    if (!Number.isInteger(n) || n <= 0 || n > 1440) return null
    out.push(n)
  }
  return out
}

const cooldownPreview = computed(() => {
  const steps = parseCooldownSteps(params.cooldownSteps)
  if (!steps) return '-'
  return steps
    .map((m, i) => (i === steps.length - 1 ? t('admin.schedulingCenter.settings.cooldownCap', { n: i + 1, m }) : t('admin.schedulingCenter.settings.cooldownStep', { n: i + 1, m })))
    .join('，')
})

const hasExitCondition = computed(
  () =>
    (exitForm.cacheOn && parseOptionalNumber(minCacheRate.value) != null) ||
    exitForm.iqOn ||
    exitForm.latencyOn ||
    exitForm.errorOn
)

// 出池门槛不比入池门槛宽时，账号可能反复进出
const hysteresisWarning = computed(() => {
  if (!exitForm.enabled) return ''
  const entryTTFT = parseOptionalNumber(maxFirstTokenMs.value)
  const exitTTFT = parseOptionalNumber(exitForm.latencyMs)
  if (exitForm.latencyOn && entryTTFT != null && exitTTFT != null && exitTTFT <= entryTTFT) {
    return t('admin.schedulingCenter.settings.hysteresisTTFT')
  }
  const drop = parseOptionalNumber(exitForm.cacheDrop)
  if (exitForm.cacheOn && drop != null && drop <= 0 && parseOptionalNumber(minCacheRate.value) != null) {
    return t('admin.schedulingCenter.settings.hysteresisCache')
  }
  return ''
})

const cronPresetOptions = computed(() => [
  ...PRESETS.map((value) => ({ value, label: t(`admin.schedulingCenter.settings.presets.${PRESET_KEYS[value]}`) })),
  { value: CUSTOM, label: t('admin.schedulingCenter.settings.presets.custom') }
])

function parseOptionalNumber(value: string | number): number | null {
  if (value === '' || value == null) return null
  const n = Number(value)
  return Number.isFinite(n) ? n : null
}

const hasCondition = computed(
  () => form.require_iq_pass || parseOptionalNumber(minCacheRate.value) != null || parseOptionalNumber(maxFirstTokenMs.value) != null
)

function applySettings(s: SchedulingCenterGroupSettings) {
  form.enabled = s.enabled
  form.enforce_enabled = s.enforce_enabled ?? false
  form.rule_enabled = s.rule_enabled
  form.require_iq_pass = s.require_iq_pass
  if (PRESETS.includes(s.cron_expression)) {
    cronPreset.value = s.cron_expression
    customCron.value = ''
  } else {
    cronPreset.value = CUSTOM
    customCron.value = s.cron_expression
  }
  modelId.value = s.model_id ?? ''
  minCacheRate.value = s.min_cache_rate ?? ''
  exitForm.enabled = s.exit_enabled
  exitForm.cacheOn = s.exit_cache_drop_points != null
  exitForm.cacheDrop = s.exit_cache_drop_points ?? 20
  exitForm.cacheMinRequests = s.exit_cache_min_requests || 20
  exitForm.iqOn = s.exit_iq_max_degraded != null
  exitForm.iqWindow = s.exit_iq_window || 10
  exitForm.iqMax = s.exit_iq_max_degraded ?? 3
  exitForm.latencyOn = s.exit_latency_ms != null
  exitForm.latencyCount = s.exit_latency_count || 3
  exitForm.latencyWindow = s.exit_latency_window || s.exit_latency_count || 3
  exitForm.errorWindow = s.exit_error_window || s.exit_error_count || 3
  exitForm.latencyMs = s.exit_latency_ms ?? 10000
  exitForm.errorOn = s.exit_error_count != null
  exitForm.errorCount = s.exit_error_count ?? 3
  exitForm.recoverCount = s.recover_success_count || 3
  params.firstTokenSamples = s.first_token_test_samples || 10
  params.recordRetention = s.record_retention || 100
  params.entryCacheMinRequests = s.entry_cache_min_requests || 1
  params.entryIQPassCount = s.entry_iq_pass_count || 1
  params.exitTrafficWindowHours = s.exit_traffic_window_hours || 24
  params.badWindowMinutes = s.bad_window_minutes || 180
  params.cooldownSteps = (s.cooldown_minutes?.length ? s.cooldown_minutes : [5, 15, 30, 60]).join(', ')
  params.recoverLatencyMs = s.recover_latency_ms ?? ''
  fallbackForm.enabled = s.fallback_enabled ?? true
  fallbackForm.minAccounts = s.fallback_min_accounts || 1
  // 补齐缺失的排序项，保持已保存的先后顺序
  const savedOrder = (s.fallback_order ?? []).filter((k) => DEFAULT_FALLBACK_ORDER.includes(k))
  fallbackForm.order = [...savedOrder, ...DEFAULT_FALLBACK_ORDER.filter((k) => !savedOrder.includes(k))]
  fallbackForm.requireIQPass = s.fallback_require_iq_pass ?? true
  params.manualOutHours = s.manual_out_hours ?? 24
  params.eventRetentionDays = s.event_retention_days || 30
  maxFirstTokenMs.value = s.max_first_token_ms ?? ''
}

watch(
  () => [props.show, props.groupId] as const,
  async ([show, groupId]) => {
    if (!show || groupId == null) return
    loading.value = true
    modelCandidates.value = []
    // 候选模型只是输入提示，加载失败不影响设置
    adminAPI.groups
      .getModelAllowlistCandidates(groupId)
      .then((models) => {
        if (groupId === props.groupId) modelCandidates.value = models
      })
      .catch(() => {})
    try {
      applySettings(await schedulingCenterAPI.getSettings(groupId))
    } catch (error) {
      appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.settings.loadFailed')))
      emit('close')
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

function positiveInt(value: NumInput, fallback: number): number | null {
  const n = parseOptionalNumber(value)
  if (n == null) return fallback
  return Number.isInteger(n) && n > 0 ? n : null
}

// 校验并组装出池规则；校验失败时提示并返回 null
function buildExitPayload() {
  const fail = (key: string) => {
    appStore.showError(t(`admin.schedulingCenter.settings.${key}`))
    return null
  }
  const cacheMinRequests = positiveInt(exitForm.cacheMinRequests, 20)
  const iqWindow = positiveInt(exitForm.iqWindow, 10)
  const latencyCount = positiveInt(exitForm.latencyCount, 3)
  const recoverCount = positiveInt(exitForm.recoverCount, 3)
  const errorCount = positiveInt(exitForm.errorCount, 3)
  if (!cacheMinRequests || !iqWindow || !latencyCount || !recoverCount || !errorCount) return fail('exitCountInvalid')
  const recordRetention = positiveInt(params.recordRetention, 100)
  if (!recordRetention || recordRetention < 10 || recordRetention > 1000) return fail('retentionInvalid')
  if (recoverCount > 50) return fail('exitCountInvalid')
  const latencyWindow = positiveInt(exitForm.latencyWindow, 3)
  const errorWindow = positiveInt(exitForm.errorWindow, 3)
  if (!latencyWindow || !errorWindow || latencyWindow > 100 || errorWindow > 100) return fail('exitWindowInvalid')
  if (exitForm.latencyOn && latencyCount > latencyWindow) return fail('exitLatencyCountTooLarge')
  if (exitForm.errorOn && errorCount > errorWindow) return fail('exitErrorCountTooLarge')
  const firstTokenSamples = positiveInt(params.firstTokenSamples, 10)
  const entryCacheMinRequests = positiveInt(params.entryCacheMinRequests, 1)
  const entryIQPassCount = positiveInt(params.entryIQPassCount, 1)
  if (!firstTokenSamples || !entryIQPassCount || !entryCacheMinRequests) return fail('paramCountInvalid')
  // 依赖历史检测记录的条数不能超过保留条数
  if (iqWindow > recordRetention || firstTokenSamples > recordRetention || entryIQPassCount > recordRetention) {
    return fail('retentionTooSmall')
  }
  const exitTrafficWindowHours = positiveInt(params.exitTrafficWindowHours, 24)
  if (!exitTrafficWindowHours || exitTrafficWindowHours > 168) return fail('trafficWindowInvalid')
  const badWindowMinutes = positiveInt(params.badWindowMinutes, 180)
  if (!badWindowMinutes || badWindowMinutes > 10080) return fail('badWindowInvalid')
  const cooldownMinutes = parseCooldownSteps(params.cooldownSteps)
  if (!cooldownMinutes) return fail('cooldownInvalid')
  const manualOutRaw = parseOptionalNumber(params.manualOutHours)
  const manualOutHours = manualOutRaw == null ? 24 : manualOutRaw
  if (!Number.isInteger(manualOutHours) || manualOutHours < 0 || manualOutHours > 720) return fail('manualOutInvalid')
  const eventRetentionDays = positiveInt(params.eventRetentionDays, 30)
  if (!eventRetentionDays || eventRetentionDays < 8 || eventRetentionDays > 365) return fail('eventRetentionInvalid')
  const fallbackMin = positiveInt(fallbackForm.minAccounts, 1)
  if (!fallbackMin || fallbackMin > 50) return fail('fallbackMinInvalid')
  const recoverLatencyMs = parseOptionalNumber(params.recoverLatencyMs)
  if (recoverLatencyMs != null && recoverLatencyMs <= 0) return fail('firstTokenRange')
  const drop = parseOptionalNumber(exitForm.cacheDrop)
  if (exitForm.cacheOn && (drop == null || drop < 0 || drop > 100)) return fail('exitCacheDropInvalid')
  const iqMax = parseOptionalNumber(exitForm.iqMax)
  if (exitForm.iqOn && (iqMax == null || !Number.isInteger(iqMax) || iqMax < 0 || iqMax >= iqWindow)) return fail('exitIQInvalid')
  const latencyMs = parseOptionalNumber(exitForm.latencyMs)
  if (exitForm.latencyOn && (latencyMs == null || latencyMs <= 0)) return fail('firstTokenRange')
  return {
    exit_enabled: exitForm.enabled,
    exit_cache_drop_points: exitForm.cacheOn ? drop : null,
    exit_cache_min_requests: cacheMinRequests,
    exit_iq_window: iqWindow,
    exit_iq_max_degraded: exitForm.iqOn ? iqMax : null,
    exit_latency_ms: exitForm.latencyOn && latencyMs != null ? Math.round(latencyMs) : null,
    exit_latency_count: Math.min(latencyCount, latencyWindow),
    exit_latency_window: latencyWindow,
    exit_error_window: errorWindow,
    exit_error_count: exitForm.errorOn ? errorCount : null,
    recover_success_count: recoverCount,
    first_token_test_samples: firstTokenSamples,
    record_retention: recordRetention,
    entry_cache_min_requests: entryCacheMinRequests,
    entry_iq_pass_count: entryIQPassCount,
    exit_traffic_window_hours: exitTrafficWindowHours,
    bad_window_minutes: badWindowMinutes,
    cooldown_minutes: cooldownMinutes,
    recover_latency_ms: recoverLatencyMs != null ? Math.round(recoverLatencyMs) : null,
    fallback_enabled: fallbackForm.enabled,
    fallback_min_accounts: fallbackMin,
    fallback_order: [...fallbackForm.order],
    fallback_require_iq_pass: fallbackForm.requireIQPass,
    manual_out_hours: manualOutHours,
    event_retention_days: eventRetentionDays
  }
}

async function save() {
  if (props.groupId == null) return
  const cron = cronPreset.value === CUSTOM ? customCron.value.trim() : cronPreset.value
  if (!cron) {
    appStore.showError(t('admin.schedulingCenter.settings.cronRequired'))
    return
  }
  const cacheRate = parseOptionalNumber(minCacheRate.value)
  if (cacheRate != null && (cacheRate < 0 || cacheRate > 100)) {
    appStore.showError(t('admin.schedulingCenter.settings.cacheRateRange'))
    return
  }
  const ttft = parseOptionalNumber(maxFirstTokenMs.value)
  if (ttft != null && ttft <= 0) {
    appStore.showError(t('admin.schedulingCenter.settings.firstTokenRange'))
    return
  }
  const exitPayload = buildExitPayload()
  if (!exitPayload) return
  saving.value = true
  try {
    const saved = await schedulingCenterAPI.updateSettings(props.groupId, {
      enabled: form.enabled,
      enforce_enabled: form.enforce_enabled,
      cron_expression: cron,
      model_id: modelId.value.trim(),
      rule_enabled: form.rule_enabled,
      min_cache_rate: cacheRate,
      require_iq_pass: form.require_iq_pass,
      max_first_token_ms: ttft != null ? Math.round(ttft) : null,
      ...exitPayload
    })
    appStore.showSuccess(t('admin.schedulingCenter.settings.saved'))
    emit('saved', saved)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.settings.saveFailed')))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.settings-section {
  @apply overflow-hidden rounded-2xl border border-gray-200 dark:border-dark-700;
}

.settings-section-head {
  @apply flex items-start gap-3 bg-white px-4 py-3.5 dark:bg-dark-800;
}

.settings-section-icon {
  @apply flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl;
}

.settings-section-body {
  @apply border-t border-gray-100 bg-gray-50/60 px-4 py-4 transition-opacity dark:border-dark-700 dark:bg-dark-900/30;
}

.sub-heading {
  @apply flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-dark-400;
}

.sub-heading::after {
  content: '';
  @apply h-px flex-1 bg-gray-200 dark:bg-dark-700;
}

.input-inline {
  @apply rounded-lg px-2.5 py-1.5 text-center tabular-nums;
}

.sc-checkbox {
  @apply h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800;
}
</style>
