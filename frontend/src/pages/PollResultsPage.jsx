import React, { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, Link, useLocation } from 'react-router-dom'
import { api } from '../services/api'
import { useAuth } from '../context/AuthContext'
import { useToast } from '../context/ToastContext'
import { useWebSocket } from '../hooks/useWebSocket'
import { AnimatedBarChart } from '../components/AnimatedBarChart'
import { StatusBadge } from '../components/StatusBadge'
import { ShareModal } from '../components/ShareModal'
import { ConfirmationModal } from '../components/ConfirmationModal'
import { Share2, Users, Vote, Lock, AlertCircle, Wifi, WifiOff, Eye } from 'lucide-react'

export function PollResultsPage() {
  const { id } = useParams()
  const location = useLocation()
  const { user } = useAuth()
  const { success, error: toastError, info } = useToast()

  const [poll, setPoll] = useState(null)
  const [loading, setLoading] = useState(true)
  const [shareModalOpen, setShareModalOpen] = useState(false)
  const [closeModalOpen, setCloseModalOpen] = useState(false)
  const [closing, setClosing] = useState(false)

  // Real-time Live Viewer Presence & Vote Burst States
  const [viewerCount, setViewerCount] = useState(1)
  const [recentlyVotedOptionId, setRecentlyVotedOptionId] = useState(null)
  const [voteBurstActive, setVoteBurstActive] = useState(false)
  const burstTimeoutRef = useRef(null)

  const userVotedOptionId = location.state?.votedOptionId || null

  // Fetch initial poll data from HTTP endpoint
  const loadPoll = useCallback(async () => {
    try {
      setLoading(true)
      const data = await api.polls.get(id)
      setPoll(data)
    } catch (err) {
      toastError(err.message || 'Poll not found')
    } finally {
      setLoading(false)
    }
  }, [id, toastError])

  useEffect(() => {
    loadPoll()
  }, [loadPoll])

  // Real-time WebSocket live updates handler
  const handleWebSocketMessage = useCallback((msg) => {
    if (!msg) return

    // 1. Live Viewer Presence Update
    if (msg.type === 'viewer_update') {
      if (msg.viewer_count !== undefined) {
        setViewerCount(Math.max(1, msg.viewer_count))
      }
      return
    }

    // 2. Vote Cast Event (with subtle burst animation)
    if (msg.type === 'vote_cast') {
      if (msg.viewer_count !== undefined && msg.viewer_count > 0) {
        setViewerCount(msg.viewer_count)
      }

      // Trigger subtle pulse & +1 indicator on the voted option
      if (msg.recent_voted_option_id) {
        setRecentlyVotedOptionId(msg.recent_voted_option_id)
        setVoteBurstActive(true)

        if (burstTimeoutRef.current) {
          clearTimeout(burstTimeoutRef.current)
        }
        burstTimeoutRef.current = setTimeout(() => {
          setRecentlyVotedOptionId(null)
          setVoteBurstActive(false)
        }, 1200)
      }
    }

    // 3. Update Poll Data State
    if (msg.type === 'vote_cast' || msg.type === 'poll_state') {
      setPoll(prev => {
        if (!prev) return prev

        const updatedOptions = prev.options.map(opt => {
          const newCount = msg.option_votes && msg.option_votes[opt.id] !== undefined
            ? msg.option_votes[opt.id]
            : opt.vote_count
          return { ...opt, vote_count: newCount }
        })

        return {
          ...prev,
          total_votes: msg.total_votes !== undefined ? msg.total_votes : prev.total_votes,
          percentages: msg.percentages || prev.percentages,
          is_closed: msg.is_closed !== undefined ? msg.is_closed : prev.is_closed,
          options: updatedOptions,
        }
      })
    } else if (msg.type === 'poll_closed') {
      setPoll(prev => prev ? { ...prev, is_closed: true } : prev)
      info('This poll has just been closed by the creator.')
    }
  }, [info])

  const { status: wsStatus } = useWebSocket(id, handleWebSocketMessage)

  // Clean up animation timeout on unmount
  useEffect(() => {
    return () => {
      if (burstTimeoutRef.current) {
        clearTimeout(burstTimeoutRef.current)
      }
    }
  }, [])

  // Handle Close Poll action by creator
  const handleClosePoll = async () => {
    setClosing(true)
    try {
      await api.polls.close(id)
      setPoll(prev => prev ? { ...prev, is_closed: true } : prev)
      setCloseModalOpen(false)
      success('Poll closed for further voting.')
    } catch (err) {
      toastError(err.message || 'Failed to close poll')
    } finally {
      setClosing(false)
    }
  }

  // Open share modal automatically if redirected right after creation
  useEffect(() => {
    if (location.state?.justCreated) {
      setShareModalOpen(true)
    }
  }, [location.state])

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '5rem 0' }}>
        <div className="pulse-dot" style={{ width: '16px', height: '16px', margin: '0 auto 1rem' }} />
        <p style={{ color: 'var(--text-secondary)' }}>Connecting to live poll stream...</p>
      </div>
    )
  }

  if (!poll) {
    return (
      <div style={{ textAlign: 'center', padding: '4rem 1rem' }}>
        <AlertCircle size={48} color="var(--accent-rose)" style={{ margin: '0 auto 1rem' }} />
        <h2>Poll Not Found</h2>
        <p style={{ color: 'var(--text-secondary)', marginTop: '0.5rem', marginBottom: '1.5rem' }}>
          This poll may have been deleted or the link is invalid.
        </p>
        <Link to="/" className="btn btn-secondary">
          Return Home
        </Link>
      </div>
    )
  }

  const isCreator = user && poll.creator_id && (user.id === poll.creator_id || user.id === poll.creator_id?.toString())

  return (
    <div style={{ maxWidth: '720px', margin: '1rem auto 0' }}>
      <div className={`glass-card ${voteBurstActive ? 'vote-burst-active' : ''}`} style={{ padding: '2.25rem 2rem' }}>
        {/* Header Badges & Actions */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: '0.75rem',
          marginBottom: '1.5rem',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.65rem', flexWrap: 'wrap' }}>
            <StatusBadge isClosed={poll.is_closed} totalVotes={poll.total_votes} />

            {/* Live Viewer Presence Badge */}
            <span className="badge badge-viewers" title="Active live viewers watching this poll">
              <Eye size={13} style={{ marginRight: '2px' }} />
              <span className="pulse-dot-indigo" />
              <span>{viewerCount} {viewerCount === 1 ? 'watching' : 'watching'}</span>
            </span>

            {/* WebSocket connection indicator */}
            <span style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '4px',
              fontSize: '0.72rem',
              color: wsStatus === 'connected' ? 'var(--accent-emerald)' : 'var(--accent-amber)',
              fontFamily: 'var(--font-mono)',
            }}>
              {wsStatus === 'connected' ? <Wifi size={13} /> : <WifiOff size={13} />}
              <span>{wsStatus === 'connected' ? 'Connected' : 'Reconnecting...'}</span>
            </span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            {!poll.is_closed && (
              <Link
                to={`/poll/${poll.id}`}
                className="btn btn-secondary btn-sm"
                style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}
              >
                <Vote size={15} />
                <span>Vote Screen</span>
              </Link>
            )}

            <button
              type="button"
              onClick={() => setShareModalOpen(true)}
              className="btn btn-primary btn-sm"
              style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}
            >
              <Share2 size={15} />
              <span>Share & QR</span>
            </button>
          </div>
        </div>

        {/* Question Header */}
        <h1 style={{ fontSize: '1.85rem', marginBottom: '0.5rem', lineHeight: 1.3 }}>
          {poll.title}
        </h1>

        {poll.description && (
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem', marginBottom: '1.5rem' }}>
            {poll.description}
          </p>
        )}

        {/* Total Votes Banner with Live Running Count */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0.85rem 1.25rem',
          background: 'var(--bg-secondary)',
          border: '1px solid var(--border-subtle)',
          borderRadius: 'var(--radius-md)',
          marginBottom: '2rem',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--text-secondary)' }}>
            <Users size={18} color="var(--primary-light)" />
            <span style={{ fontSize: '0.9rem', fontWeight: 500 }}>Total Votes Cast:</span>
          </div>
          <span style={{
            fontSize: '1.25rem',
            fontWeight: 800,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-primary)',
          }}>
            {poll.total_votes.toLocaleString()}
          </span>
        </div>

        {/* Live Visual Animated Chart with +1 highlight burst */}
        <AnimatedBarChart
          options={poll.options}
          totalVotes={poll.total_votes}
          percentages={poll.percentages}
          userVoteOptionId={userVotedOptionId || (poll.has_voted ? poll.voted_for : null)}
          recentlyVotedOptionId={recentlyVotedOptionId}
        />

        {/* Creator Control Strip */}
        {isCreator && (
          <div style={{
            marginTop: '2.5rem',
            paddingTop: '1.5rem',
            borderTop: '1px solid var(--border-subtle)',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: '1rem',
          }}>
            <span style={{ fontSize: '0.82rem', color: 'var(--text-muted)' }}>
              Creator Controls
            </span>

            {!poll.is_closed ? (
              <button
                type="button"
                onClick={() => setCloseModalOpen(true)}
                className="btn btn-secondary btn-sm"
                style={{ color: 'var(--accent-amber)', borderColor: 'rgba(245, 158, 11, 0.3)' }}
              >
                <Lock size={14} />
                <span>Close Poll to New Votes</span>
              </button>
            ) : (
              <span style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                ● This poll is closed
              </span>
            )}
          </div>
        )}
      </div>

      <ShareModal
        poll={poll}
        isOpen={shareModalOpen}
        onClose={() => setShareModalOpen(false)}
      />

      <ConfirmationModal
        isOpen={closeModalOpen}
        title="Close this Live Poll?"
        message="Closing this poll will immediately stop any further votes from being cast across all connected devices and browsers."
        confirmText="Yes, Close Poll"
        confirmVariant="danger"
        loading={closing}
        onConfirm={handleClosePoll}
        onCancel={() => setCloseModalOpen(false)}
      />
    </div>
  )
}
