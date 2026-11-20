import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import HomeView from '../HomeView.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>'
}

function mountHome() {
  return mount(HomeView, { global: { stubs: { RouterLink: RouterLinkStub } } })
}

describe('HomeView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const app = useAppStore()
    app.publicSettingsLoaded = true
    vi.spyOn(app, 'fetchPublicSettings').mockResolvedValue(null)
  })

  it('renders the built-in landing page and ignores homepage settings', () => {
    const wrapper = mountHome()

    expect(wrapper.get('#ls-home').text()).toContain('一个接口，接入')
    expect(wrapper.get('.ls-brand-title').text()).toBe('灵思智域')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.html()).not.toContain('data:image')
    expect(wrapper.get('img').attributes('src')).toBeTruthy()
  })

  it('sends anonymous visitors to login/register instead of the admin console', () => {
    const wrapper = mountHome()

    expect(wrapper.find('a[href="/admin/dashboard"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/login"]').exists()).toBe(true)
  })

  it('sends admins to the admin dashboard and users to the user dashboard', async () => {
    const auth = useAuthStore()
    auth.token = 't'
    ;(auth as any).user = { role: 'admin' }
    const adminView = mountHome()
    expect(adminView.get('a[href="/admin/dashboard"]').text()).toContain('进入控制台')

    ;(auth as any).user = { role: 'user' }
    const userView = mountHome()
    expect(userView.find('a[href="/admin/dashboard"]').exists()).toBe(false)
    expect(userView.get('a[href="/dashboard"]').text()).toContain('进入控制台')
  })
})
