<template>
<main id="ls-home" class="ls-home">
  <section class="ls-hero">
    <div class="ls-shell">
      <header class="ls-nav" aria-label="首页导航">
        <a class="ls-brand" href="/">
          <span class="ls-logo" aria-hidden="true">
            <img :src="logoUrl" alt="灵思智域" />
          </span>
          <span class="ls-brand-text">
            <span class="ls-brand-title">灵思智域</span>
          </span>
        </a>

        <nav class="ls-nav-links" aria-label="主导航">
          <a href="#ls-product">产品</a>
          <a href="#ls-features">优势</a>
          <a href="#ls-price">价格</a>
          <a href="#ls-docs">接入</a>
        </nav>

        <div class="ls-nav-actions">
          <a class="ls-btn ls-btn-primary" href="/admin/dashboard">进入控制台</a>
        </div>
      </header>

      <div class="ls-hero-main" id="ls-product">
        <h1 class="ls-title">一个接口，接入<br><span>主流 AI 模型</span></h1>
        <p class="ls-desc">灵思智域把 GPT、Claude、Gemini 收到同一个 OpenAI 兼容接口。低成本调用、用量可视化，让 Codex 和自动化工作流稳定在线。</p>
        <div class="ls-hero-actions">
          <a class="ls-btn ls-btn-primary" href="/admin/dashboard">立即进入控制台</a>
          <a class="ls-btn ls-btn-ghost" href="#ls-price">查看充值比例</a>
        </div>
      </div>

      <div class="ls-preview-wrap" aria-label="产品预览">
        <div class="ls-preview-glow"></div>
        <div class="ls-preview">
          <div class="ls-preview-top">
            <div class="ls-window-dots"><i></i><i></i><i></i></div>
            <div class="ls-preview-url">https://lingsiai.vip / admin / dashboard</div>
            <div style="width:54px"></div>
          </div>
          <div class="ls-preview-body">
            <aside class="ls-sidebar">
              <div class="ls-side-title">Console</div>
              <div class="ls-side-link is-active"><b></b>概览面板</div>
              <div class="ls-side-link"><b></b>API Key</div>
              <div class="ls-side-link"><b></b>模型倍率</div>
              <div class="ls-side-link"><b></b>消费记录</div>
              <div class="ls-side-link"><b></b>充值兑换</div>
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
                <div class="ls-real-card ls-green">
                  <div class="ls-real-icon">$</div>
                  <div><label>余额</label><strong>$0.00</strong><small>可用</small></div>
                </div>
                <div class="ls-real-card ls-blue">
                  <div class="ls-real-icon">⌘</div>
                  <div><label>API 密钥</label><strong>0</strong><small>0 启用</small></div>
                </div>
                <div class="ls-real-card ls-cyan">
                  <div class="ls-real-icon">↗</div>
                  <div><label>今日请求</label><strong>0</strong><small>总计: 0</small></div>
                </div>
                <div class="ls-real-card ls-purple">
                  <div class="ls-real-icon">¥</div>
                  <div><label>今日消费</label><strong>$0.0000</strong><small>总计: $0.0000</small></div>
                </div>
                <div class="ls-real-card ls-amber">
                  <div class="ls-real-icon">◈</div>
                  <div><label>今日 Token</label><strong>0</strong><small>输入: 0 / 输出: 0</small></div>
                </div>
                <div class="ls-real-card ls-indigo">
                  <div class="ls-real-icon">▣</div>
                  <div><label>累计 Token</label><strong>0</strong><small>输入: 0 / 输出: 0</small></div>
                </div>
                <div class="ls-real-card ls-violet">
                  <div class="ls-real-icon">⚡</div>
                  <div><label>性能指标</label><strong>0 <em>RPM</em></strong><small>0 TPM</small></div>
                </div>
                <div class="ls-real-card ls-rose">
                  <div class="ls-real-icon">◷</div>
                  <div><label>平均响应</label><strong>0ms</strong><small>实时统计</small></div>
                </div>
              </div>

              <div class="ls-real-bottom">
                <div class="ls-chart-card">
                  <div class="ls-card-title">请求趋势 <span>Today</span></div>
                  <div class="ls-bars" aria-hidden="true">
                    <i style="height:32%"></i><i style="height:58%"></i><i style="height:42%"></i><i style="height:76%"></i><i style="height:52%"></i><i style="height:88%"></i><i style="height:64%"></i><i style="height:72%"></i>
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

  <section class="ls-white" id="ls-features">
    <div class="ls-shell ls-feature-split">
      <div class="ls-feature-sticky">
        <div class="ls-feature-kicker">FEATURES</div>
        <h2>不是堆功能，<br>而是让接入更顺。</h2>
        <p class="ls-section-desc">一个 Base URL 聚合多家模型。创建 API Key 后即可接到 Codex、脚本和自己的产品里，用量和费用在控制台里看清楚。</p>
      </div>

      <div class="ls-feature-list">
        <article class="ls-feature">
          <div>
            <h3>统一 API 网关</h3>
            <p>一个 Base URL 接入多家模型服务，减少多个平台切换和密钥管理成本。</p>
          </div>
          <div class="ls-visual">
            <div class="ls-task"><b>OpenAI Compatible</b><span>https://lingsiai.vip/v1</span></div>
            <div class="ls-task"><b>Claude / GPT / Gemini</b><span>统一转发、统一鉴权、统一计费</span></div>
            <div class="ls-task"><b>Status: Online</b><span>请求稳定运行中</span></div>
          </div>
        </article>

        <article class="ls-feature">
          <div>
            <h3>清晰用量与成本</h3>
            <p>每一笔请求都能看到模型、Token、费用和时间，方便你给自己的用户做计费。</p>
          </div>
          <div class="ls-visual ls-visual-light">
            <div class="ls-task"><b>A 客户 · 25.80M</b><span>收入 ¥1.4940 · 成本自动核算</span></div>
            <div class="ls-task"><b>B 客户 · 274.09K</b><span>小额调用也可追踪利润</span></div>
            <div class="ls-task"><b>按模型倍率计费</b><span>控制台可查看当前倍率与消耗</span></div>
          </div>
        </article>

        <article class="ls-feature">
          <div>
            <h3>适合 Codex 与桌面端</h3>
            <p>用于 Codex 桌面端、内部 AI 工具、论文写作软件、自动化脚本等场景。</p>
          </div>
          <div class="ls-visual">
            <div class="ls-task"><b>Codex Desktop</b><span>配置 API Key 与 Base URL 后即可使用</span></div>
            <div class="ls-task"><b>FastAPI 后端</b><span>你的产品后端可直接调用中转接口</span></div>
            <div class="ls-task"><b>团队分发</b><span>按成员、项目、任务统计用量</span></div>
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
          </div>
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
            <li><b>2</b><span>将 Base URL 设置为 <strong>https://lingsiai.vip/v1</strong>。</span></li>
            <li><b>3</b><span>在请求头中使用 Bearer Token 传入你的 API Key。</span></li>
          </ol>
          <div class="ls-code-block">
            <div class="ls-code-head"><span>API 示例</span><span class="ls-copy-label">复制</span></div>
            <pre><code>curl https://lingsiai.vip/v1/responses \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.5",
    "input": "你好，请简单介绍灵思智域。"
  }'</code></pre>
          </div>
        </article>

        <article class="ls-doc-card">
          <h3>Codex 接入</h3>
          <p>按 Win+R，输入 <strong>%userprofile%\.codex</strong> 打开配置目录。如目录不存在，请先手动创建。</p>
          <ol class="ls-doc-steps">
            <li><b>1</b><span>修改 <strong>C:\Users\用户名\.codex</strong> 下的 <strong>config.toml</strong> 文件。</span></li>
            <li><b>2</b><span>请确保以下内容位于 <strong>config.toml</strong> 文件的开头部分。</span></li>
            <li><b>3</b><span>在同一目录创建或修改 <strong>auth.json</strong>，填入你的 API Key。</span></li>
          </ol>

          <div class="ls-code-block">
            <div class="ls-code-head"><span>%userprofile%\.codex/config.toml</span><span class="ls-copy-label">复制</span></div>
            <pre><code>model_provider = "OpenAI"
model = "gpt-5.5"
review_model = "gpt-5.5"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://lingsiai.vip"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true</code></pre>
          </div>

          <div class="ls-code-block">
            <div class="ls-code-head"><span>%userprofile%\.codex/auth.json</span><span class="ls-copy-label">复制</span></div>
            <pre><code>{
  "OPENAI_API_KEY": "sk-your-key"
}</code></pre>
          </div>
        </article>
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
        <div class="ls-final-actions" style="display:flex;gap:12px;flex-wrap:wrap;">
          <a class="ls-btn ls-btn-primary" href="/admin/dashboard">进入控制台</a>
        </div>
      </div>
    </div>
  </section>

  <footer class="ls-footer">
    <div class="ls-shell">
      <p>灵思智域是面向开发者的 AI API 中转，提供 OpenAI 兼容接口，支持 Codex 与 GPT、Claude、Gemini。</p>
      <nav aria-label="页脚导航">
        <a href="/home">首页</a>
        <a href="#ls-price">充值比例</a>
        <a href="#ls-docs">接入说明</a>
      </nav>
    </div>
  </footer>
</main>
</template>


<script setup lang="ts">
import logoUrl from '@/assets/lingsi-logo.png'
import '@/styles/lingsi-home.css'
</script>
