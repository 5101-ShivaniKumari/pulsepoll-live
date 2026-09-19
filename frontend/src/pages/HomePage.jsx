import React from 'react'
import { Link } from 'react-router-dom'
import { Activity, Zap, ShieldCheck, Radio, ArrowRight, BarChart3, QrCode, Sparkles } from 'lucide-react'
import { useAuth } from '../context/AuthContext'

export function HomePage() {
  const { isAuthenticated } = useAuth()

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '4rem', paddingBottom: '3rem' }}>
      {/* Hero Section */}
      <section style={{
        textAlign: 'center',
        padding: '3rem 0 1.5rem',
        maxWidth: '850px',
        margin: '0 auto',
      }}>
        <div style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '0.5rem',
          padding: '0.35rem 1rem',
          borderRadius: 'var(--radius-full)',
          background: 'rgba(99, 102, 241, 0.12)',
          border: '1px solid rgba(99, 102, 241, 0.3)',
          color: 'var(--primary-light)',
          fontSize: '0.85rem',
          fontWeight: 600,
          marginBottom: '1.5rem',
        }}>
          <Sparkles size={15} />
          <span>Real-Time Polling Engine Driven by Go, Redis &amp; WebSockets</span>
        </div>

        <h1 style={{
          fontSize: 'clamp(2.5rem, 5.5vw, 4rem)',
          fontWeight: 800,
          lineHeight: 1.15,
          letterSpacing: '-0.03em',
          marginBottom: '1.5rem',
        }}>
          Create live polls.<br />
          Watch votes roll in <span style={{
            background: 'linear-gradient(135deg, #6366f1 0%, #06b6d4 100%)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
          }}>instantly.</span>
        </h1>

        <p style={{
          fontSize: 'clamp(1.05rem, 2vw, 1.25rem)',
          color: 'var(--text-secondary)',
          lineHeight: 1.6,
          maxWidth: '680px',
          margin: '0 auto 2.5rem',
        }}>
          No page reloads. No polling delays. Every vote is atomically counted in Redis, durable in MongoDB, and broadcast across all connected devices within milliseconds.
        </p>

        <div style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          gap: '1rem',
          flexWrap: 'wrap',
        }}>
          <Link
            to={isAuthenticated ? "/create" : "/register"}
            className="btn btn-primary btn-lg"
            style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}
          >
            <span>Start a Live Poll</span>
            <ArrowRight size={18} />
          </Link>

          <Link
            to={isAuthenticated ? "/dashboard" : "/login"}
            className="btn btn-secondary btn-lg"
          >
            {isAuthenticated ? "My Dashboard" : "Sign In"}
          </Link>
        </div>
      </section>

      {/* Feature Grid */}
      <section style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
        gap: '1.5rem',
      }}>
        <div className="glass-card" style={{ padding: '1.75rem' }}>
          <div style={{
            width: '44px',
            height: '44px',
            borderRadius: '12px',
            background: 'rgba(99, 102, 241, 0.12)',
            border: '1px solid rgba(99, 102, 241, 0.25)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '1.25rem',
          }}>
            <Zap size={22} color="var(--primary-light)" />
          </div>
          <h3 style={{ fontSize: '1.2rem', marginBottom: '0.6rem' }}>Sub-50ms Redis Live Pub/Sub</h3>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.92rem', lineHeight: 1.6 }}>
            Vote counts are incremented atomically with Redis <code style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent-cyan)' }}>HINCRBY</code> and pushed across multi-room WebSockets instantaneously.
          </p>
        </div>

        <div className="glass-card" style={{ padding: '1.75rem' }}>
          <div style={{
            width: '44px',
            height: '44px',
            borderRadius: '12px',
            background: 'rgba(16, 185, 129, 0.12)',
            border: '1px solid rgba(16, 185, 129, 0.25)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '1.25rem',
          }}>
            <ShieldCheck size={22} color="var(--accent-emerald)" />
          </div>
          <h3 style={{ fontSize: '1.2rem', marginBottom: '0.6rem' }}>Multi-Tier Deduplication</h3>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.92rem', lineHeight: 1.6 }}>
            Guaranteed single-vote fairness via Redis atomic <code style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent-emerald)' }}>SADD</code> checks, device fingerprints, and MongoDB compound unique indexes.
          </p>
        </div>

        <div className="glass-card" style={{ padding: '1.75rem' }}>
          <div style={{
            width: '44px',
            height: '44px',
            borderRadius: '12px',
            background: 'rgba(6, 182, 212, 0.12)',
            border: '1px solid rgba(6, 182, 212, 0.25)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginBottom: '1.25rem',
          }}>
            <QrCode size={22} color="var(--accent-cyan)" />
          </div>
          <h3 style={{ fontSize: '1.2rem', marginBottom: '0.6rem' }}>Zero-Friction Sharing &amp; QR</h3>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.92rem', lineHeight: 1.6 }}>
            Voters don't need accounts. Distribute via direct link or display the instant QR code during live presentations, webinars, or classrooms.
          </p>
        </div>
      </section>

      {/* Interactive Visual Teaser */}
      <section className="glass-card" style={{ padding: '2.5rem 2rem', position: 'relative', overflow: 'hidden' }}>
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          textAlign: 'center',
          maxWidth: '650px',
          margin: '0 auto',
        }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '0.5rem',
            marginBottom: '1rem',
          }}>
            <span className="badge badge-live">
              <span className="pulse-dot" /> LIVE PREVIEW
            </span>
          </div>
          <h2 style={{ fontSize: '1.8rem', marginBottom: '0.75rem' }}>Built for high-concurrency production</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem', marginBottom: '1.75rem' }}>
            Open multiple tabs, click vote concurrently, and witness zero-lag synchronized real-time state fanout.
          </p>
          <Link to="/create" className="btn btn-primary btn-md">
            Create Your First Live Poll
          </Link>
        </div>
      </section>
    </div>
  )
}
