// Voter identification engine: creates a persistent device-bound anonymous token
const VOTER_ID_KEY = 'pulsepoll_voter_token'

export function getOrCreateVoterToken() {
  let token = localStorage.getItem(VOTER_ID_KEY)
  if (!token || token.length < 16) {
    // Generate a secure crypto-random identifier
    const randomArray = new Uint8Array(20)
    window.crypto.getRandomValues(randomArray)
    const randomHex = Array.from(randomArray)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')
    token = `vtr_${Date.now().toString(36)}_${randomHex}`
    localStorage.setItem(VOTER_ID_KEY, token)
  }
  return token
}
