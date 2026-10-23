<template>
  <section class="rx-page-hero" :class="{ 'has-aside': $slots.aside }">
    <div class="min-w-0">
      <span class="rx-page-hero__eyebrow rx-rise" style="--rx-i: 0">
        <span class="rx-page-hero__dot" aria-hidden="true"></span>
        {{ eyebrow }}
      </span>
      <h1 class="rx-page-hero__title rx-rise" style="--rx-i: 1">
        <span class="rx-page-hero__lead">{{ lead }}</span>
        <span class="rx-page-hero__accent">{{ accent }}</span>
      </h1>
      <p v-if="subtitle" class="rx-page-hero__sub rx-rise" style="--rx-i: 2">{{ subtitle }}</p>
      <div v-if="$slots.default" class="rx-page-hero__extra rx-rise" style="--rx-i: 3">
        <slot />
      </div>
    </div>
    <div v-if="$slots.aside" class="rx-page-hero__aside rx-rise" style="--rx-i: 2">
      <slot name="aside" />
    </div>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  eyebrow: string
  lead: string
  accent: string
  subtitle?: string
}>()
</script>

<style scoped>
.rx-page-hero {
  display: grid;
  gap: 1.75rem;
  align-items: end;
  padding: 1.5rem 0.25rem 0.25rem;
}

@media (min-width: 1024px) {
  .rx-page-hero.has-aside {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 3rem;
  }

  .rx-page-hero {
    padding: 2.25rem 0.5rem 0.5rem;
  }
}

.rx-page-hero__eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.8rem 0.35rem 0.65rem;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--rx-muted);
  background: var(--rx-surface);
  border: 1px solid var(--rx-line);
  -webkit-backdrop-filter: var(--rx-blur);
  backdrop-filter: var(--rx-blur);
}

.rx-page-hero__dot {
  position: relative;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: var(--rx-ink);
}

.rx-page-hero__dot::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: var(--rx-ink);
  animation: rx-hero-ping 2.4s cubic-bezier(0, 0, 0.2, 1) infinite;
}

@keyframes rx-hero-ping {
  0% { transform: scale(1); opacity: 0.45; }
  80%, 100% { transform: scale(3); opacity: 0; }
}

.rx-page-hero__title {
  margin-top: 1rem;
  font-size: clamp(1.9rem, 1.3rem + 2.4vw, 3rem);
  line-height: 1.06;
  font-weight: 800;
  letter-spacing: -0.04em;
  color: var(--rx-ink);
}

.rx-page-hero__lead {
  display: block;
}

/* 编辑型强调：衬线斜体 + 入场描出的浅色下划线 */
.rx-page-hero__accent {
  position: relative;
  display: inline-block;
  padding-bottom: 0.12em;
  font-family: var(--rx-display);
  font-style: var(--rx-display-style);
  font-weight: var(--rx-display-weight);
  letter-spacing: -0.02em;
}

.rx-page-hero__accent::after {
  content: '';
  position: absolute;
  left: 0.02em;
  right: 0.02em;
  bottom: 0.02em;
  z-index: -1;
  height: 0.2em;
  border-radius: 9999px;
  background: var(--rx-tint);
  transform: scaleX(0);
  transform-origin: left;
  animation: rx-hero-underline 1.1s var(--rx-ease) 0.45s forwards;
}

@keyframes rx-hero-underline {
  to { transform: scaleX(1); }
}

.rx-page-hero__sub {
  margin-top: 0.9rem;
  max-width: 34rem;
  font-size: 0.95rem;
  line-height: 1.7;
  color: var(--rx-muted);
}

.rx-page-hero__extra {
  margin-top: 1.5rem;
}

.rx-page-hero__aside {
  min-width: 0;
}
</style>
