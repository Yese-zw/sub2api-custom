<template>
  <AppLayout>
    <RxStage>
      <!-- 首次加载骨架 -->
      <div v-if="!status && loading" class="space-y-6" aria-busy="true">
        <div class="ck-skeleton ck-skeleton--hero"></div>
        <div class="grid gap-3 sm:grid-cols-3">
          <div v-for="i in 3" :key="i" class="ck-skeleton ck-skeleton--tile"></div>
        </div>
        <div class="ck-skeleton ck-skeleton--panel"></div>
      </div>

      <!-- 加载失败 -->
      <section v-else-if="!status" class="rx-glass ck-empty">
        <span class="ck-empty__icon" aria-hidden="true"><Icon name="exclamationCircle" size="lg" /></span>
        <h2 class="ck-empty__title">{{ t('checkin.errors.loadFailed') }}</h2>
        <button type="button" class="ck-ghost-btn" @click="loadStatus">{{ t('common.tryAgain') }}</button>
      </section>

      <!-- 功能未开启 -->
      <section v-else-if="!status.enabled" class="rx-glass ck-empty rx-rise">
        <span class="ck-empty__icon" aria-hidden="true"><Icon name="calendar" size="lg" /></span>
        <h2 class="ck-empty__title">{{ t('checkin.disabled.title') }}</h2>
        <p class="ck-empty__desc">{{ t('checkin.disabled.desc') }}</p>
      </section>

      <template v-else>
        <!-- ================= 主视觉 ================= -->
        <section class="ck-hero rx-rise" style="--rx-i: 0">
          <span class="ck-hero__glow" aria-hidden="true"></span>
          <span class="ck-hero__rings" aria-hidden="true"></span>

          <div class="ck-hero__main">
            <p class="ck-hero__kicker">
              <span>{{ t('checkin.hero.kicker') }}</span>
              <span class="ck-hero__dot" aria-hidden="true"></span>
              <span>{{ t('checkin.todayLabel', { date: todayDisplay, tz: status.timezone }) }}</span>
            </p>
            <h1 class="ck-hero__title">{{ t('checkin.title') }}</h1>
            <p class="ck-hero__sub">{{ t('checkin.subtitle') }}</p>

            <div class="ck-hero__reward">
              <p class="ck-hero__reward-label">
                {{ status.checked_in_today ? t('checkin.hero.rewardGot') : t('checkin.hero.rewardToday') }}
              </p>
              <p class="ck-hero__reward-value" aria-live="polite">
                <template v-if="status.checked_in_today">+{{ money(displayReward) }}</template>
                <template v-else>{{ rewardRangeText }}</template>
              </p>
              <div class="ck-hero__chips">
                <span v-if="status.next_bonus > 0" class="ck-chip ck-chip--gold">
                  <Icon name="gift" size="xs" />
                  {{ t('checkin.hero.bonusToday', { amount: money(status.next_bonus) }) }}
                </span>
                <span v-if="status.checked_in_today && status.today_record" class="ck-chip">
                  {{ t('checkin.history.base', { amount: money(status.today_record.base_reward) }) }}
                </span>
              </div>
            </div>

            <div class="ck-streak">
              <span class="ck-streak__flame" :class="{ 'is-lit': status.current_streak > 0 }" aria-hidden="true">
                <Icon name="fire" size="md" />
              </span>
              <div>
                <p class="ck-streak__label">{{ t('checkin.hero.streakLabel') }}</p>
                <p class="ck-streak__value">
                  {{ status.current_streak }}<small>{{ t('checkin.hero.streakUnit') }}</small>
                </p>
              </div>
              <p class="ck-streak__hint">{{ streakHint }}</p>
            </div>
          </div>

          <!-- 签到印章按钮 -->
          <div class="ck-hero__action">
            <div class="ck-seal-wrap">
              <span class="ck-seal-wrap__halo" :class="{ 'is-idle': canCheckin }" aria-hidden="true"></span>
              <button
                type="button"
                class="ck-seal"
                :class="sealClass"
                :disabled="!canCheckin || submitting"
                :aria-label="sealLabel"
                @click="handleCheckin"
              >
                <svg class="ck-seal__ring" viewBox="0 0 200 200" aria-hidden="true">
                  <defs>
                    <path id="ck-seal-arc" d="M100,100 m-78,0 a78,78 0 1,1 156,0 a78,78 0 1,1 -156,0" />
                  </defs>
                  <circle cx="100" cy="100" r="92" class="ck-seal__dash" />
                  <text class="ck-seal__arc-text">
                    <textPath href="#ck-seal-arc">DAILY · CHECK · IN · DAILY · CHECK · IN · </textPath>
                  </text>
                </svg>
                <span class="ck-seal__core">
                  <span v-if="submitting" class="ck-seal__spinner" aria-hidden="true"></span>
                  <Icon v-else :name="sealIcon" size="xl" />
                  <span class="ck-seal__label">{{ sealLabel }}</span>
                </span>
              </button>

              <!-- 金币迸发 -->
              <div v-if="burstKey" :key="burstKey" class="ck-burst" aria-hidden="true">
                <span
                  v-for="p in particles"
                  :key="p.id"
                  class="ck-burst__coin"
                  :style="{ '--a': p.angle + 'deg', '--d': p.dist + 'px', '--s': p.scale, '--delay': p.delay + 'ms' }"
                >$</span>
              </div>
              <Transition name="ck-pop">
                <span v-if="floatVisible" class="ck-float">+{{ money(lastReward) }}</span>
              </Transition>
            </div>

            <p v-if="!status.checked_in_today && !status.eligible" class="ck-lock">
              <Icon name="lock" size="xs" />
              {{ ineligibleText }}
            </p>
          </div>
        </section>

        <!-- ================= 统计 ================= -->
        <div class="ck-stats">
          <div class="rx-glass rx-metric rx-rise" style="--rx-i: 1">
            <span class="rx-metric__index">01</span>
            <p class="rx-metric__label"><Icon name="calendar" size="xs" />{{ t('checkin.stats.totalDays') }}</p>
            <p class="rx-metric__value">{{ status.totals.total_days }}<small class="ck-unit">{{ t('checkin.stats.daysUnit') }}</small></p>
          </div>
          <div class="rx-glass rx-metric rx-rise" style="--rx-i: 2">
            <span class="rx-metric__index">02</span>
            <p class="rx-metric__label"><Icon name="gift" size="xs" />{{ t('checkin.stats.totalReward') }}</p>
            <p class="rx-metric__value">{{ money(status.totals.total_reward) }}</p>
          </div>
          <div class="rx-glass rx-metric rx-rise" style="--rx-i: 3">
            <span class="rx-metric__index">03</span>
            <p class="rx-metric__label"><Icon name="creditCard" size="xs" />{{ t('checkin.stats.balance') }}</p>
            <p class="rx-metric__value">{{ money(user?.balance ?? 0) }}</p>
          </div>
        </div>

        <!-- ================= 连签进度 ================= -->
        <section class="rx-glass ck-panel rx-rise" style="--rx-i: 4">
          <header class="rx-section-head">
            <span class="rx-section-head__index">01</span>
            <h2 class="rx-section-head__title">{{ t('checkin.cycle.title') }}</h2>
            <span class="rx-section-head__hint">{{ t('checkin.cycle.hint', { current: cyclePosition, total: status.cycle.length }) }}</span>
          </header>
          <div class="ck-track" aria-hidden="true">
            <span class="ck-track__fill" :style="{ width: cycleProgress + '%' }"></span>
          </div>
          <ol class="ck-cycle">
            <li
              v-for="d in status.cycle"
              :key="d.day"
              class="ck-day"
              :class="{
                'is-claimed': d.claimed,
                'is-today': d.is_today,
                'is-bonus': d.bonus > 0,
              }"
            >
              <span class="ck-day__label">{{ d.is_today ? t('checkin.cycle.today') : t('checkin.cycle.dayN', { n: d.day }) }}</span>
              <span class="ck-day__icon" aria-hidden="true">
                <Icon v-if="d.claimed" name="check" size="sm" />
                <Icon v-else-if="d.bonus > 0" name="gift" size="sm" />
                <span v-else class="ck-day__dot"></span>
              </span>
              <span class="ck-day__reward">{{ d.bonus > 0 ? '+' + money(d.bonus) : baseShort }}</span>
            </li>
          </ol>
        </section>

        <div class="grid items-start gap-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]">
          <!-- ================= 本月日历 ================= -->
          <section class="rx-glass ck-panel rx-rise" style="--rx-i: 5">
            <header class="rx-section-head">
              <span class="rx-section-head__index">02</span>
              <h2 class="rx-section-head__title">{{ t('checkin.calendar.title') }}</h2>
              <span class="rx-section-head__hint">{{ t('checkin.calendar.checkedCount', { n: status.month_dates.length }) }}</span>
            </header>
            <p class="ck-month">{{ monthLabel }}</p>
            <div class="ck-cal">
              <span v-for="w in weekdays" :key="w" class="ck-cal__wd">{{ w }}</span>
              <span v-for="n in calendar.leading" :key="'pad-' + n" class="ck-cal__pad"></span>
              <span
                v-for="cell in calendar.days"
                :key="cell.date"
                class="ck-cal__day"
                :class="{ 'is-done': cell.done, 'is-today': cell.today, 'is-future': cell.future }"
              >
                <span class="ck-cal__num">{{ cell.day }}</span>
                <Icon v-if="cell.done" name="check" size="xs" class="ck-cal__tick" />
              </span>
            </div>
          </section>

          <!-- ================= 规则 ================= -->
          <section class="rx-glass ck-panel rx-rise" style="--rx-i: 6">
            <header class="rx-section-head">
              <span class="rx-section-head__index">03</span>
              <h2 class="rx-section-head__title">{{ t('checkin.rules.title') }}</h2>
            </header>
            <dl class="ck-rules">
              <div class="ck-rule">
                <dt><span class="ck-rule__icon"><Icon name="dollar" size="sm" /></span>{{ t('checkin.rules.reward') }}</dt>
                <dd>{{ rewardRuleText }}</dd>
              </div>
              <div class="ck-rule">
                <dt><span class="ck-rule__icon"><Icon name="shield" size="sm" /></span>{{ t('checkin.rules.threshold') }}</dt>
                <dd>
                  <template v-if="!thresholds.length">{{ t('checkin.rules.none') }}</template>
                  <ul v-else class="ck-rule__list">
                    <li v-for="item in thresholds" :key="item">{{ item }}</li>
                  </ul>
                </dd>
              </div>
              <div class="ck-rule">
                <dt><span class="ck-rule__icon"><Icon name="fire" size="sm" /></span>{{ t('checkin.rules.streak') }}</dt>
                <dd>
                  <template v-if="!status.streak_rewards.length">{{ t('checkin.rules.noStreak') }}</template>
                  <ul v-else class="ck-rule__list">
                    <li v-for="r in status.streak_rewards" :key="r.day" class="ck-rule__bonus">
                      <span>{{ t('checkin.rules.streakItem', { day: r.day }) }}</span>
                      <strong>+{{ money(r.bonus) }}</strong>
                    </li>
                  </ul>
                </dd>
              </div>
            </dl>
            <p class="ck-note">
              {{ status.streak_cycle_days > 0 ? t('checkin.rules.cycleNote', { days: status.streak_cycle_days }) : t('checkin.rules.noCycleNote') }}
              {{ t('checkin.rules.resetNote', { tz: status.timezone }) }}
            </p>
          </section>
        </div>

        <!-- ================= 记录 ================= -->
        <section class="rx-glass ck-panel rx-rise" style="--rx-i: 7">
          <header class="rx-section-head">
            <span class="rx-section-head__index">04</span>
            <h2 class="rx-section-head__title">{{ t('checkin.history.title') }}</h2>
            <span class="rx-section-head__hint">{{ historyTotal }}</span>
          </header>
          <div v-if="historyLoading && !history.length" class="ck-history-empty">{{ t('common.loading') }}</div>
          <div v-else-if="!history.length" class="ck-history-empty">
            <Icon name="calendar" size="lg" />
            <span>{{ t('checkin.history.empty') }}</span>
          </div>
          <ul v-else class="ck-history">
            <li v-for="item in history" :key="item.id" class="ck-history__item">
              <span class="ck-history__date">
                <strong>{{ item.checkin_date.slice(8, 10) }}</strong>
                <small>{{ item.checkin_date.slice(0, 7) }}</small>
              </span>
              <div class="ck-history__body">
                <p class="ck-history__title">{{ t('checkin.history.streak', { n: item.streak_days }) }}</p>
                <p class="ck-history__meta">
                  {{ t('checkin.history.base', { amount: money(item.base_reward) }) }}
                  <template v-if="item.streak_bonus > 0"> · <span class="ck-history__bonus">{{ t('checkin.history.bonus', { amount: money(item.streak_bonus) }) }}</span></template>
                </p>
              </div>
              <span class="ck-history__amount">+{{ money(item.reward_amount) }}</span>
            </li>
          </ul>
          <footer v-if="historyTotal > historyPageSize" class="ck-pager">
            <button type="button" class="ck-ghost-btn" :disabled="historyLoading || historyPage <= 1" @click="loadHistory(historyPage - 1)">
              {{ t('pagination.previous') }}
            </button>
            <span class="ck-pager__info">{{ historyPage }} / {{ historyPages }}</span>
            <button type="button" class="ck-ghost-btn" :disabled="historyLoading || historyPage >= historyPages" @click="loadHistory(historyPage + 1)">
              {{ t('pagination.next') }}
            </button>
          </footer>
        </section>
      </template>
    </RxStage>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import RxStage from '@/components/common/RxStage.vue'
import Icon from '@/components/icons/Icon.vue'
import { checkinAPI, type CheckinRecord, type CheckinStatus } from '@/api/checkin'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const user = computed(() => authStore.user)

const status = ref<CheckinStatus | null>(null)
const loading = ref(false)
const submitting = ref(false)

const history = ref<CheckinRecord[]>([])
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = 8
const historyTotal = ref(0)
const historyPages = computed(() => Math.max(1, Math.ceil(historyTotal.value / historyPageSize)))

// 成功动效
const burstKey = ref(0)
const floatVisible = ref(false)
const lastReward = ref(0)
const displayReward = ref(0)
let floatTimer: ReturnType<typeof setTimeout> | undefined
let countFrame = 0

const particles = Array.from({ length: 14 }, (_, i) => ({
  id: i,
  angle: (360 / 14) * i + (i % 2 ? 9 : -9),
  dist: 78 + (i % 3) * 22,
  scale: 0.75 + (i % 4) * 0.12,
  delay: (i % 5) * 25,
}))

function money(v: number): string {
  return `$${(Number.isFinite(v) ? v : 0).toFixed(2)}`
}

const todayDisplay = computed(() => status.value?.today ?? '')

const rewardRangeText = computed(() => {
  const s = status.value
  if (!s) return ''
  const bonus = s.next_bonus || 0
  if (s.reward_min === s.reward_max) return money(s.reward_min + bonus)
  return `${money(s.reward_min + bonus)} ~ ${money(s.reward_max + bonus)}`
})

const rewardRuleText = computed(() => {
  const s = status.value
  if (!s) return ''
  return s.reward_min === s.reward_max
    ? t('checkin.rules.rewardFixed', { amount: money(s.reward_min) })
    : t('checkin.rules.rewardRandom', { min: money(s.reward_min), max: money(s.reward_max) })
})

const baseShort = computed(() => {
  const s = status.value
  if (!s) return ''
  return s.reward_min === s.reward_max ? money(s.reward_min) : `${money(s.reward_min)}+`
})

const thresholds = computed(() => {
  const s = status.value
  if (!s) return []
  const list: string[] = []
  if (s.min_total_recharged > 0) list.push(t('checkin.rules.minRecharge', { amount: money(s.min_total_recharged) }))
  if (s.min_account_age_days > 0) list.push(t('checkin.rules.minAge', { days: s.min_account_age_days }))
  return list
})

const canCheckin = computed(() => !!status.value && status.value.enabled && status.value.eligible && !status.value.checked_in_today)

const sealClass = computed(() => {
  if (!status.value) return ''
  if (status.value.checked_in_today) return 'is-done'
  if (!status.value.eligible) return 'is-locked'
  return 'is-ready'
})

const sealIcon = computed(() => {
  if (status.value?.checked_in_today) return 'checkCircle'
  if (status.value && !status.value.eligible) return 'lock'
  return 'sparkles'
})

const sealLabel = computed(() => {
  if (submitting.value) return t('checkin.button.checking')
  if (status.value?.checked_in_today) return t('checkin.button.done')
  if (status.value && !status.value.eligible) return t('checkin.button.locked')
  return t('checkin.button.checkin')
})

const ineligibleText = computed(() => {
  const s = status.value
  if (!s?.ineligible_reason) return ''
  return t(`checkin.ineligible.${s.ineligible_reason}`, {
    amount: money(s.min_total_recharged),
    days: s.min_account_age_days,
  })
})

const cyclePosition = computed(() => {
  const cycle = status.value?.cycle ?? []
  const idx = cycle.findIndex((d) => d.is_today)
  return idx >= 0 ? idx + 1 : 0
})

const cycleProgress = computed(() => {
  const cycle = status.value?.cycle ?? []
  if (!cycle.length) return 0
  const claimed = cycle.filter((d) => d.claimed).length
  return Math.round((claimed / cycle.length) * 100)
})

const streakHint = computed(() => {
  const s = status.value
  if (!s) return ''
  // 找到今天之后（含今天未签）最近一个有加成的日子
  const todayIdx = s.cycle.findIndex((d) => d.is_today)
  const start = s.checked_in_today ? todayIdx + 1 : todayIdx
  for (let i = Math.max(start, 0); i < s.cycle.length; i++) {
    const d = s.cycle[i]
    if (d.bonus > 0 && !d.claimed) {
      const n = i - todayIdx + (s.checked_in_today ? 0 : 1)
      if (n > 0 && !(n === 1 && !s.checked_in_today)) {
        return t('checkin.hero.streakNext', { n, amount: money(d.bonus) })
      }
    }
  }
  if (s.checked_in_today) return t('checkin.hero.streakKeep')
  return s.current_streak > 0 ? t('checkin.hero.streakKeep') : t('checkin.hero.streakIdle')
})

const weekdays = computed<string[]>(() => t('checkin.calendar.weekdays').split(','))

const monthLabel = computed(() => {
  const today = status.value?.today
  if (!today) return ''
  const [y, m] = today.split('-').map(Number)
  return t('checkin.calendar.monthLabel', { year: y, month: m })
})

const calendar = computed(() => {
  const today = status.value?.today
  if (!today) return { leading: 0, days: [] as Array<{ date: string; day: number; done: boolean; today: boolean; future: boolean }> }
  const [y, m, d] = today.split('-').map(Number)
  const first = new Date(Date.UTC(y, m - 1, 1))
  const leading = (first.getUTCDay() + 6) % 7 // 周一为第一列
  const daysInMonth = new Date(Date.UTC(y, m, 0)).getUTCDate()
  const done = new Set(status.value?.month_dates ?? [])
  const prefix = `${y}-${String(m).padStart(2, '0')}-`
  const days = Array.from({ length: daysInMonth }, (_, i) => {
    const day = i + 1
    const date = prefix + String(day).padStart(2, '0')
    return { date, day, done: done.has(date), today: day === d, future: day > d }
  })
  return { leading, days }
})

function animateReward(target: number) {
  cancelAnimationFrame(countFrame)
  const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  if (reduce) {
    displayReward.value = target
    return
  }
  const start = performance.now()
  const duration = 900
  const step = (now: number) => {
    const p = Math.min(1, (now - start) / duration)
    const eased = 1 - Math.pow(1 - p, 3)
    displayReward.value = target * eased
    if (p < 1) countFrame = requestAnimationFrame(step)
  }
  countFrame = requestAnimationFrame(step)
}

async function loadStatus() {
  loading.value = true
  try {
    status.value = await checkinAPI.getStatus()
    displayReward.value = status.value.today_record?.reward_amount ?? 0
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('checkin.errors.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function loadHistory(page = 1) {
  historyLoading.value = true
  try {
    const data = await checkinAPI.getHistory(page, historyPageSize)
    history.value = data.items
    historyTotal.value = data.total
    historyPage.value = page
  } catch {
    // 记录加载失败不影响签到主流程
  } finally {
    historyLoading.value = false
  }
}

async function handleCheckin() {
  if (!canCheckin.value || submitting.value) return
  submitting.value = true
  try {
    const result = await checkinAPI.checkin()
    lastReward.value = result.record.reward_amount
    burstKey.value++
    floatVisible.value = true
    if (floatTimer) clearTimeout(floatTimer)
    floatTimer = setTimeout(() => { floatVisible.value = false }, 1800)

    await loadStatus()
    animateReward(result.record.reward_amount)
    void loadHistory(1)
    void authStore.refreshUser().catch(() => undefined)
    appStore.showSuccess(t('checkin.success.message', { amount: money(result.record.reward_amount) }))
  } catch (err) {
    const code = extractApiErrorCode(err)
    if (code === 'CHECKIN_ALREADY_DONE' || code === 'CHECKIN_DISABLED') {
      appStore.showError(t(`checkin.errors.${code}`))
      void loadStatus()
    } else {
      appStore.showError(extractApiErrorMessage(err, t('common.error')))
    }
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await loadStatus()
  if (status.value?.enabled) void loadHistory(1)
})

onBeforeUnmount(() => {
  if (floatTimer) clearTimeout(floatTimer)
  cancelAnimationFrame(countFrame)
})
</script>

<style scoped>
/* ==========================================================================
   签到页局部令牌：在玻璃主题上叠一层暖金色，作为"奖励"的专属强调色
   ========================================================================== */
.ck-hero,
.ck-panel,
.ck-stats,
.ck-empty {
  --ck-gold: #f59e0b;
  --ck-gold-2: #fbbf24;
  --ck-gold-deep: #b45309;
  --ck-gold-soft: rgba(245, 158, 11, 0.14);
  --ck-gold-line: rgba(245, 158, 11, 0.38);
}

/* ---------------- 骨架 ---------------- */
.ck-skeleton {
  border-radius: var(--rx-radius-lg);
  background: linear-gradient(90deg, var(--rx-surface-2), var(--rx-surface), var(--rx-surface-2));
  background-size: 200% 100%;
  animation: ck-shimmer 1.4s ease-in-out infinite;
  border: 1px solid var(--rx-line-soft);
}
.ck-skeleton--hero { height: 300px; }
.ck-skeleton--tile { height: 108px; }
.ck-skeleton--panel { height: 180px; }
@keyframes ck-shimmer {
  0% { background-position: 100% 0; }
  100% { background-position: -100% 0; }
}

/* ---------------- 空态 ---------------- */
.ck-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 3.5rem 1.5rem;
  text-align: center;
}
.ck-empty__icon {
  display: grid;
  place-items: center;
  width: 3.5rem;
  height: 3.5rem;
  border-radius: 1.1rem;
  color: var(--ck-gold-deep);
  background: var(--ck-gold-soft);
}
.ck-empty__title {
  font-size: 1.15rem;
  font-weight: 750;
  color: var(--rx-ink);
}
.ck-empty__desc {
  max-width: 26rem;
  font-size: 0.875rem;
  color: var(--rx-muted);
}

/* ---------------- 主视觉 ---------------- */
.ck-hero {
  position: relative;
  overflow: hidden;
  /* clip（而非 hidden）不可被滚动：金币迸发越界时不会把内容挤偏 */
  overflow: clip;
  display: grid;
  gap: 2rem;
  padding: 1.75rem;
  border-radius: var(--rx-radius-lg);
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  border: 1px solid var(--rx-card-line);
  box-shadow: var(--rx-shadow-lg);
}
@media (min-width: 768px) {
  .ck-hero {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    padding: 2.25rem 2.5rem;
  }
}
.ck-hero__glow {
  position: absolute;
  right: -6rem;
  top: -6rem;
  width: 22rem;
  height: 22rem;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(251, 191, 36, 0.32), transparent 65%);
  filter: blur(8px);
  pointer-events: none;
}
.ck-hero__rings {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(circle at 85% 50%, transparent 0 7rem, rgba(255, 255, 255, 0.05) 7rem 7.06rem, transparent 7.06rem),
    radial-gradient(circle at 85% 50%, transparent 0 10rem, rgba(255, 255, 255, 0.04) 10rem 10.06rem, transparent 10.06rem),
    radial-gradient(circle at 85% 50%, transparent 0 13rem, rgba(255, 255, 255, 0.03) 13rem 13.06rem, transparent 13.06rem);
  pointer-events: none;
}
.ck-hero__main {
  position: relative;
  min-width: 0;
}
.ck-hero__kicker {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(248, 250, 252, 0.6);
}
.ck-hero__dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--ck-gold-2);
}
.ck-hero__title {
  margin-top: 0.6rem;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-weight: var(--rx-display-weight);
  font-size: clamp(1.9rem, 4vw, 2.6rem);
  line-height: 1.05;
  letter-spacing: -0.035em;
}
.ck-hero__sub {
  margin-top: 0.5rem;
  max-width: 30rem;
  font-size: 0.9rem;
  color: rgba(248, 250, 252, 0.68);
}
.ck-hero__reward {
  margin-top: 1.5rem;
}
.ck-hero__reward-label {
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: rgba(248, 250, 252, 0.55);
}
.ck-hero__reward-value {
  margin-top: 0.2rem;
  font-size: clamp(2rem, 5vw, 2.75rem);
  font-weight: 800;
  line-height: 1.05;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  background: linear-gradient(180deg, #fff7d6, var(--ck-gold-2) 60%, var(--ck-gold));
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.ck-hero__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.75rem;
}
.ck-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.3rem 0.7rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 650;
  color: rgba(248, 250, 252, 0.85);
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
}
.ck-chip--gold {
  color: #fde68a;
  background: rgba(245, 158, 11, 0.16);
  border-color: rgba(251, 191, 36, 0.4);
}

/* 连签 */
.ck-streak {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.85rem 1rem;
  margin-top: 1.5rem;
  padding-top: 1.25rem;
  border-top: 1px dashed rgba(255, 255, 255, 0.14);
}
.ck-streak__flame {
  display: grid;
  place-items: center;
  width: 2.6rem;
  height: 2.6rem;
  border-radius: 0.9rem;
  color: rgba(248, 250, 252, 0.45);
  background: rgba(255, 255, 255, 0.07);
}
.ck-streak__flame.is-lit {
  color: #fff;
  background: linear-gradient(160deg, #fb923c, #ef4444);
  box-shadow: 0 10px 24px -10px rgba(239, 68, 68, 0.8);
  animation: ck-flicker 2.4s ease-in-out infinite;
}
@keyframes ck-flicker {
  0%, 100% { transform: scale(1) rotate(0); }
  50% { transform: scale(1.06) rotate(-3deg); }
}
.ck-streak__label {
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  color: rgba(248, 250, 252, 0.55);
}
.ck-streak__value {
  font-size: 1.5rem;
  font-weight: 800;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}
.ck-streak__value small {
  margin-left: 0.2rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(248, 250, 252, 0.6);
}
.ck-streak__hint {
  flex-basis: 100%;
  font-size: 0.8rem;
  color: rgba(248, 250, 252, 0.7);
}
@media (min-width: 640px) {
  .ck-streak__hint {
    flex-basis: auto;
    margin-left: auto;
  }
}

/* ---------------- 印章按钮 ---------------- */
.ck-hero__action {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}
.ck-seal-wrap {
  position: relative;
  display: grid;
  place-items: center;
  width: 13rem;
  height: 13rem;
}
.ck-seal-wrap__halo {
  position: absolute;
  inset: 0.5rem;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(251, 191, 36, 0.35), transparent 70%);
  opacity: 0;
}
.ck-seal-wrap__halo.is-idle {
  animation: ck-halo 2.6s ease-out infinite;
}
@keyframes ck-halo {
  0% { transform: scale(0.85); opacity: 0.9; }
  100% { transform: scale(1.25); opacity: 0; }
}
.ck-seal {
  position: relative;
  display: grid;
  place-items: center;
  width: 11.5rem;
  height: 11.5rem;
  border-radius: 50%;
  border: 0;
  cursor: pointer;
  transition: transform 0.35s var(--rx-spring), box-shadow 0.3s ease, filter 0.3s ease;
}
.ck-seal.is-ready {
  color: #3b1d00;
  background:
    radial-gradient(circle at 30% 25%, #fff3c4 0%, transparent 45%),
    linear-gradient(160deg, #fcd34d 0%, #f59e0b 55%, #d97706 100%);
  box-shadow:
    inset 0 2px 0 rgba(255, 255, 255, 0.6),
    inset 0 -6px 14px rgba(146, 64, 14, 0.35),
    0 18px 40px -14px rgba(245, 158, 11, 0.75);
}
.ck-seal.is-ready:hover:not(:disabled) {
  transform: translateY(-3px) scale(1.03);
  box-shadow:
    inset 0 2px 0 rgba(255, 255, 255, 0.6),
    inset 0 -6px 14px rgba(146, 64, 14, 0.35),
    0 24px 48px -14px rgba(245, 158, 11, 0.9);
}
.ck-seal.is-ready:active:not(:disabled) {
  transform: scale(0.96);
}
.ck-seal.is-done {
  cursor: default;
  color: #ecfdf5;
  background:
    radial-gradient(circle at 30% 25%, rgba(255, 255, 255, 0.28) 0%, transparent 45%),
    linear-gradient(160deg, #34d399 0%, #10b981 55%, #047857 100%);
  box-shadow:
    inset 0 2px 0 rgba(255, 255, 255, 0.35),
    inset 0 -6px 14px rgba(6, 78, 59, 0.4),
    0 18px 40px -16px rgba(16, 185, 129, 0.7);
  animation: ck-stamp 0.55s var(--rx-spring) both;
}
@keyframes ck-stamp {
  0% { transform: scale(1.25) rotate(-8deg); }
  60% { transform: scale(0.94) rotate(2deg); }
  100% { transform: scale(1) rotate(0); }
}
.ck-seal.is-locked {
  cursor: not-allowed;
  color: rgba(248, 250, 252, 0.6);
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.04));
  border: 1px solid rgba(255, 255, 255, 0.14);
}
.ck-seal:focus-visible {
  outline: 3px solid var(--ck-gold-2);
  outline-offset: 6px;
}
.ck-seal__ring {
  position: absolute;
  inset: -0.85rem;
  width: calc(100% + 1.7rem);
  height: calc(100% + 1.7rem);
  pointer-events: none;
}
.ck-seal.is-ready .ck-seal__ring {
  animation: ck-spin 22s linear infinite;
}
@keyframes ck-spin {
  to { transform: rotate(360deg); }
}
.ck-seal__dash {
  fill: none;
  stroke: rgba(251, 191, 36, 0.55);
  stroke-width: 1.2;
  stroke-dasharray: 2 6;
}
.ck-seal.is-done .ck-seal__dash { stroke: rgba(52, 211, 153, 0.6); }
.ck-seal.is-locked .ck-seal__dash { stroke: rgba(255, 255, 255, 0.18); }
.ck-seal__arc-text {
  font-size: 9.5px;
  font-weight: 800;
  letter-spacing: 3.2px;
  fill: rgba(255, 255, 255, 0.4);
}
.ck-seal__core {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  width: 8.25rem;
  height: 8.25rem;
  justify-content: center;
  border-radius: 50%;
  border: 2px dashed currentColor;
  opacity: 0.95;
}
.ck-seal.is-locked .ck-seal__core { border-style: dotted; opacity: 0.7; }
.ck-seal__label {
  font-size: 1.15rem;
  font-weight: 850;
  letter-spacing: 0.06em;
}
.ck-seal__spinner {
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  border: 3px solid currentColor;
  border-right-color: transparent;
  animation: ck-spin 0.8s linear infinite;
}
.ck-lock {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  max-width: 15rem;
  padding: 0.45rem 0.8rem;
  border-radius: 0.75rem;
  font-size: 0.75rem;
  text-align: center;
  color: #fde68a;
  background: rgba(245, 158, 11, 0.12);
  border: 1px solid rgba(251, 191, 36, 0.3);
}

/* 金币迸发 */
.ck-burst {
  position: absolute;
  inset: 50% auto auto 50%;
  pointer-events: none;
}
.ck-burst__coin {
  position: absolute;
  left: -0.7rem;
  top: -0.7rem;
  display: grid;
  place-items: center;
  width: 1.4rem;
  height: 1.4rem;
  border-radius: 50%;
  font-size: 0.7rem;
  font-weight: 900;
  color: #78350f;
  background: radial-gradient(circle at 35% 30%, #fff7cc, #fbbf24 55%, #d97706);
  box-shadow: 0 2px 6px rgba(146, 64, 14, 0.45);
  opacity: 0;
  animation: ck-coin 1.1s var(--rx-ease) var(--delay) forwards;
}
@keyframes ck-coin {
  0% { opacity: 0; transform: rotate(var(--a)) translateY(0) scale(0.3); }
  15% { opacity: 1; }
  70% { opacity: 1; transform: rotate(var(--a)) translateY(calc(var(--d) * -1)) scale(var(--s)); }
  100% { opacity: 0; transform: rotate(var(--a)) translateY(calc(var(--d) * -1.15)) scale(calc(var(--s) * 0.8)); }
}
.ck-float {
  position: absolute;
  top: -0.5rem;
  left: 50%;
  transform: translateX(-50%);
  padding: 0.3rem 0.8rem;
  border-radius: 999px;
  font-size: 1rem;
  font-weight: 850;
  font-variant-numeric: tabular-nums;
  color: #3b1d00;
  background: linear-gradient(180deg, #fde68a, #fbbf24);
  box-shadow: 0 10px 24px -10px rgba(245, 158, 11, 0.9);
  white-space: nowrap;
}
.ck-pop-enter-active { transition: all 0.5s var(--rx-spring); }
.ck-pop-leave-active { transition: all 0.4s ease; }
.ck-pop-enter-from { opacity: 0; transform: translate(-50%, 1.5rem) scale(0.6); }
.ck-pop-leave-to { opacity: 0; transform: translate(-50%, -1.5rem); }

/* ---------------- 统计 ---------------- */
.ck-stats {
  display: grid;
  gap: 0.75rem;
}
@media (min-width: 640px) {
  .ck-stats { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
.ck-unit {
  margin-left: 0.25rem;
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0;
  color: var(--rx-faint);
}

/* ---------------- 面板 ---------------- */
.ck-panel {
  padding: 1.4rem;
}
@media (min-width: 640px) {
  .ck-panel { padding: 1.75rem; }
}

/* 连签进度 */
.ck-track {
  position: relative;
  height: 6px;
  margin-bottom: 1.1rem;
  border-radius: 999px;
  overflow: hidden;
  background: var(--rx-line);
}
.ck-track__fill {
  position: absolute;
  inset: 0 auto 0 0;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--rx-accent), var(--ck-gold-2));
  transition: width 0.8s var(--rx-ease);
}
.ck-cycle {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(4.6rem, 1fr));
  gap: 0.6rem;
}
.ck-day {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.45rem;
  padding: 0.8rem 0.4rem 0.7rem;
  border-radius: var(--rx-radius-sm);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line);
  transition: transform 0.3s var(--rx-ease), border-color 0.3s ease;
}
.ck-day__label {
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--rx-faint);
}
.ck-day__icon {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  color: var(--rx-faint);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
}
.ck-day__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.5;
}
.ck-day__reward {
  font-size: 0.72rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--rx-muted);
}
.ck-day.is-bonus {
  border-color: var(--ck-gold-line);
  background: linear-gradient(180deg, var(--ck-gold-soft), transparent);
}
.ck-day.is-bonus .ck-day__icon {
  color: var(--ck-gold-deep);
  background: rgba(251, 191, 36, 0.18);
  border-color: var(--ck-gold-line);
}
.ck-day.is-bonus .ck-day__reward { color: var(--ck-gold-deep); }
.ck-day.is-claimed .ck-day__icon {
  color: var(--rx-on-accent);
  background: var(--rx-accent);
  border-color: transparent;
}
.ck-day.is-today {
  transform: translateY(-2px);
  border-color: var(--rx-accent);
  box-shadow: 0 0 0 3px var(--rx-accent-soft);
}
.ck-day.is-today .ck-day__label { color: var(--rx-accent); }
.ck-day.is-today:not(.is-claimed) .ck-day__icon {
  animation: ck-breathe 1.8s ease-in-out infinite;
}
@keyframes ck-breathe {
  0%, 100% { box-shadow: 0 0 0 0 var(--rx-accent-soft); }
  50% { box-shadow: 0 0 0 6px var(--rx-accent-soft); }
}
html.dark .ck-day.is-bonus .ck-day__icon,
html.dark .ck-day.is-bonus .ck-day__reward { color: #fcd34d; }

/* 日历 */
.ck-month {
  margin: -0.5rem 0 0.9rem;
  font-size: 0.8rem;
  font-weight: 650;
  color: var(--rx-muted);
}
.ck-cal {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 0.35rem;
}
.ck-cal__wd {
  padding-bottom: 0.35rem;
  text-align: center;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  color: var(--rx-faint);
}
.ck-cal__day {
  position: relative;
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  border-radius: 0.7rem;
  font-size: 0.82rem;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  color: var(--rx-ink-2);
  background: var(--rx-surface-2);
  border: 1px solid transparent;
}
.ck-cal__day.is-future { color: var(--rx-faint); background: transparent; }
.ck-cal__day.is-done {
  color: var(--rx-on-accent);
  background: linear-gradient(160deg, var(--rx-accent), color-mix(in srgb, var(--rx-accent) 70%, #000));
  box-shadow: 0 6px 14px -8px var(--rx-accent);
}
.ck-cal__day.is-today {
  border-color: var(--ck-gold);
  box-shadow: 0 0 0 3px var(--ck-gold-soft);
}
.ck-cal__tick {
  position: absolute;
  right: 0.2rem;
  bottom: 0.2rem;
  opacity: 0.85;
}
.ck-cal__day.is-done .ck-cal__num { transform: translateY(-1px); }

/* 规则 */
.ck-rules {
  display: grid;
  gap: 1rem;
}
.ck-rule dt {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  font-size: 0.8rem;
  font-weight: 750;
  color: var(--rx-ink);
}
.ck-rule__icon {
  display: grid;
  place-items: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 0.6rem;
  color: var(--rx-tint-ink);
  background: var(--rx-tint);
}
.ck-rule dd {
  margin: 0.4rem 0 0 2.3rem;
  font-size: 0.85rem;
  color: var(--rx-muted);
}
.ck-rule__list {
  display: grid;
  gap: 0.35rem;
}
.ck-rule__list li::before {
  content: '';
  display: inline-block;
  width: 4px;
  height: 4px;
  margin-right: 0.5rem;
  vertical-align: middle;
  border-radius: 50%;
  background: var(--rx-faint);
}
.ck-rule__bonus {
  display: flex;
  align-items: center;
}
.ck-rule__bonus strong {
  margin-left: auto;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  color: var(--ck-gold-deep);
}
html.dark .ck-rule__bonus strong { color: #fcd34d; }
.ck-note {
  margin-top: 1.25rem;
  padding: 0.75rem 0.9rem;
  border-radius: var(--rx-radius-sm);
  font-size: 0.76rem;
  line-height: 1.6;
  color: var(--rx-muted);
  background: var(--rx-surface-2);
  border: 1px dashed var(--rx-line-strong);
}

/* 记录 */
.ck-history-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.6rem;
  padding: 2.25rem 1rem;
  font-size: 0.85rem;
  color: var(--rx-faint);
}
.ck-history {
  display: grid;
  gap: 0.5rem;
}
.ck-history__item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0.9rem;
  border-radius: var(--rx-radius-sm);
  background: var(--rx-surface-2);
  border: 1px solid var(--rx-line-soft);
  transition: background 0.2s ease, transform 0.2s ease;
}
.ck-history__item:hover {
  background: var(--rx-surface-hover);
  transform: translateX(2px);
}
.ck-history__date {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 3rem;
  height: 3rem;
  flex-shrink: 0;
  border-radius: 0.8rem;
  color: var(--rx-tint-ink);
  background: var(--rx-tint);
}
.ck-history__date strong {
  font-size: 1.05rem;
  font-weight: 800;
  line-height: 1;
}
.ck-history__date small {
  margin-top: 0.15rem;
  font-size: 0.58rem;
  font-weight: 700;
  opacity: 0.75;
}
.ck-history__body {
  min-width: 0;
  flex: 1;
}
.ck-history__title {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--rx-ink);
}
.ck-history__meta {
  margin-top: 0.15rem;
  font-size: 0.75rem;
  color: var(--rx-faint);
}
.ck-history__bonus { color: var(--ck-gold-deep); font-weight: 650; }
html.dark .ck-history__bonus { color: #fcd34d; }
.ck-history__amount {
  font-size: 1rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: #059669;
}
html.dark .ck-history__amount { color: #34d399; }

.ck-pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}
.ck-pager__info {
  font-size: 0.8rem;
  font-variant-numeric: tabular-nums;
  color: var(--rx-faint);
}
.ck-ghost-btn {
  padding: 0.45rem 0.9rem;
  border-radius: 0.7rem;
  font-size: 0.8rem;
  font-weight: 650;
  color: var(--rx-ink-2);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
  transition: background 0.2s ease;
}
.ck-ghost-btn:hover:not(:disabled) { background: var(--rx-surface-hover); }
.ck-ghost-btn:disabled { opacity: 0.45; cursor: not-allowed; }

@media (prefers-reduced-motion: reduce) {
  .ck-seal.is-ready .ck-seal__ring,
  .ck-seal-wrap__halo.is-idle,
  .ck-streak__flame.is-lit,
  .ck-day.is-today .ck-day__icon,
  .ck-skeleton { animation: none; }
  .ck-burst { display: none; }
}
</style>
