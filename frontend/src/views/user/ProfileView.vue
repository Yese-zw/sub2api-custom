<template>
  <AppLayout>
    <RxStage>
    <RxPageHero
      :eyebrow="t('profile.hero.eyebrow')"
      :lead="t('profile.hero.titleLead')"
      :accent="t('profile.hero.titleAccent')"
      :subtitle="t('profile.description')"
    />
    <div class="profile-layout">
      <ProfileIdentityRail :user="user" :sections="sections" />

      <div
        data-testid="profile-shell"
        class="min-w-0 space-y-10"
      >
        <section id="profile-sec-basics" class="profile-sec">
          <header class="profile-sec__head">
            <span class="profile-sec__index">01</span>
            <h2>{{ sections[0].label }}</h2>
          </header>
          <ProfileInfoCard
            :user="user"
            :oidc-provider-name="oidcOAuthProviderName"
            hide-overview
          />
          <div
            v-if="contactInfo"
            class="card mt-6 flex items-center gap-4 p-6"
          >
            <div class="rx-profile-contact-icon shrink-0 p-3">
              <Icon name="chat" size="lg" />
            </div>
            <div class="min-w-0">
              <h3 class="font-semibold text-gray-900 dark:text-white">
                {{ t('common.contactSupport') }}
              </h3>
              <p class="truncate text-sm font-medium text-gray-600 dark:text-gray-300">{{ contactInfo }}</p>
            </div>
          </div>
        </section>

        <section id="profile-sec-security" class="profile-sec">
          <header class="profile-sec__head">
            <span class="profile-sec__index">02</span>
            <h2>{{ sections[1].label }}</h2>
          </header>
          <div
            data-testid="profile-settings"
            class="rx-skin-rise grid grid-cols-1 items-start gap-6 xl:grid-cols-2"
          >
            <ProfilePasswordForm />
            <ProfileAppGroupCard
              v-if="user && appDefaultGroupId > 0"
              :app-name="appName"
              :default-group-id="appDefaultGroupId"
            />
            <ProfileBalanceNotifyCard
              v-if="user && balanceLowNotifyEnabled"
              :enabled="user.balance_notify_enabled ?? true"
              :threshold="user.balance_notify_threshold"
              :extra-emails="user.balance_notify_extra_emails ?? []"
              :system-default-threshold="systemDefaultThreshold"
              :user-email="user.email"
            />
            <ProfileTotpCard />
          </div>
        </section>

        <section id="profile-sec-login" class="profile-sec">
          <header class="profile-sec__head">
            <span class="profile-sec__index">03</span>
            <h2>{{ sections[2].label }}</h2>
          </header>
          <div class="rx-skin-rise space-y-6">
            <ProfileAccountBindingsCard
              :user="user"
              :linuxdo-enabled="linuxdoOAuthEnabled"
              :dingtalk-enabled="dingtalkOAuthEnabled"
              :oidc-enabled="oidcOAuthEnabled"
              :oidc-provider-name="oidcOAuthProviderName"
              :wechat-enabled="wechatOAuthEnabled"
              :wechat-open-enabled="wechatOAuthOpenEnabled"
              :wechat-mp-enabled="wechatOAuthMPEnabled"
            />

            <ProfilePasskeyCard :enabled="passkeyEnabled" />
          </div>
        </section>
      </div>
    </div>
    </RxStage>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import AppLayout from '@/components/layout/AppLayout.vue'
import RxStage from '@/components/common/RxStage.vue'
import RxPageHero from '@/components/common/RxPageHero.vue'
import ProfileAccountBindingsCard from '@/components/user/profile/ProfileAccountBindingsCard.vue'
import ProfileAppGroupCard from '@/components/user/profile/ProfileAppGroupCard.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfileIdentityRail from '@/components/user/profile/ProfileIdentityRail.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import ProfilePasskeyCard from '@/components/user/profile/ProfilePasskeyCard.vue'
import { isWeChatWebOAuthEnabled } from '@/api/auth'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const user = computed(() => authStore.user)

// 右侧三个分区，同时驱动左栏导航
const sections = computed(() => [
  { id: 'profile-sec-basics', label: t('profile.hero.secBasics') },
  { id: 'profile-sec-security', label: t('profile.hero.secSecurity') },
  { id: 'profile-sec-login', label: t('profile.hero.secLogin') },
])

const contactInfo = ref('')
const balanceLowNotifyEnabled = ref(false)
const systemDefaultThreshold = ref(0)
const linuxdoOAuthEnabled = ref(false)
const dingtalkOAuthEnabled = ref(false)
const wechatOAuthEnabled = ref(false)
const wechatOAuthOpenEnabled = ref<boolean | undefined>(undefined)
const wechatOAuthMPEnabled = ref<boolean | undefined>(undefined)
const oidcOAuthEnabled = ref(false)
const oidcOAuthProviderName = ref('OIDC')
const passkeyEnabled = ref(false)
const appName = ref('')
const appDefaultGroupId = ref(0)

onMounted(async () => {
  const profileRefresh = authStore.refreshUser().catch((error) => {
    console.error('Failed to refresh profile:', error)
  })

  const settingsLoad = appStore.fetchPublicSettings()
    .then((settings) => {
      if (!settings) {
        return
      }
      contactInfo.value = settings.contact_info || ''
      balanceLowNotifyEnabled.value = settings.balance_low_notify_enabled ?? false
      systemDefaultThreshold.value = settings.balance_low_notify_threshold ?? 0
      linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled ?? false
      dingtalkOAuthEnabled.value = settings.dingtalk_oauth_enabled ?? false
      wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
      wechatOAuthOpenEnabled.value = typeof settings.wechat_oauth_open_enabled === 'boolean'
        ? settings.wechat_oauth_open_enabled
        : undefined
      wechatOAuthMPEnabled.value = typeof settings.wechat_oauth_mp_enabled === 'boolean'
        ? settings.wechat_oauth_mp_enabled
        : undefined
      oidcOAuthEnabled.value = settings.oidc_oauth_enabled ?? false
      oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
      passkeyEnabled.value = settings.passkey_enabled === true
      appName.value = settings.app_name || ''
      appDefaultGroupId.value = settings.app_default_group_id ?? 0
    })
    .catch((error) => {
      console.error('Failed to load settings:', error)
    })

  await Promise.all([profileRefresh, settingsLoad])
})
</script>

<style scoped>
.profile-layout {
  display: grid;
  gap: 2rem;
  align-items: start;
}

@media (min-width: 1024px) {
  .profile-layout {
    grid-template-columns: 19rem minmax(0, 1fr);
  }
}

.profile-sec {
  scroll-margin-top: 5.5rem;
}

.profile-sec__head {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
  margin: 0 0 1rem 0.25rem;
}

.profile-sec__index {
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 1.05rem;
  color: var(--rx-faint);
}

.profile-sec__head h2 {
  font-size: 1.15rem;
  font-weight: 750;
  letter-spacing: -0.02em;
  color: var(--rx-ink);
}

.rx-profile-contact-icon {
  border-radius: var(--rx-radius-sm);
  color: var(--rx-on-accent);
  background: var(--rx-accent);
}
</style>
