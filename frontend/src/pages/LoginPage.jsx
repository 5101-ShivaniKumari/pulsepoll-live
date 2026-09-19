import React, { useState } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useToast } from '../context/ToastContext'
import { LogIn, Key, Mail, Sparkles } from 'lucide-react'

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const { success, error: toastError } = useToast()
  const navigate = useNavigate()
  const location = useLocation()

  const from = location.state?.from?.pathname || '/dashboard'

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!email || !password) {
      toastError('Please fill in all fields')
      return
    }

    setLoading(true)
    try {
      await login(email, password)
      success('Logged in successfully!')
      navigate(from, { replace: true })
    } catch (err) {
      toastError(err.message || 'Login failed. Please check your credentials.')
    } finally {
      setLoading(false)
    }
  }

  const fillDemoAccount = () => {
    setEmail('demo@pulsepoll.io')
    setPassword('demopassword123')
  }

  return (
    <div style={{ maxWidth: '440px', margin: '2rem auto 0' }}>
      <div className="glass-card" style={{ padding: '2.25rem 2rem' }}>
        <div style={{ textAlign: 'center', marginBottom: '1.75rem' }}>
          <div style={{
            width: '48px',
            height: '48px',
            borderRadius: '12px',
            background: 'rgba(99, 102, 241, 0.15)',
            border: '1px solid rgba(99, 102, 241, 0.3)',
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '0.75rem',
          }}>
            <LogIn size={24} color="var(--primary-light)" />
          </div>
          <h2 style={{ fontSize: '1.65rem', marginBottom: '0.35rem' }}>Welcome Back</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
            Log in to manage your active live polls
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="email">
              <span>Email Address</span>
            </label>
            <div style={{ position: 'relative' }}>
              <input
                id="email"
                type="email"
                required
                className="form-input"
                placeholder="you@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="password">
              <span>Password</span>
            </label>
            <input
              id="password"
              type="password"
              required
              className="form-input"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="btn btn-primary"
            style={{ width: '100%', marginTop: '0.5rem', marginBottom: '1rem' }}
          >
            {loading ? 'Authenticating...' : 'Sign In'}
          </button>

          <button
            type="button"
            onClick={fillDemoAccount}
            className="btn btn-secondary btn-sm"
            style={{ width: '100%', fontSize: '0.85rem', display: 'flex', gap: '0.4rem', justifyContent: 'center' }}
          >
            <Sparkles size={14} color="var(--accent-cyan)" />
            <span>Fill Demo Credentials</span>
          </button>
        </form>

        <div style={{ textAlign: 'center', marginTop: '1.75rem', borderTop: '1px solid var(--border-subtle)', paddingTop: '1.25rem' }}>
          <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>
            Don't have an account yet?{' '}
            <Link to="/register" style={{ fontWeight: 600, color: 'var(--primary-light)' }}>
              Create an account
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
