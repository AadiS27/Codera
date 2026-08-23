'use client';

import Link from 'next/link';
import { Terminal, ShieldAlert, Cpu, BrainCircuit, Activity, Zap } from 'lucide-react';

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

      <div className="features-section">
        <h2 className="section-title">SYSTEM_CAPABILITIES //</h2>
        
        <div className="features-grid">
          <div className="feature-card cyber-chamfer holo-panel">
            <Zap className="feature-icon" size={32} />
            <h3>Hyper-Isolated Sandbox</h3>
            <p>Code runs in ephemeral Docker containers with strict CPU, memory, and PIDs limits. Zero interference. Zero compromise.</p>
          </div>
          
          <div className="feature-card cyber-chamfer holo-panel">
            <BrainCircuit className="feature-icon" size={32} />
            <h3>AI Complexity Analyzer</h3>
            <p>Every accepted submission is instantly parsed by Gemini 3.5 Flash to automatically determine Time & Space Big-O Complexity.</p>
          </div>
          
          <div className="feature-card cyber-chamfer holo-panel">
            <Activity className="feature-icon" size={32} />
            <h3>Real-Time Telemetry</h3>
            <p>Submissions are tracked live through a distributed Redis queue, giving you instant streaming updates straight to the UI.</p>
          </div>
        </div>
      </div>

      <div className="tech-stack-section">
        <div className="tech-terminal">
          <div className="tech-header">
            <span>mainframe@codera:~</span>
          </div>
          <div className="tech-body">
            <p><span className="keyword">import</span> <span className="string">"architecture"</span></p>
            <br />
            <p className="comment">// Current Stack Deployment:</p>
            <ul className="tech-list">
              <li><span className="accent">Frontend:</span> Next.js (React), Vanilla CSS Cyberpunk Design</li>
              <li><span className="accent">Backend API:</span> Golang, Goroutines, ServeMux (Go 1.22+)</li>
              <li><span className="accent">Job Queue:</span> Redis Streams & Consumer Groups</li>
              <li><span className="accent">Database:</span> PostgreSQL with pgxpool</li>
              <li><span className="accent">AI Engine:</span> Google Generative AI (Gemini Flash)</li>
            </ul>
            <p className="prompt"><span className="blinking-cursor"></span></p>
          </div>
        </div>
      </div>

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .landing-container {
          display: flex;
          flex-direction: column;
          align-items: center;
          min-height: 100vh;
          padding: 2rem;
          transform: skewY(-1deg);
          gap: 6rem;
        }
        
        .hero-section {
          max-width: 900px;
          text-align: center;
          animation: fadeUp 1s steps(5) forwards;
          opacity: 0;
          transform: translateY(20px) skewY(1deg); /* Un-skew the inner content */
          margin-top: 10vh;
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

        .features-section {
          max-width: 1200px;
          width: 100%;
          transform: skewY(1deg); /* Fix skew */
        }

        .section-title {
          font-size: 2rem;
          color: var(--accent-secondary);
          margin-bottom: 3rem;
          border-bottom: 1px solid var(--border-color);
          padding-bottom: 0.5rem;
          display: inline-block;
        }

        .features-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 2rem;
        }

        .feature-card {
          padding: 2rem;
          background: rgba(0, 0, 0, 0.4);
          transition: transform 0.2s, box-shadow 0.2s;
        }

        .feature-card:hover {
          transform: translateY(-5px);
          box-shadow: 0 5px 15px rgba(0, 255, 204, 0.1);
        }

        .feature-icon {
          color: var(--accent-primary);
          margin-bottom: 1rem;
        }

        .feature-card h3 {
          font-size: 1.5rem;
          margin-bottom: 1rem;
          color: var(--fg-default);
        }

        .feature-card p {
          color: var(--fg-muted);
          line-height: 1.6;
        }

        .tech-stack-section {
          width: 100%;
          max-width: 800px;
          transform: skewY(1deg);
          margin-bottom: 4rem;
        }

        .tech-terminal {
          background: rgba(10, 10, 15, 0.9);
          border: 1px solid var(--border-color);
          border-radius: 4px;
          overflow: hidden;
        }

        .tech-header {
          background: var(--bg-surface);
          padding: 0.5rem 1rem;
          border-bottom: 1px solid var(--border-color);
          font-family: var(--font-body);
          font-size: 0.9rem;
          color: var(--fg-muted);
        }

        .tech-body {
          padding: 1.5rem;
          font-family: var(--font-body);
          font-size: 1rem;
          line-height: 1.5;
        }

        .tech-list {
          list-style: none;
          padding: 0;
          margin: 1rem 0 1rem 1.5rem;
        }

        .tech-list li {
          margin-bottom: 0.5rem;
          position: relative;
        }

        .tech-list li::before {
          content: '>';
          position: absolute;
          left: -1.5rem;
          color: var(--accent-secondary);
        }

        .keyword { color: #ff79c6; }
        .string { color: #f1fa8c; }
        .comment { color: #6272a4; }
        .accent { color: var(--accent-primary); font-weight: bold; }
        .prompt::before {
          content: 'Codera:~$ ';
          color: var(--accent-primary);
        }
      `}</style>
    </main>
  );
}
