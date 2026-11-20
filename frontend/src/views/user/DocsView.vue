<template>
  <AppLayout>
    <div class="docs-shell mx-auto flex max-w-7xl flex-col gap-6 lg:flex-row lg:items-start">
      <!-- 目录 -->
      <aside class="docs-nav card w-full shrink-0 p-3 lg:sticky lg:top-20 lg:w-72">
        <div class="relative mb-3">
          <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            v-model="keyword"
            type="search"
            class="input w-full pl-9 text-sm"
            :placeholder="t('docsCenter.searchPlaceholder')"
            :aria-label="t('docsCenter.searchPlaceholder')"
          />
        </div>

        <nav class="docs-nav__list max-h-[40vh] overflow-y-auto lg:max-h-[calc(100vh-12rem)]" :aria-label="t('docsCenter.title')">
          <p v-if="groupedArticles.length === 0" class="px-3 py-6 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('docsCenter.noResult') }}
          </p>
          <div v-for="group in groupedArticles" :key="group.category" class="mb-3 last:mb-0">
            <p class="px-3 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
              {{ group.category }}
            </p>
            <button
              v-for="item in group.items"
              :key="item.id"
              type="button"
              class="docs-nav__item"
              :class="{ 'is-active': item.id === current.id }"
              :aria-current="item.id === current.id ? 'page' : undefined"
              @click="select(item.id)"
            >
              <span class="truncate">{{ item.title }}</span>
            </button>
          </div>
        </nav>
      </aside>

      <!-- 正文 -->
      <article class="card min-w-0 flex-1 p-5 sm:p-8" :aria-labelledby="`doc-${current.id}`">
        <div class="mb-6 border-b border-gray-100 pb-5 dark:border-dark-700">
          <div class="mb-2 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span class="rounded-md bg-primary-50 px-2 py-0.5 font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
              {{ current.category }}
            </span>
            <span>{{ t('docsCenter.updatedAt', { date: current.date }) }}</span>
          </div>
          <h2 :id="`doc-${current.id}`" class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
            {{ current.title }}
          </h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ current.summary }}</p>
        </div>

        <div class="docs-body space-y-4 text-[15px] leading-7 text-gray-700 dark:text-gray-300">
          <template v-for="(block, idx) in current.blocks" :key="`${current.id}-${idx}`">
            <h3 v-if="block.type === 'h'" class="!mt-8 text-lg font-semibold text-gray-900 dark:text-white">
              {{ block.text }}
            </h3>

            <p v-else-if="block.type === 'p'">{{ block.text }}</p>

            <ul v-else-if="block.type === 'ul'" class="list-disc space-y-1.5 pl-6 marker:text-gray-400">
              <li v-for="(li, i) in block.items" :key="i">{{ li }}</li>
            </ul>

            <ol v-else-if="block.type === 'ol'" class="docs-steps space-y-2">
              <li v-for="(li, i) in block.items" :key="i" class="flex gap-3">
                <span class="docs-steps__num">{{ i + 1 }}</span>
                <span class="min-w-0 flex-1 pt-0.5">{{ li }}</span>
              </li>
            </ol>

            <div v-else-if="block.type === 'code'" class="docs-code">
              <div class="docs-code__bar">
                <span>{{ block.title || block.lang || 'code' }}</span>
                <button type="button" class="docs-code__copy" @click="copyToClipboard(block.code)">
                  <Icon name="copy" size="xs" />
                  {{ t('docsCenter.copy') }}
                </button>
              </div>
              <pre><code>{{ block.code }}</code></pre>
            </div>

            <div v-else-if="block.type === 'table'" class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-700">
              <table class="w-full text-left text-sm">
                <thead class="bg-gray-50 text-gray-600 dark:bg-dark-800 dark:text-gray-300">
                  <tr>
                    <th v-for="h in block.head" :key="h" class="whitespace-nowrap px-4 py-2.5 font-semibold">{{ h }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr v-for="(row, r) in block.rows" :key="r">
                    <td
                      v-for="(cell, c) in row"
                      :key="c"
                      class="px-4 py-2.5 align-top"
                      :class="c === 0 ? 'whitespace-nowrap font-medium text-gray-900 dark:text-white' : 'break-all'"
                    >
                      {{ cell }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div
              v-else-if="block.type === 'tip'"
              class="flex gap-3 rounded-xl border px-4 py-3 text-sm"
              :class="block.tone === 'warn'
                ? 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200'
                : 'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-800/60 dark:bg-sky-900/20 dark:text-sky-200'"
            >
              <Icon :name="block.tone === 'warn' ? 'exclamationTriangle' : 'infoCircle'" size="sm" class="mt-0.5 shrink-0" />
              <span>{{ block.text }}</span>
            </div>

            <a
              v-else-if="block.type === 'link'"
              :href="block.href"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-1.5 font-medium text-primary-600 hover:underline dark:text-primary-400"
            >
              {{ block.text }}
              <Icon name="externalLink" size="xs" />
            </a>
          </template>
        </div>

        <!-- 上一篇 / 下一篇 -->
        <div class="mt-10 grid gap-3 border-t border-gray-100 pt-6 dark:border-dark-700 sm:grid-cols-2">
          <button v-if="prev" type="button" class="docs-pager" @click="select(prev.id)">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('docsCenter.prev') }}</span>
            <span class="truncate font-medium text-gray-900 dark:text-white">{{ prev.title }}</span>
          </button>
          <span v-else class="hidden sm:block"></span>
          <button v-if="next" type="button" class="docs-pager sm:text-right" @click="select(next.id)">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('docsCenter.next') }}</span>
            <span class="truncate font-medium text-gray-900 dark:text-white">{{ next.title }}</span>
          </button>
        </div>
      </article>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { DOC_ARTICLES, DOC_CATEGORIES, type DocArticle } from './docs/docsContent'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { copyToClipboard } = useClipboard()

const keyword = ref('')

// 按分类顺序排列，保证目录与上一篇/下一篇顺序一致
const orderedArticles: DocArticle[] = DOC_CATEGORIES.flatMap((cat) =>
  DOC_ARTICLES.filter((a) => a.category === cat)
)

const current = computed<DocArticle>(() => {
  const id = route.query.doc
  return orderedArticles.find((a) => a.id === id) ?? orderedArticles[0]
})

const groupedArticles = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const match = (a: DocArticle) =>
    !kw || a.title.toLowerCase().includes(kw) || a.summary.toLowerCase().includes(kw)
  return DOC_CATEGORIES.map((category) => ({
    category,
    items: orderedArticles.filter((a) => a.category === category && match(a))
  })).filter((g) => g.items.length > 0)
})

const currentIndex = computed(() => orderedArticles.findIndex((a) => a.id === current.value.id))
const prev = computed(() => orderedArticles[currentIndex.value - 1])
const next = computed(() => orderedArticles[currentIndex.value + 1])

function select(id: string) {
  if (id === current.value.id) return
  router.replace({ query: { ...route.query, doc: id } })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
</script>

<style scoped>
.docs-nav__item {
  display: flex;
  width: 100%;
  align-items: center;
  border-radius: 0.6rem;
  padding: 0.45rem 0.75rem;
  text-align: left;
  font-size: 0.875rem;
  color: rgb(75 85 99);
  transition: background 0.15s ease, color 0.15s ease;
}

.docs-nav__item:hover {
  background: rgb(243 244 246);
  color: rgb(17 24 39);
}

.docs-nav__item.is-active {
  background: rgb(17 24 39);
  color: #fff;
  font-weight: 600;
}

html.dark .docs-nav__item {
  color: rgb(156 163 175);
}

html.dark .docs-nav__item:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #fff;
}

html.dark .docs-nav__item.is-active {
  background: #fff;
  color: rgb(17 24 39);
}

.docs-steps__num {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 1.6rem;
  height: 1.6rem;
  margin-top: 0.15rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 700;
  color: #fff;
  background: rgb(17 24 39);
}

html.dark .docs-steps__num {
  color: rgb(17 24 39);
  background: #fff;
}

.docs-code {
  overflow: hidden;
  border-radius: 0.85rem;
  border: 1px solid rgb(31 41 55);
  background: #0f1115;
}

.docs-code__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.4rem 0.6rem 0.4rem 0.9rem;
  font-size: 0.72rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  color: rgb(156 163 175);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.docs-code__copy {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  border-radius: 0.45rem;
  padding: 0.2rem 0.5rem;
  color: rgb(209 213 219);
  transition: background 0.15s ease;
}

.docs-code__copy:hover {
  background: rgba(255, 255, 255, 0.08);
}

.docs-code pre {
  margin: 0;
  overflow-x: auto;
  padding: 0.9rem 1rem;
  font-size: 0.8rem;
  line-height: 1.6;
  color: rgb(229 231 235);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.docs-pager {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
  border-radius: 0.85rem;
  border: 1px solid rgb(229 231 235);
  padding: 0.8rem 1rem;
  text-align: left;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.docs-pager:hover {
  border-color: rgb(156 163 175);
}

html.dark .docs-pager {
  border-color: rgba(255, 255, 255, 0.1);
}

html.dark .docs-pager:hover {
  border-color: rgba(255, 255, 255, 0.25);
}
</style>
