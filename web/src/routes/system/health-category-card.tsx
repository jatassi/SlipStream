import { Link } from '@tanstack/react-router'
import { ExternalLink, FlaskConical, Settings } from 'lucide-react'
import { toast } from 'sonner'

import { StatusIndicator } from '@/components/health'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useTestHealthCategory } from '@/hooks/use-health'
import { cn } from '@/lib/utils'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthItem,
} from '@/types/health'

import { HealthItemRow } from './health-item-row'
import { getResultText, getWorstStatus } from './health-utils'

type TestResultsInput = {
  categoryName: string
  category: HealthCategory
  items: HealthItem[]
  results: { id: string; success: boolean }[]
}

function showTestResults({ categoryName, category, items, results }: TestResultsInput) {
  const passed = results.filter((r) => r.success)
  const failed = results.filter((r) => !r.success)
  const passText = getResultText({ category, allItems: items, resultItems: passed, success: true })
  const failText = getResultText({ category, allItems: items, resultItems: failed, success: false })

  if (failed.length === 0) {
    toast.success(`${categoryName}: ${passText}`)
  } else if (passed.length === 0) {
    toast.error(`${categoryName}: ${failText}`)
  } else {
    toast.warning(`${categoryName}: ${passText}, ${failText}`)
  }
}

type HealthCategoryCardProps = {
  category: HealthCategory
  items: HealthItem[]
}

export function HealthCategoryCard({ category, items }: HealthCategoryCardProps) {
  const testCategory = useTestHealthCategory()
  const worstStatus = getWorstStatus(items)
  const categoryName = getCategoryDisplayName(category)
  const settingsPath = getCategorySettingsPath(category)
  const isStorage = category === 'storage'

  const handleTestAll = async () => {
    try {
      const result = await testCategory.mutateAsync(category)
      showTestResults({ categoryName, category, items, results: result.results })
    } catch {
      toast.error(`${categoryName}: Test failed`)
    }
  }

  return (
    <Card>
      <CategoryCardHeader
        worstStatus={worstStatus}
        categoryName={categoryName}
        settingsPath={settingsPath}
        itemCount={items.length}
        showTestAll={!isStorage}
        onTestAll={handleTestAll}
        testPending={testCategory.isPending}
      />
      <CategoryCardContent items={items} settingsPath={settingsPath} isStorage={isStorage} />
    </Card>
  )
}

type CategoryCardHeaderProps = {
  worstStatus: 'ok' | 'warning' | 'error'
  categoryName: string
  settingsPath: string
  itemCount: number
  showTestAll: boolean
  onTestAll: () => void
  testPending: boolean
}

function CategoryCardHeader({
  worstStatus,
  categoryName,
  settingsPath,
  itemCount,
  showTestAll,
  onTestAll,
  testPending,
}: CategoryCardHeaderProps) {
  return (
    <CardHeader className="flex flex-row items-center justify-between pb-2">
      <div className="flex items-center gap-2">
        <StatusIndicator status={worstStatus} size="sm" />
        <CardTitle className="text-base">{categoryName}</CardTitle>
        <span className="text-muted-foreground text-sm">
          ({itemCount} item{itemCount === 1 ? '' : 's'})
        </span>
      </div>
      <div className="flex items-center gap-2">
        {showTestAll ? (
          <Button
            variant="ghost"
            size="sm"
            onClick={onTestAll}
            disabled={testPending || itemCount === 0}
            title="Test all"
          >
            <FlaskConical className={cn('mr-1 size-4', testPending && 'animate-pulse')} />
            Test All
          </Button>
        ) : null}
        <Link to={settingsPath}>
          <Button variant="ghost" size="sm" title="Settings">
            <Settings className="mr-1 size-4" />
            Settings
          </Button>
        </Link>
      </div>
    </CardHeader>
  )
}

type CategoryCardContentProps = {
  items: HealthItem[]
  settingsPath: string
  isStorage: boolean
}

function CategoryCardContent({ items, settingsPath, isStorage }: CategoryCardContentProps) {
  if (items.length === 0) {
    return (
      <CardContent className="space-y-2">
        <div className="text-muted-foreground py-4 text-center text-sm">
          No items configured.{' '}
          <Link
            to={settingsPath}
            className="text-primary inline-flex items-center gap-1 hover:underline"
          >
            Add one <ExternalLink className="size-3" />
          </Link>
        </div>
      </CardContent>
    )
  }

  return (
    <CardContent className="space-y-2">
      {items.map((item) => (
        <HealthItemRow key={item.id} item={item} hideTestButton={isStorage} />
      ))}
    </CardContent>
  )
}
