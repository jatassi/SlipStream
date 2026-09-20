import { AlertTriangle, Settings2, Users } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { Segmented } from '@/components/ui/segmented'
import type { Request } from '@/types'

import type { RequestAction } from './request-actions'
import { RequestRow } from './request-row'
import { RequestSearchModal } from './request-search-modal'
import type { QueueSegment } from './request-status'
import { QUEUE_SEGMENTS } from './request-status'
import { useRequestQueuePage } from './use-request-queue-page'

const SKELETONS = ['request-a', 'request-b', 'request-c'] as const

const EMPTY_COPY: Record<QueueSegment, string> = {
  pending: 'No requests waiting for approval',
  approved: 'No approved requests',
  downloading: 'No requests downloading',
  available: 'No requests available yet',
  denied: 'No denied requests',
}

function ManageGroup() {
  return (
    <Group>
      <Row
        leading={
          <IconTile className="bg-violet-600">
            <Users />
          </IconTile>
        }
        title="Users"
        href="/requests-admin/users"
        chevron
      />
      <Row
        leading={
          <IconTile className="bg-zinc-600">
            <Settings2 />
          </IconTile>
        }
        title="Request Settings"
        href="/requests-admin/settings"
        chevron
      />
    </Group>
  )
}

function PortalDisabledGroup({ enabled }: { enabled: boolean }) {
  if (enabled) {
    return null
  }
  return (
    <Group>
      <Row
        tone="warning"
        leading={
          <IconTile className="bg-amber-500">
            <AlertTriangle />
          </IconTile>
        }
        title="The requests portal is disabled"
        subtitle="Portal users cannot sign in or submit requests"
        href="/requests-admin/settings"
        chevron
      />
    </Group>
  )
}

function QueueGroup({
  isLoading,
  segment,
  requests,
  processingRequest,
  requesterFor,
  onAction,
}: {
  isLoading: boolean
  segment: QueueSegment
  requests: Request[]
  processingRequest: number | null
  requesterFor: (userId: number) => string | undefined
  onAction: (request: Request, action: RequestAction) => void
}) {
  if (isLoading) {
    return (
      <Group>
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} leading="poster" trailing />
        ))}
      </Group>
    )
  }

  if (requests.length === 0) {
    return (
      <Group>
        <Row title={<span className="text-muted-foreground font-normal">{EMPTY_COPY[segment]}</span>} />
      </Group>
    )
  }

  return (
    <Group>
      {requests.map((request) => (
        <RequestRow
          key={request.id}
          request={request}
          requester={requesterFor(request.userId)}
          isProcessing={processingRequest === request.id}
          onAction={(action) => {
            onAction(request, action)
          }}
        />
      ))}
    </Group>
  )
}

export function RequestQueuePage() {
  const page = useRequestQueuePage()
  const back = usePushBack()

  if (page.isError) {
    return (
      <Screen title="Requests" back={back}>
        <ErrorState onRetry={page.refetch} />
      </Screen>
    )
  }

  return (
    <Screen title="Requests" back={back}>
      <PortalDisabledGroup enabled={page.portalEnabled} />
      <ManageGroup />
      <div className="px-screen pb-5">
        <Segmented
          label="Request status"
          value={page.segment}
          onChange={page.setSegment}
          options={QUEUE_SEGMENTS}
        />
      </div>
      <QueueGroup
        isLoading={page.isLoading}
        segment={page.segment}
        requests={page.visibleRequests}
        processingRequest={page.processingRequest}
        requesterFor={page.requesterFor}
        onAction={page.handleAction}
      />
      {page.searchModal ? (
        <RequestSearchModal searchModal={page.searchModal} onClose={page.handleSearchModalClose} />
      ) : null}
    </Screen>
  )
}
