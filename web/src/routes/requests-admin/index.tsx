import { AlertCircle, FlaskConical } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import type { Request } from '@/types'

import type { RequestAction } from './request-actions'
import { RequestDialogs } from './request-dialogs'
import { RequestSearchModal } from './request-search-modal'
import { RequestTabs } from './request-tabs'
import { RequestsNav } from './requests-nav'
import { useRequestQueuePage } from './use-request-queue-page'

export function RequestQueuePage() {
  const page = useRequestQueuePage()

  if (page.isLoading) {
    return <LoadingLayout />
  }

  if (page.isError) {
    return <ErrorLayout onRetry={page.refetch} />
  }

  const handleAction = (request: Request, action: RequestAction) => {
    switch (action) {
      case 'approve': { void page.handleApproveOnly(request); break }
      case 'approve-manual-search': { void page.handleApproveAndManualSearch(request); break }
      case 'approve-auto-search': { void page.handleApproveAndAutoSearch(request); break }
      case 'deny': { page.openDenyDialog(request.id); break }
      case 'delete': { page.openDeleteDialog(request.id); break }
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="External Requests"
        description="Manage portal users and content requests"
        actions={page.developerMode ? <TestRequestButton /> : null}
      />

      <RequestsNav />
      {page.portalEnabled ? null : <PortalDisabledAlert />}

      <RequestTabs
        activeTab={page.activeTab}
        onTabChange={page.handleTabChange}
        pendingCount={page.pendingCount}
        isSomeSelected={page.isSomeSelected}
        selectedCount={page.selectedIds.size}
        requests={page.filteredRequests}
        selectedIds={page.selectedIds}
        isAllSelected={page.isAllSelected}
        processingRequest={page.processingRequest}
        onOpenDenyDialog={() => page.openDenyDialog()}
        onOpenBatchDeleteDialog={() => page.setShowBatchDeleteDialog(true)}
        onToggleSelectAll={page.toggleSelectAll}
        onToggleSelect={page.toggleSelect}
        onAction={handleAction}
      />

      <RequestDialogs page={page} />

      {page.searchModal ? (
        <RequestSearchModal searchModal={page.searchModal} onClose={page.handleSearchModalClose} />
      ) : null}
    </div>
  )
}

function LoadingLayout() {
  return (
    <div>
      <PageHeader title="Request Queue" />
      <div className="mx-auto max-w-6xl px-6 pt-6">
        <LoadingState variant="list" count={5} />
      </div>
    </div>
  )
}

function ErrorLayout({ onRetry }: { onRetry: () => void }) {
  return (
    <div>
      <PageHeader title="Request Queue" />
      <div className="mx-auto max-w-6xl px-6 pt-6">
        <ErrorState onRetry={onRetry} />
      </div>
    </div>
  )
}

function TestRequestButton() {
  return (
    <Button
      variant="outline"
      onClick={() =>
        toast.info('Test request feature coming soon', {
          description: 'This will allow creating test requests for debugging.',
        })
      }
    >
      <FlaskConical className="mr-2 size-4" />
      Test Request
    </Button>
  )
}

function PortalDisabledAlert() {
  return (
    <Alert>
      <AlertCircle className="size-4" />
      <AlertDescription>
        The external requests portal is currently disabled. Portal users cannot submit new requests
        or access the portal. You can re-enable it in the{' '}
        <a href="/requests-admin/settings" className="font-medium underline">
          Settings
        </a>{' '}
        tab.
      </AlertDescription>
    </Alert>
  )
}
