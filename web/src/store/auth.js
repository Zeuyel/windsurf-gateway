import { defineStore } from 'pinia'
import client from '../api/client'
import { clearStoredSession, readStoredSession, writeStoredSession } from './session'

export const useAuthStore = defineStore('auth', {
  state: () => ({ user: null, token: '' }),
  getters: {
    isLoggedIn: (s) => !!s.token,
    isAdmin: (s) => s.user?.role === 'admin',
  },
  actions: {
    init() {
      if (!this.token) {
        const session = readStoredSession()
        this.token = session.token
        this.user = session.user
      }
    },
    setSession(user, token = this.token) {
      this.user = user
      this.token = token
      writeStoredSession(user, token)
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
      clearStoredSession()
    },
  },
})
