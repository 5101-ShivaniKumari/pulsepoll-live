import React, { useState, useEffect, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../services/api'
import { useAuth } from '../context/AuthContext'
import { useToast } from '../context/ToastContext'
import { StatusBadge } from '../components/StatusBadge'
import { ShareModal } from '../components/ShareModal'
import { ConfirmationModal } from '../components/ConfirmationModal'
import { PlusCircle, BarChart3, Vote, Share2, Lock, Trash2, Calendar, Users, Sparkles, AlertCircle } from 'lucide-react'

export function DashboardPage() {
  const { user } = useAuth()
  const { success, error: toastError } = useToast()

  const [polls, setPolls] = useState([])
  const [loading, setLoading] = useState(true)
  const [selectedPollForShare, setSelectedPollForShare] = useState(null)
  const [pollToDelete, setPollToDelete] = useState(null)
  const [pollToClose, setPollToClose] = useState(null)
  const [actionLoading, setActionLoading] = useState(false)

  const fetchPolls = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.polls.getMyPolls()
      setPolls(data.polls || [])
    } catch (err) {
      toastError(err.message || 'Failed to fetch polls')
    } finally {
      setLoading(false)
    }
  }, [toastError])

  useEffect(() => {
    fetchPolls()
  }, [fetchPolls])

  const handleClosePoll = async () => {
    if (!pollToClose) return
    setActionLoading(true)
    try {
      await api.polls.close(pollToClose.id)
      success('Poll closed successfully.')
      setPollToClose(null)
      fetchPolls()
    } catch (err) {
      toastError(err.message || 'Failed to close poll')
    } finally {
      setActionLoading(false)
    }
  }

  const handleDeletePoll = async () => {
    if (!pollToDelete) return
    setActionLoading(true)
    try {
      await api.polls.delete(pollToDelete.id)
      success('Poll deleted successfully.')
      setPollToDelete(null)
      fetchPolls()
    } catch (err) {
      toastError(err.message || 'Failed to delete poll')
    } finally {
      setActionLoading(false)
    }
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
      {/* Dashboard Top Header */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: '1rem',
      }}>
        <div>
          <h1 style={{ fontSize: '2rem', marginBottom: '0.35rem' }}>Creator Dashboard</h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem' }}>
            Welcome back, <strong style={{ color: 'var(--text-primary)' }}>{user?.name}</strong>. Manage and monitor your live polls.
          </p>
        </div>

        <Link
          to="/create"
          className="btn btn-primary"
          style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}
        >
          <PlusCircle size={18} />
          <span>Create New Poll</span>
        </Link>
      </div>

      {/* Polls Listing */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '4rem 0' }}>
          <div className="pulse-dot" style={{ width: '14px', height: '14px', margin: '0 auto 1rem' }} />
          <p style={{ color: 'var(--text-secondary)' }}>Loading your polls...</p>
        </div>
      ) : polls.length === 0 ? (
        <div className="glass-card" style={{ padding: '3.5rem 2rem', textAlign: 'center' }}>
          <Sparkles size={40} color="var(--primary-light)" style={{ margin: '0 auto 1rem' }} />
          <h2 style={{ fontSize: '1.4rem', marginBottom: '0.5rem' }}>No polls yet</h2>
          <p style={{ color: 'var(--text-secondary)', maxWidth: '440px', margin: '0 auto 1.75rem', fontSize: '0.95rem' }}>
            Launch your first live poll in seconds and share the link with participants.
          </p>
          <Link to="/create" className="btn btn-primary">
            Create a Live Poll
          </Link>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          {polls.map((p) => {
            const formattedDate = new Date(p.created_at).toLocaleDateString('en-US', {
              month: 'short',
              day: 'numeric',
              year: 'numeric',
            })

            return (
              <div
                key={p.id}
                className="glass-card"
                style={{
                  padding: '1.5rem',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  flexWrap: 'wrap',
                  gap: '1.25rem',
                }}
              >
                <div style={{ flex: '1 1 340px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.65rem', marginBottom: '0.6rem' }}>
                    <StatusBadge isClosed={p.is_closed} totalVotes={p.total_votes} />
                    <span style={{ display: 'flex', alignItems: 'center', gap: '0.3rem', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                      <Calendar size={13} />
                      <span>{formattedDate}</span>
                    </span>
                  </div>

                  <h3 style={{ fontSize: '1.25rem', marginBottom: '0.45rem', lineHeight: '1.3' }}>
                    {p.title}
                  </h3>

                  <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                    <span style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                      <Users size={14} color="var(--accent-cyan)" />
                      <strong style={{ color: 'var(--text-primary)' }}>{p.total_votes}</strong> {p.total_votes === 1 ? 'vote' : 'votes'}
                    </span>
                    <span>•</span>
                    <span>{p.options?.length || 0} Choices</span>
                  </div>
                </div>

                {/* Actions */}
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                  <Link
                    to={`/poll/${p.id}/results`}
                    className="btn btn-primary btn-sm"
                    style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}
                  >
                    <BarChart3 size={15} />
                    <span>Live Results</span>
                  </Link>

                  {!p.is_closed && (
                    <Link
                      to={`/poll/${p.id}`}
                      className="btn btn-secondary btn-sm"
                      style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}
                    >
                      <Vote size={15} />
                      <span>Vote</span>
                    </Link>
                  )}

                  <button
                    onClick={() => setSelectedPollForShare(p)}
                    className="btn btn-secondary btn-sm"
                    style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}
                    title="Share & QR Code"
                  >
                    <Share2 size={15} />
                    <span>Share</span>
                  </button>

                  {!p.is_closed && (
                    <button
                      onClick={() => setPollToClose(p)}
                      className="btn btn-ghost btn-sm"
                      title="Close poll to new votes"
                      style={{ color: 'var(--accent-amber)' }}
                    >
                      <Lock size={15} />
                    </button>
                  )}

                  <button
                    onClick={() => setPollToDelete(p)}
                    className="btn btn-ghost btn-sm"
                    title="Delete poll"
                    style={{ color: 'var(--accent-rose)' }}
                  >
                    <Trash2 size={15} />
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Modals */}
      <ShareModal
        poll={selectedPollForShare}
        isOpen={!!selectedPollForShare}
        onClose={() => setSelectedPollForShare(null)}
      />

      <ConfirmationModal
        isOpen={!!pollToClose}
        title="Close this Live Poll?"
        message={`Are you sure you want to close "${pollToClose?.title}"? No further votes will be accepted.`}
        confirmText="Close Poll"
        confirmVariant="danger"
        loading={actionLoading}
        onConfirm={handleClosePoll}
        onCancel={() => setPollToClose(null)}
      />

      <ConfirmationModal
        isOpen={!!pollToDelete}
        title="Delete this Poll Permanently?"
        message={`Are you sure you want to permanently delete "${pollToDelete?.title}" and all its recorded votes? This action cannot be undone.`}
        confirmText="Delete Poll"
        confirmVariant="danger"
        loading={actionLoading}
        onConfirm={handleDeletePoll}
        onCancel={() => setPollToDelete(null)}
      />
    </div>
  )
}
