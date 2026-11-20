/**
 * 站内使用文档内容。
 * 所有本站链接统一指向 SITE_URL；第三方工具的官方下载/安装地址保持原样，否则命令无法使用。
 */

export const SITE_URL = 'https://lingsiai.vip'
export const API_BASE = `${SITE_URL}/v1`
export const BRAND = '灵思智域'

export type DocBlock =
  | { type: 'h'; text: string }
  | { type: 'p'; text: string }
  | { type: 'ul'; items: string[] }
  | { type: 'ol'; items: string[] }
  | { type: 'code'; code: string; lang?: string; title?: string }
  | { type: 'table'; head: string[]; rows: string[][] }
  | { type: 'tip'; tone?: 'info' | 'warn'; text: string }
  | { type: 'link'; text: string; href: string }

export interface DocArticle {
  id: string
  title: string
  category: string
  date: string
  summary: string
  blocks: DocBlock[]
}

export const DOC_CATEGORIES = ['快速开始', '客户端接入', 'CC Switch', '图像与视频', '帮助'] as const

export const DOC_ARTICLES: DocArticle[] = [
  // ==================== 快速开始 ====================
  {
    id: 'user-guide',
    title: '网站功能使用手册',
    category: '快速开始',
    date: '2026-09-28',
    summary: '从注册到第一次调用：控制台各菜单的用途与推荐使用顺序。',
    blocks: [
      { type: 'link', text: `${BRAND} 控制台`, href: SITE_URL },
      { type: 'h', text: '第一次使用' },
      {
        type: 'ol',
        items: [
          `打开 ${SITE_URL} 注册并登录账号。`,
          '在「仪表盘」查看余额、密钥数量和近期用量。',
          '进入「API 密钥」创建密钥并选择分组，创建后立即复制保存。',
          `在客户端中填写接口地址 ${API_BASE}，不要再手动追加 /v1 或 /chat/completions。`,
          '发送一条简短消息测试，再到「使用记录」确认请求已到达。'
        ]
      },
      { type: 'p', text: '所有请求均通过请求头携带密钥：' },
      { type: 'code', lang: 'http', code: 'Authorization: Bearer YOUR_API_KEY' },
      { type: 'h', text: '菜单说明' },
      {
        type: 'table',
        head: ['菜单', '用途'],
        rows: [
          ['仪表盘', '余额、密钥数量、调用量概览'],
          ['API 密钥', '创建 / 编辑 / 删除密钥，设置分组与额度，复制接口地址，一键导入 CC Switch'],
          ['使用记录', '按密钥、时间、模型筛选调用明细，支持导出'],
          ['渠道状态', '查看各渠道健康度，出现降级或不可用时可先换分组'],
          ['充值 / 订阅', '余额充值或购买订阅套餐'],
          ['我的订单', '查看充值与订阅订单'],
          ['兑换', '输入兑换码领取额度，区分大小写，每个码只能使用一次'],
          ['个人资料', '修改资料、密码，开启两步验证']
        ]
      },
      { type: 'h', text: '推荐顺序' },
      { type: 'p', text: '仪表盘 → 创建密钥 → 查看渠道状态 → 配置客户端 → 核对使用记录 → 按需充值 → 完善个人资料。' },
      { type: 'h', text: '常见情况' },
      {
        type: 'ul',
        items: [
          '提示鉴权失败：检查密钥是否完整、请求头格式、接口地址是否正确。',
          '密钥有效但调用失败：先看「渠道状态」，再到「使用记录」查看错误信息。',
          '充值后余额未更新：先确认订单状态，仍有问题请携带订单号联系管理员。'
        ]
      }
    ]
  },
  {
    id: 'faq',
    title: '常见问题',
    category: '快速开始',
    date: '2026-09-27',
    summary: '接口地址怎么填、余额与订阅的扣费顺序、支持哪些客户端。',
    blocks: [
      { type: 'tip', text: `遇到问题先核对三件事：接口地址 ${API_BASE}、API 密钥、账户余额。` },
      { type: 'h', text: '余额和订阅哪个先扣？' },
      { type: 'p', text: '有可用订阅时优先使用订阅额度，订阅不足或不覆盖该模型时再扣余额。' },
      { type: 'h', text: '支持哪些支付方式？' },
      { type: 'p', text: '支持微信支付、支付宝、银行卡及兑换码等方式，具体以充值页面展示为准。' },
      { type: 'h', text: '支持哪些客户端？' },
      {
        type: 'p',
        text: '凡是可以自定义 OpenAI 兼容 Base URL、API Key 和模型 ID 的客户端都能使用，例如 Cursor、Cherry Studio、Chatbox、Open WebUI、Claude Code、Codex CLI、CC Switch 以及各语言 SDK。注意部分客户端的内置补全功能仍会走厂商自己的服务。'
      },
      { type: 'h', text: '不同客户端的接口地址' },
      {
        type: 'table',
        head: ['场景', '填写地址'],
        rows: [
          ['OpenAI 兼容客户端', API_BASE],
          ['Claude Code（ANTHROPIC_BASE_URL）', SITE_URL],
          ['要求填写完整接口的字段', `${API_BASE}/chat/completions`]
        ]
      },
      { type: 'h', text: '模型列表为空？' },
      { type: 'p', text: '直接手动输入控制台中显示的完整模型 ID 即可。' },
      { type: 'h', text: '如何确认请求走的是本站？' },
      { type: 'p', text: '单独创建一个密钥，发送一条短消息，然后在「使用记录」中按该密钥筛选查看。' },
      { type: 'h', text: '想恢复使用官方模型？' },
      { type: 'p', text: '在客户端切回内置供应商，清除相关环境变量后重启客户端即可。' }
    ]
  },
  {
    id: 'troubleshooting',
    title: '故障排查',
    category: '快速开始',
    date: '2026-09-27',
    summary: '连通性测试命令与常见错误码对照表。',
    blocks: [
      { type: 'h', text: '按顺序快速检查' },
      {
        type: 'ol',
        items: [
          `Base URL 是否为 ${API_BASE}`,
          'API Key 是否完整、前后没有空格',
          '余额或订阅是否充足',
          '模型名称是否与控制台完全一致'
        ]
      },
      { type: 'h', text: '连通性测试' },
      {
        type: 'code',
        lang: 'bash',
        code: `curl ${API_BASE}/models \\\n  -H "Authorization: Bearer YOUR_API_KEY"`
      },
      { type: 'p', text: '返回 JSON 格式的模型列表，说明网络和密钥都正常。' },
      { type: 'h', text: '错误对照' },
      {
        type: 'table',
        head: ['现象', '可能原因与处理'],
        rows: [
          ['401', '密钥错误、已禁用或带空格；不要重复书写 Bearer'],
          ['403', '当前分组 / 渠道不包含该模型，换用对应分组的密钥'],
          ['404', '模型名或分组不匹配，请完整复制模型名'],
          ['429', '触发频率或并发限制，降低并发后重试'],
          ['余额充足仍失败', '检查预扣费、订阅范围、单次请求上限或模型权限'],
          ['超时', '先用短消息测试，检查代理，适当降低 max_tokens'],
          ['流式输出中断', '关闭 stream 测试，升级客户端或换个模型'],
          ['仍显示旧配置', '重启客户端与终端，必要时删除后重新添加供应商'],
          ['费用偏高', '上下文过长或重复携带历史消息，在使用记录中查看明细'],
          ['模型列表不刷新', `手动输入模型名；若客户端要求根地址，改填 ${SITE_URL}`]
        ]
      }
    ]
  },

  // ==================== 客户端接入 ====================
  {
    id: 'cursor',
    title: '在 Cursor 中使用',
    category: '客户端接入',
    date: '2026-09-27',
    summary: '通过 OpenAI Compatible 供应商在 Cursor 中调用本站模型。',
    blocks: [
      {
        type: 'table',
        head: ['参数', '填写'],
        rows: [
          ['Provider', 'OpenAI Compatible'],
          ['Base URL', API_BASE],
          ['API Key', '控制台创建的密钥'],
          ['Model', '控制台中的完整模型 ID']
        ]
      },
      { type: 'h', text: '配置步骤' },
      {
        type: 'ol',
        items: [
          '打开 Cursor Settings → Models（或 API Keys / 自定义供应商）。',
          '添加 OpenAI Compatible；没有该选项时选择 OpenAI 并开启覆盖 Base URL。',
          '填写 Base URL 与 API Key。',
          '添加准确的模型 ID。',
          '保存并启用，发送一条短消息或 Agent 指令测试。'
        ]
      },
      { type: 'h', text: '注意事项' },
      {
        type: 'ul',
        items: [
          '无需退出 Cursor 账号，API Key 只作用于自定义模型请求。',
          'Tab 补全使用的是 Cursor 自己的模型，不会走本站，这是正常现象。',
          '想重新使用官方模型时，需要关闭「Override OpenAI Base URL」。',
          '免费版 Cursor 的 Agent / Edit 功能可能有额度限制。'
        ]
      },
      { type: 'h', text: '常见问题' },
      {
        type: 'table',
        head: ['问题', '处理'],
        rows: [
          ['没有 Verify 按钮', '直接发送一条消息测试即可'],
          ['model not found', '完整复制模型 ID，并确认密钥分组有该模型权限'],
          ['开启覆盖后官方模型失败', '关闭 Override OpenAI Base URL'],
          ['Network / TLS 错误', '将 HTTP Compatibility Mode 设为 HTTP/1.1']
        ]
      }
    ]
  },
  {
    id: 'codebuddy',
    title: '在 CodeBuddy 中使用',
    category: '客户端接入',
    date: '2026-09-26',
    summary: 'CodeBuddy 自定义模型供应商的推荐填写参数。',
    blocks: [
      {
        type: 'ol',
        items: [
          '供应商类型选择 OpenAI Compatible；没有时选择 OpenAI 并修改 Base URL。',
          `Base URL 填写 ${API_BASE}，不要追加 /chat/completions。`,
          '填写在控制台单独创建的 API Key。',
          '选择控制台中可用的模型，发送一条短消息测试。'
        ]
      },
      { type: 'h', text: '推荐填写参数' },
      {
        type: 'table',
        head: ['字段', '值'],
        rows: [
          ['Provider Type', 'OpenAI Compatible'],
          ['Base URL', API_BASE],
          ['API Key', 'YOUR_API_KEY'],
          ['Model', '选择控制台中可用的模型']
        ]
      },
      { type: 'tip', text: 'Chat、Agent、Completion 可能分别使用不同的供应商配置，请逐一检查。无需退出 CodeBuddy 账号；恢复官方模型只需切回内置模型，必要时重启。' }
    ]
  },
  {
    id: 'vscode-grok',
    title: '在 VSCode 中调用 Grok 模型',
    category: '客户端接入',
    date: '2026-10-15',
    summary: '使用 VSCode 的 Grok 插件或 CC Switch 接入本站 Grok 模型。',
    blocks: [
      { type: 'h', text: '准备工作' },
      {
        type: 'ul',
        items: ['最新版 VSCode', '一个 Grok 分组的 API Key', '账户余额充足']
      },
      {
        type: 'table',
        head: ['参数', '值'],
        rows: [
          ['Base URL', API_BASE],
          ['Model', 'grok-4.7']
        ]
      },
      { type: 'h', text: '方式一：通过 CC Switch' },
      {
        type: 'ol',
        items: [
          '打开 CC Switch，切换到 Grok 面板。',
          '新增 Custom 供应商，填写名称、Base URL、Key 和模型。',
          '保存后点击启用（Enable）。',
          '重启终端或 Grok CLI 使配置生效。'
        ]
      },
      { type: 'h', text: '方式二：在插件设置中填写' },
      {
        type: 'ol',
        items: [
          '在 VSCode 扩展中安装 Grok 相关插件。',
          '供应商类型选择 Custom 或 OpenAI Compatible。',
          '填写 Base URL、API Key 和模型名后保存。'
        ]
      },
      { type: 'tip', text: '部分插件会自动追加 /chat/completions，Base URL 只需填到 /v1。' },
      { type: 'h', text: '验证' },
      { type: 'p', text: '发送一条简短的测试消息，然后在控制台「使用记录」中确认请求。' },
      { type: 'h', text: '排查' },
      {
        type: 'table',
        head: ['现象', '处理'],
        rows: [
          ['401', '密钥错误或带空格'],
          ['404', '路径错误，注意不要写成 /v1/v1'],
          ['model not found', '检查模型名拼写'],
          ['超时', '检查网络或代理'],
          ['额度不足', '充值或更换分组']
        ]
      }
    ]
  },
  {
    id: 'plugins',
    title: '优秀插件介绍',
    category: '客户端接入',
    date: '2026-05-11',
    summary: '在 Claude Code 中用 Codex 做第二意见：codex 与 codex-plan 技能。',
    blocks: [
      { type: 'p', text: '推荐开源仓库中的两个 Claude Code 技能：codex 与 codex-plan。' },
      { type: 'link', text: 'lldxflwb/claude-code-skills', href: 'https://github.com/lldxflwb/claude-code-skills' },
      { type: 'h', text: 'codex' },
      { type: 'p', text: '调用 Codex CLI 对代码、方案或问题给出第二意见，只读运行，不会修改文件。' },
      { type: 'code', lang: 'text', code: '/codex review 一下当前代码变更' },
      { type: 'h', text: 'codex-plan' },
      { type: 'p', text: '多轮评审：Claude 写方案，Codex 负责挑刺，适合在大型重构开始前使用。' },
      { type: 'code', lang: 'text', code: '/codex-plan @docs/migration-plan.md' },
      { type: 'h', text: '前置条件' },
      { type: 'code', lang: 'bash', code: 'npm install -g @openai/codex\ncodex login' },
      { type: 'p', text: '确认 ~/.codex/config.toml 已正确配置。' },
      { type: 'h', text: '安装' },
      {
        type: 'p',
        text: '克隆仓库后，将 skills/codex 与 skills/codex-plan 软链接或复制到 ~/.claude/skills/（Windows 为 %USERPROFILE%\\.claude\\skills\\）。'
      },
      { type: 'tip', text: '建议先把 Codex CLI 指向本站供应商，这样技能会使用同一套模型和额度。' }
    ]
  },

  // ==================== CC Switch ====================
  {
    id: 'cc-switch',
    title: 'CC Switch 配置 Claude & Codex 完整教程',
    category: 'CC Switch',
    date: '2026-09-28',
    summary: '安装 CC Switch，并一键导入配置给 Claude Code、Claude Desktop 和 Codex 使用。',
    blocks: [
      {
        type: 'p',
        text: 'CC Switch 是跨平台的桌面工具，用于统一管理 Claude Code、Claude Desktop、Codex CLI 与 Codex App 的供应商配置。'
      },
      { type: 'link', text: 'CC Switch 官网', href: 'https://ccswitch.io/zh/' },
      { type: 'tip', tone: 'warn', text: '请只从官网或 GitHub Releases 下载。任何要求付费或索要账号密码的「CC Switch」网站都不是官方渠道。' },
      { type: 'h', text: '系统要求' },
      {
        type: 'ul',
        items: [
          'Windows 10 及以上（x64 / ARM64）',
          'macOS 12 Monterey 及以上（Intel / Apple Silicon）',
          'Linux：glibc 2.35+ 与 WebKitGTK 4.1'
        ]
      },
      { type: 'h', text: '安装 CLI 工具（按需）' },
      {
        type: 'code',
        lang: 'bash',
        title: 'Claude Code',
        code: 'curl -fsSL https://claude.ai/install.sh | bash\n# Windows\nwinget install Anthropic.ClaudeCode'
      },
      { type: 'code', lang: 'bash', title: 'Codex', code: 'npm i -g @openai/codex\n# 或\nbrew install --cask codex' },
      { type: 'code', lang: 'bash', title: 'Gemini CLI', code: 'npm i -g @google/gemini-cli' },
      { type: 'code', lang: 'bash', title: 'OpenCode', code: 'curl -fsSL https://opencode.ai/install | bash\n# 或\nnpm i -g opencode-ai' },
      {
        type: 'tip',
        text: 'npm 安装需要 Node.js（Claude Code 需 Node 22+）。国内网络可使用镜像：npm config set registry https://registry.npmmirror.com，镜像落后时改回 https://registry.npmjs.org。'
      },
      { type: 'h', text: '安装 CC Switch' },
      {
        type: 'table',
        head: ['系统', '方式'],
        rows: [
          ['Windows', '.msi 安装包（ARM 选 -Windows-arm64.msi），或便携版 .zip'],
          ['macOS', 'brew install --cask cc-switch，或下载 .dmg / .zip'],
          ['Linux', 'AUR：paru -S cc-switch-bin；.deb：sudo dpkg -i；.rpm：sudo dnf install；或 AppImage']
        ]
      },
      { type: 'p', text: '安装后能打开窗口、托盘出现图标、切换器中能看到被管理的应用，即表示安装成功。启动时会自动检查更新，也可在 设置 → 关于 中手动检查。' },
      { type: 'h', text: '配置 Claude Code' },
      {
        type: 'ol',
        items: [
          `登录 ${SITE_URL}，进入「API 密钥」页面。`,
          '在 Claude 分组的密钥行点击「导入到 CCS」，先选择默认模型再导入。',
          '在 CC Switch 中启用该供应商（会写入 ~/.claude/settings.json）。',
          '打开新终端运行 claude，出现网页登录提示时按 Esc 跳过。'
        ]
      },
      { type: 'h', text: '配置 Claude Desktop' },
      {
        type: 'ol',
        items: [
          '在 CC Switch 中开启本地代理；使用非原生模型时勾选「需要模型映射」并映射到 Sonnet 或 Opus。',
          'Claude Desktop：Help → Troubleshooting → Enable Developer mode。',
          'Developer → Configure third-party inference，Connection 选择 Gateway。',
          '按下表填写后，完全退出并重启 Claude Desktop。'
        ]
      },
      {
        type: 'table',
        head: ['字段', '值'],
        rows: [
          ['Gateway base URL', 'http://127.0.0.1:15721（以 CC Switch 显示为准）'],
          ['Gateway API key', 'PROXY_MANAGED'],
          ['Gateway auth scheme', 'bearer']
        ]
      },
      { type: 'h', text: '配置 Codex' },
      {
        type: 'ol',
        items: [
          '在 Codex 分组的密钥行点击「导入到 CCS」（会写入 ~/.codex/config.toml）。',
          '在 CC Switch 中启用后，新终端运行 codex。',
          'Codex App 读取同一份全局配置，可通过 codex app 启动。'
        ]
      },
      { type: 'tip', text: '「需要本地路由映射」仅用于只支持 Chat Completions 的上游，本站使用 Responses 原生接口时请保持关闭。' },
      { type: 'h', text: '常见问题' },
      {
        type: 'ul',
        items: [
          '切换供应商后需要打开新终端才会生效。',
          'Claude Desktop 报 401：检查本地代理是否开启、API key 是否为 PROXY_MANAGED。',
          'Codex 卡住或报 URL 错误：检查 Base URL 后缀与本地路由映射开关。',
          'Claude 与 Codex 的配置相互独立，需要分别导入。'
        ]
      },
      { type: 'h', text: '升级命令' },
      {
        type: 'code',
        lang: 'bash',
        code: 'claude update\ncodex upgrade\nbrew upgrade --cask cc-switch'
      }
    ]
  },
  {
    id: 'cc-switch-gpt',
    title: '通过 CC Switch 让 Claude Code 使用 GPT 模型',
    category: 'CC Switch',
    date: '2026-05-24',
    summary: 'Opus 效果好但费用高，用 CC Switch 让 Claude Code 改用 GPT 模型。',
    blocks: [
      { type: 'link', text: '下载 CC Switch', href: 'https://ccswitch.io/zh/' },
      {
        type: 'ol',
        items: [
          `登录 ${SITE_URL}，在「API 密钥」页面对 Claude 分组的密钥点击「导入到 CCS」。`,
          '按默认设置导入。',
          '在 CC Switch 中选中刚导入的供应商并点击编辑。',
          '重要：把密钥替换为 Codex / GPT 分组的 API Key。',
          '点击「获取模型列表」，选择 GPT 模型（如 gpt-5.5 / gpt-5.4）后保存。',
          '重启 VSCode，Claude Code 插件中即可使用 GPT 模型。',
          '在控制台「使用记录」中确认调用的模型。'
        ]
      }
    ]
  },
  {
    id: 'cc-switch-balance',
    title: 'CC Switch 配置余额显示',
    category: 'CC Switch',
    date: '2026-05-11',
    summary: '在 CC Switch 中实时显示本站账户剩余额度。',
    blocks: [
      {
        type: 'ol',
        items: ['在 CC Switch 中点击「配置用量查询」。', '选择预设模板。', '将下方代码粘贴到输入框并保存。']
      },
      {
        type: 'code',
        lang: 'js',
        code: `({
  request: {
    url: "{{baseUrl}}/v1/usage",
    method: "GET",
    headers: {
      "Authorization": "Bearer {{apiKey}}"
    }
  },
  extractor: function (response) {
    const remaining = response.remaining ?? response.quota?.remaining ?? response.balance;
    return {
      remaining: remaining,
      unit: response.unit || "USD",
      isValid: response.is_active ?? true
    };
  }
})`
      },
      { type: 'tip', text: '{{baseUrl}} 与 {{apiKey}} 由 CC Switch 自动替换为当前供应商的配置，无需手动修改。' }
    ]
  },

  // ==================== 图像与视频 ====================
  {
    id: 'image-api',
    title: '调用生图 API 指南',
    category: '图像与视频',
    date: '2026-10-06',
    summary: 'gpt-image-2 文生图与图片编辑接口、参数限制和尺寸规则。',
    blocks: [
      { type: 'tip', text: '请使用图像分组的密钥。gpt-image-2 请通过 Images API 调用，Responses API 与 Chat Completions 不作为生图入口。' },
      { type: 'h', text: '接口' },
      {
        type: 'table',
        head: ['方法', '地址', '说明'],
        rows: [
          ['POST', `${API_BASE}/images/generations`, '文生图'],
          ['POST', `${API_BASE}/images/edits`, '图片编辑（multipart，image 必填，可选 PNG mask）']
        ]
      },
      { type: 'h', text: '文生图示例' },
      {
        type: 'code',
        lang: 'bash',
        code: `curl ${API_BASE}/images/generations \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "一只在雪山下奔跑的柴犬，电影感光影",
    "size": "3840x2160",
    "quality": "high",
    "output_format": "png",
    "response_format": "url",
    "n": 1
  }'`
      },
      { type: 'h', text: '图片编辑示例' },
      {
        type: 'code',
        lang: 'bash',
        code: `curl --location '${API_BASE}/images/edits' \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -F model="gpt-image-2" \\
  -F prompt="把背景换成海边日落" \\
  -F image=@input.png \\
  -F size="1536x1024" \\
  -F quality="high" \\
  -F output_format="png" \\
  -F response_format="url"`
      },
      { type: 'h', text: '参数说明' },
      {
        type: 'table',
        head: ['参数', '说明'],
        rows: [
          ['n', '仅支持 1'],
          ['stream / partial_images', '不支持'],
          ['output_format', 'png 或 jpeg（不建议 webp）'],
          ['output_compression', '0–100，仅 jpeg 生效'],
          ['background', '不支持 transparent'],
          ['response_format', 'url（默认）或 b64_json'],
          ['quality', 'low / medium / high / auto'],
          ['input_fidelity', '编辑接口可用，取值 high']
        ]
      },
      { type: 'h', text: '尺寸规则' },
      {
        type: 'ul',
        items: [
          '最长边不超过 3840 px',
          '宽和高都必须是 16 的倍数',
          '长宽比不超过 3:1',
          '总像素在 655,360 到 8,294,400 之间',
          '常用尺寸：1024x1024、1536x1024、1024x1536、2048x2048、2048x1152、3840x2160、2160x3840、auto'
        ]
      },
      { type: 'h', text: '返回格式' },
      {
        type: 'code',
        lang: 'json',
        code: `{
  "created": 1760000000,
  "data": [
    { "url": "https://...", "revised_prompt": "..." }
  ]
}`
      },
      { type: 'p', text: 'response_format 为 b64_json 时，data[].b64_json 返回图片的 Base64，不再返回 url。' }
    ]
  },
  {
    id: 'image-async',
    title: '异步生图接口指引',
    category: '图像与视频',
    date: '2026-10-06',
    summary: '提交任务 → 轮询状态 → 下载图片，适合高分辨率长耗时任务。',
    blocks: [
      {
        type: 'table',
        head: ['项目', '值'],
        rows: [
          ['Base URL', SITE_URL],
          ['默认模型', 'gpt-image-2'],
          ['分辨率', '1K / 2K / 4K（大写）'],
          ['图片有效期', '24 小时']
        ]
      },
      { type: 'h', text: '1. 提交任务' },
      {
        type: 'code',
        lang: 'bash',
        code: `curl -X POST ${API_BASE}/images/generations/async \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{ "model": "gpt-image-2", "prompt": "赛博朋克城市夜景", "size": "2K" }'`
      },
      { type: 'p', text: '返回中包含 id（形如 imgtask_xxx）、created_at、expires_at。' },
      { type: 'h', text: '2. 轮询状态' },
      {
        type: 'code',
        lang: 'bash',
        code: `curl ${API_BASE}/images/tasks/imgtask_xxx \\
  -H "Authorization: Bearer YOUR_API_KEY"`
      },
      { type: 'p', text: '当 status 为 completed 时，返回 http_status 与 image_url。若 image_url 不含 https:// 前缀，请自行补全。' },
      { type: 'h', text: '3. 下载图片' },
      { type: 'p', text: '根据 image_url 下载并保存，图片 24 小时后过期。' },
      { type: 'tip', text: '建议每 3–5 秒轮询一次，避免触发频率限制；拿到结果后尽快转存。' }
    ]
  },
  {
    id: 'video-api',
    title: '调用视频 API 指南',
    category: '图像与视频',
    date: '2026-10-06',
    summary: '异步视频生成：文生视频、参考素材生视频、首尾帧模式与轮询下载。',
    blocks: [
      { type: 'p', text: '视频生成为异步任务：提交 → 保存 task_id → 轮询状态 → 下载结果。同一任务请勿重复提交。' },
      { type: 'h', text: '接口' },
      {
        type: 'table',
        head: ['方法', '地址', '说明'],
        rows: [
          ['POST', `${API_BASE}/videos`, '创建任务'],
          ['GET', `${API_BASE}/videos/{task_id}`, '查询状态'],
          ['GET', `${API_BASE}/videos/{task_id}/content`, '下载视频'],
          ['POST', `${API_BASE}/video-assets`, '上传参考素材（kind 为 image / video / audio，每次一个文件）']
        ]
      },
      { type: 'h', text: '计费' },
      { type: 'p', text: '按计费秒数收费；任务失败全额退款；查询状态与下载不重复计费。' },
      { type: 'h', text: '文生视频' },
      {
        type: 'code',
        lang: 'bash',
        code: `curl -X POST ${API_BASE}/videos \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "Seedance-2.0",
    "prompt": "清晨的海面上，一艘帆船缓缓驶过",
    "ratio": "16:9",
    "duration": 5,
    "resolution": "720p"
  }'`
      },
      { type: 'h', text: '参考素材生视频' },
      {
        type: 'code',
        lang: 'json',
        code: `{
  "model": "Seedance-2.5",
  "prompt": "@图1 中的人物在 @视频1 的场景里跳舞，配乐使用 @音频1",
  "resolution": "480p",
  "referenceImages": ["https://example.com/a.png"],
  "referenceVideos": ["https://example.com/b.mp4"],
  "referenceAudios": ["https://example.com/c.mp3"]
}`
      },
      { type: 'tip', text: '参考素材需为公网可访问的 HTTPS 地址，不支持本地路径或 blob 链接。' },
      { type: 'h', text: '参数' },
      {
        type: 'ul',
        items: [
          'ratio：21:9、16:9、4:3、1:1、3:4、9:16、Auto',
          '首尾帧模式：使用 first_image 与可选的 last_image，ratio 设为 Auto，不能与参考素材数组混用',
          '上传限制：图片 PNG / JPEG / WebP ≤ 20MB，音频 ≤ 20MB，视频 ≤ 64MB；返回的授权链接 24 小时有效'
        ]
      },
      { type: 'h', text: '轮询与下载' },
      {
        type: 'table',
        head: ['status', '含义'],
        rows: [
          ['queued / in_progress', '继续等待，每 3–5 秒查询一次'],
          ['completed', '可以下载'],
          ['failed', '查看错误信息与退款记录']
        ]
      },
      {
        type: 'code',
        lang: 'bash',
        code: `curl -fL ${API_BASE}/videos/task_xxx/content \\
  -H "Authorization: Bearer $API_KEY" \\
  -o result.mp4`
      }
    ]
  }
]
