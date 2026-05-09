import axios from 'axios'
import { useAuthStore } from '../store/auth'
import { clearStoredSession, readStoredSession } from '../store/session'

const authFreeEndpoints = new Set([
  '/auth/login',
  '/auth/logout',
  '/user-auth/login',
  '/user-auth/register',
  '/user-auth/refresh',
  '/smart-login/sniff',
  '/smart-login/firebase',
  '/smart-login/firebase/refresh',
  '/smart-login/devin',
])

let redirectingToLogin = false

function isAuthFreeRequest(url = '') {
  return authFreeEndpoints.has(url)
}

function isAuthFailureResponse(response) {
  const code = response?.data?.code
  const message = response?.data?.msg
  const url = response?.config?.url || ''

  if (code !== 401 || isAuthFreeRequest(url)) {
    return false
  }

  return message === 'invalid token' || message === 'authorization required'
}

function redirectToLogin() {
  if (typeof window === 'undefined') {
    return
  }
  if (redirectingToLogin) {
    return
  }
  if (window.location.pathname === '/login') {
    return
  }
  redirectingToLogin = true
  window.location.replace('/login?reason=session-expired')
}

const client = axios.create({ baseURL: '/api', timeout: 30000 })

client.interceptors.request.use((config) => {
  const auth = useAuthStore()
  const token = auth.token || readStoredSession().token
  if (token && !isAuthFreeRequest(config.url)) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

client.interceptors.response.use((response) => {
  if (isAuthFailureResponse(response)) {
    clearStoredSession()
    redirectToLogin()
  }
  return response
})

export default client
