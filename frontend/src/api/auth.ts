import client from './client'

interface AuthResponse {
  token: string
  expiresIn: number
}

export const authApi = {
  async status(): Promise<{ configured: boolean }> {
    const { data } = await client.get('/auth/status')
    return data
  },

  async login(password: string): Promise<AuthResponse> {
    const { data } = await client.post('/auth/login', { password })
    return data
  },

  async setup(password: string): Promise<AuthResponse> {
    const { data } = await client.post('/auth/setup', { password })
    return data
  },

  async check(): Promise<void> {
    await client.get('/auth/check')
  },

  async changePassword(oldPassword: string, newPassword: string): Promise<void> {
    await client.put('/auth/password', { oldPassword, newPassword })
  },
}
