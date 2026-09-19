import { getOrCreateVoterToken } from './voterId'

const API_BASE = import.meta.env.VITE_API_URL 
  ? `${import.meta.env.VITE_API_URL}/api/v1`
  : '/api/v1'

async function request(endpoint, options = {}) {
  const token = localStorage.getItem('pulsepoll_token')
  const voterToken = getOrCreateVoterToken()

  const headers = {
    'Content-Type': 'application/json',
    'X-Voter-Token': voterToken,
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  })

  const data = await response.json().catch(() => ({}))

  if (!response.ok) {
    const error = new Error(data.error || `HTTP error! Status: ${response.status}`)
    error.status = response.status
    error.data = data
    throw error
  }

  return data
}

export const api = {
  // Auth endpoints
  auth: {
    register: (userData) => request('/auth/register', { method: 'POST', body: JSON.stringify(userData) }),
    login: (credentials) => request('/auth/login', { method: 'POST', body: JSON.stringify(credentials) }),
    me: () => request('/auth/me'),
  },

  // Poll endpoints
  polls: {
    create: (pollData) => request('/polls', { method: 'POST', body: JSON.stringify(pollData) }),
    get: (id) => request(`/polls/${id}`),
    getMyPolls: () => request('/polls/my'),
    close: (id) => request(`/polls/${id}/close`, { method: 'PATCH' }),
    delete: (id) => request(`/polls/${id}`, { method: 'DELETE' }),
    vote: (id, optionId) => request(`/polls/${id}/vote`, {
      method: 'POST',
      body: JSON.stringify({
        option_id: optionId,
        voter_token: getOrCreateVoterToken(),
      }),
    }),
  },
}
