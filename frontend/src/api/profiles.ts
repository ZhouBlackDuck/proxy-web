import client from './client'

export const profileApi = {
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
