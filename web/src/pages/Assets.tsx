import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, Download, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { api, fmtSize, fmtTime } from '@/lib/api'
import type { Asset } from '@/lib/types'
import MediaPreview from '@/components/MediaPreview'

export default function Assets() {
  const [kind, setKind] = useState('image')
  const [page, setPage] = useState(1)
  const [preview, setPreview] = useState<Asset | null>(null)
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['assets', kind, page],
    queryFn: () => api.listAssets({ kind, role: 'output', page, page_size: 24 }),
  })

  const remove = useMutation({
    mutationFn: (id: number) => api.deleteAsset(id),
    onSuccess: () => {
      toast.success('已删除')
      setPreview(null)
      qc.invalidateQueries({ queryKey: ['assets'] })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const total = data?.total ?? 0
  const pages = Math.max(1, Math.ceil(total / 24))
  const items = data?.items ?? []

  return (
    <div className="mx-auto max-w-6xl p-6">
      <div className="mb-4 flex items-center gap-3">
        <h1 className="text-xl font-semibold">资产库</h1>
        <span className="text-sm text-muted-foreground">{total} 个作品</span>
        <Select value={kind} onValueChange={(v) => { setKind(v); setPage(1) }}>
          <SelectTrigger className="ml-auto w-28"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="image">图片</SelectItem>
            <SelectItem value="video">视频</SelectItem>
            <SelectItem value="audio">音频</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="text-sm text-muted-foreground">加载中…</div>
      ) : items.length === 0 ? (
        <Card className="p-6 text-sm text-muted-foreground">暂无{kind === 'image' ? '图片' : kind === 'video' ? '视频' : '音频'}作品。</Card>
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {items.map((a) => (
            <Card key={a.id} className="group overflow-hidden">
              <button className="block w-full" onClick={() => setPreview(a)}>
                {a.kind === 'image' ? (
                  <img src={api.assetURL(a.id)} className="aspect-square w-full object-cover" loading="lazy" />
                ) : a.kind === 'video' ? (
                  <video src={api.assetURL(a.id)} className="aspect-square w-full bg-black object-cover" muted preload="metadata" />
                ) : (
                  <div className="flex aspect-square w-full flex-col items-center justify-center gap-2 text-xs text-muted-foreground">
                    <span className="text-2xl">🎵</span>
                    <span className="max-w-[80%] truncate px-2">{a.file_name}</span>
                  </div>
                )}
              </button>
              <div className="flex items-center justify-between px-2 py-1.5 text-xs text-muted-foreground">
                <span className="truncate">
                  {fmtSize(a.size)}
                  {a.width ? ` · ${a.width}×${a.height}` : ''}
                </span>
                <span className="flex shrink-0 gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                  <a title="下载" href={api.assetDownloadURL(a.id)} download>
                    <Download className="size-3.5 hover:text-foreground" />
                  </a>
                  <button title="删除" onClick={() => remove.mutate(a.id)}>
                    <Trash2 className="size-3.5 hover:text-destructive" />
                  </button>
                </span>
              </div>
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

      <Dialog open={!!preview} onOpenChange={(o) => !o && setPreview(null)}>
        <DialogContent className="max-w-4xl">
          {preview && (
            <div className="space-y-3">
              <MediaPreview
                asset={preview}
                className={preview.kind === 'image' ? 'max-h-[70vh] w-full' : 'max-h-[70vh] w-full'}
              />
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <div>
                  {preview.file_name} · {fmtSize(preview.size)}
                  {preview.width ? ` · ${preview.width}×${preview.height}` : ''} · {fmtTime(preview.created_at)}
                  {preview.task_id ? ` · 任务 #${preview.task_id}` : ''}
                </div>
                <a className="text-primary hover:underline" href={api.assetDownloadURL(preview.id)} download>
                  下载
                </a>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
