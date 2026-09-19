import React from 'react'
import { Trophy, Check } from 'lucide-react'

const BAR_GRADIENTS = [
  'var(--bar-color-1)',
  'var(--bar-color-2)',
  'var(--bar-color-3)',
  'var(--bar-color-4)',
  'var(--bar-color-5)',
  'var(--bar-color-6)',
]

export function AnimatedBarChart({ options = [], totalVotes = 0, percentages = {}, userVoteOptionId = null }) {
  // Find highest vote count to mark leader
  let maxCount = -1
  if (totalVotes > 0) {
    options.forEach(opt => {
      if (opt.vote_count > maxCount) {
        maxCount = opt.vote_count
      }
    })
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem', width: '100%' }}>
      {options.map((option, index) => {
        const pct = percentages[option.id] !== undefined ? percentages[option.id] : 0
        const isLeading = totalVotes > 0 && option.vote_count === maxCount && maxCount > 0
        const isUserVote = userVoteOptionId === option.id
        const gradient = BAR_GRADIENTS[index % BAR_GRADIENTS.length]

        return (
          <div
            key={option.id}
            className="result-row"
            style={{
              background: isUserVote ? 'rgba(99, 102, 241, 0.07)' : 'transparent',
              padding: isUserVote ? '0.75rem 1rem' : '0.25rem 0',
              borderRadius: isUserVote ? 'var(--radius-md)' : '0',
              border: isUserVote ? '1px solid rgba(99, 102, 241, 0.25)' : 'none',
              transition: 'all var(--transition-normal)',
            }}
          >
            <div className="result-header">
              <div className="result-option-text">
                <span>{option.text}</span>
                {isUserVote && (
                  <span style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '3px',
                    fontSize: '0.75rem',
                    padding: '2px 8px',
                    borderRadius: 'var(--radius-full)',
                    background: 'rgba(99, 102, 241, 0.2)',
                    color: 'var(--primary-light)',
                    fontWeight: 600,
                  }}>
                    <Check size={12} /> Your Vote
                  </span>
                )}
                {isLeading && (
                  <span style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '3px',
                    fontSize: '0.72rem',
                    padding: '2px 7px',
                    borderRadius: 'var(--radius-full)',
                    background: 'rgba(245, 158, 11, 0.15)',
                    color: '#fbbf24',
                    fontWeight: 600,
                  }}>
                    <Trophy size={11} /> Leading
                  </span>
                )}
              </div>

              <div className="result-stats">
                <span className="result-percentage">{pct}%</span>
                <span className="result-votes-count">
                  ({option.vote_count} {option.vote_count === 1 ? 'vote' : 'votes'})
                </span>
              </div>
            </div>

            {/* Visual Bar Track & Fill */}
            <div className="result-bar-track">
              <div
                className="result-bar-fill"
                style={{
                  width: `${pct}%`,
                  background: gradient,
                }}
              />
            </div>
          </div>
        )
      })}
    </div>
  )
}
