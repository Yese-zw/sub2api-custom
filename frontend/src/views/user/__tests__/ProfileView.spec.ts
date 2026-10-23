import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileView from '@/views/user/ProfileView.vue'

const {
  fetchPublicSettingsMock,
  refreshUserMock,
  authState
} = vi.hoisted(() => ({
  fetchPublicSettingsMock: vi.fn(),
  refreshUserMock: vi.fn(),
  authState: {
    user: null as Record<string, unknown> | null,
    refreshUser: vi.fn()
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    fetchPublicSettings: fetchPublicSettingsMock
  })
}))

vi.mock('@/utils/format', () => ({
  formatDate: () => 'April 2026'
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('ProfileView', () => {
  beforeEach(() => {
    refreshUserMock.mockReset()
    fetchPublicSettingsMock.mockReset()
    refreshUserMock.mockResolvedValue(undefined)
    authState.refreshUser = refreshUserMock
    authState.user = {
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      role: 'user',
      balance: 10,
      concurrency: 2,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: true,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      created_at: '2026-04-20T00:00:00Z',
      updated_at: '2026-04-20T00:00:00Z'
    }
    fetchPublicSettingsMock.mockResolvedValue({
      contact_info: '',
      balance_low_notify_enabled: true,
      balance_low_notify_threshold: 0,
      linuxdo_oauth_enabled: true,
      wechat_oauth_enabled: true,
      wechat_oauth_open_enabled: true,
      wechat_oauth_mp_enabled: false,
      oidc_oauth_enabled: true,
      oidc_oauth_provider_name: 'OIDC',
      app_name: 'LSAGENT',
      app_default_group_id: 3
    })
  })

  it('places login bindings below password, app group, balance alerts, and two-factor auth', async () => {
    const wrapper = mount(ProfileView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          StatCard: { template: '<div class="stat-card" />' },
          ProfileInfoCard: { template: '<div data-testid="profile-info-card" />' },
          ProfileAppGroupCard: { template: '<div data-testid="profile-app-group" />' },
          ProfileBalanceNotifyCard: { template: '<div data-testid="profile-balance-notify-card" />' },
          ProfilePasswordForm: { template: '<div data-testid="profile-password-form" />' },
          ProfileTotpCard: { template: '<div data-testid="profile-totp-card" />' },
          ProfileAccountBindingsCard: { template: '<div data-testid="profile-auth-bindings" />' },
          ProfilePasskeyCard: { template: '<div data-testid="profile-passkey" />' },
          Icon: true
        }
      }
    })

    await flushPromises()

    const html = wrapper.get('[data-testid="profile-shell"]').html()
    const order = [
      'profile-info-card',
      'profile-password-form',
      'profile-app-group',
      'profile-balance-notify-card',
      'profile-totp-card',
      'profile-auth-bindings',
      'profile-passkey'
    ].map((id) => html.indexOf(id))

    expect(wrapper.findAll('.stat-card')).toHaveLength(0)
    expect(wrapper.get('[data-testid="profile-info-card"]').find('[data-testid="profile-auth-bindings"]').exists()).toBe(false)
    expect(order.every((index) => index >= 0)).toBe(true)
    expect(order).toEqual([...order].sort((a, b) => a - b))
  })
})
