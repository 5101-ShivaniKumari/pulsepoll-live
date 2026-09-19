import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { Activity, PlusCircle, LayoutDashboard, LogOut, LogIn, UserPlus } from 'lucide-react'

export function Navbar() {
  const { user, isAuthenticated, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/')
  }

  return (
    <header style={{
      borderBottom: '1px solid var(--border-subtle)',
      background: 'rgba(9, 13, 22, 0.85)',
      backdropFilter: 'blur(12px)',
      position: 'sticky',
      top: 0,
      zIndex: 100,
    }}>
      <div style={{
        maxWidth: '1120px',
        margin: '0 auto',
        padding: '0.9rem 1.25rem',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        {/* Brand Logo */}
        <Link to="/" style={{ display: 'flex', alignItems: 'center', gap: '0.65rem', textDecoration: 'none' }}>
          <div style={{
            width: '38px',
            height: '38px',
            borderRadius: '10px',
            background: 'linear-gradient(135deg, #6366f1 0%, #a855f7 100%)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            boxShadow: '0 0 16px rgba(99, 102, 241, 0.4)',
          }}>
            <Activity size={22} color="#ffffff" />
          </div>
          <div>
            <span style={{
              fontFamily: 'var(--font-heading)',
              fontWeight: 800,
              fontSize: '1.35rem',
              background: 'linear-gradient(135deg, #ffffff 0%, #cbd5e1 100%)',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
              letterSpacing: '-0.03em',
            }}>
              PulsePoll
            </span>
            <span style={{
              fontSize: '0.65rem',
              fontFamily: 'var(--font-mono)',
              color: 'var(--accent-cyan)',
              marginLeft: '0.4rem',
              fontWeight: 600,
              padding: '1px 5px',
              background: 'rgba(6, 182, 212, 0.12)',
              borderRadius: '4px',
              border: '1px solid rgba(6, 182, 212, 0.3)',
            }}>
              LIVE
            </span>
          </div>
        </Link>

        {/* Navigation & User actions */}
        <nav style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          {isAuthenticated ? (
            <>
              <Link to="/dashboard" className="btn btn-ghost btn-sm" style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <LayoutDashboard size={16} />
                <span>Dashboard</span>
              </Link>

              <Link to="/create" className="btn btn-primary btn-sm" style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <PlusCircle size={16} />
                <span>New Poll</span>
              </Link>

              <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', marginLeft: '0.5rem' }}>
                <span style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  {user?.name}
                </span>
                <button onClick={handleLogout} className="btn btn-ghost btn-sm" title="Log out">
                  <LogOut size={16} />
                </button>
              </div>
            </>
          ) : (
            <>
              <Link to="/login" className="btn btn-ghost btn-sm" style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <LogIn size={16} />
                <span>Log In</span>
              </Link>

              <Link to="/register" className="btn btn-primary btn-sm" style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <UserPlus size={16} />
                <span>Sign Up</span>
              </Link>
            </>
          )}
        </nav>
      </div>
    </header>
  )
}
