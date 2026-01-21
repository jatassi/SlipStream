import { useState, useEffect } from 'react'
import { Link, useNavigate, useLocation } from '@tanstack/react-router'
import { Settings, Search, ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NotificationBell } from './NotificationBell'

export function PortalHeader() {
  const navigate = useNavigate()
  const location = useLocation()

  const isSearchPage = location.pathname === '/requests/search'

  // Get current search query from URL if on search page
  const searchParams = new URLSearchParams(location.search)
  const currentQuery = searchParams.get('q') || ''
  const [searchInput, setSearchInput] = useState(currentQuery)

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
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-6">
      <Link to="/requests" className="flex items-center gap-1.5 md:gap-2 font-semibold text-base md:text-lg shrink-0">
        <div className="size-7 md:size-8 rounded bg-primary/20 flex items-center justify-center text-primary font-bold text-xs md:text-sm">
          SS
        </div>
        <span className={isSearchPage ? 'hidden sm:inline' : ''}>SlipStream</span>
      </Link>

      {isSearchPage && (
        <form onSubmit={handleSearch} className="flex-1 max-w-xl mx-2 sm:mx-8">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Search movies and series..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="pl-10 pr-10"
            />
            <button
              type="submit"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowRight className="size-4" />
            </button>
          </div>
        </form>
      )}

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
