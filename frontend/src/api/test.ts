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
  async testAll(sites?: TestSite[]): Promise<TestResult[]> {
    const body = sites && sites.length > 0 ? { sites } : undefined
    const { data } = await client.post('/test', body || {})
    return data.results
  },

  async testSingle(url: string): Promise<TestResult> {
    const { data } = await client.post('/test/single', { url })
    return data
  },
}
