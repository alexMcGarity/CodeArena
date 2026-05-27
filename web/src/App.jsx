import { BrowserRouter, Routes, Route, Link, useNavigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import useAuthStore from './store/auth'
import Problems from './pages/Problems'
import Login from './pages/Login'
import Register from './pages/Register'
import Profile from './pages/Profile'

const queryClient = new QueryClient()

function Nav() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  return (
    <nav className="top-nav">
      <Link to="/" className="nav-brand">CodeArena</Link>
      <div className="nav-links">
        {user ? (
          <>
            <Link to="/profile">{user.email}</Link>
            <button
              className="nav-btn"
              onClick={() => { logout(); navigate('/login') }}
            >
              Log out
            </button>
          </>
        ) : (
          <>
            <Link to="/login">Log in</Link>
            <Link to="/register">Register</Link>
          </>
        )}
      </div>
    </nav>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Nav />
        <Routes>
          <Route path="/" element={<Problems />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/profile" element={<Profile />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
