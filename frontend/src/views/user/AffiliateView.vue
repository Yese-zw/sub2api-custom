<template>
  <AppLayout>
    <RxStage>
      <div v-if="loading" class="rx-aff-loading" aria-busy="true">
        <span v-for="n in 4" :key="n"></span>
      </div>

      <template v-else-if="detail">
        <div class="rx-invite-card">
          <span class="rx-invite-card__sheen" aria-hidden="true"></span>
          <div class="rx-invite-card__rate-block">
            <p class="rx-invite-card__label">{{ t('affiliate.stats.rebateRate') }}</p>
            <p class="rx-invite-card__rate">{{ formattedRebateRate }}<small>%</small></p>
            <p class="rx-invite-card__hint">{{ t('affiliate.stats.rebateRateHint') }}</p>
          </div>
          <div class="rx-invite-card__foot">
            <span class="rx-invite-card__avatars" aria-hidden="true">
              <span v-for="item in inviteeFaces" :key="item.user_id">{{ initialOf(item) }}</span>
              <span v-if="inviteeFaces.length === 0" class="is-empty">+</span>
            </span>
            <span>{{ t('affiliate.stats.invitedUsers') }} · <b>{{ formatCount(detail.aff_count) }}</b></span>
          </div>
        </div>

        <div class="grid gap-4 sm:grid-cols-3">
          <div class="rx-glass rx-metric rx-rise" style="--rx-i: 3">
            <span class="rx-metric__index">01</span>
            <p class="rx-metric__label">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="rx-metric__value">{{ formatCount(detail.aff_count) }}</p>
          </div>
          <div class="rx-glass rx-metric rx-rise" style="--rx-i: 4">
            <span class="rx-metric__index">02</span>
            <p class="rx-metric__label">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="rx-metric__value rx-aff-good">{{ formatCurrency(detail.aff_quota) }}</p>
          </div>
          <div class="rx-glass rx-metric rx-rise" style="--rx-i: 5">
            <span class="rx-metric__index">03</span>
            <p class="rx-metric__label">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="rx-metric__value">{{ formatCurrency(detail.aff_history_quota) }}</p>
            <p v-if="detail.aff_frozen_quota > 0" class="rx-metric__hint rx-aff-warn">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="grid items-start gap-6 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,0.7fr)]">
          <section class="rx-glass rx-aff-panel rx-rise" style="--rx-i: 6">
            <header class="rx-section-head">
              <span class="rx-section-head__index">01</span>
              <h2 class="rx-section-head__title">{{ t('affiliate.title') }}</h2>
            </header>

            <div class="space-y-4">
              <div class="space-y-2">
                <p class="rx-aff-field-label">{{ t('affiliate.yourCode') }}</p>
                <div class="rx-aff-ticket" :class="{ 'is-stamped': codeStamped }">
                  <div class="rx-aff-ticket__stub" aria-hidden="true">
                    <Icon name="gift" size="md" />
                    <span>{{ t('affiliate.hero.ticketStub') }}</span>
                  </div>
                  <div class="rx-aff-ticket__body flex flex-col items-stretch gap-2 sm:flex-row sm:items-center">
                    <code class="rx-aff-code min-w-0 break-all sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                    <span class="rx-aff-ticket__scan" aria-hidden="true"></span>
                    <button class="rx-aff-copy w-full sm:w-auto sm:shrink-0" @click="copyCode">
                      <Icon name="copy" size="sm" />
                      <span>{{ t('affiliate.copyCode') }}</span>
                    </button>
                  </div>
                  <Transition name="rx-stamp">
                    <span v-if="codeStamped" class="rx-aff-stamp" aria-hidden="true">{{ t('affiliate.hero.stamped') }}</span>
                  </Transition>
                </div>
              </div>

              <div class="space-y-2">
                <p class="rx-aff-field-label">{{ t('affiliate.inviteLink') }}</p>
                <div class="rx-aff-field flex flex-col items-stretch gap-2 sm:flex-row sm:items-center">
                  <code class="rx-aff-link min-w-0 break-all sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                  <button class="rx-aff-copy w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('affiliate.copyLink') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </section>

          <aside class="rx-glass rx-aff-panel rx-rise" style="--rx-i: 7">
            <header class="rx-section-head">
              <span class="rx-section-head__index">02</span>
              <h2 class="rx-section-head__title">{{ t('affiliate.tips.title') }}</h2>
            </header>
            <ol class="rx-aff-tips">
              <li>{{ t('affiliate.tips.line1') }}</li>
              <li>{{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>{{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">{{ t('affiliate.tips.line4') }}</li>
            </ol>
          </aside>
        </div>

        <!-- 转入余额：可用额度大号展示 + 墨色按钮 -->
        <section class="rx-glass rx-aff-transfer rx-rise" style="--rx-i: 8">
          <div class="min-w-0">
            <header class="rx-section-head !mb-1">
              <span class="rx-section-head__index">03</span>
              <h2 class="rx-section-head__title">{{ t('affiliate.transfer.title') }}</h2>
            </header>
            <p class="rx-aff-muted">{{ t('affiliate.transfer.description') }}</p>
            <p v-if="detail.aff_quota <= 0" class="rx-aff-muted rx-aff-warn mt-2">{{ t('affiliate.transfer.empty') }}</p>
          </div>
          <div class="rx-aff-transfer__action">
            <p class="rx-aff-transfer__amount">{{ formatCurrency(detail.aff_quota) }}</p>
            <button
              class="rx-aff-cta"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
        </section>

        <section class="rx-glass rx-aff-panel rx-rise" style="--rx-i: 9">
          <header class="rx-section-head">
            <span class="rx-section-head__index">04</span>
            <h2 class="rx-section-head__title">{{ t('affiliate.invitees.title') }}</h2>
            <span class="rx-section-head__hint">{{ formatCount(detail.invitees.length) }}</span>
          </header>
          <div v-if="detail.invitees.length === 0" class="rx-aff-empty">
            <span class="rx-aff-empty__orbit" aria-hidden="true"><i></i><i></i><i></i></span>
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="rx-aff-table w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr>
                  <th>{{ t('affiliate.invitees.columns.email') }}</th>
                  <th>{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th>{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in detail.invitees" :key="item.user_id">
                  <td>
                    <span class="rx-aff-person">
                      <span class="rx-aff-person__avatar" aria-hidden="true">{{ initialOf(item) }}</span>
                      <span class="truncate">{{ item.email || '-' }}</span>
                    </span>
                  </td>
                  <td class="rx-aff-muted">{{ item.username || '-' }}</td>
                  <td class="text-right font-semibold rx-aff-good">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="rx-aff-muted tabular-nums">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </RxStage>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import RxStage from '@/components/common/RxStage.vue'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const codeStamped = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

// 邀请卡上最多展示 5 位被邀请人的首字母头像
const inviteeFaces = computed(() => (detail.value?.invitees ?? []).slice(0, 5))

function initialOf(item: { email?: string; username?: string }): string {
  const source = (item.username || item.email || '?').trim()
  return (source[0] || '?').toUpperCase()
}

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
  codeStamped.value = true
  window.setTimeout(() => { codeStamped.value = false }, 1600)
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>

<style scoped>
.rx-aff-loading {
  display: grid;
  gap: 1rem;
  padding-top: 1.5rem;
}

.rx-aff-loading span {
  height: 7rem;
  border-radius: var(--rx-radius-lg);
  background: linear-gradient(100deg, var(--rx-surface-2) 30%, var(--rx-surface-hover) 50%, var(--rx-surface-2) 70%);
  background-size: 200% 100%;
  animation: rx-aff-shimmer 1.5s ease-in-out infinite;
}

@keyframes rx-aff-shimmer {
  to { background-position: -200% 0; }
}

/* ---------- 邀请卡 ---------- */
.rx-invite-card {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  justify-content: space-between;
  gap: 1rem 2rem;
  width: 100%;
  padding: 1.2rem 1.4rem 1.15rem;
  border-radius: var(--rx-radius-lg);
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12), inset 0 0 0 1px var(--rx-card-line), var(--rx-shadow-lg);
}

.rx-invite-card__sheen {
  pointer-events: none;
  position: absolute;
  inset: -40%;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, var(--rx-foil) 60deg, transparent 120deg, transparent 220deg, var(--rx-foil) 280deg, transparent 340deg);
  mix-blend-mode: soft-light;
  animation: rx-aff-spin 18s linear infinite;
}

@keyframes rx-aff-spin {
  to { --rx-angle: 360deg; }
}

.rx-invite-card__rate-block,
.rx-invite-card__label,
.rx-invite-card__rate,
.rx-invite-card__hint,
.rx-invite-card__foot {
  position: relative;
}

.rx-invite-card__label {
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  opacity: 0.6;
}

.rx-invite-card__rate {
  margin-top: 0.3rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: clamp(2.2rem, 1.6rem + 2vw, 3.2rem);
  font-weight: 500;
  line-height: 1;
  letter-spacing: -0.04em;
}

.rx-invite-card__rate small {
  margin-left: 0.15rem;
  font-size: 0.45em;
  opacity: 0.7;
}

.rx-invite-card__hint {
  position: relative;
  margin-top: 0.4rem;
  font-size: 0.75rem;
  opacity: 0.6;
}

.rx-invite-card__foot {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  margin-top: 0;
  padding-top: 0;
  border-top: 0;
  font-size: 0.78rem;
  opacity: 0.85;
}

.rx-invite-card__foot b {
  font-weight: 800;
}

.rx-invite-card__avatars {
  display: flex;
}

.rx-invite-card__avatars span {
  display: grid;
  place-items: center;
  width: 1.65rem;
  height: 1.65rem;
  margin-left: -0.45rem;
  border-radius: 9999px;
  font-size: 0.66rem;
  font-weight: 800;
  color: #11110f;
  background: var(--rx-card-ink);
  box-shadow: 0 0 0 2px #1a1a17;
  transition: transform 0.35s var(--rx-spring);
}

.rx-invite-card__avatars span:first-child {
  margin-left: 0;
}

.rx-invite-card__avatars span.is-empty {
  color: var(--rx-card-ink);
  background: transparent;
  box-shadow: inset 0 0 0 1px var(--rx-card-line);
}

.rx-invite-card:hover .rx-invite-card__avatars span:nth-child(2) { transform: translateX(3px); }
.rx-invite-card:hover .rx-invite-card__avatars span:nth-child(3) { transform: translateX(6px); }
.rx-invite-card:hover .rx-invite-card__avatars span:nth-child(4) { transform: translateX(9px); }
.rx-invite-card:hover .rx-invite-card__avatars span:nth-child(5) { transform: translateX(12px); }

/* ---------- 面板与字段 ---------- */
.rx-aff-panel {
  padding: 1.4rem;
}

@media (min-width: 640px) {
  .rx-aff-panel {
    padding: 1.75rem;
  }
}

.rx-aff-good {
  color: #15803d;
}

html.dark .rx-aff-good {
  color: #4ade80;
}

.rx-aff-warn {
  color: var(--rx-warn);
}

.rx-aff-muted {
  font-size: 0.85rem;
  color: var(--rx-muted);
}

.rx-aff-field-label {
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--rx-faint);
}

.rx-aff-ticket {
  --notch: 10px;
  position: relative;
  display: flex;
  min-height: 5.1rem;
  overflow: hidden;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-hover);
  border: 1px solid var(--rx-line);
  box-shadow: var(--rx-highlight), var(--rx-shadow);
}

.rx-aff-ticket__stub {
  display: flex;
  width: 4.6rem;
  flex-shrink: 0;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.3rem;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  font-size: 0.58rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.rx-aff-ticket__body {
  position: relative;
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0 0.9rem;
}

.rx-aff-ticket__scan {
  pointer-events: none;
  position: absolute;
  inset: 0;
  background: linear-gradient(100deg, transparent 30%, var(--rx-accent-soft) 50%, transparent 70%);
  transform: translateX(-100%);
}

.rx-aff-ticket:hover .rx-aff-ticket__scan {
  animation: rx-aff-scan 1.4s var(--rx-ease) 1;
}

@keyframes rx-aff-scan {
  to { transform: translateX(100%); }
}

.rx-aff-stamp {
  position: absolute;
  right: 5.5rem;
  top: 50%;
  padding: 0.15rem 0.45rem;
  border: 2px solid color-mix(in srgb, var(--rx-hot) 70%, white);
  border-radius: 6px;
  color: color-mix(in srgb, var(--rx-hot) 80%, white);
  font-size: 0.7rem;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  transform: translateY(-50%) rotate(-12deg);
}

.rx-stamp-enter-active { animation: rx-aff-stamp-in 0.45s var(--rx-spring); }
.rx-stamp-leave-active { transition: opacity 0.25s ease; }
.rx-stamp-leave-to { opacity: 0; }

@keyframes rx-aff-stamp-in {
  from { transform: translateY(-50%) rotate(-12deg) scale(1.6); opacity: 0; }
}

.rx-aff-field {
  padding: 0.55rem 0.55rem 0.55rem 1rem;
  border-radius: var(--rx-radius-md);
  background: var(--rx-surface-2);
  border: 1px dashed var(--rx-line-strong);
  transition: border-color 0.3s ease, background 0.3s ease;
}

.rx-aff-field:hover {
  border-style: solid;
  background: var(--rx-surface-hover);
}

.rx-aff-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  color: var(--rx-ink);
}

.rx-aff-link {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
  color: var(--rx-ink-2);
}

.rx-aff-copy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.5rem 0.9rem;
  border-radius: var(--rx-radius-sm);
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  transition: transform 0.35s var(--rx-spring);
}

.rx-aff-copy:hover {
  transform: translateY(-1px);
}

.rx-aff-copy:active {
  transform: scale(0.96);
}

/* 使用说明：竖向步骤条，序号为墨色圆点，连接线入场时自上而下描出 */
.rx-aff-tips {
  position: relative;
  counter-reset: tip;
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
}

.rx-aff-tips::before {
  content: '';
  position: absolute;
  left: 0.85rem;
  top: 0.9rem;
  bottom: 0.9rem;
  width: 1.5px;
  background: linear-gradient(var(--rx-line-strong), var(--rx-line-soft));
  transform-origin: top;
  animation: rx-aff-line 1.2s var(--rx-ease) both 0.4s;
}

@keyframes rx-aff-line {
  from { transform: scaleY(0); }
  to { transform: scaleY(1); }
}

.rx-aff-tips li {
  position: relative;
  counter-increment: tip;
  min-height: 1.75rem;
  padding-left: 2.6rem;
  font-size: 0.86rem;
  line-height: 1.6;
  color: var(--rx-ink-2);
  animation: rx-aff-step 0.5s var(--rx-ease) both;
}

.rx-aff-tips li:nth-child(1) { animation-delay: 0.45s; }
.rx-aff-tips li:nth-child(2) { animation-delay: 0.6s; }
.rx-aff-tips li:nth-child(3) { animation-delay: 0.75s; }
.rx-aff-tips li:nth-child(4) { animation-delay: 0.9s; }

@keyframes rx-aff-step {
  from { opacity: 0; transform: translateX(-6px); }
  to { opacity: 1; transform: none; }
}

.rx-aff-tips li::before {
  content: counter(tip);
  position: absolute;
  left: 0;
  top: 0;
  display: grid;
  place-items: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 9999px;
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  box-shadow: 0 0 0 4px var(--rx-surface-hover);
}

.rx-aff-tips li:last-child::before {
  color: var(--rx-accent);
  background: var(--rx-surface-hover);
  border: 1.5px solid var(--rx-accent);
}

/* ---------- 转入 ---------- */
.rx-aff-transfer {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.4rem;
}

@media (min-width: 640px) {
  .rx-aff-transfer {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    padding: 1.5rem 1.75rem;
  }
}

.rx-aff-transfer__action {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 1rem;
}

.rx-aff-transfer__amount {
  font-size: 1.6rem;
  font-weight: 800;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink);
}

.rx-aff-cta {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.75rem 1.2rem;
  border-radius: var(--rx-radius-md);
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  box-shadow: 0 1px 2px rgba(17, 17, 15, 0.14), 0 10px 24px -12px rgba(17, 17, 15, 0.45);
  transition: transform 0.35s var(--rx-spring), opacity 0.2s ease;
}

.rx-aff-cta:not(:disabled):hover {
  transform: translateY(-2px);
}

.rx-aff-cta:disabled {
  cursor: not-allowed;
  opacity: 0.35;
  box-shadow: none;
}

/* ---------- 被邀请人 ---------- */
.rx-aff-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 2.5rem 1rem;
  font-size: 0.85rem;
  color: var(--rx-muted);
}

/* 空态：三颗沿轨道转动的小点，暗示「等待第一位好友」 */
.rx-aff-empty__orbit {
  position: relative;
  width: 3.5rem;
  height: 3.5rem;
  border-radius: 9999px;
  border: 1px dashed var(--rx-line-strong);
  animation: rx-aff-orbit 9s linear infinite;
}

.rx-aff-empty__orbit i {
  position: absolute;
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 9999px;
  background: var(--rx-ink);
}

.rx-aff-empty__orbit i:nth-child(1) { top: -0.28rem; left: calc(50% - 0.28rem); }
.rx-aff-empty__orbit i:nth-child(2) { bottom: 0.2rem; left: 0.1rem; opacity: 0.5; }
.rx-aff-empty__orbit i:nth-child(3) { bottom: 0.2rem; right: 0.1rem; opacity: 0.25; }

@keyframes rx-aff-orbit {
  to { transform: rotate(360deg); }
}

.rx-aff-table th {
  padding: 0.6rem 0.75rem;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--rx-faint);
  border-bottom: 1px solid var(--rx-line);
}

.rx-aff-table td {
  padding: 0.75rem;
  color: var(--rx-ink);
  border-bottom: 1px solid var(--rx-line-soft);
}

.rx-aff-table tbody tr:last-child td {
  border-bottom: 0;
}

.rx-aff-person {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.6rem;
}

.rx-aff-person__avatar {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 1.8rem;
  height: 1.8rem;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 800;
  color: var(--rx-tint-ink);
  background: var(--rx-tint);
}
</style>
