import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../services/api'
import { useToast } from '../context/ToastContext'
import { Plus, Trash2, Clock, Calendar, ArrowRight, Sparkles } from 'lucide-react'

export function CreatePollPage() {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [options, setOptions] = useState(['', '', ''])
  const [expiryOption, setExpiryOption] = useState('none') // 'none' | '1h' | '24h' | '7d' | 'custom'
  const [customDateTime, setCustomDateTime] = useState('')
  const [loading, setLoading] = useState(false)

  const { success, error: toastError } = useToast()
  const navigate = useNavigate()

  // Calculate min datetime for picker (now + 2 minutes formatted for datetime-local)
  const getMinDateTimeLocal = () => {
    const now = new Date(Date.now() + 2 * 60 * 1000)
    const year = now.getFullYear()
    const month = String(now.getMonth() + 1).padStart(2, '0')
    const day = String(now.getDate()).padStart(2, '0')
    const hours = String(now.getHours()).padStart(2, '0')
    const minutes = String(now.getMinutes()).padStart(2, '0')
    return `${year}-${month}-${day}T${hours}:${minutes}`
  }

  const handleAddOption = () => {
    if (options.length >= 10) {
      toastError('Maximum 10 options allowed')
      return
    }
    setOptions(prev => [...prev, ''])
  }

  const handleRemoveOption = (index) => {
    if (options.length <= 2) {
      toastError('A poll must have at least 2 options')
      return
    }
    setOptions(prev => prev.filter((_, i) => i !== index))
  }

  const handleOptionChange = (index, value) => {
    setOptions(prev => {
      const next = [...prev]
      next[index] = value
      return next
    })
  }

  const fillExamplePoll = () => {
    setTitle('Which backend stack handles real-time polling best?')
    setDescription('Evaluating latency, concurrency, and developer ergonomics.')
    setOptions([
      'Go (Gin) + Redis Pub/Sub + WebSockets',
      'Node.js (Fastify) + Socket.io',
      'Elixir / Phoenix LiveView Channels',
      'Rust (Axum) + Redis Streams'
    ])
  }

  const handleSubmit = async (e) => {
    e.preventDefault()

    const trimmedTitle = title.trim()
    if (trimmedTitle.length < 3) {
      toastError('Question must be at least 3 characters long')
      return
    }

    const filteredOptions = options.map(o => o.trim()).filter(o => o.length > 0)
    if (filteredOptions.length < 2) {
      toastError('Please enter at least 2 non-empty options')
      return
    }

    // Check duplicates
    const uniqueOptions = new Set(filteredOptions.map(o => o.toLowerCase()))
    if (uniqueOptions.size !== filteredOptions.length) {
      toastError('All options must be unique')
      return
    }

    // Calculate expiry timestamp in UTC ISO format
    let expiryAt = null
    if (expiryOption === '1h') {
      expiryAt = new Date(Date.now() + 60 * 60 * 1000).toISOString()
    } else if (expiryOption === '24h') {
      expiryAt = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()
    } else if (expiryOption === '7d') {
      expiryAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString()
    } else if (expiryOption === 'custom') {
      if (!customDateTime) {
        toastError('Please select a custom close date and time')
        return
      }
      const customDate = new Date(customDateTime)
      if (isNaN(customDate.getTime()) || customDate.getTime() <= Date.now()) {
        toastError('Custom close date and time must be in the future')
        return
      }
      expiryAt = customDate.toISOString()
    }

    setLoading(true)
    try {
      const payload = {
        title: trimmedTitle,
        description: description.trim(),
        options: filteredOptions,
        ...(expiryAt ? { expiry_at: expiryAt } : {}),
      }

      const poll = await api.polls.create(payload)
      success('Live poll created successfully!')
      navigate(`/poll/${poll.id}/results`, { state: { justCreated: true } })
    } catch (err) {
      toastError(err.message || 'Failed to create poll')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: '680px', margin: '0 auto' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.75rem' }}>
        <div>
          <h1 style={{ fontSize: '2rem', marginBottom: '0.35rem' }}>Create Live Poll</h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem' }}>
            Set up your question, choices, and launch live real-time voting.
          </p>
        </div>

        <button
          type="button"
          onClick={fillExamplePoll}
          className="btn btn-secondary btn-sm"
          style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}
        >
          <Sparkles size={14} color="var(--accent-cyan)" />
          <span>Fill Example</span>
        </button>
      </div>

      <div className="glass-card" style={{ padding: '2rem 1.75rem' }}>
        <form onSubmit={handleSubmit}>
          {/* Poll Question */}
          <div className="form-group">
            <label className="form-label" htmlFor="title">
              <span>Poll Question <strong style={{ color: 'var(--accent-rose)' }}>*</strong></span>
              <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>{title.length}/300</span>
            </label>
            <input
              id="title"
              type="text"
              required
              maxLength={300}
              className="form-input"
              placeholder="e.g., What feature should we ship in the next sprint?"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              style={{ fontSize: '1.05rem', fontWeight: 500 }}
            />
          </div>

          {/* Optional Description */}
          <div className="form-group">
            <label className="form-label" htmlFor="description">
              <span>Context or Description (Optional)</span>
            </label>
            <textarea
              id="description"
              className="form-textarea"
              placeholder="Add extra details or instructions for voters..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
            />
          </div>

          {/* Options List */}
          <div className="form-group" style={{ marginTop: '1.5rem' }}>
            <label className="form-label">
              <span>Poll Options <strong style={{ color: 'var(--accent-rose)' }}>*</strong></span>
              <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{options.length}/10 options</span>
            </label>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.65rem' }}>
              {options.map((opt, idx) => (
                <div key={idx} style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                  <span style={{
                    width: '26px',
                    height: '26px',
                    borderRadius: '50%',
                    background: 'var(--bg-tertiary)',
                    color: 'var(--text-secondary)',
                    border: '1px solid var(--border-subtle)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: '0.8rem',
                    fontWeight: 600,
                    fontFamily: 'var(--font-mono)',
                    flexShrink: 0,
                  }}>
                    {idx + 1}
                  </span>

                  <input
                    type="text"
                    required
                    maxLength={150}
                    className="form-input"
                    placeholder={`Option ${idx + 1}`}
                    value={opt}
                    onChange={(e) => handleOptionChange(idx, e.target.value)}
                  />

                  {options.length > 2 && (
                    <button
                      type="button"
                      onClick={() => handleRemoveOption(idx)}
                      className="btn-ghost"
                      style={{ padding: '8px', color: 'var(--accent-rose)', border: 'none', cursor: 'pointer', flexShrink: 0 }}
                      title="Remove option"
                    >
                      <Trash2 size={16} />
                    </button>
                  )}
                </div>
              ))}
            </div>

            {options.length < 10 && (
              <button
                type="button"
                onClick={handleAddOption}
                className="btn btn-secondary btn-sm"
                style={{ marginTop: '0.85rem', display: 'inline-flex', alignItems: 'center', gap: '0.4rem' }}
              >
                <Plus size={15} />
                <span>Add Choice</span>
              </button>
            )}
          </div>

          {/* Poll Auto-Close Duration / Custom Date Configuration */}
          <div className="form-group" style={{ marginTop: '1.75rem', borderTop: '1px solid var(--border-subtle)', paddingTop: '1.25rem' }}>
            <label className="form-label" htmlFor="expiry">
              <span style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                <Clock size={16} />
                <span>Auto-Close Poll Schedule</span>
              </span>
            </label>
            <select
              id="expiry"
              className="form-select"
              value={expiryOption}
              onChange={(e) => setExpiryOption(e.target.value)}
            >
              <option value="none">Keep Open Indefinitely (Manual Close)</option>
              <option value="1h">Close Automatically in 1 Hour</option>
              <option value="24h">Close Automatically in 24 Hours</option>
              <option value="7d">Close Automatically in 7 Days</option>
              <option value="custom">📅 Pick Custom Date & Time...</option>
            </select>

            {/* Custom Datetime Picker */}
            {expiryOption === 'custom' && (
              <div style={{ marginTop: '0.85rem', animation: 'fadeIn 0.2s ease-out' }}>
                <label className="form-label" htmlFor="custom-datetime">
                  <span style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', color: 'var(--primary-light)' }}>
                    <Calendar size={15} />
                    <span>Exact Date & Time (in your local timezone):</span>
                  </span>
                </label>
                <input
                  id="custom-datetime"
                  type="datetime-local"
                  required
                  min={getMinDateTimeLocal()}
                  className="form-input"
                  value={customDateTime}
                  onChange={(e) => setCustomDateTime(e.target.value)}
                  style={{ fontFamily: 'var(--font-mono)', fontSize: '0.95rem' }}
                />
                <span style={{ display: 'block', fontSize: '0.78rem', color: 'var(--text-muted)', marginTop: '0.35rem' }}>
                  The poll will automatically stop accepting votes when this timestamp arrives.
                </span>
              </div>
            )}
          </div>

          {/* Submit Action */}
          <div style={{ marginTop: '2rem', display: 'flex', justifyContent: 'flex-end', gap: '0.75rem' }}>
            <button
              type="submit"
              disabled={loading}
              className="btn btn-primary btn-lg"
              style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}
            >
              <span>{loading ? 'Creating Poll...' : 'Launch Live Poll'}</span>
              <ArrowRight size={18} />
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
