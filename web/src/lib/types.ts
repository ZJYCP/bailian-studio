// 与后端 internal/model 对应的类型定义

export type Capability = 'image' | 'video' | 'tts'

export interface Provider {
  id: number
  name: string
  base_url: string
  key_mask: string
  remark: string
  is_default: boolean
}

export interface FieldSchema {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'boolean' | 'select' | 'voice'
  options?: string[]
  option_labels?: Record<string, string>
  allowCustom?: boolean
  default?: unknown
  min?: number
  max?: number
  step?: number
  required?: boolean
  help?: string
  placeholder?: string
}

export interface InputSpec {
  key: string
  label: string
  type: 'image' | 'video' | 'audio' | 'media'
  modes?: string[]
  min?: number
  max?: number
  required?: boolean
  help?: string
}

export interface Mode {
  key: string
  label: string
}

export interface ParamSchema {
  modes: Mode[]
  inputs?: InputSpec[]
  fields: FieldSchema[]
}

export interface ModelDef {
  id: number
  code: string
  name: string
  capability: Capability
  protocol: string
  param_schema: ParamSchema
  enabled: boolean
  is_default: boolean
  sort: number
  remark: string
}

export interface Asset {
  id: number
  task_id: number | null
  kind: 'image' | 'video' | 'audio'
  role: 'input' | 'output'
  file_path: string
  file_name: string
  mime: string
  size: number
  width: number
  height: number
  duration: number
  source_url: string
  created_at: string
}

export interface Task {
  id: number
  provider_id: number
  provider_name: string
  model_code: string
  capability: Capability
  mode: string
  status: 'queued' | 'submitting' | 'running' | 'succeeded' | 'failed' | 'canceled'
  prompt: string
  params: Record<string, unknown>
  input_assets: { asset_id: number; role: string }[] | null
  external_task_id: string
  request_id: string
  error_code: string
  error_message: string
  usage: Record<string, unknown> | null
  result_urls: string[] | null
  submitted_at: string | null
  finished_at: string | null
  created_at: string
  outputs?: Asset[]
  inputs?: Asset[]
}

export interface Voice {
  id: string
  source: 'system' | 'custom'
  status?: string
}

export interface Paged<T> {
  total: number
  items: T[]
  page: number
  page_size: number
}
