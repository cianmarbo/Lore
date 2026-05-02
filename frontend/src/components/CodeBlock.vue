<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import hljs from 'highlight.js/lib/core'
import go from 'highlight.js/lib/languages/go'
import php from 'highlight.js/lib/languages/php'
import csharp from 'highlight.js/lib/languages/csharp'
import bash from 'highlight.js/lib/languages/bash'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import sql from 'highlight.js/lib/languages/sql'
import python from 'highlight.js/lib/languages/python'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import typescript from 'highlight.js/lib/languages/typescript'
import javascript from 'highlight.js/lib/languages/javascript'

hljs.registerLanguage('go', go)
hljs.registerLanguage('php', php)
hljs.registerLanguage('csharp', csharp)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('json', json)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('python', python)
hljs.registerLanguage('dockerfile', dockerfile)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('javascript', javascript)

const props = defineProps<{
  code: string
  language?: string
}>()

const codeEl = ref<HTMLElement | null>(null)

function highlight() {
  if (!codeEl.value) return
  codeEl.value.textContent = props.code
  if (props.language && hljs.getLanguage(props.language)) {
    codeEl.value.className = `language-${props.language}`
  }
  hljs.highlightElement(codeEl.value)
}

onMounted(highlight)
watch(() => props.code, highlight)
</script>

<template>
  <pre class="code-block"><code ref="codeEl" :data-lang="language">{{ code }}</code></pre>
</template>

<style scoped>
.code-block {
  background: var(--code-bg) !important;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0 !important;
  overflow-x: auto;
  margin: 0.75rem 0;
  position: relative;
}

.code-block code {
  display: block;
  padding: 1rem !important;
  background: transparent !important;
  font-family: var(--mono);
  font-size: 0.85rem;
  line-height: 1.5;
}

.code-block code[data-lang]::before {
  content: attr(data-lang);
  position: absolute;
  top: 0;
  right: 0;
  padding: 0.15rem 0.5rem;
  font-size: 0.65rem;
  font-family: var(--mono);
  color: var(--text-muted);
  background: var(--border);
  border-radius: 0 8px 0 6px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
</style>
