import { create } from 'zustand'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const useAuthStore = create((set, get) => ({
  token: localStorage.getItem('token'),
  user: JSON.parse(localStorage.getItem('user') || 'null'),

  register: async (email, password) => {
    const res = await fetch(`${API}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text.trim() || 'Registration failed')
    }
    const data = await res.json()
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify({ id: data.user_id, email: data.email }))
    set({ token: data.token, user: { id: data.user_id, email: data.email } })
  },

  login: async (email, password) => {
    const res = await fetch(`${API}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text.trim() || 'Login failed')
    }
    const data = await res.json()
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify({ id: data.user_id, email: data.email }))
    set({ token: data.token, user: { id: data.user_id, email: data.email } })
  },

  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    set({ token: null, user: null })
  },

  authHeader: () => {
    const { token } = get()
    return token ? { Authorization: `Bearer ${token}` } : {}
  },
}))

export default useAuthStore
