<script setup lang="ts">
import { computed } from 'vue'
import type { Message, Topic } from '../types'
import MessageBubble from './MessageBubble.vue'

const props = defineProps<{
  messages: Message[]
  topics: Topic[]
  activeTopic: Topic | null
  conversationTitle?: string
}>()

// Group messages by topic for separator display
const topicMap = computed(() => {
  const map = new Map<number, Topic>()
  for (const t of props.topics) {
    map.set(t.id, t)
  }
  return map
})

// Track which topic_id we've already shown a separator for
const messagesWithSeparators = computed(() => {
  const result: Array<{ type: 'separator'; topic: Topic } | { type: 'message'; message: Message }> = []
  const seenTopics = new Set<number>()

  for (const msg of props.messages) {
    if (msg.topic_id && !seenTopics.has(msg.topic_id)) {
      seenTopics.add(msg.topic_id)
      const topic = topicMap.value.get(msg.topic_id)
      if (topic) {
        result.push({ type: 'separator', topic })
      }
    }
    result.push({ type: 'message', message: msg })
  }
  return result
})

function formatTopicTime(ts: string): string {
  try {
    return new Date(ts).toLocaleString('en-GB', {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit'
    })
  } catch { return '' }
}
</script>

<template>
  <div class="chat-view">
    <div v-if="messages.length === 0 && !conversationTitle" class="empty-state">
      <div class="empty-icon">&#128172;</div>
      <h2>Select a conversation</h2>
      <p>Choose a conversation from the sidebar, or upload one using the MCP tool from Claude Code.</p>
    </div>

    <div v-else-if="messages.length === 0" class="empty-state">
      <p>No messages match this filter.</p>
    </div>

    <template v-else>
      <div
        v-for="(item, i) in messagesWithSeparators"
        :key="i"
      >
        <div v-if="item.type === 'separator'" class="topic-separator" :id="`topic-${item.topic.id}`">
          <span class="topic-label">{{ item.topic.label }}</span>
          <span class="topic-time">{{ formatTopicTime(item.topic.started_at) }}</span>
        </div>
        <MessageBubble v-else :message="item.message" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.chat-view {
  padding-bottom: 4rem;
}

.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  color: var(--text-muted);
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.empty-state h2 {
  font-size: 1.3rem;
  color: var(--text);
  margin-bottom: 0.5rem;
}

.empty-state p {
  font-size: 0.95rem;
  max-width: 400px;
  margin: 0 auto;
  line-height: 1.6;
}

.topic-separator {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin: 2rem 0 1.25rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border);
}

.topic-separator:first-child {
  margin-top: 0;
}

.topic-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--user-accent);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topic-time {
  font-family: var(--mono);
  font-size: 0.7rem;
  color: var(--text-muted);
  flex-shrink: 0;
}
</style>
