<template>
  <div>
    <!-- 铃铛按钮：有未读时轻摆 + 墨色计数 -->
    <button
      @click="openModal"
      class="ann-bell relative flex h-9 w-9 items-center justify-center rounded-full"
      :class="{ 'has-unread': unreadCount > 0 }"
      :aria-label="t('announcements.title')"
    >
      <span class="ann-bell__icon"><Icon name="bell" size="md" /></span>
      <span v-if="unreadCount > 0" class="ann-bell__count">{{ unreadCount > 9 ? '9+' : unreadCount }}</span>
    </button>

    <!-- 公告抽屉：右侧滑入 -->
    <Teleport to="body">
      <Transition name="ann-drawer">
        <div
          v-if="isModalOpen"
          class="ann-overlay rx-theme fixed inset-0 z-[100]"
          @click="closeModal"
        >
          <aside class="ann-drawer" role="dialog" :aria-label="t('announcements.title')" @click.stop>
            <header class="ann-drawer__head">
              <div class="min-w-0">
                <p class="ann-eyebrow">
                  <span class="ann-eyebrow__dot" :class="{ 'is-live': unreadCount > 0 }" aria-hidden="true"></span>
                  {{ unreadCount > 0 ? `${unreadCount} ${t('announcements.unread')}` : t('announcements.read') }}
                </p>
                <h2 class="ann-drawer__title">{{ t('announcements.title') }}</h2>
              </div>
              <div class="flex items-center gap-2">
                <button
                  v-if="unreadCount > 0"
                  @click="markAllAsRead"
                  :disabled="loading"
                  class="ann-btn ann-btn--ghost"
                >
                  {{ t('announcements.markAllRead') }}
                </button>
                <button
                  @click="closeModal"
                  class="ann-icon-btn"
                  :aria-label="t('common.close')"
                >
                  <Icon name="x" size="sm" />
                </button>
              </div>
            </header>

            <div class="ann-drawer__body">
              <!-- Loading -->
              <div v-if="loading" class="ann-skeleton" aria-busy="true">
                <span v-for="n in 4" :key="n"></span>
              </div>

              <!-- Announcements List -->
              <ol v-else-if="announcements.length > 0" class="ann-list">
                <li
                  v-for="(item, index) in announcements"
                  :key="item.id"
                  class="ann-item group relative"
                  :class="{ 'is-unread': !item.read_at }"
                  :style="{ '--i': index }"
                  @click="openDetail(item)"
                >
                  <span class="ann-item__dot" aria-hidden="true"></span>
                  <div class="min-w-0 flex-1">
                    <h3 class="ann-item__title">{{ item.title }}</h3>
                    <p class="ann-item__meta">
                      <time>{{ formatRelativeTime(item.created_at) }}</time>
                      <span v-if="!item.read_at" class="ann-item__tag">{{ t('announcements.unread') }}</span>
                    </p>
                  </div>
                  <svg class="ann-item__arrow" viewBox="0 0 20 20" fill="none" aria-hidden="true">
                    <path d="M7.5 5l5 5-5 5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </li>
              </ol>

              <!-- Empty State：一只合上的信封 -->
              <div v-else class="ann-empty">
                <span class="ann-envelope" aria-hidden="true"><i></i></span>
                <p class="ann-empty__title">{{ t('announcements.empty') }}</p>
                <p class="ann-empty__desc">{{ t('announcements.emptyDescription') }}</p>
              </div>
            </div>
          </aside>
        </div>
      </Transition>
    </Teleport>

    <!-- 公告详情：信笺式卡片 -->
    <Teleport to="body">
      <Transition name="ann-letter">
        <div
          v-if="detailModalOpen && selectedAnnouncement"
          class="ann-overlay ann-overlay--center rx-theme fixed inset-0 z-[110] flex items-start justify-center overflow-y-auto p-4 pt-[7vh]"
          @click="closeDetail"
        >
          <article class="ann-letter" @click.stop>
            <header class="ann-letter__head">
              <div class="min-w-0 flex-1">
                <p class="ann-eyebrow">
                  <span class="ann-eyebrow__dot" :class="{ 'is-live': !selectedAnnouncement.read_at }" aria-hidden="true"></span>
                  {{ t('announcements.title') }}
                  <span aria-hidden="true">·</span>
                  {{ selectedAnnouncement.read_at ? t('announcements.read') : t('announcements.unread') }}
                </p>
                <h2 class="ann-letter__title">{{ selectedAnnouncement.title }}</h2>
                <p class="ann-letter__meta">
                  <time>{{ formatRelativeWithDateTime(selectedAnnouncement.created_at) }}</time>
                </p>
              </div>
              <button
                @click="closeDetail"
                class="ann-icon-btn"
                :aria-label="t('common.close')"
              >
                <Icon name="x" size="md" />
              </button>
            </header>

            <div class="ann-letter__body">
              <div
                class="markdown-body prose prose-sm max-w-none dark:prose-invert"
                v-html="renderMarkdown(selectedAnnouncement.content)"
              ></div>
            </div>

            <footer class="ann-letter__foot">
              <span class="ann-letter__hint">{{ selectedAnnouncement.read_at ? t('announcements.readStatus') : t('announcements.markReadHint') }}</span>
              <div class="flex items-center gap-2">
                <button @click="closeDetail" class="ann-btn ann-btn--ghost">
                  {{ t('common.close') }}
                </button>
                <button
                  v-if="!selectedAnnouncement.read_at"
                  @click="markAsReadAndClose(selectedAnnouncement.id)"
                  class="ann-btn ann-btn--ink"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {{ t('announcements.markRead') }}
                </button>
              </div>
            </footer>
          </article>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAppStore } from '@/stores/app'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import type { UserAnnouncement } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import '@/styles/announcement-markdown.css'
import '@/styles/recharge-glass.css'
import '@/styles/announcement-glass.css'

const { t } = useI18n()
const appStore = useAppStore()
const announcementStore = useAnnouncementStore()

// Configure marked
marked.setOptions({
  breaks: true,
  gfm: true,
})

// Use store state (storeToRefs for reactivity)
const { announcements, loading } = storeToRefs(announcementStore)
const unreadCount = computed(() => announcementStore.unreadCount)

// Local modal state
const isModalOpen = ref(false)
const detailModalOpen = ref(false)
const selectedAnnouncement = ref<UserAnnouncement | null>(null)

// Methods
function renderMarkdown(content: string): string {
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
}

function openModal() {
  isModalOpen.value = true
}

function closeModal() {
  isModalOpen.value = false
}

function openDetail(announcement: UserAnnouncement) {
  selectedAnnouncement.value = announcement
  detailModalOpen.value = true
  if (!announcement.read_at) {
    markAsRead(announcement.id)
  }
}

function closeDetail() {
  detailModalOpen.value = false
  selectedAnnouncement.value = null
}

async function markAsRead(id: number) {
  try {
    const marked = await announcementStore.markAsRead(id)
    if (!marked) appStore.showError(t('common.unknownError'))
    return marked
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
    return false
  }
}

async function markAsReadAndClose(id: number) {
  if (!await markAsRead(id)) return
  appStore.showSuccess(t('announcements.markedAsRead'))
  closeDetail()
}

async function markAllAsRead() {
  try {
    await announcementStore.markAllAsRead()
    appStore.showSuccess(t('announcements.allMarkedAsRead'))
  } catch (err: any) {
    appStore.showError(err?.message || t('common.unknownError'))
  }
}

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (detailModalOpen.value) {
      closeDetail()
    } else if (isModalOpen.value) {
      closeModal()
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleEscape)
  document.body.style.overflow = ''
})

watch(
  [isModalOpen, detailModalOpen, () => announcementStore.currentPopup],
  ([modal, detail, popup]) => {
    document.body.style.overflow = (modal || detail || popup) ? 'hidden' : ''
  }
)
</script>

<style scoped>
/* ---------- 铃铛 ---------- */
.ann-bell {
  color: var(--rx-muted, #4b5563);
  transition: color 0.2s ease, background 0.2s ease;
}

.ann-bell:hover {
  color: var(--rx-ink, #111827);
  background: var(--rx-surface-hover, #f3f4f6);
}

.ann-bell.has-unread {
  color: var(--rx-ink, #111827);
}

.ann-bell.has-unread .ann-bell__icon {
  display: inline-flex;
  transform-origin: 50% 2px;
  animation: ann-swing 4s ease-in-out infinite;
}

@keyframes ann-swing {
  0%, 82%, 100% { transform: rotate(0deg); }
  85% { transform: rotate(14deg); }
  88% { transform: rotate(-12deg); }
  91% { transform: rotate(8deg); }
  94% { transform: rotate(-4deg); }
}

.ann-bell__count {
  position: absolute;
  top: 0.15rem;
  right: 0.05rem;
  display: grid;
  place-items: center;
  min-width: 1.05rem;
  height: 1.05rem;
  padding: 0 0.25rem;
  border-radius: 9999px;
  font-size: 0.6rem;
  font-weight: 800;
  line-height: 1;
  color: var(--rx-on-accent, #fff);
  background: var(--rx-accent, #111827);
  box-shadow: 0 0 0 2px var(--rx-surface-hover, #fff);
}

@media (prefers-reduced-motion: reduce) {
  .ann-bell.has-unread .ann-bell__icon {
    animation: none;
  }
}
</style>
