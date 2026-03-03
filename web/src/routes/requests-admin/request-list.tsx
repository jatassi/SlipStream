import { Checkbox } from '@/components/ui/checkbox'
import type { Request } from '@/types'

import type { RequestAction } from './request-actions'
import { RequestRow } from './request-row'

export type RequestListProps = {
  requests: Request[]
  selectedIds: Set<number>
  isAllSelected: boolean
  processingRequest: number | null
  onToggleSelectAll: () => void
  onToggleSelect: (id: number) => void
  onAction: (request: Request, action: RequestAction) => void
}

export function RequestList({
  requests,
  selectedIds,
  isAllSelected,
  processingRequest,
  onToggleSelectAll,
  onToggleSelect,
  onAction,
}: RequestListProps) {
  return (
    <div className="rounded-md border">
      <div className="bg-muted/40 flex items-center gap-4 border-b p-3">
        <Checkbox checked={isAllSelected} onCheckedChange={onToggleSelectAll} />
        <span className="text-muted-foreground text-sm">
          {requests.length} request{requests.length === 1 ? '' : 's'}
        </span>
      </div>
      <div className="divide-y">
        {requests.map((request) => (
          <RequestRow
            key={request.id}
            request={request}
            selected={selectedIds.has(request.id)}
            isProcessing={processingRequest === request.id}
            onToggleSelect={() => onToggleSelect(request.id)}
            onAction={(action) => onAction(request, action)}
          />
        ))}
      </div>
    </div>
  )
}
