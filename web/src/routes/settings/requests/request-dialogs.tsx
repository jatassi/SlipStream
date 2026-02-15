import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import type { useRequestQueuePage } from './use-request-queue-page'

type PageState = ReturnType<typeof useRequestQueuePage>

export function RequestDialogs({ page }: { page: PageState }) {
  return (
    <>
      <DenyDialog
        open={page.showDenyDialog}
        onOpenChange={page.setShowDenyDialog}
        isBatch={!page.pendingDenyId}
        batchCount={page.selectedIds.size}
        reason={page.denyReason}
        onReasonChange={page.setDenyReason}
        onConfirm={page.handleDeny}
        isPending={page.denyMutation.isPending || page.batchDenyMutation.isPending}
      />

      <DeleteDialog
        open={page.showDeleteDialog}
        onOpenChange={page.setShowDeleteDialog}
        onConfirm={page.handleDelete}
        isPending={page.deleteMutation.isPending}
      />

      <BatchDeleteDialog
        open={page.showBatchDeleteDialog}
        onOpenChange={page.setShowBatchDeleteDialog}
        count={page.selectedIds.size}
        onConfirm={page.handleBatchDelete}
        isPending={page.batchDeleteMutation.isPending}
      />
    </>
  )
}

type DenyDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  isBatch: boolean
  batchCount: number
  reason: string
  onReasonChange: (reason: string) => void
  onConfirm: () => void
  isPending: boolean
}

function DenyDialog({
  open,
  onOpenChange,
  isBatch,
  batchCount,
  reason,
  onReasonChange,
  onConfirm,
  isPending,
}: DenyDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Deny Request{batchCount > 1 ? 's' : ''}</DialogTitle>
          <DialogDescription>
            {isBatch
              ? `Deny ${batchCount} selected request${batchCount === 1 ? '' : 's'}.`
              : 'Optionally provide a reason for denying this request.'}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Reason (Optional)</Label>
            <Textarea
              placeholder="e.g., Content not available in region, already in library, etc."
              value={reason}
              onChange={(e) => onReasonChange(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Deny
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DeleteDialog({
  open,
  onOpenChange,
  onConfirm,
  isPending,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  isPending: boolean
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Request</DialogTitle>
          <DialogDescription>
            Are you sure you want to permanently delete this request? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function BatchDeleteDialog({
  open,
  onOpenChange,
  count,
  onConfirm,
  isPending,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  count: number
  onConfirm: () => void
  isPending: boolean
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            Delete {count} Request{count === 1 ? '' : 's'}
          </DialogTitle>
          <DialogDescription>
            Are you sure you want to permanently delete {count} selected request
            {count === 1 ? '' : 's'}? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
            {isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
