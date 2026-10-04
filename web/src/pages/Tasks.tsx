import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, RotateCcw, Trash2, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { api, fmtSize, fmtTime } from '@/lib/api'
import type { Task } from '@/lib/types'
import StatusBadge, { activeTask } from '@/components/StatusBadge'
import MediaPreview from '@/components/MediaPreview'

const statusOptions = [
  { value: 'all', label: '全部状态' },
  { value: 'running', label: '进行中' },
  { value: 'succeeded', label: '已完成' },
  { value: 'failed', label: '失败' },
  { value: 'canceled', label: '已取消' },
]

export default function Tasks() {
  const [status, setStatus] = useState('all')
  const [capability, setCapability] = useState('all')
  const [page, setPage] = useState(1)
  const [detail, setDetail] = useState<Task | null>(null)
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['tasks', status, capability, page],
    queryFn: () =>
      api.listTasks({
        status: status === 'running' ? undefined : status === 'all' ? undefined : status,
        active: status === 'running' ? 'true' : undefined,
        capability: capability === 'all' ? undefined : capability,
        page,
        page_size: 20,
      }),
    refetchInterval: (q) => (q.state.data?.items.some((t) => activeTask(t.status)) ? 4000 : false),
  })

  const retry = useMutation({
    mutationFn: (id: number) => api.retryTask(id),
    onSuccess: (t) => {
      toast.success(`已重新提交为任务 #${t.id}`)
      qc.invalidateQueries({ queryKey: ['tasks'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const cancel = useMutation({
    mutationFn: (id: number) => api.cancelTask(id),
    onSuccess: () => {
      toast.success('已取消')
      qc.invalidateQueries({ queryKey: ['tasks'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: number) => api.deleteTask(id),
    onSuccess: () => {
      toast.success('已删除')
      setDetail(null)
      qc.invalidateQueries({ queryKey: ['tasks'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const total = data?.total ?? 0
  const pages = Math.max(1, Math.ceil(total / 20))

  return (
    <div className="mx-auto max-w-6xl p-6">
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold">任务列表</h1>
        <span className="text-sm text-muted-foreground">共 {total} 条</span>
        <div className="ml-auto flex gap-2">
          <Select value={status} onValueChange={(v) => { setStatus(v); setPage(1) }}>
            <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
            <SelectContent>
              {statusOptions.map((s) => (
                <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={capability} onValueChange={(v) => { setCapability(v); setPage(1) }}>
            <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部能力</SelectItem>
              <SelectItem value="image">图像</SelectItem>
              <SelectItem value="video">视频</SelectItem>
              <SelectItem value="tts">语音</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {isLoading ? (
        <div className="text-sm text-muted-foreground">加载中…</div>
      ) : !data?.items.length ? (
        <Card>
          <CardContent className="pt-6 text-sm text-muted-foreground">暂无任务，去「创作中心」开始吧。</CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {data.items.map((t) => (
            <Card
              key={t.id}
              className="cursor-pointer transition-colors hover:border-primary/50"
              onClick={() => setDetail(t)}
            >
              <CardContent className="flex items-center gap-4 py-3">
                {t.outputs?.[0] ? (
                  <div className="h-14 w-14 shrink-0 overflow-hidden rounded-md bg-muted">
                    {t.outputs[0].kind === 'image' ? (
                      <img src={api.assetURL(t.outputs[0].id)} className="h-full w-full object-cover" />
                    ) : t.outputs[0].kind === 'video' ? (
                      <video src={api.assetURL(t.outputs[0].id)} className="h-full w-full object-cover" muted preload="metadata" />
                    ) : (
                      <div className="flex h-full items-center justify-center text-[10px] text-muted-foreground">音频</div>
                    )}
                  </div>
                ) : (
                  <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-md bg-muted text-xs text-muted-foreground">
                    {activeTask(t.status) ? '⏳' : t.status === 'failed' ? '✕' : '—'}
                  </div>
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">#{t.id}</span>
                    <StatusBadge status={t.status} />
                    <span className="truncate text-xs text-muted-foreground">{t.model_code}</span>
                  </div>
                  <div className="mt-0.5 truncate text-xs text-muted-foreground">{t.prompt || '（无提示词）'}</div>
                </div>
                <div className="hidden shrink-0 text-right text-xs text-muted-foreground sm:block">
                  <div>{fmtTime(t.created_at)}</div>
                  {t.outputs && t.outputs.length > 0 && <div>{t.outputs.length} 个产物</div>}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {pages > 1 && (
        <div className="mt-4 flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            <ChevronLeft className="size-4" />
          </Button>
          <span className="text-sm text-muted-foreground">
            {page} / {pages}
          </span>
          <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>
            <ChevronRight className="size-4" />
          </Button>
        </div>
      )}

      <Dialog open={!!detail} onOpenChange={(o) => !o && setDetail(null)}>
        <DialogContent className="max-w-3xl">
          {detail && <TaskDetail task={detail} onRetry={(id) => { setDetail(null); retry.mutate(id) }} onCancel={(id) => cancel.mutate(id)} onDelete={(id) => remove.mutate(id)} />}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function TaskDetail({
  task,
  onRetry,
  onCancel,
  onDelete,
}: {
  task: Task
  onRetry: (id: number) => void
  onCancel: (id: number) => void
  onDelete: (id: number) => void
}) {
  const { data: fresh } = useQuery({
    queryKey: ['task', task.id],
    queryFn: () => api.getTask(task.id),
    initialData: task,
    refetchInterval: (q) => (q.state.data && activeTask(q.state.data.status) ? 3000 : false),
  })
  const t = fresh ?? task
  return (
    <div className="space-y-4">
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          任务 #{t.id} <StatusBadge status={t.status} />
        </DialogTitle>
      </DialogHeader>
      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
        <Info label="模型" value={t.model_code} />
        <Info label="模式" value={t.mode} />
        <Info label="服务" value={t.provider_name} />
        <Info label="创建时间" value={fmtTime(t.created_at)} />
        {t.external_task_id && <Info label="远端任务" value={t.external_task_id} />}
        {t.usage && <Info label="用量" value={JSON.stringify(t.usage)} />}
      </div>
      {t.prompt && (
        <div>
          <div className="mb-1 text-xs font-medium text-muted-foreground">提示词</div>
          <div className="whitespace-pre-wrap rounded-md bg-muted p-3 text-sm">{t.prompt}</div>
        </div>
      )}
      {t.params && Object.keys(t.params).length > 0 && (
        <div>
          <div className="mb-1 text-xs font-medium text-muted-foreground">参数</div>
          <div className="flex flex-wrap gap-1.5">
            {Object.entries(t.params).map(([k, v]) => (
              <span key={k} className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                {k}: {String(v)}
              </span>
            ))}
          </div>
        </div>
      )}
      {t.inputs && t.inputs.length > 0 && (
        <div>
          <div className="mb-1 text-xs font-medium text-muted-foreground">输入素材</div>
          <div className="flex flex-wrap gap-2">
            {t.inputs.map((a) => (
              <div key={a.id} className="h-16 w-16 overflow-hidden rounded border bg-muted">
                {a.kind === 'image' ? (
                  <img src={api.assetURL(a.id)} className="h-full w-full object-cover" />
                ) : (
                  <div className="flex h-full items-center justify-center text-[10px] text-muted-foreground">{a.kind}</div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
      {t.status === 'failed' && t.error_message && (
        <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          {t.error_code}: {t.error_message}
        </div>
      )}
      {t.outputs && t.outputs.length > 0 && (
        <div>
          <div className="mb-1 text-xs font-medium text-muted-foreground">结果（{t.outputs.length}）</div>
          <div className={t.outputs[0].kind === 'image' ? 'grid grid-cols-2 gap-2' : 'space-y-3'}>
            {t.outputs.map((a) => (
              <div key={a.id} className="space-y-1">
                <MediaPreview asset={a} className={a.kind === 'image' ? 'h-44 w-full rounded-md' : 'w-full rounded-md'} />
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>
                    {fmtSize(a.size)}
                    {a.width ? ` · ${a.width}×${a.height}` : ''}
                  </span>
                  <a className="text-primary hover:underline" href={api.assetDownloadURL(a.id)}>
                    下载
                  </a>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
      <div className="flex justify-end gap-2 border-t pt-3">
        {activeTask(t.status) ? (
          <Button variant="outline" size="sm" onClick={() => onCancel(t.id)}>
            <XCircle className="size-4" /> 取消任务
          </Button>
        ) : (
          <>
            <Button variant="outline" size="sm" onClick={() => onRetry(t.id)}>
              <RotateCcw className="size-4" /> 重试
            </Button>
            <Button variant="destructive" size="sm" onClick={() => onDelete(t.id)}>
              <Trash2 className="size-4" /> 删除
            </Button>
          </>
        )}
      </div>
    </div>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex gap-2 text-sm">
      <span className="shrink-0 text-xs leading-6 text-muted-foreground">{label}</span>
      <span className="truncate font-mono text-xs leading-6">{value}</span>
    </div>
  )
}
