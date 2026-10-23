<template>
  <section
    class="plaza-group overflow-hidden"
    :class="[platformBorderStrongClass(group.platform)]"
    :style="{ '--plaza-accent': platformAccentColor(group.platform) }"
  >
    <!-- 分组头部:名称/平台/倍率徽章/专属/订阅徽章 + 描述 -->
    <header class="plaza-group__head px-5 py-4">
      <div class="flex flex-wrap items-center gap-2">
        <GroupBadge
          :name="group.name"
          :platform="group.platform as GroupPlatform"
          :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
          :rate-multiplier="group.rate_multiplier"
          :user-rate-multiplier="group.user_rate_multiplier ?? null"
          :peak-rate-enabled="group.peak_rate_enabled"
          :peak-start="group.peak_start"
          :peak-end="group.peak_end"
          :peak-rate-multiplier="group.peak_rate_multiplier"
          always-show-rate
        />
        <span
          v-if="group.is_exclusive"
          class="inline-flex items-center gap-1 rounded-md bg-purple-50 px-2 py-0.5 text-xs font-medium text-purple-600 dark:bg-purple-900/20 dark:text-purple-400"
        >
          <Icon name="shield" size="xs" class="h-3 w-3" />
          {{ t('modelPlaza.badges.exclusive') }}
        </span>
        <span
          v-if="group.subscription_type === 'subscription'"
          class="inline-flex items-center rounded-md bg-violet-50 px-2 py-0.5 text-xs font-medium text-violet-600 dark:bg-violet-900/20 dark:text-violet-400"
        >
          {{ t('modelPlaza.badges.subscription') }}
        </span>
        <span class="plaza-group__count">{{ t('modelPlaza.hero.modelCount', { count: group.models.length }) }}</span>
      </div>
      <p v-if="group.description" class="mt-2 text-sm text-gray-500 dark:text-dark-400">
        {{ group.description }}
      </p>
      <p
        v-if="peakNote"
        class="mt-1.5 inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400"
      >
        <Icon name="clock" size="xs" class="h-3 w-3" />
        {{ peakNote }}
      </p>
      <p
        v-if="longContextNote"
        class="mt-1.5 flex items-center gap-1 text-xs text-gray-500 dark:text-dark-400"
      >
        <Icon name="infoCircle" size="xs" class="h-3 w-3" />
        {{ longContextNote }}
      </p>
    </header>

    <!-- 模型价格表:整行(含 hover 底色/分区底色)顶到卡片边缘,左右留白由表格首列/末列的 padding 提供 -->
    <div>
      <PlazaModelPricingTable
        v-if="group.models.length > 0"
        :models="group.models"
        :platform="group.platform"
        :rate-multiplier="group.rate_multiplier"
        :user-rate-multiplier="group.user_rate_multiplier ?? null"
        :image-rate-independent="group.image_rate_independent"
        :image-rate-multiplier="group.image_rate_multiplier"
        :video-rate-independent="group.video_rate_independent"
        :video-rate-multiplier="group.video_rate_multiplier"
        :peak-window="peakWindow"
        :peak-rate-multiplier="group.peak_rate_multiplier"
      />
      <p v-else class="px-5 py-4 text-center text-sm text-gray-400 dark:text-dark-500">
        {{ t('modelPlaza.detail.noModels') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlazaModelPricingTable from './PlazaModelPricingTable.vue'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformAccentColor, platformBorderStrongClass } from '@/utils/platformColors'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { useAppStore } from '@/stores/app'

const props = defineProps<{
  group: ModelPlazaGroup
}>()

const { t } = useI18n()
const appStore = useAppStore()

/** 高峰窗口描述(含倍率与服务器时区标注);分组未启用高峰为空串。 */
const peakWindow = computed(() => {
  if (!hasPeakRate(props.group)) return ''
  return formatPeakRateWindow(
    props.group,
    serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)
  )
})

const peakNote = computed(() => {
  if (!peakWindow.value) return ''
  return t('modelPlaza.detail.peakNote', {
    window: peakWindow.value,
    multiplier: props.group.peak_rate_multiplier
  })
})

/**
 * 分组关闭了长上下文阶梯、但组内有模型官方带阶梯时提示:实付列只展示基础档,
 * 官方阶梯仅供参考。字段缺失(旧后端)不提示。
 */
const longContextNote = computed(() => {
  if (props.group.long_context_pricing_enabled !== false) return ''
  const hasOfficialLadder = props.group.models.some(
    (m) => (m.official_pricing?.intervals?.length ?? 0) > 1
  )
  return hasOfficialLadder ? t('modelPlaza.detail.longContextDisabledNote') : ''
})
</script>

<style scoped>
/* 分组卡：玻璃底 + 左侧平台主色细条；悬浮时细条拉长、卡片轻抬 */
.plaza-group {
  position: relative;
  border-radius: var(--rx-radius-lg, 1rem);
  background: var(--rx-surface, #fff);
  border: 1px solid var(--rx-line, #e5e7eb) !important;
  box-shadow: var(--rx-highlight, 0 0 0 transparent), var(--rx-shadow, 0 1px 3px rgba(0, 0, 0, 0.05));
  transition: transform 0.35s var(--rx-ease, ease), box-shadow 0.3s ease, border-color 0.3s ease;
}

.plaza-group::before {
  content: '';
  position: absolute;
  left: 0;
  top: 1.1rem;
  height: 2.2rem;
  width: 3px;
  border-radius: 0 3px 3px 0;
  background: var(--plaza-accent, var(--rx-accent, #111827));
  transition: height 0.45s var(--rx-ease, ease);
}

.plaza-group:hover {
  border-color: var(--rx-line-strong, #d1d5db) !important;
  box-shadow: var(--rx-highlight, 0 0 0 transparent), var(--rx-shadow-lg, 0 10px 30px rgba(0, 0, 0, 0.08));
}

.plaza-group:hover::before {
  height: calc(100% - 2.2rem);
}

.plaza-group__head {
  border-bottom: 1px solid var(--rx-line-soft, #f3f4f6);
  background: linear-gradient(90deg, color-mix(in srgb, var(--plaza-accent, transparent) 6%, transparent), transparent 55%);
}

.plaza-group__count {
  margin-left: auto;
  padding: 0.1rem 0.55rem;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--rx-muted, #6b7280);
  background: var(--rx-surface-2, #f9fafb);
  border: 1px solid var(--rx-line-soft, #f3f4f6);
}
</style>
