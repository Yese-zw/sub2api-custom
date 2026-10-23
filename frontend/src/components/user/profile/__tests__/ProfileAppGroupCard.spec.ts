import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import ProfileAppGroupCard from '../ProfileAppGroupCard.vue'

const { getAppGroup, updateAppGroup, getAvailable, showSuccess, showError } = vi.hoisted(() => ({
  getAppGroup: vi.fn(),
  updateAppGroup: vi.fn(),
  getAvailable: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api', () => ({
  userAPI: { getAppGroup, updateAppGroup },
  userGroupsAPI: { getAvailable },
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params ? `${key}:${Object.values(params).join(',')}` : key,
  }),
}))
// Render the options as plain buttons so the test drives selection directly.
vi.mock('@/components/common/Select.vue', () => ({
  default: defineComponent({
    props: { modelValue: { type: Number, default: 0 }, options: { type: Array, default: () => [] } },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('div', (props.options as Array<{ value: number; label: string }>).map((option) =>
          h('button', {
            'data-value': option.value,
            'data-selected': option.value === props.modelValue,
            onClick: () => emit('update:modelValue', option.value),
          }, option.label)))
    },
  }),
}))

enableAutoUnmount(afterEach)

const mountCard = () =>
  mount(ProfileAppGroupCard, { props: { appName: 'Example Agent', defaultGroupId: 110 } })

describe('ProfileAppGroupCard', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    getAvailable.mockResolvedValue([
      { id: 110, name: 'Default', platform: 'openai' },
      { id: 7, name: 'Fast', platform: 'anthropic' },
    ])
    getAppGroup.mockResolvedValue(0)
  })

  it('offers "follow default" naming the default group plus the available groups', async () => {
    const wrapper = mountCard()
    await flushPromises()

    const labels = wrapper.findAll('button').map((item) => item.text())
    expect(labels).toEqual([
      'profile.appGroup.followDefault:Default',
      'Default · openai',
      'Fast · anthropic',
    ])
    expect(wrapper.get('[data-value="0"]').attributes('data-selected')).toBe('true')
    expect(wrapper.text()).toContain('profile.appGroup.title:Example Agent')
  })

  it('saves the selection and reverts it when the server rejects it', async () => {
    updateAppGroup.mockResolvedValueOnce(7).mockRejectedValueOnce(new Error('APP_GROUP_NOT_ALLOWED'))
    const wrapper = mountCard()
    await flushPromises()

    await wrapper.get('[data-value="7"]').trigger('click')
    await flushPromises()
    expect(updateAppGroup).toHaveBeenLastCalledWith(7)
    expect(showSuccess).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-value="7"]').attributes('data-selected')).toBe('true')

    await wrapper.get('[data-value="110"]').trigger('click')
    await flushPromises()
    expect(showError).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-value="7"]').attributes('data-selected')).toBe('true')
  })
})
