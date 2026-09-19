import { Search, X } from 'lucide-react'

import { cn } from '@/lib/utils'

type SearchFieldProps = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void
}

export function SearchField({
  value,
  onChange,
  placeholder = 'Search',
  className,
  onKeyDown,
}: SearchFieldProps) {
  return (
    <label className={cn('flex h-10 items-center gap-2 rounded-[11px] bg-foreground/8 px-3', className)}>
      <Search className="size-4 shrink-0 text-muted-foreground" />
      <input
        type="search"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        aria-label="Search"
        enterKeyHint="search"
        autoCapitalize="none"
        autoCorrect="off"
        className="min-w-0 flex-1 bg-transparent text-[16px] text-foreground outline-none placeholder:text-muted-foreground"
      />
      {value.length > 0 && (
        <button
          type="button"
          aria-label="Clear search"
          onClick={() => onChange('')}
          className="press flex size-6 items-center justify-center rounded-full bg-foreground/20 text-background"
        >
          <X className="size-3.5" strokeWidth={3} />
        </button>
      )}
    </label>
  )
}
