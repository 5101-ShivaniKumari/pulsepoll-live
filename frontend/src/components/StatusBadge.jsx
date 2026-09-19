import React from 'react'

export function StatusBadge({ isClosed = false, totalVotes = 0 }) {
  if (isClosed) {
    return (
      <span className="badge badge-closed">
        <span>●</span> Closed
      </span>
    )
  }

  return (
    <span className="badge badge-live">
      <span className="pulse-dot" />
      <span>Live Updates</span>
    </span>
  )
}
