<script setup lang="ts">
import type { ToolSummary } from '../types'

const props = defineProps<{
  tools?: ToolSummary[]
  results?: string[]
}>()
</script>

<template>
  <details v-if="tools && tools.length" class="tool-details tool-use">
    <summary>{{ tools.length }} tool call{{ tools.length !== 1 ? 's' : '' }}</summary>
    <pre>{{ tools.map(t => t.summary).join('\n') }}</pre>
  </details>
  <details v-if="results && results.length" class="tool-details tool-result">
    <summary>{{ results.length }} tool result{{ results.length !== 1 ? 's' : '' }}</summary>
    <pre>{{ results.map(r => r.length > 2000 ? r.slice(0, 2000) + `\n... [${r.length} chars total]` : r).join('\n---\n') }}</pre>
  </details>
</template>

<style scoped>
.tool-details {
  margin-top: 0.75rem;
}

.tool-details summary {
  cursor: pointer;
  font-size: 0.8rem;
  color: var(--text-muted);
  font-family: var(--mono);
  padding: 0.3rem 0;
}

.tool-details summary:hover {
  color: var(--text);
}

.tool-details pre {
  background: var(--code-bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 0.75rem;
  font-family: var(--mono);
  font-size: 0.8rem;
  overflow-x: auto;
  margin-top: 0.5rem;
  max-height: 400px;
  overflow-y: auto;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
}

.tool-use summary { color: var(--assistant-accent); }
.tool-result summary { color: var(--user-accent); }
</style>
