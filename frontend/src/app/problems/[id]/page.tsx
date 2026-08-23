'use client';

import { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { Play, Send, Plus, X, BrainCircuit } from 'lucide-react';
import { use } from 'react';

interface RunResult {
  input: string;
  output: string;
  expected?: string;
  status: string;
  stderr?: string;
}

interface SubmitResult {
  verdict: string;
  passed: number;
  total: number;
  timeMs: number;
  aiTime?: string;
  aiSpace?: string;
  aiFeedback?: string;
}

const STARTER_CODE: Record<string, string> = {
  java: 'import java.util.*;\n\npublic class Main {\n  public static void main(String[] args) {\n    Scanner sc = new Scanner(System.in);\n    // Write your code here\n  }\n}',
  cpp: '#include <iostream>\n#include <vector>\nusing namespace std;\n\nint main() {\n  // Write your code here\n  return 0;\n}',
  python: 'import sys\n\ndef main():\n  # Write your code here\n  pass\n\nif __name__ == "__main__":\n  main()',
  go: 'package main\n\nimport (\n\t"fmt"\n)\n\nfunc main() {\n\t// Write your code here\n}',
};

export default function Workspace({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const [problem, setProblem] = useState<any>(null);
  const [testCases, setTestCases] = useState<any[]>([]);
  const [language, setLanguage] = useState('java');
  const [code, setCode] = useState(STARTER_CODE['java']);
  const [loading, setLoading] = useState(true);
  const [customInputs, setCustomInputs] = useState<string[]>(['']);
  const [activeTestCase, setActiveTestCase] = useState(0);
  const [isExecuting, setIsExecuting] = useState(false);

  const [consoleTab, setConsoleTab] = useState<'testcase' | 'result'>('testcase');
  const [activeResultCase, setActiveResultCase] = useState(0);
  const [runResults, setRunResults] = useState<RunResult[]>([]);
  const [submitResult, setSubmitResult] = useState<SubmitResult | null>(null);
  const [aiOnlyResult, setAiOnlyResult] = useState<any>(null);
  const [statusMessage, setStatusMessage] = useState('');

  useEffect(() => {
    fetch(`http://localhost:8080/problems/${resolvedParams.id}`)
      .then(res => res.json())
      .then(data => {
        setProblem(data.problem);
        setTestCases(data.test_cases || []);
        if (data.test_cases && data.test_cases.length > 0) {
          setCustomInputs([data.test_cases[0].input]);
        }
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, [resolvedParams.id]);

  const handleRun = async () => {
    setIsExecuting(true);
    setRunResults([]);
    setSubmitResult(null);
    setAiOnlyResult(null);
    setStatusMessage('Initiating execution sequence...');
    setConsoleTab('result');
    try {
      const res = await fetch('http://localhost:8080/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ language, source_code: code, inputs: customInputs })
      });
      const data = await res.json();
      if (data.id) {
        setStatusMessage('Job queued. Awaiting telemetry...');
        if (!res.ok) throw new Error('Run failed');

        const finished = { current: false };
        const ws = new WebSocket(`ws://localhost:8080/ws/run/${data.id}`);
        ws.onmessage = (event) => {
          const s = JSON.parse(event.data);
          if (s.status === 'COMPLETED' || s.status === 'DEAD_LETTERED' || s.status === 'FAILED') {
            finished.current = true;
            ws.close();
            setIsExecuting(false);
            if (s.results && s.results.length > 0) {
              setRunResults(s.results.map((r: any, idx: number) => {
                const output = (r.stdout || '').trim();
                const expected = testCases[idx]?.expected_output?.trim() || '';
                const isMatch = expected ? output === expected : undefined;
                return {
                  input: customInputs[idx] || '',
                  output,
                  expected: expected || undefined,
                  status: isMatch === true ? 'ACCEPTED' : isMatch === false ? 'WRONG_ANSWER' : r.status,
                  stderr: (r.stderr || '').trim(),
                };
              }));
              setActiveResultCase(0);
              setStatusMessage('');
            } else {
              setStatusMessage(`Final: ${s.status} | ${s.last_error || 'No details'}`);
            }
          } else {
            setStatusMessage(`State: ${s.status}...`);
          }
        };
        ws.onerror = () => { if (!finished.current) { setStatusMessage('WebSocket failed.'); setIsExecuting(false); } };
      }
    } catch (err: any) {
      setIsExecuting(false);
      setStatusMessage(`ERROR: ${err.message}`);
    }
  };

  const handleSubmit = async () => {
    setIsExecuting(true);
    setRunResults([]);
    setSubmitResult(null);
    setAiOnlyResult(null);
    setStatusMessage('Submitting to Judge Engine...');
    setConsoleTab('result');
    try {
      const res = await fetch(`http://localhost:8080/problems/${resolvedParams.id}/submissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: "demo-user", language, source_code: code })
      });
      const data = await res.json();
      if (data.id) {
        setStatusMessage('Evaluating test cases...');
        const finished = { current: false };
        const ws = new WebSocket(`ws://localhost:8080/ws/submissions/${data.id}`);
        ws.onmessage = (event) => {
          const s = JSON.parse(event.data);
          if (s.verdict !== 'PENDING' && s.status !== 'QUEUED' && s.status !== 'RUNNING') {
            finished.current = true;
            ws.close();
            setIsExecuting(false);
            setSubmitResult({
              verdict: s.verdict, passed: s.passed_test_cases, total: s.total_test_cases,
              timeMs: s.execution_time_ms, aiTime: s.ai_time_complexity,
              aiSpace: s.ai_space_complexity, aiFeedback: s.ai_feedback,
            });
            setStatusMessage('');
          } else {
            setStatusMessage(`Evaluation: ${s.status}...`);
          }
        };
        ws.onerror = () => { if (!finished.current) { setStatusMessage('WebSocket failed.'); setIsExecuting(false); } };
      }
    } catch (err: any) {
      setIsExecuting(false);
      setStatusMessage(`ERROR: ${err.message}`);
    }
  };

  const handleAiCheck = async () => {
    setIsExecuting(true);
    setAiOnlyResult(null);
    setStatusMessage('Analyzing code complexity via AI...');
    try {
      const res = await fetch('http://localhost:8080/analyze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ language, source_code: code, inputs: [] })
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setAiOnlyResult(data);
      setStatusMessage('');
    } catch (err: any) {
      setStatusMessage(`AI ERROR: ${err.message}`);
    } finally {
      setIsExecuting(false);
    }
  };

  const updateCustomInput = (val: string) => {
    const n = [...customInputs]; n[activeTestCase] = val; setCustomInputs(n);
  };
  const addTestCase = () => {
    setCustomInputs([...customInputs, '']); setActiveTestCase(customInputs.length);
  };
  const removeTestCase = (i: number) => {
    if (customInputs.length === 1) return;
    const n = customInputs.filter((_, idx) => idx !== i);
    setCustomInputs(n);
    if (activeTestCase >= n.length) setActiveTestCase(n.length - 1);
    else if (activeTestCase > i) setActiveTestCase(activeTestCase - 1);
  };

  const handleLanguageChange = (newLang: string) => {
    setLanguage(newLang);
    // Overwrite with new template ONLY if user hasn't modified the existing template
    if (Object.values(STARTER_CODE).includes(code)) {
      setCode(STARTER_CODE[newLang]);
    }
  };

  if (loading) {
    return <div className="loading-screen cyber-glitch-text" data-text="INITIALIZING...">INITIALIZING...<span className="blinking-cursor"></span></div>;
  }

  const vc = (v: string) => v === 'ACCEPTED' ? 'var(--accent-primary)' : 'var(--accent-destructive)';

  return (
    <div className="workspace-container">
      {/* Left Panel */}
      <div className="left-panel terminal-panel cyber-chamfer">
        <div className="panel-header">
          <h2>{problem?.title || "Unknown"}</h2>
          <div className="meta-tags">
            <span className="tag">TIME: {problem?.time_limit_ms}ms</span>
            <span className="tag">MEM: {problem?.memory_limit_mb}MB</span>
          </div>
        </div>
        <div className="panel-body problem-content">
          <div dangerouslySetInnerHTML={{ __html: problem?.description?.replace(/\n/g, '<br/>') || '' }} />
          <h3 className="section-title"><span className="prefix">&gt; </span>Input_Format</h3>
          <div className="technical-text cyber-chamfer-sm">{problem?.input_description}</div>
          <h3 className="section-title"><span className="prefix">&gt; </span>Output_Format</h3>
          <div className="technical-text cyber-chamfer-sm">{problem?.output_description}</div>
          <h3 className="section-title"><span className="prefix">&gt; </span>Constraints</h3>
          <div className="technical-text cyber-chamfer-sm">{problem?.constraints}</div>
        </div>
      </div>

      {/* Right Panel */}
      <div className="right-panel">
        <div className="editor-container holo-panel cyber-chamfer">
          <div className="panel-header editor-header">
            <div className="language-selector">
              <select className="cyber-select cyber-chamfer-sm" value={language} onChange={e => handleLanguageChange(e.target.value)}>
                <option value="java">JAVA_</option>
                <option value="cpp">CPP_</option>
                <option value="python">PYTHON_</option>
                <option value="go">GO_</option>
              </select>
            </div>
          </div>
          <div className="editor-wrapper">
            <Editor
              height="100%" defaultLanguage={language} language={language} theme="vs-dark"
              value={code} onChange={(val) => setCode(val || '')}
              options={{ fontFamily: 'var(--font-body)', fontSize: 14, minimap: { enabled: false }, scrollBeyondLastLine: false, lineNumbersMinChars: 3, padding: { top: 16 } }}
            />
          </div>
        </div>

        {/* Console */}
        <div className="console-container holo-panel cyber-chamfer">
          <div className="panel-header console-header">
            <div className="tabs">
              <span className={`tab ${consoleTab === 'testcase' ? 'active' : ''}`} onClick={() => setConsoleTab('testcase')}>Testcase</span>
              <span className={`tab ${consoleTab === 'result' ? 'active' : ''}`} onClick={() => setConsoleTab('result')}>Test Result</span>
            </div>
            <div className="action-buttons">
              <button className="cyber-btn cyber-btn-outline cyber-chamfer-sm" onClick={handleRun} disabled={isExecuting}><Play size={14} /> RUN</button>
              <button className="cyber-btn cyber-btn-primary cyber-chamfer-sm" onClick={handleSubmit} disabled={isExecuting}><Send size={14} /> SUBMIT</button>
            </div>
          </div>

          {/* Testcase Tab */}
          {consoleTab === 'testcase' && (
            <div className="console-body">
              <div className="tc-tabs">
                {customInputs.map((_, idx) => (
                  <div key={idx} className={`tc-tab ${idx === activeTestCase ? 'active' : ''}`} onClick={() => setActiveTestCase(idx)}>
                    Case {idx + 1}
                    {customInputs.length > 1 && (
                      <span className="remove-tc" onClick={(e) => { e.stopPropagation(); removeTestCase(idx); }}><X size={12} /></span>
                    )}
                  </div>
                ))}
                <div className="tc-tab add-tc" onClick={addTestCase}><Plus size={14} /></div>
              </div>
              <textarea
                className="custom-input"
                value={customInputs[activeTestCase] || ''}
                onChange={(e) => updateCustomInput(e.target.value)}
                placeholder=">_ Enter custom test input here..."
                spellCheck={false}
              />
            </div>
          )}

          {/* Test Result Tab */}
          {consoleTab === 'result' && (
            <div className="console-body result-body">
              {statusMessage && (
                <div className="result-status">
                  {isExecuting && <span className="spinner"></span>}
                  {statusMessage}<span className="blinking-cursor"></span>
                </div>
              )}

              {submitResult && (
                <div className="verdict-banner">
                  <div className="verdict-text" style={{ color: vc(submitResult.verdict) }}>
                    {submitResult.verdict}
                  </div>
                  <div className="verdict-meta">
                    Runtime: {submitResult.timeMs}ms &bull; Passed: {submitResult.passed}/{submitResult.total}
                  </div>
                  
                  {submitResult.aiTime ? (
                    <div className="ai-section">
                      <div className="ai-label">AI ANALYSIS</div>
                      <div className="ai-row"><span>Time:</span> {submitResult.aiTime}</div>
                      <div className="ai-row"><span>Space:</span> {submitResult.aiSpace}</div>
                      <div className="ai-row"><span>Feedback:</span> {submitResult.aiFeedback}</div>
                    </div>
                  ) : aiOnlyResult ? (
                    <div className="ai-section">
                      <div className="ai-label">AI ANALYSIS</div>
                      <div className="ai-row"><span>Time:</span> {aiOnlyResult.time_complexity}</div>
                      <div className="ai-row"><span>Space:</span> {aiOnlyResult.space_complexity}</div>
                      <div className="ai-row"><span>Feedback:</span> {aiOnlyResult.feedback}</div>
                    </div>
                  ) : submitResult.verdict === 'ACCEPTED' ? (
                    <div className="ai-section" style={{ borderTop: 'none', marginTop: '1rem', paddingTop: 0 }}>
                      <button 
                        className="cyber-btn cyber-btn-outline cyber-chamfer-sm" 
                        onClick={handleAiCheck} 
                        disabled={isExecuting}
                        style={{ borderColor: 'var(--accent-secondary)', color: 'var(--accent-secondary)', width: '100%', display: 'flex', justifyContent: 'center' }}
                      >
                        <BrainCircuit size={14} /> ASK AI FOR COMPLEXITY
                      </button>
                    </div>
                  ) : null}
                </div>
              )}

              {runResults.length > 0 && (() => {
                const cur = runResults[activeResultCase];
                const isAccepted = cur?.status === 'ACCEPTED';
                const isWrong = cur?.status === 'WRONG_ANSWER';
                const hasVerdict = isAccepted || isWrong;
                return (
                  <>
                    {hasVerdict && (
                      <div className="run-verdict-header" style={{ color: isAccepted ? 'var(--accent-primary)' : 'var(--accent-destructive)' }}>
                        {isAccepted ? 'Accepted' : 'Wrong Answer'}
                      </div>
                    )}
                    <div className="tc-tabs">
                      {runResults.map((r, idx) => (
                        <div key={idx} className={`tc-tab ${idx === activeResultCase ? 'active' : ''}`} onClick={() => setActiveResultCase(idx)}>
                          {r.status === 'ACCEPTED' && <span className="tc-indicator pass">✓</span>}
                          {r.status === 'WRONG_ANSWER' && <span className="tc-indicator fail">✗</span>}
                          Case {idx + 1}
                        </div>
                      ))}
                    </div>
                    <div className="result-rows">
                      <div className="result-field">
                        <div className="field-label">Input</div>
                        <div className="field-value">{cur?.input || 'N/A'}</div>
                      </div>
                      <div className="result-field">
                        <div className="field-label">Output</div>
                        <div className={`field-value ${isWrong ? 'field-wrong' : isAccepted ? 'field-pass' : ''}`}>{cur?.output || 'N/A'}</div>
                      </div>
                      {cur?.expected && (
                        <div className="result-field">
                          <div className="field-label">Expected</div>
                          <div className="field-value">{cur.expected}</div>
                        </div>
                      )}
                      {cur?.stderr && (
                        <div className="result-field error-field">
                          <div className="field-label">Stderr</div>
                          <div className="field-value">{cur.stderr}</div>
                        </div>
                      )}
                    </div>
                  </>
                );
              })()}

              {!statusMessage && !submitResult && !aiOnlyResult && runResults.length === 0 && (
                <div className="result-empty">&gt;_ Run or submit your code to see results here.<span className="blinking-cursor"></span></div>
              )}
            </div>
          )}
        </div>
      </div>

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .loading-screen { height: 100vh; display: flex; align-items: center; justify-content: center; font-size: 1.5rem; }
        .workspace-container { display: grid; grid-template-columns: 40% 60%; height: 100vh; padding: 1.5rem; gap: 1.5rem; }
        .panel-header { padding: 1rem 1.5rem; border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center; background: rgba(0,0,0,0.4); }
        .panel-body { padding: 1.5rem; overflow-y: auto; height: calc(100% - 60px); }
        .meta-tags { display: flex; gap: 0.5rem; }
        .tag { font-size: 0.75rem; font-family: var(--font-label); padding: 0.2rem 0.5rem; background: var(--border-subtle); color: var(--accent-tertiary); box-shadow: inset 0 0 5px rgba(0,212,255,0.2); }
        .problem-content { font-size: 1rem; line-height: 1.7; color: var(--fg-primary); }
        .section-title { margin-top: 2rem; font-size: 1rem; color: var(--accent-primary); font-family: var(--font-label); }
        .prefix { color: var(--accent-secondary); }
        .technical-text { font-family: var(--font-body); background: rgba(0,0,0,0.5); padding: 1rem; border-left: 2px solid var(--accent-primary); font-size: 0.9rem; color: var(--fg-muted); white-space: pre-wrap; margin-top: 0.5rem; }
        .right-panel { display: flex; flex-direction: column; gap: 1rem; }
        .editor-container { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
        .lang-select { width: auto; padding: 0.3rem 1rem; background: transparent; border: none; color: var(--accent-tertiary); font-weight: bold; cursor: pointer; font-family: var(--font-label); }
        .editor-wrapper { flex: 1; position: relative; z-index: 100; }
        .console-container { height: 38%; display: flex; flex-direction: column; }
        .console-header { padding: 0.5rem 1rem; }
        .tabs { display: flex; gap: 1.5rem; }
        .tab { font-size: 0.85rem; font-family: var(--font-label); color: var(--fg-muted); cursor: pointer; padding-bottom: 0.5rem; }
        .tab.active { color: var(--accent-primary); border-bottom: 2px solid var(--accent-primary); text-shadow: var(--glow-primary-sm); }
        .action-buttons { display: flex; gap: 0.5rem; }
        .console-body { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
        .result-body { overflow-y: auto; }
        .tc-tabs { display: flex; background: rgba(0,0,0,0.4); overflow-x: auto; border-bottom: 1px solid var(--border-subtle); flex-shrink: 0; }
        .tc-tab { padding: 0.6rem 1rem; font-size: 0.75rem; font-family: var(--font-label); color: var(--fg-muted); cursor: pointer; border-right: 1px solid var(--border-subtle); user-select: none; display: flex; align-items: center; gap: 0.5rem; white-space: nowrap; transition: all 150ms steps(4); }
        .tc-tab:hover { background: var(--border-subtle); color: var(--fg-primary); }
        .tc-tab.active { background: rgba(0,255,136,0.1); color: var(--accent-primary); border-bottom: 2px solid var(--accent-primary); }
        .remove-tc { color: var(--accent-destructive); display: flex; align-items: center; }
        .remove-tc:hover { color: var(--fg-primary); background: var(--accent-destructive); }
        .add-tc { font-weight: bold; color: var(--accent-primary); display: flex; align-items: center; justify-content: center; }
        .add-tc:hover { background: rgba(0,255,136,0.2); }
        .custom-input { flex: 1; border: none; border-radius: 0; resize: none; background: transparent; padding: 1rem; font-family: var(--font-body); font-size: 0.9rem; color: var(--accent-tertiary); min-height: 0; }
        .custom-input:focus { outline: none; box-shadow: none; background: rgba(0,212,255,0.05); }
        .result-status { padding: 1rem 1.5rem; font-family: var(--font-body); color: var(--accent-tertiary); display: flex; align-items: center; gap: 0.75rem; }
        .spinner { width: 14px; height: 14px; border: 2px solid var(--border-subtle); border-top: 2px solid var(--accent-tertiary); border-radius: 50%; animation: spin 0.6s linear infinite; }
        @keyframes spin { to { transform: rotate(360deg); } }
        .verdict-banner { padding: 1.5rem; }
        .verdict-text { font-family: var(--font-heading); font-size: 1.5rem; font-weight: 800; margin-bottom: 0.5rem; }
        .verdict-meta { font-family: var(--font-body); color: var(--fg-muted); font-size: 0.85rem; }
        .ai-section { margin-top: 1rem; padding-top: 1rem; border-top: 1px solid var(--border-subtle); }
        .ai-label { font-family: var(--font-label); font-size: 0.7rem; color: var(--accent-secondary); letter-spacing: 0.1em; margin-bottom: 0.5rem; }
        .ai-row { font-family: var(--font-body); font-size: 0.85rem; color: var(--fg-primary); margin-bottom: 0.25rem; }
        .ai-row span { color: var(--fg-muted); }
        .result-rows { padding: 1rem 1.5rem; display: flex; flex-direction: column; gap: 1rem; }
        .result-field { display: flex; flex-direction: column; gap: 0.4rem; }
        .field-label { font-family: var(--font-label); font-size: 0.75rem; color: var(--fg-muted); letter-spacing: 0.05em; }
        .field-value { font-family: var(--font-body); font-size: 0.9rem; background: rgba(0,0,0,0.5); padding: 0.75rem 1rem; color: var(--fg-primary); white-space: pre-wrap; border: 1px solid var(--border-subtle); }
        .error-field .field-value { color: var(--accent-destructive); border-color: rgba(255,51,102,0.3); }
        .result-empty { padding: 2rem; color: var(--fg-muted); font-family: var(--font-body); text-align: center; }
        .run-verdict-header { font-family: var(--font-heading); font-size: 1.3rem; font-weight: 800; padding: 1rem 1.5rem 0; }
        .tc-indicator { font-weight: bold; font-size: 0.85rem; }
        .tc-indicator.pass { color: var(--accent-primary); }
        .tc-indicator.fail { color: var(--accent-destructive); }
        .field-wrong { border-color: rgba(255,51,102,0.4); color: var(--accent-destructive); }
        .field-pass { border-color: rgba(0,255,136,0.3); }
      `}</style>
    </div>
  );
}
