<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Document } from '../types'

const props = defineProps<{
  documents: Document[]
  activeDocument: Document | null
}>()

const emit = defineEmits<{
  selectDocument: [id: number]
  deleteDocument: [id: number]
}>()

const search = ref('')
const confirmDelete = ref<number | null>(null)

const filteredDocuments = computed(() => {
  if (!search.value) return props.documents
  const q = search.value.toLowerCase()
  return props.documents.filter(d =>
    d.title.toLowerCase().includes(q)
  )
})

function handleDelete(id: number) {
  if (confirmDelete.value === id) {
    emit('deleteDocument', id)
    confirmDelete.value = null
  } else {
    confirmDelete.value = id
    setTimeout(() => { confirmDelete.value = null }, 3000)
  }
}

function formatDate(ts: string): string {
  try {
    return new Date(ts).toLocaleDateString('en-GB', {
      year: 'numeric', month: 'short', day: 'numeric'
    })
  } catch { return '' }
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <h1 class="sidebar-title">Documents</h1>
      <input
        v-model="search"
        type="text"
        class="sidebar-search"
        placeholder="Search documents..."
      />
    </div>

    <div class="sidebar-content">
      <div class="section">
        <div
          v-for="d in filteredDocuments"
          :key="d.id"
          class="doc-item"
          :class="{ active: activeDocument?.id === d.id }"
          @click="emit('selectDocument', d.id)"
        >
          <div class="doc-title">{{ d.title || 'Untitled' }}</div>
          <div class="doc-meta">
            {{ (d.conversation_ids || []).length }} session{{ (d.conversation_ids || []).length !== 1 ? 's' : '' }}
            &middot;
            {{ formatDate(d.updated_at) }}
          </div>
          <button
            class="delete-btn"
            @click.stop="handleDelete(d.id)"
          >{{ confirmDelete === d.id ? 'confirm?' : '&times;' }}</button>
        </div>
        <div v-if="filteredDocuments.length === 0" class="empty-hint">
          {{ search ? 'No documents match.' : 'No documents yet. Upload a conversation to generate one.' }}
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  min-width: var(--sidebar-width);
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border);
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  z-index: 100;
}

.sidebar-header {
  padding: 1.25rem 1rem;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.sidebar-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 0.75rem;
}

.sidebar-search {
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  color: var(--text);
  font-family: var(--font);
  font-size: 0.85rem;
  outline: none;
}

.sidebar-search:focus {
  border-color: var(--user-accent);
}

.sidebar-search::placeholder {
  color: var(--text-muted);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
}

.section {
  padding: 0 0.5rem;
}

.doc-item {
  position: relative;
  padding: 0.75rem;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 0.25rem;
  transition: background 0.15s;
  border-left: 3px solid transparent;
}

.doc-item:hover {
  background: var(--sidebar-hover);
}

.doc-item.active {
  background: var(--sidebar-active);
  border-left-color: var(--user-accent);
}

.doc-title {
  font-size: 0.9rem;
  color: var(--text);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.4;
}

.doc-item.active .doc-title {
  color: var(--user-accent);
  font-weight: 600;
}

.doc-meta {
  font-size: 0.7rem;
  color: var(--text-muted);
  font-family: var(--mono);
  margin-top: 0.2rem;
}

.delete-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 0.85rem;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
  opacity: 0;
  transition: opacity 0.15s;
}

.doc-item:hover .delete-btn {
  opacity: 1;
}

.delete-btn:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

.empty-hint {
  text-align: center;
  color: var(--text-muted);
  font-size: 0.85rem;
  padding: 2rem 1rem;
  line-height: 1.5;
}

@media (max-width: 900px) {
  .sidebar {
    transform: translateX(-100%);
    transition: transform 0.25s ease;
  }
}
</style>
