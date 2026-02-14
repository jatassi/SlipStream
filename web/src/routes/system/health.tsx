import { Link } from '@tanstack/react-router'
import { ExternalLink, FlaskConical, Settings } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { getStatusBgColor, StatusIndicator } from '@/components/health'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useGlobalLoading } from '@/hooks'
import { useSystemHealth, useTestHealthCategory, useTestHealthItem } from '@/hooks/useHealth'
import { useIndexerMode } from '@/hooks/useProwlarr'
import { cn } from '@/lib/utils'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthItem,
} from '@/types/health'

const getItemNameById = (items: HealthItem[], id: string) => items.find((i) => i.id === id)?.name ?? id

function formatRelativeTime(dateString?: string): string {
  if (!dateString) {
    return ''
  }

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.round(diffMs / 60_000)
  const diffHours = Math.round(diffMs / 3_600_000)
  const diffDays = Math.round(diffMs / 86_400_000)

  if (diffMins < 1) {
    return 'Just now'
  }
  if (diffMins < 60) {
    return `${diffMins} min ago`
  }
  if (diffHours < 24) {
    return `${diffHours} hours ago`
  }
  return `${diffDays} days ago`
}

type HealthItemRowProps = {
  item: HealthItem
}

function HealthItemRow({ item }: HealthItemRowProps) {
  const testItem = useTestHealthItem()

  const handleTest = async () => {
    try {
      const result = await testItem.mutateAsync({ category: item.category, id: item.id })
      if (result.success) {
        toast.success(`${item.name}: ${result.message || 'Test passed'}`)
      } else {
        toast.error(`${item.name}: ${result.message || 'Test failed'}`)
      }
    } catch {
      toast.error(`${item.name}: Connection test failed`)
    }
  }

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
      {item.category !== 'storage' && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleTest}
          disabled={testItem.isPending}
          title={`Test ${item.name}`}
        >
          <FlaskConical className={cn('size-4', testItem.isPending && 'animate-pulse')} />
        </Button>
      )}
    </div>
  )
}

type HealthItemRowChildProps = {
  item: HealthItem
}

function HealthItemRowChild({ item }: HealthItemRowChildProps) {
  const testItem = useTestHealthItem()

  const handleTest = async () => {
    try {
      const result = await testItem.mutateAsync({ category: item.category, id: item.id })
      if (result.success) {
        toast.success(`${item.name}: ${result.message || 'Test passed'}`)
      } else {
        toast.error(`${item.name}: ${result.message || 'Test failed'}`)
      }
    } catch {
      toast.error(`${item.name}: Connection test failed`)
    }
  }

  return (
    <div
      className={cn(
        'flex items-center justify-between rounded-md p-3 pl-8',
        getStatusBgColor(item.status),
      )}
    >
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <div className="text-muted-foreground">└</div>
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
        onClick={handleTest}
        disabled={testItem.isPending}
        title={`Test ${item.name}`}
      >
        <FlaskConical className={cn('size-4', testItem.isPending && 'animate-pulse')} />
      </Button>
    </div>
  )
}

type ProwlarrTreeCardProps = {
  prowlarrItem: HealthItem | undefined
  indexerItems: HealthItem[]
}

function ProwlarrTreeCard({ prowlarrItem, indexerItems }: ProwlarrTreeCardProps) {
  const testCategory = useTestHealthCategory()
  const testItem = useTestHealthItem()

  const handleTestProwlarr = async () => {
    if (!prowlarrItem) {
      return
    }
    try {
      const result = await testItem.mutateAsync({ category: 'prowlarr', id: prowlarrItem.id })
      if (result.success) {
        toast.success(`Prowlarr: ${result.message || 'Connection successful'}`)
      } else {
        toast.error(`Prowlarr: ${result.message || 'Connection failed'}`)
      }
    } catch {
      toast.error('Prowlarr: Connection test failed')
    }
  }

  const handleTestAll = async () => {
    try {
      const result = await testCategory.mutateAsync('indexers')
      const passedItems = result.results.filter((r) => r.success)
      const failedItems = result.results.filter((r) => !r.success)

      const getResultText = (items: { id: string }[], success: boolean) => {
        const count = items.length
        if (indexerItems.length <= 4 && count > 0) {
          const names = items.map((r) => getItemNameById(indexerItems, r.id)).join(', ')
          return success ? `${names} connected` : `${names} failed`
        }
        return success
          ? `${count} connection${count === 1 ? '' : 's'} verified`
          : `${count} connection${count === 1 ? '' : 's'} failed`
      }

      if (failedItems.length === 0) {
        toast.success(`Indexers: ${getResultText(passedItems, true)}`)
      } else if (passedItems.length === 0) {
        toast.error(`Indexers: ${getResultText(failedItems, false)}`)
      } else {
        toast.warning(
          `Indexers: ${getResultText(passedItems, true)}, ${getResultText(failedItems, false)}`,
        )
      }
    } catch {
      toast.error('Indexers: Test failed')
    }
  }

  // Calculate worst status across Prowlarr and all indexers
  let worstStatus: 'ok' | 'warning' | 'error' = prowlarrItem?.status ?? 'ok'
  for (const item of indexerItems) {
    if (item.status === 'error') {
      worstStatus = 'error'
      break
    }
    if (item.status === 'warning' && worstStatus !== 'error') {
      worstStatus = 'warning'
    }
  }

  const totalItems = (prowlarrItem ? 1 : 0) + indexerItems.length

  return (
    <Card>
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
            onClick={handleTestAll}
            disabled={testCategory.isPending || indexerItems.length === 0}
            title="Test all indexers"
          >
            <FlaskConical
              className={cn('mr-1 size-4', testCategory.isPending && 'animate-pulse')}
            />
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
      <CardContent className="space-y-2">
        {prowlarrItem ? (
          <>
            {/* Prowlarr parent item */}
            <div
              className={cn(
                'flex items-center justify-between rounded-md p-3',
                getStatusBgColor(prowlarrItem.status),
              )}
            >
              <div className="flex min-w-0 flex-1 items-center gap-3">
                <StatusIndicator status={prowlarrItem.status} size="md" />
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">{prowlarrItem.name}</div>
                  {prowlarrItem.message ? (
                    <div className="text-muted-foreground truncate text-sm">
                      {prowlarrItem.message}
                    </div>
                  ) : null}
                  {prowlarrItem.timestamp ? (
                    <div className="text-muted-foreground text-xs">
                      {formatRelativeTime(prowlarrItem.timestamp)}
                    </div>
                  ) : null}
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleTestProwlarr}
                disabled={testItem.isPending}
                title="Test Prowlarr connection"
              >
                <FlaskConical className={cn('size-4', testItem.isPending && 'animate-pulse')} />
              </Button>
            </div>

            {/* Indexer child items */}
            {indexerItems.length === 0 ? (
              <div className="text-muted-foreground py-2 pl-8 text-sm">
                No indexers found in Prowlarr
              </div>
            ) : (
              indexerItems.map((item) => <HealthItemRowChild key={item.id} item={item} />)
            )}
          </>
        ) : (
          <div className="text-muted-foreground py-4 text-center text-sm">
            Prowlarr not configured.{' '}
            <Link
              to="/settings/downloads"
              className="text-primary inline-flex items-center gap-1 hover:underline"
            >
              Configure <ExternalLink className="size-3" />
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

type HealthCategoryCardProps = {
  category: HealthCategory
  items: HealthItem[]
}

function HealthCategoryCard({ category, items }: HealthCategoryCardProps) {
  const testCategory = useTestHealthCategory()

  const handleTestAll = async () => {
    const categoryName = getCategoryDisplayName(category)

    // Get descriptive text based on category and results
    const getResultText = (resultItems: { id: string; success: boolean }[], success: boolean) => {
      const count = resultItems.length
      const names = resultItems.map((r) => getItemNameById(items, r.id))

      // Enumerate names if 4 or fewer items total
      if (items.length <= 4 && count > 0) {
        const nameList = names.join(', ')
        if (category === 'rootFolders') {
          return success ? `${nameList} accessible` : `${nameList} inaccessible`
        }
        if (category === 'metadata') {
          return success ? `${nameList} responding` : `${nameList} unreachable`
        }
        return success ? `${nameList} connected` : `${nameList} failed`
      }

      // Fall back to counts for larger sets
      if (category === 'rootFolders') {
        return success
          ? `${count} folder${count === 1 ? '' : 's'} accessible`
          : `${count} folder${count === 1 ? '' : 's'} inaccessible`
      }
      if (category === 'metadata') {
        return success
          ? `${count} API${count === 1 ? '' : 's'} responding`
          : `${count} API${count === 1 ? '' : 's'} unreachable`
      }
      return success
        ? `${count} connection${count === 1 ? '' : 's'} verified`
        : `${count} connection${count === 1 ? '' : 's'} failed`
    }

    try {
      const result = await testCategory.mutateAsync(category)
      const passedItems = result.results.filter((r) => r.success)
      const failedItems = result.results.filter((r) => !r.success)

      if (failedItems.length === 0) {
        toast.success(`${categoryName}: ${getResultText(passedItems, true)}`)
      } else if (passedItems.length === 0) {
        toast.error(`${categoryName}: ${getResultText(failedItems, false)}`)
      } else {
        toast.warning(
          `${categoryName}: ${getResultText(passedItems, true)}, ${getResultText(failedItems, false)}`,
        )
      }
    } catch {
      toast.error(`${categoryName}: Test failed`)
    }
  }

  // Get the worst status for the category header
  let worstStatus: 'ok' | 'warning' | 'error' = 'ok'
  for (const item of items) {
    if (item.status === 'error') {
      worstStatus = 'error'
      break
    }
    if (item.status === 'warning') {
      worstStatus = 'warning'
    }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <div className="flex items-center gap-2">
          <StatusIndicator status={worstStatus} size="sm" />
          <CardTitle className="text-base">{getCategoryDisplayName(category)}</CardTitle>
          <span className="text-muted-foreground text-sm">
            ({items.length} item{items.length === 1 ? '' : 's'})
          </span>
        </div>
        <div className="flex items-center gap-2">
          {category !== 'storage' && (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleTestAll}
              disabled={testCategory.isPending || items.length === 0}
              title="Test all"
            >
              <FlaskConical
                className={cn('mr-1 size-4', testCategory.isPending && 'animate-pulse')}
              />
              Test All
            </Button>
          )}
          <Link to={getCategorySettingsPath(category)}>
            <Button variant="ghost" size="sm" title="Settings">
              <Settings className="mr-1 size-4" />
              Settings
            </Button>
          </Link>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {items.length === 0 ? (
          <div className="text-muted-foreground py-4 text-center text-sm">
            No items configured.{' '}
            <Link
              to={getCategorySettingsPath(category)}
              className="text-primary inline-flex items-center gap-1 hover:underline"
            >
              Add one <ExternalLink className="size-3" />
            </Link>
          </div>
        ) : (
          items.map((item) => <HealthItemRow key={item.id} item={item} />)
        )}
      </CardContent>
    </Card>
  )
}

export function SystemHealthPage() {
  const globalLoading = useGlobalLoading()
  const { data: health, isLoading: queryLoading, error } = useSystemHealth()
  const isLoading = queryLoading || globalLoading
  const { data: modeData } = useIndexerMode()

  const isProwlarrMode = modeData?.effectiveMode === 'prowlarr'

  if (isLoading) {
    return (
      <div>
        <PageHeader
          title="System Health"
          description="Monitor the health of your system components"
        />
        <LoadingState variant="list" count={5} />
      </div>
    )
  }

  if (error) {
    return (
      <div>
        <PageHeader
          title="System Health"
          description="Monitor the health of your system components"
        />
        <ErrorState title="Failed to load health status" />
      </div>
    )
  }

  // Build categories list based on mode
  const regularCategories: { category: HealthCategory; items: HealthItem[] }[] = [
    { category: 'downloadClients', items: health?.downloadClients || [] },
    { category: 'rootFolders', items: health?.rootFolders || [] },
    { category: 'metadata', items: health?.metadata || [] },
    { category: 'storage', items: health?.storage || [] },
  ]

  // In Prowlarr mode, we show a tree structure; in SlipStream mode, we show the regular indexers card
  const prowlarrItem = health?.prowlarr[0]
  const indexerItems = health?.indexers ?? []

  return (
    <div>
      <PageHeader
        title="System Health"
        description="Monitor the health of your system components"
      />

      <div className="space-y-4">
        <HealthCategoryCard category="downloadClients" items={health?.downloadClients || []} />

        {isProwlarrMode ? (
          <ProwlarrTreeCard prowlarrItem={prowlarrItem} indexerItems={indexerItems} />
        ) : (
          <HealthCategoryCard category="indexers" items={indexerItems} />
        )}

        {regularCategories
          .filter((c) => c.category !== 'downloadClients')
          .map(({ category, items }) => (
            <HealthCategoryCard key={category} category={category} items={items} />
          ))}
      </div>
    </div>
  )
}
