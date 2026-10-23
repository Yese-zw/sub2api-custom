import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

/**
 * 数字缓动：源值变化时用 easeOutExpo 平滑过渡到目标值。
 * 首次取值直接落定（不从 0 滚动），偏好减少动态效果或无 rAF 环境（SSR / 测试）时同样直接落定。
 */
export function useTweenedNumber(source: Ref<number>, duration = 700): Ref<number> {
  const display = ref(Number.isFinite(source.value) ? source.value : 0)
  let frame = 0

  const canAnimate = () =>
    typeof window !== 'undefined'
    && typeof window.requestAnimationFrame === 'function'
    && !window.matchMedia?.('(prefers-reduced-motion: reduce)').matches

  watch(source, (next) => {
    const target = Number.isFinite(next) ? next : 0
    if (frame) cancelAnimationFrame(frame)
    if (!canAnimate()) {
      display.value = target
      return
    }
    const from = display.value
    const start = performance.now()
    const step = (now: number) => {
      const p = Math.min(1, (now - start) / duration)
      const eased = p === 1 ? 1 : 1 - 2 ** (-10 * p)
      display.value = from + (target - from) * eased
      frame = p < 1 ? requestAnimationFrame(step) : 0
    }
    frame = requestAnimationFrame(step)
  })

  onBeforeUnmount(() => {
    if (frame) cancelAnimationFrame(frame)
  })

  return display
}
