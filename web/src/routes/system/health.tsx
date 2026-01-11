import { Link } from '@tanstack/react-router'
import { FlaskConical, Settings, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { StatusIndicator, getStatusBgColor } from '@/components/health'
import {
  useSystemHealth,
  useTestHealthCategory,
  useTestHealthItem,
} from '@/hooks/useHealth'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthItem,
} from '@/types/health'
import { cn } from '@/lib/utils'

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return ''

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.round(diffMs / 60000)
  const diffHours = Math.round(diffMs / 3600000)
  const diffDays = Math.round(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} min ago`
  if (diffHours < 24) return `${diffHours} hours ago`
  return `${diffDays} days ago`
}

interface HealthItemRowProps {
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
        'flex items-center justify-between p-3 rounded-md',
        getStatusBgColor(item.status)
      )}
    >
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <StatusIndicator status={item.status} size="md" />
        <div className="flex-1 min-w-0">
          <div className="font-medium truncate">{item.name}</div>
          {item.message && (
            <div className="text-sm text-muted-foreground truncate">
              {item.message}
            </div>
          )}
          {item.timestamp && (
            <div className="text-xs text-muted-foreground">
              {formatRelativeTime(item.timestamp)}
            </div>
          )}
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
          <FlaskConical
            className={cn('size-4', testItem.isPending && 'animate-pulse')}
          />
        </Button>
      )}
    </div>
  )
}

interface HealthCategoryCardProps {
  category: HealthCategory
  items: HealthItem[]
}

function HealthCategoryCard({ category, items }: HealthCategoryCardProps) {
  const testCategory = useTestHealthCategory()

  const handleTestAll = async () => {
    const categoryName = getCategoryDisplayName(category)

    // Get item name by ID
    const getItemName = (id: string) => items.find(i => i.id === id)?.name ?? id

    // Get descriptive text based on category and results
    const getResultText = (resultItems: { id: string; success: boolean }[], success: boolean) => {
      const count = resultItems.length
      const names = resultItems.map(r => getItemName(r.id))

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
          ? `${count} folder${count !== 1 ? 's' : ''} accessible`
          : `${count} folder${count !== 1 ? 's' : ''} inaccessible`
      }
      if (category === 'metadata') {
        return success
          ? `${count} API${count !== 1 ? 's' : ''} responding`
          : `${count} API${count !== 1 ? 's' : ''} unreachable`
      }
      return success
        ? `${count} connection${count !== 1 ? 's' : ''} verified`
        : `${count} connection${count !== 1 ? 's' : ''} failed`
    }

    try {
      const result = await testCategory.mutateAsync(category)
      const passedItems = result.results?.filter(r => r.success) ?? []
      const failedItems = result.results?.filter(r => !r.success) ?? []

      if (failedItems.length === 0) {
        toast.success(`${categoryName}: ${getResultText(passedItems, true)}`)
      } else if (passedItems.length === 0) {
        toast.error(`${categoryName}: ${getResultText(failedItems, false)}`)
      } else {
        toast.warning(`${categoryName}: ${getResultText(passedItems, true)}, ${getResultText(failedItems, false)}`)
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
          <CardTitle className="text-base">
            {getCategoryDisplayName(category)}
          </CardTitle>
          <span className="text-sm text-muted-foreground">
            ({items.length} item{items.length !== 1 ? 's' : ''})
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
                className={cn('size-4 mr-1', testCategory.isPending && 'animate-pulse')}
              />
              Test All
            </Button>
          )}
          <Link to={getCategorySettingsPath(category)}>
            <Button variant="ghost" size="sm" title="Settings">
              <Settings className="size-4 mr-1" />
              Settings
            </Button>
          </Link>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {items.length === 0 ? (
          <div className="text-sm text-muted-foreground text-center py-4">
            No items configured.{' '}
            <Link
              to={getCategorySettingsPath(category)}
              className="text-primary hover:underline inline-flex items-center gap-1"
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
  const { data: health, isLoading, error } = useSystemHealth()

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

  const categories: { category: HealthCategory; items: HealthItem[] }[] = [
    { category: 'downloadClients', items: health?.downloadClients || [] },
    { category: 'indexers', items: health?.indexers || [] },
    { category: 'rootFolders', items: health?.rootFolders || [] },
    { category: 'metadata', items: health?.metadata || [] },
    { category: 'storage', items: health?.storage || [] },
  ]

  return (
    <div>
      <PageHeader
        title="System Health"
        description="Monitor the health of your system components"
      />

      <div className="space-y-4">
        {categories.map(({ category, items }) => (
          <HealthCategoryCard key={category} category={category} items={items} />
        ))}
      </div>
    </div>
  )
}
