<template>
  <section class="rx-glass dash-actions">
    <header class="rx-section-head">
      <span class="rx-section-head__index">05</span>
      <h2 class="rx-section-head__title">{{ t('dashboard.quickActions') }}</h2>
    </header>

    <div class="dash-actions__list">
      <button type="button" class="dash-action" @click="router.push('/keys')">
        <span class="dash-action__idx"></span>
        <span class="dash-action__icon" aria-hidden="true">
          <Icon name="key" size="md" />
        </span>
        <span class="min-w-0 flex-1 text-left">
          <span class="dash-action__title">{{ t('dashboard.createApiKey') }}</span>
          <span class="dash-action__desc">{{ t('dashboard.generateNewKey') }}</span>
        </span>
        <Icon name="chevronRight" size="md" class="dash-action__go" />
      </button>

      <button type="button" class="dash-action" @click="router.push('/usage')">
        <span class="dash-action__idx"></span>
        <span class="dash-action__icon" aria-hidden="true">
          <Icon name="chart" size="md" />
        </span>
        <span class="min-w-0 flex-1 text-left">
          <span class="dash-action__title">{{ t('dashboard.viewUsage') }}</span>
          <span class="dash-action__desc">{{ t('dashboard.checkDetailedLogs') }}</span>
        </span>
        <Icon name="chevronRight" size="md" class="dash-action__go" />
      </button>

      <button v-if="canUseBatchImage" type="button" class="dash-action" @click="router.push('/batch-image')">
        <span class="dash-action__idx"></span>
        <span class="dash-action__icon" aria-hidden="true">
          <Icon name="sparkles" size="md" />
        </span>
        <span class="min-w-0 flex-1 text-left">
          <span class="dash-action__title">{{ t('dashboard.batchImageAgent') }}</span>
          <span class="dash-action__desc">{{ t('dashboard.batchImageAgentDesc') }}</span>
        </span>
        <Icon name="chevronRight" size="md" class="dash-action__go" />
      </button>

      <button type="button" class="dash-action" @click="router.push('/redeem')">
        <span class="dash-action__idx"></span>
        <span class="dash-action__icon" aria-hidden="true">
          <Icon name="gift" size="md" />
        </span>
        <span class="min-w-0 flex-1 text-left">
          <span class="dash-action__title">{{ t('dashboard.redeemCode') }}</span>
          <span class="dash-action__desc">{{ t('dashboard.addBalanceWithCode') }}</span>
        </span>
        <Icon name="chevronRight" size="md" class="dash-action__go" />
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>

<style scoped>
.dash-actions {
  padding: 1.25rem 1.25rem 1.15rem;
}

@media (min-width: 640px) {
  .dash-actions {
    padding: 1.5rem 1.45rem 1.3rem;
  }
}

.dash-actions .rx-section-head {
  margin-bottom: 0.85rem;
}

.dash-actions__list {
  counter-reset: dash-a;
  display: grid;
  gap: 0.55rem;
}

.dash-action {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.8rem;
  padding: 0.85rem 0.9rem;
  border-radius: var(--rx-radius-md);
  text-align: left;
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
  transition: transform 0.35s var(--rx-spring), border-color 0.2s ease, background 0.2s ease;
}

.dash-action:hover {
  background: var(--rx-surface-hover);
  border-color: var(--rx-line-strong);
  transform: translateY(-1px);
}

.dash-action__idx::before {
  counter-increment: dash-a;
  content: counter(dash-a, decimal-leading-zero);
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 0.82rem;
  color: var(--rx-faint);
}

.dash-action__icon {
  display: grid;
  place-items: center;
  width: 2.4rem;
  height: 2.4rem;
  flex-shrink: 0;
  border-radius: 12px;
  color: var(--rx-on-accent);
  background: var(--rx-accent);
}

.dash-action__title {
  display: block;
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--rx-ink);
}

.dash-action__desc {
  display: block;
  margin-top: 0.12rem;
  font-size: 0.72rem;
  color: var(--rx-faint);
}

.dash-action__go {
  flex-shrink: 0;
  color: var(--rx-faint);
}

.dash-action:hover .dash-action__go {
  color: var(--rx-ink);
}
</style>
