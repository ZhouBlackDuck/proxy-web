import client from './client'

export interface Profile {
  id: string
  name: string
  description?: string
  subscriptionName?: string
  createdAt: string
  updatedAt: string
  exportSettings: {
    includeSubscriptions: boolean
  }
}

export interface ProfileRegistry {
  activeProfileId: string
  profiles: Profile[]
}

export const profileApi = {
  async list(): Promise<ProfileRegistry> {
    const { data } = await client.get('/profiles')
    return data
  },

  async create(req: { name: string; description?: string; subscriptionName?: string }): Promise<Profile> {
    const { data } = await client.post('/profiles', req)
    return data
  },

  async get(id: string): Promise<Profile> {
    const { data } = await client.get(`/profiles/${id}`)
    return data
  },

  async update(id: string, patch: Partial<Profile>): Promise<void> {
    await client.put(`/profiles/${id}`, patch)
  },

  async delete(id: string): Promise<void> {
    await client.delete(`/profiles/${id}`)
  },

  async validateConfig(content: string): Promise<{ valid: boolean; errors: string[] }> {
    const { data } = await client.post('/config/validate', { content })
    return data
  },

  async getPorts(): Promise<any> {
    const { data } = await client.get('/config/ports')
    return data
  },

  async updatePorts(ports: any): Promise<void> {
    await client.put('/config/ports', ports)
  },
}
