import React from 'react'
import { AlertTriangle, X } from 'lucide-react'

export function ConfirmationModal({ isOpen, title, message, confirmText = 'Confirm', confirmVariant = 'danger', onConfirm, onCancel, loading = false }) {
  if (!isOpen) return null

  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: confirmVariant === 'danger' ? 'var(--accent-rose)' : 'var(--accent-amber)' }}>
            <AlertTriangle size={20} />
            <h3 style={{ margin: 0, fontSize: '1.2rem', color: 'var(--text-primary)' }}>{title}</h3>
          </div>
          <button
            onClick={onCancel}
            className="btn-ghost"
            style={{ padding: '4px', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', borderRadius: 'var(--radius-sm)' }}
          >
            <X size={18} />
          </button>
        </div>

        <p style={{ fontSize: '0.92rem', color: 'var(--text-secondary)', marginBottom: '1.5rem', lineHeight: '1.5' }}>
          {message}
        </p>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.75rem' }}>
          <button onClick={onCancel} disabled={loading} className="btn btn-secondary btn-sm">
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={`btn ${confirmVariant === 'danger' ? 'btn-danger' : 'btn-primary'} btn-sm`}
          >
            {loading ? 'Processing...' : confirmText}
          </button>
        </div>
      </div>
    </div>
  )
}
