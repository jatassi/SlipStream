import { AlertTriangle, CheckCircle2 } from 'lucide-react'

import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import { useSystemHealth } from '@/hooks/use-health'
import { useUIStore } from '@/stores'
import type { HealthItem, HealthResponse } from '@/types/health'

const HEALTH_KEYS: (keyof HealthResponse)[] = [
  'downloadClients',
  'indexers',
  'prowlarr',
  'rootFolders',
  'metadata',
  'storage',
  'import',
]

const SKELETONS = ['health-a', 'health-b'] as const

function collectIssues(health: HealthResponse | undefined): HealthItem[] {
  if (!health) {
    return []
  }
  return HEALTH_KEYS.flatMap((key) => health[key].filter((item) => item.status !== 'ok'))
}

function IssueRow({ issue }: { issue: HealthItem }) {
  return (
    <Row
      href="/system/health"
      leading={
        <IconTile className="bg-amber-500">
          <AlertTriangle />
        </IconTile>
      }
      title={issue.name}
      subtitle={issue.message}
      chevron
    />
  )
}

function HealthyRow() {
  return (
    <Row
      href="/system/health"
      leading={
        <IconTile className="bg-emerald-600">
          <CheckCircle2 />
        </IconTile>
      }
      title="All systems healthy"
      subtitle="No issues"
      chevron
    />
  )
}

export function HealthGroup() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data, isLoading } = useSystemHealth()

  if (isLoading || globalLoading) {
    return (
      <Group header="Health" inset={false} className="mb-0">
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} chevron />
        ))}
      </Group>
    )
  }

  const issues = collectIssues(data)
  return (
    <Group header="Health" inset={false} className="mb-0">
      {issues.length === 0 ? <HealthyRow /> : issues.map((issue) => <IssueRow key={`${issue.category}-${issue.id}`} issue={issue} />)}
    </Group>
  )
}
