'use client';

import { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import { Play, Send } from 'lucide-react';
import { use } from 'react';

export default function Workspace({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const [problem, setProblem] = useState<any>(null);
  const [testCases, setTestCases] = useState<any[]>([]);
  const [language, setLanguage] = useState('java');
  const [code, setCode] = useState('class Solution {\n  public void solve() {\n    // Write your code here\n  }\n}');
  const [loading, setLoading] = useState(true);
  const [customInput, setCustomInput] = useState('');
  const [consoleOutput, setConsoleOutput] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);

  useEffect(() => {
    fetch(`http://localhost:8080/problems/${resolvedParams.id}`)
      .then(res => res.json())
      .then(data => {
        setProblem(data.problem);
        setTestCases(data.test_cases || []);
        if (data.test_cases && data.test_cases.length > 0) {
          setCustomInput(data.test_cases[0].input);
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
    setConsoleOutput('Initiating execution sequence...\n');
    try {
      const res = await fetch('http://localhost:8080/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          language,
          source_code: code,
          input: customInput,
        })
      });
      const data = await res.json();
      
      // We would poll here if it's async, but for simplicity let's assume it returns ID and we poll
      if (data.id) {
        setConsoleOutput(prev => prev + `Job ${data.id} queued. Awaiting telemetry...\n`);
        
        let attempts = 0;
        const poll = setInterval(async () => {
          attempts++;
          const statusRes = await fetch(`http://localhost:8080/run/${data.id}`);
          const statusData = await statusRes.json();
          
          if (statusData.status === 'COMPLETED' || statusData.status === 'FAILED') {
            clearInterval(poll);
            setIsExecuting(false);
            if (statusData.result) {
              setConsoleOutput(`Execution Completed [${statusData.result.exit_code}]\n\nSTDOUT:\n${statusData.result.stdout || '(none)'}\n\nSTDERR:\n${statusData.result.stderr || '(none)'}`);
            } else {
              setConsoleOutput(`Job Final State: ${statusData.status}`);
            }
          } else if (attempts > 30) {
            clearInterval(poll);
            setIsExecuting(false);
            setConsoleOutput('Execution timed out.');
          }
        }, 1000);
      }
    } catch (err: any) {
      setIsExecuting(false);
      setConsoleOutput(`CRITICAL FAILURE: ${err.message}`);
    }
  };

  const handleSubmit = async () => {
    setIsExecuting(true);
    setConsoleOutput('Submitting to Judge Engine...\n');
    try {
      const res = await fetch(`http://localhost:8080/problems/${resolvedParams.id}/submissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: "demo-user",
          language,
          source_code: code,
        })
      });
      const data = await res.json();
      
      if (data.id) {
        setConsoleOutput(prev => prev + `Submission ${data.id} registered.\nEvaluating test cases...\n`);
        
        let attempts = 0;
        const poll = setInterval(async () => {
          attempts++;
          const statusRes = await fetch(`http://localhost:8080/submissions/${data.id}`);
          const statusData = await statusRes.json();
          
          if (statusData.verdict !== 'PENDING' && statusData.status !== 'QUEUED' && statusData.status !== 'RUNNING') {
            clearInterval(poll);
            setIsExecuting(false);
            
            let color = statusData.verdict === 'ACCEPTED' ? 'var(--accent-cyan)' : '#FF0055';
            setConsoleOutput(`\nJUDGEMENT: ${statusData.verdict}\nPassed: ${statusData.passed_test_cases} / ${statusData.total_test_cases}\nExecution Time: ${statusData.execution_time_ms}ms`);
          } else if (attempts > 30) {
            clearInterval(poll);
            setIsExecuting(false);
            setConsoleOutput('Evaluation timed out.');
          }
        }, 1000);
      }
    } catch (err: any) {
      setIsExecuting(false);
      setConsoleOutput(`SYSTEM ERROR: ${err.message}`);
    }
  };

  if (loading) {
    return <div className="loading-screen">INITIALIZING WORKSPACE...</div>;
  }

  return (
    <div className="workspace-container">
      <div className="left-panel glass-panel">
        <div className="panel-header">
          <h2>{problem?.title || "Unknown Problem"}</h2>
          <div className="meta-tags">
            <span className="tag">Time: {problem?.time_limit_ms}ms</span>
            <span className="tag">Mem: {problem?.memory_limit_mb}MB</span>
          </div>
        </div>
        <div className="panel-body problem-content">
          <div dangerouslySetInnerHTML={{ __html: problem?.description?.replace(/\n/g, '<br/>') || '' }} />
          
          <h3 className="section-title">Input Format</h3>
          <div className="technical-text">{problem?.input_description}</div>
          
          <h3 className="section-title">Output Format</h3>
          <div className="technical-text">{problem?.output_description}</div>
          
          <h3 className="section-title">Constraints</h3>
          <div className="technical-text constraints">{problem?.constraints}</div>
        </div>
      </div>

      <div className="right-panel">
        <div className="editor-container glass-panel">
          <div className="panel-header editor-header">
            <select 
              value={language} 
              onChange={(e) => setLanguage(e.target.value)}
              className="lang-select"
            >
              <option value="java">Java</option>
              <option value="python">Python</option>
              <option value="go">Go</option>
              <option value="cpp">C++</option>
            </select>
          </div>
          <div className="editor-wrapper">
            <Editor
              height="100%"
              defaultLanguage={language}
              language={language}
              theme="vs-dark"
              value={code}
              onChange={(val) => setCode(val || '')}
              options={{
                fontFamily: 'var(--font-mono)',
                fontSize: 14,
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                lineNumbersMinChars: 3,
                padding: { top: 16 }
              }}
            />
          </div>
        </div>

        <div className="console-container glass-panel">
          <div className="panel-header console-header">
            <div className="tabs">
              <span className="tab active">Custom Input</span>
              <span className="tab">Execution Log</span>
            </div>
            <div className="action-buttons">
              <button 
                className="btn-secondary sm-btn" 
                onClick={handleRun}
                disabled={isExecuting}
              >
                <Play size={14} /> Run
              </button>
              <button 
                className="btn-primary sm-btn"
                onClick={handleSubmit}
                disabled={isExecuting}
              >
                <Send size={14} /> Submit
              </button>
            </div>
          </div>
          <div className="console-body">
            <div className="split-console">
              <textarea 
                className="custom-input" 
                value={customInput}
                onChange={(e) => setCustomInput(e.target.value)}
                placeholder="Enter custom test input here..."
                spellCheck={false}
              />
              <pre className="output-log">{consoleOutput || 'Ready for execution...'}</pre>
            </div>
          </div>
        </div>
      </div>

      <style /* eslint-disable-next-line react/no-unknown-property */ jsx>{`
        .loading-screen {
          height: 100vh;
          display: flex;
          align-items: center;
          justify-content: center;
          font-family: var(--font-mono);
          color: var(--accent-cyan);
          letter-spacing: 0.2em;
          animation: pulse 2s infinite;
        }

        .workspace-container {
          display: grid;
          grid-template-columns: 40% 60%;
          height: 100vh;
          padding: 1rem;
          gap: 1rem;
          background: radial-gradient(ellipse at top, rgba(138, 43, 226, 0.05) 0%, transparent 50%),
                      radial-gradient(ellipse at bottom, rgba(0, 240, 255, 0.05) 0%, transparent 50%);
        }

        .panel-header {
          padding: 1rem 1.5rem;
          border-bottom: 1px solid var(--border-subtle);
          display: flex;
          justify-content: space-between;
          align-items: center;
          background: rgba(0,0,0,0.2);
        }

        .panel-body {
          padding: 1.5rem;
          overflow-y: auto;
          height: calc(100% - 60px);
        }

        .meta-tags {
          display: flex;
          gap: 0.5rem;
        }

        .tag {
          font-size: 0.75rem;
          font-family: var(--font-mono);
          padding: 0.2rem 0.5rem;
          border-radius: 4px;
          background: rgba(255,255,255,0.05);
          color: var(--text-secondary);
        }

        .problem-content {
          font-size: 1rem;
          line-height: 1.7;
        }

        .section-title {
          margin-top: 2rem;
          font-size: 1rem;
          color: var(--accent-cyan);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .technical-text {
          font-family: var(--font-mono);
          background: rgba(0,0,0,0.3);
          padding: 1rem;
          border-radius: 4px;
          border-left: 2px solid var(--border-subtle);
          font-size: 0.9rem;
          color: #A3B3C8;
          white-space: pre-wrap;
        }

        .right-panel {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }

        .editor-container {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
        }

        .lang-select {
          width: auto;
          padding: 0.3rem 1rem;
          background: transparent;
          border: none;
          color: var(--text-primary);
          font-weight: bold;
          cursor: pointer;
        }

        .editor-wrapper {
          flex: 1;
        }

        .console-container {
          height: 35%;
          display: flex;
          flex-direction: column;
        }

        .console-header {
          padding: 0.5rem 1rem;
        }

        .tabs {
          display: flex;
          gap: 1.5rem;
        }

        .tab {
          font-size: 0.85rem;
          text-transform: uppercase;
          letter-spacing: 0.05em;
          color: var(--text-secondary);
          cursor: pointer;
          padding-bottom: 0.5rem;
        }

        .tab.active {
          color: var(--text-primary);
          border-bottom: 2px solid var(--accent-cyan);
        }

        .action-buttons {
          display: flex;
          gap: 0.5rem;
        }

        .sm-btn {
          padding: 0.4rem 0.8rem;
          font-size: 0.8rem;
          display: flex;
          align-items: center;
          gap: 0.4rem;
        }

        .console-body {
          flex: 1;
          display: flex;
          overflow: hidden;
        }

        .split-console {
          display: grid;
          grid-template-columns: 1fr 1fr;
          width: 100%;
          height: 100%;
        }

        .custom-input {
          border: none;
          border-right: 1px solid var(--border-subtle);
          border-radius: 0;
          resize: none;
          height: 100%;
          background: transparent;
        }

        .output-log {
          padding: 1rem;
          font-family: var(--font-mono);
          font-size: 0.85rem;
          color: #A3B3C8;
          overflow-y: auto;
          white-space: pre-wrap;
          background: rgba(0,0,0,0.4);
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; text-shadow: 0 0 20px var(--accent-cyan-glow); }
          50% { opacity: 0.5; text-shadow: none; }
        }
      `}</style>
    </div>
  );
}
