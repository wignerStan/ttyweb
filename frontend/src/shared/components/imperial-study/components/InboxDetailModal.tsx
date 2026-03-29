// InboxDetailModal.tsx — Inbox item detail overlay
import { useState } from 'react';
import { X } from 'lucide-react';
import type { InboxItem, ReplyDecision } from '../types';
import { INBOX_KIND_CONFIG, INBOX_KIND_DEFAULT_ICON, INBOX_KIND_DEFAULT_COLOR } from '../constants';
import { useReplyInbox } from '../hooks/useReplyInbox';

interface InboxDetailModalProps {
    item: InboxItem;
    onClose: () => void;
    onReplied: () => void;
}

export function InboxDetailModal({ item, onClose, onReplied }: InboxDetailModalProps) {
    const [replyText, setReplyText] = useState('');
    const { submitReply, loading } = useReplyInbox();

    const cfg = INBOX_KIND_CONFIG[item.kind];
    const Icon = cfg?.icon ?? INBOX_KIND_DEFAULT_ICON;
    const iconColor = cfg?.color ?? INBOX_KIND_DEFAULT_COLOR;

    const handleAction = async (decision: ReplyDecision) => {
        try {
            await submitReply(item.id, item.study_id, decision, replyText);
            onReplied();
        } catch {
            // error surfaced by hook; keep modal open
        }
    };

    const handleOverlayClick = (e: React.MouseEvent<HTMLDivElement>) => {
        if (e.target === e.currentTarget) onClose();
    };

    return (
        <div className="is-modal-overlay" onClick={handleOverlayClick}>
            <div className="is-modal">
                <div className="is-modal__header">
                    <span className="is-modal__header-title">Inbox Detail</span>
                    <button
                        className="is-icon-btn is-modal__close"
                        onClick={onClose}
                        title="Close"
                    >
                        <X size={18} />
                    </button>
                </div>

                <div className="is-modal__meta">
                    <div className="is-modal__meta-row">
                        <span className="is-modal__meta-label">Kind:</span>
                        <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <Icon size={14} style={{ color: iconColor }} />
                            {item.kind}
                        </span>
                    </div>
                    <div className="is-modal__meta-row">
                        <span className="is-modal__meta-label">From:</span>
                        <span>{item.worker_id}</span>
                    </div>
                    <div className="is-modal__meta-row">
                        <span className="is-modal__meta-label">Time:</span>
                        <span>
                            {item.created_at
                                ? new Date(item.created_at).toLocaleString()
                                : '\u2014'}
                        </span>
                    </div>
                </div>

                <div className="is-modal__body">{item.body}</div>

                <div className="is-modal__reply">
                    <textarea
                        className="is-modal__textarea"
                        value={replyText}
                        onChange={e => setReplyText(e.target.value)}
                        placeholder="Reply message..."
                        disabled={loading}
                        rows={3}
                    />
                </div>

                <div className="is-modal__actions">
                    <button
                        className="is-btn is-btn--approve"
                        onClick={() => handleAction('approved')}
                        disabled={loading}
                    >
                        Approve
                    </button>
                    <button
                        className="is-btn is-btn--reject"
                        onClick={() => handleAction('rejected')}
                        disabled={loading}
                    >
                        Reject
                    </button>
                    <button
                        className="is-btn is-btn--reply"
                        onClick={() => handleAction('custom')}
                        disabled={loading || !replyText.trim()}
                        style={{ marginLeft: 'auto' }}
                    >
                        Reply
                    </button>
                </div>
            </div>
        </div>
    );
}
