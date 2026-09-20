import { FlaskConical } from 'lucide-react'
import { toast } from 'sonner'

import { Group, Row } from '@/components/grouped-list'
import { useTestHealthCategory } from '@/hooks/use-health'
import { cn } from '@/lib/utils'
import {
  getCategoryDisplayName,
  getCategorySettingsPath,
  type HealthCategory,
  type HealthItem,
} from '@/types/health'

import { HealthRow } from './health-row'
import { getResultText } from './health-utils'

type TestResultsInput = {
  categoryName: string
  category: HealthCategory | 'prowlarr_indexers'
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

function TestAllAction({
  category,
  categoryName,
  items,
  disabled,
}: {
  category: HealthCategory
  categoryName: string
  items: HealthItem[]
  disabled: boolean
}) {
  const testCategory = useTestHealthCategory()

  const handleTestAll = async () => {
    try {
      const result = await testCategory.mutateAsync(category)
      showTestResults({ categoryName, category, items, results: result.results })
    } catch {
      toast.error(`${categoryName}: Test failed`)
    }
  }

  return (
    <button
      type="button"
      disabled={disabled || testCategory.isPending}
      onClick={() => void handleTestAll()}
      className="press text-footnote flex items-center gap-1.5 rounded-md px-1 py-0.5 font-medium text-tv-400 focus-visible:ring-ring outline-none focus-visible:ring-[3px] disabled:opacity-50"
    >
      <FlaskConical className={cn('size-3.5', testCategory.isPending && 'animate-pulse')} />
      Test All
    </button>
  )
}

export function HealthGroup({
  category,
  items,
  leadingItem,
}: {
  category: HealthCategory
  items: HealthItem[]
  leadingItem?: HealthItem
}) {
  const categoryName = getCategoryDisplayName(category)
  const settingsPath = getCategorySettingsPath(category)
  const testable = category !== 'storage'
  const all = leadingItem === undefined ? items : [leadingItem, ...items]

  return (
    <Group
      header={categoryName}
      action={
        testable ? (
          <TestAllAction
            category={category}
            categoryName={categoryName}
            items={items}
            disabled={items.length === 0}
          />
        ) : undefined
      }
    >
      {all.length === 0 ? (
        <Row title="No items configured" subtitle="Add one in settings" href={settingsPath} chevron />
      ) : (
        all.map((item) => <HealthRow key={item.id} item={item} testable={testable} />)
      )}
    </Group>
  )
}
