import { useState } from 'react'
import { getAuthHeaders } from '../../../../utils/auth'
import { BUTLER_API_BASE } from '../constants'
import type { ReplyDecision } from '../types'

export function useReplyInbox() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const submitReply = async (
    inboxItemId: string,
    studyId: string,
    decision: ReplyDecision,
    message: string,
  ) => {
    setLoading(true)
    setError(null)
    try {
      const jsonHeaders: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }

      const replyRes = await fetch(`${BUTLER_API_BASE}/approval_replies`, {
        method: 'POST',
        headers: jsonHeaders,
        body: JSON.stringify({ inbox_item_id: inboxItemId, study_id: studyId, decision, message }),
      })
      if (!replyRes.ok) throw new Error(`Reply POST failed: HTTP ${replyRes.status}`)

      const updateRes = await fetch(`${BUTLER_API_BASE}/inbox_items/${inboxItemId}`, {
        method: 'PUT',
        headers: jsonHeaders,
        body: JSON.stringify({ status: 'replied' }),
      })
      if (!updateRes.ok) throw new Error(`Status PUT failed: HTTP ${updateRes.status}`)
    } catch (e: unknown) {
      const err = e instanceof Error ? e : new Error(String(e))
      setError(err)
      throw err
    } finally {
      setLoading(false)
    }
  }

  return { submitReply, loading, error }
}
