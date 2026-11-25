<template>
  <BaseDialog :show="show" :title="t('admin.requestAudit.detailTitle')" width="extra-wide" @close="$emit('close')">
    <div v-if="loading" class="py-10 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.requestAudit.loading') }}
    </div>
    <div v-else-if="errorMessage" class="rounded-lg bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-500/10 dark:text-amber-300" data-testid="request-audit-error">
      {{ errorMessage }}
    </div>
    <div v-else-if="record" class="space-y-4" data-testid="request-audit-detail">
      <!-- 摘要 -->
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
        <span class="font-mono text-gray-700 dark:text-gray-200">{{ record.method }} {{ record.path }}</span>
        <span v-if="record.model">{{ t('admin.requestAudit.meta.model') }}: <span class="text-gray-700 dark:text-gray-200">{{ record.model }}</span></span>
        <span>{{ t('admin.requestAudit.meta.status') }}: <span :class="record.status_code >= 400 ? 'text-red-500' : 'text-emerald-600 dark:text-emerald-400'">{{ record.status_code }}</span></span>
        <span>{{ t('admin.requestAudit.meta.duration') }}: {{ formatMs(record.duration_ms) }}</span>
        <span v-if="record.stream" class="rounded bg-sky-100 px-1.5 py-px text-sky-700 dark:bg-sky-500/20 dark:text-sky-300">stream</span>
      </div>

      <!-- Tabs -->
      <div class="flex gap-1 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in tabs"
          :key="tab"
          type="button"
          class="-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors"
          :class="activeTab === tab
            ? 'border-primary-500 text-primary-600 dark:text-primary-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="activeTab = tab"
        >
          {{ t(`admin.requestAudit.tabs.${tab}`) }}
        </button>
      </div>

      <!-- 对话 -->
      <div v-if="activeTab === 'conversation'" class="space-y-5">
        <section>
          <h4 class="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.requestAudit.input') }}</h4>
          <p v-if="record.request_truncated" class="mb-2 text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.requestAudit.truncated', { size: formatAuditBytes(record.request_body_size), limit: formatAuditBytes(record.request_body.length) }) }}
          </p>
          <div v-if="requestMessages.length" class="space-y-2">
            <div
              v-for="(msg, idx) in requestMessages"
              :key="idx"
              class="rounded-lg border border-gray-200 dark:border-dark-700"
            >
              <div class="flex items-center justify-between border-b border-gray-100 px-3 py-1.5 dark:border-dark-700">
                <span class="text-xs font-semibold uppercase tracking-wide" :class="roleClass(msg.role)">{{ msg.role }}</span>
                <button type="button" class="text-xs text-gray-400 hover:text-primary-500" @click="copy(msg.text)">{{ t('admin.requestAudit.copy') }}</button>
              </div>
              <pre class="audit-pre">{{ msg.text || t('admin.requestAudit.empty') }}</pre>
            </div>
          </div>
          <p v-else class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.requestAudit.noParsedInput') }}</p>
        </section>

        <section>
          <h4 class="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.requestAudit.output') }}</h4>
          <p v-if="record.response_truncated" class="mb-2 text-xs text-amber-600 dark:text-amber-400">
            {{ t('admin.requestAudit.truncated', { size: formatAuditBytes(record.response_body_size), limit: formatAuditBytes(record.response_body.length) }) }}
          </p>
          <div v-if="responseOutput" class="space-y-2">
            <div v-if="responseOutput.thinking" class="rounded-lg border border-violet-200 dark:border-violet-500/30">
              <div class="border-b border-violet-100 px-3 py-1.5 text-xs font-semibold text-violet-600 dark:border-violet-500/20 dark:text-violet-300">{{ t('admin.requestAudit.thinking') }}</div>
              <pre class="audit-pre">{{ responseOutput.thinking }}</pre>
            </div>
            <div v-if="responseOutput.text" class="rounded-lg border border-emerald-200 dark:border-emerald-500/30">
              <div class="flex items-center justify-between border-b border-emerald-100 px-3 py-1.5 dark:border-emerald-500/20">
                <span class="text-xs font-semibold uppercase tracking-wide text-emerald-600 dark:text-emerald-300">assistant</span>
                <button type="button" class="text-xs text-gray-400 hover:text-primary-500" @click="copy(responseOutput.text)">{{ t('admin.requestAudit.copy') }}</button>
              </div>
              <pre class="audit-pre">{{ responseOutput.text }}</pre>
            </div>
            <div v-if="responseOutput.tools.length" class="rounded-lg border border-amber-200 dark:border-amber-500/30">
              <div class="border-b border-amber-100 px-3 py-1.5 text-xs font-semibold text-amber-600 dark:border-amber-500/20 dark:text-amber-300">{{ t('admin.requestAudit.tools') }}</div>
              <pre class="audit-pre">{{ responseOutput.tools.join('\n\n') }}</pre>
            </div>
          </div>
          <p v-else class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.requestAudit.noParsedOutput') }}</p>
        </section>
      </div>

      <!-- 请求原文 / 响应原文 -->
      <div v-else-if="activeTab === 'request' || activeTab === 'response'">
        <div class="mb-2 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
          <span>
            {{ t('admin.requestAudit.size') }}: {{ formatAuditBytes(activeTab === 'request' ? record.request_body_size : record.response_body_size) }}
            <span class="ml-2 font-mono">{{ activeTab === 'request' ? record.request_content_type : record.response_content_type }}</span>
            <span v-if="activeTab === 'request' ? record.request_truncated : record.response_truncated" class="ml-2 text-amber-600 dark:text-amber-400">truncated</span>
          </span>
          <button type="button" class="btn btn-secondary btn-sm" @click="copy(activeTab === 'request' ? record.request_body : record.response_body)">{{ t('admin.requestAudit.copy') }}</button>
        </div>
        <pre class="audit-pre max-h-[60vh] rounded-lg border border-gray-200 dark:border-dark-700">{{ (activeTab === 'request' ? prettyRequest : prettyResponse) || t('admin.requestAudit.empty') }}</pre>
      </div>

      <!-- 元数据 -->
      <div v-else class="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1.5 text-sm">
        <template v-for="row in metaRows" :key="row.label">
          <span class="text-gray-500 dark:text-gray-400">{{ row.label }}</span>
          <span class="break-all font-mono text-xs text-gray-800 dark:text-gray-200">{{ row.value || '-' }}</span>
        </template>
        <span class="text-gray-500 dark:text-gray-400">{{ t('admin.requestAudit.meta.headers') }}</span>
        <pre class="audit-pre rounded-lg border border-gray-200 dark:border-dark-700">{{ headersText }}</pre>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { requestAuditAPI, type RequestAuditLog } from '@/api/admin/requestAudit'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatAuditBytes, parseAuditRequestMessages, parseAuditResponseOutput, prettyAuditBody } from '@/utils/requestAuditParse'

const props = defineProps<{ show: boolean; usageLogId: number | null }>()
defineEmits<{ close: [] }>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

type Tab = 'conversation' | 'request' | 'response' | 'meta'
const tabs: Tab[] = ['conversation', 'request', 'response', 'meta']
const activeTab = ref<Tab>('conversation')
const loading = ref(false)
const errorMessage = ref('')
const record = ref<RequestAuditLog | null>(null)
let loadSeq = 0

watch(
  () => [props.show, props.usageLogId] as const,
  async ([show, id]) => {
    if (!show || !id) return
    const seq = ++loadSeq
    activeTab.value = 'conversation'
    loading.value = true
    errorMessage.value = ''
    record.value = null
    try {
      const data = await requestAuditAPI.getByUsageLog(id)
      if (seq !== loadSeq) return
      record.value = data
    } catch (err: any) {
      if (seq !== loadSeq) return
      const status = err?.status ?? err?.response?.status
      const code = err?.code ?? err?.reason ?? err?.response?.data?.reason
      errorMessage.value = status === 404 || code === 'REQUEST_AUDIT_NOT_FOUND'
        ? t('admin.requestAudit.notFound')
        : extractApiErrorMessage(err, t('admin.requestAudit.loadFailed'))
    } finally {
      if (seq === loadSeq) loading.value = false
    }
  },
  { immediate: true }
)

const requestMessages = computed(() => (record.value ? parseAuditRequestMessages(record.value.request_body) : []))
const responseOutput = computed(() => (record.value ? parseAuditResponseOutput(record.value.response_body) : null))
const prettyRequest = computed(() => (record.value ? prettyAuditBody(record.value.request_body) : ''))
const prettyResponse = computed(() => (record.value ? prettyAuditBody(record.value.response_body) : ''))
const headersText = computed(() => {
  const headers = record.value?.request_headers ?? {}
  return Object.keys(headers).sort().map((k) => `${k}: ${headers[k]}`).join('\n') || '-'
})

const metaRows = computed(() => {
  const r = record.value
  if (!r) return []
  return [
    { label: t('admin.requestAudit.meta.createdAt'), value: new Date(r.created_at).toLocaleString() },
    { label: t('admin.requestAudit.meta.path'), value: `${r.method} ${r.path}` },
    { label: t('admin.requestAudit.meta.model'), value: r.model },
    { label: t('admin.requestAudit.meta.stream'), value: r.stream ? 'true' : 'false' },
    { label: t('admin.requestAudit.meta.status'), value: String(r.status_code) },
    { label: t('admin.requestAudit.meta.duration'), value: formatMs(r.duration_ms) },
    { label: t('admin.requestAudit.meta.clientIp'), value: r.client_ip },
    { label: t('admin.requestAudit.meta.userAgent'), value: r.user_agent },
    { label: t('admin.requestAudit.meta.requestId'), value: r.request_id },
    { label: t('admin.requestAudit.meta.clientRequestId'), value: r.client_request_id }
  ]
})

function formatMs(ms: number): string {
  if (!Number.isFinite(ms)) return '-'
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`
}

function roleClass(role: string): string {
  switch (role) {
    case 'system':
    case 'developer':
      return 'text-gray-500 dark:text-gray-400'
    case 'assistant':
    case 'model':
      return 'text-emerald-600 dark:text-emerald-300'
    case 'tool':
      return 'text-amber-600 dark:text-amber-300'
    default:
      return 'text-sky-600 dark:text-sky-300'
  }
}

function copy(text: string) {
  void copyToClipboard(text, t('admin.requestAudit.copied'))
}
</script>

<style scoped>
.audit-pre {
  @apply max-h-96 overflow-auto whitespace-pre-wrap break-words px-3 py-2 font-mono text-xs leading-relaxed text-gray-800 dark:text-gray-200;
}
</style>
