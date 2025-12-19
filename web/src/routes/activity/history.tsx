import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Film, Tv, History, Trash2, ArrowLeft } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { useHistory, useClearHistory } from '@/hooks'
import { formatRelativeTime } from '@/lib/formatters'
import { toast } from 'sonner'
import type { HistoryEventType } from '@/types'

const eventTypeColors: Record<HistoryEventType, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  grabbed: 'default',
  imported: 'secondary',
  deleted: 'destructive',
  failed: 'destructive',
  renamed: 'outline',
}

export function HistoryPage() {
  const [eventType, setEventType] = useState<HistoryEventType | 'all'>('all')
  const [page, setPage] = useState(1)

  const { data: history, isLoading, isError, refetch } = useHistory({
    eventType: eventType === 'all' ? undefined : eventType,
    page,
    pageSize: 50,
  })

  const clearMutation = useClearHistory()

  const handleClearHistory = async () => {
    try {
      await clearMutation.mutateAsync()
      toast.success('History cleared')
    } catch {
      toast.error('Failed to clear history')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="History" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="History" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="History"
        description="View past activity and events"
        breadcrumbs={[
          { label: 'Activity', href: '/activity' },
          { label: 'History' },
        ]}
        actions={
          <div className="flex gap-2">
            <Link to="/activity">
              <Button variant="ghost">
                <ArrowLeft className="size-4 mr-2" />
                Back to Queue
              </Button>
            </Link>
            <ConfirmDialog
              trigger={
                <Button variant="destructive">
                  <Trash2 className="size-4 mr-2" />
                  Clear History
                </Button>
              }
              title="Clear history"
              description="Are you sure you want to clear all history? This action cannot be undone."
              confirmLabel="Clear"
              variant="destructive"
              onConfirm={handleClearHistory}
            />
          </div>
        }
      />

      {/* Filters */}
      <div className="flex items-center gap-4 mb-4">
        <Select
          value={eventType}
          onValueChange={(v) => v && setEventType(v as HistoryEventType | 'all')}
        >
          <SelectTrigger className="w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Events</SelectItem>
            <SelectItem value="grabbed">Grabbed</SelectItem>
            <SelectItem value="imported">Imported</SelectItem>
            <SelectItem value="deleted">Deleted</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="renamed">Renamed</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* History table */}
      <Card>
        <CardContent className="p-0">
          {!history?.items?.length ? (
            <EmptyState
              icon={<History className="size-8" />}
              title="No history"
              description="Activity history will appear here"
              className="py-8"
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12"></TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>Quality</TableHead>
                  <TableHead>Date</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {history.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      {item.mediaType === 'movie' ? (
                        <Film className="size-4 text-muted-foreground" />
                      ) : (
                        <Tv className="size-4 text-muted-foreground" />
                      )}
                    </TableCell>
                    <TableCell className="font-medium">
                      {item.mediaTitle || `${item.mediaType} #${item.mediaId}`}
                    </TableCell>
                    <TableCell>
                      <Badge variant={eventTypeColors[item.eventType]}>
                        {item.eventType}
                      </Badge>
                    </TableCell>
                    <TableCell>{item.quality || '-'}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatRelativeTime(item.createdAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pagination */}
      {history && history.totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-4">
          <Button
            variant="outline"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
          >
            Previous
          </Button>
          <span className="flex items-center px-4">
            Page {page} of {history.totalPages}
          </span>
          <Button
            variant="outline"
            onClick={() => setPage((p) => Math.min(history.totalPages, p + 1))}
            disabled={page === history.totalPages}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  )
}
