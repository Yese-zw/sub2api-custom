<template>
  <AppLayout>
    <RxStage>
    <RxPageHero
      :eyebrow="t('profile.hero.eyebrow')"
      :lead="t('profile.hero.titleLead')"
      :accent="t('profile.hero.titleAccent')"
      :subtitle="t('profile.description')"
    />
    <div
      data-testid="profile-shell"
      class="rx-skin-rise space-y-6"
    >
      <ProfileInfoCard
        :user="user"
        :oidc-provider-name="oidcOAuthProviderName"
      />

      <div
        v-if="contactInfo"
        class="card flex items-center gap-4 p-6"
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

      <div
        data-testid="profile-settings"
        class="grid grid-cols-1 items-start gap-6 xl:grid-cols-2"
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
.rx-profile-contact-icon {
  border-radius: var(--rx-radius-sm);
  color: var(--rx-on-accent);
  background: var(--rx-accent);
}
</style>
