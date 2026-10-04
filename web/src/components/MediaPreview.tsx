import { api } from '@/lib/api'
import type { Asset } from '@/lib/types'

// 媒体预览：图/视频/音频自适应；className 控制尺寸
export default function MediaPreview({ asset, className = '' }: { asset: Asset; className?: string }) {
  const url = api.assetURL(asset.id)
  if (asset.kind === 'image') {
    return <img src={url} alt={asset.file_name} loading="lazy" className={`object-contain ${className}`} />
  }
  if (asset.kind === 'video') {
    return <video src={url} controls preload="metadata" className={`bg-black object-contain ${className}`} />
  }
  return <audio src={url} controls preload="metadata" className={`w-full ${className}`} />
}
