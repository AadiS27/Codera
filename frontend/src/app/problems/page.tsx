import Link from 'next/link';

// Mocked list of problems since we don't have a GET /problems endpoint yet
const MOCK_PROBLEMS = [
  { id: '1', title: 'Two Sum', difficulty: 'Easy', acceptance: '52.3%' },
  { id: '2', title: 'Add Two Numbers', difficulty: 'Medium', acceptance: '41.8%' },
  { id: '3', title: 'Longest Substring Without Repeating Characters', difficulty: 'Medium', acceptance: '34.5%' },
  { id: '4', title: 'Median of Two Sorted Arrays', difficulty: 'Hard', acceptance: '39.1%' },
];

export default function ProblemsList() {
  return (
    <div className="explorer-container">
      <header className="explorer-header">
        <h1>Command Center <span className="text-muted">/ Problems</span></h1>
        <Link href="/admin" className="btn-primary" style={{ fontSize: '0.8rem' }}>
          Initialize New Problem
        </Link>
      </header>

      <div className="problems-grid">
        {MOCK_PROBLEMS.map((prob) => (
          <Link href={`/problems/${prob.id}`} key={prob.id} className="problem-card glass-panel">
            <div className="card-top">
              <h3>{prob.title}</h3>
              <span className={`difficulty ${prob.difficulty.toLowerCase()}`}>
                {prob.difficulty}
              </span>
            </div>
            <div className="card-bottom">
              <span className="acceptance">Acc: {prob.acceptance}</span>
              <span className="action-text">Engage &rarr;</span>
            </div>
          </Link>
        ))}
      </div>

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .explorer-container {
          max-width: 1200px;
          margin: 0 auto;
          padding: 4rem 2rem;
          min-height: 100vh;
        }

        .explorer-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 3rem;
          border-bottom: 1px solid var(--border-subtle);
          padding-bottom: 1rem;
        }

        h1 {
          font-weight: 300;
          letter-spacing: -0.02em;
        }

        .text-muted {
          color: var(--text-muted);
        }

        .problems-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
          gap: 1.5rem;
        }

        .problem-card {
          display: flex;
          flex-direction: column;
          padding: 1.5rem;
          height: 160px;
          justify-content: space-between;
          text-decoration: none;
        }

        .card-top h3 {
          margin: 0 0 0.5rem 0;
          font-size: 1.2rem;
          color: var(--text-primary);
        }

        .difficulty {
          font-size: 0.8rem;
          text-transform: uppercase;
          letter-spacing: 0.1em;
          font-weight: 600;
        }

        .difficulty.easy { color: #00F0FF; }
        .difficulty.medium { color: #8A2BE2; }
        .difficulty.hard { color: #FF0055; }

        .card-bottom {
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 0.9rem;
          color: var(--text-secondary);
        }

        .action-text {
          color: var(--accent-cyan);
          opacity: 0;
          transform: translateX(-10px);
          transition: all 0.3s ease;
        }

        .problem-card:hover .action-text {
          opacity: 1;
          transform: translateX(0);
        }
      `}</style>
    </div>
  );
}
