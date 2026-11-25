import { describe, expect, it } from 'vitest'
import { parseAuditRequestMessages, parseAuditResponseOutput, prettyAuditBody } from '../requestAuditParse'

describe('parseAuditRequestMessages', () => {
  it('parses Anthropic system and block content', () => {
    const msgs = parseAuditRequestMessages(JSON.stringify({
      system: [{ type: 'text', text: 'be nice' }],
      messages: [
        { role: 'user', content: [{ type: 'text', text: 'hello' }, { type: 'image', source: {} }] },
        { role: 'assistant', content: [{ type: 'tool_use', name: 'bash', input: { cmd: 'ls' } }] }
      ]
    }))
    expect(msgs[0]).toEqual({ role: 'system', text: 'be nice' })
    expect(msgs[1].text).toBe('hello\n[image]')
    expect(msgs[2].text).toContain('[tool_use bash]')
  })

  it('parses OpenAI Responses input and instructions', () => {
    const msgs = parseAuditRequestMessages(JSON.stringify({
      instructions: 'sys',
      input: [
        { role: 'user', content: [{ type: 'input_text', text: 'q' }] },
        { type: 'function_call_output', output: 'ok' }
      ]
    }))
    expect(msgs.map((m) => m.role)).toEqual(['system', 'user', 'tool'])
    expect(msgs[1].text).toBe('q')
  })

  it('parses Gemini contents', () => {
    const msgs = parseAuditRequestMessages(JSON.stringify({
      systemInstruction: { parts: [{ text: 's' }] },
      contents: [{ role: 'user', parts: [{ text: 'hi' }] }]
    }))
    expect(msgs).toEqual([{ role: 'system', text: 's' }, { role: 'user', text: 'hi' }])
  })

  it('returns empty for non-JSON', () => {
    expect(parseAuditRequestMessages('[multipart body]')).toEqual([])
  })
})

describe('parseAuditResponseOutput', () => {
  it('assembles Anthropic SSE text, thinking and tool args', () => {
    const sse = [
      'event: content_block_start',
      'data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}',
      'data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}}',
      'data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Hel"}}',
      'data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"lo"}}',
      'data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","name":"bash"}}',
      'data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\\"a\\":"}}',
      'data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"1}"}}',
      ''
    ].join('\n')
    const out = parseAuditResponseOutput(sse)!
    expect(out.text).toBe('Hello')
    expect(out.thinking).toBe('hmm')
    expect(out.tools).toEqual(['bash {"a":1}'])
  })

  it('assembles OpenAI chat SSE', () => {
    const sse = 'data: {"choices":[{"delta":{"content":"a"}}]}\n\ndata: {"choices":[{"delta":{"content":"b"}}]}\n\ndata: [DONE]\n'
    expect(parseAuditResponseOutput(sse)!.text).toBe('ab')
  })

  it('assembles OpenAI Responses SSE and falls back to completed output', () => {
    const withDelta = 'event: response.output_text.delta\ndata: {"type":"response.output_text.delta","delta":"x"}\n\n'
    expect(parseAuditResponseOutput(withDelta)!.text).toBe('x')
    const onlyFinal = 'data: {"type":"response.completed","response":{"output":[{"type":"message","content":[{"type":"output_text","text":"final"}]}]}}\n'
    expect(parseAuditResponseOutput(onlyFinal)!.text).toBe('final')
  })

  it('parses non-stream bodies', () => {
    expect(parseAuditResponseOutput(JSON.stringify({ type: 'message', role: 'assistant', content: [{ type: 'text', text: 'hi' }] }))!.text).toBe('hi')
    expect(parseAuditResponseOutput(JSON.stringify({ choices: [{ message: { content: 'yo' } }] }))!.text).toBe('yo')
    expect(parseAuditResponseOutput(JSON.stringify({ candidates: [{ content: { parts: [{ text: 'g' }] } }] }))!.text).toBe('g')
  })

  it('returns null for error bodies', () => {
    expect(parseAuditResponseOutput('{"error":{"message":"bad"}}')).toBeNull()
    expect(parseAuditResponseOutput('')).toBeNull()
  })
})

describe('prettyAuditBody', () => {
  it('pretty prints JSON and leaves text as-is', () => {
    expect(prettyAuditBody('{"a":1}')).toBe('{\n  "a": 1\n}')
    expect(prettyAuditBody('data: x')).toBe('data: x')
  })
})
