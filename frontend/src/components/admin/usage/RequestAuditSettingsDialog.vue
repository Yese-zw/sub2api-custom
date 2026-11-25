<template>
  <BaseDialog :show="show" :title="t('admin.requestAudit.settingsTitle')" width="normal" @close="$emit('close')">
    <div class="space-y-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <label class="input-label mb-0">{{ t('admin.requestAudit.settings.enabled') }}</label>
          <p class="mt-0.5 text-xs text-gray-400">{{ t('admin.requestAudit.settings.enabledHint') }}</p>
        </div>
        <Toggle v-model="form.enabled" data-testid="request-audit-enabled" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.requestAudit.settings.retentionDays') }}</label>
        <input v-model.number="form.retention_days" type="number" min="0" max="365" class="input" data-testid="request-audit-retention" />
        <p class="input-hint">{{ t('admin.requestAudit.settings.retentionDaysHint') }}</p>
      </div>
      <div>
        <label class="input-label">{{ t('admin.requestAudit.settings.maxBodyKB') }}</label>
        <input v-model.number="maxBodyKB" type="number" min="4" max="16384" class="input" data-testid="request-audit-max-body" />
        <p class="input-hint">{{ t('admin.requestAudit.settings.maxBodyKBHint') }}</p>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="saving || loading" data-testid="request-audit-save" @click="save">{{ t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Toggle from '@/components/common/Toggle.vue'
import { requestAuditAPI } from '@/api/admin/requestAudit'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const form = reactive({ enabled: true, retention_days: 7, max_body_bytes: 1024 * 1024 })
const maxBodyKB = ref(1024)

watch(
  () => props.show,
  async (show) => {
    if (!show) return
    loading.value = true
    try {
      const s = await requestAuditAPI.getSettings()
      form.enabled = s.enabled
      form.retention_days = s.retention_days
      maxBodyKB.value = Math.round(s.max_body_bytes / 1024)
    } catch (err) {
      appStore.showError(extractApiErrorMessage(err, t('admin.requestAudit.settings.loadFailed')))
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

async function save() {
  saving.value = true
  try {
    const kb = Math.min(16384, Math.max(4, Math.round(Number(maxBodyKB.value) || 1024)))
    const saved = await requestAuditAPI.updateSettings({
      enabled: form.enabled,
      retention_days: Math.min(365, Math.max(0, Math.round(Number(form.retention_days) || 0))),
      max_body_bytes: kb * 1024
    })
    form.retention_days = saved.retention_days
    maxBodyKB.value = Math.round(saved.max_body_bytes / 1024)
    appStore.showSuccess(t('admin.requestAudit.settings.saved'))
    emit('close')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.requestAudit.settings.saveFailed')))
  } finally {
    saving.value = false
  }
}
</script>
