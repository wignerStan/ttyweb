import { Pencil, Plus, Trash2, X } from 'lucide-react'
import { useCallback, useState } from 'react'
import type { AiRole } from '../../types'
import { getAuthHeader } from '../../utils/auth'
import './RoleManagerModal.css'

interface RoleFormData {
  id: string
  emoji: string
  label: string
  desc: string
  prompt: string
  suffix: string
  model: string
  apiUrl: string
}

const emptyForm: RoleFormData = {
  id: '',
  emoji: '',
  label: '',
  desc: '',
  prompt: '',
  suffix: '',
  model: '',
  apiUrl: '',
}

interface RoleManagerModalProps {
  open: boolean
  onClose: () => void
  roles: AiRole[]
  onRolesChanged: () => void
}

export function RoleManagerModal({ open, onClose, roles, onRolesChanged }: RoleManagerModalProps) {
  const [editingRole, setEditingRole] = useState<RoleFormData | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [showLlmConfig, setShowLlmConfig] = useState(false)

  const handleSaveRole = useCallback(
    async (form: RoleFormData) => {
      try {
        const method = isCreating ? 'POST' : 'PUT'
        const url = isCreating ? '/api/roles' : `/api/roles/${form.id}`
        const auth = getAuthHeader()
        const headers: Record<string, string> = { 'Content-Type': 'application/json' }
        if (auth) headers.Authorization = auth

        const body: Record<string, string> = {
          name: form.label,
          description: form.desc,
          system_prompt: form.prompt,
        }
        if (form.suffix) body.suffix = form.suffix
        if (form.model) body.model = form.model
        if (form.apiUrl) body.api_url = form.apiUrl

        const res = await fetch(url, {
          method,
          headers,
          body: JSON.stringify(body),
        })
        if (res.ok) {
          await onRolesChanged()
          setEditingRole(null)
          setIsCreating(false)
          setShowLlmConfig(false)
        }
      } catch {
        /* ignore */
      }
    },
    [isCreating, onRolesChanged],
  )

  const handleDeleteRole = useCallback(
    async (id: string) => {
      try {
        const auth = getAuthHeader()
        const headers: Record<string, string> = {}
        if (auth) headers.Authorization = auth
        const res = await fetch(`/api/roles/${id}`, {
          method: 'DELETE',
          headers,
        })
        if (res.ok) {
          await onRolesChanged()
        }
      } catch {
        /* ignore */
      }
    },
    [onRolesChanged],
  )

  const handleStartCreate = useCallback(() => {
    setEditingRole({ ...emptyForm })
    setIsCreating(true)
    setShowLlmConfig(false)
  }, [])

  const handleStartEdit = useCallback((r: AiRole) => {
    setEditingRole({
      id: r.id,
      emoji: r.emoji,
      label: r.label,
      desc: r.desc,
      prompt: r.prompt || '',
      suffix: r.suffix || '',
      model: r.model || '',
      apiUrl: r.apiUrl || '',
    })
    setIsCreating(false)
    setShowLlmConfig(!!(r.model || r.apiUrl))
  }, [])

  const handleCancelEdit = useCallback(() => {
    setEditingRole(null)
    setIsCreating(false)
    setShowLlmConfig(false)
  }, [])

  if (!open) return null

  const customRoles = roles.filter((r) => r.isCustom)

  return (
    <>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: overlay click-outside-to-close pattern */}
      <div className="role-modal-overlay" role="presentation" onClick={onClose}>
        {/* biome-ignore lint/a11y/noStaticElementInteractions: modal stopPropagation pattern */}
        <div className="role-modal" role="presentation" onClick={(e) => e.stopPropagation()}>
          {/* Header */}
          <div className="role-modal-header">
            <span className="role-modal-title">自定义角色</span>
            <button className="role-modal-close" onClick={onClose} type="button">
              <X size={16} />
            </button>
          </div>

          {/* Role list */}
          <div className="role-modal-body">
            {customRoles.length === 0 && !editingRole && (
              <div className="role-modal-empty">暂无自定义角色</div>
            )}

            {customRoles.map((r) => (
              <div key={r.id} className="role-modal-item">
                <div className="role-modal-item-info">
                  <span className="role-modal-item-emoji">{r.emoji}</span>
                  <div className="role-modal-item-text">
                    <span className="role-modal-item-label">{r.label}</span>
                    <span className="role-modal-item-desc">{r.desc}</span>
                    {r.model && (
                      <span className="role-modal-item-desc" style={{ color: '#4d78cc' }}>
                        {r.model}
                      </span>
                    )}
                  </div>
                </div>
                <div className="role-modal-item-actions">
                  <button
                    onClick={() => handleStartEdit(r)}
                    className="role-modal-action-btn"
                    type="button"
                    title="编辑"
                  >
                    <Pencil size={13} />
                  </button>
                  <button
                    onClick={() => handleDeleteRole(r.id)}
                    className="role-modal-action-btn role-modal-delete-btn"
                    type="button"
                    title="删除"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
            ))}

            {/* Edit / Create form */}
            {editingRole && (
              <div className="role-modal-form">
                <div className="role-modal-form-header">
                  <span>{isCreating ? '新建角色' : '编辑角色'}</span>
                  <button
                    onClick={handleCancelEdit}
                    className="role-modal-action-btn"
                    type="button"
                  >
                    <X size={14} />
                  </button>
                </div>
                {isCreating && (
                  <input
                    placeholder="ID (英文)"
                    value={editingRole.id}
                    onChange={(e) => setEditingRole({ ...editingRole, id: e.target.value })}
                    className="role-modal-input"
                  />
                )}
                <div className="role-modal-row">
                  <input
                    placeholder="Emoji"
                    value={editingRole.emoji}
                    onChange={(e) => setEditingRole({ ...editingRole, emoji: e.target.value })}
                    className="role-modal-input role-modal-input-emoji"
                  />
                  <input
                    placeholder="名称"
                    value={editingRole.label}
                    onChange={(e) => setEditingRole({ ...editingRole, label: e.target.value })}
                    className="role-modal-input"
                    style={{ flex: 1 }}
                  />
                </div>
                <input
                  placeholder="描述"
                  value={editingRole.desc}
                  onChange={(e) => setEditingRole({ ...editingRole, desc: e.target.value })}
                  className="role-modal-input"
                />
                <textarea
                  placeholder="系统提示词"
                  value={editingRole.prompt}
                  onChange={(e) => setEditingRole({ ...editingRole, prompt: e.target.value })}
                  rows={4}
                  className="role-modal-textarea"
                />
                <textarea
                  placeholder="后缀提示词"
                  value={editingRole.suffix}
                  onChange={(e) => setEditingRole({ ...editingRole, suffix: e.target.value })}
                  rows={2}
                  className="role-modal-textarea"
                />

                {/* LLM Config section */}
                <button
                  type="button"
                  onClick={() => setShowLlmConfig(!showLlmConfig)}
                  className="role-modal-llm-toggle"
                >
                  {showLlmConfig ? '▼' : '▶'} LLM 配置 (可选)
                </button>

                {showLlmConfig && (
                  <div className="role-modal-llm-config">
                    <input
                      placeholder="模型 (例如: gpt-4, claude-3)"
                      value={editingRole.model}
                      onChange={(e) => setEditingRole({ ...editingRole, model: e.target.value })}
                      className="role-modal-input"
                    />
                    <input
                      placeholder="API URL (例如: https://api.openai.com/v1)"
                      value={editingRole.apiUrl}
                      onChange={(e) => setEditingRole({ ...editingRole, apiUrl: e.target.value })}
                      className="role-modal-input"
                    />
                    <div className="role-modal-llm-hint">
                      留空则使用服务器默认配置 (环境变量 OPENAI_API_URL / OPENAI_MODEL)
                    </div>
                  </div>
                )}

                <button
                  onClick={() => handleSaveRole(editingRole)}
                  className="role-modal-save-btn"
                  type="button"
                >
                  保存
                </button>
              </div>
            )}
          </div>

          {/* Footer */}
          {!editingRole && (
            <div className="role-modal-footer">
              <button onClick={handleStartCreate} className="role-modal-create-btn" type="button">
                <Plus size={14} /> 新建角色
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  )
}
