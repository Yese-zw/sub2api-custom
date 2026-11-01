<template>
  <BaseDialog :show="show" :title="t('admin.accounts.iqCheck.title')" width="normal" @close="handleClose">
    <div class="space-y-4">
      <div v-if="account" class="rounded-xl border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-500 dark:bg-dark-700">
        <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
        <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.iqCheck.hint') }}</div>
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.iqCheck.model') }}
        </label>
        <Select
          v-model="selectedModel"
          :options="modelOptions"
          :disabled="running"
          :searchable="modelOptions.length > 5"
          :placeholder="t('admin.accounts.iqCheck.modelPlaceholder')"
        />
      </div>

      <div
        v-if="state !== 'idle'"
        class="rounded-xl border p-3 text-sm"
        :class="boxClass"
      >
        <div class="font-semibold">{{ stateText }}</div>
        <div v-if="task?.duration_ms" class="mt-1 text-xs opacity-80">
          {{ t('admin.accounts.iqCheck.duration') }}: {{ (task.duration_ms / 1000).toFixed(1) }}s
        </div>
        <div v-if="errorText" class="mt-1 whitespace-pre-wrap break-all text-xs">{{ errorText }}</div>
      </div>

      <div v-if="task?.answer" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.iqCheck.reply') }}
        </label>
        <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-xl bg-gray-100 p-3 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200">{{ task.answer }}</pre>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" @click="handleClose">{{ t('common.close') }}</button>
        <button class="btn btn-primary" :disabled="running || !account" @click="start">
          {{ running ? t('admin.accounts.iqCheck.running') : t('admin.accounts.iqCheck.start') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { adminAPI } from '@/api/admin'
import type { IQCheckTask } from '@/api/admin/accounts'
import type { Account, ClaudeModel } from '@/types'

const props = defineProps<{ show: boolean; account: Account | null }>()
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()

const POLL_MS = 3000
const modelOptions = ref<SelectOption[]>([])
const selectedModel = ref('')
const task = ref<IQCheckTask | null>(null)
const state = ref<'idle' | 'running' | 'passed' | 'incorrect' | 'error'>('idle')
const errorText = ref('')
let timer: ReturnType<typeof setTimeout> | null = null
let session = 0

const running = computed(() => state.value === 'running')
const stateText = computed(() => t(`admin.accounts.iqCheck.state.${state.value}`))
const boxClass = computed(() => ({
  running: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-300',
  passed: 'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-900/20 dark:text-green-300',
  incorrect: 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300',
  error: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300',
  idle: ''
}[state.value]))

const stopPolling = () => {
  session++
  if (timer) clearTimeout(timer)
  timer = null
}

const finish = (t0: IQCheckTask) => {
  task.value = t0
  errorText.value = t0.error || ''
  state.value = t0.verdict === 'passed' ? 'passed' : t0.verdict === 'incorrect' ? 'incorrect' : 'error'
}

const poll = (taskId: string, mySession: number) => {
  timer = setTimeout(async () => {
    if (mySession !== session) return
    try {
      const next = await adminAPI.accounts.getIQCheck(taskId)
      if (mySession !== session) return
      if (['succeeded', 'failed', 'cancelled'].includes(next.status)) finish(next)
      else {
        task.value = next
        poll(taskId, mySession)
      }
    } catch (e: any) {
      if (mySession !== session) return
      state.value = 'error'
      errorText.value = e?.message || String(e)
    }
  }, POLL_MS)
}

const start = async () => {
  if (!props.account || running.value) return
  stopPolling()
  const mySession = session
  state.value = 'running'
  errorText.value = ''
  task.value = null
  try {
    const created = await adminAPI.accounts.startIQCheck(props.account.id, selectedModel.value)
    if (mySession !== session) return
    if (['succeeded', 'failed', 'cancelled'].includes(created.status)) finish(created)
    else {
      task.value = created
      poll(created.id, mySession)
    }
  } catch (e: any) {
    if (mySession !== session) return
    state.value = 'error'
    errorText.value = e?.message || String(e)
  }
}

const handleClose = () => {
  stopPolling()
  emit('close')
}

watch(
  () => [props.show, props.account?.id] as const,
  async ([visible, id]) => {
    if (!visible || !id) return
    stopPolling()
    state.value = 'idle'
    task.value = null
    errorText.value = ''
    selectedModel.value = ''
    modelOptions.value = []
    try {
      const models = await adminAPI.accounts.getAvailableModels(id)
      if (props.account?.id !== id) return
      modelOptions.value = models.map((m: ClaudeModel) => ({ value: m.id, label: m.display_name || m.id }))
    } catch {
      modelOptions.value = []
    }
  },
  { immediate: true }
)

onBeforeUnmount(stopPolling)
</script>
