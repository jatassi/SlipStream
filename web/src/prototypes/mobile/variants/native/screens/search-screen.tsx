import { useState } from 'react'

import { Search, X } from 'lucide-react'

import { cn } from '@/lib/utils'

import { KIND_LABEL, KIND_TEXT } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import { StatusDot } from '../../../shared/ui'
import { Group, Row } from '../grouped-list'
import { NativeScreen } from '../native-screen'
import type { NativeNav } from '../use-native-nav'

export function SearchField({ value, onChange, placeholder = 'Search library', className }: { value: string; onChange: (v: string) => void; placeholder?: string; className?: string }) {
  return (
    <label className={cn('flex h-10 items-center gap-2 rounded-[11px] bg-foreground/8 px-3', className)}>
      <Search className="size-4 shrink-0 text-muted-foreground" />
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        enterKeyHint="search"
        autoCapitalize="none"
        autoCorrect="off"
        className="min-w-0 flex-1 bg-transparent text-[16px] text-foreground outline-none placeholder:text-muted-foreground"
      />
      {value.length > 0 && (
        <button type="button" aria-label="Clear" onClick={() => onChange('')} className="m-press flex size-6 items-center justify-center rounded-full bg-foreground/20 text-background">
          <X className="size-3.5" strokeWidth={3} />
        </button>
      )}
    </label>
  )
}

export function SearchScreen({ nav }: { nav: NativeNav }) {
  const [query, setQuery] = useState('')
  const { library } = useMobileState()
  const q = query.trim().toLowerCase()
  const results = q.length === 0 ? [] : library.filter((m) => m.title.toLowerCase().includes(q))

  return (
    <NativeScreen title="Search">
      <div className="px-screen pb-5">
        <SearchField value={query} onChange={setQuery} />
      </div>
      {q.length === 0 && (
        <p className="px-screen text-m-body text-muted-foreground">Search {library.length} titles across movies and series.</p>
      )}
      {q.length > 0 && results.length === 0 && (
        <p className="px-screen text-m-body text-muted-foreground">No results for “{query}”.</p>
      )}
      {results.length > 0 && (
        <Group header={`${results.length} result${results.length === 1 ? '' : 's'}`}>
          {results.map((item) => (
            <Row
              key={item.id}
              leading={<Poster item={item} showTitle={false} className="w-9" radius="rounded-[4px]" />}
              title={item.title}
              subtitle={
                <span className="flex items-center gap-1.5">
                  <StatusDot status={item.status} />
                  <span className="m-nums">{item.year}</span>
                  <span>·</span>
                  <span className={KIND_TEXT[item.kind]}>{KIND_LABEL[item.kind]}</span>
                </span>
              }
              chevron
              onClick={() => nav.push({ kind: 'detail', id: item.id })}
            />
          ))}
        </Group>
      )}
    </NativeScreen>
  )
}
