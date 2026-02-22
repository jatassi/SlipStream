import { useState } from 'react'

import type { NavigateOptions } from '@tanstack/react-router'
import { Link, useLocation, useNavigate } from '@tanstack/react-router'
import { ArrowRight, Library, Search, Settings } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { NotificationBell } from './notification-bell'

function SearchForm({
  searchInput,
  searchFocused,
  onInputChange,
  onFocusChange,
  onSubmit,
}: {
  searchInput: string
  searchFocused: boolean
  onInputChange: (value: string) => void
  onFocusChange: (focused: boolean) => void
  onSubmit: (e: React.SyntheticEvent) => void
}) {
  return (
    <form onSubmit={onSubmit} className="mx-2 min-w-0 flex-1 sm:mx-8 sm:max-w-xl">
      <div
        className={`relative rounded-md transition-shadow duration-300 ${searchFocused ? 'glow-media-sm' : ''}`}
      >
        <Search className="text-muted-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2" />
        <Input
          type="text"
          placeholder="Search movies and series..."
          value={searchInput}
          onChange={(e) => onInputChange(e.target.value)}
          onFocus={() => onFocusChange(true)}
          onBlur={() => onFocusChange(false)}
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
  )
}

function LibraryLink({ active, hideText }: { active: boolean; hideText: boolean }) {
  return (
    <Link
      to="/requests/library"
      className={`flex items-center gap-1.5 text-sm font-medium transition-colors ${
        active ? 'text-white icon-glow-media' : 'text-foreground/80 hover:text-foreground'
      }`}
    >
      <Library className="size-4 md:size-5" />
      {!hideText && <span>Library</span>}
    </Link>
  )
}

function LogoLink({ isHomePage }: { isHomePage: boolean }) {
  return (
    <Link
      to="/requests"
      className="flex shrink-0 items-center gap-1.5 text-base font-semibold md:gap-2 md:text-lg"
    >
      <div className="bg-media-gradient glow-media-sm flex size-7 items-center justify-center rounded text-xs font-bold text-white md:size-8 md:text-sm">
        SS
      </div>
      <span className={`text-media-gradient ${isHomePage ? '' : 'hidden sm:inline'}`}>
        SlipStream
      </span>
    </Link>
  )
}

export function PortalHeader() {
  const navigate = useNavigate()
  const location = useLocation()

  const isHomePage = location.pathname === '/requests'
  const isSearchPage = location.pathname === '/requests/search'
  const isLibraryPage = location.pathname === '/requests/library'

  const searchParams = new URLSearchParams(
    typeof location.search === 'string' ? location.search : '',
  )
  const currentQuery = searchParams.get('q') ?? ''
  const [searchInput, setSearchInput] = useState(currentQuery)
  const [searchFocused, setSearchFocused] = useState(false)
  const [prevQuery, setPrevQuery] = useState(currentQuery)

  if (currentQuery !== prevQuery) {
    setPrevQuery(currentQuery)
    setSearchInput(currentQuery)
  }

  const handleSearch = (e: React.SyntheticEvent) => {
    e.preventDefault()
    if (searchInput.trim()) {
      void navigate({ to: '/requests/search', search: { q: searchInput.trim() } } as NavigateOptions)
    }
  }

  return (
    <header className="border-border bg-card flex h-14 items-center justify-between border-b px-3 sm:px-6">
      <LogoLink isHomePage={isHomePage} />

      {isSearchPage || isLibraryPage ? (
        <SearchForm
          searchInput={searchInput}
          searchFocused={searchFocused}
          onInputChange={setSearchInput}
          onFocusChange={setSearchFocused}
          onSubmit={handleSearch}
        />
      ) : null}

      <div className="flex items-center gap-1 md:gap-2">
        <LibraryLink active={isLibraryPage} hideText={isSearchPage} />
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
