import { AlertTriangle, Check, FileVideo, HelpCircle } from 'lucide-react'

import type { FilterType } from './filter-utils'
import { SummaryCard } from './summary-card'
import type { MigrationPreview } from './types'

type SummaryCardsProps = {
  summary: MigrationPreview['summary']
  filter: FilterType
  onFilterChange: (filter: FilterType) => void
}

export function SummaryCards({ summary, filter, onFilterChange }: SummaryCardsProps) {
  return (
    <div className="mb-2 grid shrink-0 grid-cols-4 gap-3">
      <SummaryCard
        label="All"
        value={summary.totalFiles}
        icon={FileVideo}
        active={filter === 'all'}
        onClick={() => onFilterChange('all')}
      />
      <SummaryCard
        label="Will Be Assigned"
        value={summary.filesWithSlots}
        icon={Check}
        variant="success"
        active={filter === 'assigned'}
        onClick={() => onFilterChange('assigned')}
      />
      <SummaryCard
        label="Conflicts"
        value={summary.conflicts}
        icon={AlertTriangle}
        variant={summary.conflicts > 0 ? 'warning' : 'default'}
        active={filter === 'conflicts'}
        onClick={() => onFilterChange('conflicts')}
      />
      <SummaryCard
        label="No Match"
        value={summary.filesNeedingReview}
        icon={HelpCircle}
        variant={summary.filesNeedingReview > 0 ? 'error' : 'default'}
        active={filter === 'nomatch'}
        onClick={() => onFilterChange('nomatch')}
      />
    </div>
  )
}
