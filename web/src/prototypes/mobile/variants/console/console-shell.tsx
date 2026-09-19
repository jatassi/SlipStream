import type { ReactNode } from 'react'
import { useState } from 'react'

import { useMobileState } from '../../shared/mobile-state-context'
import { CommandBar } from './command-bar'
import { SectionLabel } from './dense'
import type { Scope } from './scopes'
import { StatusStrip } from './status-strip'
import { ActivityView } from './views/activity-view'
import { BoardView } from './views/board-view'
import { DetailView } from './views/detail-view'
import { LibraryRows, LibraryView } from './views/library-view'
import { MissingView, SystemView } from './views/system-views'

import './console.css'

function BoardOrSearch({ query, onOpen, onScope }: { query: string; onOpen: (id: number) => void; onScope: (scope: Scope) => void }) {
  const { library } = useMobileState()
  const q = query.trim().toLowerCase()
  if (q.length === 0) {
    return <BoardView onOpen={onOpen} onScope={onScope} />
  }
  const matches = library.filter((m) => m.title.toLowerCase().includes(q))
  return (
    <>
      <SectionLabel trailing={`${matches.length}`}>Results for “{query}”</SectionLabel>
      <LibraryRows items={matches} onOpen={onOpen} />
    </>
  )
}

export function ConsoleShell() {
  const [scope, setScope] = useState<Scope>('board')
  const [query, setQuery] = useState('')
  const [detailId, setDetailId] = useState<number | null>(null)
  const { queue, library } = useMobileState()

  const counts = {
    activity: queue.length,
    missing: library.filter((m) => m.status === 'missing' || m.status === 'upgradable').length,
    system: 2,
  }

  const changeScope = (next: Scope) => {
    setScope(next)
    setDetailId(null)
  }

  const views: Record<Scope, ReactNode> = {
    board: <BoardOrSearch query={query} onOpen={setDetailId} onScope={changeScope} />,
    library: <LibraryView query={query} onOpen={setDetailId} />,
    activity: <ActivityView query={query} onOpen={setDetailId} />,
    missing: <MissingView query={query} onOpen={setDetailId} />,
    system: <SystemView query={query} />,
  }

  return (
    <div className="relative h-full overflow-hidden bg-background">
      <StatusStrip />
      {detailId === null && (
        <div className="m-scroll h-full" style={{ paddingTop: 'calc(var(--safe-top) + 40px)', paddingBottom: 'calc(var(--safe-bottom) + 120px)' }}>
          {views[scope]}
        </div>
      )}
      {detailId !== null && <DetailView id={detailId} onBack={() => setDetailId(null)} />}
      <CommandBar scope={scope} onScope={changeScope} query={query} onQuery={setQuery} counts={counts} />
    </div>
  )
}
