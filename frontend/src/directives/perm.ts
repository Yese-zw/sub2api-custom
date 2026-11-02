import type { Directive } from 'vue'
import { useAuthStore } from '@/stores/auth'

/**
 * v-perm="'users.edit'"：运营缺少该操作权限时隐藏元素（管理员始终可见）。
 * 仅为界面体验，真正的拦截由后端按接口校验。
 */
function apply(el: HTMLElement, perm: string | string[]) {
  const auth = useAuthStore()
  const perms = Array.isArray(perm) ? perm : [perm]
  const ok = perms.some((p) => auth.hasPermission(p))
  el.style.display = ok ? '' : 'none'
}

export const vPerm: Directive<HTMLElement, string | string[]> = {
  mounted: (el, binding) => apply(el, binding.value),
  updated: (el, binding) => apply(el, binding.value)
}
