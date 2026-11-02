<template>
  <Teleport to="body">
    <div v-if="show" class="iq-overlay rx-theme" @click.self="emit('close')">
      <section role="dialog" aria-modal="true" aria-labelledby="monitor-iq-title" class="iq-panel">
        <header class="iq-head">
          <div class="iq-head__text">
            <span class="iq-head__eyebrow">IQ</span>
            <div class="min-w-0">
              <h2 id="monitor-iq-title" class="iq-head__title">{{ t('monitorIQ.title', { name: monitorName }) }}</h2>
              <p class="iq-head__sub">{{ t('monitorIQ.subtitle') }}</p>
            </div>
          </div>
          <button class="iq-close" type="button" :aria-label="t('monitorIQ.close')" @click="emit('close')">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </header>

        <!-- 模型切换 -->
        <div v-if="!loading && !error && models.length > 0" class="iq-toolbar">
          <span class="iq-toolbar__label">{{ t('monitorIQ.modelLabel') }}</span>
          <div role="tablist" class="iq-models">
            <button
              v-for="m in models"
              :key="m"
              type="button"
              role="tab"
              :aria-selected="model === m"
              :class="['iq-model', { 'is-active': model === m }]"
              @click="model = m"
            >
              {{ m }}
            </button>
          </div>
        </div>

        <div class="iq-body">
          <p v-if="loading" class="iq-state">{{ t('monitorIQ.loading') }}</p>
          <p v-else-if="error" class="iq-state iq-state--error">{{ error }}</p>
          <p v-else-if="items.length === 0 && failures.length === 0" class="iq-state">{{ t('monitorIQ.empty') }}</p>

          <div v-else :class="['iq-grid', { 'is-single': kinds.length === 1 }]">
            <article v-for="(kind, idx) in kinds" :key="kind" class="iq-card">
              <header class="iq-card__head">
                <span class="iq-card__idx">{{ String(idx + 1).padStart(2, '0') }}</span>
                <h3 class="iq-card__title">{{ t(`monitorIQ.tabs.${kind}`) }}</h3>
              </header>

              <!-- 最近一次失败原因 -->
              <div v-if="failuresOf(kind).length" class="iq-fails">
                <div v-for="f in failuresOf(kind)" :key="`${f.model}|${f.benchmark}`" class="iq-fail">
                  <div class="iq-fail__meta">
                    <span class="iq-fail__badge">{{ t('monitorIQ.failuresTitle') }}</span>
                    <span>{{ f.model }}<template v-if="f.benchmark === 'creative'"> · {{ t('monitorIQ.tabs.creative') }}</template> · {{ formatTime(f.at) }}</span>
                  </div>
                  <p class="iq-fail__msg">{{ f.message }}</p>
                </div>
              </div>

              <p v-if="itemsOf(kind).length === 0" class="iq-state iq-state--inline">{{ t('monitorIQ.emptyKind') }}</p>

              <template v-else>
                <!-- 概览：分段环 + 通过数 -->
                <div class="iq-hero">
                  <MonitorIQRing :recent="ringOf(kind)" size="6rem" />
                  <div class="iq-hero__text">
                    <div class="iq-hero__big">{{ statsOf(kind).passed }}<small>/ {{ statsOf(kind).total }}</small></div>
                    <div class="iq-hero__label">{{ t('monitorIQ.statPassed') }}</div>
                    <div class="iq-hero__last">{{ t('monitorIQ.lastRun') }} · {{ formatTime(statsOf(kind).last) }}</div>
                  </div>
                </div>

                <!-- 时间条：点击某一格查看该次详情 -->
                <div class="iq-strip" role="listbox" :aria-label="t(`monitorIQ.tabs.${kind}`)">
                  <template v-for="(slot, i) in slotsOf(kind)" :key="slot ? slot.id : `empty-${i}`">
                    <button
                      v-if="slot"
                      type="button"
                      role="option"
                      :aria-selected="selectedByKind[kind]?.id === slot.id"
                      :title="slotTitle(slot)"
                      :class="['iq-slot', itemTone(slot), { 'is-active': selectedByKind[kind]?.id === slot.id }]"
                      @click="selectedIds[kind] = slot.id"
                    ></button>
                    <span v-else class="iq-slot is-empty" aria-hidden="true"></span>
                  </template>
                </div>
                <div class="iq-axis"><span>{{ t('monitorIQ.older') }}</span><span>{{ t('monitorIQ.newer') }}</span></div>

                <!-- 选中那次的详情 -->
                <div v-if="selectedByKind[kind]" class="iq-run">
                  <div class="iq-run__head">
                    <h4 :class="['iq-run__verdict', itemTone(selectedByKind[kind]!)]">
                      <span class="iq-run__dot" aria-hidden="true"></span>{{ itemLabel(selectedByKind[kind]!) }}
                    </h4>
                    <span v-if="selectedByKind[kind]!.benchmark === 'creative'" class="iq-pill">{{ t('monitorIQ.tabs.creative') }}</span>
                    <span class="iq-run__meta">
                      {{ formatTime(selectedByKind[kind]!.started_at) }} · {{ t('monitorIQ.duration') }} {{ formatDuration(selectedByKind[kind]!.duration_ms) }}
                    </span>
                  </div>

                  <template v-if="kind === 'candy'">
                    <dl class="iq-kv">
                      <div>
                        <dt>{{ t('monitorIQ.rawAnswer') }}</dt>
                        <dd class="iq-kv__answer">{{ selectedByKind[kind]!.answer }}</dd>
                      </div>
                      <div>
                        <dt>{{ t('monitorIQ.standardAnswerLabel') }}</dt>
                        <dd class="iq-kv__answer">21</dd>
                      </div>
                    </dl>
                    <details class="iq-details">
                      <summary>{{ t('monitorIQ.promptSnapshot') }}</summary>
                      <pre>{{ CANDY_PROMPT }}</pre>
                    </details>
                  </template>

                  <template v-else>
                    <p v-if="selectedByKind[kind]!.reason" class="iq-reason">
                      <span>{{ t('monitorIQ.reason') }}</span>{{ selectedByKind[kind]!.reason }}
                    </p>
                    <p v-if="!selectedByKind[kind]!.has_html" class="iq-state iq-state--inline">{{ t('monitorIQ.noHtml') }}</p>
                    <template v-else>
                      <div class="iq-actions">
                        <button class="iq-btn" type="button" @click="previewing = !previewing">
                          {{ previewing ? t('monitorIQ.stopPreview') : t('monitorIQ.startPreview') }}
                        </button>
                        <button class="iq-btn iq-btn--ghost" type="button" @click="showSource = !showSource">
                          {{ showSource ? t('monitorIQ.hideSource') : t('monitorIQ.viewSource') }}
                        </button>
                      </div>
                      <p v-if="detailLoading" class="iq-state iq-state--inline">{{ t('monitorIQ.loading') }}</p>
                      <iframe
                        v-else-if="previewing && detail?.html"
                        :title="t('monitorIQ.preview')"
                        sandbox="allow-scripts"
                        referrerpolicy="no-referrer"
                        :srcdoc="previewDoc"
                        class="iq-frame"
                      ></iframe>
                      <pre v-if="showSource && detail?.html" class="iq-source">{{ detail.html }}</pre>
                    </template>
                  </template>
                </div>

                <p class="iq-note">{{ kind === 'candy' ? t('monitorIQ.statsNote') : t('monitorIQ.pelicanNote') }}</p>
              </template>
            </article>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { channelMonitorUserAPI, type MonitorIQFailure, type MonitorIQResult } from '@/api/channelMonitor'
import { extractApiErrorMessage } from '@/utils/apiError'
import MonitorIQRing from './MonitorIQRing.vue'

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
const enabledKinds = ref<{ candy?: boolean; pelican?: boolean }>({})
const failures = ref<MonitorIQFailure[]>([])
const model = ref('')
// 测试分组：candy = 糖果测试；pelican = 鹈鹕测试（含自由创作结果）。
type IQKind = 'candy' | 'pelican'
const selectedIds = reactive<Record<IQKind, number | null>>({ candy: null, pelican: null })
const detail = ref<MonitorIQResult | null>(null)
const detailLoading = ref(false)
const previewing = ref(true)
const showSource = ref(false)
const detailCache = new Map<number, MonitorIQResult>()
let loadSeq = 0

const kindOf = (i: MonitorIQResult): IQKind => (i.benchmark === 'candy' ? 'candy' : 'pelican')

const models = computed(() =>
  Array.from(new Set([...items.value.map(i => i.model), ...failures.value.map(f => f.model)]))
)
// 按当前开启的测试决定显示哪些区块（关闭的测试不再显示旧结果）；
// 开启但还没有成功结果的测试也显示区块，以便展示失败原因。
// 旧后端没有返回开关时，退回「有数据才显示」。
const kinds = computed<IQKind[]>(() =>
  (['candy', 'pelican'] as IQKind[]).filter(k => {
    const flag = enabledKinds.value[k]
    return flag === undefined ? itemsOf(k).length > 0 || failuresOf(k).length > 0 : flag
  })
)

function itemsOf(kind: IQKind): MonitorIQResult[] {
  return items.value.filter(i => i.model === model.value && kindOf(i) === kind).slice(0, MAX_VISIBLE)
}
function failuresOf(kind: IQKind): MonitorIQFailure[] {
  return failures.value.filter(f => f.model === model.value && (f.benchmark === 'candy' ? 'candy' : 'pelican') === kind)
}
function selectedOf(kind: IQKind): MonitorIQResult | null {
  const list = itemsOf(kind)
  return list.find(i => i.id === selectedIds[kind]) ?? list[0] ?? null
}

const selectedByKind = computed<Record<IQKind, MonitorIQResult | null>>(() => ({
  candy: selectedOf('candy'),
  pelican: selectedOf('pelican'),
}))
const pelicanSelected = computed(() => selectedByKind.value.pelican)

// 是否通过：糖果=答对；鹈鹕/自由创作=判定为正常。
function isPass(item: MonitorIQResult): boolean {
  return item.benchmark === 'candy' ? item.verdict === 'passed' : item.quality === 'normal'
}
// 分段环数据：旧→新，最多 10 条。
function ringOf(kind: IQKind): boolean[] {
  return itemsOf(kind).map(isPass).reverse()
}
// 时间条：固定 10 格，旧→新自左向右，不足的在左侧补空位。
function slotsOf(kind: IQKind): Array<MonitorIQResult | null> {
  const list = [...itemsOf(kind)].reverse()
  return [...Array.from({ length: MAX_VISIBLE - list.length }, () => null), ...list]
}
function slotTitle(item: MonitorIQResult): string {
  return `${formatTime(item.started_at)} · ${itemLabel(item)} · ${formatDuration(item.duration_ms)}`
}
function statsOf(kind: IQKind) {
  const list = itemsOf(kind)
  const passed = list.filter(isPass).length
  return { total: list.length, passed, last: list[0]?.started_at ?? '' }
}

const previewDoc = computed(() => {
  const html = detail.value?.html ?? ''
  return `<!doctype html><meta http-equiv="Content-Security-Policy" content="${CSP}"><meta name="referrer" content="no-referrer">${html}`
})

function itemLabel(item: MonitorIQResult): string {
  if (item.benchmark === 'candy') {
    return item.verdict === 'passed' ? t('monitorIQ.correct') : t('monitorIQ.incorrect')
  }
  if (item.label) return item.label
  return item.quality && t(`monitorIQ.quality.${item.quality}`) !== `monitorIQ.quality.${item.quality}`
    ? t(`monitorIQ.quality.${item.quality}`)
    : item.quality || '-'
}

// 状态色：通过=绿，明确未通过=红，其余（无法判定等）=中性。
function itemTone(item: MonitorIQResult): 'is-pass' | 'is-fail' | 'is-neutral' {
  if (isPass(item)) return 'is-pass'
  if (item.benchmark === 'candy' || item.quality === 'degraded' || item.quality === 'suspicious') return 'is-fail'
  return 'is-neutral'
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
  failures.value = []
  detailCache.clear()
  try {
    const res = await channelMonitorUserAPI.iqResults(props.monitorId)
    if (seq !== loadSeq) return
    items.value = res.items ?? []
    failures.value = res.failures ?? []
    enabledKinds.value = { candy: res.candy_enabled, pelican: res.pelican_enabled }
  } catch (e: unknown) {
    if (seq !== loadSeq) return
    error.value = extractApiErrorMessage(e, t('monitorIQ.loadFailed'))
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

// 数据加载 / 切换模型时重置选中项，并保证选中的模型有效。
watch(items, () => {
  if (!models.value.includes(model.value)) model.value = models.value[0] ?? ''
})
watch(model, () => {
  selectedIds.candy = null
  selectedIds.pelican = null
})

// 鹈鹕/自由创作：选中后按需拉取含 html 的详情（带缓存）。
watch(
  pelicanSelected,
  async (item) => {
    detail.value = null
    showSource.value = false
    previewing.value = true
    if (!item || !item.has_html) return
    const cached = detailCache.get(item.id)
    if (cached) {
      detail.value = cached
      return
    }
    detailLoading.value = true
    try {
      const full = await channelMonitorUserAPI.iqResult(props.monitorId, item.id)
      detailCache.set(item.id, full)
      if (pelicanSelected.value?.id === item.id) detail.value = full
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

<style scoped>
/* 与渠道状态页同一套 rx 主题令牌（浮层 Teleport 到 body，靠 .rx-theme 取值） */
.iq-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.75rem;
  background: rgba(17, 17, 15, 0.3);
  -webkit-backdrop-filter: blur(6px) saturate(120%);
  backdrop-filter: blur(6px) saturate(120%);
}
:global(html.dark) .iq-overlay { background: rgba(0, 0, 0, 0.55); }

.iq-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  width: min(1240px, 100%);
  height: min(92dvh, 900px);
  border-radius: var(--rx-radius-lg, 1rem);
  color: var(--rx-ink, #0f172a);
  background: color-mix(in srgb, var(--rx-surface-hover, #fff) 96%, transparent);
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  box-shadow: var(--rx-highlight, 0 0 0 transparent), var(--rx-shadow-lg, 0 10px 40px rgba(0, 0, 0, 0.12));
  animation: iq-in 0.4s var(--rx-ease, ease) both;
}
@keyframes iq-in { from { opacity: 0; transform: translateY(10px) scale(0.99); } to { opacity: 1; transform: none; } }

/* ---------- 头部 ---------- */
.iq-head { display: flex; align-items: center; gap: 1rem; padding: 1.1rem 1.5rem; }
.iq-head__text { display: flex; min-width: 0; flex: 1; align-items: center; gap: 0.9rem; }
.iq-head__eyebrow {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.8rem;
  font-size: 0.8rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: var(--rx-accent, #0d9488);
  background: var(--rx-accent-soft, rgba(13, 148, 136, 0.14));
}
.iq-head__title {
  margin: 0;
  font-family: var(--rx-display, inherit);
  font-style: var(--rx-display-style, normal);
  font-weight: var(--rx-display-weight, 700);
  font-size: 1.15rem;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.iq-head__sub { margin: 0.15rem 0 0; font-size: 0.75rem; color: var(--rx-faint, #94a3b8); }
.iq-close {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 2rem;
  height: 2rem;
  border-radius: 999px;
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  color: var(--rx-ink-2, #334155);
  background: transparent;
  cursor: pointer;
  transition: background 0.2s ease, transform 0.3s var(--rx-spring, ease);
}
.iq-close:hover { background: var(--rx-surface-2, rgba(248, 250, 252, 0.7)); transform: rotate(90deg); }

/* ---------- 工具栏：模型切换 ---------- */
.iq-toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 1.5rem;
  border-top: 1px solid var(--rx-line-soft, rgba(15, 23, 42, 0.05));
  border-bottom: 1px solid var(--rx-line-soft, rgba(15, 23, 42, 0.05));
  background: var(--rx-surface-2, rgba(248, 250, 252, 0.7));
}
.iq-toolbar__label { flex-shrink: 0; font-size: 0.66rem; font-weight: 800; letter-spacing: 0.14em; color: var(--rx-faint, #94a3b8); }
.iq-models { display: flex; flex-wrap: wrap; gap: 0.4rem; min-width: 0; }
.iq-model {
  max-width: 100%;
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.74rem;
  color: var(--rx-ink-2, #334155);
  background: var(--rx-surface-hover, #fff);
  word-break: break-all;
  cursor: pointer;
  transition: all 0.2s ease;
}
.iq-model:hover { border-color: var(--rx-line-strong, rgba(15, 23, 42, 0.16)); }
.iq-model.is-active { border-color: var(--rx-accent, #0d9488); color: var(--rx-on-accent, #fff); background: var(--rx-accent, #0d9488); }

/* ---------- 主体：两张测试卡片并排 ---------- */
.iq-body { min-height: 0; flex: 1; overflow-y: auto; padding: 1.25rem 1.5rem 1.5rem; }
.iq-state { padding: 2.5rem 0; text-align: center; font-size: 0.85rem; color: var(--rx-faint, #94a3b8); }
.iq-state--error { color: #ef4444; }
.iq-state--inline { padding: 1.25rem 0; }

.iq-grid { display: grid; gap: 1.1rem; grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: start; }
.iq-grid.is-single { grid-template-columns: minmax(0, 1fr); }
@media (max-width: 899px) { .iq-grid { grid-template-columns: minmax(0, 1fr); } }

.iq-card {
  min-width: 0;
  padding: 1.1rem 1.2rem 1rem;
  border-radius: var(--rx-radius-md, 1rem);
  border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09));
  background: var(--rx-surface, rgba(255, 255, 255, 0.72));
  box-shadow: var(--rx-highlight, 0 0 0 transparent);
}
.iq-card__head { display: flex; align-items: baseline; gap: 0.6rem; }
.iq-card__idx { font-size: 0.72rem; font-weight: 800; letter-spacing: 0.14em; color: var(--rx-faint, #94a3b8); font-variant-numeric: tabular-nums; }
.iq-card__title { margin: 0; font-size: 1.02rem; font-weight: 700; }

/* 概览 */
.iq-hero { display: flex; align-items: center; gap: 1.1rem; margin-top: 0.9rem; }
.iq-hero__big { font-size: 2rem; font-weight: 800; line-height: 1; letter-spacing: -0.03em; font-variant-numeric: tabular-nums; }
.iq-hero__big small { margin-left: 0.3rem; font-size: 0.95rem; font-weight: 700; color: var(--rx-faint, #94a3b8); letter-spacing: 0; }
.iq-hero__label { margin-top: 0.3rem; font-size: 0.66rem; font-weight: 800; letter-spacing: 0.14em; color: var(--rx-faint, #94a3b8); }
.iq-hero__last { margin-top: 0.35rem; font-size: 0.72rem; color: var(--rx-muted, #64748b); font-variant-numeric: tabular-nums; }

/* 时间条 */
.iq-strip { display: grid; grid-template-columns: repeat(10, minmax(0, 1fr)); gap: 0.3rem; margin-top: 1.1rem; }
.iq-slot {
  height: 1.7rem;
  border: 0;
  border-radius: 0.4rem;
  padding: 0;
  background: currentColor;
  cursor: pointer;
  transition: transform 0.25s var(--rx-spring, ease), box-shadow 0.2s ease, opacity 0.2s ease;
}
.iq-slot.is-empty { cursor: default; background: var(--rx-line, #e5e7eb); }
.iq-slot:not(.is-empty):hover { transform: translateY(-2px); }
.iq-slot.is-active { box-shadow: 0 0 0 2px var(--rx-surface-hover, #fff), 0 0 0 4px currentColor; }
.iq-axis { display: flex; justify-content: space-between; margin-top: 0.4rem; font-size: 0.62rem; letter-spacing: 0.1em; color: var(--rx-faint, #94a3b8); }

.is-pass { color: hsl(140 60% 40%); }
.is-fail { color: #ef4444; }
.is-neutral { color: var(--rx-muted, #64748b); }

/* 选中那次的详情 */
.iq-run {
  margin-top: 1rem;
  padding: 0.9rem 1rem;
  border-radius: var(--rx-radius-sm, 0.75rem);
  background: var(--rx-surface-2, rgba(248, 250, 252, 0.7));
  border: 1px solid var(--rx-line-soft, rgba(15, 23, 42, 0.05));
}
.iq-run__head { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem 0.6rem; }
.iq-run__verdict { display: inline-flex; align-items: center; gap: 0.45rem; margin: 0; font-size: 1.05rem; font-weight: 800; }
.iq-run__dot { width: 0.5rem; height: 0.5rem; border-radius: 999px; background: currentColor; }
.iq-run__meta { margin-left: auto; font-size: 0.72rem; color: var(--rx-muted, #64748b); font-variant-numeric: tabular-nums; }
.iq-pill { padding: 0.05rem 0.55rem; border-radius: 999px; font-size: 0.66rem; font-weight: 700; color: var(--rx-accent, #0d9488); background: var(--rx-accent-soft, rgba(13, 148, 136, 0.14)); }

.iq-kv { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.6rem; margin: 0.8rem 0 0; }
.iq-kv > div { padding: 0.55rem 0.75rem; border-radius: 0.6rem; background: var(--rx-surface-hover, #fff); border: 1px solid var(--rx-line-soft, rgba(15, 23, 42, 0.05)); }
.iq-kv dt { font-size: 0.62rem; font-weight: 800; letter-spacing: 0.12em; color: var(--rx-faint, #94a3b8); }
.iq-kv dd { margin: 0.2rem 0 0; }
.iq-kv__answer { font-size: 1.3rem; font-weight: 800; font-variant-numeric: tabular-nums; word-break: break-word; white-space: pre-wrap; }
.iq-details { margin-top: 0.7rem; font-size: 0.76rem; color: var(--rx-ink-2, #334155); }
.iq-details summary { cursor: pointer; font-weight: 600; }
.iq-details pre { margin: 0.5rem 0 0; white-space: pre-wrap; word-break: break-word; font-family: inherit; font-size: 0.72rem; color: var(--rx-muted, #64748b); }

.iq-reason { margin: 0.75rem 0 0; font-size: 0.8rem; line-height: 1.6; color: var(--rx-ink-2, #334155); }
.iq-reason span { display: block; margin-bottom: 0.15rem; font-size: 0.62rem; font-weight: 800; letter-spacing: 0.12em; color: var(--rx-faint, #94a3b8); }
.iq-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-top: 0.75rem; }
.iq-btn {
  padding: 0.28rem 0.85rem;
  border-radius: 999px;
  border: 1px solid var(--rx-accent, #0d9488);
  font-size: 0.74rem;
  font-weight: 700;
  color: var(--rx-accent, #0d9488);
  background: transparent;
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;
}
.iq-btn:hover { background: var(--rx-accent, #0d9488); color: var(--rx-on-accent, #fff); }
.iq-btn--ghost { border-color: var(--rx-line-strong, rgba(15, 23, 42, 0.16)); color: var(--rx-ink-2, #334155); }
.iq-btn--ghost:hover { background: var(--rx-surface-hover, #fff); color: var(--rx-ink, #0f172a); }
.iq-frame { display: block; width: 100%; aspect-ratio: 16 / 10; margin-top: 0.75rem; border-radius: var(--rx-radius-sm, 0.75rem); border: 1px solid var(--rx-line, rgba(15, 23, 42, 0.09)); background: #fff; }
.iq-source { max-height: 18rem; margin: 0.75rem 0 0; padding: 0.8rem; overflow: auto; white-space: pre-wrap; word-break: break-word; border-radius: var(--rx-radius-sm, 0.75rem); font-size: 0.7rem; color: var(--rx-ink-2, #334155); background: var(--rx-surface-hover, #fff); }
.iq-note { margin: 0.8rem 0 0; font-size: 0.68rem; color: var(--rx-faint, #94a3b8); }

/* 失败原因 */
.iq-fails { display: grid; gap: 0.5rem; margin-top: 0.8rem; }
.iq-fail { padding: 0.6rem 0.8rem; border-radius: var(--rx-radius-sm, 0.75rem); border: 1px solid rgba(239, 68, 68, 0.28); background: rgba(239, 68, 68, 0.07); }
.iq-fail__meta { display: flex; flex-wrap: wrap; align-items: center; gap: 0.4rem 0.6rem; font-size: 0.7rem; color: var(--rx-ink-2, #334155); }
.iq-fail__badge { padding: 0 0.5rem; border-radius: 999px; font-size: 0.62rem; font-weight: 800; color: #fff; background: #ef4444; }
.iq-fail__msg { margin: 0.3rem 0 0; font-size: 0.74rem; word-break: break-word; color: #dc2626; }
:global(html.dark) .iq-fail__msg { color: #fca5a5; }

@media (max-width: 599px) {
  .iq-kv { grid-template-columns: minmax(0, 1fr); }
  .iq-run__meta { margin-left: 0; width: 100%; }
}
@media (prefers-reduced-motion: reduce) {
  .iq-panel { animation: none; }
  .iq-close:hover, .iq-slot:not(.is-empty):hover { transform: none; }
}
</style>
