import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Editor from '@monaco-editor/react'
import useAuthStore from '../store/auth'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const WS_BASE = import.meta.env.VITE_WS_URL || 'ws://localhost:8080'

const DEFAULT_CODE = {
  cpp: `#include <bits/stdc++.h>
using namespace std;

int main() {
  cout << "Hello World\\n";
  return 0;
}
`,
  python: `# Read input with input(), print output with print()
print("Hello World")
`,
}

const MONACO_LANG = { cpp: 'cpp', python: 'python' }

export default function Problems() {
  const [problems, setProblems] = useState([])
  const [selectedProblem, setSelectedProblem] = useState(null)
  const [language, setLanguage] = useState('cpp')
  const [code, setCode] = useState(DEFAULT_CODE.cpp)
  const [verdict, setVerdict] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [wsStatus, setWsStatus] = useState(null)

  const token = useAuthStore((s) => s.token)
  const authHeader = useAuthStore((s) => s.authHeader)
  const navigate = useNavigate()

  useEffect(() => {
    fetch(`${API}/problems`)
      .then((r) => r.json())
      .then(setProblems)
      .catch(console.error)
  }, [])

  useEffect(() => {
    if (selectedProblem) {
      setVerdict(null)
      setWsStatus(null)
    }
  }, [selectedProblem])

  const handleLanguageChange = (e) => {
    const lang = e.target.value
    setLanguage(lang)
    setCode(DEFAULT_CODE[lang])
  }

  const submitSolution = async () => {
    if (!selectedProblem) return
    if (!token) { navigate('/login'); return }

    setSubmitting(true)
    setVerdict(null)
    setWsStatus('submitting…')

    try {
      const res = await fetch(`${API}/submissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify({ problem_id: selectedProblem.id, code, language }),
      })

      if (res.status === 401) { navigate('/login'); return }
      if (!res.ok) { setWsStatus('submission error'); return }

      const { submission_id } = await res.json()
      setWsStatus('judging…')
      openResultSocket(submission_id)
    } catch (err) {
      console.error(err)
      setWsStatus('network error')
    } finally {
      setSubmitting(false)
    }
  }

  const openResultSocket = (submissionID) => {
    const ws = new WebSocket(`${WS_BASE}/submissions/${submissionID}/live`)
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data)
      setVerdict(data.verdict)
      setWsStatus(null)
      ws.close()
    }
    ws.onerror = () => { setWsStatus('connection error'); ws.close() }
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>CodeArena</h1>
        <nav>
          {problems.map((p) => (
            <button
              key={p.id}
              onClick={() => setSelectedProblem(p)}
              className={selectedProblem?.id === p.id ? 'active' : ''}
            >
              <span className={`difficulty-dot difficulty-${p.difficulty?.toLowerCase()}`} />
              {p.title}
            </button>
          ))}
        </nav>
      </aside>

      <main className="main-content">
        {selectedProblem ? (
          <>
            <div className="problem-header">
              <h2>{selectedProblem.title}</h2>
              <span className={`difficulty-badge difficulty-${selectedProblem.difficulty?.toLowerCase()}`}>
                {selectedProblem.difficulty}
              </span>
            </div>
            <p className="problem-description">{selectedProblem.description}</p>

            <div className="editor-container">
              <div className="editor-toolbar">
                <label>
                  Language:
                  <select value={language} onChange={handleLanguageChange}>
                    <option value="cpp">C++</option>
                    <option value="python">Python 3</option>
                  </select>
                </label>
                <button className="submit-btn" onClick={submitSolution} disabled={submitting}>
                  {submitting ? 'Submitting…' : 'Submit'}
                </button>
              </div>

              <Editor
                height="400px"
                language={MONACO_LANG[language]}
                theme="vs-dark"
                value={code}
                onChange={(val) => setCode(val ?? '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                }}
              />
            </div>

            {(wsStatus || verdict) && (
              <div className={`submission-result ${verdict ? `result-${verdict}` : ''}`}>
                {wsStatus && <p className="status-msg">{wsStatus}</p>}
                {verdict && <p className="verdict-msg">Verdict: <strong>{verdict}</strong></p>}
              </div>
            )}
          </>
        ) : (
          <div className="welcome-card">
            <h2>Welcome to CodeArena</h2>
            <p>Select a problem from the sidebar to begin.</p>
          </div>
        )}
      </main>
    </div>
  )
}
