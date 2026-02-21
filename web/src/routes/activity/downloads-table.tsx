import { useCallback, useState } from 'react'

import { Download } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import type { QueueItem } from '@/types'

import { DownloadRow } from './download-row'
import { TitleWidthContext } from './title-width-context'

const RELEASE_NAME_KEY = 'downloads-show-release-name'

function useReleaseName() {
  const [show, setShow] = useState(() => localStorage.getItem(RELEASE_NAME_KEY) === 'true')
  const toggle = useCallback(() => {
    setShow((prev) => {
      const next = !prev
      localStorage.setItem(RELEASE_NAME_KEY, String(next))
      return next
    })
  }, [])
  return { show, toggle }
}

export function DownloadsTable({ items }: { items: QueueItem[] }) {
  const releaseName = useReleaseName()
  const [widths, setWidths] = useState<Map<string, number>>(new Map())

  const registerWidth = useCallback((id: string, width: number) => {
    setWidths((prev) => {
      const next = new Map(prev)
      next.set(id, width)
      return next
    })
  }, [])

  const unregisterWidth = useCallback((id: string) => {
    setWidths((prev) => {
      const next = new Map(prev)
      next.delete(id)
      return next
    })
  }, [])

  const maxWidth = Math.max(0, ...widths.values())

  if (items.length === 0) {
    return (
      <EmptyState
        icon={<Download className="size-8" />}
        title="No downloads"
        description="Downloads will appear here when they start"
        className="py-8"
      />
    )
  }

  return (
    <TitleWidthContext.Provider value={{ registerWidth, unregisterWidth, maxWidth }}>
      <div className="divide-border divide-y">
        {items.map((item) => (
          <DownloadRow
            key={`${item.clientId}-${item.id}`}
            item={item}
            showReleaseName={releaseName.show}
            onToggleReleaseName={releaseName.toggle}
          />
        ))}
      </div>
    </TitleWidthContext.Provider>
  )
}
