<template>
  <!-- 玻璃风页面舞台：主题令牌 + 羽化背景层，样式见 recharge-glass.css -->
  <div class="rx-stage" :class="wide ? 'w-full' : 'mx-auto max-w-[1180px]'">
    <div class="rx-aurora" aria-hidden="true">
      <span class="rx-aurora__blob rx-aurora__blob--1"></span>
      <span class="rx-aurora__blob rx-aurora__blob--2"></span>
      <span class="rx-aurora__blob rx-aurora__blob--3"></span>
      <span class="rx-aurora__grid"></span>
      <span class="rx-aurora__grain"></span>
    </div>
    <div v-if="bare" class="rx-stage__bare">
      <slot />
    </div>
    <div v-else class="space-y-6 pb-8">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/recharge-glass.css'

defineProps<{
  /** 占满主区域宽度（表格类页面） */
  wide?: boolean
  /** 不包 space-y 容器，由页面自身布局（如 TablePageLayout）控制间距与高度 */
  bare?: boolean
}>()
</script>
