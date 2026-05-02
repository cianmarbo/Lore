import type { Conversation, Document, Message, Topic } from './types'

const BASE = '/api'

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

export function listConversations(): Promise<Conversation[]> {
  return request('/conversations')
}

export function getConversation(id: number): Promise<Conversation> {
  return request(`/conversations/${id}`)
}

export function getMessages(conversationId: number, topicId?: number): Promise<Message[]> {
  const params = topicId != null ? `?topic_id=${topicId}` : ''
  return request(`/conversations/${conversationId}/messages${params}`)
}

export function uploadConversation(payload: { session_id?: string; path?: string }): Promise<Conversation> {
  return request('/conversations/upload', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function deleteConversation(id: number): Promise<void> {
  return request(`/conversations/${id}`, { method: 'DELETE' })
}

export function searchTopics(query: string): Promise<Topic[]> {
  return request(`/search?q=${encodeURIComponent(query)}`)
}

export function listDocuments(): Promise<Document[]> {
  return request('/documents')
}

export function getDocument(id: number): Promise<Document> {
  return request(`/documents/${id}`)
}

export function deleteDocument(id: number): Promise<void> {
  return request(`/documents/${id}`, { method: 'DELETE' })
}
