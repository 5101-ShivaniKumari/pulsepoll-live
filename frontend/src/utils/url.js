/**
 * Centralized URL builder utility for PulsePoll.
 * Guarantees that QR codes, copy-link buttons, social shares, and display elements
 * all generate identical, valid, and fully-qualified public share URLs.
 */

export function getAppBaseUrl() {
  // 1. Explicit public app URL override from environment (e.g. https://pulsepoll.vercel.app)
  if (import.meta.env.VITE_PUBLIC_APP_URL) {
    return import.meta.env.VITE_PUBLIC_APP_URL.replace(/\/+$/, '')
  }

  // 2. Dynamic runtime host detection (automatically handles production domain, LAN IP, or localhost)
  if (typeof window !== 'undefined' && window.location && window.location.origin) {
    return window.location.origin
  }

  return ''
}

/**
 * Returns the fully qualified public URL for voting on a poll
 * @param {string} pollId - The ID of the poll
 * @returns {string} Fully qualified URL (e.g., https://your-app.vercel.app/poll/abc12345)
 */
export function getShareablePollUrl(pollId) {
  if (!pollId) return ''
  const baseUrl = getAppBaseUrl()
  return `${baseUrl}/poll/${pollId}`
}

/**
 * Returns the fully qualified public URL for viewing poll results
 * @param {string} pollId - The ID of the poll
 * @returns {string} Fully qualified URL (e.g., https://your-app.vercel.app/poll/abc12345/results)
 */
export function getShareablePollResultsUrl(pollId) {
  if (!pollId) return ''
  const baseUrl = getAppBaseUrl()
  return `${baseUrl}/poll/${pollId}/results`
}
