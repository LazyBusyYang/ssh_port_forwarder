import { defineStore } from 'pinia'
import api from '../api'

interface AuthState {
  token: string | null
  refreshToken: string | null
  username: string | null
  role: string | null
  expiresAt: number | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: localStorage.getItem('spf_token'),
    refreshToken: localStorage.getItem('spf_refresh_token'),
    username: localStorage.getItem('spf_username'),
    role: localStorage.getItem('spf_role'),
    expiresAt: null,
  }),

  getters: {
    isAuthenticated: (state) => !!state.token,
    isAdmin: (state) => state.role === 'admin',
  },

  actions: {
    async login(username: string, password: string) {
      const res = await api.post('/auth/login', { username, password })
      const data = res.data.data
      this.token = data.token
      this.refreshToken = data.refresh_token
      this.expiresAt = data.expires_at

      // 解析 JWT 获取用户信息
      const payload = JSON.parse(atob(data.token.split('.')[1]))
      this.username = payload.username
      this.role = payload.role

      // 持久化
      localStorage.setItem('spf_token', data.token)
      localStorage.setItem('spf_refresh_token', data.refresh_token)
      localStorage.setItem('spf_username', payload.username)
      localStorage.setItem('spf_role', payload.role)
    },

    logout() {
      this.token = null
      this.refreshToken = null
      this.username = null
      this.role = null
      this.expiresAt = null
      localStorage.removeItem('spf_token')
      localStorage.removeItem('spf_refresh_token')
      localStorage.removeItem('spf_username')
      localStorage.removeItem('spf_role')
    },

    async refresh() {
      if (!this.refreshToken) return
      try {
        const res = await api.post('/auth/refresh', { refresh_token: this.refreshToken })
        const data = res.data.data
        this.token = data.token
        localStorage.setItem('spf_token', data.token)
      } catch {
        this.logout()
      }
    },
  },
})
