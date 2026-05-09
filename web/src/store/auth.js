import { defineStore } from 'pinia'
import client from '../api/client'
export const useAuthStore = defineStore('auth', {
  state: () => ({ user: null, token: '' }),
  getters: {
    isLoggedIn: (s) => !!s.token,
    isAdmin: (s) => s.user?.role === 'admin'
  },
  actions: {
    init() {
      if (!this.token) {
        this.token = localStorage.getItem('token') || ''
        const raw = localStorage.getItem('user')
        this.user = raw ? JSON.parse(raw) : null
      }
    },
    async login(username, password, admin = false) {
      const url = admin ? '/auth/login' : '/user-auth/login'
      const res = await client.post(url, { username, password })
      if (res.data.code !== 200) throw new Error(res.data.msg)
      this.user = res.data.data.user
      this.token = res.data.data.token
      localStorage.setItem('user', JSON.stringify(this.user))
      localStorage.setItem('token', this.token)
    },
    logout() { this.user = null; this.token = ''; localStorage.clear() }
  }
})
