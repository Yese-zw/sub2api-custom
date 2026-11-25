/**
 * 把网关审计记录里的原始请求/响应体解析成可读的对话视图。
 * 兼容 Anthropic Messages、OpenAI Chat Completions、OpenAI Responses、Gemini 四种格式，
 * 流式响应按 SSE 逐事件拼接。解析失败时返回空结果，界面回退到原文展示。
 */

export interface AuditMessage {
  role: string
  text: string
}

export interface AuditOutput {
  text: string
  thinking: string
  tools: string[]
}

type Json = any

const IMAGE_PLACEHOLDER = '[image]'

function tryParseJSON(raw: string): Json | undefined {
  try {
    return JSON.parse(raw)
  } catch {
    return undefined
  }
}

function stringify(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

/** 把各家格式的 content（字符串 / block 数组 / parts 数组）压成纯文本。 */
function contentToText(content: Json): string {
  if (content == null) return ''
  if (typeof content === 'string') return content
  if (!Array.isArray(content)) {
    if (typeof content === 'object') return blockToText(content)
    return String(content)
  }
  return content.map(blockToText).filter((s) => s !== '').join('\n')
}

function blockToText(block: Json): string {
  if (block == null) return ''
  if (typeof block === 'string') return block
  if (typeof block !== 'object') return String(block)
  const type = block.type
  if (typeof block.text === 'string') {
    return block.thought ? `[thinking]\n${block.text}` : block.text
  }
  switch (type) {
    case 'thinking':
      return `[thinking]\n${block.thinking ?? ''}`
    case 'redacted_thinking':
      return '[redacted thinking]'
    case 'image':
    case 'image_url':
    case 'input_image':
      return IMAGE_PLACEHOLDER
    case 'input_audio':
      return '[audio]'
    case 'document':
    case 'file':
    case 'input_file':
      return '[file]'
    case 'tool_use':
    case 'server_tool_use':
      return `[tool_use ${block.name ?? ''}]\n${stringify(block.input)}`
    case 'tool_result':
    case 'web_search_tool_result':
      return `[tool_result]\n${contentToText(block.content)}`
    case 'function_call':
      return `[function_call ${block.name ?? ''}]\n${stringify(block.arguments)}`
    case 'function_call_output':
      return `[function_call_output]\n${stringify(block.output)}`
  }
  if (block.inlineData || block.inline_data) return IMAGE_PLACEHOLDER
  if (block.functionCall) return `[function_call ${block.functionCall.name ?? ''}]\n${stringify(block.functionCall.args)}`
  if (block.functionResponse) return `[function_response ${block.functionResponse.name ?? ''}]\n${stringify(block.functionResponse.response)}`
  if (block.content !== undefined) return contentToText(block.content)
  return stringify(block)
}

/** 解析请求体中的对话消息（含 system / instructions）。 */
export function parseAuditRequestMessages(raw: string): AuditMessage[] {
  const body = tryParseJSON(raw)
  if (!body || typeof body !== 'object') return []
  const out: AuditMessage[] = []

  // Anthropic system / OpenAI Responses instructions / Gemini systemInstruction
  if (body.system !== undefined) out.push({ role: 'system', text: contentToText(body.system) })
  if (typeof body.instructions === 'string' && body.instructions) out.push({ role: 'system', text: body.instructions })
  const sysInst = body.systemInstruction ?? body.system_instruction
  if (sysInst) out.push({ role: 'system', text: contentToText(sysInst.parts ?? sysInst) })

  if (Array.isArray(body.messages)) {
    for (const m of body.messages) {
      let text = contentToText(m?.content)
      if (Array.isArray(m?.tool_calls)) {
        const calls = m.tool_calls
          .map((tc: Json) => `[tool_call ${tc?.function?.name ?? ''}]\n${tc?.function?.arguments ?? ''}`)
          .join('\n')
        text = text ? `${text}\n${calls}` : calls
      }
      out.push({ role: String(m?.role ?? 'unknown'), text })
    }
  }

  if (body.input !== undefined && !Array.isArray(body.messages)) {
    if (typeof body.input === 'string') {
      out.push({ role: 'user', text: body.input })
    } else if (Array.isArray(body.input)) {
      for (const item of body.input) {
        if (item?.role) {
          out.push({ role: String(item.role), text: contentToText(item.content) })
        } else {
          out.push({ role: item?.type === 'function_call_output' ? 'tool' : 'assistant', text: blockToText(item) })
        }
      }
    }
  }

  if (Array.isArray(body.contents)) {
    for (const c of body.contents) {
      out.push({ role: String(c?.role ?? 'user'), text: contentToText(c?.parts) })
    }
  }

  if (typeof body.prompt === 'string' && out.length === 0) {
    out.push({ role: 'user', text: body.prompt })
  }
  return out
}

function emptyOutput(): AuditOutput {
  return { text: '', thinking: '', tools: [] }
}

/** 从 SSE 文本中提取每个 data: 事件的 JSON。 */
function parseSSEEvents(raw: string): Json[] {
  const events: Json[] = []
  for (const line of raw.split(/\r?\n/)) {
    if (!line.startsWith('data:')) continue
    const payload = line.slice(5).trim()
    if (!payload || payload === '[DONE]') continue
    const parsed = tryParseJSON(payload)
    if (parsed !== undefined) events.push(parsed)
  }
  return events
}

function applyGeminiCandidate(out: AuditOutput, body: Json) {
  const parts = body?.candidates?.[0]?.content?.parts
  if (!Array.isArray(parts)) return
  for (const p of parts) {
    if (typeof p?.text === 'string') {
      if (p.thought) out.thinking += p.text
      else out.text += p.text
    } else if (p?.functionCall) {
      out.tools.push(`${p.functionCall.name ?? ''} ${stringify(p.functionCall.args)}`)
    }
  }
}

function applyResponsesOutput(out: AuditOutput, output: Json) {
  if (!Array.isArray(output)) return
  for (const item of output) {
    if (item?.type === 'message' && Array.isArray(item.content)) {
      for (const c of item.content) {
        if (typeof c?.text === 'string') out.text += c.text
        else if (typeof c?.refusal === 'string') out.text += c.refusal
      }
    } else if (item?.type === 'reasoning') {
      for (const s of item.summary ?? []) if (typeof s?.text === 'string') out.thinking += s.text
      for (const s of item.content ?? []) if (typeof s?.text === 'string') out.thinking += s.text
    } else if (item?.type === 'function_call' || item?.type === 'custom_tool_call') {
      out.tools.push(`${item.name ?? ''} ${item.arguments ?? item.input ?? ''}`)
    } else if (item?.type) {
      out.tools.push(`[${item.type}]`)
    }
  }
}

function applyNonStreamBody(out: AuditOutput, body: Json) {
  // Anthropic Messages
  if (Array.isArray(body?.content) && (body.type === 'message' || body.role === 'assistant')) {
    for (const block of body.content) {
      if (block?.type === 'text') out.text += block.text ?? ''
      else if (block?.type === 'thinking') out.thinking += block.thinking ?? ''
      else if (block?.type === 'tool_use' || block?.type === 'server_tool_use') out.tools.push(`${block.name ?? ''} ${stringify(block.input)}`)
    }
    return
  }
  // OpenAI Chat Completions
  if (Array.isArray(body?.choices)) {
    const msg = body.choices[0]?.message ?? {}
    out.text += contentToText(msg.content)
    out.thinking += msg.reasoning_content ?? msg.reasoning ?? ''
    for (const tc of msg.tool_calls ?? []) out.tools.push(`${tc?.function?.name ?? ''} ${tc?.function?.arguments ?? ''}`)
    return
  }
  // OpenAI Responses
  if (Array.isArray(body?.output)) {
    applyResponsesOutput(out, body.output)
    return
  }
  // Gemini（非流式也可能是数组）
  if (Array.isArray(body)) {
    for (const chunk of body) applyGeminiCandidate(out, chunk)
    return
  }
  if (body?.candidates) applyGeminiCandidate(out, body)
}

function applyStreamEvents(out: AuditOutput, events: Json[]) {
  const toolArgs = new Map<string, { name: string; args: string }>()
  const toolKey = (k: unknown) => String(k ?? toolArgs.size)
  let responsesFinal: Json | undefined
  let sawResponsesDelta = false

  for (const ev of events) {
    const type: string | undefined = ev?.type
    // Anthropic
    if (type === 'content_block_start' && ev.content_block) {
      const b = ev.content_block
      if (b.type === 'tool_use' || b.type === 'server_tool_use') toolArgs.set(toolKey(ev.index), { name: b.name ?? '', args: '' })
      else if (b.type === 'text' && b.text) out.text += b.text
      continue
    }
    if (type === 'content_block_delta' && ev.delta) {
      const d = ev.delta
      if (d.type === 'text_delta') out.text += d.text ?? ''
      else if (d.type === 'thinking_delta') out.thinking += d.thinking ?? ''
      else if (d.type === 'input_json_delta') {
        const entry = toolArgs.get(toolKey(ev.index))
        if (entry) entry.args += d.partial_json ?? ''
      }
      continue
    }
    // OpenAI Responses
    if (typeof type === 'string' && type.startsWith('response.')) {
      if (type === 'response.output_text.delta' || type === 'response.refusal.delta') {
        out.text += ev.delta ?? ''
        sawResponsesDelta = true
      } else if (type === 'response.reasoning_summary_text.delta' || type === 'response.reasoning_text.delta') {
        out.thinking += ev.delta ?? ''
        sawResponsesDelta = true
      } else if (type === 'response.output_item.added' && (ev.item?.type === 'function_call' || ev.item?.type === 'custom_tool_call')) {
        toolArgs.set(toolKey(ev.item.id ?? ev.output_index), { name: ev.item.name ?? '', args: '' })
      } else if (type === 'response.function_call_arguments.delta' || type === 'response.custom_tool_call_input.delta') {
        const entry = toolArgs.get(toolKey(ev.item_id ?? ev.output_index))
        if (entry) entry.args += ev.delta ?? ''
        sawResponsesDelta = true
      } else if (type === 'response.completed' || type === 'response.done' || type === 'response.incomplete') {
        responsesFinal = ev.response
      }
      continue
    }
    // OpenAI Chat Completions chunk
    if (Array.isArray(ev?.choices)) {
      const d = ev.choices[0]?.delta
      if (d) {
        if (typeof d.content === 'string') out.text += d.content
        const reasoning = d.reasoning_content ?? d.reasoning
        if (typeof reasoning === 'string') out.thinking += reasoning
        for (const tc of d.tool_calls ?? []) {
          const key = toolKey(tc.index)
          const entry = toolArgs.get(key) ?? { name: '', args: '' }
          if (tc.function?.name) entry.name += tc.function.name
          if (tc.function?.arguments) entry.args += tc.function.arguments
          toolArgs.set(key, entry)
        }
      }
      continue
    }
    // Gemini
    if (ev?.candidates) applyGeminiCandidate(out, ev)
  }

  if (!sawResponsesDelta && responsesFinal?.output) {
    applyResponsesOutput(out, responsesFinal.output)
  }
  for (const { name, args } of toolArgs.values()) out.tools.push(`${name} ${args}`.trim())
}

/** 解析响应体中助手的输出文本 / 思考 / 工具调用；无法识别时返回 null。 */
export function parseAuditResponseOutput(raw: string): AuditOutput | null {
  if (!raw) return null
  const out = emptyOutput()
  const trimmed = raw.trimStart()
  if (trimmed.startsWith('data:') || trimmed.startsWith('event:') || /\ndata:/.test(raw.slice(0, 2048))) {
    applyStreamEvents(out, parseSSEEvents(raw))
  } else {
    const body = tryParseJSON(raw)
    if (body === undefined) return null
    applyNonStreamBody(out, body)
  }
  if (!out.text && !out.thinking && out.tools.length === 0) return null
  return out
}

/** 原文展示：JSON 美化，其它保持原样。 */
export function prettyAuditBody(raw: string): string {
  if (!raw) return ''
  const parsed = tryParseJSON(raw)
  if (parsed === undefined) return raw
  return JSON.stringify(parsed, null, 2)
}

export function formatAuditBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '0 B'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}
