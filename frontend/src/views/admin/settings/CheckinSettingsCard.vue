<template>
  <div class="space-y-6">
    <!-- 总开关 -->
    <div class="card">
      <div class="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-start gap-4">
          <span class="ck-admin-badge" aria-hidden="true">
            <Icon name="calendar" size="md" />
          </span>
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.checkin.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.checkin.description') }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span
            class="rounded-full px-2.5 py-0.5 text-xs font-medium"
            :class="form.enabled
              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
              : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'"
          >
            {{ form.enabled ? t('admin.settings.checkin.statusOn') : t('admin.settings.checkin.statusOff') }}
          </span>
          <Toggle v-model="form.enabled" :disabled="loading" />
        </div>
      </div>
    </div>

    <div v-if="loading" class="card p-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>

    <template v-else>
      <!-- 奖励区间 -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.settings.checkin.rewardTitle') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.checkin.rewardDesc') }}
          </p>
        </div>
        <div class="grid gap-6 p-6 md:grid-cols-2">
          <div>
            <label for="checkin-reward-min" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.checkin.rewardMin') }}
            </label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-gray-400">$</span>
              <input id="checkin-reward-min" v-model.number="form.reward_min" type="number" min="0" step="0.01" class="input pl-7" />
            </div>
          </div>
          <div>
            <label for="checkin-reward-max" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.checkin.rewardMax') }}
            </label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-gray-400">$</span>
              <input id="checkin-reward-max" v-model.number="form.reward_max" type="number" min="0" step="0.01" class="input pl-7" />
            </div>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400 md:col-span-2">
            {{ rewardModeHint }}
          </p>
        </div>
      </div>

      <!-- 门槛 -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.settings.checkin.thresholdTitle') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.checkin.thresholdDesc') }}
          </p>
        </div>
        <div class="grid gap-6 p-6 md:grid-cols-2">
          <div>
            <label for="checkin-min-recharge" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.checkin.minTotalRecharged') }}
            </label>
            <div class="relative">
              <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-gray-400">$</span>
              <input id="checkin-min-recharge" v-model.number="form.min_total_recharged" type="number" min="0" step="0.01" class="input pl-7" />
            </div>
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.checkin.minTotalRechargedHint') }}
            </p>
          </div>
          <div>
            <label for="checkin-min-age" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.checkin.minAccountAgeDays') }}
            </label>
            <input id="checkin-min-age" v-model.number="form.min_account_age_days" type="number" min="0" step="1" class="input" />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.checkin.minAccountAgeDaysHint') }}
            </p>
          </div>
        </div>
      </div>

      <!-- 连续签到奖励 -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.settings.checkin.streakTitle') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.checkin.streakDesc') }}
          </p>
        </div>
        <div class="space-y-6 p-6">
          <div class="max-w-xs">
            <label for="checkin-cycle" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.checkin.cycleDays') }}
            </label>
            <input id="checkin-cycle" v-model.number="form.streak_cycle_days" type="number" min="0" max="365" step="1" class="input" />
            <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.checkin.cycleDaysHint') }}
            </p>
          </div>

          <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.settings.checkin.colDay') }}</th>
                  <th class="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">{{ t('admin.settings.checkin.colBonus') }}</th>
                  <th class="w-16 px-4 py-2"></th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="(rule, index) in form.streak_rewards" :key="index">
                  <td class="px-4 py-2">
                    <div class="flex items-center gap-2">
                      <span class="text-gray-500 dark:text-gray-400">{{ t('admin.settings.checkin.dayPrefix') }}</span>
                      <input
                        v-model.number="rule.day"
                        type="number"
                        min="1"
                        :max="form.streak_cycle_days || 365"
                        step="1"
                        class="input w-24"
                        :aria-label="t('admin.settings.checkin.colDay')"
                      />
                      <span class="text-gray-500 dark:text-gray-400">{{ t('admin.settings.checkin.daySuffix') }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-2">
                    <div class="relative w-36">
                      <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-gray-400">+$</span>
                      <input
                        v-model.number="rule.bonus"
                        type="number"
                        min="0.01"
                        step="0.01"
                        class="input pl-9"
                        :aria-label="t('admin.settings.checkin.colBonus')"
                      />
                    </div>
                  </td>
                  <td class="px-4 py-2 text-right">
                    <button
                      type="button"
                      class="rounded-md p-1.5 text-gray-400 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                      :aria-label="t('common.delete')"
                      @click="removeRule(index)"
                    >
                      <Icon name="trash" size="sm" />
                    </button>
                  </td>
                </tr>
                <tr v-if="form.streak_rewards.length === 0">
                  <td colspan="3" class="px-4 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.checkin.noRules') }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="form.streak_rewards.length >= 31" @click="addRule">
            + {{ t('admin.settings.checkin.addRule') }}
          </button>

          <!-- 预览 -->
          <div v-if="previewDays.length" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-800/60">
            <p class="mb-3 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.checkin.preview') }}
            </p>
            <div class="grid grid-cols-4 gap-2 sm:grid-cols-7">
              <div
                v-for="d in previewDays"
                :key="d.day"
                class="rounded-lg border px-2 py-2 text-center"
                :class="d.bonus > 0
                  ? 'border-amber-300 bg-amber-50 dark:border-amber-700/60 dark:bg-amber-900/20'
                  : 'border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-700'"
              >
                <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.settings.checkin.dayN', { n: d.day }) }}</div>
                <div class="mt-0.5 text-xs font-semibold tabular-nums text-gray-900 dark:text-white">{{ d.label }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="flex justify-end">
        <button type="button" class="btn btn-primary" :disabled="saving" @click="save">
          {{ saving ? t('admin.settings.saving') : t('admin.settings.checkin.save') }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import type { CheckinSettings } from '@/api/admin/settings'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const saving = ref(false)
const form = reactive<CheckinSettings>({
  enabled: false,
  reward_min: 0.1,
  reward_max: 0.5,
  min_total_recharged: 0,
  min_account_age_days: 0,
  streak_cycle_days: 7,
  streak_rewards: [],
})

function money(v: number): string {
  return `$${(Number.isFinite(v) ? v : 0).toFixed(2)}`
}

const rewardModeHint = computed(() => {
  const min = Number(form.reward_min) || 0
  const max = Number(form.reward_max) || 0
  if (max < min) return t('admin.settings.checkin.rewardInvalid')
  if (min === max) return t('admin.settings.checkin.rewardFixed', { amount: money(min) })
  return t('admin.settings.checkin.rewardRandom', { min: money(min), max: money(max) })
})

const previewDays = computed(() => {
  const cycle = Number(form.streak_cycle_days) || 0
  const length = cycle > 0 ? Math.min(cycle, 14) : 7
  const base = form.reward_min === form.reward_max
    ? money(form.reward_min)
    : `${money(form.reward_min)}~`
  return Array.from({ length }, (_, i) => {
    const day = i + 1
    const bonus = form.streak_rewards
      .filter((r) => Number(r.day) === day)
      .reduce((sum, r) => sum + (Number(r.bonus) || 0), 0)
    return { day, bonus, label: bonus > 0 ? `+${money(bonus)}` : base }
  })
})

function assign(data: CheckinSettings) {
  form.enabled = data.enabled
  form.reward_min = data.reward_min
  form.reward_max = data.reward_max
  form.min_total_recharged = data.min_total_recharged
  form.min_account_age_days = data.min_account_age_days
  form.streak_cycle_days = data.streak_cycle_days
  form.streak_rewards = (data.streak_rewards || []).map((r) => ({ ...r }))
}

function addRule() {
  const used = new Set(form.streak_rewards.map((r) => Number(r.day)))
  const limit = Number(form.streak_cycle_days) || 365
  let day = 1
  while (used.has(day) && day < limit) day++
  form.streak_rewards.push({ day, bonus: 0.1 })
}

function removeRule(index: number) {
  form.streak_rewards.splice(index, 1)
}

async function load() {
  loading.value = true
  try {
    assign(await adminAPI.settings.getCheckinSettings())
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function save() {
  if (Number(form.reward_max) < Number(form.reward_min)) {
    appStore.showError(t('admin.settings.checkin.rewardInvalid'))
    return
  }
  saving.value = true
  try {
    const payload: CheckinSettings = {
      enabled: form.enabled,
      reward_min: Number(form.reward_min) || 0,
      reward_max: Number(form.reward_max) || 0,
      min_total_recharged: Number(form.min_total_recharged) || 0,
      min_account_age_days: Math.max(0, Math.trunc(Number(form.min_account_age_days) || 0)),
      streak_cycle_days: Math.max(0, Math.trunc(Number(form.streak_cycle_days) || 0)),
      streak_rewards: form.streak_rewards.map((r) => ({
        day: Math.trunc(Number(r.day) || 0),
        bonus: Number(r.bonus) || 0,
      })),
    }
    assign(await adminAPI.settings.updateCheckinSettings(payload))
    // 刷新公开设置，让侧边栏的签到入口立即出现或隐藏
    void appStore.fetchPublicSettings(true)
    appStore.showSuccess(t('admin.settings.checkin.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.ck-admin-badge {
  display: grid;
  place-items: center;
  width: 2.75rem;
  height: 2.75rem;
  flex-shrink: 0;
  border-radius: 0.875rem;
  color: #fff;
  background: linear-gradient(135deg, #f59e0b, #f97316);
  box-shadow: 0 8px 20px -10px rgba(249, 115, 22, 0.7);
}
</style>
