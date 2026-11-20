import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

describe('HomeView', () => {
  it('renders the built-in landing page and ignores homepage settings', () => {
    const wrapper = mount(HomeView)

    expect(wrapper.get('#ls-home').text()).toContain('一个接口，接入')
    expect(wrapper.get('.ls-brand-title').text()).toBe('灵思智域')
    expect(wrapper.get('a[href="/admin/dashboard"]').text()).toContain('进入控制台')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.html()).not.toContain('data:image')
    expect(wrapper.get('img').attributes('src')).toBeTruthy()
  })
})
