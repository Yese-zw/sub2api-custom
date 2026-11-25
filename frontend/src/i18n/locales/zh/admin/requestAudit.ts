export default {
  requestAudit: {
    detailTitle: '请求详情',
    settingsTitle: '请求审计设置',
    settingsButton: '请求审计',
    clickHint: '点击行查看完整输入/输出',
    loading: '加载中…',
    notFound: '未找到该请求的审计记录：可能发生在审计开启之前、已超过保留期，或为 WebSocket/GET 请求。',
    loadFailed: '加载请求详情失败',
    tabs: {
      conversation: '对话',
      request: '请求原文',
      response: '响应原文',
      meta: '元数据'
    },
    input: '输入',
    output: '输出',
    thinking: '思考',
    tools: '工具调用',
    noParsedInput: '无法解析为对话消息，请查看「请求原文」。',
    noParsedOutput: '无法解析出助手输出（可能为错误响应），请查看「响应原文」。',
    truncated: '已截断：原始 {size}，仅保存前 {limit}',
    size: '大小',
    copy: '复制',
    copied: '已复制',
    empty: '（空）',
    meta: {
      createdAt: '请求时间',
      path: '路径',
      model: '模型',
      stream: '流式',
      status: '状态码',
      duration: '耗时',
      clientIp: '客户端 IP',
      userAgent: 'User-Agent',
      requestId: '请求 ID',
      clientRequestId: '客户端请求 ID',
      headers: '请求头'
    },
    settings: {
      enabled: '启用请求审计',
      enabledHint: '记录网关每次请求的完整请求体与返回给客户端的响应体（不含 GET 与 WebSocket）。',
      retentionDays: '保留天数',
      retentionDaysHint: '超过天数的记录每小时自动清理；0 表示永久保留。',
      maxBodyKB: '单个 Body 上限 (KB)',
      maxBodyKBHint: '请求体与响应体分别截断到该大小（4 KB – 16384 KB），内联 base64 图片会折叠为占位符。',
      saved: '请求审计设置已保存',
      saveFailed: '保存请求审计设置失败',
      loadFailed: '加载请求审计设置失败'
    }
  }
}
