<template>
  <BaseDialog :show="show" :title="t('admin.schedulingCenter.export.title', { name: groupName })" width="narrow" @close="emit('close')">
    <div class="space-y-4 text-sm text-gray-700 dark:text-gray-300">
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.schedulingCenter.export.hint') }}</p>
      <div>
        <div class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.schedulingCenter.export.range') }}</div>
        <div class="flex flex-wrap gap-2">
          <label
            v-for="opt in options"
            :key="opt.hours"
            class="flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2"
            :class="hours === opt.hours ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-600'"
          >
            <input v-model="hours" type="radio" :value="opt.hours" class="text-primary-600" />
            <span>{{ opt.label }}</span>
          </label>
        </div>
      </div>
      <ul class="list-disc space-y-1 pl-5 text-xs text-gray-500 dark:text-gray-400">
        <li v-for="item in contents" :key="item">{{ item }}</li>
      </ul>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="exporting" @click="doExport">
          {{ exporting ? t('admin.schedulingCenter.export.exporting') : t('admin.schedulingCenter.export.action') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import schedulingCenterAPI from '@/api/admin/schedulingCenter'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  groupId: number | null
  groupName: string
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const hours = ref(72)
const exporting = ref(false)

const options = computed(() => [
  { hours: 24, label: t('admin.schedulingCenter.compare.range24h') },
  { hours: 72, label: t('admin.schedulingCenter.compare.range3d') },
  { hours: 168, label: t('admin.schedulingCenter.compare.range7d') }
])

const contents = computed(() => [
  t('admin.schedulingCenter.export.items.comparison'),
  t('admin.schedulingCenter.export.items.accounts'),
  t('admin.schedulingCenter.export.items.events'),
  t('admin.schedulingCenter.export.items.iq'),
  t('admin.schedulingCenter.export.items.privacy')
])

function safeName(name: string): string {
  return name.replace(/[\\/:*?"<>|\s]+/g, '_').slice(0, 40) || 'group'
}

function stamp(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}`
}

async function doExport() {
  if (props.groupId == null) return
  exporting.value = true
  try {
    const data = await schedulingCenterAPI.exportData(props.groupId, hours.value)
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `scheduling-center-${props.groupId}-${safeName(props.groupName)}-${hours.value}h-${stamp()}.json`
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
    appStore.showSuccess(t('admin.schedulingCenter.export.done'))
    emit('close')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.schedulingCenter.export.failed')))
  } finally {
    exporting.value = false
  }
}
</script>
