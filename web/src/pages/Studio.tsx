import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Send, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api, capabilityLabel } from '@/lib/api'
import type { Asset, Capability, ModelDef, Task } from '@/lib/types'
import FilePicker from '@/components/FilePicker'
import SchemaField, { defaultsFrom } from '@/components/SchemaField'
import MediaPreview from '@/components/MediaPreview'
import StatusBadge from '@/components/StatusBadge'

const promptLabel: Record<Capability, string> = {
  image: '画面描述（Prompt）',
  video: '视频描述（Prompt）',
  tts: '合成文本',
}

export default function Studio() {
  const [capability, setCapability] = useState<Capability>('image')
  const { data: models } = useQuery({
    queryKey: ['models', capability],
    queryFn: () => api.listModels(capability, true),
  })
  const { data: providers } = useQuery({ queryKey: ['providers'], queryFn: api.listProviders })
  const [providerID, setProviderID] = useState<number | undefined>()
  const [modelCode, setModelCode] = useState<string | null>(null)
  const [mode, setMode] = useState('')
  const [prompt, setPrompt] = useState('')
  const [params, setParams] = useState<Record<string, unknown>>({})
  const [inputAssets, setInputAssets] = useState<Record<string, Asset[]>>({})
  const [lastTask, setLastTask] = useState<Task | null>(null)

  const model: ModelDef | undefined = useMemo(
    () => models?.find((m) => m.code === modelCode) ?? models?.find((m) => m.is_default) ?? models?.[0],
    [models, modelCode],
  )
  const schema = model?.param_schema

  // 模型/能力切换时重置
  const switchModel = (code: string) => {
    setModelCode(code)
    const m = models?.find((x) => x.code === code)
    if (m) {
      setMode(m.param_schema.modes[0]?.key ?? '')
      setParams(defaultsFrom(m.param_schema.fields))
      setInputAssets({})
    }
  }
  const switchCapability = (cap: Capability) => {
    setCapability(cap)
    setModelCode(null)
    setMode('')
    setParams({})
    setInputAssets({})
    setPrompt('')
  }

  const effectiveMode = mode || schema?.modes[0]?.key || ''
  const visibleInputs = schema?.inputs?.filter((i) => !i.modes || i.modes.includes(effectiveMode)) ?? []

  const voicesQuery = useQuery({
    queryKey: ['voices', model?.code, providerID],
    queryFn: () => api.listVoices(model!.code, providerID),
    enabled: capability === 'tts' && !!model,
  })

  const create = useMutation({
    mutationFn: () => {
      const inputs: Record<string, number[]> = {}
      for (const [k, arr] of Object.entries(inputAssets)) inputs[k] = arr.map((a) => a.id)
      return api.createTask({
        provider_id: providerID,
        model_code: model!.code,
        mode: effectiveMode,
        prompt,
        params,
        inputs,
      })
    },
    onSuccess: (t) => {
      setLastTask(t)
      toast.success(`任务 #${t.id} 已提交`, { description: `${model?.name} · ${promptLabel[capability]}` })
    },
    onError: (e: Error) => toast.error('提交失败', { description: e.message }),
  })

  const submit = () => {
    if (!model) return toast.error('没有可用模型，请到「模型管理」启用')
    if (!prompt.trim()) return toast.error(`请填写${promptLabel[capability]}`)
    for (const spec of visibleInputs) {
      const n = inputAssets[spec.key]?.length ?? 0
      if (spec.required && n < (spec.min || 1)) return toast.error(`请上传${spec.label}`)
      if (spec.min && n > 0 && n < spec.min) return toast.error(`${spec.label}至少 ${spec.min} 个`)
    }
    create.mutate()
  }

  const reset = () => {
    setPrompt('')
    if (schema) setParams(defaultsFrom(schema.fields))
    setInputAssets({})
  }

  return (
    <div className="mx-auto max-w-6xl p-6">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-xl font-semibold">创作中心</h1>
        {providers && providers.length > 1 && (
          <Select value={String(providerID ?? '')} onValueChange={(v) => setProviderID(Number(v) || undefined)}>
            <SelectTrigger className="w-56">
              <SelectValue placeholder="默认服务配置" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">默认服务配置</SelectItem>
              {providers.map((p) => (
                <SelectItem key={p.id} value={String(p.id)}>
                  {p.name}
                  {p.is_default ? '（默认）' : ''}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </div>

      <Tabs value={capability} onValueChange={(v) => switchCapability(v as Capability)}>
        <TabsList className="mb-4">
          {(['image', 'video', 'tts'] as Capability[]).map((c) => (
            <TabsTrigger key={c} value={c}>
              {capabilityLabel[c]}创作
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      <div className="grid gap-4 lg:grid-cols-[1fr_380px]">
        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">模型与模式</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <Select value={model?.code} onValueChange={switchModel}>
                <SelectTrigger>
                  <SelectValue placeholder="选择模型" />
                </SelectTrigger>
                <SelectContent>
                  {models?.map((m) => (
                    <SelectItem key={m.code} value={m.code}>
                      {m.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {schema && schema.modes.length > 1 && (
                <Tabs value={effectiveMode} onValueChange={setMode}>
                  <TabsList>
                    {schema.modes.map((m) => (
                      <TabsTrigger key={m.key} value={m.key}>
                        {m.label}
                      </TabsTrigger>
                    ))}
                  </TabsList>
                </Tabs>
              )}
              {model?.remark && <p className="text-xs text-muted-foreground">{model.remark}</p>}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">{promptLabel[capability]}</CardTitle>
            </CardHeader>
            <CardContent>
              <Textarea
                rows={capability === 'tts' ? 6 : 5}
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder={
                  capability === 'image'
                    ? '描述想要的画面，例如：一只穿宇航服的猫站在火星上，电影级光感，细节丰富'
                    : capability === 'video'
                      ? '描述视频内容与镜头，例如：雨夜的东京街头，霓虹灯闪烁，镜头缓缓推进'
                      : '输入要合成的文本…'
                }
              />
            </CardContent>
          </Card>

          {visibleInputs.length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">输入素材</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {visibleInputs.map((spec) => (
                  <div key={spec.key} className="space-y-1.5">
                    <Label className="text-xs font-medium text-muted-foreground">{spec.label}</Label>
                    <FilePicker
                      spec={spec}
                      assets={inputAssets[spec.key] ?? []}
                      onChange={(arr) => setInputAssets((prev) => ({ ...prev, [spec.key]: arr }))}
                    />
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          {schema && schema.fields.length > 0 && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm">参数设置</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 sm:grid-cols-2">
                {schema.fields.map((f) => (
                  <div key={f.key} className={f.type === 'textarea' || f.type === 'boolean' ? 'sm:col-span-2' : ''}>
                    <SchemaField
                      field={f}
                      value={params[f.key]}
                      onChange={(v) => setParams((prev) => ({ ...prev, [f.key]: v }))}
                      voices={voicesQuery.data?.voices.map((x) => x.id)}
                    />
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          <div className="flex gap-2">
            <Button onClick={submit} disabled={create.isPending}>
              {create.isPending ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
              提交创作
            </Button>
            <Button variant="outline" onClick={reset}>
              <RotateCcw className="size-4" /> 重置
            </Button>
          </div>
        </div>

        <div className="space-y-4">
          <RecentTaskCard task={lastTask} />
        </div>
      </div>
    </div>
  )
}

// 右侧最近任务卡片（提交后轮询状态）
function RecentTaskCard({ task }: { task: Task | null }) {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { data: fresh } = useQuery({
    queryKey: ['task', task?.id],
    queryFn: () => api.getTask(task!.id),
    enabled: !!task,
    refetchInterval: (q) => {
      const t = q.state.data
      if (t && ['queued', 'submitting', 'running'].includes(t.status)) return 3000
      return false
    },
  })
  if (!task) {
    return (
      <Card>
        <CardContent className="pt-6 text-sm text-muted-foreground">
          提交创作后，这里会显示任务进度与结果预览。
        </CardContent>
      </Card>
    )
  }
  const t = fresh ?? task
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center justify-between text-sm">
          <span>任务 #{t.id}</span>
          <StatusBadge status={t.status} />
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="text-xs text-muted-foreground">
          {t.model_code} · {t.mode}
        </div>
        {['queued', 'submitting', 'running'].includes(t.status) && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" />
            {t.capability === 'video' ? '视频生成通常需要 1-5 分钟…' : '生成中…'}
          </div>
        )}
        {t.status === 'failed' && (
          <div className="rounded-md bg-destructive/10 p-2 text-xs text-destructive">
            {t.error_code}: {t.error_message}
          </div>
        )}
        {t.outputs && t.outputs.length > 0 && (
          <div className={t.outputs[0].kind === 'image' ? 'grid grid-cols-2 gap-2' : 'space-y-2'}>
            {t.outputs.map((a) => (
              <MediaPreview key={a.id} asset={a} className={a.kind === 'image' ? 'h-36 w-full rounded-md' : 'w-full rounded-md'} />
            ))}
          </div>
        )}
        <Button
          variant="outline"
          size="sm"
          className="w-full"
          onClick={() => {
            qc.invalidateQueries({ queryKey: ['tasks'] })
            navigate('/tasks')
          }}
        >
          查看任务列表
        </Button>
      </CardContent>
    </Card>
  )
}
