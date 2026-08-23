'use client';

import { useState } from 'react';
import Link from 'next/link';

export default function AdminPage() {
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState('');
  
  const [problem, setProblem] = useState({
    title: '',
    slug: '',
    description: '',
    input_description: '',
    output_description: '',
    constraints: '',
    time_limit_ms: 2000,
    memory_limit_mb: 256,
    comparison_mode: 'EXACT',
    status: 'PUBLISHED'
  });

  const [testCases, setTestCases] = useState([
    { input: '', expected_output: '', visibility: 'PUBLIC' }
  ]);

  const handleSubmit = async (e: any) => {
    e.preventDefault();
    setLoading(true);
    setSuccess('');
    
    try {
      const res = await fetch('http://localhost:8080/admin/problems/full', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          problem: {
            ...problem,
            time_limit_ms: parseInt(problem.time_limit_ms.toString(), 10),
            memory_limit_mb: parseInt(problem.memory_limit_mb.toString(), 10),
          },
          test_cases: testCases
        })
      });

      if (!res.ok) {
        throw new Error(await res.text());
      }
      
      const data = await res.json();
      setSuccess(`Problem created with ID: ${data.problem.id}`);
      // Reset form
      setProblem({ ...problem, title: '', slug: '', description: '' });
      setTestCases([{ input: '', expected_output: '', visibility: 'PUBLIC' }]);
    } catch (err: any) {
      alert(`Error: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const addTestCase = () => {
    setTestCases([...testCases, { input: '', expected_output: '', visibility: 'HIDDEN' }]);
  };

  const updateTestCase = (index: number, field: string, value: string) => {
    const newTc = [...testCases];
    newTc[index] = { ...newTc[index], [field]: value };
    setTestCases(newTc);
  };

  return (
    <div className="admin-container">
      <header className="admin-header">
        <h1>Command Center <span className="text-muted">/ System Override</span></h1>
        <Link href="/problems" className="btn-secondary" style={{ fontSize: '0.8rem' }}>
          &larr; Return to Workspace
        </Link>
      </header>

      {success && <div className="success-banner">{success}</div>}

      <form onSubmit={handleSubmit} className="admin-form glass-panel">
        <h2 className="section-title">Initialize Target Parameters</h2>
        <div className="form-grid">
          <div className="form-group">
            <label>Designation (Title)</label>
            <input required value={problem.title} onChange={e => setProblem({...problem, title: e.target.value})} />
          </div>
          <div className="form-group">
            <label>Slug Identifier</label>
            <input required value={problem.slug} onChange={e => setProblem({...problem, slug: e.target.value})} />
          </div>
        </div>

        <div className="form-group">
          <label>Mission Brief (Description)</label>
          <textarea required rows={5} value={problem.description} onChange={e => setProblem({...problem, description: e.target.value})} />
        </div>

        <div className="form-grid">
          <div className="form-group">
            <label>Input Vector Format</label>
            <textarea required rows={3} value={problem.input_description} onChange={e => setProblem({...problem, input_description: e.target.value})} />
          </div>
          <div className="form-group">
            <label>Expected Output Format</label>
            <textarea required rows={3} value={problem.output_description} onChange={e => setProblem({...problem, output_description: e.target.value})} />
          </div>
        </div>

        <div className="form-group">
          <label>System Constraints</label>
          <textarea required rows={2} value={problem.constraints} onChange={e => setProblem({...problem, constraints: e.target.value})} />
        </div>

        <div className="form-grid">
          <div className="form-group">
            <label>Time Limit (ms)</label>
            <input type="number" required value={problem.time_limit_ms} onChange={e => setProblem({...problem, time_limit_ms: parseInt(e.target.value) || 0})} />
          </div>
          <div className="form-group">
            <label>Memory Limit (MB)</label>
            <input type="number" required value={problem.memory_limit_mb} onChange={e => setProblem({...problem, memory_limit_mb: parseInt(e.target.value) || 0})} />
          </div>
          <div className="form-group">
            <label>Comparison Protocol</label>
            <select value={problem.comparison_mode} onChange={e => setProblem({...problem, comparison_mode: e.target.value})}>
              <option value="EXACT">Strict (Byte for Byte)</option>
              <option value="NORMALIZED_WHITESPACE">Forgiving Whitespace</option>
              <option value="FLOAT_EPSILON">Floating Point Epsilon</option>
            </select>
          </div>
        </div>

        <h2 className="section-title" style={{ marginTop: '3rem' }}>Validation Matrices (Test Cases)</h2>
        
        {testCases.map((tc, idx) => (
          <div key={idx} className="test-case-block">
            <div className="tc-header">
              <h3>Matrix #{idx + 1}</h3>
              <select value={tc.visibility} onChange={e => updateTestCase(idx, 'visibility', e.target.value)}>
                <option value="PUBLIC">Public Exposure</option>
                <option value="HIDDEN">Classified (Hidden)</option>
              </select>
            </div>
            <div className="form-grid">
              <div className="form-group">
                <label>Input Data</label>
                <textarea rows={3} value={tc.input} onChange={e => updateTestCase(idx, 'input', e.target.value)} />
              </div>
              <div className="form-group">
                <label>Expected Output</label>
                <textarea rows={3} value={tc.expected_output} onChange={e => updateTestCase(idx, 'expected_output', e.target.value)} />
              </div>
            </div>
          </div>
        ))}

        <div className="form-actions">
          <button type="button" className="btn-secondary" onClick={addTestCase}>
            + Add Validation Matrix
          </button>
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? 'UPLOADING...' : 'COMMIT TO CORE'}
          </button>
        </div>
      </form>

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .admin-container {
          max-width: 900px;
          margin: 0 auto;
          padding: 4rem 2rem;
          min-height: 100vh;
        }

        .admin-header {
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

        .admin-form {
          padding: 2.5rem;
        }

        .section-title {
          font-size: 1.1rem;
          color: var(--accent-cyan);
          text-transform: uppercase;
          letter-spacing: 0.1em;
          margin-bottom: 1.5rem;
          border-bottom: 1px solid rgba(0, 240, 255, 0.2);
          padding-bottom: 0.5rem;
          display: inline-block;
        }

        .form-grid {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 1.5rem;
          margin-bottom: 1.5rem;
        }

        .form-group {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
          margin-bottom: 1.5rem;
        }

        label {
          font-size: 0.85rem;
          color: var(--text-secondary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .test-case-block {
          background: rgba(0, 0, 0, 0.3);
          border: 1px solid var(--border-subtle);
          padding: 1.5rem;
          border-radius: 4px;
          margin-bottom: 1.5rem;
        }

        .tc-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 1rem;
        }

        .tc-header h3 {
          margin: 0;
          font-size: 1rem;
          color: var(--text-primary);
        }

        .form-actions {
          display: flex;
          justify-content: space-between;
          margin-top: 3rem;
          border-top: 1px solid var(--border-subtle);
          padding-top: 2rem;
        }

        .success-banner {
          background: rgba(0, 240, 255, 0.1);
          border: 1px solid var(--accent-cyan);
          color: var(--accent-cyan);
          padding: 1rem;
          border-radius: 4px;
          margin-bottom: 2rem;
          text-align: center;
          font-family: var(--font-mono);
        }
      `}</style>
    </div>
  );
}
