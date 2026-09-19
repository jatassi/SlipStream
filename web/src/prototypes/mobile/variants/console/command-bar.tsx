import { Search, X } from 'lucide-react'

import { cn } from '@/lib/utils'

import type { Scope } from './scopes'
import { SCOPES } from './scopes'

type CommandBarProps = {
  scope: Scope
  onScope: (scope: Scope) => void
  query: string
  onQuery: (query: string) => void
  counts: Partial<Record<Scope, number>>
}

export function CommandBar({ scope, onScope, query, onQuery, counts }: CommandBarProps) {
  return (
    <div className="m-material-heavy m-safe-bottom absolute inset-x-0 bottom-0 z-30 shadow-[0_-1px_0_var(--material-edge)]">
      <div className="m-scroll-x con-chip-row flex gap-1.5 px-3 pt-2.5 pb-2">
        {SCOPES.map((s) => {
          const active = s.id === scope
          const count = counts[s.id]
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => onScope(s.id)}
              aria-current={active ? 'page' : undefined}
              className={cn(
                'm-press inline-flex h-8 shrink-0 items-center gap-1.5 rounded-[8px] px-3 text-[13px] font-semibold',
                active ? 'bg-foreground text-background' : 'bg-foreground/8 text-foreground/75',
              )}
            >
              {s.label}
              {count !== undefined && count > 0 ? (
                <span className={cn('m-nums font-mono text-[11px]', active ? 'text-background/70' : 'text-muted-foreground')}>{count}</span>
              ) : null}
            </button>
          )
        })}
      </div>
      <label className="mx-3 mb-2 flex h-11 items-center gap-2 rounded-[10px] bg-foreground/8 px-3 shadow-[inset_0_0_0_1px_var(--material-edge)]">
        <Search className="size-4 shrink-0 text-muted-foreground" />
        <input
          value={query}
          onChange={(e) => onQuery(e.target.value)}
          placeholder={`Filter ${scope === 'board' ? 'everything' : scope}…`}
          enterKeyHint="search"
          autoCapitalize="none"
          autoCorrect="off"
          className="con-input min-w-0 flex-1 bg-transparent text-foreground outline-none placeholder:text-muted-foreground"
        />
        {query.length > 0 && (
          <button type="button" aria-label="Clear" onClick={() => onQuery('')} className="m-press flex size-6 items-center justify-center rounded-full bg-foreground/15">
            <X className="size-3.5" strokeWidth={3} />
          </button>
        )}
      </label>
    </div>
  )
}
