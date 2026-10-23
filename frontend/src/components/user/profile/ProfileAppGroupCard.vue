<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.appGroup.title', { app: appLabel }) }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.appGroup.description', { app: appLabel }) }}
      </p>
    </div>
    <div class="px-6 py-6">
      <label class="input-label">{{ t('profile.appGroup.label') }}</label>
      <Select
        :model-value="selectedGroupId"
        :options="options"
        :disabled="loading || saving"
        :aria-label="t('profile.appGroup.label')"
        searchable="auto"
        @update:model-value="onSelect"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { userAPI, userGroupsAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Group } from '@/types'

const props = defineProps<{
  appName: string
  defaultGroupId: number
}>()

const { t } = useI18n()
const appStore = useAppStore()

const groups = ref<Group[]>([])
const selectedGroupId = ref(0)
const loading = ref(true)
const saving = ref(false)

const appLabel = computed(() => props.appName.trim() || 'App')

const options = computed<SelectOption[]>(() => {
  const defaultGroup = groups.value.find((group) => group.id === props.defaultGroupId)
  return [
    {
      value: 0,
      label: t('profile.appGroup.followDefault', {
        group: defaultGroup?.name ?? t('profile.appGroup.defaultUnavailable'),
      }),
    },
    ...groups.value.map((group) => ({ value: group.id, label: `${group.name} · ${group.platform}` })),
  ]
})

async function onSelect(value: string | number | boolean | null) {
  const next = typeof value === 'number' ? value : 0
  if (next === selectedGroupId.value) return
  const previous = selectedGroupId.value
  selectedGroupId.value = next
  saving.value = true
  try {
    selectedGroupId.value = await userAPI.updateAppGroup(next)
    appStore.showSuccess(t('profile.appGroup.saved', { app: appLabel.value }))
  } catch (err) {
    selectedGroupId.value = previous
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const [available, current] = await Promise.all([
      userGroupsAPI.getAvailable(),
      userAPI.getAppGroup(),
    ])
    groups.value = available
    selectedGroupId.value = current
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
})
</script>
