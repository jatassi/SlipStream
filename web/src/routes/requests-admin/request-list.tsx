import { Checkbox } from '@/components/ui/checkbox'
import type { Request } from '@/types'

import { RequestRow } from './request-row'

export type RequestListProps = {
  requests: Request[]
  selectedIds: Set<number>
  isAllSelected: boolean
  processingRequest: number | null
  onToggleSelectAll: () => void
  onToggleSelect: (id: number) => void
  onApproveOnly: (request: Request) => void
  onApproveAndManualSearch: (request: Request) => void
  onApproveAndAutoSearch: (request: Request) => void
  onDeny: (id: number) => void
  onDelete: (id: number) => void
}

export function RequestList({
  requests,
  selectedIds,
  isAllSelected,
  processingRequest,
  onToggleSelectAll,
  onToggleSelect,
  onApproveOnly,
  onApproveAndManualSearch,
  onApproveAndAutoSearch,
  onDeny,
  onDelete,
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
            onApproveOnly={() => onApproveOnly(request)}
            onApproveAndManualSearch={() => onApproveAndManualSearch(request)}
            onApproveAndAutoSearch={() => onApproveAndAutoSearch(request)}
            onDeny={() => onDeny(request.id)}
            onDelete={() => onDelete(request.id)}
          />
        ))}
      </div>
    </div>
  )
}
