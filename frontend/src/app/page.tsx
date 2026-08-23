'use client';

import Link from 'next/link';

export default function Home() {
  return (
    <main className="landing-container">
      <div className="hero-section">
        <h1 className="hero-title cyber-glitch-text" data-text="EXECUTE. ITERATE. DOMINATE.">
          EXECUTE. ITERATE. DOMINATE.
        </h1>
        <p className="hero-subtitle">
          <span className="terminal-prefix">&gt; </span>
          Welcome to the next generation of online judging. Experience frictionless, hyper-isolated code execution powered by a world-class engine. 
          Step into the arena.<span className="blinking-cursor"></span>
        </p>
        
        <div className="hero-actions">
          <Link href="/problems" className="cyber-btn cyber-btn-glitch cyber-chamfer-sm">
            Enter Workspace
          </Link>
          <Link href="/admin" className="cyber-btn cyber-btn-outline cyber-chamfer-sm">
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
          /* Skew the entire section slightly for the brutalist asymmetrical look */
          transform: skewY(-1deg);
        }
        
        .hero-section {
          max-width: 900px;
          text-align: center;
          animation: fadeUp 1s steps(5) forwards;
          opacity: 0;
          transform: translateY(20px) skewY(1deg); /* Un-skew the inner content */
        }

        .hero-title {
          font-size: clamp(3rem, 7vw, 5.5rem);
          line-height: 1.1;
          margin-bottom: 2rem;
          color: var(--accent-primary);
        }

        .hero-subtitle {
          font-size: 1.25rem;
          color: var(--fg-muted);
          margin-bottom: 3.5rem;
          max-width: 650px;
          margin-inline: auto;
          line-height: 1.8;
          font-family: var(--font-body);
          text-align: left;
          background: rgba(0, 0, 0, 0.4);
          padding: 1.5rem;
          border-left: 2px solid var(--accent-primary);
        }

        .terminal-prefix {
          color: var(--accent-primary);
          font-weight: bold;
        }

        .hero-actions {
          display: flex;
          gap: 2rem;
          justify-content: center;
          transform: skewX(-5deg); /* Cyberpunk skewed buttons */
        }

        .hero-actions > * {
          transform: skewX(5deg); /* un-skew text inside */
        }

        @keyframes fadeUp {
          to {
            opacity: 1;
            transform: translateY(0) skewY(1deg);
          }
        }
      `}</style>
    </main>
  );
}
