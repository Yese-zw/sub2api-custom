<template>
  <AppLayout>
    <div class="gpt-radar-page">
      <div v-if="loading" class="gpt-radar-state">降智雷达加载中...</div>
      <div v-else-if="errorMessage" class="gpt-radar-state gpt-radar-state-error">
        <span>{{ errorMessage }}</span>
        <button type="button" @click="loadRadar">刷新</button>
      </div>
      <div v-show="!loading && !errorMessage" ref="radarHost" class="gpt-radar-host"></div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import DOMPurify from 'dompurify'
import { Chart as ChartJS, CategoryScale, LineController, LineElement, LinearScale, LogarithmicScale, PointElement, ScatterController, Tooltip } from 'chart.js'
import AppLayout from '@/components/layout/AppLayout.vue'

ChartJS.register(CategoryScale, LineController, LineElement, LinearScale, LogarithmicScale, PointElement, ScatterController, Tooltip)

const RADAR_PAGE_URL = 'https://codexradar.com/%E9%9B%B7%E8%BE%BE%E9%9B%B7%E8%BE%BE'
const RADAR_ORIGIN = 'https://codexradar.com'
const RADAR_SECTION_SELECTOR = 'section.intelligence-efficiency'
const RADAR_DATA_URL = `${RADAR_ORIGIN}/data/intelligence-efficiency.json`
const REFRESH_INTERVAL_MS = 5 * 60 * 1000

const MODEL_METADATA: Record<string, { label: string; color: string }> = {
  'gpt-5.6-sol': { label: 'Sol', color: '#eab308' },
  'gpt-5.6-terra': { label: 'Terra', color: '#3b82f6' },
  'gpt-5.6-luna': { label: 'Luna', color: '#c7d2e0' },
  'gpt-5.5': { label: 'GPT-5.5', color: '#00e5ff' },
}

const EFFORT_ORDER = ['ultra', 'max', 'xhigh', 'high', 'medium', 'low']

interface RadarPoint {
  model: string
  effort: string
  iq: number
  average_price_usd: number
  average_minutes: number
  combined_cost_index?: number
  passed?: number
  valid_tasks?: number
  total_runs?: number
  average_agent_steps?: number | null
  agent_steps_samples?: number
  average_total_tokens?: number | null
  token_samples?: number
  cache_hit_rate?: number | null
  cache_token_samples?: number
  price_samples?: number
  duration_samples?: number
  latest_graded_at?: string
}

interface RadarPayload {
  source_updated_at?: string
  points?: RadarPoint[]
  history?: Array<{ at: string; points: RadarPoint[] }>
}

const SHADOW_OVERRIDES = `
:host {
  display: block;
  width: min(1260px, 100%);
  margin: 0 auto;
  color-scheme: light;
  --bg: #f3f6f9;
  --panel: #ffffff;
  --panel-soft: #f8fafc;
  --ink: #17202b;
  --muted: #667085;
  --soft: #8a94a6;
  --line: #dce3ed;
  --line-strong: #cbd5e1;
  --green: #13865d;
  --green-dark: #0f684e;
  --green-soft: #e2f4ec;
  --amber: #a86600;
  --amber-soft: #fff0d0;
  --blue: #2d65c8;
  --blue-soft: #e7efff;
  --red: #c0392b;
  --red-soft: #ffe8e5;
  --shadow: 0 18px 44px rgba(31, 43, 59, 0.08);
  --shadow-soft: 0 10px 28px rgba(31, 43, 59, 0.06);
}

:host .intelligence-efficiency {
  margin: 0;
}

:host .intelligence-efficiency-plot-scroll {
  overflow-x: auto;
}

:host .intelligence-efficiency-svg {
  min-width: 720px;
}

:host .intelligence-efficiency-grid:empty {
  display: none;
}

:host .radar-chart-figure,
:host .radar-history-card,
:host .radar-pk-panel {
  margin: 0;
  border: 1px solid var(--line, #dce3ed);
  border-radius: 8px;
  background: var(--panel, #fff);
}

:host .radar-chart-figure {
  padding: 16px;
}

:host .radar-chart-head,
:host .radar-history-head,
:host .radar-pk-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  color: var(--ink, #17202b);
  font-size: 14px;
}

:host .radar-chart-head strong,
:host .radar-history-head strong,
:host .radar-pk-head h3 {
  margin: 0;
  font-size: 16px;
}

:host .radar-chart-head select,
:host .radar-pk-head select {
  min-height: 32px;
  border: 1px solid var(--line-strong, #cbd5e1);
  border-radius: 5px;
  background: var(--panel, #fff);
  color: inherit;
  padding: 0 9px;
}

:host .radar-canvas-wrap {
  position: relative;
  min-height: 360px;
}

:host .radar-history-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

:host .radar-history-card {
  padding: 12px;
}

:host .radar-history-card .radar-canvas-wrap {
  min-height: 240px;
}

:host .radar-pk-panel {
  margin-top: 16px;
  padding: 16px;
}

:host .radar-pk-options {
  display: grid;
  gap: 8px;
}

:host .radar-pk-family {
  display: grid;
  grid-template-columns: 72px repeat(6, minmax(0, 1fr));
  gap: 6px;
  align-items: center;
}

:host .radar-pk-option {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 5px;
  min-height: 30px;
  padding: 0 7px;
  border: 1px solid color-mix(in srgb, var(--family-color) 48%, var(--line, #dce3ed));
  border-radius: 5px;
  color: var(--ink, #17202b);
  cursor: pointer;
  font-size: 12px;
}

:host .radar-pk-option input {
  margin: 0;
  accent-color: var(--family-color);
}

:host .radar-pk-value {
  margin-left: auto;
  color: var(--family-color);
  font-weight: 800;
}

:host .radar-pk-empty {
  margin: 14px 0 0;
  color: var(--muted, #667085);
  font-size: 13px;
}

:host .radar-detail-backdrop {
  position: fixed;
  z-index: 1000;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgba(15, 23, 42, .52);
}

:host .radar-detail-panel {
  width: min(540px, 100%);
  max-height: min(700px, calc(100vh - 40px));
  overflow: auto;
  border: 1px solid var(--line-strong, #cbd5e1);
  border-radius: 8px;
  background: var(--panel, #fff);
  box-shadow: var(--shadow, 0 18px 44px rgba(31, 43, 59, .18));
  color: var(--ink, #17202b);
}

:host .radar-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 18px 20px 12px;
  border-bottom: 1px solid var(--line, #dce3ed);
}

:host .radar-detail-head h3 {
  margin: 0;
  font-size: 17px;
}

:host .radar-detail-close {
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 23px;
  line-height: 1;
}

:host .radar-detail-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
  margin: 0;
  padding: 12px 20px;
}

:host .radar-detail-summary > div {
  min-width: 0;
  padding: 10px 0;
  border-bottom: 1px solid var(--line, #dce3ed);
}

:host .radar-detail-summary dt {
  color: var(--muted, #667085);
  font-size: 12px;
}

:host .radar-detail-summary dd {
  margin: 5px 0 0;
  font-size: 14px;
  font-weight: 750;
}

:host .radar-detail-foot {
  margin: 0;
  padding: 0 20px 20px;
  color: var(--muted, #667085);
  font-size: 12px;
}

@media (max-width: 760px) {
  :host .radar-history-grid { grid-template-columns: 1fr; }
  :host .radar-pk-family { grid-template-columns: 64px repeat(3, minmax(0, 1fr)); }
  :host .radar-canvas-wrap { min-height: 280px; }
}
`

const radarHost = ref<HTMLElement | null>(null)
const loading = ref(true)
const errorMessage = ref('')

let shadowRoot: ShadowRoot | null = null
let refreshTimer: number | undefined
let chartInstances: ChartJS[] = []
let cleanupCallbacks: Array<() => void> = []

async function loadRadar() {
  loading.value = true
  errorMessage.value = ''

  try {
    const [pageResponse, dataResponse] = await Promise.all([
      fetch(RADAR_PAGE_URL, {
        cache: 'no-store',
        referrerPolicy: 'strict-origin-when-cross-origin',
      }),
      fetch(RADAR_DATA_URL, { cache: 'no-store' }),
    ])
    if (!pageResponse.ok || !dataResponse.ok) {
      throw new Error(`HTTP ${pageResponse.status} / ${dataResponse.status}`)
    }

    const [html, payload] = await Promise.all([
      pageResponse.text(),
      dataResponse.json() as Promise<RadarPayload>,
    ])
    const points = validRadarPoints(payload.points)
    if (points.length === 0) {
      throw new Error('intelligence-efficiency points missing')
    }

    const parser = new DOMParser()
    const doc = parser.parseFromString(html, 'text/html')
    const section = doc.querySelector<HTMLElement>(RADAR_SECTION_SELECTOR)

    if (!section) {
      throw new Error('intelligence-efficiency section missing')
    }

    section.querySelector('.intelligence-efficiency-callout')?.remove()
    absolutizeSectionUrls(section)
    pruneRadarSection(section)
    const sanitizedSection = DOMPurify.sanitize(section.outerHTML, {
      USE_PROFILES: { html: true, svg: true, svgFilters: true },
      ADD_ATTR: ['style', 'target', 'rel'],
      FORBID_TAGS: ['script', 'iframe', 'object', 'embed'],
    })
    const styleText = Array.from(doc.querySelectorAll('style'))
      .map((style) => style.textContent || '')
      .join('\n')

    await nextTick()
    renderRadar(styleText, sanitizedSection, payload, points)
  } catch (error) {
    console.error('Failed to load GPT radar:', error)
    errorMessage.value = '降智雷达加载失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}

function validRadarPoints(points: RadarPayload['points']): RadarPoint[] {
  if (!Array.isArray(points)) return []

  return points.filter((point): point is RadarPoint => (
    typeof point?.model === 'string'
    && typeof point.effort === 'string'
    && typeof point.iq === 'number'
    && typeof point.average_price_usd === 'number'
    && typeof point.average_minutes === 'number'
    && point.model in MODEL_METADATA
  ))
}

function absolutizeSectionUrls(section: HTMLElement) {
  for (const element of section.querySelectorAll<HTMLElement>('[src]')) {
    const src = element.getAttribute('src')
    if (src) {
      element.setAttribute('src', new URL(src, RADAR_ORIGIN).toString())
    }
  }

  for (const element of section.querySelectorAll<HTMLElement>('[href]')) {
    const href = element.getAttribute('href')
    if (href && !href.startsWith('#')) {
      element.setAttribute('href', new URL(href, RADAR_ORIGIN).toString())
    }
  }
}

function pruneRadarSection(section: HTMLElement) {
  const removableSelectors = [
    '.model-iq-actions',
    '.model-iq-copy-status',
    '.model-iq-reference-note',
    '.model-ratings',
    '[data-model-ratings]',
  ]

  for (const element of section.querySelectorAll(removableSelectors.join(','))) {
    element.remove()
  }
}

function renderRadar(styleText: string, html: string, payload: RadarPayload, points: RadarPoint[]) {
  const host = radarHost.value
  if (!host) return

  cleanupRadar()
  shadowRoot = host.shadowRoot || host.attachShadow({ mode: 'open' })
  shadowRoot.innerHTML = ''

  const styleElement = document.createElement('style')
  styleElement.textContent = `${styleText}\n${SHADOW_OVERRIDES}`
  shadowRoot.appendChild(styleElement)

  const container = document.createElement('div')
  container.innerHTML = html
  shadowRoot.appendChild(container)
  populateRadarData(shadowRoot, payload, points)
}

function populateRadarData(root: ShadowRoot, payload: RadarPayload, points: RadarPoint[]) {
  const cards = root.querySelector<HTMLElement>('[data-intelligence-efficiency-cards]')
  if (!cards) return

  cards.replaceChildren()
  for (const [model, metadata] of Object.entries(MODEL_METADATA)) {
    const modelPoints = points
      .filter((point) => point.model === model)
      .sort((left, right) => EFFORT_ORDER.indexOf(left.effort) - EFFORT_ORDER.indexOf(right.effort))
    if (modelPoints.length === 0) continue

    const row = document.createElement('div')
    row.className = 'intelligence-efficiency-family-row'
    row.dataset.efficiencyFamily = model

    for (const point of modelPoints) {
      const card = document.createElement('button')
      card.className = 'intelligence-efficiency-card'
      card.type = 'button'
      card.dataset.efficiencyCard = ''
      card.dataset.model = point.model
      card.dataset.effort = point.effort
      card.style.setProperty('--family-color', metadata.color)
      card.style.setProperty('--effort-column', String(EFFORT_ORDER.indexOf(point.effort) + 1))
      card.setAttribute('aria-label', `${metadata.label} ${point.effort} IQ ${formatNumber(point.iq)} · 详细指标`)

      const score = document.createElement('span')
      score.className = 'intelligence-efficiency-card-iq'
      const label = document.createElement('span')
      label.className = 'intelligence-efficiency-card-label'
      label.textContent = `${metadata.label} ${point.effort}`
      const iq = document.createElement('strong')
      iq.textContent = formatNumber(point.iq)
      score.append(label, iq)

      const details = document.createElement('span')
      details.className = 'intelligence-efficiency-card-meta'
      const price = document.createElement('span')
      price.textContent = `$${formatNumber(point.average_price_usd)}`
      const minutes = document.createElement('span')
      minutes.textContent = `${formatNumber(point.average_minutes)}分钟`
      details.append(price, minutes)
      card.append(score, details)
      row.appendChild(card)
    }

    cards.appendChild(row)
  }

  const updated = root.querySelector<HTMLElement>('[data-intelligence-efficiency-updated]')
  if (updated && payload.source_updated_at) {
    const date = new Date(payload.source_updated_at)
    if (!Number.isNaN(date.getTime())) {
      const value = new Intl.DateTimeFormat('zh-CN', {
        timeZone: 'Asia/Shanghai',
        month: 'numeric',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }).format(date)
      updated.textContent = `每 10 分钟刷新 · ${value} 更新`
    }
  }

  renderEfficiencyCharts(root, payload, points)
  renderHistoryComparison(root, payload, points)

  const onCardClick = (event: MouseEvent) => {
    const card = event.target instanceof Element ? event.target.closest<HTMLElement>('[data-efficiency-card]') : null
    if (!card) return
    const point = points.find((item) => item.model === card.dataset.model && item.effort === card.dataset.effort)
    if (point) openRadarDetail(root, point)
  }
  cards.addEventListener('click', onCardClick)
  cleanupCallbacks.push(() => cards.removeEventListener('click', onCardClick))
}

function formatNumber(value: number): string {
  return value.toFixed(1).replace(/\.0$/, '')
}

function renderEfficiencyCharts(root: ShadowRoot, payload: RadarPayload, points: RadarPoint[]) {
  const charts = root.querySelector<HTMLElement>('[data-intelligence-efficiency-charts]')
  if (!charts) return

  charts.replaceChildren()
  const figure = document.createElement('figure')
  figure.className = 'radar-chart-figure'
  const head = document.createElement('figcaption')
  head.className = 'radar-chart-head'
  const title = document.createElement('strong')
  const select = document.createElement('select')
  const metrics = [
    { value: 'combined_cost_index', label: '综合成本 × IQ' },
    { value: 'average_minutes', label: '时间成本 × IQ' },
    { value: 'average_price_usd', label: '费用成本 × IQ' },
  ]
  for (const metric of metrics) {
    const option = document.createElement('option')
    option.value = metric.value
    option.textContent = metric.label
    select.appendChild(option)
  }
  head.append(title, select)
  const canvasWrap = document.createElement('div')
  canvasWrap.className = 'radar-canvas-wrap'
  figure.append(head, canvasWrap)
  charts.appendChild(figure)

  let chart: ChartJS | null = null
  const draw = () => {
    chart?.destroy()
    const metric = metrics.find((item) => item.value === select.value) || metrics[0]
    title.textContent = metric.label
    const canvas = document.createElement('canvas')
    canvasWrap.replaceChildren(canvas)
    const datasets = Object.entries(MODEL_METADATA).map(([model, metadata]) => ({
      label: metadata.label,
      showLine: true,
      borderColor: metadata.color,
      backgroundColor: metadata.color,
      borderWidth: 1.8,
      pointRadius: 4,
      pointHoverRadius: 6,
      data: points
        .filter((point) => point.model === model)
        .map((point) => ({
          x: Math.max(Number(point[metric.value as keyof RadarPoint]) || 0.0001, 0.0001),
          y: point.iq,
          label: `${metadata.label} ${point.effort}`,
        })),
    }))
    chart = new ChartJS(canvas, {
      type: 'scatter',
      data: { datasets },
      options: radarChartOptions({
        xTitle: metric.label.replace(' × IQ', ''),
        yTitle: 'IQ',
        logarithmic: true,
        tooltipLabel: (context) => `${context.raw.label}: IQ ${formatNumber(context.parsed.y)} · ${metric.label.replace(' × IQ', '')} ${formatNumber(context.parsed.x)}`,
      }),
    })
    chartInstances.push(chart)
  }
  select.addEventListener('change', draw)
  cleanupCallbacks.push(() => {
    select.removeEventListener('change', draw)
    chart?.destroy()
  })
  draw()

  const history = validHistory(payload.history)
  if (history.length === 0) return

  const historyFigure = document.createElement('figure')
  historyFigure.className = 'radar-chart-figure'
  const historyHead = document.createElement('figcaption')
  historyHead.className = 'radar-chart-head'
  const historyTitle = document.createElement('strong')
  historyTitle.textContent = '近 48 小时 IQ 历史趋势'
  const historyCaption = document.createElement('span')
  historyCaption.textContent = '每 4 小时观察点 · 每 10 分钟刷新'
  historyHead.append(historyTitle, historyCaption)
  const historyGrid = document.createElement('div')
  historyGrid.className = 'radar-history-grid'
  historyFigure.append(historyHead, historyGrid)
  charts.appendChild(historyFigure)

  for (const [model, metadata] of Object.entries(MODEL_METADATA)) {
    const available = points.some((point) => point.model === model)
    if (!available) continue
    const card = document.createElement('section')
    card.className = 'radar-history-card'
    card.style.setProperty('--family-color', metadata.color)
    const cardHead = document.createElement('div')
    cardHead.className = 'radar-history-head'
    const cardTitle = document.createElement('strong')
    cardTitle.textContent = metadata.label
    const range = document.createElement('span')
    const values = points.filter((point) => point.model === model).map((point) => point.iq)
    range.textContent = `${formatNumber(Math.min(...values))}-${formatNumber(Math.max(...values))}`
    cardHead.append(cardTitle, range)
    const wrap = document.createElement('div')
    wrap.className = 'radar-canvas-wrap'
    const canvas = document.createElement('canvas')
    wrap.appendChild(canvas)
    card.append(cardHead, wrap)
    historyGrid.appendChild(card)
    chartInstances.push(createHistoryChart(canvas, history, model, metadata.color, 'iq', 'IQ'))
  }
}

function renderHistoryComparison(root: ShadowRoot, payload: RadarPayload, points: RadarPoint[]) {
  const panel = root.querySelector<HTMLElement>('[data-efficiency-pk]')
  if (!panel) return
  const history = validHistory(payload.history)
  if (history.length === 0) {
    panel.remove()
    return
  }

  panel.replaceChildren()
  panel.classList.add('radar-pk-panel')
  const head = document.createElement('header')
  head.className = 'radar-pk-head'
  const title = document.createElement('h3')
  title.textContent = '历史数据比较'
  const select = document.createElement('select')
  const metrics = [
    { value: 'iq', label: 'IQ', field: 'iq' as const },
    { value: 'price', label: '费用', field: 'average_price_usd' as const },
    { value: 'minutes', label: '耗时', field: 'average_minutes' as const },
    { value: 'steps', label: 'Agent steps', field: 'average_agent_steps' as const },
    { value: 'cache', label: 'cache 命中率', field: 'cache_hit_rate' as const },
    { value: 'tokens', label: '总 tokens', field: 'average_total_tokens' as const },
  ]
  for (const metric of metrics) {
    const option = document.createElement('option')
    option.value = metric.value
    option.textContent = metric.label
    select.appendChild(option)
  }
  const hint = document.createElement('span')
  hint.textContent = '选择一个或多个模型档位，对比近 48 小时历史。'
  head.append(title, select, hint)
  const options = document.createElement('div')
  options.className = 'radar-pk-options'
  const chartWrap = document.createElement('div')
  panel.append(head, options, chartWrap)

  const selected = new Set<string>()
  for (const [model, metadata] of Object.entries(MODEL_METADATA)) {
    const family = document.createElement('div')
    family.className = 'radar-pk-family'
    const modelName = document.createElement('strong')
    modelName.textContent = metadata.label
    modelName.style.color = metadata.color
    family.appendChild(modelName)
    for (const effort of EFFORT_ORDER) {
      const point = points.find((item) => item.model === model && item.effort === effort)
      if (!point) continue
      const label = document.createElement('label')
      label.className = 'radar-pk-option'
      label.style.setProperty('--family-color', metadata.color)
      const input = document.createElement('input')
      input.type = 'checkbox'
      input.value = `${model}|${effort}`
      const text = document.createElement('span')
      text.textContent = effort
      const value = document.createElement('span')
      value.className = 'radar-pk-value'
      value.textContent = formatNumber(point.iq)
      label.append(input, text, value)
      family.appendChild(label)
      input.addEventListener('change', () => {
        if (input.checked) selected.add(input.value)
        else selected.delete(input.value)
        drawComparison()
      })
      cleanupCallbacks.push(() => input.replaceWith(input.cloneNode(true)))
    }
    options.appendChild(family)
  }

  let chart: ChartJS | null = null
  const drawComparison = () => {
    chart?.destroy()
    chartWrap.replaceChildren()
    if (selected.size === 0) {
      const empty = document.createElement('p')
      empty.className = 'radar-pk-empty'
      empty.textContent = '暂未选择模型档位。'
      chartWrap.appendChild(empty)
      return
    }
    const metric = metrics.find((item) => item.value === select.value) || metrics[0]
    const wrap = document.createElement('div')
    wrap.className = 'radar-canvas-wrap'
    const canvas = document.createElement('canvas')
    wrap.appendChild(canvas)
    chartWrap.appendChild(wrap)
    chart = createHistoryChart(canvas, history, '', '', metric.field, metric.label, selected)
    chartInstances.push(chart)
  }
  select.addEventListener('change', drawComparison)
  cleanupCallbacks.push(() => {
    select.removeEventListener('change', drawComparison)
    chart?.destroy()
  })
  drawComparison()
}

function createHistoryChart(
  canvas: HTMLCanvasElement,
  history: Array<{ at: string; points: RadarPoint[] }>,
  model: string,
  color: string,
  metric: keyof RadarPoint,
  label: string,
  selected?: Set<string>,
) {
  const keys = selected
    ? Array.from(selected)
    : EFFORT_ORDER.map((effort) => `${model}|${effort}`)
  const datasets = keys.map((key) => {
    const [entryModel, effort] = key.split('|')
    const metadata = MODEL_METADATA[entryModel]
    const pointColor = color || metadata?.color || '#2d65c8'
    return {
      label: `${metadata?.label || entryModel} ${effort}`,
      borderColor: pointColor,
      backgroundColor: pointColor,
      borderWidth: 1.7,
      pointRadius: 2.5,
      pointHoverRadius: 4,
      tension: 0.18,
      data: history.map((snapshot) => {
        const point = snapshot.points.find((item) => item.model === entryModel && item.effort === effort)
        const value = point?.[metric]
        return typeof value === 'number' ? value : null
      }),
    }
  })
  return new ChartJS(canvas, {
    type: 'line',
    data: {
      labels: history.map((snapshot) => formatHistoryTime(snapshot.at)),
      datasets,
    },
    options: radarChartOptions({ xTitle: '', yTitle: label }),
  })
}

function radarChartOptions({ xTitle, yTitle, logarithmic = false, tooltipLabel }: {
  xTitle: string
  yTitle: string
  logarithmic?: boolean
  tooltipLabel?: (context: any) => string
}) {
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'nearest' as const, intersect: false },
    plugins: {
      legend: { position: 'top' as const, labels: { boxWidth: 10, usePointStyle: true } },
      tooltip: { callbacks: tooltipLabel ? { label: tooltipLabel } : undefined },
    },
    scales: {
      x: {
        type: logarithmic ? 'logarithmic' as const : 'category' as const,
        title: { display: Boolean(xTitle), text: xTitle },
        grid: { color: 'rgba(148, 163, 184, .2)' },
      },
      y: {
        title: { display: Boolean(yTitle), text: yTitle },
        grid: { color: 'rgba(148, 163, 184, .2)' },
      },
    },
  }
}

function validHistory(history: RadarPayload['history']): Array<{ at: string; points: RadarPoint[] }> {
  if (!Array.isArray(history)) return []
  return history
    .map((snapshot) => ({ at: snapshot?.at, points: validRadarPoints(snapshot?.points) }))
    .filter((snapshot): snapshot is { at: string; points: RadarPoint[] } => typeof snapshot.at === 'string' && snapshot.points.length > 0)
}

function formatHistoryTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
  }).format(date)
}

function openRadarDetail(root: ShadowRoot, point: RadarPoint) {
  root.querySelector<HTMLElement>('.radar-detail-backdrop')?.remove()
  const metadata = MODEL_METADATA[point.model]
  if (!metadata) return

  const backdrop = document.createElement('div')
  backdrop.className = 'radar-detail-backdrop'
  const panel = document.createElement('section')
  panel.className = 'radar-detail-panel'
  panel.setAttribute('role', 'dialog')
  panel.setAttribute('aria-modal', 'true')
  panel.setAttribute('aria-label', `${metadata.label} ${point.effort} · 详细指标`)
  const head = document.createElement('header')
  head.className = 'radar-detail-head'
  const title = document.createElement('h3')
  title.textContent = `${metadata.label} ${point.effort} · 详细指标`
  const close = document.createElement('button')
  close.className = 'radar-detail-close'
  close.type = 'button'
  close.setAttribute('aria-label', '关闭')
  close.textContent = '×'
  head.append(title, close)
  const summary = document.createElement('dl')
  summary.className = 'radar-detail-summary'
  const successRate = point.valid_tasks ? `${formatNumber((100 * (point.passed || 0)) / point.valid_tasks)}%` : '—'
  const items: Array<[string, string]> = [
    ['IQ', formatNumber(point.iq)],
    ['通过率', successRate],
    ['通过数 / 有效题数', `${point.passed ?? '—'}/${point.valid_tasks ?? '—'}`],
    ['已记录运行', point.total_runs ? `${point.total_runs} 次` : '—'],
    ['平均 Agent steps', sampleValue(point.average_agent_steps, point.agent_steps_samples)],
    ['平均价格', `$${formatNumber(point.average_price_usd)}`],
    ['cache 命中率', point.cache_hit_rate == null ? '—' : sampleValue(point.cache_hit_rate * 100, point.cache_token_samples, '%')],
    ['平均耗时', `${formatNumber(point.average_minutes)} 分钟`],
    ['平均总 tokens', point.average_total_tokens == null ? '—' : sampleValue(point.average_total_tokens / 10000, point.token_samples, '万')],
    ['费用样本', point.price_samples ? `${point.price_samples} 次` : '—'],
    ['耗时样本', point.duration_samples ? `${point.duration_samples} 次` : '—'],
    ['最近判分', point.latest_graded_at ? formatHistoryTime(point.latest_graded_at) : '—'],
  ]
  for (const [label, value] of items) {
    const entry = document.createElement('div')
    const term = document.createElement('dt')
    term.textContent = label
    const definition = document.createElement('dd')
    definition.textContent = value
    entry.append(term, definition)
    summary.appendChild(entry)
  }
  const foot = document.createElement('p')
  foot.className = 'radar-detail-foot'
  foot.textContent = 'IQ 与平均值均按每题最新有效结果统计。'
  panel.append(head, summary, foot)
  backdrop.appendChild(panel)
  root.appendChild(backdrop)
  const dismiss = () => backdrop.remove()
  close.addEventListener('click', dismiss)
  backdrop.addEventListener('click', (event) => {
    if (event.target === backdrop) dismiss()
  })
}

function sampleValue(value: number | null | undefined, samples?: number, suffix = ''): string {
  if (value == null) return '—'
  return `${formatNumber(value)}${suffix}${samples ? ` · ${samples} 个样本` : ''}`
}

function cleanupRadar() {
  for (const callback of cleanupCallbacks) callback()
  cleanupCallbacks = []
  for (const chart of chartInstances) chart.destroy()
  chartInstances = []
}

onMounted(() => {
  loadRadar()
  refreshTimer = window.setInterval(loadRadar, REFRESH_INTERVAL_MS)
})

onBeforeUnmount(() => {
  cleanupRadar()
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.gpt-radar-page {
  width: 100%;
}

.gpt-radar-host {
  width: 100%;
}

.gpt-radar-state {
  display: flex;
  min-height: 240px;
  align-items: center;
  justify-content: center;
  gap: 12px;
  border: 1px solid rgba(203, 213, 225, 0.86);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.92);
  color: #667085;
  font-size: 14px;
  font-weight: 720;
}

.gpt-radar-state-error {
  color: #c0392b;
}

.gpt-radar-state button {
  border: 1px solid rgba(45, 101, 200, 0.22);
  border-radius: 8px;
  background: rgba(45, 101, 200, 0.08);
  color: #2d65c8;
  min-height: 34px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 820;
}
</style>
