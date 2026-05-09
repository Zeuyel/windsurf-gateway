const USER_KEY = 'user'
const TOKEN_KEY = 'token'

export function readStoredSession() {
  const token = localStorage.getItem(TOKEN_KEY) || ''
  const rawUser = localStorage.getItem(USER_KEY)

  if (!rawUser) {
    return { user: null, token }
  }

  try {
    return { user: JSON.parse(rawUser), token }
  } catch (error) {
    clearStoredSession()
    return { user: null, token: '' }
  }
}

export function writeStoredSession(user, token) {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
  localStorage.setItem(TOKEN_KEY, token || '')
}

export function clearStoredSession() {
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem(TOKEN_KEY)
}
