import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Loader2, Mic, Pencil, Plus, Trash2, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import FilePicker from '@/components/FilePicker'
import { LogoutButton } from '@/components/AuthGate'
import { api } from '@/lib/api'
import type { Asset, ModelDef, Provider } from '@/lib/types'

export default function Settings() {
  const qc = useQueryClient()
  const { data: providers } = useQuery({ queryKey: ['providers'], queryFn: api.listProviders })
  const [editing, setEditing] = useState<Provider | 'new' | null>(null)

  const remove = useMutation({
    mutationFn: (id: number) => api.deleteProvider(id),
    onSuccess: () => {
      toast.success('已删除')
      qc.invalidateQueries({ queryKey: ['providers'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h1 className="text-xl font-semibold">服务配置</h1>
          <Button size="sm" onClick={() => setEditing('new')}>
            <Plus className="size-4" /> 新增
          </Button>
        </div>
        <div className="space-y-2">
          {providers?.map((p) => (
            <Card key={p.id}>
              <CardContent className="flex items-center gap-3 py-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 text-sm font-medium">
                    {p.name}
                    {p.is_default && <span className="rounded bg-emerald-100 px-1.5 py-0.5 text-xs text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300">默认</span>}
                  </div>
                  <div className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                    {p.base_url} · {p.key_mask}
                  </div>
                </div>
                <ProviderTestButton id={p.id} />
                <Button variant="ghost" size="sm" onClick={() => setEditing(p)}>
                  <Pencil className="size-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-muted-foreground hover:text-destructive"
                  onClick={() => confirm(`删除服务配置「${p.name}」？`) && remove.mutate(p.id)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </CardContent>
            </Card>
          ))}
          {providers?.length === 0 && (
            <Card>
              <CardContent className="pt-6 text-sm text-muted-foreground">
                还没有配置。点击「新增」填入百炼 apiUrl 与 API Key（北京：
                <span className="font-mono"> https://dashscope.aliyuncs.com</span>；新加坡：
                <span className="font-mono"> https://dashscope-intl.aliyuncs.com</span>）。
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      <VoiceCloneSection />

      <AuthLockSection />

      <ProviderDialog
        provider={editing}
        onClose={() => setEditing(null)}
        onSaved={() => {
          setEditing(null)
          qc.invalidateQueries({ queryKey: ['providers'] })
        }}
      />
    </div>
  )
}

function ProviderTestButton({ id }: { id: number }) {
  const test = useMutation({
    mutationFn: () => api.testProvider(id),
    onSuccess: (r) => {
      if (r.ok) toast.success('连接成功')
      else toast.error('连接失败', { description: r.error })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  return (
    <Button variant="outline" size="sm" disabled={test.isPending} onClick={() => test.mutate()}>
      {test.isPending ? <Loader2 className="size-4 animate-spin" /> : test.data?.ok ? <CheckCircle2 className="size-4 text-emerald-600" /> : <XCircle className="size-4" />}
      测试
    </Button>
  )
}

function ProviderDialog({
  provider,
  onClose,
  onSaved,
}: {
  provider: Provider | 'new' | null
  onClose: () => void
  onSaved: () => void
}) {
  const isNew = provider === 'new'
  const [name, setName] = useState('')
  const [baseURL, setBaseURL] = useState('https://dashscope.aliyuncs.com')
  const [apiKey, setApiKey] = useState('')
  const [remark, setRemark] = useState('')
  const [isDefault, setIsDefault] = useState(false)
  const initialized = useRef<string | null>(null)

  if (provider && initialized.current !== (isNew ? 'new' : String((provider as Provider).id))) {
    initialized.current = isNew ? 'new' : String((provider as Provider).id)
    if (!isNew) {
      const p = provider as Provider
      setName(p.name)
      setBaseURL(p.base_url)
      setApiKey('')
      setRemark(p.remark)
      setIsDefault(p.is_default)
    } else {
      setName('')
      setBaseURL('https://dashscope.aliyuncs.com')
      setApiKey('')
      setRemark('')
      setIsDefault(false)
    }
  }

  const save = useMutation({
    mutationFn: () => {
      const body = { name, base_url: baseURL, api_key: apiKey, remark, is_default: isDefault }
      return isNew ? api.createProvider(body) : api.updateProvider((provider as Provider).id, body)
    },
    onSuccess: () => {
      toast.success('已保存')
      onSaved()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <Dialog open={!!provider} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isNew ? '新增服务配置' : '编辑服务配置'}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label>名称</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如：百炼-北京" />
          </div>
          <div className="space-y-1.5">
            <Label>API URL</Label>
            <Input value={baseURL} onChange={(e) => setBaseURL(e.target.value)} placeholder="https://dashscope.aliyuncs.com" />
          </div>
          <div className="space-y-1.5">
            <Label>API Key {isNew ? '' : '（留空则不修改）'}</Label>
            <Input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="sk-..." />
          </div>
          <div className="space-y-1.5">
            <Label>备注</Label>
            <Input value={remark} onChange={(e) => setRemark(e.target.value)} />
          </div>
          <div className="flex items-center justify-between rounded-md border px-3 py-2">
            <Label>设为默认</Label>
            <Switch checked={isDefault} onCheckedChange={setIsDefault} />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={onClose}>
              取消
            </Button>
            <Button onClick={() => save.mutate()} disabled={save.isPending || !name || !baseURL || (isNew && !apiKey)}>
              {save.isPending && <Loader2 className="size-4 animate-spin" />} 保存
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// 访问口令锁定（仅服务端启用 ACCESS_TOKEN 时显示）
function AuthLockSection() {
  const { data } = useQuery({ queryKey: ['auth-status'], queryFn: api.authStatus })
  if (!data?.auth_required) return null
  return (
    <Card>
      <CardContent className="flex items-center justify-between py-4">
        <div className="text-sm text-muted-foreground">
          平台已启用访问口令保护，锁定后需重新输入口令
        </div>
        <LogoutButton />
      </CardContent>
    </Card>
  )
}

// 声音复刻：上传音频样本 → create_voice
function VoiceCloneSection() {
  const qc = useQueryClient()
  const { data: ttsModels } = useQuery({
    queryKey: ['models', 'tts'],
    queryFn: () => api.listModels('tts', true),
  })
  const [targetModel, setTargetModel] = useState<string>('')
  const [prefix, setPrefix] = useState('')
  const [audio, setAudio] = useState<Asset[]>([])
  const model: ModelDef | undefined =
    ttsModels?.find((m) => m.code === targetModel) ?? ttsModels?.find((m) => m.is_default) ?? ttsModels?.[0]

  const clone = useMutation({
    mutationFn: () =>
      api.createVoiceClone({
        target_model: model!.code,
        prefix,
        audio_asset_id: audio[0].id,
      }),
    onSuccess: (r) => {
      toast.success(`复刻成功，音色 ID：${r.voice_id}`, { description: '可在语音创作的音色下拉框中选择（复刻）' })
      setPrefix('')
      setAudio([])
      qc.invalidateQueries({ queryKey: ['voices'] })
    },
    onError: (e: Error) => toast.error('复刻失败', { description: e.message }),
  })

  const submit = () => {
    if (!model) return toast.error('没有可用的 TTS 模型')
    if (!/^[a-z0-9]{1,10}$/.test(prefix)) return toast.error('音色前缀：1-10 位小写字母/数字')
    if (!audio.length) return toast.error('请上传 10 秒以上的音频样本')
    clone.mutate()
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Mic className="size-4" /> 声音复刻
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-xs text-muted-foreground">
          上传一段 10 秒以上的人声样本（wav/mp3），即可复刻专属音色，用于语音合成。生成的音色会出现在语音创作的音色列表中。
        </p>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label>驱动模型（复刻音色将用于该模型）</Label>
            <Select value={model?.code} onValueChange={setTargetModel}>
              <SelectTrigger><SelectValue placeholder="选择模型" /></SelectTrigger>
              <SelectContent>
                {ttsModels?.filter((m) => m.protocol === 'tts_http').map((m) => (
                  <SelectItem key={m.code} value={m.code}>{m.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>音色前缀（1-10 位小写字母/数字）</Label>
            <Input value={prefix} onChange={(e) => setPrefix(e.target.value)} placeholder="myvoice" />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label>音频样本</Label>
          <FilePicker
            spec={{ key: 'sample', label: '音频', type: 'audio', max: 1, required: true }}
            assets={audio}
            onChange={setAudio}
          />
        </div>
        <Separator />
        <Button size="sm" onClick={submit} disabled={clone.isPending}>
          {clone.isPending ? <Loader2 className="size-4 animate-spin" /> : <Mic className="size-4" />}
          开始复刻
        </Button>
      </CardContent>
    </Card>
  )
}
