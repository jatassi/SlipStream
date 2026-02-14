import { Link } from '@tanstack/react-router'
import { Activity, FlaskConical, Settings } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useGlobalLoading } from '@/hooks'
import { useSystemHealthSummary, useTestHealthCategory } from '@/hooks/useHealth'
import { cn } from '@/lib/utils'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthStatus,
} from '@/types/health'

import { StatusIndicator } from './StatusIndicator'
import { getStatusBgColor } from './statusUtils'

type CategoryRowProps = {
  category: HealthCategory
  ok: number
  warning: number
  error: number
}

function CategoryRow({ category, ok, warning, error }: CategoryRowProps) {
  const testCategory = useTestHealthCategory()
  const total = ok + warning + error

  // Determine worst status for the category
  let worstStatus: HealthStatus = 'ok'
  if (error > 0) {
    worstStatus = 'error'
  } else if (warning > 0) {
    worstStatus = 'warning'
  }

  // Build status summary text
  const parts: string[] = []
  if (ok > 0) {
    parts.push(`${ok} OK`)
  }
  if (warning > 0) {
    parts.push(`${warning} Warning`)
  }
  if (error > 0) {
    parts.push(`${error} Error`)
  }
  const statusText = parts.length > 0 ? parts.join(', ') : 'None'

  const handleTest = async () => {
    const categoryName = getCategoryDisplayName(category)

    // Get descriptive text based on category
    const getResultText = (count: number, success: boolean, isAll: boolean) => {
      // Use simpler "all" phrasing for small sets
      if (isAll && total <= 4) {
        if (category === 'rootFolders') {
          return success ? 'All folders accessible' : 'All folders inaccessible'
        }
        if (category === 'metadata') {
          return success ? 'All APIs responding' : 'All APIs unreachable'
        }
        return success ? 'All connected' : 'All failed'
      }

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
      const passed = result.results.filter((r) => r.success).length
      const failed = result.results.filter((r) => !r.success).length

      if (failed === 0) {
        toast.success(`${categoryName}: ${getResultText(passed, true, true)}`)
      } else if (passed === 0) {
        toast.error(`${categoryName}: ${getResultText(failed, false, true)}`)
      } else {
        toast.warning(
          `${categoryName}: ${getResultText(passed, true, false)}, ${getResultText(failed, false, false)}`,
        )
      }
    } catch {
      toast.error(`${categoryName}: Test failed`)
    }
  }

  return (
    <div
      className={cn(
        'flex items-center justify-between rounded-md p-2',
        getStatusBgColor(worstStatus),
      )}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <StatusIndicator status={worstStatus} size="sm" />
        <span className="truncate text-sm font-medium">{getCategoryDisplayName(category)}</span>
        <span className="text-muted-foreground hidden text-xs sm:inline">
          {total > 0 ? statusText : 'No items'}
        </span>
      </div>
      <div className="flex items-center gap-1">
        {category !== 'storage' && category !== 'import' && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={handleTest}
            disabled={testCategory.isPending || total === 0}
            title={`Test all ${getCategoryDisplayName(category).toLowerCase()}`}
          >
            <FlaskConical className={cn('size-3', testCategory.isPending && 'animate-pulse')} />
          </Button>
        )}
        <Link to={getCategorySettingsPath(category)}>
          <Button variant="ghost" size="sm" className="h-6 w-6 p-0" title="Settings">
            <Settings className="size-3" />
          </Button>
        </Link>
      </div>
    </div>
  )
}

export function HealthWidget() {
  const globalLoading = useGlobalLoading()
  const { data: summary, isLoading: queryLoading, error } = useSystemHealthSummary()
  const isLoading = queryLoading || globalLoading

  if (isLoading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">System Health</CardTitle>
          <Activity className="text-muted-foreground size-4" />
        </CardHeader>
        <CardContent className="space-y-2">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-8 w-full" />
          ))}
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-muted-foreground text-sm font-medium">System Health</CardTitle>
          <Activity className="text-muted-foreground size-4" />
        </CardHeader>
        <CardContent>
          <p className="text-destructive text-sm">Failed to load health status</p>
        </CardContent>
      </Card>
    )
  }

  // Determine overall status
  let overallStatus: HealthStatus = 'ok'
  if (summary?.hasIssues) {
    const hasErrors = summary.categories.some((c) => c.error > 0)
    overallStatus = hasErrors ? 'error' : 'warning'
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-muted-foreground text-sm font-medium">System Health</CardTitle>
        <div className="flex items-center gap-2">
          <StatusIndicator status={overallStatus} size="sm" />
          <Activity className="text-muted-foreground size-4" />
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        {summary?.categories.map((cat) => (
          <CategoryRow
            key={cat.category}
            category={cat.category}
            ok={cat.ok}
            warning={cat.warning}
            error={cat.error}
          />
        ))}
        <div className="pt-2">
          <Link to="/system/health">
            <Button variant="outline" size="sm" className="w-full">
              View Details
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}
