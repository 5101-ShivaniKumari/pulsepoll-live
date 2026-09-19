import React from 'react'
import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './context/AuthContext'
import { Navbar } from './components/Navbar'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { DashboardPage } from './pages/DashboardPage'
import { CreatePollPage } from './pages/CreatePollPage'
import { PollVotePage } from './pages/PollVotePage'
import { PollResultsPage } from './pages/PollResultsPage'
import { NotFoundPage } from './pages/NotFoundPage'

function ProtectedRoute({ children }) {
  const { isAuthenticated, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '5rem 0' }}>
        <div className="pulse-dot" style={{ width: '16px', height: '16px', margin: '0 auto 1rem' }} />
        <p style={{ color: 'var(--text-secondary)' }}>Checking session...</p>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return children
}

export function App() {
  return (
    <div className="app-container">
      <Navbar />

      <main className="main-content">
        <Routes>
          {/* Public Home & Auth */}
          <Route path="/" element={<HomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />

          {/* Protected Creator Routes */}
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/create"
            element={
              <ProtectedRoute>
                <CreatePollPage />
              </ProtectedRoute>
            }
          />

          {/* Public Voter & Live Results Routes */}
          <Route path="/poll/:id" element={<PollVotePage />} />
          <Route path="/poll/:id/results" element={<PollResultsPage />} />

          {/* 404 Fallback */}
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </main>

      <footer style={{
        borderTop: '1px solid var(--border-subtle)',
        padding: '2rem 1.25rem',
        textAlign: 'center',
        fontSize: '0.85rem',
        color: 'var(--text-muted)',
        background: 'transparent',
        transition: 'border-color var(--transition-normal)',
      }}>
        <div style={{ maxWidth: '1120px', margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '1rem' }}>
          <div>
            <strong style={{ color: 'var(--text-secondary)' }}>PulsePoll</strong> — Real-Time Live Polling System
          </div>
          <div>
            Built with Go (Gin), Redis Pub/Sub, WebSockets, MongoDB &amp; React
          </div>
        </div>
      </footer>
    </div>
  )
}
