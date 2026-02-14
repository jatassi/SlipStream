import { useEffect, useState } from 'react'

import { Link, useLocation, useNavigate } from '@tanstack/react-router'
import { ArrowRight, Search, Settings } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { NotificationBell } from './NotificationBell'

export function PortalHeader() {
  const navigate = useNavigate()
  const location = useLocation()

  const isSearchPage = location.pathname === '/requests/search'

  // Get current search query from URL if on search page
  const searchParams = new URLSearchParams(
    typeof location.search === 'string' ? location.search : '',
  )
  const currentQuery = searchParams.get('q') ?? ''
  const [searchInput, setSearchInput] = useState(currentQuery)
  const [searchFocused, setSearchFocused] = useState(false)

  // Keep search input in sync with URL query
  useEffect(() => {
    setSearchInput(currentQuery)
  }, [currentQuery])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (searchInput.trim()) {
      navigate({ to: '/requests/search', search: { q: searchInput.trim() } })
    }
  }

  return (
    <header className="border-border bg-card flex h-14 items-center justify-between border-b px-6">
      <Link
        to="/requests"
        className="flex shrink-0 items-center gap-1.5 text-base font-semibold md:gap-2 md:text-lg"
      >
        <div className="bg-media-gradient glow-media-sm flex size-7 items-center justify-center rounded text-xs font-bold text-white md:size-8 md:text-sm">
          SS
        </div>
        <span className={`text-media-gradient ${isSearchPage ? 'hidden sm:inline' : ''}`}>
          SlipStream
        </span>
      </Link>

      {isSearchPage ? (
        <form onSubmit={handleSearch} className="mx-2 max-w-xl flex-1 sm:mx-8">
          <div
            className={`relative rounded-md transition-shadow duration-300 ${searchFocused ? 'glow-media-sm' : ''}`}
          >
            <Search className="text-muted-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2" />
            <Input
              type="text"
              placeholder="Search movies and series..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onFocus={() => setSearchFocused(true)}
              onBlur={() => setSearchFocused(false)}
              className="pr-10 pl-10"
            />
            <button
              type="submit"
              className="text-muted-foreground hover:text-foreground absolute top-1/2 right-3 z-10 -translate-y-1/2 transition-colors"
            >
              <ArrowRight className="size-4" />
            </button>
          </div>
        </form>
      ) : null}

      <div className="flex items-center gap-1 md:gap-2">
        <NotificationBell />

        <Button
          variant="ghost"
          size="icon"
          onClick={() => navigate({ to: '/requests/settings' })}
          className="size-8 md:size-9"
        >
          <Settings className="size-4 md:size-5" />
        </Button>
      </div>
    </header>
  )
}
