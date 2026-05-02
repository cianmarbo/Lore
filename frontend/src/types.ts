export interface Conversation {
  id: number
  session_id: string
  title: string
  source_path: string
  created_at: string
  updated_at: string
  message_count: number
  topic_count: number
  topics?: Topic[]
}

export interface Topic {
  id: number
  conversation_id: number
  seq: number
  label: string
  started_at: string
  message_count: number
}

export interface Message {
  id: number
  conversation_id: number
  topic_id: number | null
  seq: number
  role: 'user' | 'assistant' | 'system'
  content: string
  tools_json?: string
  tool_results_json?: string
  timestamp: string
}

export interface ToolSummary {
  name: string
  summary: string
}

export interface Document {
  id: number
  title: string
  content: string
  created_at: string
  updated_at: string
  conversation_ids?: number[]
}
