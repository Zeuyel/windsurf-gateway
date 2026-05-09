import { defineStore } from 'pinia'
import client from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({ user: null, token: '' }),
  getters: {
    isLoggedIn: (s) => !!s.token,
    isAdmin: (s) => s.user?.role === 'admin',
  },
  actions: {
    init() {
      if (!this.token) {
        this.token = localStorage.getItem('token') || ''
        const raw = localStorage.getItem('user')
        this.user = raw ? JSON.parse(raw) : null
      }
    },
    setSession(user, token = this.token) {
      this.user = user
      this.token = token
      localStorage.setItem('user', JSON.stringify(user))
      localStorage.setItem('token', token || '')
    },
    async login(username, password, admin = false) {
      const url = admin ? '/auth/login' : '/user-auth/login'
      const res = await client.post(url, { username, password })
      if (res.data.code !== 200) throw new Error(res.data.msg)
      this.setSession(res.data.data.user, res.data.data.token)
    },
    async fetchUser() {
      const endpoint = this.user?.role === 'admin' ? '/auth/me' : '/user/me'
      const res = await client.get(endpoint)
      if (res.data.code !== 200) throw new Error(res.data.msg || 'failed to fetch user')
      this.setSession(res.data.data, this.token)
      return this.user
    },
    logout() {
      this.user = null
      this.token = ''
      localStorage.clear()
    },
  },
})
