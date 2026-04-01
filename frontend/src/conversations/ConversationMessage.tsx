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
    <div
      data-testid={message.role === 'user' ? 'conv-msg-user' : 'conv-msg-assistant'}
      className={`conv-msg max-w-[85%] rounded-md px-3.5 py-2.5 text-sm leading-relaxed font-mono break-words whitespace-pre-wrap ${
        message.role === 'user'
          ? 'self-end border border-edge bg-surface-2 text-on-surface rounded-br-sm'
          : 'self-start border border-surface-2 bg-surface-3 text-on-surface-2 rounded-bl-sm'
      }`}
    >
      <div
        className={`mb-1 text-2xs font-semibold uppercase tracking-wide ${
          message.role === 'user' ? 'text-primary' : 'text-success'
        }`}
      >
        {message.role}
      </div>
      {message.content && (
        <div
          className={`whitespace-pre-wrap font-mono text-xs ${
            message.role === 'assistant' ? '[&_code]:bg-surface' : '[&_code]:bg-surface-2'
          } [&_code]:rounded-sm [&_code]:px-1 [&_code]:text-xs`}
        >
          {message.content}
        </div>
      )}

      {message.tool_use && message.tool_use.length > 0 && (
        <div className="mt-1.5 flex flex-col gap-1">
          {message.tool_use.map((tu) => {
            const result = findToolResult(message.tool_result, tu.id)
            // Show inline summary for tool use blocks without a dedicated result
            if (!result) {
              return (
                <span
                  key={tu.id}
                  className="mt-1.5 inline-block cursor-default rounded-sm border border-edge bg-surface-2 px-2 py-0.5 font-mono text-xs text-secondary"
                >
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
        <div
          data-testid="conv-msg-timestamp"
          className="mt-1 text-right font-mono text-2xs text-on-surface-muted"
        >
          {formatTimestamp(message.timestamp)}
        </div>
      )}
    </div>
  )
}

export const ConversationMessage = memo(ConversationMessageInner)
