<template>
  <Teleport to="body">
    <Transition name="ann-letter">
      <div
        v-if="displayedAnnouncement"
        class="ann-overlay rx-theme fixed inset-0 z-[120] flex items-start justify-center overflow-y-auto p-4 pt-[8vh]"
      >
        <article class="ann-letter ann-letter--popup" @click.stop>
          <!-- 新公告：右上角一枚缓慢旋转的「新」字印章 -->
          <span v-if="!preview" class="ann-seal" aria-hidden="true">NEW</span>
          <header class="ann-letter__head">
            <div class="min-w-0 flex-1">
              <p class="ann-eyebrow">
                <span class="ann-eyebrow__dot is-live" aria-hidden="true"></span>
                {{ t('announcements.title') }}
                <span aria-hidden="true">·</span>
                {{ t('announcements.unread') }}
              </p>
              <h2 class="ann-letter__title">{{ displayedAnnouncement.title }}</h2>
              <p class="ann-letter__meta">
                <time>{{ formatRelativeWithDateTime(displayedAnnouncement.created_at) }}</time>
              </p>
            </div>
          </header>

          <div class="ann-letter__body">
            <div
              class="markdown-body prose prose-sm max-w-none dark:prose-invert"
              v-html="renderedContent"
            ></div>
          </div>

          <footer class="ann-letter__foot !justify-end">
            <button
              @click="handleDismiss"
              data-testid="announcement-popup-dismiss"
              class="ann-btn ann-btn--ink"
            >
              <svg v-if="preview" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
              <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              {{ preview ? t('common.close') : t('announcements.markRead') }}
            </button>
          </footer>
        </article>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeWithDateTime } from '@/utils/format'
import type { Announcement, UserAnnouncement } from '@/types'
import '@/styles/announcement-markdown.css'
import '@/styles/recharge-glass.css'
import '@/styles/announcement-glass.css'

type PreviewAnnouncement = Pick<Announcement | UserAnnouncement, 'title' | 'content' | 'created_at'>

const props = withDefaults(defineProps<{
  announcement?: PreviewAnnouncement | null
  preview?: boolean
}>(), {
  announcement: null,
  preview: false,
})

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const displayedAnnouncement = computed(() => (
  props.preview ? props.announcement : announcementStore.currentPopup
))

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = displayedAnnouncement.value?.content
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

function handleDismiss() {
  if (props.preview) {
    emit('close')
    return
  }
  announcementStore.dismissPopup()
}

// Manage body overflow — only set, never unset (bell component handles restore)
watch(
  displayedAnnouncement,
  (popup) => {
    if (popup) {
      document.body.style.overflow = 'hidden'
    } else if (props.preview) {
      document.body.style.overflow = ''
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (props.preview) {
    document.body.style.overflow = ''
  }
})
</script>

<style scoped>
/* 新公告印章：墨色描边圆章，外圈虚线缓慢旋转 */
.ann-seal {
  position: absolute;
  top: 1.4rem;
  right: 1.5rem;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 3.6rem;
  height: 3.6rem;
  border-radius: 9999px;
  border: 1.5px solid var(--rx-ink);
  font-size: 0.62rem;
  font-weight: 900;
  letter-spacing: 0.18em;
  color: var(--rx-ink);
  transform: rotate(-12deg);
}

.ann-seal::before {
  content: '';
  position: absolute;
  inset: 3px;
  border-radius: inherit;
  border: 1px dashed var(--rx-line-strong);
  animation: ann-seal-spin 16s linear infinite;
}

@keyframes ann-seal-spin {
  to { transform: rotate(360deg); }
}

/* 有印章时标题给右侧留出空间 */
.ann-letter--popup .ann-letter__head {
  padding-right: 6rem;
}

@media (prefers-reduced-motion: reduce) {
  .ann-seal::before {
    animation: none;
  }
}
</style>
