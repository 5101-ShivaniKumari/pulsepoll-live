import React, { useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { Copy, Check, X, Share2, MessageCircle, Send } from 'lucide-react'
import { useToast } from '../context/ToastContext'
import { getShareablePollUrl } from '../utils/url'

export function ShareModal({ poll, isOpen, onClose }) {
  const [copied, setCopied] = useState(false)
  const { success } = useToast()

  if (!isOpen || !poll) return null

  const shareUrl = getShareablePollUrl(poll.id)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl)
      setCopied(true)
      success('Link copied to clipboard!')
      setTimeout(() => setCopied(false), 2500)
    } catch (err) {
      console.error('Clipboard copy failed:', err)
    }
  }

  const twitterUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(`Vote in this live poll: "${poll.title}"`)}&url=${encodeURIComponent(shareUrl)}`
  const whatsappUrl = `https://api.whatsapp.com/send?text=${encodeURIComponent(`Vote in this live poll: "${poll.title}"\n${shareUrl}`)}`

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            <Share2 size={20} color="var(--primary-light)" />
            <h3 style={{ margin: 0, fontSize: '1.25rem' }}>Share Live Poll</h3>
          </div>
          <button
            onClick={onClose}
            className="btn-ghost"
            style={{ padding: '4px', border: 'none', cursor: 'pointer', color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', borderRadius: 'var(--radius-sm)' }}
          >
            <X size={18} />
          </button>
        </div>

        <p style={{ fontSize: '0.9rem', color: 'var(--text-secondary)', marginBottom: '1.25rem' }}>
          Share this link with your audience. Votes will roll in live instantly on the results screen.
        </p>

        {/* QR Code — always white bg so QR is scannable in both themes */}
        <div className="qr-code-wrapper">
          <QRCodeSVG
            value={shareUrl}
            size={180}
            level="H"
            includeMargin={true}
          />
          <span className="qr-code-caption">Scan to Vote Immediately</span>
        </div>

        {/* Copy Link Input Box */}
        <div style={{
          display: 'flex',
          gap: '0.5rem',
          marginBottom: '1.25rem',
        }}>
          <input
            type="text"
            readOnly
            value={shareUrl}
            className="form-input"
            style={{ fontFamily: 'var(--font-mono)', fontSize: '0.85rem' }}
          />
          <button
            onClick={handleCopy}
            className={`btn ${copied ? 'btn-secondary' : 'btn-primary'}`}
            style={{ flexShrink: 0 }}
          >
            {copied ? <Check size={16} /> : <Copy size={16} />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>

        {/* Quick Social Shares */}
        <div style={{ display: 'flex', gap: '0.65rem' }}>
          <a
            href={twitterUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-secondary"
            style={{ flex: 1, fontSize: '0.85rem' }}
          >
            <Send size={15} />
            <span>X (Twitter)</span>
          </a>
          <a
            href={whatsappUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-secondary"
            style={{ flex: 1, fontSize: '0.85rem' }}
          >
            <MessageCircle size={15} />
            <span>WhatsApp</span>
          </a>
        </div>
      </div>
    </div>
  )
}
