import React, { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import confetti from 'canvas-confetti'
import { api } from '../services/api'
import { useToast } from '../context/ToastContext'
import { useWebSocket } from '../hooks/useWebSocket'
import { StatusBadge } from '../components/StatusBadge'
import { CheckCircle2, Lock, BarChart3, AlertCircle, ArrowRight, Share2, Clock } from 'lucide-react'
import { ShareModal } from '../components/ShareModal'

export function PollVotePage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { success, error: toastError, info } = useToast()

  const [poll, setPoll] = useState(null)
  const [selectedOption, setSelectedOption] = useState('')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [shareModalOpen, setShareModalOpen] = useState(false)

  const loadPoll = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.polls.get(id)
      setPoll(data)
      if (data.has_voted && data.voted_for) {
        setSelectedOption(data.voted_for)
      }
    } catch (err) {
      toastError(err.message || 'Poll not found')
    } finally {
      setLoading(false)
    }
  }, [id, toastError])

  useEffect(() => {
    loadPoll()
  }, [loadPoll])

  // Real-time WebSocket listener so voter tab auto-closes when scheduled close time arrives
  const handleWebSocketMessage = useCallback((msg) => {
    if (!msg) return
    if (msg.type === 'poll_closed') {
      setPoll(prev => prev ? { ...prev, is_closed: true } : prev)
      info('This poll has reached its scheduled close time.')
    }
  }, [info])

  useWebSocket(id, handleWebSocketMessage)

  const handleVote = async (e) => {
    e.preventDefault()
    if (!selectedOption) {
      toastError('Please select an option before submitting')
      return
    }

    setSubmitting(true)
    try {
      await api.polls.vote(id, selectedOption)

      // Trigger celebratory confetti burst
      try {
        confetti({
          particleCount: 80,
          spread: 70,
          origin: { y: 0.7 },
          colors: ['#6366f1', '#06b6d4', '#10b981', '#f59e0b', '#a855f7'],
        })
      } catch (confettiErr) {
        // Safe fallback
      }

      success('Your vote has been counted in real time!')
      navigate(`/poll/${id}/results`, { state: { voted: true, votedOptionId: selectedOption } })
    } catch (err) {
      if (err.status === 409) {
        toastError('You have already voted on this poll.')
        setTimeout(() => navigate(`/poll/${id}/results`), 1200)
      } else {
        toastError(err.message || 'Failed to submit vote')
      }
    } finally {
      setSubmitting(false)
    }
  }

  // Format scheduled close time in user's local timezone
  const formatCloseTime = (expiryString) => {
    if (!expiryString) return null
    const date = new Date(expiryString)
    if (isNaN(date.getTime())) return null
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    })
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '5rem 0' }}>
        <div className="pulse-dot" style={{ width: '16px', height: '16px', margin: '0 auto 1rem' }} />
        <p style={{ color: 'var(--text-secondary)' }}>Loading live poll...</p>
      </div>
    )
  }

  if (!poll) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 1rem' }}>
        <AlertCircle size={48} color="var(--accent-rose)" style={{ margin: '0 auto 1rem' }} />
        <h2>Poll Not Found</h2>
        <p style={{ color: 'var(--text-secondary)', marginTop: '0.5rem', marginBottom: '1.5rem' }}>
          This poll may have been deleted or the link is incorrect.
        </p>
        <Link to="/" className="btn btn-secondary">
          Return Home
        </Link>
      </div>
    )
  }

  const formattedCloseTime = formatCloseTime(poll.expiry_at)

  return (
    <div style={{ maxWidth: '640px', margin: '1rem auto 0' }}>
      <div className="glass-card" style={{ padding: '2.25rem 2rem' }}>
        {/* Top Header info */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.75rem', marginBottom: '1.5rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.65rem', flexWrap: 'wrap' }}>
            <StatusBadge isClosed={poll.is_closed} totalVotes={poll.total_votes} />

            {/* Scheduled Auto-Close Time Pill */}
            {!poll.is_closed && formattedCloseTime && (
              <span className="badge" style={{
                background: 'rgba(245, 158, 11, 0.1)',
                color: 'var(--accent-amber)',
                border: '1px solid rgba(245, 158, 11, 0.28)',
                textTransform: 'none',
                fontSize: '0.78rem',
                fontFamily: 'var(--font-sans)',
              }}>
                <Clock size={13} style={{ marginRight: '2px' }} />
                <span>Closes {formattedCloseTime}</span>
              </span>
            )}
          </div>

          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <button
              type="button"
              onClick={() => setShareModalOpen(true)}
              className="btn btn-ghost btn-sm"
              title="Share Poll"
            >
              <Share2 size={16} />
              <span>Share</span>
            </button>

            <Link
              to={`/poll/${poll.id}/results`}
              className="btn btn-secondary btn-sm"
              style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}
            >
              <BarChart3 size={16} />
              <span>Live Results</span>
            </Link>
          </div>
        </div>

        {/* Question Title */}
        <h1 style={{ fontSize: '1.75rem', marginBottom: '0.65rem', lineHeight: 1.3 }}>
          {poll.title}
        </h1>

        {poll.description && (
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem', marginBottom: '1.75rem', lineHeight: 1.5 }}>
            {poll.description}
          </p>
        )}

        {/* Closed or Expired State Notice */}
        {poll.is_closed ? (
          <div style={{
            background: 'var(--closed-bg)',
            border: '1px solid var(--closed-border)',
            borderRadius: 'var(--radius-md)',
            padding: '1.5rem',
            textAlign: 'center',
            margin: '1.5rem 0',
          }}>
            <Lock size={32} color="var(--text-muted)" style={{ margin: '0 auto 0.75rem' }} />
            <h3 style={{ fontSize: '1.2rem', marginBottom: '0.35rem' }}>Voting Closed</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', marginBottom: '1.25rem' }}>
              This poll is no longer accepting new votes. You can view the final verified results below.
            </p>
            <Link to={`/poll/${poll.id}/results`} className="btn btn-primary btn-md">
              View Results Screen
            </Link>
          </div>
        ) : poll.has_voted ? (
          /* Already Voted Notice */
          <div style={{
            background: 'var(--voted-bg)',
            border: '1px solid var(--voted-border)',
            borderRadius: 'var(--radius-md)',
            padding: '1.5rem',
            textAlign: 'center',
            margin: '1.5rem 0',
          }}>
            <CheckCircle2 size={36} color="#10b981" style={{ margin: '0 auto 0.75rem' }} />
            <h3 style={{ fontSize: '1.2rem', marginBottom: '0.35rem' }}>You Have Voted!</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', marginBottom: '1.25rem' }}>
              Your vote has been counted. Watch the numbers change in real-time as other participants submit their votes.
            </p>
            <Link to={`/poll/${poll.id}/results`} className="btn btn-primary btn-md">
              Watch Live Results
            </Link>
          </div>
        ) : (
          /* Active Voting Form */
          <form onSubmit={handleVote} style={{ marginTop: '1.5rem' }}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', marginBottom: '1.75rem' }}>
              {poll.options.map((opt) => {
                const isSelected = selectedOption === opt.id
                return (
                  <div
                    key={opt.id}
                    onClick={() => setSelectedOption(opt.id)}
                    className={`vote-option-card ${isSelected ? 'selected' : ''}`}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => {
                      if (e.key === ' ' || e.key === 'Enter') {
                        setSelectedOption(opt.id)
                      }
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center' }}>
                      <div className="vote-radio-indicator">
                        {isSelected && <div className="vote-radio-inner" />}
                      </div>
                      <span style={{ fontSize: '1rem', fontWeight: isSelected ? 600 : 400 }}>
                        {opt.text}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>

            <button
              type="submit"
              disabled={submitting || !selectedOption}
              className="btn btn-primary btn-lg"
              style={{ width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.5rem' }}
            >
              <span>{submitting ? 'Casting Vote...' : 'Submit Vote'}</span>
              <ArrowRight size={18} />
            </button>
          </form>
        )}
      </div>

      <ShareModal
        poll={poll}
        isOpen={shareModalOpen}
        onClose={() => setShareModalOpen(false)}
      />
    </div>
  )
}
