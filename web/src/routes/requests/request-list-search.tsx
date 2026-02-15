import { ArrowRight, Search } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

type SearchSectionProps = {
  searchQuery: string
  setSearchQuery: (q: string) => void
  searchFocused: boolean
  setSearchFocused: (f: boolean) => void
  onSearch: (e: React.FormEvent) => void
}

export function SearchSection({
  searchQuery,
  setSearchQuery,
  searchFocused,
  setSearchFocused,
  onSearch,
}: SearchSectionProps) {
  return (
    <section className="border-border from-movie-500/5 to-tv-500/5 flex flex-col items-center justify-center border-b bg-gradient-to-b via-transparent py-8">
      <div className="w-full max-w-2xl space-y-4 px-6">
        <div className="space-y-1 text-center">
          <Search className="text-media-gradient mx-auto size-10" />
          <h2 className="text-xl font-semibold">Search for Content</h2>
          <p className="text-muted-foreground text-sm">Find movies and TV series to request</p>
        </div>
        <form onSubmit={onSearch} className="flex gap-2">
          <div
            className={`relative flex-1 rounded-md transition-shadow duration-300 ${searchFocused ? 'glow-media-sm' : ''}`}
          >
            <Search className="text-muted-foreground absolute top-1/2 left-3 z-10 size-4 -translate-y-1/2" />
            <Input
              type="text"
              placeholder="Search movies and series..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onFocus={() => setSearchFocused(true)}
              onBlur={() => setSearchFocused(false)}
              className="h-11 pl-10"
            />
          </div>
          <Button type="submit" size="icon" className="h-11 w-11">
            <ArrowRight className="size-5" />
          </Button>
        </form>
      </div>
    </section>
  )
}
