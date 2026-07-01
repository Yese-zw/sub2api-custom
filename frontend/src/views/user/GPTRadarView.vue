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
import AppLayout from '@/components/layout/AppLayout.vue'

const RADAR_PAGE_URL = 'https://codexradar.com/%E9%9B%B7%E8%BE%BE%E9%9B%B7%E8%BE%BE'
const RADAR_ORIGIN = 'https://codexradar.com'
const REFRESH_INTERVAL_MS = 5 * 60 * 1000

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

:host .model-iq {
  margin: 0;
}
`

const radarHost = ref<HTMLElement | null>(null)
const loading = ref(true)
const errorMessage = ref('')

let shadowRoot: ShadowRoot | null = null
let refreshTimer: number | undefined
let cleanupCallbacks: Array<() => void> = []

async function loadRadar() {
  loading.value = true
  errorMessage.value = ''

  try {
    const response = await fetch(RADAR_PAGE_URL, {
      cache: 'no-store',
      referrerPolicy: 'strict-origin-when-cross-origin',
    })
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const html = await response.text()
    const parser = new DOMParser()
    const doc = parser.parseFromString(html, 'text/html')
    const section = doc.querySelector<HTMLElement>('section.model-iq')

    if (!section) {
      throw new Error('model-iq section missing')
    }

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
    renderRadar(styleText, sanitizedSection)
  } catch (error) {
    console.error('Failed to load GPT radar:', error)
    errorMessage.value = '降智雷达加载失败，请稍后重试。'
  } finally {
    loading.value = false
  }
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

function renderRadar(styleText: string, html: string) {
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

  initModelIqInteractions(shadowRoot)
}

function initModelIqInteractions(root: ShadowRoot) {
  const sections = Array.from(root.querySelectorAll<HTMLElement>('.model-iq-score'))

  for (const section of sections) {
    const metricSelect = section.querySelector<HTMLSelectElement>('[data-model-iq-chart-metric]')
    const onMetricChange = () => setMetric(section, metricSelect?.value || 'iq')
    metricSelect?.addEventListener('change', onMetricChange)
    if (metricSelect) {
      cleanupCallbacks.push(() => metricSelect.removeEventListener('change', onMetricChange))
    }
    setMetric(section, metricSelect?.value || 'iq')

    const onClick = (event: MouseEvent) => {
      if (event.target instanceof Element && event.target.closest('[data-model-iq-chart-metric]')) return
      const target = event.target instanceof Element ? event.target.closest<HTMLElement>('[data-model-key]') : null
      if (!target || !section.contains(target)) {
        if (section.querySelector<HTMLElement>('.model-iq-chart')?.dataset.selectedModels) {
          setSelection(section, [])
        }
        return
      }
      toggleSelection(section, target.dataset.modelKey || '')
    }

    const onKeydown = (event: KeyboardEvent) => {
      if (event.target instanceof Element && event.target.closest('[data-model-iq-chart-metric]')) return
      if (event.key === 'Escape') {
        setSelection(section, [])
        return
      }
      if (event.key !== 'Enter' && event.key !== ' ') return

      const target = event.target instanceof Element ? event.target.closest<HTMLElement>('[data-model-key]') : null
      if (!target || !section.contains(target)) return
      event.preventDefault()
      toggleSelection(section, target.dataset.modelKey || '')
    }

    section.addEventListener('click', onClick)
    section.addEventListener('keydown', onKeydown)
    cleanupCallbacks.push(() => {
      section.removeEventListener('click', onClick)
      section.removeEventListener('keydown', onKeydown)
    })
  }
}

function selectedKeys(section: HTMLElement): Set<string> {
  const raw = section.querySelector<HTMLElement>('.model-iq-chart')?.dataset.selectedModels || ''
  return new Set(raw.split(',').map((key) => key.trim()).filter(Boolean))
}

function setSelection(section: HTMLElement, selected: Iterable<string>) {
  const selectedSet = new Set(Array.from(selected).filter(Boolean))
  const chart = section.querySelector<HTMLElement>('.model-iq-chart')
  if (chart) {
    if (selectedSet.size > 0) {
      chart.dataset.selectedModels = Array.from(selectedSet).join(',')
    } else {
      delete chart.dataset.selectedModels
    }
  }

  for (const target of section.querySelectorAll<HTMLElement>('[data-model-key]')) {
    const matches = selectedSet.has(target.dataset.modelKey || '')
    target.classList.toggle('is-selected', matches)
    target.classList.toggle('is-muted', selectedSet.size > 0 && !matches)
    if (target.getAttribute('role') === 'button') {
      target.setAttribute('aria-pressed', matches ? 'true' : 'false')
    }
  }
}

function toggleSelection(section: HTMLElement, key: string) {
  if (!key) return
  const keys = selectedKeys(section)
  if (keys.has(key)) {
    keys.delete(key)
  } else {
    keys.add(key)
  }
  setSelection(section, keys)
}

function setMetric(section: HTMLElement, metric: string) {
  const chart = section.querySelector<HTMLElement>('.model-iq-chart')
  if (!chart) return

  chart.dataset.selectedMetric = metric
  for (const view of chart.querySelectorAll<HTMLElement>('[data-model-iq-chart-view]')) {
    view.hidden = view.dataset.modelIqChartView !== metric
  }
}

function cleanupRadar() {
  for (const callback of cleanupCallbacks) {
    callback()
  }
  cleanupCallbacks = []
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
