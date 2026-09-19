import { useEffect, useRef, useState, useCallback } from 'react'

export function useWebSocket(pollId, onMessage) {
  const [status, setStatus] = useState('connecting') // 'connecting' | 'connected' | 'disconnected' | 'error'
  const wsRef = useRef(null)
  const reconnectTimeoutRef = useRef(null)
  const onMessageRef = useRef(onMessage)

  // Keep latest message callback without reconnecting
  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  const connect = useCallback(() => {
    if (!pollId) return

    // Determine WS protocol and URL
    const isSecure = window.location.protocol === 'https:'
    const defaultWsHost = window.location.host
    let wsUrl = ''

    if (import.meta.env.VITE_WS_URL) {
      wsUrl = `${import.meta.env.VITE_WS_URL}/api/v1/polls/${pollId}/ws`
    } else if (import.meta.env.VITE_API_URL) {
      const url = new URL(import.meta.env.VITE_API_URL)
      const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
      wsUrl = `${protocol}//${url.host}/api/v1/polls/${pollId}/ws`
    } else {
      // Local proxy or relative path
      const protocol = isSecure ? 'wss:' : 'ws:'
      wsUrl = `${protocol}//${defaultWsHost}/api/v1/polls/${pollId}/ws`
    }

    try {
      setStatus('connecting')
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setStatus('connected')
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current)
          reconnectTimeoutRef.current = null
        }
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (onMessageRef.current) {
            onMessageRef.current(data)
          }
        } catch (err) {
          console.error('[WS] Failed to parse message:', err)
        }
      }

      ws.onerror = (err) => {
        console.warn('[WS] Connection error:', err)
        setStatus('error')
      }

      ws.onclose = (event) => {
        setStatus('disconnected')
        wsRef.current = null
        // Attempt reconnect after 2.5 seconds if poll is still active
        if (!event.wasClean) {
          reconnectTimeoutRef.current = setTimeout(() => {
            connect()
          }, 2500)
        }
      }
    } catch (err) {
      console.error('[WS] Setup exception:', err)
      setStatus('error')
    }
  }, [pollId])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmounted')
      }
    }
  }, [connect])

  return { status, reconnect: connect }
}
