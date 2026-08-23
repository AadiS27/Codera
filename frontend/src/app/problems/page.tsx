'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { Terminal, ShieldAlert, Cpu } from 'lucide-react';

export default function Problems() {
  const [problems, setProblems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('http://localhost:8080/problems')
      .then(res => res.json())
      .then(data => {
        setProblems(data);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, []);

  return (
    <div className="explorer-container">
      <div className="dashboard-header">
        <h1 className="cyber-glitch-text" data-text="SYS.ARCHIVE // CHALLENGES">
          SYS.ARCHIVE // CHALLENGES
        </h1>
        <p className="terminal-subtitle">
          <span className="prefix">&gt; </span> Accessing mainframe problem sets... <span className="blinking-cursor"></span>
        </p>
      </div>

      {loading ? (
        <div className="loading-state">
          <Cpu className="spin-icon" size={48} />
          <p>DECRYPTING DATA STREAM...</p>
        </div>
      ) : (
        <div className="problems-grid">
          {problems.map(prob => (
            <Link href={`/problems/${prob.id}`} key={prob.id}>
              <div className="cyber-card cyber-chamfer problem-card">
                <div className="card-header">
                  <h3>{prob.title}</h3>
                  <ShieldAlert size={18} className="status-icon" />
                </div>
                
                <div className="card-metrics">
                  <div className="metric">
                    <span className="label">DIFF</span>
                    <span className="value">LEVEL_{prob.difficulty || 1}</span>
                  </div>
                  <div className="metric">
                    <span className="label">TIME</span>
                    <span className="value">{prob.time_limit_ms}ms</span>
                  </div>
                  <div className="metric">
                    <span className="label">MEM</span>
                    <span className="value">{prob.memory_limit_mb}MB</span>
                  </div>
                </div>

                <div className="card-action">
                  <span className="cyber-btn cyber-btn-outline cyber-chamfer-sm action-btn">
                    <Terminal size={14} /> INFILTRATE
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .explorer-container {
          padding: 4rem 2rem;
          max-width: 1200px;
          margin: 0 auto;
        }

        .dashboard-header {
          margin-bottom: 4rem;
          border-left: 4px solid var(--accent-primary);
          padding-left: 1.5rem;
        }

        .terminal-subtitle {
          color: var(--fg-muted);
          font-family: var(--font-body);
          margin-top: 1rem;
        }

        .prefix {
          color: var(--accent-secondary);
          font-weight: bold;
        }

        .loading-state {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          height: 300px;
          color: var(--accent-tertiary);
          font-family: var(--font-label);
          gap: 1.5rem;
        }

        .spin-icon {
          animation: spin 4s linear infinite;
        }

        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }

        .problems-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
          gap: 2rem;
        }

        .problem-card {
          display: flex;
          flex-direction: column;
          gap: 1.5rem;
          cursor: pointer;
        }

        .problem-card:hover .status-icon {
          color: var(--accent-secondary);
          filter: drop-shadow(var(--glow-secondary));
        }

        .problem-card:hover .action-btn {
          background: var(--accent-primary);
          color: var(--bg-void);
          border-color: var(--accent-primary);
          box-shadow: var(--glow-primary);
        }

        .card-header {
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
        }

        .card-header h3 {
          margin: 0;
          color: var(--accent-primary);
          font-size: 1.25rem;
        }

        .status-icon {
          color: var(--fg-muted);
          transition: all 0.3s ease;
        }

        .card-metrics {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          background: rgba(0, 0, 0, 0.4);
          padding: 1rem;
          border-left: 2px solid var(--accent-tertiary);
        }

        .metric {
          display: flex;
          flex-direction: column;
          gap: 0.25rem;
        }

        .label {
          font-family: var(--font-label);
          font-size: 0.7rem;
          color: var(--fg-muted);
        }

        .value {
          font-family: var(--font-body);
          font-size: 0.9rem;
          color: var(--fg-primary);
        }

        .card-action {
          margin-top: auto;
          display: flex;
          justify-content: flex-end;
        }
      `}</style>
    </div>
  );
}
