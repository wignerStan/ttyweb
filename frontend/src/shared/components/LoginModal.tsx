import { useState, FormEvent } from 'react';
import { Terminal } from 'lucide-react';
import { login } from '../../utils/auth';
import './LoginModal.css';

interface Props {
  onLogin: () => void;
}

export function LoginModal({ onLogin }: Props) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    const result = await login(username, password);
    setLoading(false);
    if (result.success) {
      onLogin();
    } else {
      setError(result.error || 'Login failed');
    }
  }

  return (
    <div className="login-overlay">
      <form className="login-modal" onSubmit={handleSubmit}>
        <div className="login-header">
          <div className="login-icon">
            <Terminal size={32} />
          </div>
          <h1>TmuxWeb</h1>
        </div>

        <div className="login-field">
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Username"
            autoFocus
            disabled={loading}
          />
        </div>

        <div className="login-field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            disabled={loading}
          />
        </div>

        {error && <div className="login-error">{error}</div>}

        <button type="submit" className="login-submit" disabled={loading || !username || !password}>
          {loading ? 'Authenticating...' : 'Sign In'}
        </button>
      </form>
    </div>
  );
}
