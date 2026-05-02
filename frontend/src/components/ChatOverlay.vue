<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Conversation, Message, Topic } from '../types'
import ChatView from './ChatView.vue'
import * as api from '../api'

const props = defineProps<{
  conversationIds: number[]
}>()

const emit = defineEmits<{
  close: []
}>()

const conversations = ref<Conversation[]>([])
const activeConvo = ref<Conversation | null>(null)
const messages = ref<Message[]>([])
const topics = ref<Topic[]>([])
const activeTopic = ref<Topic | null>(null)
const loading = ref(false)

watch(() => props.conversationIds, async (ids) => {
  if (!ids.length) return
  loading.value = true
  try {
    const convos = await Promise.all(ids.map(id => api.getConversation(id)))
    conversations.value = convos
    // Auto-select first conversation
    if (convos.length > 0) {
      await selectConvo(convos[0])
    }
  } catch {
    conversations.value = []
  } finally {
    loading.value = false
  }
}, { immediate: true })

async function selectConvo(convo: Conversation) {
  activeConvo.value = convo
  activeTopic.value = null
  topics.value = convo.topics || []
  loading.value = true
  try {
    messages.value = (await api.getMessages(convo.id)) || []
  } catch {
    messages.value = []
  } finally {
    loading.value = false
  }
}

async function selectTopic(topic: Topic | null) {
  if (!activeConvo.value) return
  activeTopic.value = topic
  loading.value = true
  try {
    messages.value = (await api.getMessages(activeConvo.value.id, topic?.id)) || []
  } catch {
    messages.value = []
  } finally {
    loading.value = false
  }
}

function formatDate(ts: string): string {
  try {
    return new Date(ts).toLocaleDateString('en-GB', {
      year: 'numeric', month: 'short', day: 'numeric',
    })
  } catch { return '' }
}
</script>

<template>
  <Teleport to="body">
    <div class="overlay-backdrop" @click="emit('close')">
      <div class="overlay-panel" @click.stop>
        <div class="overlay-header">
          <div class="overlay-title-row">
            <h2 class="overlay-title">
              {{ activeConvo?.title || 'Chat Session' }}
            </h2>
            <button class="overlay-close" @click="emit('close')">&times;</button>
          </div>

          <!-- Conversation picker if multiple -->
          <div v-if="conversations.length > 1" class="convo-tabs">
            <button
              v-for="c in conversations"
              :key="c.id"
              class="convo-tab"
              :class="{ active: activeConvo?.id === c.id }"
              @click="selectConvo(c)"
            >
              {{ c.title }}
              <span class="convo-tab-date">{{ formatDate(c.created_at) }}</span>
            </button>
          </div>

          <!-- Topic filter -->
          <div v-if="topics.length > 0" class="topic-bar">
            <button
              class="topic-chip"
              :class="{ active: !activeTopic }"
              @click="selectTopic(null)"
            >All</button>
            <button
              v-for="t in topics"
              :key="t.id"
              class="topic-chip"
              :class="{ active: activeTopic?.id === t.id }"
              @click="selectTopic(t)"
            >{{ t.label }}</button>
          </div>
        </div>

        <div class="overlay-body">
          <div v-if="loading" class="overlay-loading">Loading...</div>
          <ChatView
            v-else
            :messages="messages"
            :topics="topics"
            :active-topic="activeTopic"
            :conversation-title="activeConvo?.title"
          />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.overlay-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
  animation: fade-in 0.2s ease;
}

@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slide-in {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}

.overlay-panel {
  width: min(75vw, 900px);
  height: 100vh;
  background: var(--surface);
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.3);
  animation: slide-in 0.25s ease;
}

.overlay-header {
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.overlay-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.overlay-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overlay-close {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  cursor: pointer;
  font-size: 1.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.overlay-close:hover {
  color: var(--text);
  border-color: var(--text-muted);
}

.convo-tabs {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  overflow-x: auto;
}

.convo-tab {
  display: flex;
  flex-direction: column;
  padding: 0.4rem 0.75rem;
  border-radius: 6px;
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text-muted);
  cursor: pointer;
  font-family: var(--font);
  font-size: 0.8rem;
  white-space: nowrap;
  transition: all 0.15s;
}

.convo-tab.active {
  border-color: var(--user-accent);
  color: var(--text);
}

.convo-tab:hover {
  color: var(--text);
}

.convo-tab-date {
  font-family: var(--mono);
  font-size: 0.65rem;
  color: var(--text-muted);
  margin-top: 0.15rem;
}

.topic-bar {
  display: flex;
  gap: 0.35rem;
  margin-top: 0.75rem;
  overflow-x: auto;
  padding-bottom: 0.25rem;
}

.topic-chip {
  padding: 0.3rem 0.65rem;
  border-radius: 999px;
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text-muted);
  cursor: pointer;
  font-family: var(--font);
  font-size: 0.75rem;
  white-space: nowrap;
  transition: all 0.15s;
}

.topic-chip.active {
  background: var(--user-accent);
  border-color: var(--user-accent);
  color: var(--bg);
}

.topic-chip:hover:not(.active) {
  color: var(--text);
  border-color: var(--text-muted);
}

.overlay-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
}

.overlay-loading {
  text-align: center;
  padding: 3rem;
  color: var(--text-muted);
}

@media (max-width: 900px) {
  .overlay-panel {
    width: 100vw;
  }
}
</style>
