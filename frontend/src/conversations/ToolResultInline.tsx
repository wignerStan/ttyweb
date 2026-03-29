import { useState } from 'react'
import type { ToolUseBlock, ToolResultBlock } from './types'

interface ToolResultInlineProps {
  toolUse: ToolUseBlock
  toolResult?: ToolResultBlock
}

function formatToolInput(input: Record<string, unknown>): string {
  const entries = Object.entries(input)
  if (entries.length === 0) return ''
  const parts: string[] = []
  for (const [key, value] of entries) {
    if (key === 'command' && typeof value === 'string') {
      parts.push(value)
    } else if (key === 'file_path' && typeof value === 'string') {
      parts.push(value)
    } else if (typeof value === 'string' && value.length > 60) {
      parts.push(`${key}: ${value.slice(0, 57)}...`)
    } else if (typeof value === 'string') {
      parts.push(`${key}: ${value}`)
    } else {
      parts.push(`${key}: ${JSON.stringify(value)}`)
    }
  }
  return parts.join(' | ')
}

export function ToolResultInline({ toolUse, toolResult }: ToolResultInlineProps) {
  const [expanded, setExpanded] = useState(false)

  const inputSummary = formatToolInput(toolUse.input)
  const output = toolResult?.output
  const hasOutput = output !== undefined && output !== ''

  if (!hasOutput) {
    return (
      <div className="conv-tool-result">
        <div className="conv-tool-result-header">
          <span className="conv-tool-result-name">{toolUse.name}</span>
          {inputSummary && <span className="conv-tool-result-input">{inputSummary}</span>}
        </div>
      </div>
    )
  }

  return (
    <div className={`conv-tool-result${expanded ? ' conv-tool-result--expanded' : ''}`}>
      <div
        className="conv-tool-result-header"
        onClick={() => setExpanded((prev) => !prev)}
        role="button"
        tabIndex={0}
        aria-expanded={expanded}
      >
        <span className="conv-tool-result-chevron">&#9654;</span>
        <span className="conv-tool-result-name">{toolUse.name}</span>
        {inputSummary && <span className="conv-tool-result-input">{inputSummary}</span>}
      </div>
      {expanded && (
        <div className="conv-tool-result-output">{output}</div>
      )}
    </div>
  )
}
