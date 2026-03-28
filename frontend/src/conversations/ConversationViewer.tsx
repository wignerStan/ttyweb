import { useConversation } from './useConversations'
import { ConversationMessage } from './ConversationMessage'
import type { AISession } from './types'
import './conversations.css'

interface ConversationViewerProps {
  sessionId: string | null
  sessionInfo: AISession | null
  onBack: () => void
}

export function ConversationViewer({ sessionId, sessionInfo, onBack }: ConversationViewerProps) {
  const { messages, loading, error, refresh } = useConversation(sessionId)

  const handleRefresh = () => {
    refresh()
  }

  if (!sessionId) {
    return (
      <div className="conv-viewer-container">
        <div className="conv-viewer-header">
          <span className="conv-viewer-header-title">Conversation Viewer</span>
        </div>
        <div className="conv-viewer-empty">Select a conversation to view</div>
      </div>
    )
  }

  return (
    <div className="conv-viewer-container">
      <div className="conv-viewer-header">
        <button className="conv-viewer-header-btn" onClick={onBack}>
          &#9664; Back
        </button>
        <span className="conv-viewer-header-title">
          {sessionInfo?.title ?? 'Loading...'}
        </span>
        {sessionInfo && (
          <span className="conv-viewer-header-session-info">
            {sessionInfo.model}
          </span>
        )}
        <button
          className={`conv-viewer-header-btn${loading ? ' conv-viewer-header-btn--disabled' : ''}`}
          onClick={handleRefresh}
          disabled={loading}
          title="Refresh conversation"
        >
          <span className={loading ? 'conv-spinning' : ''}>&#8635;</span>
        </button>
      </div>

      {error && (
        <div className="conv-viewer-error">
          <span>{error}</span>
          <button className="conv-viewer-header-btn" onClick={handleRefresh}>Retry</button>
        </div>
      )}

      <div className="conv-viewer-body">
        {loading && messages.length === 0 && !error ? (
          <div className="conv-viewer-loading">
            <span className="conv-spinning">&#8635;</span> Loading conversation...
          </div>
        ) : messages.length === 0 && !error ? (
          <div className="conv-viewer-empty">No messages in this conversation</div>
        ) : (
          messages.map((msg, idx) => (
            <ConversationMessage key={`${msg.role}-${msg.timestamp}-${idx}`} message={msg} />
          ))
        )}
      </div>
    </div>
  )
}
