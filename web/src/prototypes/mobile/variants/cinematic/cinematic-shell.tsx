import { useState } from 'react'

import { DetailSheet } from './detail-sheet'
import type { CineView } from './dock'
import { Dock } from './dock'
import { ActivityView } from './views/activity-view'
import { HomeView } from './views/home-view'
import { LibraryView } from './views/library-view'
import { SearchOverlay } from './views/search-overlay'

import './cinematic.css'

export function CinematicShell() {
  const [view, setView] = useState<CineView>('home')
  const [lastView, setLastView] = useState<Exclude<CineView, 'search'>>('home')
  const [detailId, setDetailId] = useState<number | null>(null)

  const changeView = (next: CineView) => {
    if (next !== 'search') {
      setLastView(next)
    }
    setView(next)
  }

  const closeSearch = () => setView(lastView)

  return (
    <div className="relative h-full overflow-hidden bg-background">
      <div className="absolute inset-0" hidden={lastView !== 'home'}><HomeView onOpen={setDetailId} /></div>
      <div className="absolute inset-0" hidden={lastView !== 'library'}><LibraryView onOpen={setDetailId} /></div>
      <div className="absolute inset-0" hidden={lastView !== 'activity'}><ActivityView onOpen={setDetailId} /></div>
      {view === 'search' && <SearchOverlay onClose={closeSearch} onOpen={setDetailId} />}
      <Dock view={view} onChange={changeView} />
      {detailId !== null && <DetailSheet id={detailId} onClose={() => setDetailId(null)} />}
    </div>
  )
}
