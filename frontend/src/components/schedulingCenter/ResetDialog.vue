<template>
  <BaseDialog :show="show" :title="t('admin.schedulingCenter.reset.title', { name: groupName })" width="normal" @close="emit('close')">
    <div class="space-y-4 text-sm text-gray-700 dark:text-gray-300">
      <p class="rounded-lg bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300">
        {{ t('admin.schedulingCenter.reset.warning') }}
      </p>

      <div class="space-y-2">
        <label
          v-for="item in items"
          :key="item.key"
          class="flex cursor-pointer items-start gap-3 rounded-lg border border-gray-200 px-3 py-2.5 hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700/40"
        >
          <input v-model="options[item.key]" type="checkbox" class="sc-checkbox mt-0.5" />
          <span class="min-w-0">
            <span class="block font-medium text-gray-900 dark:text-white">{{ item.label }}</span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{{ item.hint }}</span>
          </span>
        </label>
      </div>

      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.reset.after') }}</p>

      <label class="block">
        <span class="mb-1 block text-xs text-gray-600 dark:text-gray-300">
          {{ t('admin.schedulingCenter.reset.confirmLabel', { name: groupName }) }}
        </span>
        <input v-model="confirmText" type="text" class="input text-sm" :placeholder="groupName" />
      </label>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-danger" :disabled="!canSubmit || resetting" @click="doReset">
          {{ resetting ? t('admin.schedulingCenter.reset.running') : t('admin.schedulingCenter.reset.action') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import schedulingCenterAPI, {
  type SchedulingCenterResetOptions,
  type SchedulingCenterResetResult
} from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  groupId: number | null
  groupName: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'done', result: SchedulingCenterResetResult): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const defaults = (): SchedulingCenterResetOptions => ({
  clear_pool: true,
  clear_states: true,
  clear_events: true,
  clear_records: true,
  reset_settings: false
})
const options = reactive<SchedulingCenterResetOptions>(defaults())
const confirmText = ref('')
const resetting = ref(false)

watch(
  () => props.show,
  (show) => {
    if (!show) return
    Object.assign(options, defaults())
    confirmText.value = ''
  }
)

const items = computed(() =>
  (['clear_pool', 'clear_states', 'clear_events', 'clear_records', 'reset_settings'] as const).map((key) => ({
    key,
    label: t(`admin.schedulingCenter.reset.items.${key}`),
    hint: t(`admin.schedulingCenter.reset.hints.${key}`)
  }))
)

const anySelected = computed(() => Object.values(options).some(Boolean))
// 输入分组名确认，防止误操作
const canSubmit = computed(() => anySelected.value && confirmText.value.trim() === props.groupName.trim())

async function doReset() {
  if (props.groupId == null || !canSubmit.value) return
  resetting.value = true
  try {
    const result = await schedulingCenterAPI.resetGroup(props.groupId, { ...options })
    appStore.showSuccess(
      t('admin.schedulingCenter.reset.done', {
        pool: result.pool_members,
        states: result.states,
        events: result.events,
        records: result.records
      })
    )
    emit('done', result)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.reset.failed')))
  } finally {
    resetting.value = false
  }
}
</script>
