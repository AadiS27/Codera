'use client';

import Link from 'next/link';

export default function Home() {
  return (
    <main className="landing-container">
      <div className="hero-section">
        <h1 className="hero-title">
          Execute. <span className="text-cyan">Iterate.</span> Dominate.
        </h1>
        <p className="hero-subtitle">
          Welcome to the next generation of online judging. Experience frictionless, hyper-isolated code execution powered by a world-class engine. 
          Step into the arena.
        </p>
        
        <div className="hero-actions">
          <Link href="/problems" className="btn-primary">
            Enter Workspace
          </Link>
          <Link href="/admin" className="btn-secondary">
            Author Problem
          </Link>
        </div>
      </div>

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .landing-container {
          display: flex;
          align-items: center;
          justify-content: center;
          min-height: 100vh;
          padding: 2rem;
          background: radial-gradient(circle at center, rgba(0,240,255,0.05) 0%, transparent 70%);
        }
        
        .hero-section {
          max-width: 800px;
          text-align: center;
          animation: fadeUp 1s cubic-bezier(0.16, 1, 0.3, 1) forwards;
          opacity: 0;
          transform: translateY(20px);
        }

        .hero-title {
          font-size: clamp(3rem, 8vw, 5rem);
          line-height: 1.1;
          letter-spacing: -0.04em;
          margin-bottom: 1.5rem;
          font-weight: 800;
        }

        .text-cyan {
          color: var(--accent-cyan);
          text-shadow: 0 0 30px var(--accent-cyan-glow);
        }

        .hero-subtitle {
          font-size: 1.25rem;
          color: var(--text-secondary);
          margin-bottom: 3rem;
          max-width: 600px;
          margin-inline: auto;
          line-height: 1.8;
        }

        .hero-actions {
          display: flex;
          gap: 1.5rem;
          justify-content: center;
        }

        @keyframes fadeUp {
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </main>
  );
}
