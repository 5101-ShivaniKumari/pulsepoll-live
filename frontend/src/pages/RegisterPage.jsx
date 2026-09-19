import React, { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useToast } from '../context/ToastContext'
import { UserPlus, Sparkles } from 'lucide-react'

export function RegisterPage() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const { register } = useAuth()
  const { success, error: toastError } = useToast()
  const navigate = useNavigate()

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!name.trim() || !email.trim() || !password) {
      toastError('Please fill in all fields')
      return
    }
    if (password.length < 6) {
      toastError('Password must be at least 6 characters')
      return
    }

    setLoading(true)
    try {
      await register(name.trim(), email.trim(), password)
      success('Account created! Welcome to PulsePoll.')
      navigate('/dashboard')
    } catch (err) {
      toastError(err.message || 'Registration failed. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const fillDemoAccount = () => {
    setName('Demo Creator')
    setEmail(`creator_${Math.floor(Math.random()*1000)}@pulsepoll.io`)
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
            <UserPlus size={24} color="var(--primary-light)" />
          </div>
          <h2 style={{ fontSize: '1.65rem', marginBottom: '0.35rem' }}>Create Account</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
            Start hosting live real-time polls in seconds
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="name">
              <span>Full Name</span>
            </label>
            <input
              id="name"
              type="text"
              required
              className="form-input"
              placeholder="Alex Smith"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="email">
              <span>Email Address</span>
            </label>
            <input
              id="email"
              type="email"
              required
              className="form-input"
              placeholder="alex@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          <div className="form-group">
            <label className="form-label" htmlFor="password">
              <span>Password (min. 6 characters)</span>
            </label>
            <input
              id="password"
              type="password"
              required
              minLength={6}
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
            {loading ? 'Creating Account...' : 'Sign Up'}
          </button>

          <button
            type="button"
            onClick={fillDemoAccount}
            className="btn btn-secondary btn-sm"
            style={{ width: '100%', fontSize: '0.85rem', display: 'flex', gap: '0.4rem', justifyContent: 'center' }}
          >
            <Sparkles size={14} color="var(--accent-cyan)" />
            <span>Generate Random Demo User</span>
          </button>
        </form>

        <div style={{ textAlign: 'center', marginTop: '1.75rem', borderTop: '1px solid var(--border-subtle)', paddingTop: '1.25rem' }}>
          <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>
            Already have an account?{' '}
            <Link to="/login" style={{ fontWeight: 600, color: 'var(--primary-light)' }}>
              Sign In
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
