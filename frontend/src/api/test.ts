import client from './client'

export interface TestSite {
  name: string
  url: string
  icon: string
}

export interface TestResult {
  name: string
  url: string
  icon: string
  ok: boolean
  latency: number
  error?: string
}

export const testApi = {
  async testAll(): Promise<TestResult[]> {
    const { data } = await client.get('/test')
    return data.results
  },

  async testSingle(url: string): Promise<TestResult> {
    const { data } = await client.post('/test', { url })
    return data
  },
}
