import type { Asset, Capability, ModelDef, Paged, Provider, Task, Voice } from './types'

const BASE = ''

class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(BASE + path, {
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })
  if (!resp.ok) {
    let msg = `HTTP ${resp.status}`
    try {
      const data = await resp.json()
      if (data.error) msg = data.error
    } catch {
      /* ignore */
    }
    throw new ApiError(resp.status, msg)
  }
  if (resp.status === 204) return undefined as T
  return resp.json() as Promise<T>
}

export const api = {
  // providers
  listProviders: () => request<Provider[]>('/api/providers'),
  createProvider: (p: Partial<Provider> & { api_key?: string }) =>
    request<Provider>('/api/providers', { method: 'POST', body: JSON.stringify(p) }),
  updateProvider: (id: number, p: Partial<Provider> & { api_key?: string }) =>
    request<Provider>(`/api/providers/${id}`, { method: 'PUT', body: JSON.stringify(p) }),
  deleteProvider: (id: number) => request<{ ok: boolean }>(`/api/providers/${id}`, { method: 'DELETE' }),
  testProvider: (id: number) => request<{ ok: boolean; error?: string }>(`/api/providers/${id}/test`, { method: 'POST' }),

  // models
  listModels: (capability?: Capability, enabledOnly?: boolean) => {
    const q = new URLSearchParams()
    if (capability) q.set('capability', capability)
    if (enabledOnly) q.set('enabled', 'true')
    return request<ModelDef[]>(`/api/models?${q}`)
  },
  updateModel: (id: number, m: Partial<ModelDef>) =>
    request<ModelDef>(`/api/models/${id}`, { method: 'PUT', body: JSON.stringify(m) }),
  createModel: (m: Partial<ModelDef>) =>
    request<ModelDef>('/api/models', { method: 'POST', body: JSON.stringify(m) }),
  deleteModel: (id: number) => request<{ ok: boolean }>(`/api/models/${id}`, { method: 'DELETE' }),

  // tasks
  createTask: (t: {
    provider_id?: number
    model_code: string
    mode: string
    prompt: string
    params: Record<string, unknown>
    inputs: Record<string, number[]>
  }) => request<Task>('/api/tasks', { method: 'POST', body: JSON.stringify(t) }),
  listTasks: (q: Record<string, string | number | undefined>) => {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) if (v !== undefined && v !== '') qs.set(k, String(v))
    return request<Paged<Task>>(`/api/tasks?${qs}`)
  },
  getTask: (id: number) => request<Task>(`/api/tasks/${id}`),
  retryTask: (id: number) => request<Task>(`/api/tasks/${id}/retry`, { method: 'POST' }),
  cancelTask: (id: number) => request<Task>(`/api/tasks/${id}/cancel`, { method: 'POST' }),
  deleteTask: (id: number) => request<{ ok: boolean }>(`/api/tasks/${id}`, { method: 'DELETE' }),

  // assets
  listAssets: (q: Record<string, string | number | undefined>) => {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) if (v !== undefined && v !== '') qs.set(k, String(v))
    return request<Paged<Asset>>(`/api/assets?${qs}`)
  },
  deleteAsset: (id: number) => request<{ ok: boolean }>(`/api/assets/${id}`, { method: 'DELETE' }),
  assetURL: (id: number) => `${BASE}/api/assets/${id}/file`,
  assetDownloadURL: (id: number) => `${BASE}/api/assets/${id}/download`,

  // uploads & voices
  upload: async (file: File): Promise<Asset> => {
    const fd = new FormData()
    fd.append('file', file)
    const resp = await fetch(BASE + '/api/uploads', { method: 'POST', body: fd })
    if (!resp.ok) {
      const data = await resp.json().catch(() => ({ error: `HTTP ${resp.status}` }))
      throw new Error(data.error || '上传失败')
    }
    return resp.json()
  },
  listVoices: (model: string, providerID?: number) => {
    const q = new URLSearchParams({ model })
    if (providerID) q.set('provider_id', String(providerID))
    return request<{ voices: Voice[] }>(`/api/voices?${q}`)
  },
  createVoiceClone: (req: {
    provider_id?: number
    target_model: string
    prefix: string
    audio_asset_id: number
    language?: string
  }) => request<{ voice_id: string }>('/api/voice-clones', { method: 'POST', body: JSON.stringify(req) }),

  // 访问口令
  authStatus: () => request<{ auth_required: boolean; authenticated: boolean }>('/api/auth/status'),
  login: (token: string) =>
    request<{ ok: boolean }>('/api/auth', { method: 'POST', body: JSON.stringify({ token }) }),
  logout: () => request<{ ok: boolean }>('/api/auth/logout', { method: 'POST' }),
}

export const capabilityLabel: Record<Capability, string> = {
  image: '图像',
  video: '视频',
  tts: '语音',
}

export function fmtSize(bytes: number): string {
  if (bytes >= 1 << 30) return (bytes / (1 << 30)).toFixed(1) + ' GB'
  if (bytes >= 1 << 20) return (bytes / (1 << 20)).toFixed(1) + ' MB'
  if (bytes >= 1 << 10) return (bytes / (1 << 10)).toFixed(0) + ' KB'
  return bytes + ' B'
}

export function fmtTime(s: string | null | undefined): string {
  if (!s) return '-'
  const d = new Date(s)
  return d.toLocaleString('zh-CN', { hour12: false })
}
