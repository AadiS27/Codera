'use client';

import { useState } from 'react';
import { Database, Plus, Save } from 'lucide-react';

export default function AdminPanel() {
  const [formData, setFormData] = useState({
    title: '',
    description: '',
    difficulty: 1,
    time_limit_ms: 2000,
    memory_limit_mb: 256,
    input_description: '',
    output_description: '',
    constraints: '',
    test_cases: [
      { input: '', expected_output: '', is_hidden: false, explanation: '' }
    ]
  });

  const [status, setStatus] = useState({ type: '', message: '' });

  const [showBulkModal, setShowBulkModal] = useState(false);
  const [bulkJSON, setBulkJSON] = useState('');
  const [bulkError, setBulkError] = useState('');

  const handleBulkImport = () => {
    try {
      const parsed = JSON.parse(bulkJSON);
      if (!Array.isArray(parsed)) throw new Error('Root must be a JSON array');
      
      const newCases = parsed.map(tc => ({
        input: tc.input || '',
        expected_output: tc.expected_output || tc.output || '',
        is_hidden: !!tc.is_hidden,
        explanation: tc.explanation || ''
      }));

      const current = formData.test_cases;
      if (current.length === 1 && current[0].input === '' && current[0].expected_output === '') {
        setFormData(prev => ({ ...prev, test_cases: newCases }));
      } else {
        setFormData(prev => ({ ...prev, test_cases: [...prev.test_cases, ...newCases] }));
      }
      
      setShowBulkModal(false);
      setBulkJSON('');
      setBulkError('');
    } catch (e) {
      const err = e as Error;
      setBulkError(`JSON Error: ${err.message}`);
    }
  };

  const addTestCase = () => {
    setFormData(prev => ({
      ...prev,
      test_cases: [...prev.test_cases, { input: '', expected_output: '', is_hidden: false, explanation: '' }]
    }));
  };

  const updateTestCase = (index: number, field: string, value: string | boolean) => {
    const newTestCases = [...formData.test_cases];
    newTestCases[index] = { ...newTestCases[index], [field]: value };
    setFormData(prev => ({ ...prev, test_cases: newTestCases }));
  };

  const removeTestCase = (index: number) => {
    const newTestCases = formData.test_cases.filter((_, i) => i !== index);
    setFormData(prev => ({ ...prev, test_cases: newTestCases }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatus({ type: 'loading', message: '> TRANSMITTING PROTOCOL...' });

    try {
      const res = await fetch('http://localhost:8080/admin/problems/full', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData)
      });

      if (!res.ok) {
        throw new Error(`HTTP error! status: ${res.status}`);
      }

      const result = await res.json();
      setStatus({ type: 'success', message: `> SUCCESS: ENTRY [${result.id}] COMMITTED.` });
      
      // Reset form on success
      setFormData({
        title: '',
        description: '',
        difficulty: 1,
        time_limit_ms: 2000,
        memory_limit_mb: 256,
        input_description: '',
        output_description: '',
        constraints: '',
        test_cases: [{ input: '', expected_output: '', is_hidden: false, explanation: '' }]
      });
      
    } catch (err) {
      const e = err as Error;
      setStatus({ type: 'error', message: `> ERROR: ${e.message}` });
    }
  };

  return (
    <div className="admin-container">
      <div className="admin-header">
        <h1 className="cyber-glitch-text" data-text="SYS.OVERRIDE // AUTHOR">
          SYS.OVERRIDE // AUTHOR
        </h1>
        <p className="terminal-subtitle">
          <span className="prefix">&gt; </span> WARNING: Level 4 Authorization required. Modifying core algorithms...<span className="blinking-cursor"></span>
        </p>
      </div>

      <div className="admin-body">
        <form onSubmit={handleSubmit} className="cyber-form holo-panel cyber-chamfer">
          
          <div className="form-grid">
            <div className="form-group span-full">
              <label>PROTOCOL_NAME [TITLE]</label>
              <input 
                type="text" 
                className="cyber-chamfer-sm"
                value={formData.title} 
                onChange={e => setFormData({...formData, title: e.target.value})} 
                required 
              />
            </div>

            <div className="form-group span-half">
              <label>THREAT_LEVEL [DIFFICULTY 1-5]</label>
              <input 
                type="number" 
                className="cyber-chamfer-sm"
                min="1" max="5" 
                value={formData.difficulty} 
                onChange={e => setFormData({...formData, difficulty: parseInt(e.target.value)})} 
                required 
              />
            </div>

            <div className="form-group span-full">
              <label>PROBLEM_STATEMENT [MD_SUPPORTED]</label>
              <textarea 
                className="cyber-chamfer-sm"
                rows={6}
                value={formData.description} 
                onChange={e => setFormData({...formData, description: e.target.value})} 
                required 
              />
            </div>

            <div className="form-group span-half">
              <label>INPUT_SCHEMA</label>
              <textarea 
                className="cyber-chamfer-sm"
                rows={3}
                value={formData.input_description} 
                onChange={e => setFormData({...formData, input_description: e.target.value})} 
              />
            </div>

            <div className="form-group span-half">
              <label>OUTPUT_SCHEMA</label>
              <textarea 
                className="cyber-chamfer-sm"
                rows={3}
                value={formData.output_description} 
                onChange={e => setFormData({...formData, output_description: e.target.value})} 
              />
            </div>

            <div className="form-group span-full">
              <label>SYSTEM_CONSTRAINTS</label>
              <textarea 
                className="cyber-chamfer-sm"
                rows={3}
                value={formData.constraints} 
                onChange={e => setFormData({...formData, constraints: e.target.value})} 
              />
            </div>
          </div>

          <div className="test-cases-section">
            <div className="section-header">
              <h3><Database size={18} /> VALIDATION_MATRICES [TEST_CASES]</h3>
              <div style={{ display: 'flex', gap: '1rem' }}>
                <button type="button" onClick={() => setShowBulkModal(!showBulkModal)} className="cyber-btn cyber-btn-outline cyber-chamfer-sm">
                  <Plus size={14} /> BULK_IMPORT (JSON)
                </button>
                <button type="button" onClick={addTestCase} className="cyber-btn cyber-btn-outline cyber-chamfer-sm">
                  <Plus size={14} /> ADD_MATRIX
                </button>
              </div>
            </div>

            {showBulkModal && (
              <div className="bulk-import-panel cyber-card cyber-chamfer-sm" style={{ marginBottom: '1.5rem', padding: '1.5rem' }}>
                <h4 style={{ margin: '0 0 0.5rem 0', color: 'var(--fg-primary)', fontFamily: 'var(--font-label)' }}>BULK IMPORT VIA JSON</h4>
                <p className="terminal-subtitle" style={{ margin: '0 0 1rem 0' }}>Paste an array of objects with &quot;input&quot; and &quot;expected_output&quot; keys.</p>
                <textarea
                  className="cyber-input"
                  rows={8}
                  placeholder='[&#10;  { "input": "1 2\n", "expected_output": "3\n" },&#10;  { "input": "5 5\n", "expected_output": "10\n", "is_hidden": true }&#10;]'
                  value={bulkJSON}
                  onChange={e => setBulkJSON(e.target.value)}
                  style={{ width: '100%', marginBottom: '1rem', fontFamily: 'monospace' }}
                />
                {bulkError && <div className="status-message error" style={{ marginBottom: '1rem' }}>{bulkError}</div>}
                <div style={{ display: 'flex', gap: '1rem', justifyContent: 'flex-end' }}>
                  <button type="button" onClick={() => setShowBulkModal(false)} className="cyber-btn cyber-btn-outline cyber-chamfer-sm">
                    CANCEL
                  </button>
                  <button type="button" onClick={handleBulkImport} className="cyber-btn cyber-btn-primary cyber-chamfer-sm">
                    IMPORT
                  </button>
                </div>
              </div>
            )}
            
            <div className="test-cases-grid">
              {formData.test_cases.map((tc, idx) => (
                <div key={idx} className="test-case-card cyber-card cyber-chamfer-sm">
                  <div className="tc-header">
                    <h4>MATRIX_{idx + 1}</h4>
                    {formData.test_cases.length > 1 && (
                      <button 
                        type="button" 
                        onClick={() => removeTestCase(idx)}
                        className="remove-btn"
                      >
                        [X]
                      </button>
                    )}
                  </div>
                  
                  <div className="tc-body">
                    <div className="form-group">
                      <label>INPUT_PAYLOAD</label>
                      <textarea 
                        className="cyber-input"
                        rows={3}
                        value={tc.input} 
                        onChange={e => updateTestCase(idx, 'input', e.target.value)} 
                        required 
                      />
                    </div>
                    
                    <div className="form-group">
                      <label>EXPECTED_STATE</label>
                      <textarea 
                        className="cyber-input"
                        rows={3}
                        value={tc.expected_output} 
                        onChange={e => updateTestCase(idx, 'expected_output', e.target.value)} 
                        required 
                      />
                    </div>
                    
                    <div className="tc-footer">
                      <label className="checkbox-label">
                        <input 
                          type="checkbox" 
                          checked={tc.is_hidden} 
                          onChange={e => updateTestCase(idx, 'is_hidden', e.target.checked)} 
                        />
                        <span className="checkbox-text">CLASSIFIED (HIDDEN)</span>
                      </label>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="form-actions">
            {status.message && (
              <div className={`status-message ${status.type}`}>
                {status.message}
              </div>
            )}
            <button type="submit" className="cyber-btn cyber-btn-primary cyber-chamfer" disabled={status.type === 'loading'}>
              <Save size={16} /> INITIALIZE_OVERRIDE
            </button>
          </div>
        </form>
      </div>

      <style jsx>{`
        .admin-container {
          padding: 4rem 2rem;
          max-width: 1200px;
          margin: 0 auto;
        }

        .admin-header {
          margin-bottom: 4rem;
          border-left: 4px solid var(--accent-destructive);
          padding-left: 1.5rem;
        }

        .terminal-subtitle {
          color: var(--fg-muted);
          font-family: var(--font-body);
          margin-top: 1rem;
        }

        .prefix {
          color: var(--accent-destructive);
          font-weight: bold;
        }

        .cyber-form {
          padding: 3rem;
        }

        .form-grid {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 2rem;
          margin-bottom: 4rem;
        }

        .form-group {
          display: flex;
          flex-direction: column;
          gap: 0.75rem;
        }

        .span-full {
          grid-column: 1 / -1;
        }

        .span-half {
          grid-column: span 3;
        }
        @media (min-width: 768px) {
          .span-half {
            grid-column: span 1.5; /* Note: Grid doesn't support fractional spans, so we use 3 columns and span 2/1 usually. Let's adjust layout. */
          }
        }

        label {
          font-family: var(--font-label);
          color: var(--accent-tertiary);
          font-size: 0.85rem;
          letter-spacing: 0.05em;
        }

        textarea {
          resize: vertical;
        }

        .test-cases-section {
          margin-bottom: 3rem;
          border-top: 1px solid var(--border-subtle);
          padding-top: 3rem;
        }

        .section-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 2rem;
        }

        .section-header h3 {
          margin: 0;
          color: var(--accent-primary);
          display: flex;
          align-items: center;
          gap: 0.75rem;
        }

        .test-cases-grid {
          display: grid;
          grid-template-columns: 1fr;
          gap: 1.5rem;
        }

        .test-case-card {
          background: rgba(0, 0, 0, 0.4);
          padding: 1.5rem;
          border: 1px solid var(--border-subtle);
        }

        .tc-header {
          display: flex;
          justify-content: space-between;
          margin-bottom: 1.5rem;
          border-bottom: 1px solid var(--border-subtle);
          padding-bottom: 0.5rem;
        }

        .tc-header h4 {
          margin: 0;
          color: var(--fg-primary);
          font-family: var(--font-label);
        }

        .remove-btn {
          background: transparent;
          border: none;
          color: var(--accent-destructive);
          font-family: var(--font-label);
          cursor: pointer;
        }

        .remove-btn:hover {
          text-shadow: var(--glow-destructive);
        }

        .tc-body {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 1.5rem;
        }

        .cyber-input {
          background: transparent;
          border: 1px solid var(--border-subtle);
          color: var(--fg-primary);
          padding: 0.5rem;
          font-family: var(--font-body);
        }

        .cyber-input:focus {
          border-color: var(--accent-tertiary);
          outline: none;
        }

        .tc-footer {
          grid-column: 1 / -1;
          display: flex;
          justify-content: flex-end;
        }

        .checkbox-label {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          cursor: pointer;
        }

        .checkbox-label input {
          width: auto;
          accent-color: var(--accent-destructive);
        }

        .checkbox-text {
          color: var(--fg-muted);
        }

        .form-actions {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-top: 3rem;
          border-top: 1px solid var(--border-subtle);
          padding-top: 2rem;
        }

        .status-message {
          font-family: var(--font-body);
          font-size: 0.9rem;
        }

        .status-message.loading { color: var(--accent-tertiary); }
        .status-message.success { color: var(--accent-primary); text-shadow: var(--glow-primary-sm); }
        .status-message.error { color: var(--accent-destructive); text-shadow: var(--glow-destructive); }
      `}</style>
    </div>
  );
}
