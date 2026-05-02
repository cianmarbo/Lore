<script setup lang="ts">
import { computed } from 'vue'
import type { Message, ToolSummary } from '../types'
import CodeBlock from './CodeBlock.vue'
import ToolDetails from './ToolDetails.vue'

const props = defineProps<{ message: Message }>()

const tools = computed<ToolSummary[]>(() => {
  if (!props.message.tools_json) return []
  try { return JSON.parse(props.message.tools_json) } catch { return [] }
})

const toolResults = computed<string[]>(() => {
  if (!props.message.tool_results_json) return []
  try { return JSON.parse(props.message.tool_results_json) } catch { return [] }
})

const formattedTime = computed(() => {
  try {
    return new Date(props.message.timestamp).toLocaleString('en-GB', {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit'
    })
  } catch { return '' }
})

// Parse markdown content into segments: text and code blocks
interface Segment { type: 'text' | 'code'; content: string; lang?: string }

const segments = computed<Segment[]>(() => {
  const text = props.message.content || ''
  const result: Segment[] = []
  const codeBlockRe = /```(\w*)\n([\s\S]*?)```/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = codeBlockRe.exec(text)) !== null) {
    if (match.index > lastIndex) {
      result.push({ type: 'text', content: text.slice(lastIndex, match.index) })
    }
    result.push({ type: 'code', content: match[2], lang: match[1] || undefined })
    lastIndex = match.index + match[0].length
  }

  if (lastIndex < text.length) {
    result.push({ type: 'text', content: text.slice(lastIndex) })
  }

  return result
})

function renderMarkdown(text: string): string {
  let html = escapeHtml(text)

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')
  // Bold
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  // Italic
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  // Headers
  html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>')
  html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>')

  // Tables
  html = html.replace(/(\|.+\|(?:\n\|.+\|)+)/g, (match) => {
    const rows = match.trim().split('\n')
    let tableHtml = '<table>'
    rows.forEach((row, i) => {
      const cells = row.replace(/^\||\|$/g, '').split('|').map(c => c.trim())
      if (cells.every(c => /^[-:]+$/.test(c))) return
      const tag = i === 0 ? 'th' : 'td'
      tableHtml += '<tr>' + cells.map(c => `<${tag}>${c}</${tag}>`).join('') + '</tr>'
    })
    return tableHtml + '</table>'
  })

  // Paragraphs
  html = html.replace(/\n\n+/g, '</p><p>')
  html = html.replace(/\n/g, '<br>')
  html = `<p>${html}</p>`

  return html
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

const roleClass = computed(() => `${props.message.role}-message`)
const roleLabel = computed(() => {
  switch (props.message.role) {
    case 'user': return 'You'
    case 'assistant': return 'Claude'
    case 'system': return 'System'
    default: return props.message.role
  }
})
</script>

<template>
  <div class="message" :class="roleClass">
    <div class="message-header">
      <span class="role" :class="`${message.role}-role`">{{ roleLabel }}</span>
      <span class="timestamp">{{ formattedTime }}</span>
    </div>
    <div class="message-body" v-if="message.content">
      <template v-for="(seg, i) in segments" :key="i">
        <div v-if="seg.type === 'text'" v-html="renderMarkdown(seg.content)" />
        <CodeBlock v-else :code="seg.content" :language="seg.lang" />
      </template>
    </div>
    <ToolDetails :tools="tools" :results="toolResults" />
  </div>
</template>

<style scoped>
.message {
  margin-bottom: 1.5rem;
  border-radius: 12px;
  padding: 1rem 1.25rem;
  border: 1px solid var(--border);
}

.user-message {
  background: var(--user-bg);
  border-left: 3px solid var(--user-accent);
}

.assistant-message {
  background: var(--assistant-bg);
  border-left: 3px solid var(--assistant-accent);
}

.system-message {
  background: var(--system-bg);
  border-left: 3px solid var(--system-accent);
  font-size: 0.85rem;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.role {
  font-weight: 700;
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.user-role { color: var(--user-accent); }
.assistant-role { color: var(--assistant-accent); }
.system-role { color: var(--system-accent); }

.timestamp {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-family: var(--mono);
}

.message-body {
  font-size: 0.95rem;
}

.message-body :deep(p) {
  margin-bottom: 0.75rem;
}

.message-body :deep(p:last-child) {
  margin-bottom: 0;
}

.message-body :deep(h2),
.message-body :deep(h3),
.message-body :deep(h4) {
  margin: 1rem 0 0.5rem;
  color: var(--text);
}

.message-body :deep(code) {
  font-family: var(--mono);
  font-size: 0.85em;
  background: var(--code-bg);
  padding: 0.15em 0.4em;
  border-radius: 4px;
  color: var(--code-inline);
}

.message-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0.75rem 0;
  font-size: 0.9rem;
}

.message-body :deep(th),
.message-body :deep(td) {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border);
  text-align: left;
}

.message-body :deep(th) {
  background: var(--code-bg);
  font-weight: 600;
}

.message-body :deep(strong) {
  color: var(--text-strong);
}
</style>
