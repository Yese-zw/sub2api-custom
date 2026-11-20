import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountCacheRateCell from '../AccountCacheRateCell.vue'
import type { AccountCacheRate24h } from '@/api/admin/accounts'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const makeData = (overrides: Partial<AccountCacheRate24h> = {}): AccountCacheRate24h => ({
  rate: 70,
  requests: 4,
  input_tokens: 20,
  cache_creation_tokens: 10,
  cache_read_tokens: 70,
  ...overrides
})

const mountCell = (props: Record<string, unknown>) =>
  mount(AccountCacheRateCell, {
    props,
    global: { stubs: { Teleport: true } }
  })

describe('AccountCacheRateCell', () => {
  it('没有请求时显示 -', () => {
    const wrapper = mountCell({ data: makeData({ rate: null, requests: 0 }) })
    expect(wrapper.get('[data-testid="account-cache-rate-empty"]').text()).toBe('-')
    expect(wrapper.find('[data-testid="account-cache-rate-value"]').exists()).toBe(false)
  })

  it('没有数据时显示 -', () => {
    const wrapper = mountCell({ data: null })
    expect(wrapper.get('[data-testid="account-cache-rate-empty"]').text()).toBe('-')
  })

  it('首次加载显示占位', () => {
    const wrapper = mountCell({ data: null, loading: true })
    expect(wrapper.find('[data-testid="account-cache-rate-loading"]').exists()).toBe(true)
  })

  it('加载失败且没有旧数据时显示错误', () => {
    const wrapper = mountCell({ data: null, error: 'Failed' })
    expect(wrapper.text()).toContain('Failed')
  })

  it.each([
    [70, 'text-emerald-600'],
    [99.5, 'text-emerald-600'],
    [69.99, 'text-amber-600'],
    [30, 'text-amber-600'],
    [29.99, 'text-red-600'],
    [0, 'text-red-600']
  ])('缓存率 %s%% 使用 %s', (rate, cls) => {
    const wrapper = mountCell({ data: makeData({ rate }) })
    const value = wrapper.get('[data-testid="account-cache-rate-value"]')
    expect(value.text()).toBe(`${rate.toFixed(2)}%`)
    expect(value.classes()).toContain(cls)
  })

  it('悬停提示包含请求数和各类 token 数', () => {
    const wrapper = mountCell({
      data: makeData({ requests: 12, input_tokens: 345, cache_creation_tokens: 678, cache_read_tokens: 901 })
    })
    const tooltip = wrapper.get('[data-testid="account-cache-rate-tooltip"]').text()
    expect(tooltip).toContain('admin.accounts.cacheRate24h.formula')
    expect(tooltip).toContain('admin.accounts.cacheRate24h.requests')
    expect(tooltip).toContain('12')
    expect(tooltip).toContain('345')
    expect(tooltip).toContain('678')
    expect(tooltip).toContain('901')
  })
})
