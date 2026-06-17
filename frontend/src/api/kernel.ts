import client from './client'

export interface KernelVersion {
  version: string
  meta: boolean
  premium: boolean
}

export interface KernelConfig {
  port: number
  'socks-port': number
  'redir-port': number
  'tproxy-port': number
  'mixed-port': number
  'allow-lan': boolean
  'bind-address': string
  mode: string
  'log-level': string
  ipv6: boolean
  sniffing: boolean
  tun: {
    enable: boolean
    stack: string
    'auto-route': boolean
  }
}

export interface ProxyNode {
  name: string
  type: string
  history: { time: string; delay: number }[]
  now?: string
  all?: string[]
}

export interface ProxiesResponse {
  proxies: Record<string, ProxyNode>
}

export interface Rule {
  index: number
  type: string
  payload: string
  proxy: string
  size: number
  extra?: { disabled: boolean; hitCount: number }
}

export interface ConnectionMeta {
  network: string
  type: string
  sourceIP: string
  destinationIP: string
  sourcePort: string
  destinationPort: string
  host: string
  dnsMode: string
  processPath: string
}

export interface Connection {
  id: string
  metadata: ConnectionMeta
  upload: number
  download: number
  start: string
  chains: string[]
  rule: string
  rulePayload: string
}

export interface ConnectionsSnapshot {
  downloadTotal: number
  uploadTotal: number
  connections: Connection[]
}

export const kernelApi = {
  // Version
  async getVersion(): Promise<KernelVersion> {
    const { data } = await client.get('/kernel/version')
    return data
  },

  // Config
  async getConfigs(): Promise<KernelConfig> {
    const { data } = await client.get('/kernel/configs')
    return data
  },

  async patchConfig(patch: Record<string, any>): Promise<void> {
    await client.patch('/kernel/configs', patch)
  },

  async putConfig(payload: string): Promise<void> {
    await client.put('/kernel/configs', { payload })
  },

  // Proxies
  async getProxies(): Promise<ProxiesResponse> {
    const { data } = await client.get('/kernel/proxies')
    return data
  },

  async switchProxy(groupName: string, nodeName: string): Promise<void> {
    await client.put(`/kernel/proxies/${encodeURIComponent(groupName)}`, { name: nodeName })
  },

  async testDelay(name: string, url?: string, timeout?: number): Promise<Record<string, number>> {
    const { data } = await client.get(`/kernel/proxies/${encodeURIComponent(name)}/delay`, {
      params: { url: url || 'https://www.gstatic.com/generate_204', timeout: timeout || 5000 },
    })
    return data
  },

  // Groups
  async getGroups(): Promise<ProxiesResponse> {
    const { data } = await client.get('/kernel/group')
    return data
  },

  async testGroupDelay(name: string, url?: string, timeout?: number): Promise<Record<string, number>> {
    const { data } = await client.get(`/kernel/group/${encodeURIComponent(name)}/delay`, {
      params: { url: url || 'https://www.gstatic.com/generate_204', timeout: timeout || 5000 },
    })
    return data
  },

  // Rules
  async getRules(): Promise<{ rules: Rule[] }> {
    const { data } = await client.get('/kernel/rules')
    return data
  },

  // Connections
  async getConnections(): Promise<ConnectionsSnapshot> {
    const { data } = await client.get('/kernel/connections')
    return data
  },

  async closeAllConnections(): Promise<void> {
    await client.delete('/kernel/connections')
  },

  async closeConnection(id: string): Promise<void> {
    await client.delete(`/kernel/connections/${id}`)
  },

  // Restart
  async restart(): Promise<void> {
    await client.post('/kernel/restart')
  },

  // GeoIP
  async getGeoStatus(): Promise<any> {
    const { data } = await client.get('/geo/status')
    return data
  },

  async updateGeo(): Promise<void> {
    await client.post('/geo/update')
  },
}
