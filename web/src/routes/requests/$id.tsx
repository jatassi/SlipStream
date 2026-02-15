import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

import { RequestDetailCard } from './request-detail-card'
import { RequestDetailHeader } from './request-detail-header'
import { STATUS_CONFIG } from './request-status-config'
import { useRequestDetail } from './use-request-detail'

export function RequestDetailPage() {
  const state = useRequestDetail()

  if (state.isLoading) {
    return (
      <div className="mx-auto max-w-4xl space-y-6 px-6 pt-6">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (state.error || !state.request) {
    return (
      <div className="mx-auto max-w-4xl px-6 pt-6">
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">Request not found</p>
            <Button onClick={state.goBack} className="mt-4">
              Back to Requests
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-6 pt-6">
      <RequestDetailHeader
        isOwner={state.isOwner}
        isWatching={state.request.isWatching}
        onBack={state.goBack}
        onWatch={state.handleWatch}
      />

      <RequestDetailCard
        request={state.request}
        isMovie={state.isMovie}
        statusConfig={STATUS_CONFIG[state.request.status]}
        canCancel={state.canCancel}
        cancelDialogOpen={state.cancelDialogOpen}
        setCancelDialogOpen={state.setCancelDialogOpen}
        cancelPending={state.cancelPending}
        download={state.download}
        onCancel={state.handleCancel}
      />
    </div>
  )
}
