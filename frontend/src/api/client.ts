import axios from 'axios'
import router from '../router'

const client = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: add auth token
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401
client.interceptors.response.use(
  (response) => response,
  (error) => {
    // Don't redirect for password change endpoint
    if (error.response?.status === 401 && !error.config?.url?.includes('/auth/password')) {
      localStorage.removeItem('token')
      router.push('/login')
    }
    return Promise.reject(error)
  }
)

export default client
