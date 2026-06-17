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

  async activate(id: string): Promise<void> {
    await client.post(`/profiles/${id}/activate`)
  },

  async preview(id: string): Promise<string> {
    const { data } = await client.get(`/profiles/${id}/preview`)
    return data
  },

  async export(id: string): Promise<Blob> {
    const { data } = await client.post(`/profiles/${id}/export`, null, {
      responseType: 'blob',
    })
    return data
  },

  async import(file: File, importSubscriptions?: boolean): Promise<any> {
    const formData = new FormData()
    formData.append('file', file)
    if (importSubscriptions !== undefined) {
      formData.append('importSubscriptions', String(importSubscriptions))
    }
    const { data } = await client.post('/profiles/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
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

  async getRules(id: string): Promise<string> {
    const { data } = await client.get(`/profiles/${id}/rules`)
    return data
  },

  async updateRules(id: string, content: string): Promise<void> {
    await client.put(`/profiles/${id}/rules`, { content })
  },

  async getOverride(id: string): Promise<string> {
    const { data } = await client.get(`/profiles/${id}/override`)
    return data
  },

  async updateOverride(id: string, content: string): Promise<void> {
    await client.put(`/profiles/${id}/override`, { content })
  },
}
