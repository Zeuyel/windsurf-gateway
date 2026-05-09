import axios from 'axios'
import { useAuthStore } from '../store/auth'
const client = axios.create({ baseURL: '/api', timeout: 30000 })
client.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) config.headers.Authorization = `Bearer ${auth.token}`
  return config
})
export default client
