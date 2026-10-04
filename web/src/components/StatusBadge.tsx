import { Badge } from '@/components/ui/badge'
import type { Task } from '@/lib/types'

const map: Record<string, { label: string; className: string }> = {
  queued: { label: '排队中', className: 'bg-muted text-muted-foreground' },
  submitting: { label: '提交中', className: 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300' },
  running: { label: '生成中', className: 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300' },
  succeeded: { label: '已完成', className: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' },
  failed: { label: '失败', className: 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300' },
  canceled: { label: '已取消', className: 'bg-muted text-muted-foreground' },
}

export default function StatusBadge({ status }: { status: Task['status'] }) {
  const s = map[status] ?? { label: status, className: '' }
  return <Badge className={s.className} variant="secondary">{s.label}</Badge>
}

export function activeTask(status: Task['status']) {
  return status === 'queued' || status === 'submitting' || status === 'running'
}
