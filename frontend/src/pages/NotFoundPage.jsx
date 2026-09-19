import React from 'react'
import { Link } from 'react-router-dom'
import { HelpCircle } from 'lucide-react'

export function NotFoundPage() {
  return (
    <div style={{ textAlign: 'center', padding: '5rem 1rem' }}>
      <HelpCircle size={56} color="var(--primary-light)" style={{ margin: '0 auto 1rem' }} />
      <h1 style={{ fontSize: '2.5rem', marginBottom: '0.5rem' }}>404 — Page Not Found</h1>
      <p style={{ color: 'var(--text-secondary)', marginBottom: '1.75rem' }}>
        The link you followed doesn't exist or has moved.
      </p>
      <Link to="/" className="btn btn-primary">
        Return to Home
      </Link>
    </div>
  )
}
