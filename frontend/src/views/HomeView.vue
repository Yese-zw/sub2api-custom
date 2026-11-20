<template>
<main id="ls-home" class="ls-home">
  <header class="ls-topbar" :class="{ 'is-scrolled': scrolled }">
    <div class="ls-shell ls-nav" aria-label="首页导航">
      <a class="ls-brand" href="/home">
        <span class="ls-logo" aria-hidden="true">
          <img :src="logoUrl" alt="灵思智域" />
        </span>
        <span class="ls-brand-text">
          <span class="ls-brand-title">灵思智域</span>
        </span>
      </a>

      <nav class="ls-nav-links" aria-label="主导航">
        <a
          v-for="link in navLinks"
          :key="link.id"
          :href="`#${link.id}`"
          :class="{ 'is-active': activeSection === link.id }"
        >{{ link.label }}</a>
        <router-link v-if="showModelPlaza" to="/model-plaza">模型广场</router-link>
      </nav>

      <div class="ls-nav-actions">
        <template v-if="isLoggedIn">
          <router-link class="ls-btn ls-btn-primary" :to="consolePath">进入控制台</router-link>
        </template>
        <template v-else>
          <router-link v-if="registrationEnabled" class="ls-btn ls-btn-ghost ls-nav-login" to="/login">登录</router-link>
          <router-link class="ls-btn ls-btn-primary" :to="signupPath">{{ signupLabel }}</router-link>
        </template>
        <button
          type="button"
          class="ls-menu-btn"
          :aria-expanded="menuOpen"
          aria-label="打开菜单"
          @click="menuOpen = !menuOpen"
        >
          <span></span><span></span>
        </button>
      </div>
    </div>

    <transition name="ls-fade">
      <nav v-if="menuOpen" class="ls-mobile-menu" aria-label="移动端导航">
        <a v-for="link in navLinks" :key="link.id" :href="`#${link.id}`" @click="menuOpen = false">{{ link.label }}</a>
        <router-link v-if="showModelPlaza" to="/model-plaza">模型广场</router-link>
        <router-link v-if="!isLoggedIn" to="/login">登录</router-link>
      </nav>
    </transition>
  </header>

  <section class="ls-hero" id="ls-product">
    <div class="ls-shell">
      <div class="ls-hero-main">
        <span class="ls-eyebrow"><i class="ls-dot"></i>OpenAI 兼容 · 一个 Key 调用多家模型</span>
        <h1 class="ls-title">一个接口，接入<br><span>主流 AI 模型</span></h1>
        <p class="ls-desc">灵思智域把 GPT、Claude、Gemini 收到同一个 OpenAI 兼容接口。低成本调用、用量可视化，让 Codex 和自动化工作流稳定在线。</p>
        <div class="ls-hero-actions">
          <router-link class="ls-btn ls-btn-primary" :to="primaryCta.to">{{ primaryCta.label }} →</router-link>
          <a class="ls-btn ls-btn-ghost" href="#ls-docs">查看接入教程</a>
        </div>

        <button type="button" class="ls-baseurl" @click="copy('base', baseUrl)">
          <span class="ls-baseurl-label">Base URL</span>
          <code>{{ baseUrl }}</code>
          <span class="ls-baseurl-copy">{{ copied === 'base' ? '已复制 ✓' : '复制' }}</span>
        </button>

        <div class="ls-models" aria-label="支持的模型">
          <span v-for="m in modelFamilies" :key="m">{{ m }}</span>
        </div>
      </div>

      <div class="ls-preview-wrap" aria-label="产品预览">
        <div class="ls-preview-glow"></div>
        <div class="ls-preview">
          <div class="ls-preview-top">
            <div class="ls-window-dots"><i></i><i></i><i></i></div>
            <div class="ls-preview-url">{{ host }} / dashboard</div>
            <div style="width:54px"></div>
          </div>
          <div class="ls-preview-body">
            <aside class="ls-sidebar">
              <div class="ls-side-title">Console</div>
              <div
                v-for="(item, i) in sideLinks"
                :key="item"
                class="ls-side-link"
                :class="{ 'is-active': i === 0 }"
              ><b></b>{{ item }}</div>
            </aside>
            <div class="ls-dashboard ls-dashboard-real">
              <div class="ls-dashboard-head ls-real-head">
                <div>
                  <h2>仪表盘</h2>
                  <p>欢迎回来！这是您账户的概览。</p>
                </div>
                <div class="ls-user-badge">
                  <span>LS</span>
                  <div><b>演示账号</b><small>preview</small></div>
                </div>
              </div>

              <div class="ls-real-metrics">
                <div v-for="m in demoMetrics" :key="m.label" class="ls-real-card" :class="m.tone">
                  <div class="ls-real-icon">{{ m.icon }}</div>
                  <div><label>{{ m.label }}</label><strong>{{ m.value }}</strong><small>{{ m.sub }}</small></div>
                </div>
              </div>

              <div class="ls-real-bottom">
                <div class="ls-chart-card">
                  <div class="ls-card-title">请求趋势 <span>Today</span></div>
                  <div class="ls-bars" aria-hidden="true">
                    <i v-for="(h, i) in bars" :key="i" :style="{ height: h + '%' }"></i>
                  </div>
                </div>
                <div class="ls-quick-card">
                  <div class="ls-card-title">快速操作 <span>Console</span></div>
                  <div class="ls-quick-list">
                    <div><b>创建 API Key</b><span>为客户端生成独立密钥</span></div>
                    <div><b>查看渠道状态</b><span>确认模型服务是否可用</span></div>
                    <div><b>充值兑换</b><span>余额到账后立即可用</span></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <section class="ls-steps-strip">
    <div class="ls-shell">
      <ol class="ls-quickstart">
        <li v-for="(s, i) in quickSteps" :key="s.title">
          <b>0{{ i + 1 }}</b>
          <div><strong>{{ s.title }}</strong><span>{{ s.desc }}</span></div>
        </li>
      </ol>
    </div>
  </section>

  <section class="ls-white" id="ls-features">
    <div class="ls-shell ls-feature-split">
      <div class="ls-feature-sticky">
        <div class="ls-feature-kicker">FEATURES</div>
        <h2>不是堆功能，<br>而是让接入更顺。</h2>
        <p class="ls-section-desc">一个 Base URL 聚合多家模型。创建 API Key 后即可接到 Codex、脚本和自己的产品里，用量和费用在控制台里看清楚。</p>
      </div>

      <div class="ls-feature-list">
        <article v-for="(f, i) in features" :key="f.title" class="ls-feature">
          <div>
            <h3>{{ f.title }}</h3>
            <p>{{ f.desc }}</p>
          </div>
          <div class="ls-visual" :class="{ 'ls-visual-light': i % 2 === 1 }">
            <div v-for="t in f.tasks" :key="t[0]" class="ls-task"><b>{{ t[0] }}</b><span>{{ t[1] }}</span></div>
          </div>
        </article>
      </div>
    </div>
  </section>

  <section class="ls-pricing" id="ls-price">
    <div class="ls-shell">
      <div class="ls-price-card">
        <div class="ls-price-copy">
          <h2>简单比例，直接开用。</h2>
          <p>面向 Codex、AI 编程、论文写作、自动化脚本等高频场景，降低模型调用门槛，让余额、倍率和消耗都一眼看明白。</p>
          <div class="ls-price-boxes">
            <div class="ls-price-box"><strong>1 : 1</strong><span>充值比例</span></div>
            <div class="ls-price-box"><strong>按量</strong><span>按 Token 与模型倍率计费</span></div>
          </div>
          <router-link class="ls-btn ls-btn-light ls-price-cta" :to="primaryCta.to">{{ primaryCta.label }}</router-link>
        </div>
        <div class="ls-price-list">
          <div class="ls-check-row"><div class="ls-check">✓</div><div><strong>控制台查看用量</strong><span>余额、请求、消耗记录集中展示，便于排查和核算。</span></div></div>
          <div class="ls-check-row"><div class="ls-check">✓</div><div><strong>支持主流 AI 工具</strong><span>适合 Codex、OpenAI SDK、兼容 API 的桌面端和自动化项目。</span></div></div>
          <div class="ls-check-row"><div class="ls-check">✓</div><div><strong>API Key 独立管理</strong><span>不同用途可以拆分不同 Key，降低混用和泄露风险。</span></div></div>
        </div>
      </div>
    </div>
  </section>

  <section class="ls-docs" id="ls-docs">
    <div class="ls-shell">
      <div class="ls-docs-head">
        <div>
          <div class="ls-docs-kicker">INTEGRATION</div>
          <h2>多种方式，<br>快速接入。</h2>
        </div>
        <p>先在控制台创建 API Key，再按使用场景配置 Base URL。普通项目使用 OpenAI 兼容接口；Codex 桌面端按下面的配置文件写入即可。</p>
      </div>

      <div class="ls-docs-grid">
        <article class="ls-doc-card">
          <h3>API 接入</h3>
          <p>适合后端服务、自动化脚本、论文工具和所有支持 OpenAI 兼容接口的客户端。</p>
          <ol class="ls-doc-steps">
            <li><b>1</b><span>进入控制台创建 API Key，并妥善保存密钥。</span></li>
            <li><b>2</b><span>将 Base URL 设置为 <strong>{{ baseUrl }}</strong>。</span></li>
            <li><b>3</b><span>在请求头中使用 Bearer Token 传入你的 API Key。</span></li>
          </ol>
          <div class="ls-code-block">
            <div class="ls-code-head">
              <span>API 示例</span>
              <button type="button" class="ls-copy-label" @click="copy('curl', curlSnippet)">{{ copied === 'curl' ? '已复制 ✓' : '复制' }}</button>
            </div>
            <pre><code>{{ curlSnippet }}</code></pre>
          </div>
        </article>

        <article class="ls-doc-card">
          <h3>Codex 接入</h3>
          <div class="ls-os-tabs" role="tablist" aria-label="操作系统">
            <button
              v-for="o in osOptions"
              :key="o.id"
              type="button"
              role="tab"
              :aria-selected="os === o.id"
              :class="{ 'is-active': os === o.id }"
              @click="os = o.id"
            >{{ o.label }}</button>
          </div>
          <p v-if="os === 'win'">按 Win+R，输入 <strong>%userprofile%\.codex</strong> 打开配置目录。如目录不存在，请先手动创建。</p>
          <p v-else>在终端执行 <strong>mkdir -p ~/.codex &amp;&amp; open ~/.codex</strong>（Linux 用文件管理器打开）进入配置目录。</p>
          <ol class="ls-doc-steps">
            <li><b>1</b><span>修改 <strong>{{ codexDir }}</strong> 下的 <strong>config.toml</strong> 文件。</span></li>
            <li><b>2</b><span>请确保以下内容位于 <strong>config.toml</strong> 文件的开头部分。</span></li>
            <li><b>3</b><span>在同一目录创建或修改 <strong>auth.json</strong>，填入你的 API Key。</span></li>
          </ol>

          <div class="ls-code-block">
            <div class="ls-code-head">
              <span>{{ codexDir }}/config.toml</span>
              <button type="button" class="ls-copy-label" @click="copy('toml', codexToml)">{{ copied === 'toml' ? '已复制 ✓' : '复制' }}</button>
            </div>
            <pre><code>{{ codexToml }}</code></pre>
          </div>

          <div class="ls-code-block">
            <div class="ls-code-head">
              <span>{{ codexDir }}/auth.json</span>
              <button type="button" class="ls-copy-label" @click="copy('auth', authJson)">{{ copied === 'auth' ? '已复制 ✓' : '复制' }}</button>
            </div>
            <pre><code>{{ authJson }}</code></pre>
          </div>
        </article>
      </div>
    </div>
  </section>

  <section class="ls-faq" id="ls-faq">
    <div class="ls-shell ls-faq-grid">
      <div>
        <div class="ls-feature-kicker">FAQ</div>
        <h2>常见问题</h2>
      </div>
      <div class="ls-faq-list">
        <details v-for="(q, i) in faqs" :key="q.q" :open="i === 0">
          <summary>{{ q.q }}</summary>
          <p>{{ q.a }}</p>
        </details>
      </div>
    </div>
  </section>

  <section class="ls-final">
    <div class="ls-shell">
      <div class="ls-final-box">
        <div>
          <h2>现在开始接入灵思智域。</h2>
          <p>进入控制台创建 API Key，把 Base URL 配置到你的 Codex、客户端或项目里，即可开始调用。</p>
        </div>
        <div class="ls-final-actions">
          <router-link class="ls-btn ls-btn-primary" :to="primaryCta.to">{{ primaryCta.label }}</router-link>
          <router-link v-if="!isLoggedIn && registrationEnabled" class="ls-btn ls-btn-ghost" to="/login">已有账号，登录</router-link>
        </div>
      </div>
    </div>
  </section>

  <footer class="ls-footer">
    <div class="ls-shell">
      <p>© {{ year }} 灵思智域 · 面向开发者的 AI API 中转，提供 OpenAI 兼容接口，支持 Codex 与 GPT、Claude、Gemini。</p>
      <nav aria-label="页脚导航">
        <a href="#ls-price">充值比例</a>
        <a href="#ls-docs">接入说明</a>
        <a href="#ls-faq">常见问题</a>
        <router-link v-if="showModelPlaza" to="/model-plaza">模型广场</router-link>
      </nav>
    </div>
  </footer>

  <div class="ls-toast" :class="{ 'is-show': toast }" role="status">{{ toast }}</div>
</main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import logoUrl from '@/assets/lingsi-logo.png'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import '@/styles/lingsi-home.css'

const authStore = useAuthStore()
const appStore = useAppStore()

const FALLBACK_ORIGIN = 'https://lingsiai.vip'
const origin =
  typeof window !== 'undefined' && /^https?:/.test(window.location.origin) && !/localhost|127\.0\.0\.1/.test(window.location.hostname)
    ? window.location.origin
    : FALLBACK_ORIGIN
const baseUrl = `${origin}/v1`
const host = origin.replace(/^https?:\/\//, '')
const year = new Date().getFullYear()

// ---------- auth-aware CTAs ----------
const isLoggedIn = computed(() => authStore.isAuthenticated)
const consolePath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const registrationEnabled = computed(() => appStore.cachedPublicSettings?.registration_enabled !== false)
const signupPath = computed(() => (registrationEnabled.value ? '/register' : '/login'))
const signupLabel = computed(() => (registrationEnabled.value ? '免费注册' : '登录控制台'))
const primaryCta = computed(() =>
  isLoggedIn.value
    ? { to: consolePath.value, label: '进入控制台' }
    : { to: signupPath.value, label: registrationEnabled.value ? '免费注册，立即开始' : '登录控制台' }
)
const showModelPlaza = computed(() => {
  const s = appStore.cachedPublicSettings
  if (!s?.model_plaza_enabled) return false
  return !(s.model_plaza_require_auth && !isLoggedIn.value)
})

// ---------- static content ----------
const navLinks = [
  { id: 'ls-product', label: '产品' },
  { id: 'ls-features', label: '优势' },
  { id: 'ls-price', label: '价格' },
  { id: 'ls-docs', label: '接入' },
  { id: 'ls-faq', label: '问答' }
]
const modelFamilies = ['GPT', 'Claude', 'Gemini', 'Codex', 'OpenAI SDK']
const sideLinks = ['概览面板', 'API Key', '模型倍率', '消费记录', '充值兑换']
const demoMetrics = [
  { tone: 'ls-green', icon: '$', label: '余额', value: '$128.40', sub: '可用' },
  { tone: 'ls-blue', icon: '⌘', label: 'API 密钥', value: '4', sub: '3 启用' },
  { tone: 'ls-cyan', icon: '↗', label: '今日请求', value: '1,284', sub: '总计: 36,902' },
  { tone: 'ls-purple', icon: '¥', label: '今日消费', value: '$3.1820', sub: '总计: $86.5400' },
  { tone: 'ls-amber', icon: '◈', label: '今日 Token', value: '2.41M', sub: '输入: 1.92M / 输出: 0.49M' },
  { tone: 'ls-indigo', icon: '▣', label: '累计 Token', value: '68.7M', sub: '输入: 55.1M / 输出: 13.6M' },
  { tone: 'ls-violet', icon: '⚡', label: '性能指标', value: '42 RPM', sub: '86K TPM' },
  { tone: 'ls-rose', icon: '◷', label: '平均响应', value: '820ms', sub: '实时统计' }
]
const bars = [32, 58, 42, 76, 52, 88, 64, 72]
const quickSteps = [
  { title: '注册账号', desc: '一分钟完成注册' },
  { title: '充值余额', desc: '1:1 比例，到账即用' },
  { title: '创建 API Key', desc: '按用途拆分独立密钥' },
  { title: '配置 Base URL', desc: '接入 Codex / SDK / 脚本' }
]
const features = [
  {
    title: '统一 API 网关',
    desc: '一个 Base URL 接入多家模型服务，减少多个平台切换和密钥管理成本。',
    tasks: [
      ['OpenAI Compatible', baseUrl],
      ['Claude / GPT / Gemini', '统一转发、统一鉴权、统一计费'],
      ['Status: Online', '请求稳定运行中']
    ]
  },
  {
    title: '清晰用量与成本',
    desc: '每一笔请求都能看到模型、Token、费用和时间，方便你给自己的用户做计费。',
    tasks: [
      ['A 客户 · 25.80M', '收入 ¥1.4940 · 成本自动核算'],
      ['B 客户 · 274.09K', '小额调用也可追踪利润'],
      ['按模型倍率计费', '控制台可查看当前倍率与消耗']
    ]
  },
  {
    title: '适合 Codex 与桌面端',
    desc: '用于 Codex 桌面端、内部 AI 工具、论文写作软件、自动化脚本等场景。',
    tasks: [
      ['Codex Desktop', '配置 API Key 与 Base URL 后即可使用'],
      ['FastAPI 后端', '你的产品后端可直接调用中转接口'],
      ['团队分发', '按成员、项目、任务统计用量']
    ]
  }
]
const faqs = [
  { q: '支持哪些客户端和工具？', a: '所有支持 OpenAI 兼容接口的客户端都可以使用，包括 Codex 桌面端 / CLI、OpenAI 官方 SDK、各类桌面 AI 工具以及你自己的后端服务。' },
  { q: '费用如何计算？', a: '充值比例 1:1，调用时按实际消耗的 Token 和所选模型的倍率扣费。每笔请求的模型、Token 与费用都可以在控制台的消费记录里查到。' },
  { q: 'API Key 泄露了怎么办？', a: '在控制台立即禁用或删除对应的 Key，再新建一个替换即可。建议按用途拆分多个 Key，出问题时只影响单个场景。' },
  { q: 'Codex 配置后无法连接？', a: '请确认 config.toml 中的配置位于文件开头、base_url 填写正确，auth.json 中的 Key 有效且账户余额充足。修改配置后需重启 Codex。' }
]

// ---------- Codex OS tabs ----------
type OS = 'win' | 'unix'
const osOptions: { id: OS; label: string }[] = [
  { id: 'win', label: 'Windows' },
  { id: 'unix', label: 'macOS / Linux' }
]
const os = ref<OS>(
  typeof navigator !== 'undefined' && /Mac|Linux|X11/i.test(navigator.userAgent) && !/Android|iPhone|iPad/i.test(navigator.userAgent)
    ? 'unix'
    : 'win'
)
const codexDir = computed(() => (os.value === 'win' ? '%userprofile%\\.codex' : '~/.codex'))

const curlSnippet = `curl ${baseUrl}/responses \\
  -H "Authorization: Bearer sk-your-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.5",
    "input": "你好，请简单介绍灵思智域。"
  }'`

const codexToml = computed(() =>
  [
    'model_provider = "OpenAI"',
    'model = "gpt-5.5"',
    'review_model = "gpt-5.5"',
    'model_reasoning_effort = "xhigh"',
    'disable_response_storage = true',
    'network_access = "enabled"',
    ...(os.value === 'win' ? ['windows_wsl_setup_acknowledged = true'] : []),
    '',
    '[model_providers.OpenAI]',
    'name = "OpenAI"',
    `base_url = "${origin}"`,
    'wire_api = "responses"',
    'requires_openai_auth = true',
    '',
    '[features]',
    'goals = true'
  ].join('\n')
)

const authJson = `{
  "OPENAI_API_KEY": "sk-your-key"
}`

// ---------- copy ----------
const copied = ref<string | null>(null)
const toast = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined

async function copy(key: string, text: string) {
  let ok = false
  try {
    await navigator.clipboard.writeText(text)
    ok = true
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    ok = document.execCommand('copy')
    document.body.removeChild(ta)
  }
  copied.value = ok ? key : null
  toast.value = ok ? '已复制到剪贴板' : '复制失败，请手动选择复制'
  clearTimeout(copyTimer)
  copyTimer = setTimeout(() => {
    copied.value = null
    toast.value = ''
  }, 1800)
}

// ---------- sticky nav / scroll spy ----------
const scrolled = ref(false)
const menuOpen = ref(false)
const activeSection = ref('ls-product')
let observer: IntersectionObserver | undefined

function onScroll() {
  scrolled.value = window.scrollY > 12
}

onMounted(() => {
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings().catch(() => {})

  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) if (e.isIntersecting) activeSection.value = e.target.id
    },
    { rootMargin: '-45% 0px -50% 0px' }
  )
  navLinks.forEach((l) => {
    const el = document.getElementById(l.id)
    if (el) observer!.observe(el)
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  observer?.disconnect()
  clearTimeout(copyTimer)
})
</script>
