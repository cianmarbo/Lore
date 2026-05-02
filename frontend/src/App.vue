<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Sidebar from './components/Sidebar.vue'
import DocumentView from './components/DocumentView.vue'
import ChatOverlay from './components/ChatOverlay.vue'
import { useDocuments } from './composables/useDocuments'

const {
  documents,
  activeDocument,
  loading,
  loadDocuments,
  selectDocument,
  clearDocument,
  removeDocument,
} = useDocuments()

const sidebarOpen = ref(false)
const chatOverlayIds = ref<number[] | null>(null)

const theme = ref<'dark' | 'light'>(
  (localStorage.getItem('theme') as 'dark' | 'light') || 'dark'
)

function applyTheme(t: 'dark' | 'light') {
  document.documentElement.setAttribute('data-theme', t)
  localStorage.setItem('theme', t)
}

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  applyTheme(theme.value)
}

onMounted(() => {
  applyTheme(theme.value)
  loadDocuments()
})

function handleSelectDocument(id: number) {
  selectDocument(id)
  sidebarOpen.value = false
}

function handleViewChat(conversationIds: number[]) {
  chatOverlayIds.value = conversationIds
}

function handleCloseChat() {
  chatOverlayIds.value = null
}

async function handleDeleteDocument(id: number) {
  await removeDocument(id)
}
</script>

<template>
  <div class="layout">
    <div
      class="sidebar-backdrop"
      :class="{ visible: sidebarOpen }"
      @click="sidebarOpen = false"
    />
    <div :class="{ 'sidebar-mobile-open': sidebarOpen }">
      <Sidebar
        :documents="documents"
        :active-document="activeDocument"
        @select-document="handleSelectDocument"
        @delete-document="handleDeleteDocument"
      />
    </div>
    <main class="main">
      <div class="container">
        <!-- Document selected -->
        <template v-if="activeDocument">
          <div class="header">
            <button class="back-link" @click="clearDocument">&larr; Documents</button>
            <h1>{{ activeDocument.title }}</h1>
          </div>
          <div class="card">
            <div v-if="loading" class="loading">Loading...</div>
            <DocumentView
              v-else
              :document="activeDocument"
              @view-chat="handleViewChat"
            />
          </div>
        </template>

        <!-- No document selected -->
        <template v-else>
          <div class="header">
            <h1>Lore</h1>
            <div class="subtitle">Your living knowledge base</div>
          </div>
          <div class="card">
            <div v-if="loading" class="loading">Loading...</div>
            <div v-else-if="documents.length === 0" class="empty-state">
              <p>No documents yet.</p>
              <p class="empty-hint">Upload a conversation with an LLM configured to generate your first document.</p>
            </div>
            <div v-else class="doc-grid">
              <div
                v-for="d in documents"
                :key="d.id"
                class="doc-card"
                @click="handleSelectDocument(d.id)"
              >
                <div class="doc-card-title">{{ d.title || 'Untitled' }}</div>
                <div class="doc-card-meta">
                  {{ (d.conversation_ids || []).length }} session{{ (d.conversation_ids || []).length !== 1 ? 's' : '' }}
                  &middot;
                  {{ new Date(d.updated_at).toLocaleDateString('en-GB', { month: 'short', day: 'numeric' }) }}
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </main>

    <!-- Chat overlay -->
    <ChatOverlay
      v-if="chatOverlayIds"
      :conversation-ids="chatOverlayIds"
      @close="handleCloseChat"
    />

    <button
      class="theme-toggle"
      @click="toggleTheme"
      :aria-label="theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
    >{{ theme === 'dark' ? 'Light' : 'Dark' }}</button>

    <button
      class="sidebar-toggle"
      @click="sidebarOpen = !sidebarOpen"
      aria-label="Toggle sidebar"
    >&#9776;</button>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
}

.main {
  flex: 1;
  margin-left: var(--sidebar-width);
  min-width: 0;
}

.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.header {
  text-align: center;
  padding: 1.5rem 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 2rem;
  position: relative;
}

.header h1 {
  font-family: 'Lexend Deca', sans-serif;
  font-size: 2.5rem;
  font-weight: 400;
  color: var(--text);
}

.back-link {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-family: var(--font);
  font-size: 0.85rem;
  padding: 0.3rem 0;
  transition: color 0.15s;
}

.back-link:hover {
  color: var(--text);
}

.subtitle {
  font-family: var(--mono);
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-top: 0.5rem;
}

.card {
  background: var(--surface);
  border-radius: 16px;
  box-shadow: var(--card-shadow);
  padding: 1.5rem 2rem;
}

.loading {
  text-align: center;
  padding: 3rem;
  color: var(--text-muted);
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--text-muted);
}

.empty-state p {
  margin: 0.5rem 0;
}

.empty-hint {
  font-size: 0.85rem;
}

/* Document grid on home */
.doc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 1rem;
}

.doc-card {
  padding: 1.25rem;
  border-radius: 12px;
  border: 1px solid var(--border);
  cursor: pointer;
  transition: all 0.15s;
}

.doc-card:hover {
  border-color: var(--user-accent);
  background: var(--sidebar-hover);
}

.doc-card-title {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--text);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.4;
  margin-bottom: 0.5rem;
}

.doc-card-meta {
  font-size: 0.7rem;
  color: var(--text-muted);
  font-family: var(--mono);
}

.theme-toggle {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 200;
  padding: 0.4rem 0.8rem;
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-muted);
  border: 1px solid var(--border);
  cursor: pointer;
  font-family: var(--font);
  font-size: 0.8rem;
  box-shadow: var(--card-shadow);
  transition: color 0.2s, background 0.2s, border-color 0.2s;
}

.theme-toggle:hover {
  color: var(--text);
  border-color: var(--text-muted);
}

/* Mobile sidebar */
.sidebar-backdrop {
  display: none;
}

.sidebar-toggle {
  display: none;
  position: fixed;
  bottom: 1.5rem;
  left: 1.5rem;
  z-index: 200;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--user-accent);
  color: var(--bg);
  border: none;
  cursor: pointer;
  font-size: 1.3rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  align-items: center;
  justify-content: center;
}

@media (max-width: 900px) {
  .main {
    margin-left: 0;
  }

  .sidebar-toggle {
    display: flex;
  }

  .sidebar-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 99;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.25s;
  }

  .sidebar-backdrop.visible {
    opacity: 1;
    pointer-events: auto;
  }

  .sidebar-mobile-open :deep(.sidebar) {
    transform: translateX(0) !important;
  }
}
</style>
