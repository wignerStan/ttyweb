import { Terminal } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { login } from '../../utils/auth'

interface Props {
  onLogin: () => void
}

export function LoginModal({ onLogin }: Props) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    const result = await login(username, password)
    setLoading(false)
    if (result.success) {
      onLogin()
    } else {
      setError(result.error || 'Login failed')
    }
  }

  return (
    <div className="fixed inset-0 bg-base-100 flex items-center justify-center">
      <form
        className="w-full max-w-[360px] px-10 py-12 bg-base-200 border border-base-300 rounded-2xl shadow-lg"
        onSubmit={handleSubmit}
      >
        <div className="text-center mb-9">
          <div className="mb-3 flex items-center justify-center text-primary">
            <Terminal size={32} />
          </div>
          <h1 className="text-2xl font-semibold text-base-content tracking-tight">TmuxWeb</h1>
        </div>

        <div className="mb-6">
          <label
            htmlFor="username"
            className="block text-xs font-medium text-base-content/50 uppercase tracking-widest mb-2"
          >
            Username
          </label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Username"
            disabled={loading}
            className="input input-bordered w-full"
          />
        </div>

        <div className="mb-6">
          <label
            htmlFor="password"
            className="block text-xs font-medium text-base-content/50 uppercase tracking-widest mb-2"
          >
            Password
          </label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            disabled={loading}
            className="input input-bordered w-full"
          />
        </div>

        {error && (
          <div className="px-3.5 py-3 mb-5 text-sm text-error bg-error/10 border border-error/20 rounded-lg">
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={loading || !username || !password}
          className="btn btn-primary w-full"
        >
          {loading ? 'Authenticating...' : 'Sign In'}
        </button>
      </form>
    </div>
  )
}
