import { useEffect, useRef, useState, useSyncExternalStore } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { Loader2, Search, X } from 'lucide-react'

import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

type Breakpoint = 'sm' | 'md' | 'lg'

function getBreakpoint(): Breakpoint {
  if (globalThis.window === undefined) {
    return 'lg'
  }
  const width = window.innerWidth
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

export function SearchBar() {
  const navigate = useNavigate()
  const breakpoint = useBreakpoint()
  const [searchQuery, setSearchQuery] = useState('')
  const [isSearching, setIsSearching] = useState(false)
  const [isFocused, setIsFocused] = useState(false)
  const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleSearchChange = (value: string) => {
    setSearchQuery(value)

    if (debounceTimeoutRef.current) {
      clearTimeout(debounceTimeoutRef.current)
    }

    if (value.trim()) {
      setIsSearching(true)
      debounceTimeoutRef.current = setTimeout(() => {
        navigate({ to: '/search', search: { q: value.trim() } })
        setIsSearching(false)
      }, 500)
    } else {
      setIsSearching(false)
    }
  }

  useEffect(() => {
    return () => {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current)
      }
    }
  }, [])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && searchQuery.trim()) {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current)
      }
      navigate({ to: '/search', search: { q: searchQuery.trim() } })
      setIsSearching(false)
    }
  }

  return (
    <div
      className={cn(
        'relative rounded-md transition-shadow duration-300',
        isSearching && 'glow-media-pulse',
        isFocused && !isSearching && 'glow-media-sm',
      )}
    >
      {isSearching ? (
        <Loader2 className="text-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2 animate-spin" />
      ) : (
        <Search className="text-muted-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2" />
      )}
      <Input
        type="text"
        placeholder={PLACEHOLDER_TEXT[breakpoint]}
        value={searchQuery}
        onChange={(e) => handleSearchChange(e.target.value)}
        onKeyDown={handleKeyDown}
        onFocus={() => setIsFocused(true)}
        onBlur={() => setIsFocused(false)}
        id="global-search"
        className="border-white/50 pr-8 pl-9 focus-visible:border-white"
      />
      {searchQuery ? (
        <button
          type="button"
          onClick={() => setSearchQuery('')}
          className="absolute top-1/2 right-2 z-10 -translate-y-1/2 text-white/70 hover:text-white"
        >
          <X className="size-4" />
        </button>
      ) : null}
    </div>
  )
}
