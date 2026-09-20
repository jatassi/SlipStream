import { FileClock } from 'lucide-react'

import { Group, IconTile, Row } from '@/components/grouped-list'
import { usePendingImports, useRetryImport } from '@/hooks'
import type { PendingImport } from '@/types'

function PendingRow({ item, onRetry }: { item: PendingImport; onRetry: (id: number) => void }) {
  const failed = item.status === 'failed'
  const retryId = item.id
  const canRetry = failed && retryId !== undefined
  return (
    <Row
      tone={failed ? 'destructive' : 'default'}
      leading={
        <IconTile className={failed ? 'bg-red-500' : 'bg-muted-foreground'}>
          <FileClock />
        </IconTile>
      }
      title={item.fileName}
      subtitle={item.error ?? item.status}
      trailing={canRetry ? 'Retry' : undefined}
      onClick={
        canRetry
          ? () => {
              onRetry(retryId)
            }
          : undefined
      }
    />
  )
}

export function PendingImportsGroup() {
  const { data: pending } = usePendingImports()
  const retry = useRetryImport()

  if (!pending || pending.length === 0) {
    return null
  }

  return (
    <Group header="Pending Imports">
      {pending.map((item) => (
        <PendingRow
          key={item.id ?? item.filePath}
          item={item}
          onRetry={(id) => {
            retry.mutate(id)
          }}
        />
      ))}
    </Group>
  )
}
