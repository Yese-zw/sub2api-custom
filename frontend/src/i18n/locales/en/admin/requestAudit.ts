export default {
  requestAudit: {
    detailTitle: 'Request Details',
    settingsTitle: 'Request Audit Settings',
    settingsButton: 'Request Audit',
    clickHint: 'Click a row to view the full input/output',
    loading: 'Loading…',
    notFound: 'No audit record found for this request: it may predate enabling audit, be past the retention period, or be a WebSocket/GET request.',
    loadFailed: 'Failed to load request details',
    tabs: {
      conversation: 'Conversation',
      request: 'Raw Request',
      response: 'Raw Response',
      meta: 'Metadata'
    },
    input: 'Input',
    output: 'Output',
    thinking: 'Thinking',
    tools: 'Tool Calls',
    noParsedInput: 'Could not parse conversation messages. See "Raw Request".',
    noParsedOutput: 'Could not parse assistant output (possibly an error response). See "Raw Response".',
    truncated: 'Truncated: original {size}, only the first {limit} stored',
    size: 'Size',
    copy: 'Copy',
    copied: 'Copied',
    empty: '(empty)',
    meta: {
      createdAt: 'Request Time',
      path: 'Path',
      model: 'Model',
      stream: 'Stream',
      status: 'Status',
      duration: 'Duration',
      clientIp: 'Client IP',
      userAgent: 'User-Agent',
      requestId: 'Request ID',
      clientRequestId: 'Client Request ID',
      headers: 'Request Headers'
    },
    settings: {
      enabled: 'Enable request audit',
      enabledHint: 'Store the full request body and the response body returned to the client for every gateway request (excluding GET and WebSocket).',
      retentionDays: 'Retention (days)',
      retentionDaysHint: 'Older records are deleted hourly; 0 keeps them forever.',
      maxBodyKB: 'Max body size (KB)',
      maxBodyKBHint: 'Request and response bodies are each truncated to this size (4 KB – 16384 KB). Inline base64 images are folded into placeholders.',
      saved: 'Request audit settings saved',
      saveFailed: 'Failed to save request audit settings',
      loadFailed: 'Failed to load request audit settings'
    }
  }
}
