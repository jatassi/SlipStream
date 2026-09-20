import { AlertTriangle, CheckCircle2, FlaskConical, XCircle } from 'lucide-react'
import { toast } from 'sonner'

import { IconTile, Row } from '@/components/grouped-list'
import { useTestHealthItem } from '@/hooks/use-health'
import { cn } from '@/lib/utils'
import type { HealthItem, HealthStatus } from '@/types/health'

import { formatRelativeTime } from './health-utils'

const STATUS_TILE: Record<HealthStatus, { className: string; icon: typeof CheckCircle2 }> = {
  ok: { className: 'bg-emerald-600', icon: CheckCircle2 },
  warning: { className: 'bg-amber-500', icon: AlertTriangle },
  error: { className: 'bg-destructive', icon: XCircle },
}

export function StatusTile({ status }: { status: HealthStatus }) {
  const tile = STATUS_TILE[status]
  return (
    <IconTile className={tile.className}>
      <tile.icon />
    </IconTile>
  )
}

function subtitleFor(item: HealthItem): string | undefined {
  const age = formatRelativeTime(item.timestamp)
  if (item.message !== undefined && item.message !== '') {
    return age === '' ? item.message : `${item.message} · ${age}`
  }
  return age === '' ? undefined : age
}

export function TestItemButton({ item }: { item: HealthItem }) {
  const testItem = useTestHealthItem()

  const handleTest = async () => {
    try {
      const result = await testItem.mutateAsync({ category: item.category, id: item.id })
      const message = result.message || (result.success ? 'Test passed' : 'Test failed')
      ;(result.success ? toast.success : toast.error)(`${item.name}: ${message}`)
    } catch {
      toast.error(`${item.name}: Connection test failed`)
    }
  }

  return (
    <button
      type="button"
      aria-label={`Test ${item.name}`}
      disabled={testItem.isPending}
      onClick={() => void handleTest()}
      className="press flex size-tap items-center justify-center rounded-md text-muted-foreground focus-visible:ring-ring outline-none focus-visible:ring-[3px]"
    >
      <FlaskConical className={cn('size-4', testItem.isPending && 'animate-pulse')} />
    </button>
  )
}

export function HealthRow({ item, testable = true }: { item: HealthItem; testable?: boolean }) {
  return (
    <Row
      leading={<StatusTile status={item.status} />}
      title={item.name}
      subtitle={subtitleFor(item)}
      // eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean
      trailing={testable && <TestItemButton item={item} />}
    />
  )
}
