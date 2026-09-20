import { useState } from 'react'

import { Group } from '@/components/grouped-list'
import { cn } from '@/lib/utils'

export function MediaOverview({ text }: { text?: string }) {
  const [expanded, setExpanded] = useState(false)

  if (!text) {
    return null
  }

  return (
    <Group header="Overview">
      <button
        type="button"
        aria-expanded={expanded}
        aria-label={expanded ? 'Collapse overview' : 'Expand overview'}
        onClick={() => setExpanded((prev) => !prev)}
        className="press-row w-full px-4 py-3 text-left"
      >
        <span className={cn('text-body text-foreground/90 block', !expanded && 'line-clamp-3')}>
          {text}
        </span>
      </button>
    </Group>
  )
}
