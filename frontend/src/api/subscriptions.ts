import client from './client'

export interface Subscription {
  name: string
  displayName?: string
  url?: string
  source?: string
  content?: string
  ua?: string
  process?: any[]
  updatedAt?: number
}

export const subscriptionApi = {
  async list(): Promise<Subscription[]> {
    const { data } = await client.get('/subscriptions')
    return data.data || []
  },

  async get(name: string): Promise<Subscription> {
    const { data } = await client.get(`/subscriptions/${encodeURIComponent(name)}`)
    return data.data
  },

  async create(sub: Partial<Subscription>): Promise<void> {
    await client.post('/subscriptions', sub)
  },

  async update(name: string, patch: Record<string, any>): Promise<void> {
    await client.put(`/subscriptions/${encodeURIComponent(name)}`, patch)
  },

  async delete(name: string): Promise<void> {
    await client.delete(`/subscriptions/${encodeURIComponent(name)}`)
  },

  async sync(name: string): Promise<void> {
    await client.post(`/subscriptions/${encodeURIComponent(name)}/sync`)
  },

  async download(name: string, target = 'ClashMeta'): Promise<string> {
    const { data } = await client.get(`/subscriptions/${encodeURIComponent(name)}/download`, {
      params: { target },
    })
    return data
  },

  async getFlow(name: string): Promise<any> {
    const { data } = await client.get(`/subscriptions/${encodeURIComponent(name)}/flow`)
    return data
  },
}
