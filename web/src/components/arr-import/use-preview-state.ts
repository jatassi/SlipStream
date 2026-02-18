import { useState } from 'react'

import type { ImportPreview, MoviePreview, SeriesPreview } from '@/types/arr-import'

type PreviewFilter = 'all' | 'new' | 'duplicate' | 'skip'

function collectNewIds(preview: ImportPreview, isMovie: boolean): Set<number> {
  const ids = new Set<number>()
  if (isMovie) {
    for (const m of preview.movies) {
      if (m.status === 'new') {
        ids.add(m.tmdbId)
      }
    }
  } else {
    for (const s of preview.series) {
      if (s.status === 'new') {
        ids.add(s.tvdbId)
      }
    }
  }
  return ids
}

export function usePreviewState(preview: ImportPreview, isMovie: boolean) {
  const [filter, setFilter] = useState<PreviewFilter>('all')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(() => collectNewIds(preview, isMovie))

  const items: (MoviePreview | SeriesPreview)[] = isMovie ? preview.movies : preview.series
  const filtered = items.filter((item) => filter === 'all' || item.status === filter)
  const newItems = items.filter((i) => i.status === 'new')

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const toggleSelectAll = () => {
    const newIds = newItems.map((i) => (isMovie ? (i as MoviePreview).tmdbId : (i as SeriesPreview).tvdbId))
    const allSelected = newIds.every((id) => selectedIds.has(id))
    setSelectedIds(allSelected ? new Set() : new Set(newIds))
  }

  return { filter, setFilter, selectedIds, filtered, newCount: newItems.length, toggleSelect, toggleSelectAll }
}
