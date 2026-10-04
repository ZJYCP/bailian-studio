import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { FieldSchema } from '@/lib/types'

// 按 schema 渲染单个参数控件
export default function SchemaField({
  field,
  value,
  onChange,
  voices,
}: {
  field: FieldSchema
  value: unknown
  onChange: (v: unknown) => void
  voices?: string[] // voice 类型字段的可选项（内置+复刻）
}) {
  const label = (
    <Label className="text-xs font-medium text-muted-foreground">
      {field.label}
      {field.required ? ' *' : ''}
    </Label>
  )
  const help = field.help ? <p className="mt-1 text-xs text-muted-foreground">{field.help}</p> : null

  if (field.type === 'voice') {
    const opts = voices?.length ? voices : field.options || []
    const known = opts.includes(String(value ?? ''))
    return (
      <div className="space-y-1.5">
        {label}
        <div className="flex gap-2">
          <Select value={known ? String(value ?? '') : '__custom__'} onValueChange={(v) => onChange(v === '__custom__' ? '' : v)}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="选择或输入音色" />
            </SelectTrigger>
            <SelectContent>
              {opts.map((o) => (
                <SelectItem key={o} value={o}>
                  {field.option_labels?.[o] || o}
                  {voices?.includes(o) && !field.options?.includes(o) ? '（复刻）' : ''}
                </SelectItem>
              ))}
              <SelectItem value="__custom__">自定义…</SelectItem>
            </SelectContent>
          </Select>
          {!known && (
            <Input
              placeholder="音色 ID"
              value={String(value ?? '')}
              onChange={(e) => onChange(e.target.value)}
              className="w-44"
            />
          )}
        </div>
        {help}
      </div>
    )
  }

  if (field.type === 'select') {
    const opts = field.options || []
    return (
      <div className="space-y-1.5">
        {label}
        <Select value={value != null && value !== '' ? String(value) : undefined} onValueChange={onChange}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder={field.placeholder || '选择'} />
          </SelectTrigger>
          <SelectContent>
            {opts.map((o) => (
              <SelectItem key={o} value={o}>
                {field.option_labels?.[o] || o}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {field.allowCustom && (
          <Input
            placeholder="或输入自定义值"
            value={opts.includes(String(value ?? '')) ? '' : String(value ?? '')}
            onChange={(e) => onChange(e.target.value)}
          />
        )}
        {help}
      </div>
    )
  }

  if (field.type === 'boolean') {
    return (
      <div className="flex items-center justify-between rounded-md border px-3 py-2.5">
        <div>
          {label}
          {field.help && <p className="text-xs text-muted-foreground">{field.help}</p>}
        </div>
        <Switch checked={Boolean(value)} onCheckedChange={onChange} />
      </div>
    )
  }

  if (field.type === 'number') {
    return (
      <div className="space-y-1.5">
        {label}
        <Input
          type="number"
          value={value === undefined || value === null ? '' : String(value)}
          min={field.min}
          max={field.max}
          step={field.step || 1}
          placeholder={field.placeholder}
          onChange={(e) => {
            const v = e.target.value
            onChange(v === '' ? undefined : Number(v))
          }}
        />
        {help}
      </div>
    )
  }

  if (field.type === 'textarea') {
    return (
      <div className="space-y-1.5">
        {label}
        <Textarea
          rows={2}
          value={String(value ?? '')}
          placeholder={field.placeholder}
          onChange={(e) => onChange(e.target.value)}
        />
        {help}
      </div>
    )
  }

  return (
    <div className="space-y-1.5">
      {label}
      <Input
        value={String(value ?? '')}
        placeholder={field.placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
      {help}
    </div>
  )
}

// schema 默认值 → 表单初值
export function defaultsFrom(fields: FieldSchema[]): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const f of fields) {
    if (f.default !== undefined) out[f.key] = f.default
    else if (f.type === 'boolean') out[f.key] = false
  }
  return out
}
