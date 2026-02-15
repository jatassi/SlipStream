import { Link } from '@tanstack/react-router'
import { Activity, FlaskConical, Settings } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useGlobalLoading } from '@/hooks'
import { useSystemHealthSummary, useTestHealthCategory } from '@/hooks/use-health'
import { cn } from '@/lib/utils'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthStatus,
} from '@/types/health'

import { StatusIndicator } from './status-indicator'
import { getStatusBgColor } from './status-utils'

type CategoryRowProps = {
  category: HealthCategory
  ok: number
  warning: number
  error: number
}

function getWorstStatus(errorCount: number, warningCount: number): HealthStatus {
  if (errorCount > 0) {return 'error'}
  if (warningCount > 0) {return 'warning'}
  return 'ok'
}

function buildStatusText(ok: number, warning: number, error: number): string {
  const parts: string[] = []
  if (ok > 0) {parts.push(`${ok} OK`)}
  if (warning > 0) {parts.push(`${warning} Warning`)}
  if (error > 0) {parts.push(`${error} Error`)}
  return parts.length > 0 ? parts.join(', ') : 'None'
}

const CATEGORY_LABELS: Record<string, { item: string; successVerb: string; failVerb: string }> = {
  rootFolders: { item: 'folder', successVerb: 'accessible', failVerb: 'inaccessible' },
  metadata: { item: 'API', successVerb: 'responding', failVerb: 'unreachable' },
}

const DEFAULT_LABEL = { item: 'connection', successVerb: 'verified', failVerb: 'failed' }

function getResultText(category: HealthCategory, count: number, success: boolean): string {
  const { item, successVerb, failVerb } = CATEGORY_LABELS[category] ?? DEFAULT_LABEL
  const verb = success ? successVerb : failVerb
  const plural = count === 1 ? '' : 's'
  return `${count} ${item}${plural} ${verb}`
}

function getAllResultText(category: HealthCategory, success: boolean): string {
  const { successVerb, failVerb } = CATEGORY_LABELS[category] ?? DEFAULT_LABEL
  const verb = success ? successVerb : failVerb
  const ALL_PREFIXES: Partial<Record<HealthCategory, string>> = {
    rootFolders: 'All folders',
    metadata: 'All APIs',
  }
  const prefix = ALL_PREFIXES[category] ?? 'All'
  return `${prefix} ${verb}`
}

async function runCategoryTest(
  testCategory: ReturnType<typeof useTestHealthCategory>,
  category: HealthCategory,
  total: number,
) {
  const categoryName = getCategoryDisplayName(category)
  try {
    const result = await testCategory.mutateAsync(category)
    const passed = result.results.filter((r) => r.success).length
    const failed = result.results.filter((r) => !r.success).length
    showTestToast({ name: categoryName, category, passed, failed, total })
  } catch {
    toast.error(`${categoryName}: Test failed`)
  }
}

type TestToastInput = {
  name: string
  category: HealthCategory
  passed: number
  failed: number
  total: number
}

function showTestToast({ name, category, passed, failed, total }: TestToastInput) {
  const useAllPhrasing = total <= 4
  if (failed === 0) {
    const text = useAllPhrasing ? getAllResultText(category, true) : getResultText(category, passed, true)
    toast.success(`${name}: ${text}`)
    return
  }
  if (passed === 0) {
    const text = useAllPhrasing ? getAllResultText(category, false) : getResultText(category, failed, false)
    toast.error(`${name}: ${text}`)
    return
  }
  toast.warning(
    `${name}: ${getResultText(category, passed, true)}, ${getResultText(category, failed, false)}`,
  )
}

function CategoryRow({ category, ok, warning, error }: CategoryRowProps) {
  const testCategory = useTestHealthCategory()
  const total = ok + warning + error
  const worstStatus = getWorstStatus(error, warning)
  const statusText = buildStatusText(ok, warning, error)
  const canTest = category !== 'storage' && category !== 'import'

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
        {canTest ? (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={() => void runCategoryTest(testCategory, category, total)}
            disabled={testCategory.isPending || total === 0}
            title={`Test all ${getCategoryDisplayName(category).toLowerCase()}`}
          >
            <FlaskConical className={cn('size-3', testCategory.isPending && 'animate-pulse')} />
          </Button>
        ) : null}
        <Link to={getCategorySettingsPath(category)}>
          <Button variant="ghost" size="sm" className="h-6 w-6 p-0" title="Settings">
            <Settings className="size-3" />
          </Button>
        </Link>
      </div>
    </div>
  )
}

function WidgetHeader({ trailing }: { trailing?: React.ReactNode }) {
  return (
    <CardHeader className="flex flex-row items-center justify-between pb-2">
      <CardTitle className="text-muted-foreground text-sm font-medium">System Health</CardTitle>
      <div className="flex items-center gap-2">
        {trailing}
        <Activity className="text-muted-foreground size-4" />
      </div>
    </CardHeader>
  )
}

function getOverallStatus(summary?: { hasIssues: boolean; categories: { error: number }[] }): HealthStatus {
  if (!summary?.hasIssues) {return 'ok'}
  return summary.categories.some((c) => c.error > 0) ? 'error' : 'warning'
}

function WidgetSkeleton() {
  return (
    <Card>
      <WidgetHeader />
      <CardContent className="space-y-2">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-8 w-full" />
        ))}
      </CardContent>
    </Card>
  )
}

function WidgetError() {
  return (
    <Card>
      <WidgetHeader />
      <CardContent>
        <p className="text-destructive text-sm">Failed to load health status</p>
      </CardContent>
    </Card>
  )
}

export function HealthWidget() {
  const globalLoading = useGlobalLoading()
  const { data: summary, isLoading: queryLoading, error } = useSystemHealthSummary()

  if (queryLoading || globalLoading) {return <WidgetSkeleton />}
  if (error) {return <WidgetError />}

  return (
    <Card>
      <WidgetHeader
        trailing={<StatusIndicator status={getOverallStatus(summary)} size="sm" />}
      />
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
