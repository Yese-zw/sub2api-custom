<template>
  <!-- 个人资料左栏：身份卡 + 分区导航（滚动时高亮当前分区） -->
  <aside class="id-rail">
    <div class="id-card">
      <span class="id-card__sheen" aria-hidden="true"></span>
      <div class="id-avatar">
        <span class="id-avatar__ring" aria-hidden="true"></span>
        <span class="id-avatar__face">
          <img v-if="avatarUrl" :src="avatarUrl" :alt="displayName" />
          <template v-else>{{ initial }}</template>
        </span>
        <span class="id-avatar__status" :class="{ 'is-off': user?.status !== 'active' }" aria-hidden="true"></span>
      </div>
      <p class="id-card__name">{{ displayName }}</p>
      <p v-if="email" class="id-card__email">{{ email }}</p>
      <p class="id-card__serial">•••• {{ memberSerial }}</p>
      <div class="id-card__badges">
        <span>{{ user?.role === 'admin' ? t('profile.administrator') : t('profile.user') }}</span>
        <span :class="{ 'is-off': user?.status !== 'active' }">{{ user?.status === 'active' ? t('common.active') : t('common.disabled') }}</span>
      </div>
      <dl class="id-card__stats">
        <div>
          <dt>{{ t('profile.accountBalance') }}</dt>
          <dd>{{ formatCurrency(user?.balance || 0) }}</dd>
        </div>
        <div>
          <dt>{{ t('profile.concurrencyLimit') }}</dt>
          <dd>{{ user?.concurrency || 0 }}</dd>
        </div>
        <div>
          <dt>{{ t('profile.memberSince') }}</dt>
          <dd>{{ memberSince }}</dd>
        </div>
      </dl>
    </div>

    <nav class="id-nav" :aria-label="t('profile.hero.sections')">
      <span class="id-nav__track" aria-hidden="true">
        <span class="id-nav__thumb" :style="{ '--idx': activeIndex, '--count': sections.length }"></span>
      </span>
      <a
        v-for="(section, index) in sections"
        :key="section.id"
        :href="`#${section.id}`"
        class="id-nav__item"
        :class="{ 'is-active': activeIndex === index }"
        @click.prevent="scrollTo(section.id)"
      >
        <span class="id-nav__index">{{ String(index + 1).padStart(2, '0') }}</span>
        {{ section.label }}
      </a>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { User } from '@/types'

const props = defineProps<{
  user: User | null
  sections: { id: string; label: string }[]
}>()

const { t } = useI18n()

// 余额统一按美元两位小数展示（与顶栏一致）
function formatCurrency(value: number): string {
  return `$${(Number(value) || 0).toFixed(2)}`
}

const displayName = computed(() => props.user?.username?.trim() || props.user?.email?.trim() || t('profile.user'))
const initial = computed(() => displayName.value.charAt(0).toUpperCase() || 'U')
const avatarUrl = computed(() => props.user?.avatar_url?.trim() || '')
// 未绑定邮箱时后端用 .invalid 占位，不展示
const email = computed(() => {
  const raw = props.user?.email?.trim() || ''
  return raw.endsWith('.invalid') ? '' : raw
})

const memberSerial = computed(() => String(props.user?.id || 0).padStart(4, '0'))

const memberSince = computed(() => {
  const raw = props.user?.created_at?.trim()
  if (!raw) return '-'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return '-'
  // 固定为 YYYY-MM，三列窄格内不截断
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
})

// ---------- 分区导航 ----------
const activeIndex = ref(0)
let observer: IntersectionObserver | null = null

function scrollTo(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(
    (entries) => {
      const visible = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
      if (visible.length === 0) return
      const idx = props.sections.findIndex((s) => s.id === visible[0].target.id)
      if (idx >= 0) activeIndex.value = idx
    },
    { rootMargin: '-20% 0px -55% 0px' },
  )
  for (const s of props.sections) {
    const el = document.getElementById(s.id)
    if (el) observer.observe(el)
  }
})

onBeforeUnmount(() => observer?.disconnect())
</script>

<style scoped>
.id-rail {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

@media (min-width: 1024px) {
  .id-rail {
    position: sticky;
    top: 5.5rem;
  }
}

/* ---------- 身份卡 ---------- */
.id-card {
  position: relative;
  overflow: hidden;
  padding: 1.75rem 1.5rem 1.35rem;
  border-radius: var(--rx-radius-lg);
  text-align: center;
  color: var(--rx-card-ink);
  background: var(--rx-card-bg);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12), inset 0 0 0 1px var(--rx-card-line), var(--rx-shadow-lg);
  animation: rx-rise 0.7s var(--rx-ease) both;
}

.id-card__sheen {
  pointer-events: none;
  position: absolute;
  inset: -40%;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, var(--rx-foil) 60deg, transparent 130deg, transparent 220deg, var(--rx-foil) 290deg, transparent 350deg);
  mix-blend-mode: soft-light;
  animation: id-spin 20s linear infinite;
}

@keyframes id-spin {
  to { --rx-angle: 360deg; }
}

.id-avatar {
  position: relative;
  width: 6rem;
  height: 6rem;
  margin: 0 auto;
}

/* 头像外圈：一段缓慢旋转的渐隐弧 */
.id-avatar__ring {
  position: absolute;
  inset: -6px;
  border-radius: 9999px;
  background: conic-gradient(from var(--rx-angle), transparent 0deg, var(--rx-card-ink) 90deg, transparent 180deg);
  -webkit-mask: radial-gradient(farthest-side, transparent calc(100% - 2px), #000 calc(100% - 1.5px));
  mask: radial-gradient(farthest-side, transparent calc(100% - 2px), #000 calc(100% - 1.5px));
  opacity: 0.7;
  animation: id-spin 6s linear infinite;
}

.id-avatar__face {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 9999px;
  font-size: 2.2rem;
  font-weight: 800;
  color: #11110f;
  background: var(--rx-card-ink);
  box-shadow: 0 0 0 4px rgba(255, 255, 255, 0.06);
}

.id-avatar__face img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.id-avatar__status {
  position: absolute;
  right: 0.35rem;
  bottom: 0.35rem;
  width: 0.95rem;
  height: 0.95rem;
  border-radius: 9999px;
  background: #22c55e;
  box-shadow: 0 0 0 3px #1a1a17;
}

.id-avatar__status.is-off {
  background: #9ca3af;
}

.id-card__name {
  position: relative;
  margin-top: 1rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 1.6rem;
  font-weight: var(--rx-display-weight);
  letter-spacing: -0.02em;
}

.id-card__serial {
  position: relative;
  margin-top: 0.45rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.72rem;
  letter-spacing: 0.22em;
  opacity: 0.45;
}

.id-card__email {
  position: relative;
  margin-top: 0.15rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8rem;
  opacity: 0.6;
}

.id-card__badges {
  position: relative;
  display: flex;
  justify-content: center;
  gap: 0.4rem;
  margin-top: 0.85rem;
}

.id-card__badges span {
  padding: 0.12rem 0.6rem;
  border-radius: 9999px;
  font-size: 0.68rem;
  font-weight: 700;
  border: 1px solid var(--rx-card-line);
}

.id-card__badges span.is-off {
  opacity: 0.6;
}

.id-card__stats {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--rx-card-line);
}

.id-card__stats div + div {
  border-left: 1px solid var(--rx-card-line);
}

.id-card__stats dt {
  font-size: 0.58rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  opacity: 0.55;
}

.id-card__stats dd {
  margin-top: 0.25rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 0.25rem;
  font-size: 0.92rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

/* ---------- 分区导航 ---------- */
.id-nav {
  position: relative;
  display: none;
  flex-direction: column;
  padding: 0.5rem 0.5rem 0.5rem 1.5rem;
  border-radius: var(--rx-radius-lg);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
  box-shadow: var(--rx-highlight), var(--rx-shadow);
  animation: rx-rise 0.7s var(--rx-ease) both 0.12s;
}

@media (min-width: 1024px) {
  .id-nav {
    display: flex;
  }
}

.id-nav__track {
  position: absolute;
  left: 0.85rem;
  top: 0.75rem;
  bottom: 0.75rem;
  width: 2px;
  border-radius: 9999px;
  background: var(--rx-line);
}

/* 滑块：按当前分区序号在轨道上平移 */
.id-nav__thumb {
  position: absolute;
  left: 0;
  top: 0;
  width: 2px;
  height: calc(100% / var(--count));
  border-radius: 9999px;
  background: var(--rx-accent);
  transform: translateY(calc(var(--idx) * 100%));
  transition: transform 0.5s var(--rx-spring);
}

.id-nav__item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.65rem 0.75rem;
  border-radius: var(--rx-radius-sm);
  font-size: 0.86rem;
  font-weight: 600;
  color: var(--rx-muted);
  transition: color 0.2s ease, background 0.2s ease, transform 0.3s var(--rx-ease);
}

.id-nav__item:hover {
  color: var(--rx-ink);
  background: var(--rx-surface-2);
}

.id-nav__item.is-active {
  color: var(--rx-ink);
  font-weight: 750;
  transform: translateX(3px);
}

.id-nav__index {
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-size: 0.85rem;
  color: var(--rx-faint);
}

@media (prefers-reduced-motion: reduce) {
  .id-card__sheen,
  .id-avatar__ring {
    animation: none;
  }
}
</style>
