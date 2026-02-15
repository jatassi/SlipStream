import { FlaskConical } from 'lucide-react'
import { toast } from 'sonner'

import { getStatusBgColor, StatusIndicator } from '@/components/health'
import { Button } from '@/components/ui/button'
import { useTestHealthItem } from '@/hooks/use-health'
import { cn } from '@/lib/utils'
import type { HealthItem } from '@/types/health'

import { formatRelativeTime } from './health-utils'

type HealthItemRowProps = {
  item: HealthItem
  indented?: boolean
  hideTestButton?: boolean
}

function HealthItemDetails({ item }: { item: HealthItem }) {
  return (
    <div className="min-w-0 flex-1">
      <div className="truncate font-medium">{item.name}</div>
      {item.message ? <div className="text-muted-foreground truncate text-sm">{item.message}</div> : null}
      {item.timestamp ? <div className="text-muted-foreground text-xs">{formatRelativeTime(item.timestamp)}</div> : null}
    </div>
  )
}

export function HealthItemRow({ item, indented, hideTestButton }: HealthItemRowProps) {
  const testItem = useTestHealthItem()

  const handleTest = async () => {
    try {
      const result = await testItem.mutateAsync({ category: item.category, id: item.id })
      const msg = result.message || (result.success ? 'Test passed' : 'Test failed')
      ;(result.success ? toast.success : toast.error)(`${item.name}: ${msg}`)
    } catch {
      toast.error(`${item.name}: Connection test failed`)
    }
  }

  return (
    <div className={cn('flex items-center justify-between rounded-md p-3', indented && 'pl-8', getStatusBgColor(item.status))}>
      <div className="flex min-w-0 flex-1 items-center gap-3">
        {indented ? <div className="text-muted-foreground">{'\u2514'}</div> : null}
        <StatusIndicator status={item.status} size="md" />
        <HealthItemDetails item={item} />
      </div>
      {hideTestButton ? null : (
        <Button variant="ghost" size="sm" onClick={handleTest} disabled={testItem.isPending} title={`Test ${item.name}`}>
          <FlaskConical className={cn('size-4', testItem.isPending && 'animate-pulse')} />
        </Button>
      )}
    </div>
  )
}
