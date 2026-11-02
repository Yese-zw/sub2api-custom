<template>
  <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
    <div class="mb-2 flex items-center justify-between">
      <label class="input-label !mb-0">{{ t('admin.users.operatorPermissions.title') }}</label>
      <div class="flex gap-2 text-xs">
        <button type="button" class="text-primary-600 hover:underline" @click="selectAll">{{ t('admin.users.operatorPermissions.selectAll') }}</button>
        <button type="button" class="text-gray-500 hover:underline" @click="clearAll">{{ t('admin.users.operatorPermissions.clear') }}</button>
      </div>
    </div>
    <p class="input-hint mb-3">{{ t('admin.users.operatorPermissions.hint') }}</p>
    <div v-if="loading" class="text-sm text-gray-500">{{ t('common.loading') }}</div>
    <div v-else class="max-h-72 space-y-2 overflow-y-auto pr-1">
      <div v-for="tab in catalog" :key="tab.key" class="flex flex-wrap items-center gap-x-4 gap-y-1">
        <span class="w-24 flex-shrink-0 text-sm font-medium text-gray-700 dark:text-gray-200">{{ tab.label }}</span>
        <label v-for="perm in tab.permissions" :key="perm.id" class="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-300">
          <input type="checkbox" :checked="selected.has(perm.id)" @change="toggle(tab.key, perm.id, ($event.target as HTMLInputElement).checked)" />
          {{ perm.label }}
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { OperatorPermissionTab } from '@/api/admin/users'

const props = defineProps<{ userId: number; initialRole: string }>()
const { t } = useI18n()

const catalog = ref<OperatorPermissionTab[]>([])
const selected = ref<Set<string>>(new Set())
const loading = ref(true)
const loaded = ref(false)

onMounted(async () => {
  try {
    catalog.value = await adminAPI.users.getOperatorPermissionCatalog()
    // 非运营用户尚无权限记录，直接从空集合开始
    if (props.initialRole === 'operator') {
      selected.value = new Set(await adminAPI.users.getOperatorPermissions(props.userId))
    }
    loaded.value = true
  } finally {
    loading.value = false
  }
})

function toggle(tabKey: string, id: string, checked: boolean) {
  const next = new Set(selected.value)
  if (checked) {
    next.add(id)
    next.add(`${tabKey}.view`) // 任何操作都需要能看到该标签页
  } else {
    next.delete(id)
    if (id === `${tabKey}.view`) {
      for (const p of catalog.value.find((x) => x.key === tabKey)?.permissions ?? []) next.delete(p.id)
    }
  }
  selected.value = next
}
function selectAll() {
  selected.value = new Set(catalog.value.flatMap((x) => x.permissions.map((p) => p.id)))
}
function clearAll() {
  selected.value = new Set()
}

/** 供父组件保存时读取；目录未加载成功时返回 null，避免误清空 */
defineExpose({
  getPermissions: (): string[] | null => (loaded.value ? [...selected.value] : null)
})
</script>
