export interface AISession {
  id: string
  type: 'claude_code' | 'codex'
  model: string
  title: string
  message_count: number
  project_path?: string
  timestamp?: string
}

export interface ToolUseBlock {
  id: string
  name: string
  input: Record<string, unknown>
}

export interface ToolResultBlock {
  tool_use_id: string
  output?: string
}

export interface ConversationMessage {
  role: 'user' | 'assistant'
  content: string
  tool_use?: ToolUseBlock[]
  tool_result?: ToolResultBlock[]
  timestamp: string
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
}
