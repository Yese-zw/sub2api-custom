<template>
  <div class="auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <div class="auth-grid absolute inset-0"></div>
    <div class="auth-scanline pointer-events-none absolute inset-x-0 top-0"></div>
    <div class="auth-coordinate auth-coordinate-top pointer-events-none">SYS//AUTH-01</div>
    <div class="auth-coordinate auth-coordinate-bottom pointer-events-none">SECURE TERMINAL</div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="auth-brand mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="auth-logo mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="auth-brand-name mb-2 text-3xl font-bold">
            {{ siteName }}
          </h1>
          <p class="auth-brand-subtitle text-sm text-gray-500 dark:text-dark-400">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div class="auth-panel p-8">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="auth-copyright mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

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
html.pixel-ui .auth-coordinate { position: absolute; color: var(--pixel-muted); font-size: 10px; }
html.pixel-ui .auth-coordinate-top { top: 18px; left: 22px; }
html.pixel-ui .auth-coordinate-bottom { right: 22px; bottom: 18px; }
html.pixel-ui .auth-logo { border: 2px solid var(--pixel-cyan); background: var(--pixel-surface); box-shadow: 5px 5px 0 var(--pixel-shadow), 0 0 22px var(--pixel-glow); }
html.pixel-ui .auth-brand-name { color: var(--pixel-ink); text-shadow: 3px 3px 0 var(--pixel-text-shadow); }
html.pixel-ui .auth-brand-subtitle { color: var(--pixel-muted) !important; }
html.pixel-ui .auth-panel { border: 1px solid var(--pixel-line-strong); background: var(--pixel-surface); box-shadow: 7px 7px 0 var(--pixel-shadow), 0 0 30px var(--pixel-glow-soft); clip-path: polygon(0 10px, 10px 0, 100% 0, 100% calc(100% - 10px), calc(100% - 10px) 100%, 0 100%); }
html.pixel-ui .auth-shell .input { border-radius: 0; border-color: var(--pixel-line); background: var(--pixel-input); color: var(--pixel-ink); font-family: inherit; }
html.pixel-ui .auth-shell .input:focus { border-color: var(--pixel-cyan); box-shadow: 3px 3px 0 var(--pixel-glow-soft); }
html.pixel-ui .auth-shell .input-label { color: var(--pixel-muted); font-family: inherit; }
html.pixel-ui .auth-shell .btn { border-radius: 4px; font-family: inherit; }
html.pixel-ui .auth-shell .btn-primary { background: var(--pixel-cyan); color: var(--pixel-cyan-ink); box-shadow: 4px 4px 0 var(--pixel-button-shadow); }
html.pixel-ui .auth-shell .btn-primary:hover { background: var(--pixel-cyan-bright); box-shadow: 6px 6px 0 var(--pixel-button-shadow); transform: translate(-2px, -2px); }
html.pixel-ui .auth-shell .btn-secondary { border-color: var(--pixel-line); background: var(--pixel-surface-alt); color: var(--pixel-ink); }
html.pixel-ui .auth-shell a { color: var(--pixel-cyan) !important; }
html.pixel-ui .auth-copyright { color: var(--pixel-muted) !important; }

html:not(.pixel-ui) .auth-shell { background: #f9fafb; }
html:not(.pixel-ui).dark .auth-shell { background: #020617; }
html:not(.pixel-ui) .auth-grid { background: linear-gradient(135deg, rgba(20, 184, 166, 0.08), transparent 55%); }
html:not(.pixel-ui) .auth-scanline,
html:not(.pixel-ui) .auth-coordinate { display: none; }
html:not(.pixel-ui) .auth-logo { border-radius: 1rem; box-shadow: 0 10px 20px rgba(20, 184, 166, 0.22); }
html:not(.pixel-ui) .auth-panel { border: 1px solid rgba(255, 255, 255, 0.2); border-radius: 1rem; background: rgba(255, 255, 255, 0.7); box-shadow: 0 8px 32px rgba(0, 0, 0, 0.08); }
html:not(.pixel-ui).dark .auth-panel { border-color: rgba(51, 65, 85, 0.5); background: rgba(30, 41, 59, 0.7); }

@media (max-width: 640px) {
  .auth-coordinate { display: none; }
  .auth-panel { padding: 1.5rem; }
}
</style>
