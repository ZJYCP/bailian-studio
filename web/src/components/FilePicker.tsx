import { useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Loader2, Upload, X } from 'lucide-react'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import type { Asset, InputSpec } from '@/lib/types'

const acceptByType: Record<string, string> = {
  image: 'image/*',
  video: 'video/*',
  audio: 'audio/*',
  media: 'image/*,video/*',
}

// 文件选择与上传：选择后立即上传到平台，返回 asset 列表
export default function FilePicker({
  spec,
  assets,
  onChange,
}: {
  spec: InputSpec
  assets: Asset[]
  onChange: (assets: Asset[]) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const max = spec.max || 1
  const accept = acceptByType[spec.type] || '*/*'

  const upload = useMutation({
    mutationFn: async (files: File[]) => {
      const added: Asset[] = []
      for (const f of files) {
        added.push(await api.upload(f))
      }
      return added
    },
    onMutate: () => setUploading(true),
    onSettled: () => setUploading(false),
    onSuccess: (added) => {
      const next = [...assets, ...added].slice(0, max)
      if (added.length + assets.length > max) {
        toast.warning(`最多 ${max} 个文件，已截断`)
      }
      onChange(next)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="space-y-2">
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={max > 1}
        className="hidden"
        onChange={(e) => {
          // e.target.files 是 input 的实时视图，value 重置后会被清空；
          // 必须先快照成普通数组再交给异步的 mutate
          const files = Array.from(e.target.files ?? [])
          if (files.length) upload.mutate(files)
          e.target.value = ''
        }}
      />
      <div className="flex flex-wrap gap-2">
        {assets.map((a) => (
          <div key={a.id} className="group relative h-20 w-20 overflow-hidden rounded-md border bg-muted">
            {a.kind === 'image' ? (
              <img src={api.assetURL(a.id)} className="h-full w-full object-cover" />
            ) : a.kind === 'video' ? (
              <video src={api.assetURL(a.id)} className="h-full w-full object-cover" muted />
            ) : (
              <div className="flex h-full items-center justify-center text-xs text-muted-foreground">音频</div>
            )}
            <button
              type="button"
              className="absolute right-0.5 top-0.5 rounded-full bg-black/60 p-0.5 text-white opacity-0 transition-opacity group-hover:opacity-100"
              onClick={() => onChange(assets.filter((x) => x.id !== a.id))}
            >
              <X className="size-3" />
            </button>
          </div>
        ))}
        {assets.length < max && (
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            disabled={uploading}
            className="flex h-20 w-20 flex-col items-center justify-center gap-1 rounded-md border border-dashed text-muted-foreground transition-colors hover:border-primary hover:text-foreground"
          >
            {uploading ? <Loader2 className="size-4 animate-spin" /> : <Upload className="size-4" />}
            <span className="text-[10px]">{uploading ? '上传中' : spec.type === 'media' ? '图/视频' : '上传'}</span>
          </button>
        )}
      </div>
      {(spec.help || spec.required) && (
        <p className="text-xs text-muted-foreground">
          {spec.help}
          {spec.required ? '（必填）' : ''}
        </p>
      )}
    </div>
  )
}
