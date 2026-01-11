import { Link } from '@tanstack/react-router'
import { Activity, Settings, FlaskConical } from 'lucide-react'
import { toast } from 'sonner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { StatusIndicator, getStatusBgColor } from './StatusIndicator'
import { useSystemHealthSummary, useTestHealthCategory } from '@/hooks/useHealth'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthStatus,
} from '@/types/health'
import { cn } from '@/lib/utils'

interface CategoryRowProps {
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
  if (error > 0) worstStatus = 'error'
  else if (warning > 0) worstStatus = 'warning'

  // Build status summary text
  const parts: string[] = []
  if (ok > 0) parts.push(`${ok} OK`)
  if (warning > 0) parts.push(`${warning} Warning`)
  if (error > 0) parts.push(`${error} Error`)
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
      const passed = result.results?.filter(r => r.success).length ?? 0
      const failed = result.results?.filter(r => !r.success).length ?? 0

      if (failed === 0) {
        toast.success(`${categoryName}: ${getResultText(passed, true, true)}`)
      } else if (passed === 0) {
        toast.error(`${categoryName}: ${getResultText(failed, false, true)}`)
      } else {
        toast.warning(`${categoryName}: ${getResultText(passed, true, false)}, ${getResultText(failed, false, false)}`)
      }
    } catch {
      toast.error(`${categoryName}: Test failed`)
    }
  }

  return (
    <div
      className={cn(
        'flex items-center justify-between p-2 rounded-md',
        getStatusBgColor(worstStatus)
      )}
    >
      <div className="flex items-center gap-2 flex-1 min-w-0">
        <StatusIndicator status={worstStatus} size="sm" />
        <span className="text-sm font-medium truncate">
          {getCategoryDisplayName(category)}
        </span>
        <span className="text-xs text-muted-foreground hidden sm:inline">
          {total > 0 ? statusText : 'No items'}
        </span>
      </div>
      <div className="flex items-center gap-1">
        {category !== 'storage' && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={handleTest}
            disabled={testCategory.isPending || total === 0}
            title={`Test all ${getCategoryDisplayName(category).toLowerCase()}`}
          >
            <FlaskConical
              className={cn('size-3', testCategory.isPending && 'animate-pulse')}
            />
          </Button>
        )}
        <Link to={getCategorySettingsPath(category)}>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            title="Settings"
          >
            <Settings className="size-3" />
          </Button>
        </Link>
      </div>
    </div>
  )
}

export function HealthWidget() {
  const { data: summary, isLoading, error } = useSystemHealthSummary()

  if (isLoading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            System Health
          </CardTitle>
          <Activity className="size-4 text-muted-foreground" />
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
          <CardTitle className="text-sm font-medium text-muted-foreground">
            System Health
          </CardTitle>
          <Activity className="size-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">Failed to load health status</p>
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
        <CardTitle className="text-sm font-medium text-muted-foreground">
          System Health
        </CardTitle>
        <div className="flex items-center gap-2">
          <StatusIndicator status={overallStatus} size="sm" />
          <Activity className="size-4 text-muted-foreground" />
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
