import { Clock, Trash2, XCircle } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import type { RequestListProps } from './request-list'
import { RequestList } from './request-list'

type RequestTabsProps = {
  activeTab: string
  onTabChange: (value: string) => void
  pendingCount: number
  isSomeSelected: boolean
  selectedCount: number
  onOpenDenyDialog: () => void
  onOpenBatchDeleteDialog: () => void
} & RequestListProps

export function RequestTabs(props: RequestTabsProps) {
  return (
    <Tabs value={props.activeTab} onValueChange={props.onTabChange}>
      <TabHeader
        pendingCount={props.pendingCount}
        isSomeSelected={props.isSomeSelected}
        selectedCount={props.selectedCount}
        onOpenDenyDialog={props.onOpenDenyDialog}
        onOpenBatchDeleteDialog={props.onOpenBatchDeleteDialog}
      />

      <TabsContent value={props.activeTab} className="mt-0">
        {props.requests.length === 0 ? (
          <RequestsEmptyState activeTab={props.activeTab} />
        ) : (
          <RequestList
            requests={props.requests}
            selectedIds={props.selectedIds}
            isAllSelected={props.isAllSelected}
            processingRequest={props.processingRequest}
            onToggleSelectAll={props.onToggleSelectAll}
            onToggleSelect={props.onToggleSelect}
            onApproveOnly={props.onApproveOnly}
            onApproveAndManualSearch={props.onApproveAndManualSearch}
            onApproveAndAutoSearch={props.onApproveAndAutoSearch}
            onDeny={props.onDeny}
            onDelete={props.onDelete}
          />
        )}
      </TabsContent>
    </Tabs>
  )
}

function TabHeader({
  pendingCount,
  isSomeSelected,
  selectedCount,
  onOpenDenyDialog,
  onOpenBatchDeleteDialog,
}: {
  pendingCount: number
  isSomeSelected: boolean
  selectedCount: number
  onOpenDenyDialog: () => void
  onOpenBatchDeleteDialog: () => void
}) {
  return (
    <div className="mb-4 flex items-center justify-between">
      <TabsList>
        <TabsTrigger value="pending">
          Pending{' '}
          {pendingCount > 0 && (
            <Badge variant="secondary" className="ml-1">
              {pendingCount}
            </Badge>
          )}
        </TabsTrigger>
        <TabsTrigger value="approved">Approved</TabsTrigger>
        <TabsTrigger value="downloading">Downloading</TabsTrigger>
        <TabsTrigger value="available">Available</TabsTrigger>
        <TabsTrigger value="denied">Denied</TabsTrigger>
        <TabsTrigger value="all">All</TabsTrigger>
      </TabsList>

      {isSomeSelected ? (
        <BatchActions
          selectedCount={selectedCount}
          onDeny={onOpenDenyDialog}
          onDelete={onOpenBatchDeleteDialog}
        />
      ) : null}
    </div>
  )
}

function BatchActions({
  selectedCount,
  onDeny,
  onDelete,
}: {
  selectedCount: number
  onDeny: () => void
  onDelete: () => void
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground text-sm">{selectedCount} selected</span>
      <Button size="sm" variant="destructive" onClick={onDeny}>
        <XCircle className="mr-1 size-4" />
        Deny
      </Button>
      <Button size="sm" variant="outline" onClick={onDelete}>
        <Trash2 className="mr-1 size-4" />
        Delete
      </Button>
    </div>
  )
}

function RequestsEmptyState({ activeTab }: { activeTab: string }) {
  const description =
    activeTab === 'pending'
      ? 'No requests waiting for approval'
      : `No requests with status "${activeTab}"`

  return (
    <EmptyState
      icon={<Clock className="size-8" />}
      title={`No ${activeTab === 'all' ? '' : activeTab} requests`}
      description={description}
    />
  )
}
