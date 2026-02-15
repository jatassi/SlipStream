import { useEffect, useRef, useState, useSyncExternalStore } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Loader2, Search, X } from 'lucide-react'

import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

type Breakpoint = 'sm' | 'md' | 'lg'

function getBreakpoint(): Breakpoint {
  const width = globalThis.window.innerWidth
  if (width < 640) {
    return 'sm'
  }
  if (width < 1024) {
    return 'md'
  }
  return 'lg'
}

function subscribeToResize(callback: () => void): () => void {
  window.addEventListener('resize', callback)
  return () => window.removeEventListener('resize', callback)
}

function useBreakpoint(): Breakpoint {
  return useSyncExternalStore(subscribeToResize, getBreakpoint, () => 'lg')
}

const PLACEHOLDER_TEXT: Record<Breakpoint, string> = {
  sm: 'Search...',
  md: 'Search movies, series...',
  lg: 'Search for and add movies, series...',
}

function SearchInputIcon({ isSearching }: { isSearching: boolean }) {
  const cls = isSearching
    ? 'text-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2 animate-spin'
    : 'text-muted-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2'
  const Icon = isSearching ? Loader2 : Search
  return <Icon className={cls} />
}

type SearchInputProps = {
  searchQuery: string
  isSearching: boolean
  isFocused: boolean
  breakpoint: Breakpoint
  onChange: (value: string) => void
  onKeyDown: (e: React.KeyboardEvent) => void
  onFocus: () => void
  onBlur: () => void
  onClear: () => void
}

function SearchInput({
  searchQuery, isSearching, isFocused, breakpoint,
  onChange, onKeyDown, onFocus, onBlur, onClear,
}: SearchInputProps) {
  return (
    <div
      className={cn(
        'relative rounded-md transition-shadow duration-300',
        isSearching && 'glow-media-pulse',
        isFocused && !isSearching && 'glow-media-sm',
      )}
    >
      <SearchInputIcon isSearching={isSearching} />
      <Input
        type="text"
        placeholder={PLACEHOLDER_TEXT[breakpoint]}
        value={searchQuery}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        onFocus={onFocus}
        onBlur={onBlur}
        id="global-search"
        className="border-white/50 pr-8 pl-9 focus-visible:border-white"
      />
      {searchQuery ? (
        <button
          type="button"
          onClick={onClear}
          className="absolute top-1/2 right-2 z-10 -translate-y-1/2 text-white/70 hover:text-white"
        >
          <X className="size-4" />
        </button>
      ) : null}
    </div>
  )
}

function clearRef(ref: React.RefObject<ReturnType<typeof setTimeout> | null>) {
  if (ref.current) {
    clearTimeout(ref.current)
  }
}

export function SearchBar() {
  const navigate = useNavigate()
  const breakpoint = useBreakpoint()
  const [searchQuery, setSearchQuery] = useState('')
  const [isSearching, setIsSearching] = useState(false)
  const [isFocused, setIsFocused] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleSearchChange = (value: string) => {
    setSearchQuery(value)
    clearRef(timerRef)
    if (value.trim()) {
      setIsSearching(true)
      timerRef.current = setTimeout(() => {
        void navigate({ to: '/search', search: { q: value.trim() } })
        setIsSearching(false)
      }, 500)
    } else {
      setIsSearching(false)
    }
  }

  useEffect(() => () => clearRef(timerRef), [])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && searchQuery.trim()) {
      clearRef(timerRef)
      void navigate({ to: '/search', search: { q: searchQuery.trim() } })
      setIsSearching(false)
    }
  }

  return (
    <SearchInput
      searchQuery={searchQuery}
      isSearching={isSearching}
      isFocused={isFocused}
      breakpoint={breakpoint}
      onChange={handleSearchChange}
      onKeyDown={handleKeyDown}
      onFocus={() => setIsFocused(true)}
      onBlur={() => setIsFocused(false)}
      onClear={() => setSearchQuery('')}
    />
  )
}
