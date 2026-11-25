import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import RequestAuditDetailModal from '../RequestAuditDetailModal.vue'

const { getByUsageLog } = vi.hoisted(() => ({ getByUsageLog: vi.fn() }))
vi.mock('@/api/admin/requestAudit', () => ({ requestAuditAPI: { getByUsageLog } }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
beforeEach(() => { vi.clearAllMocks() })

const stubs = { BaseDialog: { template: '<div><slot/></div>' } }

const record = {
  id: 1, created_at: '2026-11-25T10:00:00Z', request_id: 'r1', client_request_id: 'c1',
  method: 'POST', path: '/v1/messages', model: 'claude-x', stream: true, status_code: 200, duration_ms: 1500,
  client_ip: '1.2.3.4', user_agent: 'ua', request_headers: { 'content-type': 'application/json' },
  request_content_type: 'application/json',
  request_body: JSON.stringify({ system: 'sys prompt', messages: [{ role: 'user', content: 'what is 2+2' }] }),
  request_body_size: 80, request_truncated: false, response_content_type: 'text/event-stream',
  response_body: 'data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"4"}}\n\n',
  response_body_size: 90, response_truncated: false
}

describe('RequestAuditDetailModal', () => {
  it('loads by usage log id and renders parsed input/output', async () => {
    getByUsageLog.mockResolvedValueOnce(record)
    const wrapper = mount(RequestAuditDetailModal, { props: { show: true, usageLogId: 42 }, global: { stubs } })
    await flushPromises()
    expect(getByUsageLog).toHaveBeenCalledWith(42)
    const text = wrapper.text()
    expect(text).toContain('sys prompt')
    expect(text).toContain('what is 2+2')
    expect(text).toContain('assistant')
    expect(wrapper.findAll('pre').some((p) => p.text() === '4')).toBe(true)

    const rawTab = wrapper.findAll('button').find((b) => b.text() === 'admin.requestAudit.tabs.response')!
    await rawTab.trigger('click')
    expect(wrapper.text()).toContain('text_delta')
  })

  it('shows not-found hint on 404', async () => {
    getByUsageLog.mockRejectedValueOnce({ status: 404, code: 'REQUEST_AUDIT_NOT_FOUND' })
    const wrapper = mount(RequestAuditDetailModal, { props: { show: true, usageLogId: 7 }, global: { stubs } })
    await flushPromises()
    expect(wrapper.find('[data-testid="request-audit-error"]').text()).toBe('admin.requestAudit.notFound')
  })
})
