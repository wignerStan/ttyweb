import { useState, useEffect, type FormEvent, type KeyboardEvent } from 'react';
import { Send, MessageSquare } from 'lucide-react';
import type { KanbanComment } from './types';

interface CommentThreadProps {
  taskId: string;
  fetchComments: (taskId: string) => Promise<KanbanComment[]>;
  createComment: (taskId: string, content: string) => Promise<KanbanComment | null>;
}

function formatTimestamp(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function CommentThread({ taskId, fetchComments, createComment }: CommentThreadProps) {
  const [comments, setComments] = useState<KanbanComment[]>([]);
  const [newComment, setNewComment] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchComments(taskId).then((data) => {
      if (!cancelled) {
        setComments(data);
        setLoading(false);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [taskId, fetchComments]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = newComment.trim();
    if (!trimmed || submitting) return;

    setSubmitting(true);
    const created = await createComment(taskId, trimmed);
    if (created) {
      setComments((prev) => [...prev, created]);
      setNewComment('');
    }
    setSubmitting(false);
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      void handleSubmit(e);
    }
  }

  return (
    <div className="comment-thread">
      <h4 className="task-dialog__label" style={{ marginBottom: 4 }}>
        <MessageSquare size={12} style={{ display: 'inline', marginRight: 4, verticalAlign: 'middle' }} />
        Comments
      </h4>

      {loading ? (
        <div className="comment-thread__empty">Loading...</div>
      ) : comments.length === 0 ? (
        <div className="comment-thread__empty">No comments yet</div>
      ) : (
        <div className="comment-thread__list">
          {comments.map((comment) => (
            <div key={comment.id} className="comment-thread__item">
              <div className="comment-thread__content">{comment.content}</div>
              <div className="comment-thread__timestamp">
                {formatTimestamp(comment.created_at)}
              </div>
            </div>
          ))}
        </div>
      )}

      <form className="comment-thread__add" onSubmit={handleSubmit}>
        <textarea
          className="comment-thread__input"
          value={newComment}
          onChange={(e) => setNewComment(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Add a comment... (Cmd+Enter to submit)"
          rows={1}
          disabled={submitting}
        />
        <button
          className="comment-thread__submit"
          type="submit"
          disabled={submitting || !newComment.trim()}
          title="Send comment"
        >
          <Send size={14} />
        </button>
      </form>
    </div>
  );
}

export { CommentThread };
export type { CommentThreadProps };
