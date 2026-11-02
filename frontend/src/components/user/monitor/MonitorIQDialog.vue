<template>
  <Teleport to="body">
    <div
      v-if="show"
      class="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 p-3"
      @click.self="emit('close')"
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby="monitor-iq-title"
        class="flex flex-col overflow-hidden rounded-2xl bg-white shadow-xl dark:bg-dark-800"
        style="width: calc(100vw - 48px); height: 92dvh; max-width: 1280px; max-height: 960px"
      >
        <header class="flex shrink-0 items-start gap-3 border-b p-3 dark:border-dark-600">
          <div class="min-w-0 flex-1">
            <h2 id="monitor-iq-title" class="truncate text-lg font-bold text-gray-900 dark:text-gray-100">
              {{ t('monitorIQ.title', { name: monitorName }) }}
            </h2>
            <p class="mt-1 text-xs text-gray-500">{{ t('monitorIQ.subtitle') }}</p>
          </div>
          <button class="btn btn-secondary btn-sm" type="button" @click="emit('close')">
            {{ t('monitorIQ.close') }}
          </button>
        </header>

        <div role="tablist" class="flex shrink-0 flex-wrap gap-2 border-b px-3 py-2 dark:border-dark-600">
          <button
            v-for="tab in tabs"
            :key="tab"
            type="button"
            role="tab"
            :aria-selected="benchmark === tab"
            :class="['btn', benchmark === tab ? 'btn-primary' : 'btn-secondary']"
            @click="benchmark = tab"
          >
            {{ t(`monitorIQ.tabs.${tab}`) }}
          </button>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto p-4">
          <p v-if="loading" class="text-sm text-gray-500">{{ t('monitorIQ.loading') }}</p>
          <p v-else-if="error" class="text-sm text-red-600">{{ error }}</p>
          <p v-else-if="benchmarkItems.length === 0" class="text-sm text-gray-500">{{ t('monitorIQ.empty') }}</p>
          <template v-else>
            <div
              role="tablist"
              class="flex flex-wrap items-center gap-2 border-b pb-3 dark:border-dark-600"
            >
              <button
                v-for="m in models"
                :key="m"
                type="button"
                role="tab"
                :aria-selected="model === m"
                :class="[
                  'btn btn-secondary break-all',
                  model === m ? 'border-primary-500 text-primary-700 dark:text-primary-300' : ''
                ]"
                @click="model = m"
              >
                {{ m }}
              </button>
            </div>

            <template v-if="benchmark === 'candy'">
              <div class="my-3 grid gap-3 sm:grid-cols-2">
                <div class="rounded-xl border p-3 dark:border-dark-600">
                  <div class="text-xs text-gray-500">
                    {{ t('monitorIQ.statsTotal') }}
                  </div>
                  <strong class="text-xl text-gray-900 dark:text-gray-100">
                    {{ modelItems.length }} · {{ accuracy }}
                  </strong>
                </div>
                <div class="rounded-xl border p-3 dark:border-dark-600">
                  <div class="text-xs text-gray-500">{{ t('monitorIQ.statsValid') }}</div>
                  <strong class="text-xl text-gray-900 dark:text-gray-100">
                    {{ modelItems.length }} / {{ correctCount }}
                  </strong>
                </div>
              </div>
              <p class="mb-3 text-xs text-gray-500">{{ t('monitorIQ.statsNote') }}</p>
            </template>
            <p v-else class="my-3 text-xs text-gray-500">{{ t('monitorIQ.pelicanNote') }}</p>

            <div class="grid gap-4 md:grid-cols-[270px_minmax(0,1fr)]">
              <aside class="space-y-2">
                <button
                  v-for="item in modelItems"
                  :key="item.id"
                  type="button"
                  :class="[
                    'w-full rounded-xl border p-3 text-left dark:border-dark-600',
                    selected?.id === item.id ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20' : ''
                  ]"
                  @click="selectedId = item.id"
                >
                  <strong class="block tabular-nums text-gray-900 dark:text-gray-100">{{ formatTime(item.started_at) }}</strong>
                  <span :class="['mt-1 block', itemTone(item)]">{{ itemLabel(item) }}</span>
                  <span class="block text-xs text-gray-500">{{ item.model }} · {{ formatDuration(item.duration_ms) }}</span>
                </button>
              </aside>

              <section v-if="selected" class="min-w-0 space-y-3 rounded-xl border p-4 dark:border-dark-600">
                <div class="flex flex-wrap items-center gap-3">
                  <h3 :class="['text-xl font-bold', itemTone(selected)]">{{ itemLabel(selected) }}</h3>
                  <span v-if="benchmark === 'candy'" class="text-sm text-gray-500">{{ t('monitorIQ.standardAnswer') }}</span>
                </div>

                <template v-if="benchmark === 'candy'">
                  <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
                    <p class="text-xs text-gray-500">{{ t('monitorIQ.rawAnswer') }}</p>
                    <pre class="mt-2 whitespace-pre-wrap break-words text-xl text-gray-900 dark:text-gray-100">{{ selected.answer }}</pre>
                  </div>
                  <details>
                    <summary class="cursor-pointer text-sm text-gray-600 dark:text-gray-300">
                      {{ t('monitorIQ.promptSnapshot') }}
                    </summary>
                    <pre class="mt-2 whitespace-pre-wrap break-words text-sm text-gray-700 dark:text-gray-300">{{ CANDY_PROMPT }}</pre>
                  </details>
                </template>

                <template v-else>
                  <p v-if="selected.reason" class="text-sm text-gray-600 dark:text-gray-300">
                    <span class="text-xs text-gray-500">{{ t('monitorIQ.reason') }}：</span>{{ selected.reason }}
                  </p>
                  <p v-if="!selected.has_html" class="text-sm text-gray-500">{{ t('monitorIQ.noHtml') }}</p>
                  <template v-else>
                    <div class="flex flex-wrap gap-2">
                      <button class="btn btn-secondary btn-sm" type="button" @click="previewing = !previewing">
                        {{ previewing ? t('monitorIQ.stopPreview') : t('monitorIQ.startPreview') }}
                      </button>
                      <button class="btn btn-ghost btn-sm" type="button" @click="showSource = !showSource">
                        {{ showSource ? t('monitorIQ.hideSource') : t('monitorIQ.viewSource') }}
                      </button>
                    </div>
                    <p v-if="detailLoading" class="text-sm text-gray-500">{{ t('monitorIQ.loading') }}</p>
                    <iframe
                      v-else-if="previewing && detail?.html"
                      :title="t('monitorIQ.preview')"
                      sandbox="allow-scripts"
                      referrerpolicy="no-referrer"
                      :srcdoc="previewDoc"
                      class="h-[420px] w-full rounded-xl border bg-white dark:border-dark-600"
                    ></iframe>
                    <pre
                      v-if="showSource && detail?.html"
                      class="max-h-[420px] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-300"
                    >{{ detail.html }}</pre>
                  </template>
                </template>

                <p class="text-xs text-gray-500">
                  {{ t('monitorIQ.started') }}：{{ formatTime(selected.started_at) }} ·
                  {{ t('monitorIQ.finished') }}：{{ formatTime(selected.finished_at) }}
                </p>
              </section>
            </div>
          </template>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  channelMonitorUserAPI,
  type MonitorIQBenchmark,
  type MonitorIQResult,
} from '@/api/channelMonitor'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  monitorId: number
  monitorName: string
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()

// 与后端 iqCheckQuestion 保持一致，仅用于「提示词快照」展示。
const CANDY_PROMPT =
  '在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。' +
  '现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？' +
  '（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）\n' +
  '形状 | 苹果味 | 桃子味 | 西瓜味\n圆形 | 7 | 9 | 8\n五角星形 | 7 | 6 | 4\n' +
  '不许联网，自己计算。只回答最少取出的糖果总数，使用一个整数，不要解释。'

const MAX_VISIBLE = 10
const CSP =
  "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; frame-src 'none'; object-src 'none'; worker-src 'none'; base-uri 'none'; form-action 'none'"

const loading = ref(false)
const error = ref('')
const items = ref<MonitorIQResult[]>([])
const creativeEnabled = ref(false)
const benchmark = ref<MonitorIQBenchmark>('candy')
const model = ref('')
const selectedId = ref<number | null>(null)
const detail = ref<MonitorIQResult | null>(null)
const detailLoading = ref(false)
const previewing = ref(true)
const showSource = ref(false)
const detailCache = new Map<number, MonitorIQResult>()
let loadSeq = 0

const tabs = computed<MonitorIQBenchmark[]>(() => {
  const base: MonitorIQBenchmark[] = ['candy', 'pelican']
  if (creativeEnabled.value || items.value.some(i => i.benchmark === 'creative')) base.push('creative')
  return base
})

const benchmarkItems = computed(() => items.value.filter(i => i.benchmark === benchmark.value))
const models = computed(() => Array.from(new Set(benchmarkItems.value.map(i => i.model))))
const modelItems = computed(() =>
  benchmarkItems.value.filter(i => i.model === model.value).slice(0, MAX_VISIBLE)
)
const selected = computed(() => modelItems.value.find(i => i.id === selectedId.value) ?? modelItems.value[0] ?? null)
const correctCount = computed(() => modelItems.value.filter(i => i.verdict === 'passed').length)
const accuracy = computed(() =>
  modelItems.value.length ? `${((correctCount.value / modelItems.value.length) * 100).toFixed(1)}%` : '-'
)

const previewDoc = computed(() => {
  const html = detail.value?.html ?? ''
  return `<!doctype html><meta http-equiv="Content-Security-Policy" content="${CSP}"><meta name="referrer" content="no-referrer">${html}`
})

function itemLabel(item: MonitorIQResult): string {
  if (item.benchmark === 'candy') {
    return item.verdict === 'passed' ? `✓ ${t('monitorIQ.correct')}` : `✗ ${t('monitorIQ.incorrect')}`
  }
  if (item.label) return item.label
  return item.quality && t(`monitorIQ.quality.${item.quality}`) !== `monitorIQ.quality.${item.quality}`
    ? t(`monitorIQ.quality.${item.quality}`)
    : item.quality || '-'
}

function itemTone(item: MonitorIQResult): string {
  if (item.benchmark === 'candy') {
    return item.verdict === 'passed' ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'
  }
  if (item.quality === 'normal') return 'text-emerald-700 dark:text-emerald-400'
  if (item.quality === 'degraded' || item.quality === 'suspicious') return 'text-red-600 dark:text-red-400'
  return 'text-gray-600 dark:text-gray-300'
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

function formatDuration(ms: number): string {
  return `${(ms / 1000).toFixed(1)}s`
}

async function load() {
  const seq = ++loadSeq
  loading.value = true
  error.value = ''
  items.value = []
  detailCache.clear()
  try {
    const res = await channelMonitorUserAPI.iqResults(props.monitorId)
    if (seq !== loadSeq) return
    items.value = res.items ?? []
    creativeEnabled.value = res.creative_enabled
    const first = (['candy', 'pelican', 'creative'] as MonitorIQBenchmark[]).find(b =>
      items.value.some(i => i.benchmark === b)
    )
    benchmark.value = first ?? 'candy'
  } catch (e: unknown) {
    if (seq !== loadSeq) return
    error.value = extractApiErrorMessage(e, t('monitorIQ.loadFailed'))
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

// 切换检测类型 / 模型时选中第一条。
watch([benchmark, items], () => {
  if (!models.value.includes(model.value)) model.value = models.value[0] ?? ''
})
watch([model, benchmark], () => {
  selectedId.value = modelItems.value[0]?.id ?? null
})

// 鹈鹕/自由创作：选中后按需拉取含 html 的详情（带缓存）。
watch(
  selected,
  async (item) => {
    detail.value = null
    showSource.value = false
    previewing.value = true
    if (!item || item.benchmark === 'candy' || !item.has_html) return
    const cached = detailCache.get(item.id)
    if (cached) {
      detail.value = cached
      return
    }
    detailLoading.value = true
    try {
      const full = await channelMonitorUserAPI.iqResult(props.monitorId, item.id)
      detailCache.set(item.id, full)
      if (selected.value?.id === item.id) detail.value = full
    } catch {
      // 详情拉取失败时保留列表信息
    } finally {
      detailLoading.value = false
    }
  },
  { immediate: false }
)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      window.addEventListener('keydown', onKeydown)
      void load()
    } else {
      window.removeEventListener('keydown', onKeydown)
      loadSeq++
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>
