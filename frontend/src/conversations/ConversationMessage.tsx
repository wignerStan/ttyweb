import { memo } from 'react'
import { ToolResultInline } from './ToolResultInline'
import type { ConversationMessage as ConversationMessageType, ToolResultBlock } from './types'

interface ConversationMessageProps {
  message: ConversationMessageType
}

function findToolResult(
  toolResults: ToolResultBlock[] | undefined,
  toolUseId: string,
): ToolResultBlock | undefined {
  if (!toolResults) return undefined
  return toolResults.find((tr) => tr.tool_use_id === toolUseId)
}

function formatTimestamp(ts: string): string {
  try {
    const date = new Date(ts)
    if (Number.isNaN(date.getTime())) return ''
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}

function formatToolInputSummary(name: string, input: Record<string, unknown>): string {
  if (name === 'Bash' || name === 'bash') {
    const cmd = input.command
    if (typeof cmd === 'string')
      return `[${name}: ${cmd.length > 80 ? `${cmd.slice(0, 77)}...` : cmd}]`
  }
  if (name === 'Read' || name === 'Edit' || name === 'Write') {
    const fp = input.file_path
    if (typeof fp === 'string') {
      const short = fp.split('/').slice(-2).join('/')
      return `[${name}: ${short}]`
    }
  }
  return `[${name}]`
}

function ConversationMessageInner({ message }: ConversationMessageProps) {
  return (
    <div className={`conv-msg conv-msg--${message.role}`}>
      <div className="conv-msg-role">{message.role}</div>
      {message.content && <div className="conv-msg-content">{message.content}</div>}

      {message.tool_use && message.tool_use.length > 0 && (
        <div style={{ marginTop: '6px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {message.tool_use.map((tu) => {
            const result = findToolResult(message.tool_result, tu.id)
            // Show inline summary for tool use blocks without a dedicated result
            if (!result) {
              return (
                <span key={tu.id} className="conv-tool-summary">
                  {formatToolInputSummary(tu.name, tu.input)}
                </span>
              )
            }
            // Show expandable tool result
            return <ToolResultInline key={tu.id} toolUse={tu} toolResult={result} />
          })}
        </div>
      )}

      {message.timestamp && (
        <div className="conv-msg-timestamp">{formatTimestamp(message.timestamp)}</div>
      )}
    </div>
  )
}

export const ConversationMessage = memo(ConversationMessageInner)
