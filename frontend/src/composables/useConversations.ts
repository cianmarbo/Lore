import { ref, computed } from 'vue'
import type { Conversation, Topic, Message } from '../types'
import * as api from '../api'

const conversations = ref<Conversation[]>([])
const activeConversation = ref<Conversation | null>(null)
const activeTopic = ref<Topic | null>(null)
const messages = ref<Message[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

export function useConversations() {
  async function loadConversations() {
    loading.value = true
    error.value = null
    try {
      conversations.value = (await api.listConversations()) || []
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function selectConversation(id: number) {
    loading.value = true
    error.value = null
    activeTopic.value = null
    try {
      activeConversation.value = await api.getConversation(id)
      messages.value = (await api.getMessages(id)) || []
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function selectTopic(topic: Topic | null) {
    if (!activeConversation.value) return
    activeTopic.value = topic
    loading.value = true
    try {
      messages.value = (await api.getMessages(
        activeConversation.value.id,
        topic?.id
      )) || []
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function removeConversation(id: number) {
    await api.deleteConversation(id)
    conversations.value = conversations.value.filter(c => c.id !== id)
    if (activeConversation.value?.id === id) {
      activeConversation.value = null
      messages.value = []
    }
  }

  const topics = computed<Topic[]>(() => activeConversation.value?.topics || [])

  return {
    conversations,
    activeConversation,
    activeTopic,
    messages,
    topics,
    loading,
    error,
    loadConversations,
    selectConversation,
    selectTopic,
    removeConversation,
  }
}
