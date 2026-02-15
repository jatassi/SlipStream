import { Link } from '@tanstack/react-router'
import { ExternalLink, FlaskConical, Settings } from 'lucide-react'
import { toast } from 'sonner'

import { getStatusBgColor, StatusIndicator } from '@/components/health'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useTestHealthCategory, useTestHealthItem } from '@/hooks/use-health'
import { cn } from '@/lib/utils'
import type { HealthItem } from '@/types/health'

import { HealthItemRow } from './health-item-row'
import { formatRelativeTime, getResultText, getWorstStatus } from './health-utils'

function showIndexerTestResults(indexerItems: HealthItem[], results: { id: string; success: boolean }[]) {
  const passed = results.filter((r) => r.success)
  const failed = results.filter((r) => !r.success)
  const passText = getResultText({ category: 'prowlarr_indexers', allItems: indexerItems, resultItems: passed, success: true })
  const failText = getResultText({ category: 'prowlarr_indexers', allItems: indexerItems, resultItems: failed, success: false })

  if (failed.length === 0) {
    toast.success(`Indexers: ${passText}`)
  } else if (passed.length === 0) {
    toast.error(`Indexers: ${failText}`)
  } else {
    toast.warning(`Indexers: ${passText}, ${failText}`)
  }
}

type ProwlarrTreeCardProps = {
  prowlarrItem: HealthItem | undefined
  indexerItems: HealthItem[]
}

export function ProwlarrTreeCard({ prowlarrItem, indexerItems }: ProwlarrTreeCardProps) {
  const testCategory = useTestHealthCategory()
  const testItem = useTestHealthItem()
  const worstStatus = getWorstStatus(indexerItems, prowlarrItem?.status ?? 'ok')
  const totalItems = (prowlarrItem ? 1 : 0) + indexerItems.length

  const handleTestProwlarr = async () => {
    if (!prowlarrItem) {
      return
    }
    try {
      const result = await testItem.mutateAsync({ category: 'prowlarr', id: prowlarrItem.id })
      const msg = result.success
        ? (result.message || 'Connection successful')
        : (result.message || 'Connection failed')
      if (result.success) {
        toast.success(`Prowlarr: ${msg}`)
      } else {
        toast.error(`Prowlarr: ${msg}`)
      }
    } catch {
      toast.error('Prowlarr: Connection test failed')
    }
  }

  const handleTestAll = async () => {
    try {
      const result = await testCategory.mutateAsync('indexers')
      showIndexerTestResults(indexerItems, result.results)
    } catch {
      toast.error('Indexers: Test failed')
    }
  }

  return (
    <Card>
      <ProwlarrCardHeader
        worstStatus={worstStatus}
        totalItems={totalItems}
        onTestAll={handleTestAll}
        testPending={testCategory.isPending}
        disableTestAll={indexerItems.length === 0}
      />
      <ProwlarrCardContent
        prowlarrItem={prowlarrItem}
        indexerItems={indexerItems}
        onTestProwlarr={handleTestProwlarr}
        isPending={testItem.isPending}
      />
    </Card>
  )
}

type ProwlarrCardHeaderProps = {
  worstStatus: 'ok' | 'warning' | 'error'
  totalItems: number
  onTestAll: () => void
  testPending: boolean
  disableTestAll: boolean
}

function ProwlarrCardHeader({ worstStatus, totalItems, onTestAll, testPending, disableTestAll }: ProwlarrCardHeaderProps) {
  return (
    <CardHeader className="flex flex-row items-center justify-between pb-2">
      <div className="flex items-center gap-2">
        <StatusIndicator status={worstStatus} size="sm" />
        <CardTitle className="text-base">Indexers</CardTitle>
        <span className="text-muted-foreground text-sm">
          ({totalItems} item{totalItems === 1 ? '' : 's'})
        </span>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={onTestAll}
          disabled={testPending || disableTestAll}
          title="Test all indexers"
        >
          <FlaskConical className={cn('mr-1 size-4', testPending && 'animate-pulse')} />
          Test All
        </Button>
        <Link to="/settings/downloads">
          <Button variant="ghost" size="sm" title="Settings">
            <Settings className="mr-1 size-4" />
            Settings
          </Button>
        </Link>
      </div>
    </CardHeader>
  )
}

type ProwlarrCardContentProps = {
  prowlarrItem: HealthItem | undefined
  indexerItems: HealthItem[]
  onTestProwlarr: () => void
  isPending: boolean
}

function ProwlarrCardContent({ prowlarrItem, indexerItems, onTestProwlarr, isPending }: ProwlarrCardContentProps) {
  if (!prowlarrItem) {
    return (
      <CardContent className="space-y-2">
        <div className="text-muted-foreground py-4 text-center text-sm">
          Prowlarr not configured.{' '}
          <Link
            to="/settings/downloads"
            className="text-primary inline-flex items-center gap-1 hover:underline"
          >
            Configure <ExternalLink className="size-3" />
          </Link>
        </div>
      </CardContent>
    )
  }

  return (
    <CardContent className="space-y-2">
      <ProwlarrParentRow item={prowlarrItem} onTest={onTestProwlarr} isPending={isPending} />
      {indexerItems.length === 0 ? (
        <div className="text-muted-foreground py-2 pl-8 text-sm">
          No indexers found in Prowlarr
        </div>
      ) : (
        indexerItems.map((item) => <HealthItemRow key={item.id} item={item} indented />)
      )}
    </CardContent>
  )
}

type ProwlarrParentRowProps = {
  item: HealthItem
  onTest: () => void
  isPending: boolean
}

function ProwlarrParentRow({ item, onTest, isPending }: ProwlarrParentRowProps) {
  return (
    <div
      className={cn(
        'flex items-center justify-between rounded-md p-3',
        getStatusBgColor(item.status),
      )}
    >
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <StatusIndicator status={item.status} size="md" />
        <div className="min-w-0 flex-1">
          <div className="truncate font-medium">{item.name}</div>
          {item.message ? (
            <div className="text-muted-foreground truncate text-sm">{item.message}</div>
          ) : null}
          {item.timestamp ? (
            <div className="text-muted-foreground text-xs">
              {formatRelativeTime(item.timestamp)}
            </div>
          ) : null}
        </div>
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={onTest}
        disabled={isPending}
        title="Test Prowlarr connection"
      >
        <FlaskConical className={cn('size-4', isPending && 'animate-pulse')} />
      </Button>
    </div>
  )
}
