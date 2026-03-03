import { useCallback, useMemo, useRef, useState } from 'react'

import { Download } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import type { QueueItem } from '@/types'

import { DownloadRow } from './download-row'
import { TitleWidthContext } from './title-width-context'

const RELEASE_NAME_KEY = 'downloads-show-release-name'

function useReleaseName() {
  const [show, setShow] = useState(() => {
    try { return localStorage.getItem(RELEASE_NAME_KEY) === 'true' } catch { return false }
  })
  const toggle = useCallback(() => {
    setShow((prev) => {
      const next = !prev
      try { localStorage.setItem(RELEASE_NAME_KEY, String(next)) } catch { /* storage unavailable */ }
      return next
    })
  }, [])
  return { show, toggle }
}

export function DownloadsTable({ items }: { items: QueueItem[] }) {
  const releaseName = useReleaseName()
  const widthsRef = useRef(new Map<string, number>())
  const [maxWidth, setMaxWidth] = useState(0)

  const registerWidth = useCallback((id: string, width: number) => {
    widthsRef.current.set(id, width)
    const newMax = Math.max(0, ...widthsRef.current.values())
    setMaxWidth((prev) => (prev === newMax ? prev : newMax))
  }, [])

  const unregisterWidth = useCallback((id: string) => {
    widthsRef.current.delete(id)
    const newMax = widthsRef.current.size > 0 ? Math.max(0, ...widthsRef.current.values()) : 0
    setMaxWidth((prev) => (prev === newMax ? prev : newMax))
  }, [])

  const contextValue = useMemo(
    () => ({ registerWidth, unregisterWidth, maxWidth }),
    [registerWidth, unregisterWidth, maxWidth],
  )

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
    <TitleWidthContext.Provider value={contextValue}>
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
