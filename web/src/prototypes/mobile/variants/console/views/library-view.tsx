import { useState } from 'react'

import { formatGb } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import type { MediaItem } from '../../../shared/types'
import { KindMark } from '../../../shared/ui'
import { DenseRow, SectionLabel, StatusTag, Tag } from '../dense'

type Sort = 'title' | 'added' | 'size'

const SORTS: { id: Sort; label: string }[] = [
  { id: 'title', label: 'Title' },
  { id: 'added', label: 'Added' },
  { id: 'size', label: 'Size' },
]

function sortItems(items: MediaItem[], sort: Sort): MediaItem[] {
  if (sort === 'title') {
    return items.toSorted((a, b) => a.title.localeCompare(b.title))
  }
  if (sort === 'size') {
    return items.toSorted((a, b) => b.sizeGb - a.sizeGb)
  }
  return items
}

export function LibraryRows({ items, onOpen }: { items: MediaItem[]; onOpen: (id: number) => void }) {
  return (
    <div className="con-hairline">
      {items.map((item) => (
        <DenseRow
          key={item.id}
          leading={<Poster item={item} showTitle={false} className="w-7" radius="rounded-[3px]" />}
          title={<><KindMark kind={item.kind} className="mr-1.5 -translate-y-px" />{item.title}<span className="ml-1.5 font-mono text-[11px] text-muted-foreground">{item.year}</span></>}
          meta={<><StatusTag status={item.status} /><span className="mx-1.5 text-foreground/20">·</span>{item.profile}</>}
          trailing={<><Tag>{item.quality}</Tag><span className="w-14 text-right font-mono">{formatGb(item.sizeGb)}</span></>}
          onClick={() => onOpen(item.id)}
        />
      ))}
      {items.length === 0 && <p className="px-3 py-6 font-mono text-[12px] text-muted-foreground">no matches</p>}
    </div>
  )
}

export function LibraryView({ onOpen, query }: { onOpen: (id: number) => void; query: string }) {
  const [sort, setSort] = useState<Sort>('title')
  const { library } = useMobileState()
  const q = query.trim().toLowerCase()
  const filtered = q.length === 0 ? library : library.filter((m) => m.title.toLowerCase().includes(q))
  const items = sortItems(filtered, sort)

  return (
    <>
      <SectionLabel
        trailing={
          <span className="flex gap-2">
            {SORTS.map((s) => (
              <button key={s.id} type="button" onClick={() => setSort(s.id)} className={`m-press-dim font-mono text-[11px] ${sort === s.id ? 'text-foreground underline underline-offset-2' : 'text-muted-foreground'}`}>
                {s.label}
              </button>
            ))}
          </span>
        }
      >
        Library · {items.length}
      </SectionLabel>
      <LibraryRows items={items} onOpen={onOpen} />
    </>
  )
}
