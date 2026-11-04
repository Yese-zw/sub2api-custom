import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import MonitorIQDialog from '../monitor/MonitorIQDialog.vue'

const { iqResults } = vi.hoisted(() => ({ iqResults: vi.fn() }))

vi.mock('@/api/channelMonitor', () => ({
  channelMonitorUserAPI: { iqResults, iqResult: vi.fn() },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, te: () => true }) }
})

const candyItem = {
  id: 1, monitor_id: 9, model: 'gpt-x', benchmark: 'candy', verdict: 'passed', answer: '21',
  has_html: false, duration_ms: 1000,
  started_at: '2026-11-02T10:00:00Z', finished_at: '2026-11-02T10:00:01Z',
}
const pelicanItem = {
  id: 2, monitor_id: 9, model: 'gpt-x', benchmark: 'pelican', quality: 'normal', label: 'ok',
  has_html: false, duration_ms: 2000,
  started_at: '2026-11-02T10:01:00Z', finished_at: '2026-11-02T10:01:02Z',
}

async function mountDialog(res: Record<string, unknown>) {
  iqResults.mockResolvedValue({ interval_minutes: 20, max_per_model: 10, ...res })
  const wrapper = mount(MonitorIQDialog, {
    props: { show: true, monitorId: 9, monitorName: 'm' },
    global: { stubs: { Teleport: true } },
  })
  await flushPromises()
  return wrapper
}

const headings = (w: Awaited<ReturnType<typeof mountDialog>>) =>
  w.findAll('h3').map(h => h.text()).filter(t => t.startsWith('monitorIQ.tabs.'))

describe('MonitorIQDialog sections', () => {
  beforeEach(() => iqResults.mockReset())

  it('shows both candy and pelican when both are enabled and both have results', async () => {
    const w = await mountDialog({ candy_enabled: true, pelican_enabled: true, items: [candyItem, pelicanItem] })
    expect(headings(w)).toEqual(['monitorIQ.tabs.candy', 'monitorIQ.tabs.pelican'])
  })

  it('still shows the pelican section with its failure reason when only candy has results', async () => {
    const w = await mountDialog({
      candy_enabled: true, pelican_enabled: true, items: [candyItem],
      failures: [{ model: 'gpt-x', benchmark: 'pelican', message: 'manxue.ai HTTP 400', at: '2026-11-02T10:02:00Z' }],
    })
    expect(headings(w)).toEqual(['monitorIQ.tabs.candy', 'monitorIQ.tabs.pelican'])
    expect(w.text()).toContain('manxue.ai HTTP 400')
  })

  it('hides a disabled test even when old results remain', async () => {
    const w = await mountDialog({ candy_enabled: false, pelican_enabled: true, items: [candyItem, pelicanItem] })
    expect(headings(w)).toEqual(['monitorIQ.tabs.pelican'])
  })
})
