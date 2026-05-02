<script setup lang="ts">
import { computed } from 'vue'
import { Marked } from 'marked'
import hljs from 'highlight.js'
import type { Document } from '../types'

const props = defineProps<{
  document: Document
}>()

const emit = defineEmits<{
  viewChat: [conversationIds: number[]]
}>()

const marked = new Marked({
  renderer: {
    code({ text, lang }) {
      const language = lang && hljs.getLanguage(lang) ? lang : 'plaintext'
      const highlighted = hljs.highlight(text, { language }).value
      return `<pre><code class="hljs language-${language}">${highlighted}</code></pre>`
    },
  },
})

const renderedContent = computed(() => {
  return marked.parse(props.document.content) as string
})

const conversationCount = computed(() => {
  return props.document.conversation_ids?.length || 0
})

function formatDate(ts: string): string {
  try {
    return new Date(ts).toLocaleDateString('en-GB', {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit',
    })
  } catch { return '' }
}
</script>

<template>
  <div class="document-view">
    <div class="document-toolbar">
      <div class="document-meta">
        Updated {{ formatDate(document.updated_at) }}
        <span class="meta-sep">&middot;</span>
        {{ conversationCount }} session{{ conversationCount !== 1 ? 's' : '' }}
      </div>
      <button
        v-if="conversationCount > 0"
        class="view-chat-btn"
        @click="emit('viewChat', document.conversation_ids || [])"
      >
        View Chat Session{{ conversationCount !== 1 ? 's' : '' }}
      </button>
    </div>
    <div
      ref="contentEl"
      class="document-content"
      v-html="renderedContent"
    />
  </div>
</template>

<style scoped>
.document-view {
  position: relative;
}

.document-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border);
}

.document-meta {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-family: var(--mono);
}

.meta-sep {
  margin: 0 0.4rem;
}

.view-chat-btn {
  padding: 0.45rem 1rem;
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-muted);
  border: 1px solid var(--border);
  cursor: pointer;
  font-family: var(--font);
  font-size: 0.8rem;
  transition: all 0.15s;
  white-space: nowrap;
}

.view-chat-btn:hover {
  color: var(--text);
  border-color: var(--user-accent);
  background: var(--sidebar-hover);
}

.document-content {
  line-height: 1.7;
  color: var(--text);
}

.document-content :deep(h1) {
  font-size: 1.6rem;
  font-weight: 600;
  margin: 0 0 1rem;
  color: var(--text);
}

.document-content :deep(h2) {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 1.75rem 0 0.75rem;
  color: var(--text);
}

.document-content :deep(h3) {
  font-size: 1.05rem;
  font-weight: 600;
  margin: 1.5rem 0 0.5rem;
  color: var(--text);
}

.document-content :deep(p) {
  margin: 0.75rem 0;
}

.document-content :deep(ul),
.document-content :deep(ol) {
  margin: 0.75rem 0;
  padding-left: 1.5rem;
}

.document-content :deep(li) {
  margin: 0.3rem 0;
}

.document-content :deep(pre) {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1rem;
  overflow-x: auto;
  margin: 1rem 0;
}

.document-content :deep(code) {
  font-family: var(--mono);
  font-size: 0.85rem;
}

.document-content :deep(:not(pre) > code) {
  background: var(--bg);
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  font-size: 0.85em;
}

.document-content :deep(blockquote) {
  border-left: 3px solid var(--user-accent);
  margin: 1rem 0;
  padding: 0.5rem 1rem;
  color: var(--text-muted);
}

.document-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border);
  margin: 1.5rem 0;
}

.document-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 1rem 0;
}

.document-content :deep(th),
.document-content :deep(td) {
  border: 1px solid var(--border);
  padding: 0.5rem 0.75rem;
  text-align: left;
}

.document-content :deep(th) {
  background: var(--bg);
  font-weight: 600;
}

.document-content :deep(a) {
  color: var(--user-accent);
  text-decoration: none;
}

.document-content :deep(a:hover) {
  text-decoration: underline;
}
</style>
