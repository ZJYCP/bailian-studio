import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Star, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { api, capabilityLabel } from '@/lib/api'
import type { Capability, ModelDef } from '@/lib/types'

const caps: Capability[] = ['image', 'video', 'tts']

export default function Models() {
  const qc = useQueryClient()
  const { data: models } = useQuery({ queryKey: ['models-all'], queryFn: () => api.listModels() })
  const [saving, setSaving] = useState<number | null>(null)

  const update = useMutation({
    mutationFn: ({ id, patch }: { id: number; patch: Partial<ModelDef> }) => {
      setSaving(id)
      return api.updateModel(id, patch)
    },
    onSettled: () => setSaving(null),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['models-all'] }),
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: number) => api.deleteModel(id),
    onSuccess: () => {
      toast.success('已删除')
      qc.invalidateQueries({ queryKey: ['models-all'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="mx-auto max-w-5xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <h1 className="text-xl font-semibold">模型管理</h1>
        <span className="text-sm text-muted-foreground">
          {models?.filter((m) => m.enabled).length ?? 0} / {models?.length ?? 0} 已启用
        </span>
      </div>

      {caps.map((cap) => {
        const list = (models ?? []).filter((m) => m.capability === cap)
        if (!list.length) return null
        return (
          <div key={cap} className="mb-6">
            <h2 className="mb-2 text-sm font-semibold text-muted-foreground">
              {capabilityLabel[cap]}模型
            </h2>
            <div className="space-y-2">
              {list.map((m) => (
                <Card key={m.id}>
                  <CardContent className="flex items-center gap-3 py-3">
                    <Switch
                      checked={m.enabled}
                      onCheckedChange={(v) => update.mutate({ id: m.id, patch: { enabled: v } })}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-sm font-medium">{m.name || m.code}</span>
                        {m.is_default && (
                          <Badge variant="secondary" className="shrink-0">
                            默认
                          </Badge>
                        )}
                        {saving === m.id && <span className="text-xs text-muted-foreground">保存中…</span>}
                      </div>
                      <div className="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs text-muted-foreground">
                        <span className="font-mono">{m.code}</span>
                        <span>协议 {m.protocol}</span>
                        <span>{m.param_schema.modes.map((x) => x.label).join(' / ')}</span>
                        {m.remark && <span className="truncate">{m.remark}</span>}
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      {!m.is_default && m.enabled && (
                        <Button
                          variant="ghost"
                          size="sm"
                          title="设为该能力默认模型"
                          onClick={() => update.mutate({ id: m.id, patch: { is_default: true } })}
                        >
                          <Star className="size-4" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-muted-foreground hover:text-destructive"
                        onClick={() => {
                          if (confirm(`删除模型 ${m.code}？`)) remove.mutate(m.id)
                        }}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
