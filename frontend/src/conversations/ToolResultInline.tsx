import { useState, useCallback } from 'react'
import type { ToolUseBlock, ToolResultBlock } from './types'

interface ToolResultInlineProps {
  toolUse: ToolUseBlock
  toolResult?: ToolResultBlock
}

function formatToolInput(input: Record<string, unknown>): string {
  const entries = Object.entries(input)
  if (entries.length === 0) return ''
  // Show a compact summary of the input
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
  const [output, setOutput] = useState<string | null>(null)

  const inputSummary = formatToolInput(toolUse.input)
  const hasOutput = toolResult !== undefined && toolResult.output !== undefined && toolResult.output !== ''

  const handleToggle = useCallback(() => {
    if (!hasOutput) return
    if (!expanded && output === null && toolResult?.output) {
      setOutput(toolResult.output)
    }
    setExpanded((prev) => !prev)
  }, [expanded, hasOutput, output, toolResult?.output])

  return (
    <div className={`conv-tool-result${expanded ? ' conv-tool-result--expanded' : ''}`}>
      <div className="conv-tool-result-header" onClick={handleToggle} role="button" tabIndex={0} aria-expanded={expanded}>
        {hasOutput && <span className="conv-tool-result-chevron">&#9654;</span>}
        <span className="conv-tool-result-name">{toolUse.name}</span>
        {inputSummary && <span className="conv-tool-result-input">{inputSummary}</span>}
      </div>
      {expanded && hasOutput && output !== null && (
        <div className="conv-tool-result-output">{output}</div>
      )}
    </div>
  )
}
