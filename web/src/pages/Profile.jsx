import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import useAuthStore from '../store/auth'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

function fetchMySubmissions(authHeader) {
  return fetch(`${API}/users/me/submissions`, { headers: authHeader }).then((r) => {
    if (!r.ok) throw new Error('Failed to load submissions')
    return r.json()
  })
}

export default function Profile() {
  const user = useAuthStore((s) => s.user)
  const authHeader = useAuthStore((s) => s.authHeader)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  const { data: submissions, isLoading, error } = useQuery({
    queryKey: ['my-submissions'],
    queryFn: () => fetchMySubmissions(authHeader()),
    enabled: !!user,
  })

  if (!user) {
    navigate('/login')
    return null
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>CodeArena</h1>
        <nav>
          <button onClick={() => navigate('/')}>Problems</button>
        </nav>
      </aside>

      <main className="main-content">
        <div className="profile-header">
          <h2>{user.email}</h2>
          <button onClick={() => { logout(); navigate('/login') }}>Log out</button>
        </div>

        <h3>Submission History</h3>

        {isLoading && <p>Loading…</p>}
        {error && <p className="auth-error">Failed to load submissions.</p>}
        {submissions && submissions.length === 0 && <p>No submissions yet.</p>}
        {submissions && submissions.length > 0 && (
          <table className="submission-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Problem</th>
                <th>Language</th>
                <th>Verdict</th>
                <th>Submitted</th>
              </tr>
            </thead>
            <tbody>
              {submissions.map((s) => (
                <tr key={s.submission_id}>
                  <td>{s.submission_id}</td>
                  <td>{s.problem_title}</td>
                  <td>{s.language}</td>
                  <td className={`verdict-${s.verdict}`}>{s.verdict}</td>
                  <td>{new Date(s.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </main>
    </div>
  )
}
