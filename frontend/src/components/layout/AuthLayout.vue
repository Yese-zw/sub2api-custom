<template>
  <div class="auth-shell relative min-h-screen overflow-hidden">
    <div class="auth-grid absolute inset-0"></div>
    <div class="auth-scanline pointer-events-none absolute inset-x-0 top-0"></div>

    <!-- 单一卡片：左侧介绍 + 右侧表单，高度固定不随登录/注册切换跳动 -->
    <div class="auth-stage">
      <div class="auth-card">
        <div class="auth-card-grid">
          <!-- 左：系统介绍 -->
          <aside class="auth-intro">
            <div class="auth-intro-glow" aria-hidden="true"></div>
            <div class="auth-intro-rule" aria-hidden="true"></div>

            <div class="auth-intro-inner">
              <p class="auth-eyebrow">
                <span class="auth-eyebrow-dot" aria-hidden="true"></span>
                {{ t('auth.brandPanel.eyebrow') }}
              </p>

              <h2 class="auth-headline">{{ t('auth.brandPanel.headline') }}</h2>
              <p class="auth-lede">{{ t('auth.brandPanel.lede') }}</p>

              <span class="auth-hairline" aria-hidden="true"></span>

              <ul class="auth-features">
                <li
                  v-for="(item, index) in brandFeatures"
                  :key="item.title"
                  class="auth-feature"
                  :style="{ '--i': index }"
                >
                  <span class="auth-index">{{ String(index + 1).padStart(2, '0') }}</span>
                  <div class="auth-feature-copy">
                    <b>{{ item.title }}</b>
                    <span>{{ item.desc }}</span>
                  </div>
                </li>
              </ul>

              <p class="auth-footnote">{{ t('auth.brandPanel.footnote') }}</p>
            </div>
          </aside>

          <!-- 右：表单 -->
          <div class="auth-form-pane">
            <div class="auth-form-scroll">
              <slot />
            </div>

            <div v-if="$slots.footer" class="auth-footer">
              <slot name="footer" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'

const { t } = useI18n()

const appStore = useAppStore()

const brandFeatures = computed(() => [
  {
    title: t('auth.brandPanel.features.gateway.title'),
    desc: t('auth.brandPanel.features.gateway.desc'),
  },
  {
    title: t('auth.brandPanel.features.metering.title'),
    desc: t('auth.brandPanel.features.metering.desc'),
  },
  {
    title: t('auth.brandPanel.features.stable.title'),
    desc: t('auth.brandPanel.features.stable.desc'),
  },
])

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style>
html.pixel-ui .auth-shell {
  background: var(--pixel-canvas);
  color: var(--pixel-ink);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
}

html.pixel-ui .auth-grid {
  background-color: var(--pixel-canvas);
  background-image: linear-gradient(var(--pixel-grid) 1px, transparent 1px), linear-gradient(90deg, var(--pixel-grid) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.95), rgba(0, 0, 0, 0.45));
}

html.pixel-ui .auth-scanline { height: 3px; background: var(--pixel-cyan); box-shadow: 0 0 18px var(--pixel-glow); }
html.pixel-ui .auth-shell .input { border-radius: 0; border-color: var(--pixel-line); background: var(--pixel-input); color: var(--pixel-ink); font-family: inherit; }
html.pixel-ui .auth-shell .input:focus { border-color: var(--pixel-cyan); box-shadow: 3px 3px 0 var(--pixel-glow-soft); }
html.pixel-ui .auth-shell .input-label { color: var(--pixel-muted); font-family: inherit; }
html.pixel-ui .auth-shell .btn { border-radius: 4px; font-family: inherit; }
html.pixel-ui .auth-shell .btn-primary { background: var(--pixel-cyan); color: var(--pixel-cyan-ink); box-shadow: 4px 4px 0 var(--pixel-button-shadow); }
html.pixel-ui .auth-shell .btn-primary:hover { background: var(--pixel-cyan-bright); box-shadow: 6px 6px 0 var(--pixel-button-shadow); transform: translate(-2px, -2px); }
html.pixel-ui .auth-shell .btn-secondary { border-color: var(--pixel-line); background: var(--pixel-surface-alt); color: var(--pixel-ink); }
html.pixel-ui .auth-shell a { color: var(--pixel-cyan) !important; }

/* 单一卡片：像素风保留硬边与裁切角 */
html.pixel-ui .auth-card {
  border: 1px solid var(--pixel-line-strong);
  background: var(--pixel-surface);
  box-shadow: 8px 8px 0 var(--pixel-shadow), 0 0 40px var(--pixel-glow-soft);
  clip-path: polygon(0 12px, 12px 0, 100% 0, 100% calc(100% - 12px), calc(100% - 12px) 100%, 0 100%);
}
html.pixel-ui .auth-intro {
  background: var(--pixel-surface-alt);
  border-color: var(--pixel-line-strong);
}
html.pixel-ui .auth-intro-glow,
html.pixel-ui .auth-intro-rule,
html.pixel-ui .auth-hairline { display: none; }
html.pixel-ui .auth-form-pane { background: var(--pixel-surface); }
html.pixel-ui .auth-eyebrow { color: var(--pixel-cyan); }
html.pixel-ui .auth-eyebrow-dot { background: var(--pixel-cyan); box-shadow: 0 0 0 4px var(--pixel-glow-soft); }
html.pixel-ui .auth-headline { color: var(--pixel-ink); text-shadow: 3px 3px 0 var(--pixel-text-shadow); }
html.pixel-ui .auth-lede { color: var(--pixel-muted); }
html.pixel-ui .auth-index { color: var(--pixel-cyan); border-color: var(--pixel-line); border-radius: 0; }
html.pixel-ui .auth-feature + .auth-feature { border-color: var(--pixel-line); }
html.pixel-ui .auth-feature-copy b { color: var(--pixel-ink); }
html.pixel-ui .auth-feature-copy span { color: var(--pixel-muted); }
html.pixel-ui .auth-footnote { color: var(--pixel-muted); border-color: var(--pixel-line); }

/* ---------- 编辑型 ---------- */
html.editorial-ui .auth-shell {
  background: var(--ed-canvas);
  color: var(--ed-ink);
}

html.editorial-ui .auth-grid {
  background-color: var(--ed-canvas);
  background-image:
    radial-gradient(1200px 560px at 8% -10%, rgba(255, 255, 255, 0.94), transparent 60%),
    radial-gradient(1000px 520px at 96% 8%, rgba(220, 233, 255, 0.34), transparent 64%);
  mask-image: none;
}

html.editorial-ui .auth-scanline { height: 2px; background: var(--ed-soft); box-shadow: none; }

html.editorial-ui .auth-card {
  border: 1px solid var(--ed-line);
  border-radius: var(--ed-radius);
  background: var(--ed-grad-surface);
  box-shadow: var(--ed-shadow-lg), var(--ed-hairline);
}
html.editorial-ui.dark .auth-card { box-shadow: var(--ed-shadow-lg), var(--ed-hairline-dark); }

/* 左半：深色编辑面板 */
html.editorial-ui .auth-intro {
  background:
    radial-gradient(720px 420px at 82% 6%, rgba(216, 227, 243, 0.14), transparent 62%),
    linear-gradient(168deg, #1c1c1a 0%, #0f0f0e 58%, #131318 100%);
  border-color: var(--ed-line);
}
html.editorial-ui .auth-intro-glow {
  position: absolute;
  top: -30%;
  right: -16%;
  width: 62%;
  aspect-ratio: 1;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(220, 233, 255, 0.2), transparent 68%);
  filter: blur(10px);
  pointer-events: none;
}
html.editorial-ui .auth-intro-rule {
  position: absolute;
  inset-inline: 2.25rem;
  top: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.26), transparent);
  pointer-events: none;
}
html.editorial-ui .auth-eyebrow { color: rgba(244, 242, 236, 0.62); }
html.editorial-ui .auth-eyebrow-dot { background: #d8e3f3; box-shadow: 0 0 0 4px rgba(216, 227, 243, 0.15); }
html.editorial-ui .auth-headline { color: #fbfaf7; }
html.editorial-ui .auth-lede { color: rgba(244, 242, 236, 0.6); }
html.editorial-ui .auth-hairline { background: rgba(255, 255, 255, 0.09); }
html.editorial-ui .auth-index { color: rgba(216, 227, 243, 0.8); border-color: rgba(255, 255, 255, 0.14); }
html.editorial-ui .auth-feature + .auth-feature { border-color: rgba(255, 255, 255, 0.08); }
html.editorial-ui .auth-feature-copy b { color: #f4f2ec; }
html.editorial-ui .auth-feature-copy span { color: rgba(244, 242, 236, 0.5); }
html.editorial-ui .auth-footnote { color: rgba(244, 242, 236, 0.42); border-color: rgba(255, 255, 255, 0.08); }
html.editorial-ui .auth-form-pane { background: var(--ed-grad-surface); }

html.editorial-ui .auth-shell .input { border-radius: var(--ed-radius-sm); }
html.editorial-ui .auth-shell .btn { border-radius: 999px; }
html.editorial-ui .auth-shell a { color: var(--ed-ink) !important; text-decoration: underline; text-underline-offset: 3px; }

/* ---------- 原始主题 ---------- */
html:not(.pixel-ui):not(.editorial-ui) .auth-shell { background: #f9fafb; }
html:not(.pixel-ui):not(.editorial-ui).dark .auth-shell { background: #020617; }
html:not(.pixel-ui):not(.editorial-ui) .auth-grid { background: linear-gradient(135deg, rgba(20, 184, 166, 0.08), transparent 55%); }
html:not(.pixel-ui):not(.editorial-ui) .auth-scanline { display: none; }
html:not(.pixel-ui):not(.editorial-ui) .auth-card {
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.7);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.08);
}
html:not(.pixel-ui):not(.editorial-ui).dark .auth-card { border-color: rgba(51, 65, 85, 0.5); background: rgba(30, 41, 59, 0.7); }
html:not(.pixel-ui):not(.editorial-ui) .auth-intro {
  background: linear-gradient(160deg, #0f172a 0%, #134e4a 100%);
  border-color: rgba(255, 255, 255, 0.08);
}
html:not(.pixel-ui):not(.editorial-ui) .auth-intro-glow {
  position: absolute;
  top: -28%;
  right: -14%;
  width: 58%;
  aspect-ratio: 1;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(153, 246, 228, 0.22), transparent 68%);
  pointer-events: none;
}
html:not(.pixel-ui):not(.editorial-ui) .auth-intro-rule { display: none; }
html:not(.pixel-ui):not(.editorial-ui) .auth-eyebrow,
html:not(.pixel-ui):not(.editorial-ui) .auth-lede,
html:not(.pixel-ui):not(.editorial-ui) .auth-feature-copy span { color: rgba(248, 250, 252, 0.68); }
html:not(.pixel-ui):not(.editorial-ui) .auth-headline { color: #f8fafc; }
html:not(.pixel-ui):not(.editorial-ui) .auth-index { color: #99f6e4; border-color: rgba(255, 255, 255, 0.18); }
html:not(.pixel-ui):not(.editorial-ui) .auth-feature + .auth-feature { border-color: rgba(255, 255, 255, 0.1); }
html:not(.pixel-ui):not(.editorial-ui) .auth-feature-copy b { color: #f8fafc; }
html:not(.pixel-ui):not(.editorial-ui) .auth-footnote { color: rgba(248, 250, 252, 0.55); border-color: rgba(255, 255, 255, 0.12); }
html:not(.pixel-ui):not(.editorial-ui) .auth-hairline { background: rgba(255, 255, 255, 0.14); }
html:not(.pixel-ui):not(.editorial-ui) .auth-form-pane { background: transparent; }
</style>

<style scoped>
/* 卡片固定高度：登录与注册切换时不再跳动 */
.auth-stage {
  position: relative;
  z-index: 10;
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 2rem 1.5rem;
}

.auth-card {
  width: 100%;
  max-width: 62rem;
  overflow: hidden;
}

.auth-card-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

/* ---------- 左半：系统介绍 ---------- */
.auth-intro {
  position: relative;
  overflow: hidden;
  display: none;
}

.auth-intro-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
  height: 100%;
  justify-content: center;
  padding: 2.5rem;
}

.auth-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.auth-eyebrow-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  flex: 0 0 auto;
}

/* 文案克制：主标题约 27px，正文 13px */
.auth-headline {
  margin: 0;
  font-family: var(--ed-serif, Georgia, 'Times New Roman', 'Songti SC', 'STSong', serif);
  font-size: 1.7rem;
  font-weight: 500;
  line-height: 1.22;
  letter-spacing: -0.035em;
  text-wrap: balance;
}

.auth-lede {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.85;
  max-width: 28rem;
}

.auth-hairline {
  display: block;
  height: 1px;
  width: 3rem;
  margin: 0.35rem 0 0.15rem;
}

.auth-features {
  list-style: none;
  margin: 0.2rem 0 0;
  padding: 0;
  display: grid;
  gap: 0.95rem;
}

.auth-feature {
  display: grid;
  grid-template-columns: 1.9rem 1fr;
  gap: 0.85rem;
  align-items: start;
  padding-top: 0.95rem;
  animation: auth-rise 0.6s var(--ed-ease, cubic-bezier(0.22, 1, 0.36, 1)) both;
  animation-delay: calc(120ms + var(--i, 0) * 90ms);
}

.auth-feature + .auth-feature {
  border-top: 1px solid transparent;
}

.auth-index {
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
  border: 1px solid transparent;
  border-radius: 8px;
  width: 1.9rem;
  height: 1.9rem;
  display: grid;
  place-items: center;
}

.auth-feature-copy b {
  display: block;
  font-size: 0.8125rem;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.auth-feature-copy span {
  display: block;
  margin-top: 0.3rem;
  font-size: 0.75rem;
  line-height: 1.72;
}

.auth-footnote {
  margin: 0.5rem 0 0;
  padding-top: 1.1rem;
  font-size: 0.6875rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

/* ---------- 右半：表单 ---------- */
.auth-form-pane {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* 表单内容不再产生滚动条；高度不足时由紧凑排版保证容纳 */
.auth-form-scroll {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 2.5rem;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.auth-footer {
  flex-shrink: 0;
  padding: 0 2.5rem 2rem;
  text-align: center;
  font-size: 0.875rem;
}

@keyframes auth-rise {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ---------- 桌面端：左右分栏 + 固定高度 ---------- */
@media (min-width: 1024px) {
  .auth-card {
    /* 固定高度：登录 / 注册切换时卡片尺寸完全一致 */
    height: 40rem;
    max-height: calc(100vh - 4rem);
  }

  .auth-card-grid {
    grid-template-columns: minmax(0, 1.05fr) minmax(0, 0.95fr);
    height: 100%;
  }

  .auth-intro {
    display: block;
    border-right: 1px solid var(--ed-line, #e5e5e5);
  }

  .auth-intro-inner {
    animation: auth-rise 0.7s var(--ed-ease, cubic-bezier(0.22, 1, 0.36, 1)) both;
  }

  .auth-form-scroll {
    padding: 2.25rem 2.5rem;
  }

  /* 紧凑排版：压缩表单纵向间距与控件高度，保证注册页无需滚动即可完整显示 */
  .auth-form-scroll :deep(.space-y-6) { margin-top: 0; row-gap: 1rem; }
  .auth-form-scroll :deep(.space-y-5) { row-gap: 0.9rem; }
  .auth-form-scroll :deep(h2) { font-size: 1.375rem; }
  .auth-form-scroll :deep(.input) { padding-top: 0.55rem; padding-bottom: 0.55rem; }
  .auth-form-scroll :deep(.input-hint) { margin-top: 0.3rem; font-size: 0.6875rem; }

  .auth-footer {
    padding: 0 2.5rem 1.75rem;
  }
}

@media (max-width: 1023px) {
  .auth-card { height: auto; }
  .auth-form-scroll { overflow-y: visible; padding: 2rem 1.5rem; }
  .auth-footer { padding: 0 1.5rem 1.5rem; }
}

@media (max-width: 640px) {
  .auth-stage { padding: 1rem; }
}

@media (prefers-reduced-motion: reduce) {
  .auth-feature,
  .auth-intro-inner { animation: none !important; }
}
</style>
